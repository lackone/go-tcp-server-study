package main

import (
	"fmt"
	"github.com/lackone/go-tcp-server-study/2-epoll-server/epoll"
	"io"
	"log"
	"net"
	"strings"
)

var newEpoll *epoll.Epoll

func main() {
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	newEpoll, err = epoll.NewEpoll()
	if err != nil {
		log.Fatalln(err)
	}

	go run()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		err = newEpoll.Add(conn)
		if err != nil {
			//如果加入epoll监控失败，则关闭连接
			log.Println(err)
			conn.Close()
		}
	}
}

func run() {
	buf := make([]byte, 1024)
	for {
		//等待有事件发生的连接
		conns, err := newEpoll.Wait()
		if err != nil {
			log.Println("epoll wait ", err)
			continue
		}
		//遍历连接，读取数据
		for _, conn := range conns {
			n, err := conn.Read(buf)
			if err != nil {
				log.Println("conn read ", err)
				if err == io.EOF {
					err = newEpoll.Del(conn)
					if err != nil {
						log.Println("epoll del ", err)
					}
					conn.Close()
					continue
				}
			}
			fmt.Println("client msg :", strings.Trim(string(buf[:n]), "\r\n"))

			conn.Write([]byte("server msg :" + string(buf[:n])))
		}
	}
}
