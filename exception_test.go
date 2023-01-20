package tegenaria

import (
	"errors"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestErrorWithExtras(t *testing.T) {
convey.Convey("test error with extra",t,func(){
	extras:=map[string]interface{}{
	}
	extras["errExt"] = "ext"
	spider := &TestSpider{
		NewBaseSpider("testspider", []string{"https://www.baidu.com"}),
	}
	request := NewRequest("http://www.example.com", GET, spider.Parser)
	ctx := NewContext(request, spider)
	err:=NewError(ctx,errors.New("ctx error"),ErrorWithExtras(extras))
	convey.So(err.Extras, convey.ShouldContainKey,"errExt")
})

}
