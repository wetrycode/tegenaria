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
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/elliotchance/redismock/v8"
	"github.com/go-redis/redis/v8"
)

// newTestRedis returns a redis.Cmdable.
func newTestRedis() *redismock.ClientMock {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return redismock.NewNiceMock(client)
}
func TestRequestSerialization(t *testing.T) {
	server := newTestServer()
	gob.Register(&TestParser{})
	gob.Register(Proxy{})
	gob.Register(&Request{})
	gob.Register(&RequestResult{})
	gob.Register(&TestSpider{})
	gob.Register(&Response{})
	gob.Register(bytes.Buffer{})
	gob.Register(&HandleError{})

	request := NewRequest(server.URL+"/testGET", GET, &TestParser{})
	spider := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	ctx := NewContext(request, spider)
	distributed := DistributedCache{}
	// db, _ := redismock.NewClientMock()

	distributed.RDB = newTestRedis()
	buffer, err := distributed.encode(ctx)
	if err != nil {
		t.Errorf("Encode request fail with error %s", err.Error())
	}
	req, err := distributed.decode(buffer)
	if err != nil {
		t.Errorf("Decode request fail with error %s", err.Error())
	}
	if req.Request.Url != server.URL+"/testGET" {
		t.Errorf("Request URL except %s but get %s", server.URL, req.Request.Url)

	}
}

func TestRedisBloomFilter(t *testing.T) {
	gob.Register(&TestParser{})
	gob.Register(Proxy{})
	gob.Register(&Request{})
	gob.Register(&RequestResult{})
	gob.Register(&TestSpider{})
	gob.Register(&Response{})
	gob.Register(bytes.Buffer{})
	gob.Register(&HandleError{})
	server := newTestServer()

	FRP := NewRedisBloomFilter(0.0001, 10000)


	FRP.RDB = newTestRedis()
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

	if r1, err := FRP.DoDupeFilter(ctx1); r1 {
		if err != nil {
			t.Errorf("Request1  error %s", err.Error())

		}
		t.Errorf("Request1 error expected=%v, get=%v", false, true)
	}
	if r2, err := FRP.DoDupeFilter(ctx2); !r2 {
		if err != nil {
			t.Errorf("Request2  error %s", err.Error())

		}
		t.Errorf("Request2  error expected=%v, get=%v", true, false)

	}
	if r3, err := FRP.DoDupeFilter(ctx3); r3 {
		if err != nil {
			t.Errorf("Request3  error %s", err.Error())

		}
		t.Errorf("Request3  error expected=%v, get=%v", false, true)

	}
}
