// Fleamane のエントリーポイント。
//
// このバイナリは「メインプロセス」と「サービスプロセス」の2役を兼ねる。
// 第1引数が "serve" かどうかで動作が切り替わる。
//
// ┌─ メインプロセス（引数なし・ユーザーがダブルクリックで起動）
// │   → runMain() を呼ぶ
// │   → DB 初期化 → 有効サービスを子プロセスとして起動 → React UI 配信 → ブラウザ起動
// │   → シグナル（Ctrl+C）を受け取るまで動き続ける
// │
// └─ サービスプロセス（"serve" サブコマンド・メインプロセスが子プロセスとして自分自身を起動）
//     → runServe() を呼ぶ
//     → 例: ./fleamane serve --service=mercari --port=8001
//     → 指定されたサービスの HTTP サーバーだけを起動して動き続ける
//
// つまりメインプロセスが自分自身を複数の子プロセスとして起動することで
// 各サービス（mercari/:8001、yahooauction/:8002 …）を独立したプロセスとして管理している。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog" // Go 1.21 標準の構造化ログ。サードパーティ不要。
	"os"
	"os/signal"
	"syscall"
	"time"

	"fleamane/internal/browser"
	"fleamane/internal/launcher"
	"fleamane/internal/server"
	"fleamane/internal/shared/db"
)

func main() {
	// ログ出力をテキスト形式（key=value）に設定
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// 第1引数が "serve" ならサービスプロセスとして動作
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		if err := runServe(os.Args[2:]); err != nil {
			slog.Error("service failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := runMain(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// runMain はユーザーが直接バイナリを起動したときの処理。
// DB 初期化・子プロセス起動・React UI 配信・ブラウザ起動をまとめて行い、
// アプリ全体のライフサイクルを管理する。
func runMain() error {
	// SIGINT（Ctrl+C）・SIGTERM でキャンセルされる Context を作成
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// DB（SQLite） 接続（WALモード・外部キー有効）
	conn, err := db.Open("./fleamane.db")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	// 未適用のマイグレーションを順番に実行
	if err := db.Migrate(conn); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// app_settings で enabled=true になっているサービス（ユーザが使いたいサービス）を子プロセスとして起動する
	l, err := launcher.New(conn)
	if err != nil {
		return fmt.Errorf("create launcher: %w", err)
	}
	if err := l.Start(ctx); err != nil {
		return fmt.Errorf("start services: %w", err)
	}
	defer l.Stop()

	// フロントエンド（React dist） を配信するメインHTTPサーバー（:8080）
	srv := server.New(":8080")
	go func() {
		if err := srv.Start(ctx); err != nil {
			slog.Error("server error", "error", err)
		}
	}()

	// サーバー起動を少し待ってからブラウザを開く（goroutineで非同期）
	time.Sleep(500 * time.Millisecond)
	go browser.Open("http://localhost:8080")

	// シグナルを受け取るまでブロック
	<-ctx.Done()
	slog.Info("shutting down")
	return nil
}

// runServe はメインプロセスが子プロセスとして自分自身を起動するときの処理。
// ユーザーが直接呼ぶことはなく、launcher が内部的に呼び出す。
// --service でサービス名、--port でポート番号を受け取り、そのサービスだけを起動する。
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	service := fs.String("service", "", "service name")
	port := fs.Int("port", 0, "port number")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *service == "" {
		return fmt.Errorf("--service is required")
	}
	if *port == 0 {
		return fmt.Errorf("--port is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := db.Open("./fleamane.db")
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	// TODO: 各サービスの実装ができたらここでサービス名で振り分ける
	slog.Info("service started (stub)", "service", *service, "port", *port)
	<-ctx.Done()
	return nil
}
