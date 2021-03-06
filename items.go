// Copyright 2022 geebytes
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tegenaria

import "context"

// Item as meta data process interface
type ItemInterface interface {
}
type ItemMeta struct {
	CtxId string
	Item  ItemInterface
	Ctx   context.Context
}

func NewItem(ctx *Context, item ItemInterface) *ItemMeta {
	return &ItemMeta{
		CtxId: ctx.CtxId,
		Item:  item,
		Ctx:   ctx.parent,
	}
}
