package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

//Test send and receive message
func TestGetHelloMessage(t *testing.T) {
	port := "8080"
	listener := makeServer(port)

	client := CreateClient("eugene", port)

	client.SendMessage("hello\n")

	mess := client.GetMessage()

	if mess != client.name+":\thello" {
		t.Errorf("Expect '"+client.name+":\thello', got %v\n", mess)
	}
	listener.Close()
	wgMainLoop.Wait()
}

//Test server sends disconnecting message when servers stops
func TestStopServerMessage(t *testing.T) {
	port := "8081"
	listener := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)

	listener.Close()

	//skip message about client2 connection
	client1.GetMessage()

	mess1 := client1.GetMessage()
	mess2 := client2.GetMessage()

	if !(mess1 == "Server:\tServer shut down." && mess2 == "Server:\tServer shut down.") {
		t.Errorf(`
Expect:
'Server:	Server shut down.'
'Server:	Server shut down.'
Got:
%v
%v`, mess1, mess2)
	}
}

//Message receives first on 3rd client, 1st or 2nd does not block it
func TestNotBlockingMessage(t *testing.T) {
	port := "8082"
	listener := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)
	client3 := CreateClient("eugene3", port)

	client1.SendMessage("hello\n")

	mess := client3.GetMessage()

	if mess != client1.name+":\thello" {
		t.Errorf("Expect '"+client1.name+":\thello', got %v\n", mess)
	}

	client1.GetMessage()
	client2.GetMessage()

	listener.Close()
	wgMainLoop.Wait()
}

// Two clients send messages, third receives them in send order
func TestMessageOrder(t *testing.T) {
	port := "8083"
	listener := makeServer(port)

	client1 := CreateClient("eugene1", port)
	client2 := CreateClient("eugene2", port)
	client3 := CreateClient("eugene3", port)

	client1.SendMessage("hello from eugene1\n")
	time.Sleep(time.Microsecond)
	client2.SendMessage("hello from eugene2\n")
	time.Sleep(time.Microsecond)
	client3.SendMessage("hello from eugene3\n")

	mess1 := client3.GetMessage()
	fmt.Printf("GOT MESS1, %v\n", mess1)
	mess2 := client3.GetMessage()
	fmt.Printf("GOT MESS2, %v\n", mess2)
	mess3 := client3.GetMessage()
	fmt.Printf("GOT MESS3, %v\n", mess3)

	if !(mess1 == (client1.name+":\thello from eugene1") &&
		mess2 == (client2.name+":\thello from eugene2") &&
		mess3 == (client3.name+":\thello from eugene3")) {
		t.Errorf(`
Expect:
	message 1: '`+client1.name+`:	hello from eugene1'
	message 2: '`+client2.name+`:	hello from eugene2'
	message 3: '`+client3.name+`:	hello from eugene3'
Got:
	message 1: '%v'
	message 2: '%v'
	message 3: '%v'
`, mess1, mess2, mess3)
	}
	fmt.Print("TEST PASS\n")

	//client1.GetMessage()
	//client1.GetMessage()
	//client1.GetMessage()
	//
	//client2.GetMessage()
	//client2.GetMessage()
	//client2.GetMessage()

	listener.Close()
	wgMainLoop.Wait()
}

func makeServer(port string) net.Listener {
	listener, err := CreateServer(port)
	if err != nil {
		panic(err)
	}
	go sendAll()
	wgMainLoop.Add(1)
	go MainLoop(listener)
	return listener
}

func CreateClient(login string, port string) *Client {

	conn, err := net.Dial("tcp", ":"+port)
	if err != nil {
		panic(err)
	}
	client := new(Client)
	client.address = conn.RemoteAddr().String()
	client.name = strings.TrimSpace(login)
	client.conn = conn

	_, err = conn.Write([]byte("login:" + login + "\n"))
	if err != nil {
		panic(err)
	}

	//skip *connected* message
	client.GetMessage()

	return client
}

func (client *Client) SendMessage(mess string) {
	_, err := client.conn.Write([]byte(mess))
	if err != nil {
		panic(err)
	}
}

func (client *Client) GetMessage() string {
	mess := ""
	scanner := bufio.NewScanner(client.conn)
	if scanner.Scan() {
		mess = scanner.Text()
	}
	return mess
}
