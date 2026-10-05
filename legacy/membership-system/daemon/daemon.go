package daemon

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"strconv"
	"path/filepath"
	"sync"
	"time"
)

var hostToPorts = make(map[string]int)

type Daemon struct {
	nodeID           string               // Hostname of the current machine
	membershipList   map[string]string    // Hostname -> Status (alive/failed/left/crashed)
	mutex            sync.Mutex           // To protect the membership list
	pingInterval     time.Duration        // Interval for Ping messages
	pingTimeout      time.Duration        // Timeout to wait for an Ack
	suspicionEnabled bool				          // Flag to enable/dosable suspicion
	suspicionTimeout time.Duration        // Timeout to decide whether a node is failed
	suspectedNodes   map[string]bool      // Track suspected nodes
	lastAckReceived  map[string]time.Time // Track last time an Ack was received
	conn             *net.UDPConn         // UDP connection
}

func initHostToPorts() {
	for i := 1; i <= 10; i++ {
		hostID := fmt.Sprintf("fa24-cs425-35%02d.cs.illinois.edu", i)
		port := 8000 + i
		hostToPorts[hostID] = port
	}
}

func ResolveHostname(hostname string) (string, error) {
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return "", fmt.Errorf("error resolving hostname: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for hostname: %s", hostname)
	}
	return ips[0], nil
}

func (d *Daemon) sendUDPMessage(target string, message string, portNum int) {
	if d.membershipList[target] == "left" {
		fmt.Println("Node", target, "has left")
		return
	}
	if d.membershipList[target] == "failed" {
		return
	}
	if d.membershipList[target] == "crashed" {
		return
	}
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", target, portNum))
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing UDP:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error sending UDP message:", err)
	}
}

func (d *Daemon) pingNodes() {
	for {
		d.mutex.Lock()
		for nodeID := range d.membershipList {
			if nodeID != d.nodeID {
				go d.pingNode(nodeID)
			}
		}
		d.mutex.Unlock()
		time.Sleep(d.pingInterval)
	}
}

func (d *Daemon) pingNode(nodeID string) {
	portNum := hostToPorts[nodeID]
	d.sendUDPMessage(nodeID, "ping|"+d.nodeID, portNum)
	d.waitForAck(nodeID)
}

func (d *Daemon) waitForAck(nodeID string) {
	time.Sleep(d.pingTimeout)

	waitTime := time.Now().Format(time.RFC3339)
	waitTime_parse, _ := time.Parse(time.RFC3339, waitTime)
	d.mutex.Lock()
	defer d.mutex.Unlock()

	timeDiffStr := fmt.Sprintf("%.3f", waitTime_parse.Sub(d.lastAckReceived[nodeID]).Seconds()/2)
	timeDiffSlice := timeDiffStr[:5]

	timeDiffFloat, _ := strconv.ParseFloat(timeDiffSlice, 64)
	timeDiffFloat /= 10000

	if d.suspicionEnabled == true && timeDiffFloat > d.pingTimeout.Seconds(){
		fmt.Printf("Node %s confirmed as sus.\n", nodeID)
		d.suspectedNodes[nodeID] = true
		go d.confirmNodeFailure(nodeID)
	} else if d.suspicionEnabled == false && timeDiffFloat > d.pingTimeout.Seconds() {
		go d.confirmNodeFailure(nodeID)
	}
}

func (d *Daemon) handleUDPMessage(message string, source string) {
	parts := strings.Split(message, "|")
	if len(parts) < 2 {
		fmt.Println("Invalid message format")
		return
	}

	switch parts[0] {
		case "ack":
			if len(parts) < 3 {
				fmt.Println("Invalid mesage len is: ", len(parts))
				return
			}
			parseTime, e := time.Parse(time.RFC3339, parts[2])

			if e != nil {
				fmt.Println("Error with parsing")
			}
			
			d.lastAckReceived[parts[1]] = parseTime
			if d.suspectedNodes[parts[1]] {
				delete(d.suspectedNodes, parts[1])
			}
		case "crash":
			d.membershipList[parts[1]] = "crashed"
			time.Sleep(d.pingTimeout)
			if d.suspicionEnabled {
				fmt.Printf("Node %s confirmed as sus.\n", parts[1])
				time.Sleep(d.suspicionTimeout)
				d.membershipList[parts[1]] = "failed"
			} else {
				d.membershipList[parts[1]] = "failed"
			}

			var machineNum = getMachineNum(parts[1])
			var introNum = getMachineNum(d.nodeID)
			if machineNum != introNum {
				err := logOperation(machineNum, "fail", introNum)
				if err != nil {
					fmt.Printf("Error logging operation: %v\n", err)
				}
			}
	}
}

