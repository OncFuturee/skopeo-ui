# Skopeo Image Downloader

**中文** | [English](README_US.md)

## 中文介绍

这是一个基于 [Skopeo](https://github.com/containers/skopeo) 开发的**容器镜像可视化下载器**。

**原项目地址**：https://github.com/containers/skopeo

### 项目目标

官方的 Skopeo 虽然功能强大，但作为命令行工具，对于仅需要下载镜像的用户来说门槛稍高，且在 Windows 平台上的体验不够友好。

本项目旨在提供一个**简单、直观的 Web 界面**，专注于帮助用户**下载容器镜像**到本地，无需安装 Docker Desktop 或配置复杂的运行环境。

### 主要特性

1.  **可视化下载器**：提供专注于下载的 Web 界面，支持将镜像下载为本地文件夹、Tar 归档包等格式。
2.  **Windows 友好**：
    *   针对 Windows 环境优化，解决了默认架构识别错误的问题（自动适配 Linux/amd64）。
    *   无需 Docker Daemon 守护进程即可运行。
3.  **镜像浏览**：支持查看远程仓库的镜像标签（Tags）和详细信息（Inspect），方便确认下载目标。
4.  **零依赖构建**：移除 `gpgme` CGO 依赖，开箱即用。

### 使用方法

#### 1. 构建

在 Windows 环境下，推荐使用 `containers_image_openpgp` 标签进行构建：

```powershell
go build -tags containers_image_openpgp ./cmd/skopeo
```

#### 2. 启动下载器

运行 `ui` 子命令启动 Web 服务：

```powershell
.\skopeo.exe ui
```

启动后，浏览器访问 `http://localhost:8080` 即可开始下载镜像。

#### 3. 命令行使用

本项目完全兼容原版 Skopeo 的所有命令行功能。
