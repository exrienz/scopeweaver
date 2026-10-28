package server

import "net/http"

// startBackgroundWork serializes descendant registration with the drain barrier.
// A rejected callback must not perform its work synchronously instead.
func (s *Server) startBackgroundWork(work func()) bool {
	done, ok := s.reserveBackgroundWork()
	if !ok {
		return false
	}
	go func() {
		defer done()
		work()
	}()
	return true
}

// reserveBackgroundWork also owns SDK callbacks whose parent may return early
// on ordinary cancellation. Shutdown alone closes their admission.
func (s *Server) reserveBackgroundWork() (func(), bool) {
	s.chatMu.Lock()
	defer s.chatMu.Unlock()
	if s.chatClosing {
		return nil, false
	}
	s.backgroundWG.Add(1)
	return s.backgroundWG.Done, true
}

// trackHTTP joins handlers that survive http.Server.Shutdown's deadline/Close.
// Requests admitted before closing may finish their synchronous database work.
func (s *Server) trackHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.chatMu.Lock()
		if s.chatClosing {
			s.chatMu.Unlock()
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		s.httpWG.Add(1)
		s.chatMu.Unlock()
		defer s.httpWG.Done()
		next.ServeHTTP(w, r)
	})
}
