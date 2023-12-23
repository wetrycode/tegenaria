#### 限速器

限速器主要是用于控制请求的并发，避免高并发导致请求被封禁

- 接口定义

```go
// LimitInterface 限速器接口
type LimitInterface interface {
	// checkAndWaitLimiterPass 检查当前并发量
	// 如果并发量达到上限则等待
	CheckAndWaitLimiterPass() error
	// setCurrrentSpider 设置当前正在的运行的spider
	SetCurrentSpider(spider SpiderInterface)
}
```

#### 默认限速器

 tegenaria提供了基于[uber实现的漏斗桶限速器](go.uber.org/ratelimit)
 - 具体定义如下:  
 ```go
 // defaultLimiter 默认的限速器
type DefaultLimiter struct {
	limiter ratelimit.Limiter
	spider  SpiderInterface
}
// NewDefaultLimiter 创建一个新的限速器
// limitRate 最大请求速率
func NewDefaultLimiter(limitRate int) *DefaultLimiter {
	return &DefaultLimiter{
		limiter: ratelimit.New(limitRate, ratelimit.WithoutSlack),
	}
}
 ```
 - 在构造该限速器时需要指定速率,当前给定的默认值是16/s
- 若要修改参数在构造```DefaultComponents```是传入参数```DefaultComponentsWithDefaultLimiter(NewDefaultLimiter(32))```即可  