func (d *Daemon) listenUDP() {
	ip, err := ResolveHostname(d.nodeID)
	if err != nil {
		fmt.Println("Error resolving hostname:", err)
		return
	}

	for _, basePort := range hostToPorts {
		port := basePort + 10
		go func(p int) {
			addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, p))
			if err != nil {
				fmt.Printf("Error resolving UDP address for port %d: %v\n", p, err)
				return
			}
			fmt.Printf("Listening to UDP on %s\n", addr)

			conn, err := net.ListenUDP("udp", addr)
			if err != nil {
				fmt.Printf("Error listening on UDP for port %d: %v\n", p, err)
				return
			}
			defer conn.Close()

			buf := make([]byte, 1024)
			for {
				n, remoteAddr, err := conn.ReadFromUDP(buf)
				if err != nil {
					fmt.Printf("Error reading from UDP on port %d: %v\n", p, err)
					continue
				}
				message := string(buf[:n])
				d.handleUDPMessage(message, remoteAddr.String())
			}
		} (port)
	}

	select {}
}

func (d *Daemon) confirmNodeFailure(nodeID string) {
	time.Sleep(d.suspicionTimeout)

	waitTime := time.Now().Format(time.RFC3339)
	waitTime_parse,_ := time.Parse(time.RFC3339, waitTime)
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.suspicionEnabled &&  waitTime_parse.Sub(d.lastAckReceived[nodeID]).Seconds() / 2  > d.suspicionTimeout.Seconds() {
		delete(d.suspectedNodes, nodeID)
	}
	fmt.Printf("Node %s confirmed as failed.\n", nodeID)
	d.membershipList[nodeID] = "failed"
}

func (d * Daemon) enableSuspicion() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.suspicionEnabled = true
	fmt.Println("Suspicion Mechanism Enabled")
}

func (d * Daemon) disableSuspicion() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.suspicionEnabled = false
	fmt.Println("Suspicion Mechanism Disabled")
}

func (d* Daemon) checkSus() {
	if d.suspicionEnabled {
		fmt.Println("Suspicion Mechanism Is Currently Enabled")
	} else {
		fmt.Println("Suspicion Mechinism Is Currently Disabled")
	}
}

func (d *Daemon) joinGroup(node string) {
	ip, err := ResolveHostname(node)
	if err != nil {
		fmt.Println("Error resolving node hostname:", err)
		return
	}
	d.membershipList[node] = "alive"
	conn, err := net.Dial("tcp", ip+":1111")
	if err != nil {
		fmt.Println("Error contacting node:", err)
		return
	}
	defer conn.Close()

	joinMessage := fmt.Sprintf("join|%s", d.nodeID)
	_, err = conn.Write([]byte(joinMessage))
	if err != nil {
		fmt.Println("Error sending join message:", err)
	}
}

func (d *Daemon) leaveGroup(node string) {
	d.membershipList[node] = "left"
}

func (d *Daemon) listMembership() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fmt.Println("Current Membership List:")
	for nodeID, status := range d.membershipList {
		if status != "left" && status != "failed" {
			fmt.Printf("%s\n", nodeID)
		}
	}
}

func (d *Daemon) listSelf() {
	fmt.Printf("Self ID: %s\n", d.nodeID)
}

func (d *Daemon) Run() {
	go d.pingNodes()
	go d.listenUDP()
	select {}
}

func getMachineNum(hostname string) int {
	parts := strings.Split(hostname, ".")
	if len(parts) == 0 {
		return -1
	}

	vmPart := parts[0]

	index := strings.Index(vmPart, "35")
	if index == -1 || len(vmPart) <= index+2 {
		return -1
	}

	vmNumberStr := vmPart[index+2:]
	if len(vmNumberStr) > 2 {
		vmNumberStr = vmNumberStr[:2]
	}

	vmNumber, err := strconv.Atoi(vmNumberStr)
	if err != nil {
		return -1
	}
	return vmNumber
}

