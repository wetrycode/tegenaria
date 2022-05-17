/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wetrycode/tegenaria"
)

var rootEngine *tegenaria.SpiderEngine

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tegenaria",
	Short: "tegenaria is a crawler framework based on golang",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}
var crawlCmd = &cobra.Command{
	Use:   "crawl",
	Short: "Start spider by name",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		rootEngine.Start(args[0])
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(engine *tegenaria.SpiderEngine) {
	rootEngine = engine
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tegenaria.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(crawlCmd)
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
