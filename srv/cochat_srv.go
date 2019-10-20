package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Message struct {
	name    string
	message string
}
type Client struct {
	address  string
	conn     net.Conn
	name     string
	sendChan chan Message
}

var (
	muConnections = sync.Mutex{}
	connections   = make(map[string]*Client)
	brChan        = make(chan Message, 10000)
	wg            = sync.WaitGroup{}
)

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
	go sendAll()
	wg.Add(1)
	go MainLoop(listener)
	wg.Wait()
}

func MainLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			brChan <- Message{
				name:    "Server",
				message: "Server shut down.",
			}
			time.Sleep(time.Second)
			fmt.Printf("after accept: %v\n", err)
			wg.Done()
			break
		}
		connectionAddress := conn.RemoteAddr().String()
		muConnections.Lock()
		connections[connectionAddress] = &Client{address: connectionAddress, conn: conn, sendChan: make(chan Message, 10)}
		go connections[connectionAddress].handleConnection(conn, listener)
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
			wg.Add(1)
			client.sendChan <- mess
			go func(client *Client) {
				mess := <-client.sendChan
				_, err := fmt.Fprintf(client.conn, "%v\n", &mess)
				if err != nil {
					client.deleteConnection()
				}
				wg.Done()
			}(client)
		}
		muConnections.Unlock()
	}
}

func (client *Client) handleConnection(conn net.Conn, listener net.Listener) {
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
	}
	defer conn.Close()
	client.name = name
	fmt.Printf("%v %v connected\n", name, addr)
	brChan <- Message{name, "*connected*"}

	for scanner.Scan() {
		text := scanner.Text()

		if strings.ToLower(text) == "exit" {
			conn.Write([]byte("Bye\n"))
			fmt.Println(name, "disconnected")
			client.deleteConnection()
			brChan <- Message{name: name, message: "*client disconnected*"}
			break
		} else if strings.ToLower(text) == "#count" {
			muConnections.Lock()
			fmt.Fprintf(conn, "%v\n", len(connections))
			muConnections.Unlock()
		} else if strings.ToLower(text) == "#quit_server" {
			listener.Close()
		} else if strings.ToLower(text) == "#list" {
			mess := ""
			muConnections.Lock()
			for _, cli := range connections {
				mess += cli.name + " " + cli.address + "\n"
			}
			muConnections.Unlock()
			conn.Write([]byte(mess))
		} else if text != "" {
			fmt.Printf("%v enters '%v'\n", name, text)
			brChan <- Message{name: name, message: text}
		}

	}
	brChan <- Message{name, "*disconnected*"}
	fmt.Printf("%v %v disconnected", name, client.address)
}

func (client *Client) deleteConnection() {
	muConnections.Lock()
	client.conn.Close()
	delete(connections, client.address)
	muConnections.Unlock()
}
