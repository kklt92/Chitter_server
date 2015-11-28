package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)


func helpInfo() string{
  helpInfo := "Usage:\n" +        
  "/rooms\t\t\t\tDisplay active rooms.\n" + 
  "/join <room_name>\t\tJoin chat room.\n" +
  "/leave\t\t\t\tLeave current chat room.\n" +
  "/quit\t\t\t\tQuit.\n" +
  "/tell <user_name> <message>\tSend private message to target user.\n" +
  "/reply <message>\t\tReply to last user sent you private message.\n" +
  "/help\t\t\t\tDisplay this help infomation.\n" +
  "/whoami\t\t\t\tDisplay user infomation.\n" + 
  "<message>\t\t\tSend message to chat room, seen by all user in same chat room.\n"

  return helpInfo
}


/**
* each client has a Client.
*/
type Client struct {
  conn net.Conn      // connection info
  id   int           // user ID
  name string        // username
  room *Room          // chat room
	ch   chan<- string // output channel
  replyTo *Client    // reply to last person send you a Tell
}

/**
 * construct message. Using a specific format
 */
type Msg struct {
	msg string
	src string
	dst string
}

type Cmd struct {
  command string
  src     string
  dst     string
  msg     *Msg
}

/**
 * 
 */
type Room struct {
  name string     // room name
  creator  *Client // room creator/manager
  clients  map[string]*Client  // online clients in current room
}

func NewRoom(name string) *Room {
  if len(name) <= 0 {
    return nil
  }
  r := new(Room)
  r.name = name
  r.creator = nil
  r.clients = make(map[string]*Client)
  return r
}

/**
 *
 */
type Server struct {
  name string                 // chat server name
  rooms map[string]*Room       // map: key->room name, value->room reference
  clients map[string]*Client   // map: key->client name, value->client reference
  pre_select map[string]bool   // 
  addchan chan *Client
  rmchan  chan *Client
  msgchan chan Msg
  cmdchan chan Cmd
}

func NewServer(name string) *Server {
  if len(name) <= 0 {
    name = "XYZ"
  }
  s := new(Server)
  s.name = name
  s.rooms = make(map[string]*Room)
  s.clients = make(map[string]*Client)
	s.addchan = make(chan *Client, 10)
	s.rmchan = make(chan *Client, 10)
	s.msgchan = make(chan Msg, 10)
  s.cmdchan = make(chan Cmd, 10)

  return s
}


/**
 * handle Connection.
 * Add client to Client channel. Remove it as well when disconnected
 */
func handleConnection(con net.Conn, id int, s *Server) {
	welcome := "<= Welcome to " + s.name + " chat server.\n"
	_, err := con.Write([]byte(welcome))
	if err != nil {
		fmt.Println(err)
		con.Close()
	}

	buffer := make([]byte, 4096)
  
  login_name := "<= Login Name?\n"
	_, err = con.Write([]byte(login_name))
	if err != nil {
		fmt.Println(err)
		con.Close()
  }

  client_name := ""
  ch := make(chan string)
  client := Client{con, id, client_name, nil ,ch, nil}

  for {
    con.Write([]byte("=> "))
    n, err := con.Read(buffer)
    if err != nil {
      fmt.Println(err)
      con.Close()
    }

    client_name = string(buffer[0:n])
    client_name = client_name[0: len(client_name)-2]
    if !isValidName(client_name) {
      login_name = "<= Sorry, " + client_name + " is not valid, please only contains a-zA-Z0-9 or _ or -\n"
      _, err := con.Write([]byte(login_name))
      if err != nil {
        fmt.Println(err)
        con.Close()
      }
      continue
    }

    client.name = client_name
    s.addchan <- &client
    addClient := <- ch
    if addClient == "YES" {
      login_name = "<= Welcome " + client_name + "!\n"
      _, err := con.Write([]byte(login_name))
      if err != nil {
        fmt.Println(err)
        con.Close()
      }
      break
    } else {
      login_name = "<= Sorry, " + client_name + " has been taken, please choose another name\n"
      _, err := con.Write([]byte(login_name))
      if err != nil {
        fmt.Println(err)
        con.Close()
      }
    }
  }


  defer func() {
    fmt.Printf("Connection from %v closed.\n", con.RemoteAddr())
    s.rmchan <- &client
  }()

  /**
  * always read from connection.
  */
  go func() {
    con.Write([]byte(helpInfo()))
    for {
      con.Write([]byte("=> "))
      n, err := con.Read(buffer)
      if err != nil {
        fmt.Println(err)
        con.Close()
        break
      }

      str := strings.TrimSpace(string(buffer[0:n]))
      if len(str) == 0 {
      } else if str[0] == '/' {
        cmd := parseCMD(str[1:len(str)], &client)
        s.cmdchan <- cmd
      } else {
        msg := parseMsg(str, client_name)
        s.msgchan <- formatMsg(msg)
      }
    }
  }()

  // Allways read from its own message channel
  for {
    msg := <-ch
    msg = "\n<= " + msg
    if msg[len(msg)-1] != '\n' {
      msg = msg + "\n"
    }
    _, err := con.Write([]byte(msg))
    if err != nil {
      fmt.Println(err)
      con.Close()
      break
    }
  }
}

