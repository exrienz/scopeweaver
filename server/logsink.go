package server

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Autumn-27/artex/db"
)

// LogLine is one captured backend log entry exposed by the /api/logs endpoints.
type LogLine struct {
	Seq   int64  `json:"seq"`
	DBID  int64  `json:"db_id,omitempty"` // server_logs.id; 0 for pre-persistence entries
	TS    string `json:"ts"`
	Level string `json:"level"` // info | warn | error
	Tag   string `json:"tag"`   // the leading [tag] (pg / planner / activity / …), if any
	Text  string `json:"text"`
}

// dbWriteReq carries a log line to the async DB writer.
type dbWriteReq struct {
	seq int64
	ll  LogLine
}

// logSinkT tees the standard logger into an in-memory ring buffer and fans new
// lines out to SSE subscribers, so the UI can show a live backend log stream.
// It still passes everything through to stderr (the terminal keeps working).
type logSinkT struct {
	mu   sync.Mutex
	ring []LogLine
	cap  int
	seq  int64
	subs map[chan LogLine]struct{}
	out  io.Writer // passthrough (stderr)

	dbOnce sync.Once
	dbCh   chan dbWriteReq // buffered enqueue; nil once detached (never closed)
	// Writer lifecycle, bound to exactly dbPG: dbWriteCancel bounds in-flight DB
	// work; dbStop asks the writer to finish; dbDone is closed by the writer on
	// exit and joined by every same-owner StopDB. dbStopping makes close/cancel
	// happen exactly once; ownership fields survive the stop so repeated
	// same-owner stops keep sharing the same completion.
	dbWriteCancel context.CancelFunc
	dbStop        chan struct{}
	dbDone        chan struct{}
	dbPG          *db.DB
	dbStopping    bool
}

var logSink = &logSinkT{cap: 3000, subs: map[chan LogLine]struct{}{}, out: os.Stderr}

// StartLogCapture redirects the standard log package through the in-memory sink
// (still writing to stderr). Call once at startup, as early as possible.
func StartLogCapture() { log.SetOutput(logSink) }

// SetDB wires a postgres DB into the sink once (idempotent). It:
//  1. Restores the last 100 log rows from DB into the ring so the /logs page
//     shows history immediately after restart.
//  2. Starts an async writer tracked by the server's background drain.
func (s *logSinkT) SetDB(ctx context.Context, pg *db.DB, background ...*sync.WaitGroup) {
	if pg == nil {
		return
	}
	s.dbOnce.Do(func() {
		ch := make(chan dbWriteReq, 2000)
		stop := make(chan struct{})
		done := make(chan struct{})
		// Server cancellation stops dequeuing but lets an in-flight insert finish
		// under the background drain. StopDB explicitly cancels both restore and
		// insert SQL so direct manager close can join without waiting for a lock.
		writeCtx, writeCancel := context.WithCancel(context.WithoutCancel(ctx))
		restoreCtx, restoreCancel := context.WithCancel(ctx)
		defer restoreCancel()
		s.mu.Lock()
		s.dbCh = ch
		s.dbWriteCancel = func() {
			restoreCancel()
			writeCancel()
		}
		s.dbStop = stop
		s.dbDone = done
		s.dbPG = pg
		s.mu.Unlock()

		// Restore last 100 rows from DB → prepend to ring as history context.
		if logs, err := pg.RecentLogsContext(restoreCtx, 100); err == nil && len(logs) > 0 {
			s.mu.Lock()
			restored := make([]LogLine, 0, len(logs))
			for _, l := range logs {
				s.seq++
				restored = append(restored, LogLine{
					Seq:   s.seq,
					DBID:  l.ID,
					TS:    l.CreatedAt.Format(time.RFC3339),
					Level: l.Level,
					Tag:   l.Tag,
					Text:  l.Text,
				})
			}
			// Prepend history before any in-memory startup logs already in ring.
			s.ring = append(restored, s.ring...)
			if len(s.ring) > s.cap {
				s.ring = s.ring[len(s.ring)-s.cap:]
			}
			s.mu.Unlock()
		}

		// Async writer: drains the queue into server_logs. Errors go to stderr
		// directly (no recursive logging). It exits on stop, on ctx.Done, or if
		// the channel is ever closed, and always closes done so StopDB joins
		// deterministically. StopDB cancels writeCtx to bound stuck inserts.
		writer := func() {
			defer close(done)
			defer writeCancel()
			for {
				select {
				case req, ok := <-ch:
					// Shutdown may leave queued or later lines out of the DB;
					// Write still passes every line through to stderr.
					if !ok || ctx.Err() != nil || writeCtx.Err() != nil {
						return
					}
					id, err := pg.InsertLogContext(writeCtx, req.ll.Level, req.ll.Tag, req.ll.Text)
					if err != nil {
						_, _ = os.Stderr.Write([]byte("[logsink] db write: " + err.Error() + "\n"))
						continue
					}
					// Stamp DBID back into the ring entry so history pagination works.
					s.mu.Lock()
					for i := range s.ring {
						if s.ring[i].Seq == req.seq {
							s.ring[i].DBID = id
							break
						}
					}
					s.mu.Unlock()
				case <-stop:
					return
				case <-ctx.Done():
					return
				}
			}
		}
		if len(background) > 0 && background[0] != nil {
			background[0].Go(writer)
		} else {
			go writer()
		}
	})
}

