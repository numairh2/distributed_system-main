package caller

import (
	"fmt"
	"membership-system/utils"
	"os"
	"sync"
	"testing"
)

type MockRPCClient struct {
	Callable
}

func (client *MockRPCClient) Call(serviceMethod string, grepCommandArgs any, grepCommandResult any) error {
	if serviceMethod != "LogHandler.HandleLog" {
		return fmt.Errorf("incorrect serviceMethod")
	}
	grepCommandResultPtr := grepCommandResult.(*utils.RPCResponseJSON)
	*grepCommandResultPtr = utils.RPCResponseJSON{Entries: "UNITTEST OUTPUT \nHere is the command output \n Here is another line of output", Err: nil}
	return nil
}

func TestCommunicateWithMachine(t *testing.T) {
	err := os.MkdirAll("result", 0777)
	if err != nil {
		t.Fatalf("Failed to make test dir: %v", err)
	}
	machineNum := 1
	client := MockRPCClient{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	err = CommunicateWithMachine(machineNum, &client, &wg)
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected_output := "UNITTEST OUTPUT \nHere is the command output \n Here is another line of output"
	path := fmt.Sprintf("result/output.%d.txt", machineNum)
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Could not find file %s => %v", path, err)
	}
	err = os.RemoveAll("result")
	if err != nil {
		t.Fatalf("Failed to remove test dir: %v", err)
	}
	if expected_output != string(bytes) {
		t.Fatalf("Output does not match. %s != %s", expected_output, string(bytes))
	}
}

func TestWriteOutputToFile(t *testing.T) {
	machineNum := 1
	output := "UNITTEST OUTPUT \nThis is the output from a grep command \nHere is another line \n\nHere is a final line"
	path := fmt.Sprintf("../result/output.%d.txt", machineNum)
	err := WriteOutputToFile(path, output)
	if err != nil {
		t.Fatalf(`WriteOutputToFile(machineNum, output) => %v`, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Could not find file %s => %v", path, err)
	}
	if output != string(bytes) {
		t.Fatalf("Output does not match. %s != %s", output, string(bytes))
	}
}

func TestOverwrite_WriteOutputToFile(t *testing.T) {
	machineNum := 1
	another_output := "UNITTEST OUTPUT \nThis is another output from a grep command \nHere is ANOTHER line \n\nHere is another final line"
	path := fmt.Sprintf("../result/output.%d.txt", machineNum)
	err := WriteOutputToFile(path, another_output)
	if err != nil {
		t.Fatalf(`Another WriteOutputToFile(machineNum, output) => %v`, err)
	}
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Could not find file %s => %v", path, err)
	}
	if another_output != string(bytes) {
		t.Fatalf("Output does not match. %s != %s", another_output, string(bytes))
	}
}

func TestGetConfig(t *testing.T) {
	path := "../config.json"
	machineConfigs, err := getConfig(path)
	if err != nil {
		t.Fatalf("Could not get configs => %v", err)
	}
	// Bare test
	if len(machineConfigs) != 10 {
		t.Fatalf("Incorrect number of machine configs. 10 != %d", len(machineConfigs))
	}
	if machineConfigs[0].Host != "fa24-cs425-3501.cs.illinois.edu" {
		t.Fatalf("Host mismatch on first machine")
	}
	if machineConfigs[9].Port != "1111" {
		t.Fatalf("Port mismatch on last machine")
	}
}