func isValidName(name string) bool {
  if len(name) == 0 {
    return false
  }
  for i:=0; i<len(name); i++ {
    if name[i] >= 'a' && name[i] <= 'z' {
      continue
    } else if name[i] >= 'A' && name[i] <= 'Z' {
      continue
    } else if name[i] >= '0' && name[i] <= '9' {
      continue
    } else if name[i] == '_' || name[i] == '-' {
      continue
    } else {
      return false
    }
  }
  return true
}

// convert message into standard  Msg format
func formatMsg(msg Msg) Msg {
  i := 0
  for ; msg.msg[i] == ' '; i++ {
  }
  msg.msg = msg.msg[i:]
  return msg
}

func parseCMD(str string, sender *Client) Cmd {
  cmd := Cmd{"", sender.name, "", nil}

  firstWordIndex := strings.Index(str, " ")
  firstWord := str[0:]
  if firstWordIndex != -1 {
    firstWord = str[0:firstWordIndex]
  }

  switch firstWord {
  case "join":
    cmd.dst = strings.TrimSpace(str[firstWordIndex:])
    cmd.command = "join"
  case "leave":
    cmd.command = "leave"
  case "rooms":
    cmd.command = "rooms"
  case "quit":
    cmd.command = "quit"
  case "tell", "t", "w":
    if firstWordIndex == -1 || firstWordIndex >= len(str) {
      errCMD(&cmd, "USAGE: " + firstWord + " <user_name> <message>\n")
      break
    }
    remainStr := strings.TrimSpace(str[firstWordIndex:])
    firstWordIndex = strings.Index(remainStr, " ")
    if firstWordIndex == -1 || firstWordIndex >= len(remainStr) {
      errCMD(&cmd, "USAGE: " + firstWord + " <user_name> <message>\n")
      break
    }
    cmd.dst = remainStr[0:firstWordIndex]
    msg := strings.TrimSpace(remainStr[firstWordIndex:])
    cmd.command = "tell"
    cmd.msg = new(Msg)
    cmd.msg.msg = msg
    cmd.msg.src = cmd.src
    cmd.msg.dst = cmd.dst
  case "reply", "r":
    if sender.replyTo == nil {
      errCMD(&cmd, "Cannot find last user sent you private message\n")
      break
    }
    cmd.command = "tell"
    remainStr := strings.TrimSpace(str[firstWordIndex:])
    cmd.msg = new(Msg)
    cmd.dst = sender.replyTo.name
    cmd.src = sender.name
    cmd.msg.src = cmd.src
    cmd.msg.dst = cmd.dst
    cmd.msg.msg = remainStr
  case "help", "h":
    cmd.command = "help"
    cmd.msg = new(Msg)
    cmd.msg.msg = helpInfo()

  case "whoami":
    cmd.command = "whoami"
    cmd.dst = cmd.src
    cmd.msg = new(Msg)
    
  default:
    cmd.command = "error"
    cmd.src = sender.name
    cmd.dst = sender.name
    errCMD(&cmd, "Cannot find command: " + firstWord + ", please type '/help' find help\n")
  }

  return cmd
}

func errCMD(cmd *Cmd, err string) {
  cmd.command = "error"
  if cmd.msg == nil {
    cmd.msg = new(Msg)
  }
  cmd.msg.msg = err
  cmd.dst = cmd.src
  cmd.msg.dst = cmd.src
  cmd.msg.src = cmd.src
}

// Parse message and handle the command.
func parseMsg(msg string, sender string) Msg {
  message := Msg{"", "", ""}
	message.dst = "all"
	message.src = sender
	message.msg = msg
	return message
}

func getClientInfo(client *Client) string {
  var currRoomName string
  if client.room == nil {
    currRoomName = ""
  } else {
    currRoomName = client.room.name
  }
  info := "Name:\t\t" + client.name + "\n" + 
           "Current room:\t\t" + currRoomName + "\n"
          //+ "Connection info:\t" + string(client.conn.RemoteAddr()) + "\n"

  return info
}

func createRoom(client *Client, roomName string, s *Server) *Room {
  if s.rooms[roomName] != nil {
    // exsiting room name
    return nil
  }
  r := NewRoom(roomName)
  r.creator = client
  r.clients[client.name] = client
  s.rooms[roomName] = r
  return r
}

