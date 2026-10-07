package api

import (
	"context"
	"encoding/json"
	"errors"
	"kanban/cmd/app/server/app/dro"
	"kanban/cmd/app/server/global/db"
	"kanban/cmd/config"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

// TestHealthReturnsReadyForDisposableDatabase 验证连接临时 SQLite 数据库时健康检查返回 200。
func TestHealthReturnsReadyForDisposableDatabase(t *testing.T) {
	previous := config.Global
	config.Global = &config.Model{Database: config.DatabaseModel{
		Type: config.DatabaseSQLite,
		File: filepath.Join(t.TempDir(), "health.db"),
	}}
	t.Cleanup(func() { config.Global = previous })
	if _, err := db.Create(); err != nil {
		t.Fatalf("创建临时数据库失败: %v", err)
	}
	sqlDB, err := db.Get().DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	req := httptest.NewRequest(http.MethodGet, "/api/sys/info/health", nil)
	recorder := httptest.NewRecorder()
	if err = Sys.Health(echo.New().NewContext(req, recorder)); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("health 状态=%d, body=%s", recorder.Code, recorder.Body.String())
	}
	var health dro.SysHealth
	if err = json.Unmarshal(recorder.Body.Bytes(), &health); err != nil {
		t.Fatal(err)
	}
	if health.Status != "healthy" || health.Checks["database"] != "healthy" {
		t.Fatalf("health 响应错误: %#v", health)
	}
}

// TestRunHealthChecksAggregatesAllChecks 验证所有检查都会执行，任一失败则整体失败。
func TestRunHealthChecksAggregatesAllChecks(t *testing.T) {
	runs := 0
	health := runHealthChecks(context.Background(), map[string]healthCheck{
		"healthy": func(context.Context) error {
			runs++
			return nil
		},
		"unhealthy": func(context.Context) error {
			runs++
			return errors.New("failed")
		},
	})
	if runs != 2 {
		t.Fatalf("只执行了 %d 个检查", runs)
	}
	if health.Status != "unhealthy" || health.Checks["healthy"] != "healthy" || health.Checks["unhealthy"] != "unhealthy" {
		t.Fatalf("聚合结果错误: %#v", health)
	}
}

// TestRunHealthChecksHonorsTimeout 验证检查超时时判为失败。
func TestRunHealthChecksHonorsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	health := runHealthChecks(ctx, map[string]healthCheck{
		"database": func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	})
	if health.Status != "unhealthy" || health.Checks["database"] != "unhealthy" {
		t.Fatalf("超时结果错误: %#v", health)
	}
}
