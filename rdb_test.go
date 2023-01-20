package tegenaria

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/smartystreets/goconvey/convey"
)

func TestRdbClusterCLient(t *testing.T) {
	convey.Convey("test rdb cluster connect", t, func() {
		mockRedis := miniredis.RunT(t)
		nodes := []string{mockRedis.Addr()}
		worker := NewRdbClusterCLient(&WorkerConfigWithRdbCluster{
			DistributedWorkerConfig: &DistributedWorkerConfig{
				RedisAddr:          "",
				RedisPasswd:        "",
				RedisUsername:      "",
				RedisDB:            0,
				RdbConnectionsSize: 15,
				RdbTimeout:         5 * time.Second,
				RdbMaxRetry:        3,
				getLimitKey:        getLimiterDefaultKey,
				GetqueueKey:        getQueueDefaultKey,
				GetBFKey:           getBloomFilterDefaultKey,
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
		addr:=mockRedis.Addr()
		mockRedis.Close()
		f := func() {
			NewDistributedWorker(addr, NewDistributedWorkerConfig("","",0))
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
					RedisAddr:          "",
					RedisPasswd:        "",
					RedisUsername:      "",
					RedisDB:            0,
					RdbConnectionsSize: 15,
					RdbTimeout:         5 * time.Second,
					RdbMaxRetry:        3,
					getLimitKey:        getLimiterDefaultKey,
					GetqueueKey:        getQueueDefaultKey,
					GetBFKey:           getBloomFilterDefaultKey,
				},
				RdbNodes: nodes,
			})
		}
		convey.So(f, convey.ShouldPanic)
	})

}
