package main

import "database/sql"

type Database struct {
	db   *sql.DB
	name string
}

func (database *Database) connect(dbIP string, dbPort string, dbName string) {
	db, err := sql.Open("mysql", "root:@tcp("+dbIP+":"+dbPort+")/")
	if err != nil {
		panic(err)
	}
	database.db = db
	database.name = dbName
	database.init()
}

func (database *Database) init() {
	if database == nil || database.db == nil {
		return
	}
	_, err := database.db.Exec(`
	CREATE DATABASE IF NOT EXISTS ` + database.name + ` CHARACTER SET utf8mb3;`)
	if err != nil {
		panic(err)
	}
	_, err = database.db.Exec(`
	CREATE TABLE IF NOT EXISTS ` + database.name + `.history (
		id int auto_increment primary key,
		name text(65535),
		message text(65535) not null
	)`)
	if err != nil {
		panic(err)
	}
}

func (database *Database) drop() {
	if database == nil || database.db == nil {
		return
	}
	_, err := database.db.Exec(`
	DROP DATABASE ` + database.name + `;`)
	if err != nil {
		panic(err)
	}
}

func (database *Database) addMessage(message Message) {
	if database == nil || database.db == nil {
		return
	}
	_, err := database.db.Exec(`INSERT INTO `+database.name+`.history (name, message) values (?,?)`, message.name, message.message)
	if err != nil {
		panic(err)
	}
}

func (database *Database) getHistory() string {
	if database == nil || database.db == nil {
		return ""
	}

	result, err := database.db.Query(`SELECT name, message FROM ` + database.name + `.history ORDER BY id`)
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
