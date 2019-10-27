package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net"
	"strconv"
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
	mu       sync.Mutex
}

type Globals struct {
	muConnections sync.Mutex
	connections   map[string]*Client
	brChan        chan Message
	wgMainLoop    sync.WaitGroup
	wgSendMess    sync.WaitGroup
	db            *sql.DB
	ticker        *time.Ticker
}

func (mess *Message) String() string {
	return mess.name + ":\t" + mess.message
}

func (client *Client) Write(mess string) error {
	client.mu.Lock()
	_, err := client.conn.Write([]byte(mess))
	client.mu.Unlock()
	return err
}
func main() {
	srvPort := flag.String("port", "8080", "Port of a server")
	dbIP := flag.String("dbip", "127.0.0.1", "IP of a db server")
	dbPort := flag.String("dbport", "3306", "Port of a db server")
	flag.Parse()

	globals := &Globals{
		muConnections: sync.Mutex{},
		connections:   make(map[string]*Client),
		brChan:        make(chan Message, 10000),
		wgMainLoop:    sync.WaitGroup{},
		wgSendMess:    sync.WaitGroup{},
		db:            nil,
		ticker:        time.NewTicker(time.Second * 5),
	}
	defer globals.ticker.Stop()
	connectDB(globals, *dbIP, *dbPort)
	defer func() {
		if globals.db != nil {
			err := globals.db.Close()
			printError(err)
		}
	}()

	initDB(globals)
	listener, err := CreateServer(*srvPort)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Console chat server started at %v\n", listener.Addr().String())
	go sendAll(globals)
	go checkConnections(globals)
	globals.wgMainLoop.Add(1)
	go MainLoop(globals, listener)
	globals.wgMainLoop.Wait()
}

func MainLoop(globals *Globals, listener net.Listener) {
	for {
		conn, err := listener.Accept()

		if err != nil {
			globals.brChan <- Message{
				name:    "Server",
				message: "Server shut down.",
			}
			globals.wgSendMess.Wait()
			fmt.Printf("Connection closed. Accept error: %v\n", err)
			globals.wgMainLoop.Done()
			break
		}
		connectionAddress := conn.RemoteAddr().String()
		globals.muConnections.Lock()
		globals.connections[connectionAddress] = &Client{address: connectionAddress, conn: conn, sendChan: make(chan Message, 100)}
		go globals.connections[connectionAddress].handleConnection(globals)
		globals.muConnections.Unlock()
	}
}

func CreateServer(srvPort string) (net.Listener, error) {
	listener, err := net.Listen("tcp", ":"+srvPort)
	return listener, err
}

func checkConnections(globals *Globals) {
	for range globals.ticker.C {
		globals.muConnections.Lock()
		for _, client := range globals.connections {
			go func(client *Client, globals *Globals) {
				err := client.Write("##test message##\n")
				if err != nil {
					client.deleteConnection(globals)
				}
			}(client, globals)
		}
		globals.muConnections.Unlock()
	}
}

func sendAll(globals *Globals) {
	for mess := range globals.brChan {
		globals.muConnections.Lock()
		for _, client := range globals.connections {
			globals.wgSendMess.Add(1)
			go func(client *Client, mess Message) {
				client.sendChan <- mess
			}(client, mess)
			go func(client *Client) {
				mess := <-client.sendChan
				err := client.Write(mess.String() + "\n")
				if err != nil {
					client.deleteConnection(globals)
				}
				globals.wgSendMess.Done()
			}(client)
		}
		globals.muConnections.Unlock()
	}
}

func (client *Client) handleConnection(globals *Globals) {

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

	history := getHistoryFromDB(globals)
	err := client.Write(history)
	if err != nil {
		client.deleteConnection(globals)
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
			client.deleteConnection(globals)
			//brChan <- Message{name: name, message: "*client disconnected*"}
			break loopScanner
		case lowerText == "#count":
			globals.muConnections.Lock()
			err = client.Write(strconv.Itoa(len(globals.connections)) + "\n")
			printError(err)
			globals.muConnections.Unlock()
		case lowerText == "#list":
			mess := ""
			globals.muConnections.Lock()
			for _, cli := range globals.connections {
				mess += cli.name + " " + cli.address + "\n"
			}
			globals.muConnections.Unlock()
			err = client.Write(mess)
			printError(err)
		case lowerText == "#history":
			mess := getHistoryFromDB(globals)
			err = client.Write(mess)
			printError(err)
		case text != "":
			fmt.Printf("%v enters '%v'\n", name, text)
			mess := Message{name: name, message: text}
			globals.brChan <- mess
			addToDB(globals, mess)
		}
	}
	//brChan <- Message{name, "*disconnected*"}
	fmt.Printf("%v %v disconnected\n", name, client.address)
}

func printError(err error) {
	if err != nil {
		fmt.Print(err)
	}
}

func (client *Client) deleteConnection(globals *Globals) {
	globals.muConnections.Lock()
	client.conn.Close()
	delete(globals.connections, client.address)
	globals.muConnections.Unlock()
}

func connectDB(globals *Globals, dbIP string, dbPort string) {
	db, err := sql.Open("mysql", "root:@tcp("+dbIP+":"+dbPort+")/")
	if err != nil {
		panic(err)
	}
	globals.db = db
}

func initDB(globals *Globals) {
	if globals.db == nil {
		return
	}
	_, err := globals.db.Exec(`
	CREATE DATABASE IF NOT EXISTS cochatdb CHARACTER SET utf8mb3;`)
	if err != nil {
		panic(err)
	}
	_, err = globals.db.Exec(`
	CREATE TABLE IF NOT EXISTS cochatdb.history (
		id int auto_increment primary key,
		name text(65535),
		message text(65535) not null
	)`)
	if err != nil {
		panic(err)
	}
}

func addToDB(globals *Globals, message Message) {
	if globals.db == nil {
		return
	}
	_, err := globals.db.Exec(`INSERT INTO cochatdb.history (name, message) values (?,?)`, message.name, message.message)
	if err != nil {
		panic(err)
	}
}

func getHistoryFromDB(globals *Globals) string {
	if globals.db == nil {
		return ""
	}

	result, err := globals.db.Query(`SELECT name, message FROM cochatdb.history ORDER BY id`)
	if err != nil {
		panic(err)
	}
	defer func() {
		if result != nil {
			err := result.Close()
			printError(err)
		}
	}()
	history := ""
	name, mess := new(string), new(string)
	for result.Next() {
		err := result.Scan(name, mess)
		if err != nil {
			panic(err)
		}
		history += *name + ":\t" + *mess + "\n"
	}
	return history
}
