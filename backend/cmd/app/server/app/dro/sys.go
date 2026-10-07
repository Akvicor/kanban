package dro

// SysHealth 表示服务及各运行依赖的就绪状态。
type SysHealth struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}
