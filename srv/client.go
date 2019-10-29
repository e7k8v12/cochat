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
	_, err := client.conn.Write([]byte(mess + "\n"))
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

	go client.sendFromChan(server)

	fmt.Printf("%v %v connected\n", name, client.conn.RemoteAddr().String())

	history := server.database.getHistory()
	if history != "" {
		err := client.Write(history)
		if err != nil {
			fmt.Println(err)
		}
	}

loopScanner:
	for scanner.Scan() {
		text := scanner.Text()

		lowerText := strings.ToLower(text)
		switch {
		case lowerText == "#exit":
			err := client.Write("Bye")
			printError(err)
			fmt.Println(name, "disconnected")
			server.deleteConnection(client)
			break loopScanner
		case lowerText == "#count":
			server.muConnections.Lock()
			err := client.Write(strconv.Itoa(len(server.connections)))
			printError(err)
			server.muConnections.Unlock()
		case lowerText == "#list":
			mess := ""
			server.muConnections.Lock()
			for _, cli := range server.connections {
				mess += cli.name + " " + cli.address
			}
			server.muConnections.Unlock()
			err := client.Write(mess)
			printError(err)
		case lowerText == "#history":
			mess := server.database.getHistory()
			err := client.Write(mess)
			printError(err)
		case text != "":
			fmt.Printf("%v enters '%v'\n", name, text)
			mess := Message{name: name, message: text}
			server.brChan <- mess
			server.database.addMessage(mess)
		}
	}
	fmt.Printf("%v %v disconnected\n", name, client.address)
}

func (client *Client) checkConnection() error {
	err := client.Write("##test message##")
	return err
}

func (client *Client) sendFromChan(server *Server) {
	for mess := range client.sendChan {
		err := client.Write(mess.String())
		server.wgBroadcastSend.Done()
		printError(err)
	}
}
