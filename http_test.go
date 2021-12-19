package tegenaria

import (
	"context"
	"testing"

	bloom "github.com/bits-and-blooms/bloom/v3"
)

func TestRequestGet(t *testing.T) {
	request := NewRequest("http://httpbin.org/get", GET)
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}

}

func TestRequestPost(t *testing.T) {
	body := map[string]interface{}{
		"params1":    "params1",
		"intparams":  1.0,
		"boolparams": false,
	}
	request := NewRequest("http://httpbin.org/post", POST, WithRequestBody(body))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	data, _ := resp.Json()["json"].(map[string]interface{})
	for key, value := range body {
		if val, ok := data[key]; ok {
			if value != val {
				t.Errorf("post request error %s %s %s", key, value, val)
			}
		} else {
			t.Errorf("post request error")
		}
	}

}

func TestRequestCookie(t *testing.T) {
	cookies := map[string]string{
		"test1": "test1",
		"test2": "test2",
	}
	request := NewRequest("http://httpbin.org/cookies", GET, WithRequestCookies(cookies))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Download(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	data, _ := resp.Json()["cookies"].(map[string]interface{})
	for key, value := range cookies {
		if val, ok := data[key]; ok {
			if value != val {
				t.Errorf("cookies request error")
			}
		} else {
			t.Errorf("cookies request error")
		}
	}
	if err != nil {
		t.Errorf("request error with cookies")

	}
	if resp.Status != 200 {
		t.Errorf("response with cookies status = %d; expected %d", resp.Status, 200)

	}
}

func TestRequestQueryParams(t *testing.T) {
	params := map[string]string{
		"query1": "query",
		"query2": "1",
		"query3": "true",
	}
	request := NewRequest("http://httpbin.org/get", GET, WithRequestParams(params))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	data, _ := resp.Json()["args"].(map[string]interface{})
	for key, value := range params {
		if val, ok := data[key]; ok {
			if value != val {
				t.Errorf("params request error")
			}
		} else {
			t.Errorf("params request error")
		}
	}
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status >= 400 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
}

func TestRequestProxy(t *testing.T) {
	request := NewRequest("http://httpbin.org/get", GET, WithRequestProxy("local.proxy:1081"))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	origin, _ := resp.Json()["origin"].(string)
	if origin != "45.63.38.155" {
		t.Errorf("proxy request error")
	}
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status >= 400 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}

}
func TestRequestTLS(t *testing.T) {
	request := NewRequest("https://httpbin.org/get", GET, WithRequestTLS(false))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil {
		t.Errorf("request error")
	}
	if resp.Status >= 400 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
}

func TestRequestHeaders(t *testing.T) {
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request := NewRequest("http://httpbin.org/headers", GET, WithRequestHeader(headers))
	var MainCtx context.Context = context.Background()

	ctx, cancel := context.WithCancel(MainCtx)
	defer func() {
		cancel()
	}()
	resultChan := make(chan *RequestResult, 1)
	go GoSpiderDownloader.Do(ctx, request, resultChan)
	result := <-resultChan
	err := result.Error
	resp := result.Response
	if err != nil {
		t.Errorf("request error")

	}
	if resp.Status != 200 {
		t.Errorf("response status = %d; expected %d", resp.Status, 200)

	}
	respHeaders, _ := resp.Json()["headers"].(map[string]interface{})
	for key, value := range headers {
		if data, ok := respHeaders[key]; ok {
			if value != data {
				t.Errorf("header request error key=%s, value=%s, data=%s", key, value, data)
			}
		} else {
			t.Errorf("header request error %s not in response headers", key)
		}
	}
}
func TestFingerprint(t *testing.T) {
	headers := map[string]string{
		"Params1":    "params1",
		"Intparams":  "1",
		"Boolparams": "false",
	}
	request1 := NewRequest("http://httpbin.org/headers", GET, WithRequestHeader(headers))
	request2 := NewRequest("http://httpbin.org/headers", GET, WithRequestHeader(headers))
	request3 := NewRequest("http://httpbin.org/headers1", GET, WithRequestHeader(headers))

	bloomFilter := bloom.New(1024*1024, 5)
	res1 := bloomFilter.TestOrAdd(request1.Fingerprint())
	res2 := bloomFilter.TestOrAdd(request2.Fingerprint())
	res3 := bloomFilter.TestOrAdd(request3.Fingerprint())
	if res1 {
		t.Errorf("Request1 igerprint sum error expected=%v, get=%v", false, res1)
	}
	if !res2 {
		t.Errorf("Request2 igerprint sum error expected=%v, get=%v", true, res2)

	}
	if res3 {
		t.Errorf("Request3 igerprint sum error expected=%v, get=%v", false, res3)

	}
}