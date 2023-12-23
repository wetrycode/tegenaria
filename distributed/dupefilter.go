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
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spaolacci/murmur3"

	"github.com/wetrycode/tegenaria"
)

type DistributedDupefilter struct {
	// rdb redis客户端支持redis单机实例和redis cluster集群模式
	rdb redis.Cmdable
	// getBloomFilterKey 布隆过滤器对应的生成key的函数，允许用户自定义
	getBloomFilterKey GetRDBKey
	// bloomP 布隆过滤器的容错率
	bloomP float64
	// bloomN 数据规模，比如1024 * 1024
	bloomN int
	// bloomM bitset 大小
	bloomM int
	// bloomK hash函数个数
	bloomK int
	// currentSpider 当前的spider
	currentSpider string

	// dupeFilter 去重组件
	dupeFilter *tegenaria.DefaultRFPDupeFilter
}

// getBloomFilterKey 自定义的布隆过滤器key生成函数
func getBloomFilterKey() (string, time.Duration) {
	return "tegenaria:v1:bf", 0 * time.Second
}

type DistributeDupefilterOptions func(w *DistributedDupefilter)

func NewDistributedDupefilter(bloomN int, bloomP float64, rdb redis.Cmdable, bfKeyFunc GetRDBKey) *DistributedDupefilter {
	// 获取最优bit 数组的大小
	m := tegenaria.OptimalNumOfBits(bloomN, bloomP)
	// 获取最优的hash函数个数
	k := tegenaria.OptimalNumOfHashFunctions(bloomN, m)
	return &DistributedDupefilter{
		rdb:               rdb,
		bloomP:            bloomP,
		bloomK:            k,
		bloomN:            bloomN,
		bloomM:            m,
		getBloomFilterKey: bfKeyFunc,
	}
}

// Fingerprint request指纹计算
func (d *DistributedDupefilter) Fingerprint(ctx *tegenaria.Context) ([]byte, error) {
	fp, err := d.dupeFilter.Fingerprint(ctx)
	if err != nil {
		return nil, err
	}
	return fp, nil
}

// DoDupeFilter request去重
func (d *DistributedDupefilter) DoDupeFilter(ctx *tegenaria.Context) (bool, error) {
	fp, err := d.Fingerprint(ctx)
	if err != nil {
		return false, err
	}
	return d.TestOrAdd(fp)
}

// TestOrAdd 如果指纹已经存在则返回True,否则为False
// 指纹不存在的情况下会将指纹添加到缓存
func (d *DistributedDupefilter) TestOrAdd(fingerprint []byte) (bool, error) {
	isExists, err := d.isExists(fingerprint)
	if err != nil {
		return false, err
	}
	if isExists {
		return true, nil
	}
	err = d.Add(fingerprint)
	return false, err
}

func (d *DistributedDupefilter) bfKey() (string, time.Duration) {
	key, ttl := d.getBloomFilterKey()
	return fmt.Sprintf("%s:%s", key, d.currentSpider), ttl
}

// baseHashes 生成hash值
func (d *DistributedDupefilter) baseHashes(data []byte) [4]uint64 {
	a1 := []byte{1} // to grab another bit of data
	hasher := murmur3.New128()
	hasher.Write(data) // #nosec
	v1, v2 := hasher.Sum128()
	hasher.Write(a1) // #nosec
	v3, v4 := hasher.Sum128()
	return [4]uint64{
		v1, v2, v3, v4,
	}
}

// location 获取第i个的hash值
func (d *DistributedDupefilter) location(h [4]uint64, i int) uint64 {
	ii := uint64(i)
	return h[ii%2] + ii*h[2+(((ii+(ii%2))%4)/2)]
}

// getOffset 计算偏移量
func (d *DistributedDupefilter) getOffset(hash [4]uint64, index int) uint {
	return uint(d.location(hash, index) % uint64(d.bloomM))
}

// Add 添加指纹到布隆过滤器
func (d *DistributedDupefilter) Add(fingerprint []byte) error {
	h := d.baseHashes(fingerprint)
	pipe := d.rdb.Pipeline()
	key, _ := d.bfKey()
	for i := 0; i < d.bloomK; i++ {
		value := d.getOffset(h, i)
		pipe.SetBit(context.TODO(), key, int64(value), 1)
	}
	_, err := pipe.Exec(context.TODO())
	if err != nil {
		return err
	}
	return nil
}

// isExists 判断指纹是否存在
func (d *DistributedDupefilter) isExists(fingerprint []byte) (bool, error) {
	h := d.baseHashes(fingerprint)
	pipe := d.rdb.Pipeline()
	result := []*redis.IntCmd{}
	key, _ := d.bfKey()
	for i := 0; i < d.bloomK; i++ {
		value := d.getOffset(h, i)
		result = append(result, pipe.GetBit(context.TODO(), key, int64(value)))
	}
	_, err := pipe.Exec(context.TODO())
	if err != nil {
		return false, err
	}
	for _, val := range result {
		r, err := val.Result()
		if err != nil {
			return false, err
		}
		if r == 0 {
			return false, nil
		}
	}
	return true, nil
}

// SetCurrentSpider 设置当前的spider
func (d *DistributedDupefilter) SetCurrentSpider(spider tegenaria.SpiderInterface) {
	d.currentSpider = spider.GetName()
	funcs := []GetRDBKey{}
	funcs = append(funcs, d.bfKey)
	for _, f := range funcs {
		key, ttl := f()
		if ttl > 0 {
			d.rdb.Expire(context.TODO(), key, ttl)
		}

	}
}
