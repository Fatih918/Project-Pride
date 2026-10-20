package server

import (
	"context"
	"fmt"
	"net"

	"pride-ai-module/execution"
	aiv1 "pride-ai-module/proto"
	grpcadapter "pride-ai-module/transport/grpc"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

func NewGRPCServer(
	executor *execution.Executor,
	address string,
) (*GRPCServer, error) {
	if executor == nil {
		return nil, fmt.Errorf(
			"executor must not be nil",
		)
	}

	if address == "" {
		return nil, fmt.Errorf(
			"server address must not be empty",
		)
	}

	listener, err := net.Listen(
		"tcp",
		address,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"listen on %s: %w",
			address,
			err,
		)
	}

	grpcServer := grpc.NewServer()

	aiServer := grpcadapter.NewServer(
		executor,
	)

	aiv1.RegisterAIServiceServer(
		grpcServer,
		aiServer,
	)

	return &GRPCServer{
		server:   grpcServer,
		listener: listener,
	}, nil
}

func (s *GRPCServer) Serve() error {
	if s == nil {
		return fmt.Errorf(
			"grpc server must not be nil",
		)
	}

	return s.server.Serve(
		s.listener,
	)
}

func (s *GRPCServer) Address() string {
	if s == nil || s.listener == nil {
		return ""
	}

	return s.listener.Addr().String()
}

func (s *GRPCServer) Shutdown(
	ctx context.Context,
) error {
	if s == nil || s.server == nil {
		return nil
	}

	if ctx == nil {
		return fmt.Errorf(
			"context must not be nil",
		)
	}

	done := make(chan struct{})

	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil

	case <-ctx.Done():
		s.server.Stop()

		return ctx.Err()
	}
}

func (s *GRPCServer) Stop() {
	if s == nil || s.server == nil {
		return
	}

	s.server.Stop()
}

func (s *GRPCServer) ListenerAddr() net.Addr {
	if s == nil || s.listener == nil {
		return nil
	}

	return s.listener.Addr()
}
