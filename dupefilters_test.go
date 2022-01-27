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

	duplicates := NewRFPDupeFilter(1024*1024, 5)
	if r1, _ := duplicates.DoDupeFilter(request1); r1 {
		t.Errorf("Request1 igerprint sum error expected=%v, get=%v", false, true)
	}
	if r2, _ := duplicates.DoDupeFilter(request2); !r2 {
		t.Errorf("Request2 igerprint sum error expected=%v, get=%v", true, false)

	}
	if r3, _ := duplicates.DoDupeFilter(request3); r3 {
		t.Errorf("Request3 igerprint sum error expected=%v, get=%v", false, true)

	}
}
