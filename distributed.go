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
	"context"
	"encoding/gob"
	"fmt"
	"hash"
	"math"

	"github.com/go-redis/redis/v8"
	"github.com/spaolacci/murmur3"
)

// DistributedCache distributed spider crawl cache
type DistributedCache struct {
	RDB    redis.Cmdable
	buffer *bytes.Buffer
	keyPrefix string

}

// Message is data struct when save to cache
type Message struct {
	Request Request
	CtxId   string
	Spider  SpiderInterface
}
type RedisBloomFilter struct {
	RDB       redis.Cmdable
	ErrorRate float64
	BitSize   uint64
	FRP       *FingerprintCalculator
	HashFunc  []hash.Hash64
	keyPrefix string
}
func NewDistributedCache() *DistributedCache{
	return &DistributedCache{
		RDB: rdb,
		keyPrefix: "tegenaria:v1:request",
	}
}
// NewRedisBloomFilter
func NewRedisBloomFilter(p float64, n uint64) *RedisBloomFilter {
	var HashFunc []hash.Hash64
	m, k := EstimateParameters(p, n)
	for i := 0; i < int(k); i++ {
		HashFunc = append(HashFunc, murmur3.New64WithSeed(uint32(i)))
	}
	return &RedisBloomFilter{
		RDB:       rdb,
		ErrorRate: p,
		BitSize:   m,
		HashFunc:  HashFunc,
		FRP:       &FingerprintCalculator{},
		keyPrefix: "tegenaria:v1:bloom",
	}
}

//EstimateParameters 计算bit位个数和hash函数个数，
// p是错误率，n是数据规模
func EstimateParameters(p float64, n uint64) (m uint64, k uint) {
	m = uint64(math.Ceil(-1 * float64(n) * math.Log(p) / math.Pow(math.Log(2), 2)))
	k = uint(math.Ceil(math.Log(2) * float64(m) / float64(n)))
	return
}
func (bf *RedisBloomFilter) Add(data []byte, spider SpiderInterface) error {
	key := fmt.Sprintf("%s:%s", bf.keyPrefix, spider.GetName())
	pipeline := bf.RDB.Pipeline()
	for _, hashFunc := range bf.HashFunc {
		_, err := hashFunc.Write(data)
		if err != nil {
			return err
		}
		hashVal := hashFunc.Sum64()
		position := hashVal % bf.BitSize
		pipeline.SetBit(context.TODO(), key, int64(position), 1)
		hashFunc.Reset()

	}
	_, err := pipeline.Exec(context.TODO())
	if err != nil {
		return err
	}
	return nil
}
func (bf *RedisBloomFilter) Exists(data []byte, spider SpiderInterface) (bool, error) {
	key := fmt.Sprintf("%s:%s", bf.keyPrefix, spider.GetName())
	pipeline := bf.RDB.Pipeline()
	result := make([]*redis.IntCmd, 0)

	for _, hashFunc := range bf.HashFunc {
		_, err := hashFunc.Write(data)
		if err != nil {
			return false, err
		}
		hashVal := hashFunc.Sum64()
		position := hashVal % bf.BitSize
		result = append(result, pipeline.GetBit(context.TODO(), key, int64(position)))
		hashFunc.Reset()
	}

	_, err := pipeline.Exec(context.TODO())
	if err != nil {
		return false, err
	}
	for _, rs := range result {
		val, err := rs.Result()
		if err != nil {
			return false, err
		}
		if val == 0 {
			return false, nil
		}
	}
	return true, nil
}


func (bf *RedisBloomFilter) DoDupeFilter(ctx *Context) (bool, error) {
	data, err := bf.FRP.Fingerprint(ctx.Request)
	if err != nil {
		return false, err
	}
	exists,err := bf.Exists(data, ctx.Spider)
	if err!=nil{
		return false, err
	}
	if !exists{
		err=bf.Add(data, ctx.Spider)
		return false,err
	}
	return true, nil
}

// encode request message data encode use gob
func (d *DistributedCache) encode(ctx *Context) (*bytes.Buffer, error) {

	var network bytes.Buffer // Stand-in for a network connection
	msg := Message{
		Request: *ctx.Request,
		CtxId:   ctx.CtxId,
		Spider:  ctx.Spider,
	}
	enc := gob.NewEncoder(&network)
	err := enc.Encode(&msg)
	if err != nil {
		return nil, err
	}
	return &network, nil
}

// decode message from cache use gob
func (d *DistributedCache) decode(buffer *bytes.Buffer) (*Context, error) {
	msg := Message{}

	decoder := gob.NewDecoder(buffer) //创建解密器
	err := decoder.Decode(&msg)       //解密
	if err != nil {
		return nil, err
	}
	ctx := NewContext(&msg.Request, msg.Spider, WithContextID(msg.CtxId))
	return ctx, nil
}

// enqueue put request to cache
func (d *DistributedCache) enqueue(ctx *Context) error {
	key := fmt.Sprintf("%s:%s", d.keyPrefix,ctx.Spider.GetName())
	buffer, err := d.encode(ctx)
	if err != nil {
		return err
	}
	rs:=d.RDB.LPush(context.TODO(), key, buffer.Bytes())
	return rs.Err()
}

// dequeue get request from cache
func (d *DistributedCache) dequeue(Spider SpiderInterface) (interface{}, error) {
	key := fmt.Sprintf("%s:%s", d.keyPrefix,Spider.GetName())
	rs:=d.RDB.RPop(context.TODO(),key)
	if rs.Err() != nil {
		return nil, rs.Err()
	}
	data, err := rs.Bytes()
	if err!=nil{
		return nil, err
	}
	ctx, err := d.decode(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

// getSize get cache size
func (d *DistributedCache) getSize(Spider SpiderInterface) (int64, error) {
	key := fmt.Sprintf("%s:%s", d.keyPrefix,Spider.GetName())
	rs, err := d.RDB.LLen(context.TODO(),key).Result()
	if err != nil {
		return -1, err
	}
	return rs, nil
}
