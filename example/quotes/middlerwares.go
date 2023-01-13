package quotes

import "github.com/wetrycode/tegenaria"

type HeadersDownloadMiddler struct {
	Priority int
	Name     string
}
type ProxyDownloadMiddler struct {
	Priority int
	Name     string
}

func (m HeadersDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m HeadersDownloadMiddler) ProcessRequest(ctx *tegenaria.Context) error {
	header := map[string]string{
		"Accept":       "*/*",
		"Content-Type": "application/json",
		
		"User-Agent":   "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.64 Safari/537.36",
	}

	for key, value := range header {
		if _, ok := ctx.Request.Header[key]; !ok {
			ctx.Request.Header[key] = value
		}
	}
	// ctx.Request.Header = header
	return nil
}

func (m HeadersDownloadMiddler) ProcessResponse(ctx *tegenaria.Context, req chan<- *tegenaria.Context) error {
	return nil

}
func (m HeadersDownloadMiddler) GetName() string {
	return m.Name
}