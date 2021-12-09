package pipelines

type Pipelines interface{
	Do() error
	Register(name string, pipeline interface{}) error
}
