package util

import (
	"fmt"
)

func Log(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
