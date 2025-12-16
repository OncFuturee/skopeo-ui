# Skopeo Image Downloader

[中文](README.md) | **English**

## English Introduction

This is a **Visual Container Image Downloader** based on [Skopeo](https://github.com/containers/skopeo).

**Original Project**: https://github.com/containers/skopeo

### Project Goal

While the official Skopeo is a powerful command-line tool, it can be challenging for users who simply want to download images, especially on Windows.

This project aims to provide a **simple, intuitive Web Interface** focused on **downloading container images** locally, without requiring Docker Desktop or complex environment configurations.

### Key Features

1.  **Visual Downloader**: A web interface focused on downloading images, supporting saving as local directories, Tar archives, etc.
2.  **Windows Friendly**:
    *   Optimized for Windows environments, fixing default architecture issues (automatically adapts to Linux/amd64).
    *   Runs without a Docker Daemon.
3.  **Image Browsing**: Supports viewing remote repository tags and image details (Inspect) to verify download targets.
4.  **Zero Dependency**: Removes `gpgme` CGO dependency, ready to use out of the box.

### Usage

#### 1. Build

On Windows, it is recommended to build with the `containers_image_openpgp` tag:

```powershell
go build -tags containers_image_openpgp ./cmd/skopeo
```

#### 2. Start Downloader

Run the `ui` subcommand to start the web server:

```powershell
.\skopeo.exe ui
```

Once started, visit `http://localhost:8080` in your browser to start downloading images.

#### 3. CLI Usage

This project remains fully compatible with all original Skopeo command-line features.
