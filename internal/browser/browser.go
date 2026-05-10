package browser

import (
	"log/slog"
	"os/exec"
	"runtime"
)

// Open はデフォルトブラウザで URL を開く。
// OS ごとに異なるコマンドを使用する。
func Open(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	// macOS のカーネル
	case "darwin":
		cmd = exec.Command("open", url)
	// windows
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		slog.Warn("browser open failed", "url", url, "error", err)
	}
}
