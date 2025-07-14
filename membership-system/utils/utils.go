package utils

import "fmt"

type RPCResponseJSON struct {
	Entries   string
	LineCount int
	Err       error
}

func GetAddress(hostname string, port string) string {
	return fmt.Sprintf("%s:%s", hostname, port)
}
