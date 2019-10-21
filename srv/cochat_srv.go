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
	address  string
	conn     net.Conn
	name     string
	sendChan chan Message
}

var (
	muConnections = sync.Mutex{}
	connections   = make(map[string]*Client)
	brChan        = make(chan Message, 10000)
	wgMainLoop    = sync.WaitGroup{}
	wgSendMess    = sync.WaitGroup{}
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
	wgMainLoop.Add(1)
	go MainLoop(listener)
	wgMainLoop.Wait()
}

func MainLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			brChan <- Message{
				name:    "Server",
				message: "Server shut down.",
			}
			wgSendMess.Wait()
			fmt.Printf("Connection closed. Accept error: %v\n", err)
			wgMainLoop.Done()
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
			wgSendMess.Add(1)
			client.sendChan <- mess
			//fmt.Printf("Start send to %v message %v\n", client.name, &mess)
			go func(client *Client) {
				mess := <-client.sendChan
				//fmt.Printf("\tSending to %v message %v\n", client.name, &mess)
				_, err := fmt.Fprintf(client.conn, "%v\n", &mess)
				//_, err := client.conn.Write([]byte(mess.String() + "\n"))
				if err != nil {
					client.deleteConnection()
					//fmt.Printf("Error send to %v message %v\n", client.name, &mess)
				}
				//fmt.Printf("\t\tStop send to %v message %v\n", client.name, &mess)
				wgSendMess.Done()
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
	//brChan <- Message{name, "*connected*"}

	for scanner.Scan() {
		text := scanner.Text()

		lowerText := strings.ToLower(text)
		if lowerText == "exit" {
			conn.Write([]byte("Bye\n"))
			fmt.Println(name, "disconnected")
			client.deleteConnection()
			//brChan <- Message{name: name, message: "*client disconnected*"}
			break
		} else if lowerText == "#count" {
			muConnections.Lock()
			fmt.Fprintf(conn, "%v\n", len(connections))
			muConnections.Unlock()
		} else if lowerText == "#quit_server" {
			listener.Close()
		} else if lowerText == "#list" {
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
	//brChan <- Message{name, "*disconnected*"}
	fmt.Printf("%v %v disconnected\n", name, client.address)
}

func (client *Client) deleteConnection() {
	muConnections.Lock()
	client.conn.Close()
	delete(connections, client.address)
	muConnections.Unlock()
}
