# 知人（zhiren）· 干部信息管理系统

面向组织人事场景的干部信息管理系统：干部组集中维护，多科室在**局域网**内同步查看、搜索、导出，并支持到龄提醒、批量维护、数据分析等。

> 当前为**骨架版**：已贯通「代码 → Windows 安装包 → GitHub Release」发布链路，业务功能按 v1 计划逐步填入。

## 形态

一台局域网内的 Windows 机器（≥ Win7）安装服务端（Go 单文件 + SQLite + 自带绿色浏览器），开机自起；其他 Win7 电脑 / 华为平板点桌面图标即用，零安装。详见 [`docs/adr/`](docs/adr/)。

## 设计文档

- [`CONTEXT.md`](CONTEXT.md) — 领域术语表（干部 / 科室 / 维护者 / 查看者 / 到龄 / 精准搜索 …）
- [`docs/adr/0001`](docs/adr/0001-lan-server-browser-architecture.md) — 局域网 B/S 架构
- [`docs/adr/0002`](docs/adr/0002-offline-intranet-deployment.md) — 离线内网部署，GitHub 仅作分发中转
- [`docs/adr/0003`](docs/adr/0003-configurable-retirement-age.md) — 退休年龄可配置
- [`docs/adr/0004`](docs/adr/0004-tech-stack-go-sqlite-single-binary.md) — 技术栈与一键安装包
- 原始需求：[`docs/requirements/`](docs/requirements/)

## v1 范围

干部档案（含家庭成员、简历）· 计算字段（年龄/工作年限/到龄）· 组合精确筛选 + 全局快查 · 维护者/查看者/管理员 + 操作日志 · 自动备份 + 一键恢复 · 批量维护 · 到龄提醒 · Excel 导入导出 · 结构统计报表 · 基础字段校验。

**推到 v2**：政策文件功能、`zzb` 格式对接、完整填写指引、高级可视化分析。

## 本地运行（需 Go 1.20）

```bash
go run .              # 默认监听 :8080，浏览器打开 http://localhost:8080
go run . -addr :9000  # 自定义端口
```

## 发布安装包

推送一个 `v*` 标签即触发 [release 工作流](.github/workflows/release.yml)：在 Windows runner 上编译 exe → 用 Inno Setup 打成安装包 → 发布到 GitHub Releases。

```bash
git tag v0.0.1
git push origin v0.0.1
```

随后从 Releases 下载 `zhiren-setup-*.exe`，经移动介质带入内网交付。
