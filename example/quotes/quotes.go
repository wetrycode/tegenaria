package quotes

import (
	"crypto/tls"

	"github.com/wetrycode/tegenaria"
)

// NewQuotesEngine 创建引擎
func NewQuotesEngine(opts ...tegenaria.EngineOption) *tegenaria.CrawlEngine {
	ExampleSpiderInstance := &ExampleSpider{
		Name:     "example",
		FeedUrls: []string{"http://quotes.toscrape.com/"},
	}
	// 设置下载组件
	Downloader := tegenaria.NewDownloader(tegenaria.DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true, MaxVersion: tls.VersionTLS12}))
	opts = append(opts, tegenaria.EngineWithDownloader(Downloader))
	Engine := tegenaria.NewEngine(opts...)
	Engine.RegisterSpiders(ExampleSpiderInstance)
	// 实例化pipeline
	pipe1 := QuotesbotItemPipeline{Priority: 1}
	pipe2 := QuotesbotItemPipeline2{Priority: 2}
	pipe3 := QuotesbotItemPipeline3{Priority: 3}
	// 向引擎注册pipe
	Engine.RegisterPipelines(&pipe1)
	Engine.RegisterPipelines(&pipe2)
	Engine.RegisterPipelines(&pipe3)
	middleware := HeadersDownloadMiddler{Priority: 1, Name: "AddHeader"}
	Engine.RegisterDownloadMiddlewares(middleware)
	return Engine

}