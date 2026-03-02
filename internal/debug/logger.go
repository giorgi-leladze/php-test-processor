package debug

import (
	"fmt"
	"os"
	"time"
)

var enabled bool

func Enable() {
	enabled = true
}

func IsEnabled() bool {
	return enabled
}

func Log(msg string) {
	if !enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", time.Now().Format("15:04:05.000"), msg)
}

func Logf(format string, args ...interface{}) {
	if !enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[DEBUG %s] %s\n", time.Now().Format("15:04:05.000"), msg)
}
