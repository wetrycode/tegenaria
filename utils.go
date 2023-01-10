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
	"sync"

	"github.com/google/uuid"
)

func GetUUID() string {
	u4 := uuid.New()
	uuid := u4.String()
	return uuid

}
type GoFunc func()

func GoSyncWait(wg *sync.WaitGroup, funcs ...GoFunc){
	for _, readyFunc := range funcs{
		_func := readyFunc
		wg.Add(1)
		go func(){
			defer wg.Done()
			_func()
		}()
	}
}
