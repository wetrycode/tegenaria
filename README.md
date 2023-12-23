# Tegenaria crawl framework

[![Go Report Card](https://goreportcard.com/badge/github.com/wetrycode/tegenaria)](https://goreportcard.com/report/github.com/wetrycode/tegenaria)
[![codecov](https://codecov.io/gh/wetrycode/tegenaria/branch/master/graph/badge.svg?token=XMW3K1JYPB)](https://codecov.io/gh/wetrycode/tegenaria)
[![go workflow](https://github.com/wetrycode/tegenaria/actions/workflows/go.yml/badge.svg)](https://github.com/wetrycode/tegenaria/actions/workflows/go.yml/badge.svg)
[![CodeQL](https://github.com/wetrycode/tegenaria/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/wetrycode/tegenaria/actions/workflows/codeql-analysis.yml)  
tegenaria是一个基于golang开发的快速、高效率的网络爬虫框架

# 特性

- 支持分布式  

- 支持自定义分布式组件,包括去重、request缓存队列和基本统计指标的分布式运行    

- 支持自定义的事件监控

- 支持命令行控制  

- 支持gRPC和web api远程控制  

- ~~支持定时轮询启动爬虫~~
  
  ## 安装
1. go 版本要求>=1.19 

```bash
go get -u github.com/wetrycode/tegenaria@latest
```

2. 在您的项目中导入

```go
import "github.com/wetrycode/tegenaria"
```

# 快速开始

 - 查看实例demo [example](example) 

 - 使用脚手架工具[teg-cli]("https://github.com/wetrycode/teg-cli")创建项目

# 文档

- [入门](https://wetrycode.github.io/tegenaria/#/quickstart)

# TODO

- ~~管理WEB API~~


# Contribution

Feel free to PR and raise issues.  
Send me an email directly, vforfreedom96@gmail.com  

## License

[MIT](LICENSE) © wetrycode  
