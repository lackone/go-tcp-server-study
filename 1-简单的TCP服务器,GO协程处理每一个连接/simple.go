package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	// 保存所有的TCP连接
	var connections = make([]net.Conn, 0)

	//监听8080端口
	listen, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		//遍历所有连接，并关闭
		for _, c := range connections {
			c.Close()
		}
	}()

	for {
		//接受客户端的连接请求
		conn, err := listen.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		//把连接加入connections中
		connections = append(connections, conn)

		//每一个连接过来，都会创建一个GO协程来处理，连接上的读写操作
		go handleConn(conn)
	}
}

// 每一个连接过来，都会创建一个GO协程来处理，连接上的读写操作
// 这种模式有一个问题，就是短时间内如果有大量的连接请求，GO协程的数量会爆涨，导致GO语言本身的调度压力很大
// 如果是读写分离的情况下，一个连接对应二个GO协程，协程数量会更多。
func handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	for {
		data, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			if err == io.EOF {
				return
			}
		}
		fmt.Println("client msg :", strings.Trim(data, "\r\n"))

		conn.Write([]byte("server msg :" + data))
	}
}
