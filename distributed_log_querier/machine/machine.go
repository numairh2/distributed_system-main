package machine

import (
	"bytes"
	"distributed_log_querier/utils"
	"fmt"
	"net"
	"net/rpc"
	"os/exec"
	"strconv"
	"strings"
)

type LogHandler struct{}

type LogHandlerInterface interface {
	HandleLog() error
}
//Runs grep command to find line count and pattern given from caller then sends back reply of entries and linecount for that machine
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
	*grepCommandResult = utils.RPCResponseJSON{
		Entries:   commandOutput.String(),
		LineCount: lineCountint,
		Err:       nil,
	}
	fmt.Printf("LOG: result saved into *grepCommandResult. Linecount: %d \n", lineCountint)
	return nil
}
// Start Machine to listen for caller
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