func logOperation(vmNumber int, operation string, introNum int) error {
	dir := "mp_2_logs"
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	introLogFilePath := filepath.Join(dir, fmt.Sprintf("output.%d.log", introNum))

	file, err := os.OpenFile(introLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	introLogEntry := fmt.Sprintf("%s: %s operation: VM%d \n", time.Now().Format(time.RFC3339), operation, vmNumber)
	_, err = file.WriteString(introLogEntry)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	if vmNumber != introNum {
		nodeLogFilePath := filepath.Join(dir, fmt.Sprintf("output.%d.log", vmNumber))

		nodeFile, err := os.OpenFile(nodeLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %v", err)
		}
		defer nodeFile.Close()

		nodeLogEntry := fmt.Sprintf("%s: %s operation: VM%d \n", time.Now().Format(time.RFC3339), operation, vmNumber)
		_, err = nodeFile.WriteString(nodeLogEntry)
		if err != nil {
			return fmt.Errorf("failed to write to log file: %v", err)
		}
	}

	return nil
}

func StartDaemon() {
	new_node := flag.String("new node", "", "Hostname of the new node joining the list (for join command)")
	flag.Parse()
	initHostToPorts()

	nodeID, _ := os.Hostname()
	daemon := Daemon{
		nodeID:           nodeID,
		membershipList:   make(map[string]string),
		pingInterval:     5 * time.Second,
		pingTimeout:      5 * time.Second,
		suspicionTimeout: 10 * time.Second,
		suspectedNodes:   make(map[string]bool),
		lastAckReceived:  make(map[string]time.Time),
		conn:             nil,
	}

	var introducerNum = getMachineNum(nodeID)
	err := logOperation(introducerNum, "join", introducerNum)
	if err != nil {
		fmt.Printf("Error logging operation: %v\n", err)
	}

	daemon.membershipList[daemon.nodeID] = "alive"

	go daemon.Run()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the membership system. Type a command:")

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		command := strings.Split(input, " ")

		switch command[0] {
		case "list_mem":
			daemon.listMembership()
		case "list_self":
			daemon.listSelf()
		case "join":
			if *new_node == "" && len(command) < 2 {
				fmt.Println("Please provide an node hostname using 'join <hostname>'")
			} else {
				status, ok := daemon.membershipList[nodeID]
				if ok && status == "alive" {
					node := *new_node
					if len(command) >= 2 {
						node = command[1]
					}
					daemon.joinGroup(node)

					var machineNum = getMachineNum(node)
					var introNum = getMachineNum(nodeID)
					if machineNum != introducerNum {
						err := logOperation(machineNum, "join", introNum)
						if err != nil {
							fmt.Printf("Error logging operation: %v\n", err)
						}
					}
				} else {
					fmt.Println("Cannot join. The introducer has left the membership list")
				}
			}
		case "join_all":
			status, ok := daemon.membershipList[nodeID]
			if ok && status == "alive" {
				var numMachines = command[1]
				n, err := strconv.Atoi(numMachines)
				if err != nil {
					fmt.Println("Error converting string to integer")
				}

				fmt.Println("Please enter the names of the machines that will join the membership list")
				var machineNames = make([]string, n)

				for i := 0; i < n; i++ {
					fmt.Printf("Enter the name of machine %d: ", i + 1)
					machineNames[i], _ = reader.ReadString('\n')
					machineNames[i] = strings.TrimSpace(machineNames[i])
				}

				for i := 0; i < n; i++ {
					daemon.joinGroup(machineNames[i])

					var machineNum = getMachineNum(machineNames[i])
					var introNum = getMachineNum(nodeID)
					if machineNum != introducerNum {
						err := logOperation(machineNum, "join", introNum)
						if err != nil {
							fmt.Printf("Error logging operation: %v\n", err)
						}
					}
				}
			} else {
				fmt.Println("Cannot join. The introducer has left the membership list")
			}
		case "leave":
			if *new_node == "" && len(command) < 2 {
				fmt.Println("Please provide an node hostname using 'leave <hostname>'")
			} else {
				node := *new_node
				if len(command) >= 2 {
					node = command[1]
				}
				daemon.leaveGroup(node)

				var machineNum = getMachineNum(node)
				var introNum = getMachineNum(nodeID)
				err := logOperation(machineNum, "leave", introNum)
				if err != nil {
					fmt.Printf("Error logging operation: %v\n", err)
				}
			}
		case "leave_all":
			var numMachines = command[1]
			n, err := strconv.Atoi(numMachines)
			if err != nil {
				fmt.Println("Error converting string to integer")
			}

			fmt.Println("Please enter the names of the machines that will leave the membership list")
			var machineNames = make([]string, n)

			for i := 0; i < n; i++ {
				fmt.Printf("Enter the name of machine %d: ", i + 1)
				machineNames[i], _ = reader.ReadString('\n')
				machineNames[i] = strings.TrimSpace(machineNames[i])
			}

			for i := 0; i < n; i++ {
				daemon.leaveGroup(machineNames[i])

				var machineNum = getMachineNum(machineNames[i])
				var introNum = getMachineNum(nodeID)
				if machineNum != introducerNum {
					err := logOperation(machineNum, "leave", introNum)
					if err != nil {
						fmt.Printf("Error logging operation: %v\n", err)
					}
				}
			}
		case "enable_sus":
			daemon.enableSuspicion()
		case "disable_sus":
			daemon.disableSuspicion()
		case "check_sus":
			daemon.checkSus()
		case "exit":
			fmt.Println("Exiting membership system.")
			return
		default:
			fmt.Printf("Invalid command: %s. Available commands: list_mem, list_self, join, leave, exit\n", command[0])
		}
	}
}