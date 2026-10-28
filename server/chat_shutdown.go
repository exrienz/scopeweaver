package server

import (
	"context"
	"errors"
	"github.com/Autumn-27/artex/sidequestion"
)

// DrainChats refuses further dispatches and waits for admitted background chats,
// including side answers, checkpoint publication and run finalization, and all
// startup/periodic DB workers and task runtimes. The caller must cancel the
// server context and initiate HTTP shutdown first, and keep the database open until
// this method returns. A deadline bounds uncooperative tools or providers.
func (s *Server) DrainChats(ctx context.Context) error {
	s.chatDrainOnce.Do(func() {
		s.chatDrainDone = make(chan struct{})
		go func() {
			// Admission locks may themselves be held by blocked database work.
			// Keep barrier acquisition inside the deadline-selectable coordinator.
			s.chatMu.Lock()
			s.chatClosing = true
			if s.engine != nil {
				s.engine.deleteMu.Lock()
				s.engine.closing = true
				s.engine.deleteMu.Unlock()
			}
			s.chatMu.Unlock()
			s.httpWG.Wait()
			if s.m != nil {
				s.m.enrich.Close()
			}
			// Side admission and run registration already serialize under commands.
			// Close that barrier before taking the completion-channel snapshot.
			var sideDone []<-chan struct{}
			if s.side != nil {
				s.side.commands.Lock()
				s.side.closing = true
				sideDone = s.cancelSideWhere(func(sidequestion.Parent) bool { return true })
				s.side.commands.Unlock()
			}
			s.chatWG.Wait()
			for _, ch := range sideDone {
				<-ch
			}
			if s.side != nil {
				<-s.side.done
			}
			s.backgroundWG.Wait()
			s.archiveWG.Wait()
			if s.engine != nil {
				// Admission is closed, so no runtime can be added after this snapshot.
				// StopTask only removes a runtime after joining its existing routines.
				s.engine.runtimeMu.Lock()
				runtimes := make([]*taskRuntime, 0, len(s.engine.runtimes))
				for _, rt := range s.engine.runtimes {
					runtimes = append(runtimes, rt)
				}
				s.engine.runtimeMu.Unlock()
				for _, rt := range runtimes {
					rt.wg.Wait()
				}
				// Bootstrap and surviving HTTP operations already own inflight reservations.
				// Reuse the task-deletion drain to join their final writes as well.
				s.engine.inflight.Range(func(taskID, _ any) bool {
					_ = s.waitTaskQuiescent(context.Background(), taskID.(string))
					return true
				})
			}
			if s.side != nil {
				// Task/SDK descendants can publish while unwinding cancellation too.
				// Flush only after every producer has finished.
				s.chatDrainErr = errors.Join(s.flushSideSnapshots(), s.flushChatContexts())
			}
			close(s.chatDrainDone)
		}()
	})
	done := s.chatDrainDone
	select {
	case <-done:
		return s.chatDrainErr
	case <-ctx.Done():
		return ctx.Err()
	}
}
