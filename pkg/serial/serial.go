package serial

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

var lastClientId = atomic.Uint64{}

type connectedClient struct {
	*Client
	aliveChan chan struct{}
}

type Server struct {
	ctx        context.Context
	listener   net.Listener
	clients    map[uint64]*connectedClient
	connChan   chan net.Conn
	clientChan chan *Client
	mu         *sync.Mutex
}

func NewServer(ctx context.Context) *Server {
	return &Server{
		ctx:        ctx,
		listener:   nil,
		connChan:   make(chan net.Conn),
		clientChan: make(chan *Client),
		clients:    map[uint64]*connectedClient{},
		mu:         &sync.Mutex{},
	}
}

func (s *Server) closeClients() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Println("closing all clients ...")

	for _, client := range s.clients {
		if err := client.Close(); err != nil {
			log.Println(err)
		}
	}

	log.Println("all clients closed.")
}

func (s *Server) Close() error {
	log.Println("closing server ...")

	s.closeClients()

	err := s.listener.Close()

	s.listener = nil

	log.Println("server closed.")

	return err
}

func (s *Server) Listen(sockFile string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.listener != nil {
		return errors.New("already listening")
	}

	if err := os.Remove(sockFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	l, err := net.Listen("unix", sockFile)

	if err != nil {
		return err
	}

	s.listener = l

	return nil
}

func (s *Server) checkAlive(cc *connectedClient) {
	select {
	case <-s.ctx.Done():
		log.Println(s.ctx.Err())
		break
	case <-cc.aliveChan:
		log.Printf("client %d is dead.\n", cc.id)
		break
	}

	err := cc.Close()

	if err != nil {
		log.Println(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, cc.id)

	log.Printf("client %d is removed.\n", cc.id)
}

func (s *Server) newConnectedClient(conn net.Conn) *connectedClient {
	aliveChan := make(chan struct{})

	cc := &connectedClient{
		Client:    NewClient(lastClientId.Add(1), aliveChan, conn),
		aliveChan: aliveChan,
	}

	return cc
}

func (s *Server) acceptConns() {
	defer close(s.connChan)

	for {
		conn, err := s.listener.Accept()

		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Println(err)
			return
		}

		log.Println("client connecting ...")

		s.connChan <- conn
	}
}

func (s *Server) acceptClients() {
	defer close(s.clientChan)

	for {
		select {
		case <-s.ctx.Done():
			return
		case conn := <-s.connChan:
			if conn == nil {
				return
			}

			cc := s.newConnectedClient(conn)

			s.mu.Lock()
			s.clients[cc.id] = cc
			s.mu.Unlock()

			log.Printf("client %d connected.\n", cc.id)

			go s.checkAlive(cc)

			s.clientChan <- cc.Client

			break
		}
	}
}

func (s *Server) Start() {
	go s.acceptConns()
	go s.acceptClients()
}

func (s *Server) Accept() <-chan *Client {
	return s.clientChan
}
