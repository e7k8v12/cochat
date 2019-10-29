package main

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

//Test send and receive message
func TestGetHelloMessage(t *testing.T) {
	port := "8081"
	server := makeServer(port)
	client := createClient("eugene", port)
	client.sendMessage("hello")
	mess := client.GetMessage()
	if mess != client.name+":\thello" {
		t.Errorf("Expect '"+client.name+":\thello', got %v\n", mess)
	}
	server.listener.Close()
	server.wgListenConnections.Wait()
}

//Test server sends disconnecting message when server stops
func TestStopServerMessage(t *testing.T) {
	port := "8082"
	server := makeServer(port)

	client1 := createClient("eugene1", port)
	client2 := createClient("eugene2", port)

	server.signals <- syscall.SIGTERM

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

	server.wgListenConnections.Wait()
}

//Message receives first on 3rd client, 1st or 2nd does not block it
func TestNotBlockingMessage(t *testing.T) {
	port := "8083"
	server := makeServer(port)

	client1 := createClient("eugene1", port)
	client2 := createClient("eugene2", port)
	client3 := createClient("eugene3", port)

	time.Sleep(time.Millisecond)

	client1.sendMessage("hello")

	mess := client3.GetMessage()

	if mess != client1.name+":\thello" {
		t.Errorf("Expect '"+client1.name+":\thello', got %v\n", mess)
	}

	client1.GetMessage()
	client2.GetMessage()

	server.listener.Close()
	server.wgListenConnections.Wait()
}

// Three clients send messages, third receives them in send order
func TestMessageOrder(t *testing.T) {
	port := "8084"
	server := makeServer(port)

	client1 := createClient("eugene1", port)
	client2 := createClient("eugene2", port)
	client3 := createClient("eugene3", port)

	client1.sendMessage("hello from eugene1")
	time.Sleep(time.Millisecond)
	client2.sendMessage("hello from eugene2")
	time.Sleep(time.Millisecond)
	client3.sendMessage("hello from eugene3")
	time.Sleep(time.Millisecond)

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

	server.listener.Close()
	server.wgListenConnections.Wait()
}

// Two clients send messages, third receives them in send order
func TestMessageOrder2Mess(t *testing.T) {
	port := "8085"
	server := makeServer(port)

	client1 := createClient("eugene1", port)

	client1.sendMessage("hello1 from eugene1")
	time.Sleep(time.Microsecond)
	client1.sendMessage("hello2 from eugene1")

	mess1 := client1.GetMessage()
	fmt.Printf("GOT MESS1, %v\n", mess1)
	mess2 := client1.GetMessage()
	fmt.Printf("GOT MESS2, %v\n", mess2)

	if !(mess1 == (client1.name+":\thello1 from eugene1") &&
		mess2 == (client1.name+":\thello2 from eugene1")) {
		t.Errorf(`
	Expect:
		message 1: '`+client1.name+`:	hello1 from eugene1'
		message 2: '`+client1.name+`:	hello2 from eugene1'
	Got:
		message 1: '%v'
		message 2: '%v'
	`, mess1, mess2)
	}

	server.listener.Close()
	server.wgListenConnections.Wait()
}

//Connect 1000 clients, send from random client, read from random client
func TestCreate1000Clients(t *testing.T) {
	port := "8086"
	server := makeServer(port)
	clients := make([]*Client, 1000)

	for i := 0; i < 1000; i++ {
		clients[i] = createClient("eugene"+strconv.Itoa(i), port)
	}

	rand.Seed(time.Now().Unix())
	outCln := rand.Intn(1000)
	inCln1 := rand.Intn(1000)
	inCln2 := rand.Intn(1000)

	clients[outCln].sendMessage("hello")
	mess1 := clients[inCln1].GetMessage()
	mess2 := clients[inCln2].GetMessage()
	if !(mess1 == clients[outCln].name+":\thello" && mess2 == clients[outCln].name+":\thello") {
		t.Errorf(`
Expect:
`+clients[outCln].name+`:	hello
`+clients[outCln].name+`:	hello
Got:
%v
%v`, mess1, mess2)
	}
	server.listener.Close()
	server.wgListenConnections.Wait()
}

//Compare sent message with DB history
func TestDBHistory(t *testing.T) {
	port := "8087"
	server := makeServer(port)
	server.database.connect("127.0.0.1", "3306", "cochatdb_test1")
	defer func() {
		if server.database != nil && server.database.db != nil {
			err := server.database.db.Close()
			printError(err)
		}
	}()

	client := createClient("eugene", port)
	client.sendMessage("hello")
	time.Sleep(time.Millisecond)
	client.sendMessage("how are you?")
	time.Sleep(time.Millisecond)
	client.sendMessage("#history")
	time.Sleep(time.Millisecond)
	client.GetMessage()
	client.GetMessage()
	history1 := client.GetMessage()
	history2 := client.GetMessage()

	//history := server.database.getHistory()

	if !(history1 == client.name+":\thello" && history2 == client.name+":\thow are you?") {
		t.Errorf(`
Expect:
`+client.name+`:	hello
`+client.name+`:	how are you?
Got:
%v
%v`, history1, history2)
	}

	server.listener.Close()
	server.wgListenConnections.Wait()

	time.Sleep(time.Millisecond)
	server.database.drop()
}

func makeServer(port string) *Server {
	server := &Server{}
	err := server.initServer(port)
	if err != nil {
		panic(err)
	}

	go server.catchSignal()
	go server.messageForwarding()
	server.wgListenConnections.Add(1)
	go server.listenForNewConnections()
	return server
}

func createClient(login string, port string) *Client {

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

	//skip history message
	//client.GetMessage()

	return client
}

func (client *Client) sendMessage(mess string) {
	err := client.Write(mess)
	if err != nil {
		panic(err)
	}
}

func (client *Client) GetMessage() string {
	//mess := ""

	////not responding
	//scanner := bufio.NewScanner(client.conn)
	//if scanner.Scan() {
	//	mess = scanner.Text()
	//}

	////not responding
	//reader := bufio.NewReader(client.conn)
	//mess, err := reader.ReadString('\n')
	//if err != nil {
	//	panic(err)
	//}

	//this works
	buf1 := make([]byte, 1)
	buf := make([]byte, 0, 1024)
	lineBreak := byte('\n')
	for buf1[0] != lineBreak {
		_, err := client.conn.Read(buf1)
		if err != nil {
			panic(err)
		}
		buf = append(buf, buf1...)
	}
	mess := string(buf[:len(buf)-1])
	return mess
}
