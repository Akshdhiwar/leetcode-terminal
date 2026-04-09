package main

import (
	"os"

	"github.com/user/leetcode-cli/internal/cli"
)

func main() {
	app := cli.NewApp()
	if err := app.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
