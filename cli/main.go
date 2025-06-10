package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/tfo-dot/parts"
)

func main() {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {

		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading all from stdin:", err)
			os.Exit(1)
		}

		_, err = parts.RunString(string(stdinBytes))

		if err != nil {
			panic(err)
		}

		return
	} else {
		var (
			code string
		)

		flag.StringVar(&code, "code", "", "Code to execute")
		flag.Parse()

		if code == "" {
			fmt.Println("Error: --code flag is required.")
			os.Exit(1)
		}

		_, err := parts.RunString(code)

		if err != nil {
			panic(err)
		}
	}
}
