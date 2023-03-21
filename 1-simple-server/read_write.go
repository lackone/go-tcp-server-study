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

		msg := make(chan []byte)

		//把连接加入connections中
		connections = append(connections, conn)

		//每一个连接过来，都会创建二个GO协程来处理，读写分离
		go readLoop(conn, msg)
		go writeLoop(conn, msg)
	}
}

// 读循环
func readLoop(conn net.Conn, msg chan []byte) {
	defer conn.Close()

	connReader := bufio.NewReader(conn)
	for {
		clientData, err := connReader.ReadString('\n')
		if err != nil {
			log.Println(err)
			if err == io.EOF {
				return
			}
		}

		fmt.Println("client msg :", strings.Trim(clientData, "\r\n"))

		msg <- []byte(clientData)
	}
}

// 写循环
func writeLoop(conn net.Conn, msg chan []byte) {
	for {
		select {
		case data := <-msg:
			conn.Write([]byte("server msg :" + string(data)))
		}
	}
}
