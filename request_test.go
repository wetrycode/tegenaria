package tegenaria

import (
	"errors"
	"testing"

	"github.com/go-kiss/monkey"
	jsoniter "github.com/json-iterator/go"
)

func TestRequestBodyReadError(t *testing.T) {
	defer func(){
		if p:=recover();p!=nil{

		}else{
			t.Errorf("Reading request body except error creat request with read body")
		}
	}()
	monkey.Patch(jsoniter.Marshal, func(v interface{}) ([]byte, error) {
		return nil, errors.New("creat request with read body")
	})
	body := make(map[string]interface{})
	body["test"] = "test"
	NewRequest("http://www.example.com", GET, testParser, RequestWithRequestBody(body))
}
