//go:build linux
// +build linux

package epoll

import (
	"net"
	"sync"
	"syscall"
)

type Epoll struct {
	epollFd int              //epoll的FD
	conns   map[int]net.Conn //保存监控的所有连接
	lock    sync.RWMutex     //连接锁
}

func NewEpoll() (*Epoll, error) {
	//创建一个epoll实例
	fd, err := syscall.EpollCreate(1)
	if err != nil {
		return nil, err
	}
	return &Epoll{
		epollFd: fd,
		conns:   make(map[int]net.Conn),
		lock:    sync.RWMutex{},
	}, nil
}

// 获取连接fd
func (e *Epoll) SocketFD(conn net.Conn) int {
	//断言，转换成*net.TCPConn
	tcpConn, _ := conn.(*net.TCPConn)
	//获取底层os.File指针
	tcpConnFile, _ := tcpConn.File()
	//获取FD
	return tcpConnFile.Fd()
}

// 从epoll中删除监听事件
func (e *Epoll) Del(conn net.Conn) error {
	fd := e.SocketFD(conn)
	event := syscall.EpollEvent{
		Events: syscall.EPOLLIN | syscall.EPOLLERR | syscall.EPOLLHUP,
		Fd:     int32(fd),
	}
	err := syscall.EpollCtl(e.epollFd, syscall.EPOLL_CTL_DEL, fd, &event)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	delete(e.conns, fd)

	return nil
}

// 从epoll中添加监听事件
func (e *Epoll) Add(conn net.Conn) error {
	fd := e.SocketFD(conn)
	event := syscall.EpollEvent{
		Events: syscall.EPOLLIN | syscall.EPOLLERR | syscall.EPOLLHUP,
		Fd:     int32(fd),
	}
	err := syscall.EpollCtl(e.epollFd, syscall.EPOLL_CTL_ADD, fd, &event)
	if err != nil {
		return err
	}
	e.lock.Lock()
	defer e.lock.Unlock()

	e.conns[fd] = conn

	return nil
}

// 等待事件触发
func (e *Epoll) Wait() ([]net.Conn, error) {
	events := make([]syscall.EpollEvent, 1024)
retry:
	size, err := syscall.EpollWait(e.epollFd, events, -1)
	if err != nil {
		if err == unix.EINTR {
			goto retry
		}
		return nil, err
	}

	e.lock.RLock()
	defer e.lock.RUnlock()

	conns := make([]net.Conn, 0)

	//遍历事件
	for i := 0; i < size; i++ {
		fd := int(events[i].Fd)
		conn := e.conns[fd]
		conns = append(conns, conn)
	}

	return conns, nil
}
