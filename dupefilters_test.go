package tegenaria

import (
	"testing"
)

func TestDoDupeFilter(t *testing.T) {
	server := newTestServer()
	// downloader := NewDownloader()
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request1 := NewRequest(server.URL+"/testHeader", GET, &TestParser{}, RequestWithRequestHeader(headers))
	request2 := NewRequest(server.URL+"/testHeader", GET, &TestParser{}, RequestWithRequestHeader(headers))
	request3 := NewRequest(server.URL+"/testHeader2", GET, &TestParser{}, RequestWithRequestHeader(headers))
	spider1 := &TestSpider{
		NewBaseSpider("testSpiderParseError", []string{"https://www.baidu.com"}),
	}
	ctx1 := NewContext(request1, spider1)
	ctx2 := NewContext(request2, spider1)
	ctx3 := NewContext(request3, spider1)

	duplicates := NewRFPDupeFilter(1024*1024, 5)
	if r1, _ := duplicates.DoDupeFilter(ctx1); r1 {
		t.Errorf("Request1 error expected=%v, get=%v", false, true)
	}
	if r2, _ := duplicates.DoDupeFilter(ctx2); !r2 {
		t.Errorf("Request2  error expected=%v, get=%v", true, false)

	}
	if r3, _ := duplicates.DoDupeFilter(ctx3); r3 {
		t.Errorf("Request3  error expected=%v, get=%v", false, true)

	}
}

func TestDoBodyDupeFilter(t *testing.T) {
	server := newTestServer()
	// downloader := NewDownloader()
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	body := make(map[string]interface{})
	body["test"] = "tegenaria"

	body1 := make(map[string]interface{})
	body1["test"] = "tegenaria2"
	request1 := NewRequest(server.URL+"/testHeader", GET, &TestParser{}, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
	request2 := NewRequest(server.URL+"/testHeader", GET, &TestParser{}, RequestWithRequestHeader(headers), RequestWithRequestBody(body))
	request3 := NewRequest(server.URL+"/testHeader2", GET, &TestParser{}, RequestWithRequestHeader(headers))
	request4 := NewRequest(server.URL+"/testHeader", GET, &TestParser{}, RequestWithRequestHeader(headers), RequestWithRequestBody(body))

	spider1 := &TestSpider{
		NewBaseSpider("testSpiderParseError", []string{"https://www.baidu.com"}),
	}
	ctx1 := NewContext(request1, spider1)
	ctx2 := NewContext(request2, spider1)
	ctx3 := NewContext(request3, spider1)
	ctx4 := NewContext(request4, spider1)

	duplicates := NewRFPDupeFilter(1024*1024, 5)
	if r1, err := duplicates.DoDupeFilter(ctx1); r1 || err != nil {
		t.Errorf("Request1 error expected=%v, get=%v", false, true)
	}
	if r2, err := duplicates.DoDupeFilter(ctx2); !r2 || err != nil {
		t.Errorf("Request2 error expected=%v, get=%v", true, false)

	}
	if r3, err := duplicates.DoDupeFilter(ctx3); r3 || err != nil {
		t.Errorf("Request3 error expected=%v, get=%v", false, true)

	}
	if r4, err := duplicates.DoDupeFilter(ctx4); !r4 || err != nil {
		t.Errorf("Request4 error expected=%v, get=%v", true, false)

	}

}
