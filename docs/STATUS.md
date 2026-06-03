# 项目状态与续作指引

> 最后更新：2026-06-03。新 session 接手时先读这份 + `CONTEXT.md` + `docs/adr/`。

## 当前状态

- **v1.0.0 已发布**到 GitHub Releases（私有仓库 `zlxtqbdgdgd/zhiren`）。
- 安装包 `zhiren-setup-v1.0.0.exe` ≈ 84 MB，**已含绿色浏览器**（ungoogled-chromium 109）。
- 全部 v1 功能已完成并在本地（macOS）真机跑通验证：登录/三角色、干部 CRUD、精准搜索（张锋≠张峰）、组合筛选、计算字段（年龄/工作年限/到龄）、到龄提醒+导出、批量维护、CSV(BOM) 导入导出、结构报表、字段校验（身份证校验码）、自动备份+一键恢复。

## 交付前必做（尚未做）

- [ ] **在真 Windows 7 机器上 smoke test**：装一遍 → 双击桌面"知人" → 确认自带浏览器能打开页面。
      （我在 macOS 上无法验证 Win7 运行时；若因缺 VC++ 运行库打不开，换 **Supermium** 便携版重打。）
- [ ] 现场确认：是否允许运行随包的绿色浏览器（涉密内网软件白名单）。

## v2 待办（已与用户确认推后）

- 开机自启做成 **Windows 服务**（当前靠"把桌面图标放进启动文件夹"）。
- **zzb** 导入导出（需现场提供样例文件再定字段映射）。
- 政策文件管理、完整"填写指引"文案、高级可视化分析。
- 退休年龄**渐进式延迟**对照表的自动套用（当前只实现"按性别/身份的固定周岁规则 + 个案手工覆盖"，ADR-0003 的渐进表尚未自动套）。
- 真正的 **.xlsx** 导入导出（当前是 Excel 可直接打开的 UTF-8 CSV/BOM）。

## 本地开发环境

- Go：`brew install go`（当前 1.26，日常用）；CI 同版本校验用 `~/go/bin/go1.20`（Win7 目标，最后支持 Win7 的 Go）。
- **代码必须保持 Go 1.20 可编译**：勿用 1.21+ 标准库（slices/maps/cmp、min/max/clear、log/slog）及 1.22+ 的 ServeMux 方法路由（`"GET /x"`）。提交前 `~/go/bin/go1.20 build ./... && ~/go/bin/go1.20 vet ./...`。
- 跑起来：`cd zhiren && go run .` → http://localhost:8080，默认账号 `admin/admin123`，数据写 `./data/zhiren.json`。

## 构建与发布

- 推 `v*` tag 触发 `.github/workflows/release.yml`：Windows runner 用 go1.20 编译 exe → 下载 ungoogled-chromium 109 → Inno Setup 打包 → 发布到 Releases。`ci.yml` 在每次 push 交叉编译验证。
- 重出同一版本：`gh release delete vX.Y.Z --yes --cleanup-tag`，再 `git tag -a vX.Y.Z && git push origin vX.Y.Z`。
- 取包给朋友：`gh release download v1.0.0 -R zlxtqbdgdgd/zhiren`（物理隔离内网，gh 仅作下载渠道，U 盘带入）。

## 代码地图

| 文件 | 职责 |
|---|---|
| `model.go` | ~40 字段定义（元数据驱动）+ 日期解析 + 计算字段（年龄/工作年限/到龄） |
| `store.go` | 内存+JSON 原子存储、干部/用户/规则/设置、审计、备份/恢复/定时备份 |
| `auth.go` | 盐+迭代哈希、会话、用户增删改、改密 |
| `query.go` | 精准筛选、全局快查、到龄名单计算 |
| `validate.go` | 字段校验（必填/日期/枚举/身份证校验码） |
| `csvio.go` | CSV(BOM) 导入导出 |
| `reports.go` | 结构统计报表 |
| `server.go` | 路由、鉴权中间件、渲染、各页 handler |
| `admin.go` | 管理员：账号/规则/设置/备份/恢复 |
| `main.go` | 接线、局域网 IP 探测、`-data`/`-browser`/`-addr` 参数 |
| `web/templates/*.html`、`web/static/style.css` | 服务端渲染页面（兼容老 Chromium） |
| `installer/zhiren.iss` | Inno 安装脚本（Win7+、ProgramData 数据目录、绿色浏览器接口） |

## 运行期约定

- 数据：服务器机 `C:\ProgramData\zhiren\data\zhiren.json`（可写），同目录 `backups\` 每日自动备份、滚动保留 30 份。
- 安装包靠 `installer/zhiren.iss` 中 `#if FileExists(...\dist\browser\chrome.exe)` 决定是否打入浏览器；CI 已自动放入。
