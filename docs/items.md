#### ItemMata
`ItemMata`在解析函数生成,存储解析出来的数据字段并在pipeline中被处理,其定义如下:
```go
// ItemMeta item元数据结构
type ItemMeta struct {
	// CtxID 对应的context id
	CtxID string
	// Item item对象
	Item ItemInterface
}

// NewItem 构建新的ItemMeta对象
func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxID: ctx.CtxID,
		Item:  item,
	}
}
```
#### 说明 

- 用户需要定义好数据模型`Item`,例如：
```go
type QuotesbotItem struct {
	Text   string
	Author string
	Tags   string
}
```

- 在构造`ItemMata`对象时传入`Context`和`Item对象`  

- `ItemMata`对象通过`Context`对象绑定的channel提交到引擎  

#### 示例  

```go
	var quoteItem = QuotesbotItem{
		Text:   qText,
		Author: author,
		Tags:   strings.Join(tags, ","),
	}
	// 构建item发送到指定的channel
	itemCtx := tegenaria.NewItem(ctx, &quoteItem)
	ctx.Items <- itemCtx
```