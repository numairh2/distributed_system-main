package daemon

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Daemon struct {
	nodeID           string               // Hostname of the current machine
	membershipList   map[string]string    // Hostname -> Status (alive/failed)
	mutex            sync.Mutex           // To protect the membership list
	pingInterval     time.Duration        // Interval for Ping messages
	pingTimeout      time.Duration        // Timeout to wait for an Ack
	suspicionEnabled bool				  // Flag to enable/dosable suspicion
	suspicionTimeout time.Duration        // Timeout to decide whether a node is failed
	suspectedNodes   map[string]bool      // Track suspected nodes
	lastAckReceived  map[string]time.Time // Track last time an Ack was received
}

// ResolveHostname resolves a hostname to an IP address
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

// Send a UDP message to a target hostname
func (d *Daemon) sendUDPMessage(target string, message string) {
	ip, err := ResolveHostname(target)
	if err != nil {
		fmt.Println("Error resolving hostname:", err)
		return
	}
	conn, err := net.Dial("udp", ip+":3000")
	if err != nil {
		fmt.Println("Error sending UDP message:", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(message))
	if err != nil {
		fmt.Println("Error writing UDP message:", err)
	}
}

// Ping function to ping nodes and handle failure detection
func (d *Daemon) pingNodes() {
	for {
		d.mutex.Lock()
		for nodeID := range d.membershipList {
			if nodeID != d.nodeID && d.membershipList[nodeID] == "alive" {
				go d.pingNode(nodeID) // Ping each node concurrently
			}
		}
		d.mutex.Unlock()
		time.Sleep(d.pingInterval)
	}
}

func (d *Daemon) pingNode(nodeID string) {
	d.mutex.Lock()
	// Initialize lastAckReceived for the node if it doesn't exist
	if _, exists := d.lastAckReceived[nodeID]; !exists {
		d.lastAckReceived[nodeID] = time.Now() // Set to current time or zero-value
	}
	d.mutex.Unlock()

	d.sendUDPMessage(nodeID, "ping")
	d.waitForAck(nodeID)
}


func (d *Daemon) waitForAck(nodeID string) {
	select {
		case <-time.After(d.pingTimeout):
			d.mutex.Lock()
			defer d.mutex.Unlock()

			// If we haven't received an Ack within the timeout, mark the node as suspected
			fmt.Println("last ack: ", time.Since(d.lastAckReceived[nodeID]))
			if time.Since(d.lastAckReceived[nodeID]) > d.pingTimeout {
				if d.suspicionEnabled {
					fmt.Printf("Node %s didn't respond, marking as suspected.\n", nodeID)
					d.suspectedNodes[nodeID] = true
					go d.confirmNodeFailure(nodeID) // Start the suspicion process
				} else {
					fmt.Printf("Node %s didn't respond, marking as suspected.\n", nodeID)
					d.membershipList[nodeID] = "failed"
					d.broadcastMembershipList()
				}
			}
		}
	}


// Handle incoming UDP messages
func (d *Daemon) handleUDPMessage(message string, source string) {
	parts := strings.Split(message, "|")
	msgType := parts[0]

	switch msgType {
	case "ping":
		// Respond to ping with ack
		d.sendUDPMessage(source, "ack")
		d.lastAckReceived[source] = time.Now()
		fmt.Printf("Received ack from %s\n", source)
	case "ack":
		// Update last ack time and remove node from suspected list
		d.mutex.Lock()
		d.lastAckReceived[source] = time.Now()
		delete(d.suspectedNodes, source) // Clear suspicion if ack is received
		d.mutex.Unlock()
		fmt.Printf("Received ack from %s\n", source)
	case "join":
		d.mutex.Lock()
		if _, exists := d.membershipList[source]; !exists {
			// Add the new node to the membership list
			d.membershipList[source] = "alive"
			// Initialize the lastAckReceived for the new node
			d.lastAckReceived[source] = time.Now() // Set initial time as current time
			// Broadcast the updated membership list to others
			d.broadcastMembershipList()
			fmt.Printf("New node %s joined. Broadcasting update...\n", source)
		}
		d.mutex.Unlock()	
	case "update":
		// Handle membership list updates
		if len(parts) < 2 {
			fmt.Printf("Invalid update message from %s\n", source)
			return
		}
		membershipData := parts[1]
		d.updateMembershipList(membershipData)
		fmt.Printf("Received membership update from %s: %s\n", source, membershipData)
	}
}

// Update the local membership list based on received data
func (d *Daemon) updateMembershipList(data string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// Example data format: "NodeA:alive,NodeB:failed,NodeC:alive"
	nodes := strings.Split(data, ",")
	for _, node := range nodes {
		nodeInfo := strings.Split(node, ":")
		if len(nodeInfo) == 2 {
			nodeID := nodeInfo[0]
			status := nodeInfo[1]

			// Only update if the node status is different or not present
			if currentStatus, exists := d.membershipList[nodeID]; !exists || currentStatus != status {
				d.membershipList[nodeID] = status
				fmt.Printf("Updated node %s status to %s\n", nodeID, status)
			}
		}
	}
}


// Start UDP listener
func (d *Daemon) listenUDP() {
	fmt.Println("Listenning to UDP")
	addr, err := net.ResolveUDPAddr("udp", ":8081")
	if err != nil {
		fmt.Println("Error Connection", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error conection", err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}
		message := string(buf[:n])
		d.handleUDPMessage(message, remoteAddr.String())
	}
}

// Confirm whether a suspected node is failed
func (d *Daemon) confirmNodeFailure(nodeID string) {
	time.Sleep(d.suspicionTimeout) // Wait for the suspicion period

	d.mutex.Lock()
	defer d.mutex.Unlock()

	// If the node is still suspected after the suspicion timeout, mark it as failed
	if d.suspectedNodes[nodeID] {
		fmt.Printf("Node %s confirmed as failed.\n", nodeID)
		d.membershipList[nodeID] = "failed"
		delete(d.suspectedNodes, nodeID)
		d.broadcastMembershipList()
	}

	// else {
	//     fmt.Printf("Node %s suspicion cleared.\n", nodeID)
	// }
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

	d.suspicionEnabled = true
	fmt.Println("Suspicion Mechanism Disable")
}

func (d* Daemon) checkSus() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	fmt.Println("status_sus: ", d.suspicionEnabled)
	fmt.Println("If true, then Suspicion is ENABLED. Else, it is DISABLE")
}

// Join the group by contacting the introducer node
func (d *Daemon) joinGroup(introducer string) {
	// Resolve introducer's hostname
	ip, err := ResolveHostname(introducer)
	if err != nil {
		fmt.Println("Error resolving introducer hostname:", err)
		return
	}
	d.membershipList[introducer] = "alive"
	conn, err := net.Dial("tcp", ip+":1111")
	if err != nil {
		fmt.Println("Error contacting introducer:", err)
		return
	}
	defer conn.Close()

	// Send a join request
	joinMessage := fmt.Sprintf("join|%s", d.nodeID)
	_, err = conn.Write([]byte(joinMessage))
	if err != nil {
		fmt.Println("Error sending join message:", err)
	}
}


// Leave the group voluntarily
func (d *Daemon) leaveGroup() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	fmt.Println("Leaving the group voluntarily.")
	// delete(d.membershipList, d.nodeID)
	d.membershipList[d.nodeID] = "left"
	d.broadcastMembershipList() // Inform others about the leave
}

