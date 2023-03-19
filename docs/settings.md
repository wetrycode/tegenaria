#### 配置管理  

tegenaria的配置项管理基于[viper](github.com/spf13/viper)实现：
```go
type Settings interface {
	// GetValue 获取指定的参数值
	GetValue(key string) (interface{}, error)
}

type Configuration struct {
	// Log *Logger `ymal:"log"`
	*viper.Viper
}

```
- tegenaria的配置管理器完全继承于```viper```,因此配置的读写接口可以参照[viper](github.com/spf13/viper)

- 需要注意的是配置文件的加载顺序是```tegenaria```->```your project```,  
先加载```tegenaria```的```settings.yaml```再加载用户项目根目录下的```settings.yaml```,  
因此用户项目配置项会覆盖```tegenaria```的配置项  