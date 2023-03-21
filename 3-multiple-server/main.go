package main

import (
	"flag"
	"fmt"
	"github.com/lackone/go-tcp-server-study/2-epoll-server/epoll"
	"github.com/libp2p/go-reuseport"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
)

var c int

func init() {
	flag.IntVar(&c, "c", 10, "启用多少个协程")
	flag.Parse()
}

func main() {

	go func() {
		if err := http.ListenAndServe(":8888", nil); err != nil {
			log.Fatalln(err)
		}
	}()

	for i := 0; i < c; i++ {
		go startEpoll()
	}

	select {}
}

func startEpoll() {
	//socket 端口复用
	//在 Linux 中, 我们可以调用 setsockopt 将 SO_REUSEADDR 和 SO_REUSEPORT 选项置为 1 以启用地址和端口复用.
	//cfg := net.ListenConfig{
	//	Control: func(network, address string, c syscall.RawConn) error {
	//		return c.Control(func(fd uintptr) {
	//			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	//			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	//		})
	//	},
	//}
	//tcp, err := cfg.Listen(context.Background(), "tcp", ":8080")
	listen, err := reuseport.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln(err)
	}

	newEpoll, err := epoll.NewEpoll()
	if err != nil {
		log.Fatalln(err)
	}

	go run(newEpoll)

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

func run(epoll *epoll.Epoll) {
	buf := make([]byte, 1024)
	for {
		//等待有事件发生的连接
		conns, err := epoll.Wait()
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
					err = epoll.Del(conn)
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
