#### 分布式限速器
tegenaria 提供了基于redis实现的漏斗算法限速器用于控制并发，该过滤器实现了`LimitInterface`[接口](limit.md)   
定义如下:
```go
type LeakyBucketLimiterWithRdb struct {
	// safetyLevel 最高水位
	safetyLevel int
	// currentLevel 当前水位
	currentLevel int
	// waterVelocity 水流速度/秒
	waterVelocity int
	// currentSpider 当前正在运行的爬虫名
	currentSpider string
	// rdb redis客户端实例
	rdb redis.Cmdable // redis 客户端
	// script redis lua脚本
	script *redis.Script // lua脚本
	// keyFunc 限速器使用的缓存key函数
	keyFunc GetRDBKey
}

// NewLeakyBucketLimiterWithRdb LeakyBucketLimiterWithRdb 构造函数
func NewLeakyBucketLimiterWithRdb(safetyLevel int, rdb redis.Cmdable, keyFunc GetRDBKey) *LeakyBucketLimiterWithRdb {
	script := readLuaScript()
	return &LeakyBucketLimiterWithRdb{
		safetyLevel:   safetyLevel,
		currentLevel:  0,
		waterVelocity: safetyLevel,
		rdb:           rdb,
		script:        script,
		keyFunc:       keyFunc,
	}

}
```

#### 说明

##### 关键参数

- `safetyLevel`安全水位，即单位时间内的最大请求量,默认值16

- `currentLevel`当前水位，即当前单位时间内的请求量  

- `waterVelocity` 水流速度，即请求速率req_num/s

- `script` lua脚本

- 注意，`safetyLevel`和 `waterVelocity`在数值上相等

```lua
-- 最高水位
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
return 1
```
##### 如何使用?

在构建分布式组件`NewDistributedComponents`,通过`DistributedWorkerConfig`传入配置参数`LimiterRate`即可
