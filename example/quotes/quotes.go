package quotes

import (
	"crypto/tls"

	"github.com/wetrycode/tegenaria"
)

// var ExampleSpiderInstance *ExampleSpider
// var once sync.Once
// var FileDownloader tegenaria.Downloader
// var IpfsFileDownloader tegenaria.Downloader
// var Engine *tegenaria.CrawlEngine
func NewQuotesEngine(opts ...tegenaria.EngineOption) *tegenaria.CrawlEngine {
	ExampleSpiderInstance := &ExampleSpider{
		Name:     "example",
		FeedUrls: []string{"http://quotes.toscrape.com/"},
	}
	// tegenaria.EngineWithUniqueReq(true), tegenaria.EngineWithDownloader(Downloader), tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(128))
	Downloader := tegenaria.NewDownloader(tegenaria.DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true, MaxVersion: tls.VersionTLS12}))
	opts = append(opts, tegenaria.EngineWithDownloader(Downloader))
	Engine := tegenaria.NewEngine(opts...)
	Engine.RegisterSpiders(ExampleSpiderInstance)
	// register item handle pipelines
	pipe1 := QuotesbotItemPipeline{Priority: 1}
	pipe2 := QuotesbotItemPipeline2{Priority: 2}
	pipe3 := QuotesbotItemPipeline3{Priority: 3}

	Engine.RegisterPipelines(&pipe1)
	Engine.RegisterPipelines(&pipe2)
	Engine.RegisterPipelines(&pipe3)
	// Register Downloade middlers
	middleware := HeadersDownloadMiddler{Priority: 1, Name: "AddHeader"}
	Engine.RegisterDownloadMiddlewares(middleware)
	return Engine

}

// func init() {
// 	once.Do(func() {
// 		ExampleSpiderInstance = &ExampleSpider{
// 			Name:     "example",
// 			FeedUrls: []string{"http://quotes.toscrape.com/"},
// 		}
// 		Downloader = tegenaria.NewDownloader(tegenaria.DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true, MaxVersion: tls.VersionTLS12}))
// 		Engine = tegenaria.NewEngine(tegenaria.EngineWithUniqueReq(true), tegenaria.EngineWithDownloader(Downloader), tegenaria.EngineWithUniqueReq(false), tegenaria.EngineWithLimiter(tegenaria.NewDefaultLimiter(128)))
// 		Engine.RegisterSpiders(ExampleSpiderInstance)
// 		// register item handle pipelines
// 		pipe1 := QuotesbotItemPipeline{Priority: 1}
// 		pipe2 := QuotesbotItemPipeline2{Priority: 2}
// 		pipe3 := QuotesbotItemPipeline3{Priority: 3}

// 		Engine.RegisterPipelines(&pipe1)
// 		Engine.RegisterPipelines(&pipe2)
// 		Engine.RegisterPipelines(&pipe3)
// 		// Register Downloade middlers
// 		middleware := HeadersDownloadMiddler{Priority: 1, Name: "AddHeader"}
// 		Engine.RegisterDownloadMiddlewares(middleware)
// 	})
// }

// func Start() {
// 	// cmd.Execute(Engine)
// 	Engine:=NewQuotesEngine()
// 	Engine.Start("example")
// }