// Send message to different channel
func (s *Server) HandleMsg() {
	for {
		select {
		// When central channel has message.
		case msg := <-s.msgchan:
			if msg.dst == "all" {
        sender, ok := s.clients[msg.src]
        if !ok {
          // TODO handle not ok
        }
        if sender.room != nil {
          for _, client := range sender.room.clients {
            go func(mch chan<- string) { mch <- "[" + msg.src +"]" + ": " + msg.msg }(client.ch)
          }
        } else {
          // TODO when sender is not in a chat room
        }
      } else {
				dst:= msg.dst
				_, ok := s.clients[dst]
				if ok {
          sender := s.clients[msg.src]
					client := s.clients[dst]
          client.replyTo = sender
					go func(mch chan<- string) { mch <- "["+msg.src + "]: " + msg.msg }(client.ch)
				} else {
					src := msg.src
					client := s.clients[src]
					go func(mch chan<- string) { mch <- "Sorry, target user is offline\n" }(client.ch)
				}
			}
    case cmd := <- s.cmdchan:
      s.handleCMD(cmd)

		// When add client to client channel
		case client := <-s.addchan:
      if s.clients[client.name] == nil {
        s.clients[client.name] = client
        client.ch <- "YES"
      } else {
        client.ch <- "NO"
      }
    // When remove client from client channel
    case client := <-s.rmchan:
      fmt.Printf("Client %v disconnected\n", client.conn.RemoteAddr())
      delete(s.clients, client.name)
      client.conn.Close()
    }
  }
}

func addToRoom(client *Client, room *Room) {
  if client.room != nil {
    removeFromRoom(client)
  }
  client.room = room
  room.clients[client.name] = client
}

func removeFromRoom(client *Client) {
  room := client.room
  if room != nil {
    delete(room.clients, client.name)
    client.room = nil
  }
}

func (s *Server) handleCMD(cmd Cmd) {
  sender := s.clients[cmd.src]
  switch cmd.command {
  case "join":
    room := s.rooms[cmd.dst]
    if room == nil {
      go func(mch chan<- string) { mch <- "Cannot find room: " + cmd.dst +"\n" }(sender.ch)
      break
    }
    addToRoom(sender, room)
    replyStr := "Entering room: " + room.name + "\n" 
    for clientName, client := range room.clients {
      replyStr += "* " + clientName + "\n"
      go func(mch chan<- string) { mch <- "* new user joined "+room.name+ ": " + sender.name + "\n"}(client.ch)
    }
    replyStr += "End of list.\n"
    go func(mch chan<- string) { mch <- replyStr }(sender.ch)
  case "leave":
    room := sender.room
    if room == nil {
      go func(mch chan<- string) { mch <- "You are not in a room\n" }(sender.ch)
      break
    }
    removeFromRoom(sender)
    for _, client := range room.clients {
      go func(mch chan<- string) { mch <- "User has left "+room.name+": " + sender.name+"\n" }(client.ch)
    }
  case "rooms":
    replyStr := "Active rooms are:\n"
    for roomName, room := range s.rooms {
      replyStr +=  " * " + roomName+" (" +strconv.Itoa(len(room.clients)) + ")\n" 
    }
    replyStr += "End of list\n"
    go func(mch chan<- string) { mch <- replyStr}(sender.ch)
  case "quit":
    go func(mch chan<- string) { mch <- "BYE\n"}(sender.ch)
    removeFromRoom(sender)
    s.rmchan <- sender
  case "tell":
    s.msgchan  <- *cmd.msg 

  case "help":
    go func(mch chan<- string) { mch <- cmd.msg.msg + "\n"}(sender.ch)
    
  case "whoami":
    go func(mch chan<- string) { mch <- getClientInfo(sender) + "\n"}(sender.ch)
    

  case "error":
    go func(mch chan<- string) { mch <- "Error: " + cmd.msg.msg + "\n"}(sender.ch)
    
  default:
    fmt.Println("Cannot find case: " + cmd.command)
  }
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		fmt.Println("Usage: chitter [port] [server_name]")
		return
	}

	// Read specific port.
	port, _ := strconv.Atoi(os.Args[1])

	if port > 65535 || port < 1023 {
		fmt.Println("Port range from 1024 - 65535")
		return
	}

	ln, err := net.Listen("tcp", ":"+os.Args[1])
	if err != nil {
		fmt.Println("cannot listen on " + os.Args[1] + ", please change to another port")
		fmt.Println(err)
		return
	}

  server_name := os.Args[2]

  s := NewServer(server_name)
	go s.HandleMsg()

  rootUser := Client{nil, 0, "root", nil, nil, nil}
  s.clients[rootUser.name] = &rootUser
  createRoom(&rootUser, "chat", s)
  createRoom(&rootUser, "hothub", s)

	num := 1
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
		}

		num++
		go handleConnection(conn, num, s)
	}
}
