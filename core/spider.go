package core

type Spider struct{
	Name string // spdier name
	Urls []string  // Feeds url
}
func NewSpider(name string, urls []string) * Spider{
	return &Spider{
		Name: name,
		Urls: urls,
	}
}
type SpiderProcesser interface {
	StartRequest() error
	Parser() (ItemProcesser, error)
	ErrorHandler()

}

