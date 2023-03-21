//go:build !linux
// +build !linux

package epoll

import (
	"net"
	"sync"
)

type Epoll struct {
	epollFd int
	conns   map[int]net.Conn
	lock    sync.RWMutex
}

func NewEpoll() (*Epoll, error) {
	return nil, nil
}

func (e *Epoll) SocketFD(conn net.Conn) int {
	return 0
}

func (e *Epoll) Del(conn net.Conn) error {
	return nil
}

func (e *Epoll) Add(conn net.Conn) error {
	return nil
}

func (e *Epoll) Wait() ([]net.Conn, error) {
	return nil, nil
}
