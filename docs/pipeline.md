#### Pipeline

item pipelines 用于对 item 进行处理，例如持久化存储到数据库、数据去重等操作，所有的 pipeline 都应实现`PipelinesInterface`接口其中包含的两个函数如下:

- `GetPriority() int`给引擎提供当前 pipeline 的优先级，请注意数字越低优先级越高越早调用

- `ProcessItem(spider SpiderInterface, item *ItemMeta) error` 处理 item 的核心逻辑

#### 定义 item pipeline

```go
// QuotesbotItemPipeline tegenaria.PipelinesInterface 接口示例
// 用于item处理的pipeline
type QuotesbotItemPipeline struct {
    Priority int
}
// ProcessItem item处理函数
func (p *QuotesbotItemPipeline) ProcessItem(spider tegenaria.SpiderInterface, item *tegenaria.ItemMeta) error {
    i:=item.Item.(*QuotesbotItem)
    exampleLog.Infof("%s 抓取到数据:%s",item.CtxID, i.Text)
    return nil

}

// GetPriority 获取该pipeline的优先级
func (p *QuotesbotItemPipeline) GetPriority() int {
    return p.Priority
}
```

#### 响应引擎注册 piplines

```go
pipe := QuotesbotItemPipeline{Priority: 1}
Engine.RegisterPipelines(pipe)
```