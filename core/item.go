// package core

/*
 The go-scrapy core modules package
 */ 
package core

// Item as meta data process interface
type ItemProcesser interface {
	ProcessItem() map[interface{}] interface{}
}