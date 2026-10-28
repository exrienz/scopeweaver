package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/norma/llm"
)

type shutdownCompactionProvider struct {
	started   chan struct{}
	release   chan struct{}
	cancelled chan struct{}
}

func (p *shutdownCompactionProvider) Complete(ctx context.Context, _ llm.CompletionRequest) (llm.Message, string, llm.Usage, error) {
	close(p.started)
	if p.cancelled != nil {
		stop := context.AfterFunc(ctx, func() { close(p.cancelled) })
		defer stop()
	}
	<-p.release
	return llm.UserText("shutdown digest body"), "end_turn", llm.Usage{}, nil
}

func (p *shutdownCompactionProvider) Stream(ctx context.Context, _ llm.CompletionRequest) iter.Seq2[llm.StreamEvent, error] {
	return func(yield func(llm.StreamEvent, error) bool) {
		yield(llm.StreamEvent{}, ctx.Err())
	}
}

func shutdownCompactionRound(t *testing.T, s *Server, task *Task) {
	t.Helper()
	// Plan still performs production compaction maintenance before its cancelled
	// session exits; no planner response or synthetic compactor is needed.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	id, err := parseTaskID(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _ = s.agentsForTask(task).pl.Plan(ctx, id, s.m.Assets(), task.Store, task.Goal, nil, nil)
}

