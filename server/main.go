package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"chatroom/common"
)

const timeout = 6 * time.Second // no poll for this long => client considered disconnected

type user struct {
	queue    []string
	lastSeen time.Time
}

type ChatService struct {
	mu    sync.Mutex
	users map[string]*user
}

// must hold s.mu
func (s *ChatService) broadcast(except, msg string) {
	for name, u := range s.users {
		if name != except {
			u.queue = append(u.queue, msg)
		}
	}
}

func (s *ChatService) Join(a common.JoinArgs, _ *common.Empty) error {
	if a.Username == "" {
		return errors.New("username cannot be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[a.Username]; ok {
		return fmt.Errorf("username %q is already taken", a.Username)
	}
	s.users[a.Username] = &user{lastSeen: time.Now()}
	s.broadcast(a.Username, "* "+a.Username+" joined the chat")
	log.Println(a.Username, "joined")
	return nil
}

func (s *ChatService) removeLocked(name, reason string) {
	delete(s.users, name)
	s.broadcast(name, "* "+name+" "+reason)
	log.Println(name, reason)
}

func (s *ChatService) Leave(a common.LeaveArgs, _ *common.Empty) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[a.Username]; ok {
		s.removeLocked(a.Username, "left the chat")
	}
	return nil
}

func (s *ChatService) Send(a common.SendArgs, _ *common.Empty) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[a.Username]
	if !ok {
		return errors.New("not joined")
	}
	u.lastSeen = time.Now()
	s.broadcast(a.Username, fmt.Sprintf("[%s] %s: %s", time.Now().Format("15:04:05"), a.Username, a.Text))
	return nil
}

func (s *ChatService) Poll(a common.PollArgs, r *common.PollReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[a.Username]
	if !ok {
		return errors.New("not joined")
	}
	u.lastSeen = time.Now()
	r.Messages, u.queue = u.queue, nil
	return nil
}

func (s *ChatService) List(_ common.Empty, r *common.ListReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for n := range s.users {
		r.Users = append(r.Users, n)
	}
	sort.Strings(r.Users)
	return nil
}

// removes clients that stopped polling (crash, network loss)
func (s *ChatService) reaper() {
	for range time.Tick(2 * time.Second) {
		s.mu.Lock()
		for n, u := range s.users {
			if time.Since(u.lastSeen) > timeout {
				s.removeLocked(n, "disconnected unexpectedly")
			}
		}
		s.mu.Unlock()
	}
}

func main() {
	addr := flag.String("addr", ":1234", "listen address")
	flag.Parse()

	svc := &ChatService{users: map[string]*user{}}
	if err := rpc.RegisterName("Chat", svc); err != nil {
		log.Fatal(err)
	}
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("server listening on", *addr)
	go svc.reaper()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go rpc.ServeConn(conn)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	ln.Close()
}
