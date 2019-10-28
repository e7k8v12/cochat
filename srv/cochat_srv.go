package main

import (
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type Message struct {
	name    string
	message string
}

func (mess *Message) String() string {
	return mess.name + ":\t" + mess.message
}

func main() {
	srvPort := flag.String("port", "8080", "Port of a server")
	dbIP := flag.String("dbip", "127.0.0.1", "IP of a database server")
	dbPort := flag.String("dbport", "3306", "Port of a database server")
	flag.Parse()

	server := &Server{}
	err := server.initServer(*srvPort)
	if err != nil {
		panic(err)
	}

	server.database.connect(*dbIP, *dbPort, "cochatdb")

	defer func() {
		if server.database.db != nil {
			err := server.database.db.Close()
			printError(err)
		}
	}()

	fmt.Printf("Console chat server started at %v\n", server.listener.Addr().String())

	go server.messageForwarding()
	go server.deleteBrokenConnections()

	server.wgListenConnections.Add(1)
	go server.listenForNewConnections()
	server.wgListenConnections.Wait()
}

func printError(err error) {
	if err != nil {
		fmt.Print(err)
	}
}
