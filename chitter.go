package main

import (
	"fmt"
	"net"
	"os"
	//	"strconv"
)

type Client struct {
	conn net.Conn
	id   int
	ch   chan<- string
}

func handleConnection(con net.Conn, msgchan chan string, client chan Client) {
	ch := make(chan string)
	client <- Client{con, -1, ch}
	buffer := make([]byte, 4096)
	go func() {
		for {
			n, err := con.Read(buffer)
			if err != nil {
				//TODO handle error
				fmt.Println(err)
				con.Close()
				break
			}

			msgchan <- string(buffer[0:n])
			fmt.Println("after msgchan")

			/*		msg := <-ch
					n, err = con.Write([]byte(msg))
					if err != nil {
						fmt.Println(err)
						con.Close()
						break

					}
			*/

		}
	}()

	go func() {
		for {
			msg := <-ch
			_, err := con.Write([]byte(msg))
			if err != nil {
				fmt.Println(err)
				con.Close()
				break
			}
		}
	}()

}

func handleMsg(msgchan <-chan string, addclient <-chan Client) {
	num := 1
	clients := make(map[net.Conn]chan<- string)
	for {
		select {
		case msg := <-msgchan:
			for _, ch := range clients {
				go func(mch chan<- string) { mch <- msg }(ch)

			}
		case client := <-addclient:
			client.id = num
			clients[client.conn] = client.ch
			num++
		}
	}

}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: chitter [port]")
		return
	}

	/* Read specific port. */
	port := os.Args[1]
	//	if strconv.Atoi(port) > 65535 || strconv.Atoi(port) < 1023 {
	//		fmt.Println("Port range from 1024 - 65535")
	//		return
	//	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("cannot listen on " + port + ", please change to another port")
		fmt.Println(err)
		return
	}

	addchan := make(chan Client)

	publicMessages := make(chan string, 10)
	go handleMsg(publicMessages, addchan)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}

		go handleConnection(conn, publicMessages, addchan)
	}
}
