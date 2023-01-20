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

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/google/uuid"
)

func GetUUID() string {
	u4 := uuid.New()
	uuid := u4.String()
	return uuid

}
// GoFunc 协程函数
type GoFunc func() error

// AddGo 向指定的wg添加协程函数
// 使用chan error 传递异常
func AddGo(wg *sync.WaitGroup, funcs ...GoFunc) <-chan error {
	ch := make(chan error, len(funcs))
	for _, readyFunc := range funcs {
		_func := readyFunc
		wg.Add(1)
		go func() {
			defer func() {
				if p:=recover();p!=nil{
					ch<-fmt.Errorf("call go funcs paninc %s", p)
				}
				wg.Done()
			}()
			ch <- _func()

		}()
	}
	return ch
}
// GetFunctionName 提取解析函数名
func GetFunctionName(fn Parser) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	nodes := strings.Split(name, ".")
	return strings.ReplaceAll(nodes[len(nodes)-1], "-fm", "")

}
// GetParserByName 通过函数名从spider实例中获取解析函数
func GetParserByName(spider SpiderInterface, name string) Parser {
	return func(resp *Context, req chan<- *Context) error {
		args := make([]reflect.Value, 2)
		args[0] = reflect.ValueOf(resp)
		args[1] = reflect.ValueOf(req)
		rets := reflect.ValueOf(spider).MethodByName(name).Call(args)
		if rets[0].IsNil() {
			return nil
		}
		return rets[0].Interface().(error)
	}
}
// GetAllParserMethod 获取spider实例所有的解析函数
func GetAllParserMethod(spider SpiderInterface) map[string]Parser {
	val := reflect.ValueOf(spider)
	sType := reflect.TypeOf(spider)
	parsers := make(map[string]Parser)
	for i := 0; i < val.NumMethod(); i++ {
		f := val.Method(i)
		switch f.Interface().(type) {
		case func(resp *Context, req chan<- *Context) error:
			name := sType.Method(i).Name
			parsers[fmt.Sprintf("%s.%s", spider.GetName(), name)] = GetParserByName(spider, name)
		default:
			// fmt.Printf("method is %s, type is %v, kind is %s.\n", s.Method(i).Name, f.Type(), s.Method(i).Type.Kind())

		}
	}
	return parsers
}
// OptimalNumOfHashFunctions计算最优的布隆过滤器哈希函数个数
func OptimalNumOfHashFunctions(n int64, m int64) int64 {
	// (m / n) * log(2), but avoid truncation due to division!
	// return math.max(1, (int) Math.round((double) m / n * Math.log(2)));
	return int64(math.Max(1, math.Round(float64(m)/float64(n)*math.Log(2))))
}

// OptimalNumOfBits 计算位数组长度
func OptimalNumOfBits(n int64, p float64) int64 {
	return (int64)(-float64(n) * math.Log(p) / (math.Log(2) * math.Log(2)))
}
// Map2String 将map转为string
func Map2String(m interface{}) string {
	dataType, _ := json.Marshal(m)
	dataString := string(dataType)
	return dataString

}
// GetMachineIp 获取本机ip
func GetMachineIp() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""

}
