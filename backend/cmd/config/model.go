package config

// Model 是 config.yaml 的完整结构。字段的 comment 标签会写进生成的示例配置，作为键上方的注释。
type Model struct {
	AppName  string        `yaml:"app-name"`
	Debug    bool          `yaml:"debug" comment:"true 时从 server.web-path 目录实时读取前端文件，false 时使用嵌入二进制的前端"`
	Server   ServerModel   `yaml:"server"`
	Database DatabaseModel `yaml:"database"`
	Storage  StorageModel  `yaml:"storage"`
	Log      LogModel      `yaml:"log"`
}

// StorageModel 是附件文件的存储配置。
type StorageModel struct {
	Path string `yaml:"path" comment:"附件文件、缩略图和未完成上传的存放目录；为空时使用配置文件所在目录下的 files"`
}

// ServerModel 是 HTTP 服务的监听、证书和反向代理配置。
type ServerModel struct {
	HttpIp      string `yaml:"http-ip"`
	HttpPort    int    `yaml:"http-port"`
	WebPath     string `yaml:"web-path"`
	EnableHttps bool   `yaml:"enable-https"`
	CrtFile     string `yaml:"crt-file"`
	KeyFile     string `yaml:"key-file"`
	// TrustedProxies 是受信任的反向代理地址。客户端 IP（登录限流按它计数）只从这些代理转发的 X-Forwarded-For 中读取，
	// 为空时直接使用连接的来源地址。
	TrustedProxies []string `yaml:"trusted-proxies" comment:"受信任的反向代理地址（CIDR 或单个 IP），部署在反向代理后时填写代理的地址；为空时直接使用连接的来源地址作为客户端 IP"`
}

// 数据库类型，对应 DatabaseModel.Type。
const (
	DatabaseSQLite   = "sqlite"
	DatabasePostgres = "postgres"
)

// DatabaseModel 是数据库连接配置。Type 为 sqlite 时只使用 File，为 postgres 时使用其余连接参数。
type DatabaseModel struct {
	Type     string `yaml:"type" comment:"数据库类型：sqlite 或 postgres"`
	File     string `yaml:"file" comment:"SQLite 数据库文件路径，type 为 sqlite 时生效"`
	Host     string `yaml:"host" comment:"PostgreSQL 连接参数，type 为 postgres 时生效；数据库不存在时 migrate 会创建"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// LogModel 是日志级别、格式、输出文件和调试输出配置。
type LogModel struct {
	EnableFile bool     `yaml:"enable-file"`
	File       string   `yaml:"file"`
	Mask       []string `yaml:"mask" comment:"unknown, debug, trace, info, warning, error, fatal"`
	Flag       []string `yaml:"flag" comment:"date, time, long_file, short_file, func, prefix, suffix"`
	Debug      []string `yaml:"debug" comment:"database"`
}
