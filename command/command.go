// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package command

import (
	"github.com/spf13/cobra"
	"github.com/wetrycode/tegenaria"
)

var logger = tegenaria.GetLogger("command")

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "crawl spiderName",
	Short: "tegenaria is a crawler framework based on golang",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// ExecuteCmd manage engine by command
func ExecuteCmd(engine *tegenaria.CrawlEngine) {

	var crawlCmd = &cobra.Command{
		Use:   "crawl spiderName",
		Short: "Start spider by name",
		// Uncomment the following line if your bare application
		// has an action associated with it:

		Run: func(_ *cobra.Command, args []string) {
			logger.Infof("准备启动%s爬虫", args[0])
			engine.Execute(args[0])
		},
	}
	rootCmd.AddCommand(crawlCmd)

	err := rootCmd.Execute()
	if err != nil {
		panic(err.Error())
	}

}
