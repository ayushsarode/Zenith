package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func main() {
	if err := rootCmd.Execute(); err!= nil {
		fmt.Println(err);
		os.Exit(1)
	}
}