package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

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
			timed  bool
		)

		flag.StringVar(&part, "part", "", "Path to syntax part")
		flag.BoolVar(&decomp, "decomp", false, "Show decompiled")
		flag.BoolVar(&timed, "timed", false, "Measure and show time")

		flag.Parse()

		if decomp {
			p := parts.GetParserWithSource(string(stdinBytes), "./")

			btc, err := p.ParseAll()

			if err != nil {
				panic(err)
			}

			fmt.Printf("Bytecode: %v\n", btc)

			println("Literals:")

			for idx, val := range p.Literals {
				switch val.LiteralType {
				case parts.RefLiteral:
					fmt.Printf("Ref bytecode at idx %d - %s \n", idx, val.Value)
				case parts.FunLiteral:
					fmt.Printf("Function bytecode at idx %d - %v \n", idx, val.Value.(parts.FunctionDeclaration).Body)
				case parts.ListLiteral:
					fmt.Printf("List bytecode at idx %d - %v \n", idx, val.Value.(parts.ListDefinition).Entries)
				case parts.ObjLiteral:
					fmt.Printf("Object bytecode at idx %d - %v \n", idx, val.Value.(parts.ObjDefinition).Entries)
				}
			}

			return
		}

		if part == "" {
			startTime := time.Now()

			_, err := parts.RunString(string(stdinBytes), "./")

			if err != nil {
				panic(err)
			}

			if timed {
				fmt.Printf("Execution took - %s\n", time.Now().Sub(startTime).String())
			}
		} else {
			rawFile, err := os.ReadFile(part)

			if err != nil {
				panic(err)
			}

			startTime := time.Now()

			_, err = parts.RunStringWithSyntax(string(stdinBytes), string(rawFile), "./")

			if err != nil {
				panic(err)
			}

			if timed {
				fmt.Printf("Execution took - %s\n", time.Now().Sub(startTime).String())
			}
		}

		return
	} else {
		var (
			codePath string
			part     string
			decomp   bool
			timed    bool
		)

		flag.StringVar(&codePath, "code", "", "Path to code to execute")
		flag.StringVar(&part, "part", "", "Path to syntax part")
		flag.BoolVar(&decomp, "decomp", false, "Show decompiled")
		flag.BoolVar(&timed, "timed", false, "Measure and show time")
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

			fmt.Printf("Bytecode: %v", btc)

			println("Literals:")

			for idx, val := range p.Literals {
				switch val.LiteralType {
				case parts.FunLiteral:
					fmt.Printf("Function bytecode at idx %d - %v \n", idx, val.Value.(parts.FunctionDeclaration).Body)
				case parts.ListLiteral:
					fmt.Printf("List bytecode at idx %d - %v \n", idx, val.Value.(parts.ListDefinition).Entries)
				case parts.ObjLiteral:
					fmt.Printf("List bytecode at idx %d - %v \n", idx, val.Value.(parts.ObjDefinition).Entries)
				}
			}

			return
		}

		if part == "" {
			startTime := time.Now()

			_, err := parts.RunString(string(codeData), codePath)

			if err != nil {
				panic(err)
			}

			if timed {
				fmt.Printf("Execution took - %s\n", time.Now().Sub(startTime).String())
			}
		} else {
			rawFile, err := os.ReadFile(part)

			if err != nil {
				panic(err)
			}

			startTime := time.Now()

			_, err = parts.RunStringWithSyntax(string(codeData), string(rawFile), codePath)

			if err != nil {
				panic(err)
			}

			if timed {
				fmt.Printf("Execution took - %s\n", time.Now().Sub(startTime).String())
			}
		}
	}
}
