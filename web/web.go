package web

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/hound-search/hound/api"
	"github.com/hound-search/hound/config"
	"github.com/hound-search/hound/searcher"
	"github.com/hound-search/hound/ui"
)

// Server is an HTTP server that handles all
// http traffic for hound. It is able to serve
// some traffic before indexes are built and
// then transition to all traffic afterwards.
type Server struct {
	cfg *config.Config
	dev bool
	ch  chan error

	mux *http.ServeMux
	lck sync.RWMutex
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == s.cfg.HealthCheckURI {
		fmt.Fprintln(w, "👍")
		return
	}

	s.lck.RLock()
	defer s.lck.RUnlock()
	if m := s.mux; m != nil {
		m.ServeHTTP(w, r)
	} else {
		http.Error(w,
			"Hound is not ready.",
			http.StatusServiceUnavailable)
	}
}

func (s *Server) serveWith(m *http.ServeMux) {
	s.lck.Lock()
	defer s.lck.Unlock()
	s.mux = m
}

// Start creates a new server that will immediately start handling HTTP traffic.
// The HTTP server will return 200 on the health check, but a 503 on every other
// request until ServeWithIndex is called to begin serving search traffic with
// the given searchers.
func Start(cfg *config.Config, addr string, dev bool) *Server {
	ch := make(chan error)

	s := &Server{
		cfg: cfg,
		dev: dev,
		ch:  ch,
	}

	go func() {

		if []byte(addr)[0] != '/' {
			ch <- http.ListenAndServe(addr, s)
		}

		if _, err := os.Stat(addr); !os.IsNotExist(err) {
			os.Remove(addr)
		}

		listener, err := net.Listen("unix", addr)
		if err != nil {
			ch <- err
		}

		if err := os.Chmod(addr, 0666); err != nil {
			ch <- err
		}

		srv := &http.Server{Handler: s}

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sigs
			os.Remove(addr)
			os.Exit(0)
		}()

		ch <- srv.Serve(listener)
	}()

	return s
}

// ServeWithIndex allow the server to start offering the search UI and the
// search APIs operating on the given indexes.
func (s *Server) ServeWithIndex(idx map[string]*searcher.Searcher) error {
	h, err := ui.Content(s.dev, s.cfg)
	if err != nil {
		return err
	}

	m := http.NewServeMux()
	m.Handle("/", h)
	api.Setup(m, idx)

	s.serveWith(m)

	return <-s.ch
}
