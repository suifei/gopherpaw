// GopherPaw is the CoPaw Go language reimplementation.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "gopherpaw: %v\n", err)
		os.Exit(1)
	}
}
