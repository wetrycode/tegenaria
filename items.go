package tegenaria

// Item as meta data process interface
type ItemInterface interface {
}
type ItemMeta struct{
	CtxId string
	Item ItemInterface
}


func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxId: ctx.CtxId,
		Item: item,
	}
}
