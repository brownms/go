package helpers

import "fmt"

func Log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
