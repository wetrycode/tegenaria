// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package distributed

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestRdbClusterCLient(t *testing.T) {
	convey.Convey("test rdb cluster connect", t, func() {
		mockRedis := miniredis.RunT(t)
		nodes := []string{mockRedis.Addr()}
		worker := NewRdbClusterCLient(&WorkerConfigWithRdbCluster{
			DistributedWorkerConfig: &DistributedWorkerConfig{
				redisConfig: NewRedisConfig("", "", "", 0),
				getLimitKey: getLimiterDefaultKey,
			},
			RdbNodes: nodes,
		})
		defer mockRedis.Close()
		status, err := worker.Set(context.TODO(), "testCluster", "test", 0).Result()
		if err != nil {
			t.Errorf("setup redis cluster client fail %s", err.Error())
		}
		if status != "OK" {
			t.Errorf("redis cluster client set fail")

		}
	})
	convey.Convey("test rdb connect error", t, func() {
		mockRedis := miniredis.RunT(t)
		addr := mockRedis.Addr()
		mockRedis.Close()
		f := func() {
			NewDistributedWorker(NewDistributedWorkerConfig(NewRedisConfig(addr, "", "", 0), &InfluxdbConfig{}))
		}
		convey.So(f, convey.ShouldPanic)
	})
	convey.Convey("test rdb cluster connect error", t, func() {
		mockRedis := miniredis.RunT(t)

		nodes := []string{mockRedis.Addr()}
		mockRedis.Close()
		f := func() {
			NewRdbClusterCLient(&WorkerConfigWithRdbCluster{
				DistributedWorkerConfig: &DistributedWorkerConfig{
					redisConfig: NewRedisConfig("", "", "", 0),

					getLimitKey: getLimiterDefaultKey,
				},
				RdbNodes: nodes,
			})
		}
		convey.So(f, convey.ShouldPanic)
	})

}
