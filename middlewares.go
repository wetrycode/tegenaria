// Copyright 2022 vforfreedom96@gmail.com
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

// MiddlewaresInterface Download middleware interface for Request and 
// Response processing, the smaller the priority number the higher the priority
type MiddlewaresInterface interface {
	// GetPriority get middlerware priority
	GetPriority() int

	// ProcessRequest process request before request to do download
	ProcessRequest(ctx *Context) error

	// ProcessResponse process response before response to parse
	ProcessResponse(ctx *Context, req chan<- *Context) error

	// GetName get middlerware name
	GetName()string
}
type ProcessResponse func(ctx *Context) error
type MiddlewaresBase struct {
	Priority int
}

type Middlewares []MiddlewaresInterface

func (p Middlewares) Len() int           { return len(p) }
func (p Middlewares) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p Middlewares) Less(i, j int) bool { return p[i].GetPriority() < p[j].GetPriority() }


