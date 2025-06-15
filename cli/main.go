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

		var (
			part   string
			decomp bool
		)

		flag.StringVar(&part, "part", "", "Path to syntax part")
		flag.BoolVar(&decomp, "decomp", false, "Show decompiled")
		flag.Parse()

		if decomp {
			p := parts.GetParserWithSource(string(stdinBytes), "./")

			btc, err := p.ParseAll()

			if err != nil {
				panic(err)
			}

			fmt.Printf("%v", btc)
			return
		}

		if part == "" {
			_, err := parts.RunString(string(stdinBytes), "./")

			if err != nil {
				panic(err)
			}
		} else {
			rawFile, err := os.ReadFile(part)

			if err != nil {
				panic(err)
			}

			_, err = parts.RunStringWithSyntax(string(stdinBytes), string(rawFile), "./")

			if err != nil {
				panic(err)
			}
		}

		return
	} else {
		var (
			codePath string
			part     string
			decomp   bool
		)

		flag.StringVar(&codePath, "code", "", "Path to code to execute")
		flag.StringVar(&part, "part", "", "Path to syntax part")
		flag.BoolVar(&decomp, "decomp", false, "Show decompiled")
		flag.Parse()

		if codePath == "" {
			fmt.Println("Error: --code flag is required.")
			os.Exit(1)
		}

		codeData, err := os.ReadFile(codePath)

		if err != nil {
			panic(err)
		}

		if decomp {
			p := parts.GetParserWithSource(string(codeData), codePath)

			btc, err := p.ParseAll()

			if err != nil {
				panic(err)
			}

			fmt.Printf("%v", btc)
			return
		}

		if part == "" {
			_, err := parts.RunString(string(codeData), codePath)

			if err != nil {
				panic(err)
			}
		} else {
			rawFile, err := os.ReadFile(part)

			if err != nil {
				panic(err)
			}

			_, err = parts.RunStringWithSyntax(string(codeData), string(rawFile), codePath)

			if err != nil {
				panic(err)
			}
		}
	}
}
