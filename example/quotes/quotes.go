package quotes

import (
	"crypto/tls"
	"sync"

	"github.com/wetrycode/tegenaria"
)

var ExampleSpiderInstance *ExampleSpider
var once sync.Once
var Downloader tegenaria.Downloader
var FileDownloader tegenaria.Downloader
var IpfsFileDownloader tegenaria.Downloader
var Engine *tegenaria.CrawlEngine

func init() {
	once.Do(func() {
		ExampleSpiderInstance = &ExampleSpider{
			Name:     "example",
			FeedUrls: []string{"http://192.168.3.89:12138/"},
		}
		Downloader = tegenaria.NewDownloader(tegenaria.DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true, MaxVersion: tls.VersionTLS12}))
		FileDownloader = tegenaria.NewDownloader(tegenaria.DownloadWithTlsConfig(&tls.Config{InsecureSkipVerify: true}))
		IpfsFileDownloader = tegenaria.NewDownloader()

		Engine = tegenaria.NewEngine(tegenaria.EngineWithUniqueReq(true), tegenaria.EngineWithDownloader(Downloader), tegenaria.EngineWithUniqueReq(false))
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
	})
}

func Start() {
	// cmd.Execute(Engine)
	Engine.Start("example")
}
