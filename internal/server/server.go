package server

import (
	"context"
	"log/slog"
	"net/http"
)

type Server struct {
	addr string // 待ち受けるアドレス（例: ":8080"）
}

func New(addr string) *Server {
	return &Server{addr: addr}
}

// Start は HTTP サーバーを起動する。React の静的ファイルを配信するだけ。
// ctx がキャンセルされるとグレースフルシャットダウンで処理中のリクエストを完了させてから安全に停止する。
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// フロントエンド（React）未実装のため暫定ページを返す
	// TODO: React dist を embed して配信する
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<h1>Fleamane</h1><p>Starting up...</p>"))
	})

	srv := &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	// ctx キャンセル時（Ctrl+C など）にサーバーを停止する
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	slog.Info("server listening", "addr", s.addr)

	// ErrServerClosed は Shutdown() による正常停止なのでエラー扱いしない
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}
