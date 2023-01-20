package tegenaria

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestRdbClusterCLient(t *testing.T) {
	mockRedis, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	nodes:=[]string{mockRedis.Addr()}
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
	status,err:=worker.Set(context.TODO(),"testCluster","test", 0).Result()
	if err!=nil{
		t.Errorf("setup redis cluster client fail %s", err.Error())
	}
	if status !="OK"{
		t.Errorf("redis cluster client set fail")

	}
}
