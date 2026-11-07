package main

import (
	"flag"
	"fmt"
	"io"
)

func parseArgs(args []string) (bool, error) {
	flags := flag.NewFlagSet("chess-server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	migrateOnly := flags.Bool("migrate", false, "apply database migrations and exit")
	if err := flags.Parse(args); err != nil {
		return false, err
	}
	if flags.NArg() != 0 {
		return false, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return *migrateOnly, nil
}
