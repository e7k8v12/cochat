package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"sync"
)

type Message struct {
	name    string
	message string
}
type Client struct {
	address string
	conn    net.Conn
}

var muConnections sync.Mutex = sync.Mutex{}
var connections map[string]*Client = make(map[string]*Client)
var brChan chan Message = make(chan Message, 1000)

func (mess *Message) String() string {
	return mess.name + ":\t" + mess.message
}

func main() {
	srvPort := flag.String("port", "8080", "Port of a server")
	flag.Parse()
	listener, err := CreateServer(*srvPort)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Console chat server started at %v\n", listener.Addr().String())
	MainLoop(listener)
}

func MainLoop(listener net.Listener) {
	go sendAll()
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		connectionAddress := conn.RemoteAddr().String()
		muConnections.Lock()
		connections[connectionAddress] = &Client{connectionAddress, conn}
		go connections[connectionAddress].handleConnection(conn)
		muConnections.Unlock()
	}
}

func CreateServer(srvPort string) (net.Listener, error) {
	listener, err := net.Listen("tcp", ":"+srvPort)
	return listener, err
}

func sendAll() {
	for mess := range brChan {
		muConnections.Lock()
		for _, client := range connections {
			go func(client *Client, mess *Message) {
				_, err := fmt.Fprintln(client.conn, mess)
				if err != nil {
					client.deleteConnection()
				}
			}(client, &mess)
		}
		muConnections.Unlock()
	}
}

func (client *Client) handleConnection(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			client.deleteConnection()
		}
	}()

	addr := conn.RemoteAddr().String()
	scanner := bufio.NewScanner(conn)
	name := ""
	if scanner.Scan() {
		name = scanner.Text()[len("login:"):]
		fmt.Fprintf(conn, "Hello, %v\n", name)
	}
	defer conn.Close()
	fmt.Printf("%v at %+v connected\n", name, addr)

	for scanner.Scan() {
		text := scanner.Text()

		if strings.ToLower(text) == "exit" {
			conn.Write([]byte("Bye\n\r"))
			fmt.Println(name, "disconnected")
			client.deleteConnection()
			brChan <- Message{name: name, message: "*client disconnected*"}
			break
		} else if strings.ToLower(text) == "#count" {
			muConnections.Lock()
			fmt.Fprintf(conn, "%v\n", len(connections))
			muConnections.Unlock()
		} else if text != "" {
			fmt.Println(name, "enters", text)
			brChan <- Message{name: name, message: text}
		}

	}
}

func (client *Client) deleteConnection() {
	muConnections.Lock()
	client.conn.Close()
	delete(connections, client.address)
	muConnections.Unlock()
}
