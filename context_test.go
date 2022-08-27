// Copyright 2022 geebytes
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tegenaria

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWithDeadline(t *testing.T) {
	server := newTestServer()
	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	deadLine, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*3))
	defer cancel()
	t1 := time.Now() // get current time

	ctx := NewContext(request, spider1, WithContext(deadLine))
	<-ctx.Done()
	elapsed := time.Since(t1)
	if elapsed.Seconds() < 3.0 {
		t.Errorf("Deadline fail")
	}

}

func TestWithTimeout(t *testing.T) {
	server := newTestServer()
	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	request := NewRequest(server.URL+"/testGET", GET, testParser)
	timeout, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	ctx := NewContext(request,spider1, WithContext(timeout))
	time.Sleep(time.Second * 5)
	<-ctx.Done()
	if !strings.Contains(ctx.Err().Error(), "context deadline exceeded") {
		t.Errorf("Timeout fail it should context deadline exceeded")
	}

}

func TestWithValue(t *testing.T) {
	type ContextKey string
	k := ContextKey("test_key")
	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	server := newTestServer()

	request := NewRequest(server.URL+"/testGET", GET, testParser)
	valueCtx := context.WithValue(context.Background(), k, "tegenaria")

	ctx := NewContext(request,spider1, WithContext(valueCtx))
	if ctx.Value(k).(string) != "tegenaria" {
		t.Errorf("Set context value fail it should tegenaria")

	}
	if ctx.GetCtxId() == "" {
		t.Errorf("Get context id value fail")

	}

}
func TestWithEmptyContext(t *testing.T) {
	server := newTestServer()
	spider1 := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	request := NewRequest(server.URL+"/testGET", GET, testParser)

	ctx := NewContext(request, spider1)
	c :=ctx.Done()
	if c ==nil{
		t.Errorf("Context done channel is nil")

	}
	_, ok:=ctx.Deadline()
	if ok{
		t.Errorf("Context deadline is not false")

	}
	if ctx.Err() !=nil {
		t.Errorf("Context error is not nil")

	}
	type ContextKey string
	k := ContextKey("test_key")
	if ctx.Value(k) !=nil{
		t.Errorf("Context value is not nil")

	}

}
