package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig 把内容写入临时目录中的 config.yaml 并返回路径。
func writeConfig(t *testing.T, content []byte) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return configPath
}

// TestExampleLoads 验证生成的示例配置带注释，并且可以被 Load 原样加载。
func TestExampleLoads(t *testing.T) {
	dir := t.TempDir()
	data, err := marshalWithComments(Example(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# 数据库类型：sqlite 或 postgres\n  type: sqlite") {
		t.Fatalf("示例配置缺少字段注释:\n%s", data)
	}

	if err = Load(writeConfig(t, data)); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if Global.Database.Type != DatabaseSQLite || Global.Database.File != filepath.Join(dir, "kanban.db") {
		t.Fatalf("Global.Database = %+v", Global.Database)
	}
	if Global.Storage.Path != filepath.Join(dir, "files") {
		t.Fatalf("Global.Storage = %+v", Global.Storage)
	}
	if string(FileData) != string(data) {
		t.Fatal("FileData 未保留已加载的配置内容")
	}
}

// TestPostgresExampleLoads 验证仓库中的 PostgreSQL 示例配置可以被加载。
func TestPostgresExampleLoads(t *testing.T) {
	if err := Load("../../../config.postgres.yaml.example"); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if Global.Database.Type != DatabasePostgres {
		t.Fatalf("Global.Database.Type = %q", Global.Database.Type)
	}
}

// TestStorageDefault 验证没有存储配置的旧配置文件使用配置文件旁边的 files 目录。
func TestStorageDefault(t *testing.T) {
	configPath := writeConfig(t, []byte("server:\n  http-port: 3000\ndatabase:\n  type: sqlite\n  file: kanban.db\n"))
	if err := Load(configPath); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(filepath.Dir(configPath), "files"); Global.Storage.Path != want {
		t.Fatalf("Global.Storage.Path = %q, want %q", Global.Storage.Path, want)
	}
}

// TestTrustedProxyNets 验证受信任代理支持 CIDR 和单个 IP，非法项使加载失败。
func TestTrustedProxyNets(t *testing.T) {
	server := ServerModel{TrustedProxies: []string{"10.0.0.0/8", "192.168.1.5", "::1", "fd00::/8"}}
	nets, err := server.TrustedProxyNets()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/8", "192.168.1.5/32", "::1/128", "fd00::/8"}
	if len(nets) != len(want) {
		t.Fatalf("网段 = %v", nets)
	}
	for i, network := range nets {
		if network.String() != want[i] {
			t.Fatalf("第 %d 项 = %s, want %s", i, network, want[i])
		}
	}

	content := "server:\n  http-port: 3000\n  trusted-proxies: [\"10.0.0.1\", \"proxy.local\"]\ndatabase:\n  type: sqlite\n  file: kanban.db\n"
	if err = Load(writeConfig(t, []byte(content))); err == nil || !strings.Contains(err.Error(), "proxy.local") {
		t.Fatalf("Load() 对非法代理地址的错误 = %v", err)
	}
}

// TestLoadReturnsErrors 验证缺失文件和无效配置通过错误返回。
func TestLoadReturnsErrors(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("Load() 对不存在的配置文件未返回错误")
	}

	cases := map[string]string{
		"不是 YAML":         "server: [",
		"端口越界":            "server:\n  http-port: 70000\ndatabase:\n  type: sqlite\n  file: kanban.db\n",
		"未知数据库类型":         "server:\n  http-port: 3000\ndatabase:\n  type: mysql\n",
		"SQLite 缺少文件":     "server:\n  http-port: 3000\ndatabase:\n  type: sqlite\n",
		"PostgreSQL 缺少库名": "server:\n  http-port: 3000\ndatabase:\n  type: postgres\n  host: db\n  port: 5432\n  username: kanban\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Load(writeConfig(t, []byte(content))); err == nil {
				t.Fatal("Load() 对无效配置未返回错误")
			}
		})
	}
}
