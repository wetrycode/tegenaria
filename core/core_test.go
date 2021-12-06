package core_test

import (
	"context"
	"testing"

	"github.com/geebytes/go-scrapy/core"
)

func TestRequestGet(t *testing.T) {
	request := core.NewRequest("http://httpbin.org/get", core.GET)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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
		"intparams":  1,
		"boolparams": false,
	}
	request := core.NewRequest("http://httpbin.org/post", core.POST, core.WithRequestBody(body))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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

func TestRequestCookie(t *testing.T) {
	cookies := map[string]string{
		"test1": "test1",
		"test2": "test2",
	}
	request := core.NewRequest("http://httpbin.org/post", core.POST, core.WithRequestCookies(cookies))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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

func TestRequestQueryParams(t *testing.T) {
	params := map[string]string{
		"query1": "query",
		"query2": "1",
		"query3": "true",
	}
	request := core.NewRequest("http://httpbin.org/get", core.GET, core.WithRequestParams(params))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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

func TestRequestProxy(t *testing.T) {
	request := core.NewRequest("http://httpbin.org/get", core.GET, core.WithRequestProxy("192.168.154.1:1081"))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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
func TestRequestTLS(t *testing.T) {
	request := core.NewRequest("https://httpbin.org/get", core.GET, core.WithRequestTLS(true))
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()
	resultChan := make(chan core.Result, 1)
	go core.GoSpiderDownloader.Do(ctx, request, resultChan)
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
