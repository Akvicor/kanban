package config

import (
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Akvicor/glog"
	"github.com/Akvicor/util"
	"gopkg.in/yaml.v3"
)

// Global 是当前进程加载的配置，Load 成功后才可使用。
var Global *Model

// FileData 是 Global 对应的配置文件原文。
var FileData []byte

// Load 加载并校验 YAML 配置，同时应用日志设置。
func Load(p string) error {
	if util.FileStat(p).NotFile() {
		return fmt.Errorf("配置文件不存在: %s", p)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}
	model := new(Model)
	if err = yaml.Unmarshal(data, model); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	if err = validate(model); err != nil {
		return err
	}
	// 没有配置存储目录的旧配置文件，使用配置文件旁边的 files 目录。
	if model.Storage.Path == "" {
		model.Storage.Path = filepath.Join(filepath.Dir(p), "files")
	}

	// 设置日志级别掩码。
	mask := uint32(0)
	for _, v := range model.Log.Mask {
		switch v {
		case "unknown":
			mask |= glog.MaskUNKNOWN
		case "debug":
			mask |= glog.MaskDEBUG
		case "trace":
			mask |= glog.MaskTRACE
		case "info":
			mask |= glog.MaskINFO
		case "warning":
			mask |= glog.MaskWARNING
		case "error":
			mask |= glog.MaskERROR
		case "fatal":
			mask |= glog.MaskFATAL
		}
	}
	glog.SetMask(mask)

	// 设置日志格式。
	flg := uint32(0)
	for _, v := range model.Log.Flag {
		switch v {
		case "date":
			flg |= glog.FlagDate
		case "time":
			flg |= glog.FlagTime
		case "long_file":
			flg |= glog.FlagLongFile
		case "short_file":
			flg |= glog.FlagShortFile
		case "func":
			flg |= glog.FlagFunc
		case "prefix":
			flg |= glog.FlagPrefix
		case "suffix":
			flg |= glog.FlagSuffix
		}
	}
	glog.SetFlag(flg)

	// 设置日志文件。
	if model.Log.EnableFile {
		logDir := path.Dir(model.Log.File)
		if util.FileStat(logDir).NotExist() {
			if err = os.MkdirAll(logDir, os.ModePerm); err != nil {
				return fmt.Errorf("创建日志目录失败: %w", err)
			}
		}
		if err = glog.SetLogFile(model.Log.File); err != nil {
			return fmt.Errorf("设置日志文件失败: %w", err)
		}
	}

	FileData = data
	Global = model
	return nil
}

// validate 校验启动必需的配置项。
func validate(model *Model) error {
	if model.Server.HttpPort < 1 || model.Server.HttpPort > 65535 {
		return fmt.Errorf("HTTP 端口必须在 1 到 65535 之间")
	}
	if _, err := model.Server.TrustedProxyNets(); err != nil {
		return err
	}
	database := model.Database
	switch database.Type {
	case DatabaseSQLite:
		if database.File == "" {
			return fmt.Errorf("SQLite 数据库文件路径不能为空")
		}
	case DatabasePostgres:
		if database.Host == "" || database.Database == "" || database.Username == "" {
			return fmt.Errorf("PostgreSQL 的 host、database、username 不能为空")
		}
		if database.Port < 1 || database.Port > 65535 {
			return fmt.Errorf("PostgreSQL 端口必须在 1 到 65535 之间")
		}
	default:
		return fmt.Errorf("不支持的数据库类型: %q，可选 sqlite 或 postgres", database.Type)
	}
	return nil
}

// TrustedProxyNets 把 TrustedProxies 解析为网段。每项可以是 CIDR，也可以是单个 IP（按 /32 或 /128 处理）。
func (s ServerModel) TrustedProxyNets() ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(s.TrustedProxies))
	for _, item := range s.TrustedProxies {
		item = strings.TrimSpace(item)
		if _, network, err := net.ParseCIDR(item); err == nil {
			nets = append(nets, network)
			continue
		}
		ip := net.ParseIP(item)
		if ip == nil {
			return nil, fmt.Errorf("server.trusted-proxies 中的 %q 不是有效的 IP 或 CIDR", item)
		}
		bits := 8 * net.IPv6len
		if ip4 := ip.To4(); ip4 != nil {
			ip, bits = ip4, 8*net.IPv4len
		}
		nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return nets, nil
}

// Example 返回以 basePath 为数据目录的示例配置，默认使用 SQLite。
func Example(basePath string) *Model {
	return &Model{
		AppName: "Kanban",
		Debug:   false,
		Server: ServerModel{
			HttpIp:         "0.0.0.0",
			HttpPort:       3000,
			WebPath:        "build",
			EnableHttps:    false,
			CrtFile:        path.Join(basePath, "cert/example.com.crt"),
			KeyFile:        path.Join(basePath, "cert/example.com.key"),
			TrustedProxies: []string{},
		},
		Database: DatabaseModel{
			Type:     DatabaseSQLite,
			File:     path.Join(basePath, "kanban.db"),
			Host:     "postgres",
			Port:     5432,
			Database: "kanban",
			Username: "kanban",
			Password: "",
		},
		Storage: StorageModel{
			Path: path.Join(basePath, "files"),
		},
		Log: LogModel{
			EnableFile: false,
			File:       path.Join(basePath, "kanban.log"),
			Mask:       []string{"unknown", "debug", "trace", "info", "warning", "error", "fatal"},
			Flag:       []string{"date", "time", "short_file", "prefix", "suffix"},
			Debug:      []string{},
		},
	}
}

// GenerateExample 向标准输出写入带注释的 YAML 示例配置。
func GenerateExample(basePath string) error {
	data, err := marshalWithComments(Example(basePath))
	if err != nil {
		return fmt.Errorf("生成示例配置失败: %w", err)
	}
	if _, err = os.Stdout.Write(data); err != nil {
		return fmt.Errorf("输出示例配置失败: %w", err)
	}
	return nil
}
