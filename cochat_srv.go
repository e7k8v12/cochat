package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"sync"
)

type Message struct {
	name    string
	message string
}
type Client struct {
	address string
	conn    *net.Conn
}

var mu sync.Mutex = sync.Mutex{}
var connections map[string]*Client = make(map[string]*Client)
var brChan chan Message = make(chan Message, 1000)

func (mess *Message) String() string {
	return mess.name + ":\t" + mess.message
}

func main() {
	srvPort := flag.String("port", "8080", "Port of a server")
	flag.Parse()

	listener, err := net.Listen("tcp", ":"+*srvPort)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Console chat server started at %v\n", listener.Addr().String())
	go senAll()
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		connectionAddress := conn.RemoteAddr().String()
		//mu.Lock()
		connections[connectionAddress] = &Client{connectionAddress, &conn}
		go connections[connectionAddress].handleConnection(conn)
		//mu.Unlock()
	}
}

func senAll() {
	for mess := range brChan {
		//mu.Lock()
		for _, client := range connections {
			go func(client *Client, mess *Message) {
				_, err := fmt.Fprintln(*client.conn, mess)
				if err != nil {
					//client.conn
					delete(connections, client.address)
				}
			}(client, &mess)
		}
		//mu.Unlock()
	}
}

func (client *Client) handleConnection(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			delete(connections, client.address)
		}
	}()

	addr := conn.RemoteAddr().String()
	scanner := bufio.NewScanner(conn)
	name := ""
	if scanner.Scan() {
		name = scanner.Text()[len("login:"):]
		conn.Write([]byte("Hello, " + name + "\n\r"))
	}
	defer conn.Close()
	fmt.Printf("%v at %+v connected\n", name, addr)

	for scanner.Scan() {
		text := scanner.Text()

		if text == "Exit" {
			conn.Write([]byte("Bye\n\r"))
			fmt.Println(name, "disconnected")
			break
		} else if text != "" {
			fmt.Println(name, "enters", text)
			brChan <- Message{name: name, message: text}
		}

	}
}
