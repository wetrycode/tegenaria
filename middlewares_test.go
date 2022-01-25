package tegenaria

import (
	"errors"
	"fmt"
	"strconv"
)

type TestDownloadMiddler struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler) ProcessRequest(ctx *Context) error {
	header := fmt.Sprintf("priority-%d", m.Priority)
	ctx.Request.Header[header] = strconv.Itoa(m.Priority)
	return nil
}

func (m TestDownloadMiddler) ProcessResponse(ctx *Context) error {
	return nil

}
func (m TestDownloadMiddler) GetName() string {
	return m.Name
}

type TestDownloadMiddler2 struct {
	Priority int
	Name     string
}

func (m TestDownloadMiddler2) GetPriority() int {
	return m.Priority
}
func (m TestDownloadMiddler2) ProcessRequest(ctx *Context) error {
	return errors.New("process request fail")
}

func (m TestDownloadMiddler2) ProcessResponse(ctx *Context) error {
	return errors.New("process response fail")

}
func (m TestDownloadMiddler2) GetName() string {
	return m.Name
}
