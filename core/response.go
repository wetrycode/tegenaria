package core

type Response struct {
	Text   string
	Status int
	Body   []byte
	Header map[string][]byte
	Req    *Request
}
