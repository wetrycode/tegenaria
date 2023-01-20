package tegenaria

import (
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xxx",
	Short: "tegenaria is a crawler framework based on golang",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteCmd(engine *CrawlEngine) {
	var crawlCmd = &cobra.Command{
		Use:   "crawl",
		Short: "Start spider by name",
		// Uncomment the following line if your bare application
		// has an action associated with it:
		Run: func(_ *cobra.Command, args []string) {
			engineLog.Infof("准备启动%s爬虫", args[0])
			engine.start(args[0])
		},
	}
	rootCmd.AddCommand(crawlCmd)

	crawlCmd.Flags().BoolVarP(&engine.isMaster, "master", "m", false, "Whether to set the current node as the master node,defualt false")
	rootCmd.Execute()
}
