package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	muConnections       sync.Mutex
	connections         map[string]*Client
	brChan              chan Message
	wgListenConnections sync.WaitGroup
	database            *Database
	ticker              *time.Ticker
	listener            net.Listener
	signals             chan os.Signal
	wgBroadcastSend     sync.WaitGroup
}

func (server *Server) deleteConnection(client *Client) {
	server.muConnections.Lock()
	client.conn.Close()
	close(client.sendChan)
	delete(server.connections, client.address)
	server.muConnections.Unlock()
}

func (server *Server) listenForNewConnections() {
	for {
		conn, err := server.listener.Accept()

		if err != nil {
			server.brChan <- Message{
				name:    "Server",
				message: "Server shut down.",
			}
			fmt.Printf("Connection closed. Accept error: %v\n", err)
			server.wgBroadcastSend.Wait()
			server.wgListenConnections.Done()
			break
		}
		connectionAddress := conn.RemoteAddr().String()
		server.muConnections.Lock()
		server.connections[connectionAddress] = &Client{address: connectionAddress, conn: conn, sendChan: make(chan Message, 100)}
		go server.connections[connectionAddress].handleConnection(server)
		server.muConnections.Unlock()
	}
}

func (server *Server) initServer(srvPort string) error {
	server.muConnections = sync.Mutex{}
	server.connections = make(map[string]*Client)
	server.brChan = make(chan Message, 10000)
	server.wgListenConnections = sync.WaitGroup{}
	server.database = &Database{}
	server.ticker = time.NewTicker(time.Second * 5)
	server.signals = make(chan os.Signal, 1)
	signal.Notify(server.signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	listener, err := net.Listen("tcp", ":"+srvPort)
	server.listener = listener

	return err
}

func (server *Server) messageForwarding() {
	for mess := range server.brChan {
		server.muConnections.Lock()
		for _, client := range server.connections {
			server.wgBroadcastSend.Add(1)
			go func(client *Client, mess Message) {
				client.sendChan <- mess
			}(client, mess)
		}
		server.muConnections.Unlock()
	}
}

func (server *Server) deleteBrokenConnections() {
	for range server.ticker.C {
		server.muConnections.Lock()
		for _, client := range server.connections {
			go func(client *Client, server *Server) {
				err := client.checkConnection()
				if err != nil {
					server.deleteConnection(client)
				}
			}(client, server)
		}
		server.muConnections.Unlock()
	}
}
func (server *Server) catchSignal() {
	<-server.signals
	server.listener.Close()
}
