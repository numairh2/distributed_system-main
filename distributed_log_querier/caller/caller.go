package caller

import (
	"distributed_log_querier/utils"
	"encoding/json"
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"sync"
)

type Callable interface {
	Call(serviceMethod string, args any, reply any) error
}

var totalLineCountLock sync.Mutex
var totalLineCount int

// Sends grep command via RPC and then takes reply and writes to file using WriteOutputToFile() function
func CommunicateWithMachine(machineNum int, conn Callable, wg *sync.WaitGroup) error {
	defer wg.Done()
	grepCommandArgs := append(os.Args[2:], fmt.Sprintf("%d", machineNum))
	var grepCommandResult utils.RPCResponseJSON
	err := conn.Call("LogHandler.HandleLog", grepCommandArgs, &grepCommandResult)
	if exitErr, ok := err.(*exec.ExitError); ok {
		fmt.Println("LOG: command exit error", exitErr)
		return err
	} else if err != nil {
		fmt.Printf("LOG: RPC call failed: %v \n", err)
		return err
	}
	if conn == nil {
		return fmt.Errorf("LOG: RPC connection is nil")
	}

	if grepCommandResult.Err != nil {
		fmt.Printf("LOG: got error from machine %s \n", grepCommandResult.Err.Error())
		return grepCommandResult.Err
	}
	fmt.Printf("Line Count machine.%d.log: %d\n", machineNum, grepCommandResult.LineCount)

	// Update global total line count
	totalLineCountLock.Lock()
	totalLineCount += grepCommandResult.LineCount
	totalLineCountLock.Unlock()

	//Writes to file locally on machine
	err = WriteOutputToFile(fmt.Sprintf("result/output.%d.txt", machineNum), grepCommandResult.Entries)
	if err != nil {
		return err
	}
	fmt.Printf("LOG: wrote output to file for machine %d\n", machineNum)
	return nil
}

// Writes output from grep command to file on Machine
func WriteOutputToFile(path string, output string) error {
	// get file
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(output)
	return err
}

// Starts caller and dials to each machine inside of config.json file
func StartCaller() {
	machineConfigs, err := getConfig("config.json")
	if err != nil {
		return
	}
	wg := sync.WaitGroup{}
	for i, machineConfig := range machineConfigs {
		numMachine := i + 1
		conn, err := rpc.Dial("tcp", utils.GetAddress(machineConfig.Host, machineConfig.Port))
		if err != nil {
			fmt.Printf("LOG: Failed to dial machine %d: %v \n", numMachine, err)
			continue
		}
		wg.Add(1)
		fmt.Printf("LOG: Dialing machine %d\n", numMachine)
		go CommunicateWithMachine(numMachine, conn, &wg)
	}
	wg.Wait()
	fmt.Printf("Total Line Count: %d\n", totalLineCount)
}

// gets each machine information inside of config.json
func getConfig(path string) ([]MachineConfig, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return config.Machines, nil
}

type MachineConfig struct {
	Host string
	Port string
}

type Config struct {
	Machines []MachineConfig `json:"machines"`
}
