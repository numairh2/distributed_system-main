package machine

import (
	"fmt"
	"membership-system/utils"
	"net/rpc"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHandleLog(t *testing.T) {
	machineNum := 1
	err := os.WriteFile("../machine.1.log", []byte("Hello \nHi \nHello again \nHiiii \n"), os.ModePerm)
	if err != nil {
		t.Fatalf("Failed to create initial file")
	}
	defer os.Remove("../machine.1.log")

	lh := LogHandler{}
	grepCommandArgs := []string{"grep", "Hello"}
	grepCommandArgs = append(grepCommandArgs, fmt.Sprintf("%d", machineNum))

	result := utils.RPCResponseJSON{}
	err = lh.HandleLog(grepCommandArgs, &result)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result.Entries, "machine.1.log") {
		t.Fatalf("Expected log entries to contain 'machine.1.log', got %s", result.Entries)
	}
	if result.LineCount != 2 {
		t.Fatalf("Expected 2 matching lines, got %d", result.LineCount)
	}
}

func TestStartMachine(t *testing.T) {
	machineNum := 1
	err := os.WriteFile("../machine.1.log", []byte("Hello \nhi Hello \nHello again \nHiiii \n"), os.ModePerm)
	if err != nil {
		t.Fatalf("Failed to create initial file")
	}
	defer os.Remove("../machine.1.log")

	go func() {
		StartMachine("localhost", "1111")
	}()
	time.Sleep(1 * time.Second)

	client, err := rpc.Dial("tcp", utils.GetAddress("localhost", "1111"))
	if err != nil {
		t.Fatalf("Error dialing: %v", err)
	}
	defer client.Close()

	var result utils.RPCResponseJSON
	grepCommandArgs := []string{"grep", "Hello"}
	grepCommandArgs = append(grepCommandArgs, fmt.Sprintf("%d", machineNum))

	err = client.Call("LogHandler.HandleLog", grepCommandArgs, &result)
	if err != nil {
		t.Fatalf("RPC call failed: %v", err)
	}

	if !strings.Contains(result.Entries, "machine.1.log") {
		t.Fatalf("Expected log entries to contain 'machine.1.log', got %s", result.Entries)
	}
	if result.LineCount != 3 {
		t.Fatalf("Expected 3 matching lines, got %d", result.LineCount)
	}
}
