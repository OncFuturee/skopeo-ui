# Skopeo UI (Fork)

[中文](README.md) | **English**

## English Introduction

This is a fork of [Skopeo](https://github.com/containers/skopeo).
**Original Project**: https://github.com/containers/skopeo

### What's New

The official Skopeo is a powerful command-line utility for performing various operations on container images and image repositories. However, it lacks a Graphical User Interface (UI) and has limited pre-built resources for Windows.

This fork builds upon the original Skopeo and adds the following features:

1.  **Web UI**: A built-in lightweight web server providing a visual interface. Currently supports image inspection, allowing users to quickly view image details.
2.  **Windows Friendly**: Optimized for Windows environments.
    *   Fixes the issue where Skopeo defaults to looking for Windows architecture images when running on Windows OS (automatically falling back to Linux/amd64).
    *   Provides easy build instructions for Windows.
3.  **Easy Build**: Removes the CGO dependency on `gpgme` by using a pure Go implementation of OpenPGP, making compilation on Windows straightforward.

### Usage

#### 1. Build

On Windows, it is recommended to build with the `containers_image_openpgp` tag to avoid complex CGO dependencies:

```powershell
go build -tags containers_image_openpgp ./cmd/skopeo
```

#### 2. Start UI

After building, run the `ui` subcommand to start the web server:

```powershell
.\skopeo.exe ui
```

Once started, open your browser and visit the address shown in the console (usually [http://localhost:8080](http://localhost:8080)).

#### 3. CLI Usage

This project remains fully compatible with all original Skopeo command-line features. For example:

```powershell
# Inspect an image
.\skopeo.exe inspect docker://docker.io/library/alpine

# Copy an image
.\skopeo.exe copy docker://docker.io/library/alpine dir:./my-alpine
```
