package server

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"multithreadredis/internal/pool"
)

const CommunicationProtocol = "tcp"

type Server struct {
	listener net.Listener
	pool     *pool.Pool
}

// Listen binds addr and starts the worker pool; call Serve to accept connections.
func Listen(addr string, p *pool.Pool) (*Server, error) {
	listener, err := net.Listen(CommunicationProtocol, addr)

	if err != nil {
		return nil, err
	}

	p.Start()

	return &Server{
		listener: listener,
		pool:     p,
	}, nil
}

func (s *Server) Addr() net.Addr { return s.listener.Addr() }

// Serve accepts connections until ctx is cancelled.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()

		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}

			slog.Warn("Failed to accept connection", "error", err)
			continue
		}

		s.pool.Submit(conn)
	}
}
