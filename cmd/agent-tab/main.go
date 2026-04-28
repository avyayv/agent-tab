package main

import (
	"fmt"
	"os"

	"github.com/avyayv/agent-tab/internal/agenttab"
)

func main() {
	if err := agenttab.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
