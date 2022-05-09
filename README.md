# MiniC2

MiniC2 is a Command and Control / Post Exploitation tool that targets Linux hosts (for now!)

## Features

Here are MiniC2's current features:

- Communicates completely over HTTP
- Send OS/Shell commands and recieve output
- View system information such as network interfaces, architecture, etc
- Can handle multiple hosts
- Adjustable sleep times for agents (beacons)

## Usage

MiniC2's usage is quite simple:

On your C2 Server:
```
$ minic2 -p 80
```

After configuring your host and IP in the agent.go file, run this on your target:
```
$ ./agent # That's it!
```

## Planned Features

- SSL
- Build agents from the MiniC2 menu
- Upload and Download file functionality
- Persistence Mechanisms
