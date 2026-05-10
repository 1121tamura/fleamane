package launcher

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Launcher struct {
	conn  *sql.DB      // app_settings を読むための DB 接続
	procs []*exec.Cmd  // 起動した子プロセスの一覧
	mu    sync.Mutex
}

func New(conn *sql.DB) (*Launcher, error) {
	return &Launcher{conn: conn}, nil
}

type serviceConfig struct {
	name string // サービス名（例: mercari）
	port string // ポート番号（例: 8001）
}

// loadServices は app_settings から enabled=true のサービス一覧を取得する。
func (l *Launcher) loadServices() ([]serviceConfig, error) {
	rows, err := l.conn.Query(`
		SELECT key FROM app_settings
		WHERE category = 'service' AND key LIKE '%.enabled' AND value = 'true'
	`)
	if err != nil {
		return nil, fmt.Errorf("query services: %w", err)
	}
	defer rows.Close()

	var services []serviceConfig
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		// "service.mercari.enabled" → "mercari"
		name := strings.TrimPrefix(strings.TrimSuffix(key, ".enabled"), "service.")

		var port string
		if err := l.conn.QueryRow(
			`SELECT value FROM app_settings WHERE key = ?`,
			"service."+name+".port",
		).Scan(&port); err != nil {
			return nil, fmt.Errorf("get port for %s: %w", name, err)
		}

		services = append(services, serviceConfig{name: name, port: port})
	}
	return services, rows.Err()
}

// Start は有効なサービスを子プロセスとして起動する。
// 自分自身のバイナリを "serve --service=xxx --port=xxx" で起動する。
func (l *Launcher) Start(ctx context.Context) error {
	services, err := l.loadServices()
	if err != nil {
		return fmt.Errorf("load services: %w", err)
	}

	// 実行中のバイナリパスを取得（自分自身を子プロセスとして起動するため）
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, svc := range services {
		cmd := exec.CommandContext(ctx, exe,
			"serve",
			"--service="+svc.name,
			"--port="+svc.port,
		)
		cmd.Stdout = os.Stdout // 子プロセスのログをメインプロセスに集約
		cmd.Stderr = os.Stderr

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start service %s: %w", svc.name, err)
		}
		// l.procs に子プロセスを追加
		slog.Info("service started", "service", svc.name, "port", svc.port)
		l.procs = append(l.procs, cmd)
	}
	return nil
}

// Stop は全子プロセスに SIGINT を送り、終了を待つ。
// main の defer で呼ばれる。
func (l *Launcher) Stop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, cmd := range l.procs {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}
	for _, cmd := range l.procs {
		_ = cmd.Wait()
	}
}
