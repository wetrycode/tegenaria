package middlewares

type Middlewares interface{
	Do() error
	Register(name string, middleware interface{}) error
}