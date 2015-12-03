NU Chitter Server
==================

It is a chatroom with private message support. Implemented in Go.

Usage
-----------------
1. Server side:

  Running server directly:
  
  `go run chitter.go [port] [chat_room name]`

  Or compile it and run:
  
  `go build chitter.go`
  
  `./chitter [port] //port should be larger than 1024.`

2. Client side:
  
  using nc:

  `nc server_ip_address port`
  
  or using telnet:
  
  `telnet server_ip_address port`

