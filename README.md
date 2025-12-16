# Skopeo UI (Fork)

**中文** | [English](README_US.md)

## 中文介绍

这是一个 [Skopeo](https://github.com/containers/skopeo) 的 Fork 版本。
**原项目地址**：https://github.com/containers/skopeo

### 本项目的主要工作

官方的 Skopeo 是一个强大的命令行工具，用于对容器镜像和镜像仓库执行各种操作。但它缺乏图形用户界面（UI），且在 Windows 平台上的预构建资源较少。

本项目在保留 Skopeo 所有原有功能的基础上，增加了以下特性：

1.  **Web UI 界面**：内置了一个轻量级的 Web 服务器，提供可视化的操作界面。目前已实现镜像检查 (Inspect) 功能，方便用户快速查看镜像信息。
2.  **Windows 友好**：针对 Windows 环境进行了优化。
    *   解决了在 Windows 下运行 `inspect` 等命令时，默认查找 Windows 架构镜像导致 "no image found" 的问题（自动回退到 Linux/amd64）。
    *   提供了适用于 Windows 的构建方案。
3.  **零依赖构建**：通过使用纯 Go 实现的 OpenPGP 库，移除了对 `gpgme` 的 CGO 依赖，使得在 Windows 上编译变得非常简单。

### 使用方法

#### 1. 构建

在 Windows 环境下，推荐使用 `containers_image_openpgp` 标签进行构建，以避免安装 GPGME 库的麻烦：

```powershell
go build -tags containers_image_openpgp ./cmd/skopeo
```

#### 2. 启动 UI

构建完成后，运行 `ui` 子命令启动 Web 服务：

```powershell
.\skopeo.exe ui
```

启动后，控制台会提示服务地址（通常为 `http://localhost:8080`）。打开浏览器访问该地址即可使用图形界面。

#### 3. 命令行使用

本项目完全兼容原版 Skopeo 的所有命令行功能。例如：

```powershell
# 检查镜像信息
.\skopeo.exe inspect docker://docker.io/library/alpine

# 复制镜像
.\skopeo.exe copy docker://docker.io/library/alpine dir:./my-alpine
```
