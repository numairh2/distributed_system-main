package machine

import (
	"bytes"
	"membership-system/utils"
	"fmt"
	"net"
	"time"
	"net/rpc"
	"os/exec"
	"strconv"
	"strings"
	"os"
	"os/signal"
	"syscall"
)

type LogHandler struct{}

type LogHandlerInterface interface {
	HandleLog() error
}

// Runs grep command to find line count and pattern given from caller then sends back reply of entries and linecount for that machine
func (l *LogHandler) HandleLog(grepCommandArgs []string, grepCommandResult *utils.RPCResponseJSON) error {
	fmt.Println("LOG: rpc HandleLog called with grepCommandArgs: ", grepCommandArgs)
	machineNum := grepCommandArgs[len(grepCommandArgs)-1]
	// skip the first word to make sure that grep is always run, for security
	grepCommandArgs = grepCommandArgs[1 : len(grepCommandArgs)-1]

	var quotedArgs []string
	for _, arg := range grepCommandArgs {
		quotedArgs = append(quotedArgs, fmt.Sprintf(`"%s"`, arg))
	}
	//concatinate grep command and run for finding pattern in machine.i.log
	cmdStr := fmt.Sprintf("grep %s -Hn ./vm%s.log", strings.Join(quotedArgs, " "), machineNum)
	fmt.Println("LOG: Running command: ", cmdStr)
	cmd := exec.Command("bash", "-c", cmdStr)

	commandOutput := bytes.Buffer{}
	cmd.Stdout = &commandOutput
	err := cmd.Run()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			// this means grep could not find the pattern, do not return an error
			*grepCommandResult = utils.RPCResponseJSON{Entries: "", LineCount: 0, Err: nil}
			fmt.Println("LOG: no pattern found: ", err)
			return nil
		}
		*grepCommandResult = utils.RPCResponseJSON{Entries: "", LineCount: 0, Err: err}
		fmt.Println("LOG: failed to run command: ", err)
		return err
	}
	//runs command to find linecount of pattern found in machine.i.log
	countCmdStr := fmt.Sprintf("grep %s -c ./vm%s.log", strings.Join(quotedArgs, " "), machineNum)
	fmt.Println("LOG: Running count command: ", countCmdStr)
	countCmd := exec.Command("bash", "-c", countCmdStr)

	countOutput := bytes.Buffer{}
	countCmd.Stdout = &countOutput
	err = countCmd.Run()
	if err != nil {
		*grepCommandResult = utils.RPCResponseJSON{Entries: commandOutput.String(), LineCount: 0, Err: err}
		fmt.Println("LOG: error during count command: ", err)
		return err
	}

	lineCount := strings.TrimSpace(countOutput.String())
	lineCountint, err := strconv.Atoi(lineCount)
	if err != nil {
		fmt.Println(err)
		return err
	}
	//creates reply json containing entries and linecount
	*grepCommandResult = utils.RPCResponseJSON {
		Entries:   commandOutput.String(),
		LineCount: lineCountint,
		Err:       nil,
	}
	fmt.Printf("LOG: result saved into *grepCommandResult. Linecount: %d \n", lineCountint)
	return nil
}

func StartMachine(hostname string, port string) {
	logHandler := &LogHandler{}
	rpc.Register(logHandler)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("LOG: listener created on port %s \n", port)
	for {
		clientConn, err := listener.Accept()
		fmt.Println("LOG: client connected")
		if err != nil {
			fmt.Println(err)
			continue
		}
		go rpc.ServeConn(clientConn)
	}
}

func handleUDPMessage(conn *net.UDPConn, buf []byte, port int) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var lastAddr *net.UDPAddr

	go func() {
		<-sigChan
		if lastAddr != nil {
			fmt.Println("LOG: received SIGINT, sending crash message and exiting")
			
			portNum := port + 10
			ip := lastAddr.IP.String()
			pingBackAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, portNum))
			if err != nil {
				fmt.Println("Error resolving ping back address:", err)
			}

			nodeID, _ := os.Hostname()
			
			crashMessage := fmt.Sprintf("crash|%s|%s", nodeID, time.Now().Format(time.RFC3339))
			_, err = conn.WriteToUDP([]byte(crashMessage), pingBackAddr)
			if err != nil {
				fmt.Println("Error sending crash message:", err)
			}
			fmt.Printf("LOG: sent crash to %s%d\n", nodeID+":", portNum)
		}
		conn.Close()
		os.Exit(0)
	}()

	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		lastAddr = addr

		fmt.Printf("LOG: received message from %s\n", addr)
		message := string(buf[:n])
		parts := strings.Split(message, "|")
		if len(parts) < 2 {
			fmt.Println("Invalid message format")
			continue
		}

		portNum := port + 10

		ip := addr.IP.String()

		pingBackAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", ip, portNum))
		if err != nil {
			fmt.Println("Error resolving ping back address:", err)
			continue
		}

		ackMessage := fmt.Sprintf("ack|%s|%s", parts[1], time.Now().Format(time.RFC3339))
		_, err = conn.WriteToUDP([]byte(ackMessage), pingBackAddr)
		if err != nil {
			fmt.Println("Error sending ack:", err)
		}
		fmt.Printf("LOG: sent ack to %s%d\n", ip+":", portNum)
	}
}

func StartUDPListener(port int) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error listening on UDP:", err)
		return
	}
	fmt.Printf("LOG: listening on UDP address: %s\n", conn.LocalAddr())
	defer conn.Close()

	buf := make([]byte, 1024)
	handleUDPMessage(conn, buf, port)
}