// StopDB detaches DB persistence for exactly pg and joins the writer, so the
// caller can close pg without racing it. It cancels the writer's DB context
// first, bounding any in-flight query, then asks it to finish and waits for
// completion. Nil-safe and idempotent; concurrent same-owner stops share the
// same completion; a pg that was never bound (or a different manager's DB) is
// ignored. The ring, stderr passthrough and SSE subscribers keep working. The
// join waits without sink.mu (the writer stamps DBIDs under that lock), and
// dbCh is never closed (add sends after unlocking, so a close would race an
// enqueue; detached references only fill the abandoned buffer).
func (s *logSinkT) StopDB(pg *db.DB) {
	if pg == nil {
		return
	}
	s.mu.Lock()
	if s.dbPG != pg {
		s.mu.Unlock()
		return
	}
	if !s.dbStopping {
		s.dbStopping = true
		s.dbCh = nil // detach: no new enqueue can observe the channel
		s.dbWriteCancel()
		close(s.dbStop)
	}
	done := s.dbDone
	s.mu.Unlock()
	<-done // shared join: every same-owner stop waits for the same completion
}

// Write implements io.Writer for the log package: one call per log.Printf line.
func (s *logSinkT) Write(p []byte) (int, error) {
	_, _ = s.out.Write(p) // keep the terminal output
	line := strings.TrimRight(string(p), "\n")
	if strings.TrimSpace(line) != "" {
		s.add(parseLog(line))
	}
	return len(p), nil
}

func (s *logSinkT) add(l LogLine) {
	s.mu.Lock()
	s.seq++
	l.Seq = s.seq
	s.ring = append(s.ring, l)
	if len(s.ring) > s.cap {
		s.ring = s.ring[len(s.ring)-s.cap:]
	}
	ch := s.dbCh
	subs := make([]chan LogLine, 0, len(s.subs))
	for sub := range s.subs {
		subs = append(subs, sub)
	}
	s.mu.Unlock()

	// Async DB write (non-blocking; drops if channel is full under extreme load).
	if ch != nil {
		select {
		case ch <- dbWriteReq{seq: l.Seq, ll: l}:
		default:
		}
	}

	for _, sub := range subs {
		select {
		case sub <- l:
		default:
		}
	}
}

// recent returns ring lines with Seq > since (capped to limit), and the latest seq.
func (s *logSinkT) recent(since int64, limit int) (lines []LogLine, cursor int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cursor = s.seq
	if limit <= 0 || limit > s.cap {
		limit = s.cap
	}
	for _, l := range s.ring {
		if l.Seq > since {
			lines = append(lines, l)
		}
	}
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines, cursor
}

func (s *logSinkT) subscribe() (<-chan LogLine, func()) {
	ch := make(chan LogLine, 256)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subs, ch)
			s.mu.Unlock()
			close(ch)
		})
	}
}

// parseLog pulls a level + [tag] out of a standard-logger line; the timestamp is
// the capture time (RFC3339).
func parseLog(line string) LogLine {
	msg := line
	// strip the "2006/01/02 15:04:05" LstdFlags prefix if present
	if len(msg) >= 20 && msg[4] == '/' && msg[7] == '/' && msg[10] == ' ' {
		msg = strings.TrimSpace(msg[19:])
	}
	tag := ""
	if strings.HasPrefix(msg, "[") {
		if i := strings.IndexByte(msg, ']'); i > 1 {
			tag = msg[1:i]
		}
	}
	return LogLine{TS: time.Now().Format(time.RFC3339), Level: levelOf(msg), Tag: tag, Text: msg}
}

// levelOf recognizes current English/Korean and historical Chinese log wording.
// Recognition never rewrites the stored log text or event protocol fields.
func levelOf(msg string) string {
	low := strings.ToLower(msg)
	for _, k := range []string{"fatal", "panic", "error", "err:", "failed", "discarded", "rejected", "unreachable", "실패", "버림", "거부", "도달할 수 없", "失败", "丢弃", "拒绝", "✕", "不可达"} {
		if strings.Contains(low, k) {
			return "error"
		}
	}
	for _, k := range []string{"warn", "disabled", "retry", "비활성", "건너뜀", "중지", "재시도", "禁用", "skip", "stopped", "⚠", "重试"} {
		if strings.Contains(low, k) {
			return "warn"
		}
	}
	return "info"
}
