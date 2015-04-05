package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Client struct {
	conn net.Conn
	id   int
	ch   chan<- string
}

type Msg struct {
	msg string
	src string
	dst string
}

func handleConnection(con net.Conn, id int, msgchan chan Msg,
	addclient chan Client, rmclient chan Client) {
	ch := make(chan string)
	client := Client{con, id, ch}
	addclient <- client
	buffer := make([]byte, 4096)

	welcome := "Welcome to chatroom, your id is " + strconv.Itoa(id) + ".\n"
	_, err := con.Write([]byte(welcome))
	if err != nil {
		fmt.Println(err)
		con.Close()
	}
	go func() {
		for {
			n, err := con.Read(buffer)
			if err != nil {
				fmt.Println(err)
				con.Close()
				break
			}

			msg := parseMsg(string(buffer[0:n]), id)

			msgchan <- formatMsg(msg)
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

func formatMsg(msg Msg) Msg {
	i := 0
	for ; msg.msg[i] == ' '; i++ {
	}
	msg.msg = msg.msg[i:]
	return msg
}

func parseMsg(msg string, id int) Msg {
	message := Msg{"", "", ""}
	i := 0
	for i = 0; msg[i] >= '0' && msg[i] <= '9'; i++ {
	}
	for ; msg[i] == ' '; i++ {
	}
	if msg[i] == ':' {
		message.dst = msg[0:i]
		message.msg = msg[i+1:]
		message.src = strconv.Itoa(id)
		return message
	}

	if len(msg) >= 4 {
		if strings.EqualFold(msg[0:3], "ALL") {
			for i = 3; msg[i] == ' '; i++ {
			}
			if msg[i] == ':' {
				message.dst = "ALL"
				message.src = strconv.Itoa(id)
				message.msg = msg[i+1:]
				return message
			}
		}
	}

	if len(msg) >= 7 {
		if msg[0:7] == "whoami:" {
			message.dst = strconv.Itoa(id)
			message.src = "chitter"
			message.msg = strconv.Itoa(id) + "\n"
			return message
		}
	}

	message.dst = "ALL"
	message.src = strconv.Itoa(id)
	message.msg = msg
	return message
}

func handleMsg(msgchan <-chan Msg, addclient <-chan Client, rmclient <-chan Client) {
	clients := make(map[int]Client)
	for {
		select {
		case msg := <-msgchan:
			if msg.dst == "ALL" {
				for _, client := range clients {
					go func(mch chan<- string) { mch <- msg.src + ": " + msg.msg }(client.ch)
				}
			} else {
				dst, _ := strconv.Atoi(msg.dst)
				_, ok := clients[dst]
				if ok {

					client := clients[dst]
					go func(mch chan<- string) { mch <- msg.src + ": " + msg.msg }(client.ch)
				} else {
					src, _ := strconv.Atoi(msg.src)
					client := clients[src]
					go func(mch chan<- string) { mch <- "Sorry, target user is offline\n" }(client.ch)
				}
			}
		case client := <-addclient:
			clients[client.id] = client
		case client := <-rmclient:
			delete(clients, client.id)
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
	rmchan := make(chan Client)

	publicMessages := make(chan Msg, 10)
	go handleMsg(publicMessages, addchan, rmchan)

	num := 0
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}

		num++
		go handleConnection(conn, num, publicMessages, addchan, rmchan)
	}
}
