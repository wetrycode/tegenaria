#### 爬虫启动前置动作

tegenaria允许用户实现```ComponentInterface```接口```SpiderBeforeStart```函数，用于控制爬虫启动前的行为，例如分布式组件的```SpiderBeforeStart```的实现,用于在从节点启动前检查主节点的状态

```go
func (d *DistributedComponents) SpiderBeforeStart(engine *tegenaria.CrawlEngine, spider tegenaria.SpiderInterface) error {
	if !d.worker.IsMaster() {
		// 分布式模式下的启动流程
		// 如果节点角色不是master则检查是否有主节点在线
		// 若没有主节点在线则不启动爬虫
		start := time.Now()
		for {
			time.Sleep(3 * time.Second)
			live, err := d.worker.CheckMasterLive()
			logger.Infof("有主节点存在:%v", live)
			if (!live) || (err != nil) {
				if err != nil {
					panic(fmt.Sprintf("check master nodes status error %s", err.Error()))
				}
				logger.Warnf("正在等待主节点上线")
				// 超过30s直接panic
				if time.Since(start) > 30*time.Second {
					panic(tegenaria.ErrNoMaterNodeLive)
				}
				continue
			}
			break
		}
		// 从节点不需要启动StartRequest
		return errors.New("No need to start this node 'StartRequest'")
	}
	return nil
}
```