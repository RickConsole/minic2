// Contains methods for reading and writing to agents database
package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"

	"github.com/cheynewallace/tabby"
)

func Demo() {
	fmt.Println("Hello!!")
}

func CreateAgentsTable(db *sql.DB) {
	createAgentsTableSQL := `CREATE TABLE agents (
		"agentid" TEXT NOT NULL PRIMARY KEY,
		"hostname" TEXT,
		"username" TEXT
		);`
	statement, err := db.Prepare(createAgentsTableSQL)
	if err != nil {
		fmt.Println(err)
	}

	statement.Exec()
	log.Println("Agents Table Created!")
}

func InsertNewAgent(db *sql.DB, agentid string, hostname string, username string) {
	insertAgentSQL := `INSERT INTO agents (agentid, hostname, username) VALUES (?,?,?)`
	statement, err := db.Prepare(insertAgentSQL)
	if err != nil {
		fmt.Println(err)
	}
	_, err = statement.Exec(agentid, hostname, username)
	if err != nil {
		log.Fatal(err)
	}
}

func DeleteAgent(db *sql.DB, agentid string) {
	deleteAgentSQL := "DELETE FROM agents WHERE agentid LIKE \"" + agentid + "%\";"
	statement, err := db.Prepare(deleteAgentSQL)
	if err != nil {
		fmt.Println(err)
	}
	_, err = statement.Exec()
	if err != nil {
		log.Fatal(err)
	}
}

func CheckAgent(db *sql.DB, agentid string) bool {
	checkAgentSQL := `SELECT agentid FROM agents WHERE agentid = ?`
	err := db.QueryRow(checkAgentSQL, agentid)
	if err != nil {
		return false
	} else {
		return true
	}
}

func ShowAgents(db *sql.DB) {
	row, err := db.Query("SELECT * FROM agents")
	if err != nil {
		log.Fatal(err)
	}

	defer row.Close()
	t := tabby.New()
	t.AddHeader("AGENT ID", "HOSTNAME", "USERNAME")
	for row.Next() {
		var agentid string
		var hostname string
		var username string
		row.Scan(&agentid, &hostname, &username)
		t.AddLine(agentid, hostname, username)
	}
	t.Print()
}
