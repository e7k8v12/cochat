package main

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Client struct {
	address  string
	conn     net.Conn
	name     string
	sendChan chan Message
	mu       sync.Mutex
}

func (client *Client) Write(mess string) error {
	client.mu.Lock()
	_, err := client.conn.Write([]byte(mess))
	client.mu.Unlock()
	return err
}

func (client *Client) handleConnection(server *Server) {

	scanner := bufio.NewScanner(client.conn)
	name := ""
	if scanner.Scan() {
		name = scanner.Text()[len("login:"):]
	}
	defer func() {
		if client.conn != nil {
			client.conn.Close()
		}
	}()
	client.name = name
	client.mu = sync.Mutex{}
	fmt.Printf("%v %v connected\n", name, client.conn.RemoteAddr().String())
	//brChan <- Message{name, "*connected*"}

	history := server.database.getHistory()
	err := client.Write(history)
	if err != nil {
		fmt.Println(err)
	}

loopScanner:
	for scanner.Scan() {
		text := scanner.Text()

		lowerText := strings.ToLower(text)
		switch {
		case lowerText == "#exit":
			err := client.Write("Bye\n")
			printError(err)
			fmt.Println(name, "disconnected")
			server.deleteConnection(client)
			//brChan <- Message{name: name, message: "*client disconnected*"}
			break loopScanner
		case lowerText == "#count":
			server.muConnections.Lock()
			err = client.Write(strconv.Itoa(len(server.connections)) + "\n")
			printError(err)
			server.muConnections.Unlock()
		case lowerText == "#list":
			mess := ""
			server.muConnections.Lock()
			for _, cli := range server.connections {
				mess += cli.name + " " + cli.address + "\n"
			}
			server.muConnections.Unlock()
			err = client.Write(mess)
			printError(err)
		case lowerText == "#history":
			mess := server.database.getHistory()
			err = client.Write(mess)
			printError(err)
		case text != "":
			fmt.Printf("%v enters '%v'\n", name, text)
			mess := Message{name: name, message: text}
			server.brChan <- mess
			server.database.addMessage(mess)
		}
	}
	//brChan <- Message{name, "*disconnected*"}
	fmt.Printf("%v %v disconnected\n", name, client.address)
}

func (client *Client) checkConnection() error {
	err := client.Write("##test message##\n")
	return err
}
