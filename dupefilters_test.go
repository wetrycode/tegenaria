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
	request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
	request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers))
	request3 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))

	duplicates := NewRFPDupeFilter(0.001,1024*1024)
	if r1, _ := duplicates.DoDupeFilter(request1); r1 {
		t.Errorf("Request1 error expected=%v, get=%v", false, true)
	}
	if r2, _ := duplicates.DoDupeFilter(request2); !r2 {
		t.Errorf("Request2  error expected=%v, get=%v", true, false)

	}
	if r3, _ := duplicates.DoDupeFilter(request3); r3 {
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
	request1 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers),RequestWithRequestBody(body))
	request2 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers),RequestWithRequestBody(body))
	request3 := NewRequest(server.URL+"/testHeader2", GET, testParser, RequestWithRequestHeader(headers))
	request4 := NewRequest(server.URL+"/testHeader", GET, testParser, RequestWithRequestHeader(headers),RequestWithRequestBody(body))
	duplicates := NewRFPDupeFilter(0.001,1024*1024)
	if r1, err := duplicates.DoDupeFilter(request1); r1||err!=nil {
		t.Errorf("Request1 error expected=%v, get=%v", false, true)
	}
	if r2, err := duplicates.DoDupeFilter(request2); !r2||err!=nil {
		t.Errorf("Request2 error expected=%v, get=%v", true, false)

	}
	if r3, err := duplicates.DoDupeFilter(request3); r3||err!=nil {
		t.Errorf("Request3 error expected=%v, get=%v", false, true)

	}
	if r4, err := duplicates.DoDupeFilter(request4); !r4||err!=nil {
		t.Errorf("Request4 error expected=%v, get=%v", true, false)

	}
	
}
