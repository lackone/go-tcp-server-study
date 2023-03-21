//go:build linux
// +build linux

package epoll

import (
	"net"
	"reflect"
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
// 使用该方法获取FD有问题，因为tcpConn.File底层调用的是c.fd.dup()，会创建一个新的文件描述符指向原来的FD，我们通过该FD向Epoll中添加或删除时，会有问题，因为FD的值变了。
func (e *Epoll) SocketFD2(conn net.Conn) int {
	//断言，转换成*net.TCPConn
	tcpConn, _ := conn.(*net.TCPConn)
	//获取底层os.File指针
	tcpConnFile, _ := tcpConn.File()
	//获取FD
	return int(tcpConnFile.Fd())
}

// 获取连接fd
func (e *Epoll) SocketFD(conn net.Conn) int {
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")
	return int(pfdVal.FieldByName("Sysfd").Int())
}

// 从epoll中删除监听事件
func (e *Epoll) Del(conn net.Conn) error {
	fd := e.SocketFD(conn)
	event := syscall.EpollEvent{
		Events: syscall.EPOLLIN | syscall.EPOLLHUP,
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
		Events: syscall.EPOLLIN | syscall.EPOLLHUP,
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
		if err == syscall.EINTR {
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
