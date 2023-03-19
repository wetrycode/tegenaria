// MIT License

// Copyright (c) 2023 wetrycode

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package tegenaria

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sourcegraph/conc"
)

func GetUUID() string {
	u4 := uuid.New()
	uuid := u4.String()
	return uuid

}

// GoFunc 协程函数
type GoFunc func() error

// GoRunner 执行协程任务
func GoRunner(wg *conc.WaitGroup, funcs ...GoFunc) <-chan error {
	ch := make(chan error, len(funcs))
	for _, readyFunc := range funcs {
		_func := readyFunc
		wg.Go(func() {
			ch <- _func()
		})
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
func GetParserByName(spider SpiderInterface, name string) reflect.Value {
	return reflect.ValueOf(spider).MethodByName(name)
}

// OptimalNumOfHashFunctions 计算最优的布隆过滤器哈希函数个数
func OptimalNumOfHashFunctions(n int, m int) int {
	// (m / n) * log(2), but avoid truncation due to division!
	// return math.max(1, (int) Math.round((double) m / n * Math.log(2)));
	return int(math.Max(1, math.Round(float64(m)/float64(n)*math.Log(2))))
}

// OptimalNumOfBits 计算位数组长度
func OptimalNumOfBits(n int, p float64) int {
	return (int)(-float64(n) * math.Log(p) / (math.Log(2) * math.Log(2)))
}

// Map2String 将map转为string
func Map2String(m interface{}) string {
	dataType, _ := json.Marshal(m)
	dataString := string(dataType)
	return dataString

}

// GetMachineIP 获取本机ip
func GetMachineIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", err

}

// GetEngineID 获取当前进程的引擎实例id
func GetEngineID() string {
	return engineID
}
func MD5(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
func Interface2Uint(value interface{}) uint {
	switch interfaceType := value.(type) {
	case int:
		return uint(value.(int))
	case float64:
		return uint(int(value.(float64)))
	case string:
		ret, _ := strconv.Atoi(value.(string))
		return uint(ret)
	case uint:
		return value.(uint)
	case int64:
		return uint(value.(int64))
	case int32:
		return uint(value.(int32))
	case uint64:
		return uint(value.(uint64))
	case nil:
		return 0
	default:
		panic(fmt.Sprintf("unknown type %s", interfaceType))
	}
}
