package main

import (
	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "fs [dir]",
		Short: "Start the fs server",
		Args:  cobra.ExactArgs(1),
	}
	flagHttpPort := cmd.Flags().IntP("port", "p", 8090, "http port")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		s := NewFsServer(args[0], *flagHttpPort)
		return s.Start()
	}

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
