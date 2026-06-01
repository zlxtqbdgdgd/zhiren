package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net"
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
	dataPath := flag.String("data", "data/zhiren.json", "数据文件路径")
	open := flag.Bool("open", true, "启动后自动打开浏览器")
	browser := flag.String("browser", "", "指定打开用的浏览器可执行文件路径（留空用系统默认）")
	flag.Parse()

	store, err := NewStore(*dataPath)
	if err != nil {
		log.Fatalf("初始化数据失败: %v", err)
	}
	store.StartBackupScheduler()

	staticSub, err := fs.Sub(webFS, "web/static")
	if err != nil {
		log.Fatalf("静态资源加载失败: %v", err)
	}

	app, err := newApp(store, webFS)
	if err != nil {
		log.Fatalf("模板加载失败: %v", err)
	}

	localURL := "http://localhost" + *addr
	log.Printf("知人 · 干部信息管理系统 %s 已启动：%s（默认账号 admin / admin123，请尽快修改）", version, localURL)
	for _, ip := range localIPs() {
		log.Printf("局域网其他设备可访问：http://%s%s", ip, *addr)
	}
	if *open {
		openBrowser(localURL, *browser)
	}
	if err := http.ListenAndServe(*addr, app.routes(staticSub)); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func openBrowser(url, browserPath string) {
	if browserPath != "" {
		_ = exec.Command(browserPath, url).Start()
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}

// localIPs 返回本机的局域网 IPv4 地址，供提示其他设备如何访问。
func localIPs() []string {
	var out []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return out
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil && isPrivateIPv4(ip4) {
				out = append(out, ip4.String())
			}
		}
	}
	return out
}

// isPrivateIPv4 仅保留常见局域网网段（10/172.16-31/192.168），过滤虚拟网卡等噪声。
func isPrivateIPv4(ip net.IP) bool {
	switch {
	case ip[0] == 10:
		return true
	case ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31:
		return true
	case ip[0] == 192 && ip[1] == 168:
		return true
	}
	return false
}
