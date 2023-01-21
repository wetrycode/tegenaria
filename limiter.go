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

package tegenaria

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/ratelimit"
)
// LimitInterface 限速器接口
type LimitInterface interface {
	// checkAndWaitLimiterPass 检查当前并发量
	// 如果并发量达到上限则等待
	checkAndWaitLimiterPass() error
	// setCurrrentSpider 设置当前正在的运行的spider
	setCurrrentSpider(spider string)
}
// leakyBucketLuaScript 漏桶算法lua脚本
const leakyBucketLuaScript string = `-- 最高水位
local safetyLevel = tonumber(ARGV[1])
-- 水流速度
local waterVelocity = tonumber(ARGV[2])
-- 当前时间
local now = tonumber(ARGV[3])
local key = KEYS[1]
-- 最后一次放水时间
local lastOutTime = tonumber(redis.call("hget", key, "lastOutTime"))
-- 当前的水位
local currentLevel = tonumber(redis.call("hget", key, "currentLevel"))
-- 初始化
if lastOutTime == nil then 
   -- 以点当前时间作为最后一次放水时间
   lastOutTime = now
   currentLevel = 0
   redis.call("hmset", key, "currentLevel", currentLevel, "lastOutTime", lastOutTime)
end 

-- 放水时间间隔
local interval = now - lastOutTime
if interval > 0 then
   -- 当前水位-距离上次放水的时间(秒)*水流速度
   local newLevel = currentLevel - interval * waterVelocity
   if newLevel < 0 then 
      newLevel = 0
   end 
   currentLevel = newLevel
   redis.call("hmset", KEYS[1], "currentLevel", newLevel, "lastOutTime", now)
end

-- 若到达最高水位，请求失败
if currentLevel >= safetyLevel then
   return 0
end
-- 若没有到达最高水位，当前水位+1，请求成功
redis.call("hincrby", key, "currentLevel", 1)
redis.call("expire", key, safetyLevel / waterVelocity)
return 1`

// leakyBucketLimiterWithRdb单机redis下的漏桶限速器
type leakyBucketLimiterWithRdb struct {
	// safetyLevel 最高水位
	safetyLevel   int 
	// currentLevel 当前水位
	currentLevel  int 
	// waterVelocity 水流速度/秒
	waterVelocity int 
	// currentSpider 当前正在运行的爬虫名
	currentSpider string
	// rdb redis客户端实例
	rdb           redis.Cmdable // redis 客户端
	// script redis lua脚本
	script        *redis.Script // lua脚本
	// keyFunc 限速器使用的缓存key函数
	keyFunc       GetRDBKey
}
// defaultLimiter 默认的限速器
type defaultLimiter struct {
	limiter ratelimit.Limiter
}

// NewDefaultLimiter 创建一个新的限速器
// limitRate 最大请求速率
func NewDefaultLimiter(limitRate int) *defaultLimiter {
	return &defaultLimiter{
		limiter: ratelimit.New(limitRate, ratelimit.WithoutSlack),
	}
}
// checkAndWaitLimiterPass 检查当前并发量
// 如果并发量达到上限则等待
func (d *defaultLimiter) checkAndWaitLimiterPass() error {
	d.limiter.Take()
	return nil
}
// setCurrrentSpider 设置当前的spider名
func (d *defaultLimiter) setCurrrentSpider(spider string) {

}
// NewLeakyBucketLimiterWithRdb leakyBucketLimiterWithRdb 构造函数
func NewLeakyBucketLimiterWithRdb(safetyLevel int, rdb redis.Cmdable, keyFunc GetRDBKey) *leakyBucketLimiterWithRdb {
	script := readLuaScript()
	return &leakyBucketLimiterWithRdb{
		safetyLevel:   safetyLevel,
		currentLevel:  0,
		waterVelocity: safetyLevel,
		rdb:           rdb,
		script:        script,
		keyFunc:       keyFunc,
	}

}
// tryPassLimiter 尝试通过限速器
func (l *leakyBucketLimiterWithRdb) tryPassLimiter() (bool, error) {
	now := time.Now().Unix()
	key, _ := l.keyFunc()
	key = fmt.Sprintf("%s:%s", key, l.currentSpider)
	pass, err := l.script.Run(context.TODO(), l.rdb, []string{key}, l.safetyLevel, l.waterVelocity, now).Int()
	if err != nil {
		return false, err
	}
	return pass == 1, nil

}

// setCurrrentSpider 设置当前的spider 名称
func (l *leakyBucketLimiterWithRdb) setCurrrentSpider(spider string) {
	l.currentSpider = spider
	key, ttl := l.keyFunc()
	key = fmt.Sprintf("%s:%s", key, l.currentSpider)
	if ttl>0{
		l.rdb.Expire(context.TODO(), key, ttl)
	}

}

// checkAndWaitLimiterPass 检查当前并发量
// 如果并发量达到上限则等待
func (l *leakyBucketLimiterWithRdb) checkAndWaitLimiterPass() error {
	for {
		pass, err := l.tryPassLimiter()
		if err != nil {
			return err
		}
		if pass {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
func readLuaScript() *redis.Script {
	return redis.NewScript(leakyBucketLuaScript)
}
