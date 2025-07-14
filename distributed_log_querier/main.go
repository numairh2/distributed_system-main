package main

import (
	"distributed_log_querier/caller"
	"distributed_log_querier/machine"
	"os"
)

func main() {
	args := os.Args
	// typeArg is either "machine" or "caller", otherwise invalid
	typeArg := args[1]
	if typeArg == "machine" {
		machine.StartMachine("localhost", "1111")
	}
	if typeArg == "caller" {
		caller.StartCaller()
	}

}