func waitForCompactionShutdownCleanup(t *testing.T, committed func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		done, err := committed()
		if err != nil {
			t.Errorf("cleanup compaction write: %v", err)
			return
		}
		if done {
			return
		}
		if time.Now().After(deadline) {
			t.Error("cleanup compaction write did not finish")
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func TestShutdownWaitsForMinorCompactionWrite(t *testing.T) {
	testShutdownMinorCompaction(t, false)
}

func TestShutdownCancelsCompactionProvider(t *testing.T) {
	testShutdownMinorCompaction(t, true)
}

func testShutdownMinorCompaction(t *testing.T, providerOnly bool) {
	t.Helper()
	m := backportTestManager(t)
	task, err := m.CreateTask(t.Name(), "minor compaction shutdown fixture", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.DeleteTask(task.ID, DeleteTaskOptions{}) }()
	ctx, cancel := context.WithCancel(context.Background())
	p := &shutdownCompactionProvider{started: make(chan struct{}), release: make(chan struct{})}
	if providerOnly {
		p.cancelled = make(chan struct{})
	}
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m), taskAgents: map[string]*taskAgentBundle{}, llmOn: true, llmProv: p}
	var members []int64
	for i := 0; i < 20; i++ {
		id, err := task.Store.AddNode(db.KindFact, map[string]any{"summary": fmt.Sprintf("settled result %d", i)}, 0, "confirmed", "worker", nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(members) > 0 {
			if err := task.Store.Link(members[len(members)-1], db.RelDerivedFrom, id); err != nil {
				t.Fatal(err)
			}
		}
		members = append(members, id)
	}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	released, launched, observedCancel := false, false, false
	committed := func() (bool, error) {
		covered, err := task.Store.CoveredMembers()
		return len(covered) == len(members), err
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		if !released {
			close(p.release)
		}
		// The red baseline returns from DrainChats early. Wait for its actual
		// write before task deletion or Manager.Close can remove its database.
		if launched && !observedCancel {
			waitForCompactionShutdownCleanup(t, committed)
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	// Production thresholds are N=20 and R=6: the first round stamps cold
	// nodes and the seventh makes the connected block eligible.
	for i := 0; i < 7; i++ {
		shutdownCompactionRound(t, s, task)
	}
	select {
	case <-p.started:
		launched = true
	case <-time.After(5 * time.Second):
		t.Fatal("production minor compaction never called the provider")
	}
	if providerOnly {
		cancel()
		select {
		case <-p.cancelled:
			observedCancel = true
		case <-time.After(time.Second):
			t.Error("compaction provider did not observe server shutdown")
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
		err := s.DrainChats(drainCtx)
		stop()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("drain returned while compaction provider could still write: %v", err)
		}
		close(p.release)
		released = true
		drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Fatal(err)
		}
		digests, err := task.Store.ActiveDigests()
		if err != nil || len(digests) != 0 {
			t.Errorf("cancelled provider persisted a digest: %v err=%v", digests, err)
		}
		return
	}
	if _, err := lock.Exec(`LOCK TABLE exploration_edges IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	close(p.release)
	released = true
	waitForShutdownTaskWrite(t, m, "%INSERT INTO exploration_edges(%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with minor compaction AddDigest still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	// Check immediately after drain; only the deferred cleanup may poll.
	digests, err := task.Store.ActiveDigests()
	if err != nil || len(digests) != 1 {
		t.Fatalf("minor digest not committed at drain completion: digests=%v err=%v", digests, err)
	}
	var payload struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(digests[0].Payload, &payload); err != nil || payload.Body != "shutdown digest body" {
		t.Fatalf("minor digest payload=%s err=%v", digests[0].Payload, err)
	}
	covered, err := task.Store.CoveredMembers()
	if err != nil || len(covered) != len(members) {
		t.Fatalf("minor coverage at drain completion=%v err=%v", covered, err)
	}
	for _, id := range members {
		if covered[id] != digests[0].ID {
			t.Errorf("member %d not covered by committed digest %d: %v", id, digests[0].ID, covered)
		}
	}
}

func TestShutdownWaitsForMajorCompactionWrite(t *testing.T) {
	m := backportTestManager(t)
	task, err := m.CreateTask(t.Name(), "major compaction shutdown fixture", nil, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = m.DeleteTask(task.ID, DeleteTaskOptions{}) }()
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{ctx: ctx, m: m, engine: NewEngine(m), taskAgents: map[string]*taskAgentBundle{}}
	var digests []int64
	// Eight active digests trigger the production major threshold. Their source
	// nodes have not aged R rounds, so all stale signatures must be superseded.
	for i := 0; i < 8; i++ {
		member, err := task.Store.AddNode(db.KindFact, map[string]any{"summary": "old source"}, 0, "confirmed", "worker", nil)
		if err != nil {
			t.Fatal(err)
		}
		id, err := task.Store.AddDigest(map[string]any{"body": "old digest", "signature": fmt.Sprintf("stale-%d", i), "generation": 1}, []int64{member})
		if err != nil {
			t.Fatal(err)
		}
		digests = append(digests, id)
	}
	lock, err := m.pg.BeginTx(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	launched := false
	committed := func() (bool, error) {
		var count int
		err := m.pg.QueryRow(`SELECT count(*) FROM exploration_nodes
			WHERE exploration_id=$1 AND kind='digest' AND state='superseded'`, task.ExpID).Scan(&count)
		return count == len(digests), err
	}
	defer func() {
		cancel()
		_ = lock.Rollback()
		if launched {
			waitForCompactionShutdownCleanup(t, committed)
		}
		drainCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := s.DrainChats(drainCtx); err != nil {
			t.Errorf("cleanup drain: %v", err)
		}
	}()
	if _, err := lock.Exec(`LOCK TABLE exploration_edges IN SHARE MODE`); err != nil {
		t.Fatal(err)
	}
	shutdownCompactionRound(t, s, task)
	launched = true
	waitForShutdownTaskWrite(t, m, "DELETE FROM exploration_edges%")
	cancel()
	drainCtx, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	err = s.DrainChats(drainCtx)
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("drain returned with major compaction SupersedeDigests still using the database: %v", err)
	}
	if err := lock.Rollback(); err != nil {
		t.Fatal(err)
	}
	drainCtx, stop = context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := s.DrainChats(drainCtx); err != nil {
		t.Fatal(err)
	}
	for _, id := range digests {
		node, err := task.Store.GetNode(id)
		if err != nil || node == nil || node.State != db.StateDigestSuperseded {
			t.Fatalf("major digest %d not superseded at drain completion: node=%+v err=%v", id, node, err)
		}
		members, err := task.Store.DigestMembers(id)
		if err != nil || len(members) != 0 {
			t.Fatalf("superseded digest %d retains covers edges: %v err=%v", id, members, err)
		}
	}
	active, err := task.Store.ActiveDigests()
	if err != nil || len(active) != 0 {
		t.Fatalf("major still has active digests: %v err=%v", active, err)
	}
}
