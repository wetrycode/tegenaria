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
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	bloom "github.com/bits-and-blooms/bloom/v3"
	"github.com/spaolacci/murmur3"
)

// RFPDupeFilterInterface Request Fingerprint duplicates filter interface
type RFPDupeFilterInterface interface {
	// Fingerprint compute request fingerprint
	Fingerprint(request *Request) ([]byte, error)

	// DoDupeFilter do request fingerprint duplicates filter
	DoDupeFilter(request *Request) (bool, error)
}

type RFPDupeFilter struct {
	bloomFilter *bloom.BloomFilter
}

func NewRFPDupeFilter(bloomM uint, bloomK uint) *RFPDupeFilter {
	return &RFPDupeFilter{
		bloomFilter: bloom.New(bloomM, bloomK),
	}
}

// canonicalizeUrl canonical request url before calculate request fingerprint
func (f *RFPDupeFilter) canonicalizetionUrl(request *Request, keepFragment bool) url.URL {
	u, _ := url.ParseRequestURI(request.Url)
	u.RawQuery = u.Query().Encode()
	u.ForceQuery = true
	if !keepFragment {
		u.Fragment = ""
	}
	return *u
}

// encodeHeader encode request header before calculate request fingerprint
func (f *RFPDupeFilter) encodeHeader(request *Request) string {
	h := request.Header
	if h == nil {
		return ""
	}
	var buf bytes.Buffer
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	// Sort by Header key
	sort.Strings(keys)
	for _, k := range keys {
		// Sort by value
		buf.WriteString(fmt.Sprintf("%s:%s;\n", strings.ToUpper(k), strings.ToUpper(h[k])))
	}
	return buf.String()
}

func (f *RFPDupeFilter) Fingerprint(request *Request) ([]byte, error) {
	// get sha128
	sha := murmur3.New128()
	_, err := io.WriteString(sha, request.Method)
	if err != nil {
		return nil, err
	}
	// canonical request url
	u := f.canonicalizetionUrl(request, false)
	_, err = io.WriteString(sha, u.String())
	if err != nil {
		return nil, err
	}
	// get request body
	if request.Body != nil {
		body := request.Body
		sha.Write(body)
	}
	// to handle request header
	if len(request.Header) != 0 {
		_, err := io.WriteString(sha, f.encodeHeader(request))
		if err != nil {
			return nil, err
		}
	}
	res := sha.Sum(nil)
	return res, nil
}

// DoDupeFilter deduplicate request filter by bloom
func (f *RFPDupeFilter) DoDupeFilter(request *Request) (bool, error) {
	// Use bloom filter to do fingerprint deduplication
	data, err := f.Fingerprint(request)
	if err != nil {
		return false, err
	}
	return f.bloomFilter.TestOrAdd(data), nil
}
