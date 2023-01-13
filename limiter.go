package tegenaria

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/ratelimit"
)

type LimitInterface interface {
	// tryPassLimiter() (bool, error)
	checkAndWaitLimiterPass() error
}

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

type leakyBucketLimiterWithRdb struct {
	safetyLevel   int            // 最高水位
	currentLevel  int            // 当前水位
	waterVelocity int            // 水流速度/秒
	rdb           redis.Cmdable // redis 客户端
	script        *redis.Script  // lua脚本
	key string
}
type defaultLimiter struct{
	limiter ratelimit.Limiter
}
func NewDefaultLimiter(limitRate int) *defaultLimiter{
	return &defaultLimiter{
		limiter: ratelimit.New(limitRate,ratelimit.WithoutSlack),
	}
}
func (d *defaultLimiter)checkAndWaitLimiterPass() error{
	d.limiter.Take()
	return nil
}

func NewLeakyBucketLimiterWithRdb(safetyLevel int, rdb redis.Cmdable, key string) *leakyBucketLimiterWithRdb {
	script := readLuaScript()
	return &leakyBucketLimiterWithRdb{
		safetyLevel:   safetyLevel,
		currentLevel:  0,
		waterVelocity: safetyLevel,
		rdb:           rdb,
		script:        script,
		key: key,
	}

}
func (l *leakyBucketLimiterWithRdb) tryPassLimiter() (bool, error) {
	now := time.Now().Unix()
	pass, err := l.script.Run(context.TODO(), l.rdb, []string{l.key}, l.safetyLevel, l.waterVelocity, now).Int()
	if err != nil {
		return false, err
	}
	return pass==1, nil

}
func (l *leakyBucketLimiterWithRdb) checkAndWaitLimiterPass()error{
	for {
		pass, err:= l.tryPassLimiter()
		if err!=nil{
			return err
		}
		if pass{
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
}
func readLuaScript() *redis.Script {
	return redis.NewScript(leakyBucketLuaScript)
}
