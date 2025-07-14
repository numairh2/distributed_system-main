package main

import (
	"fmt"
	"membership-system/caller"
	"membership-system/daemon"
	"membership-system/machine"
	"os"
)

var hostToPorts = make(map[string]int)

func initHostToPorts() {
	for i := 1; i <= 10; i++ {
		hostID := fmt.Sprintf("fa24-cs425-35%02d.cs.illinois.edu", i)
		port := 8000 + i
		hostToPorts[hostID] = port
	}
}

func main() {
	initHostToPorts()
	args := os.Args
	typeArg := args[1]
	
	// typeArg is either "machine", "caller", "listener", or "daemon", otherwise invalid
	if typeArg == "machine" {
		machine.StartMachine("localhost", "1111")
	} else if typeArg == "caller" {
		caller.StartCaller()
	} else if typeArg == "daemon" {
		daemon.StartDaemon()
	} else if typeArg == "listener" {
		host, err := os.Hostname()
		if err != nil {
			fmt.Println("Error getting hostname:", err)
			return
		}
		port := hostToPorts[host]
		machine.StartUDPListener(port)
	} else {
		fmt.Printf("Invalid command: %s. Available commands: machine, caller, listener, daemon\n", typeArg)
	}
}
