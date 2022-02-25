// The store command displays detailed information about FreeTSDB data files.
package main

import (
	"fmt"
	"io"
	"os"

	"influxdb.cluster/cmd"
	"influxdb.cluster/cmd/store/help"
	"influxdb.cluster/cmd/store/query"
	"influxdb.cluster/logger"
	_ "influxdb.cluster/tsdb/engine"
	"go.uber.org/zap"
)

func main() {
	m := NewMain()
	if err := m.Run(os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program execution.
type Main struct {
	Logger *zap.Logger

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Logger: logger.New(os.Stderr),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Run determines and runs the command specified by the CLI args.
func (m *Main) Run(args ...string) error {
	name, args := cmd.ParseCommandName(args)

	// Extract name from args.
	switch name {
	case "", "help":
		if err := help.NewCommand().Run(args...); err != nil {
			return fmt.Errorf("help: %s", err)
		}
	case "query":
		name := query.NewCommand()
		name.Logger = m.Logger
		if err := name.Run(args...); err != nil {
			return fmt.Errorf("query: %s", err)
		}
	default:
		return fmt.Errorf(`unknown command "%s"`+"\n"+`Run 'store help' for usage`+"\n\n", name)
	}

	return nil
}
