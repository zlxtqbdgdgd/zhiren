package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"
)

//go:embed web
var webFS embed.FS

// version 由构建时 -ldflags "-X main.version=..." 注入
var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址，如 :8080")
	open := flag.Bool("open", true, "启动后自动打开浏览器（仅 Windows）")
	flag.Parse()

	content, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("加载内嵌资源失败: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(content)))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprintf(w, `{"name":"zhiren","version":%q}`, version)
	})

	url := "http://localhost" + *addr
	log.Printf("知人 · 干部信息管理系统 %s 已启动：%s", version, url)
	if *open && runtime.GOOS == "windows" {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