// List the current membership list
func (d *Daemon) listMembership() {
	// d.mutex.Lock()
	// defer d.mutex.Unlock()

	fmt.Println("Current Membership List:")
	for nodeID, status := range d.membershipList {
		fmt.Printf("%s: %s\n", nodeID, status)
	}
}

// List the node's own ID
func (d *Daemon) listSelf() {
	fmt.Printf("Self ID: %s\n", d.nodeID)
}

// Broadcast membership list updates to all nodes
func (d *Daemon) broadcastMembershipList() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	membershipStr := ""
	for nodeID, status := range d.membershipList {
		membershipStr += fmt.Sprintf("%s:%s,", nodeID, status)
	}

	for nodeID := range d.membershipList {
		if nodeID != d.nodeID {
			d.sendUDPMessage(nodeID, "update|"+membershipStr)
		}
	}
}

// Display suspected nodes (if the node suspects failure)
func (d *Daemon) displaySuspectedNode(nodeID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.suspectedNodes[nodeID] {
		fmt.Printf("Suspected node: %s\n", nodeID)
	} else {
		fmt.Printf("No suspicions for node: %s\n", nodeID)
	}
}

// Run the Daemon: listening and periodically pinging nodes
func (d *Daemon) Run() {
	go d.pingNodes() // Start the ping process
	go d.listenUDP() // Start listening for UDP messages
	select {}        // Keep the process alive
}

func StartDaemon() {
	// Parse the introducer flag
	introducer := flag.String("introducer", "", "Hostname of the introducer (for join command)")
	flag.Parse()

	// Initialize the daemon
	nodeID, _ := os.Hostname()
	daemon := Daemon{
		nodeID:           nodeID,
		membershipList:   make(map[string]string),
		pingInterval:     1 * time.Second,
		pingTimeout:      5 * time.Second,  // Wait 3 seconds for an ack
		suspicionTimeout: 10 * time.Second, // Wait 10 seconds before marking as failed
		suspectedNodes:   make(map[string]bool),
		lastAckReceived:  make(map[string]time.Time),
	}

	// Add self to the membership list
	daemon.membershipList[daemon.nodeID] = "alive"

	daemon.mutex.Lock()
	currentTime := time.Now()
	daemon.lastAckReceived[daemon.nodeID] = currentTime
	// for nodeID := range daemon.membershipList {
	// 	daemon.lastAckReceived[nodeID] = currentTime
	// }
	daemon.mutex.Unlock()

	// Start the daemon in the background
	go daemon.Run()

	// Start an interactive command prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the membership system. Type a command:")

	for {
		fmt.Print("> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input) // Clean up the input

		// Split the command and arguments
		command := strings.Split(input, " ")

		switch command[0] {
		case "list_mem":
			daemon.listMembership()
		case "list_self":
			daemon.listSelf()
		case "join":
			if *introducer == "" && len(command) < 2 {
				fmt.Println("Please provide an introducer hostname using 'join <hostname>'")
			} else {
				// Use introducer from command if provided, otherwise use the flag
				intro := *introducer
				if len(command) >= 2 {
					intro = command[1]
				}
				daemon.joinGroup(intro)
			}
		case "leave":
			daemon.leaveGroup()
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
