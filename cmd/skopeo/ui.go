package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"go.podman.io/common/pkg/auth"
	"go.podman.io/common/pkg/retry"
)

//go:embed ui_assets/*
var uiAssets embed.FS

func uiCmd(global *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Start the Skopeo Web UI",
		Long:  `Starts a local web server that provides a user interface for Skopeo operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI(global)
		},
	}
	return cmd
}

func runUI(global *globalOptions) error {
	// Ensure policy.json exists
	if err := ensurePolicyFile(); err != nil {
		fmt.Printf("Warning: Failed to create policy.json: %v\n", err)
	}

	// Serve static files
	fsys, err := fs.Sub(uiAssets, "ui_assets")
	if err != nil {
		return err
	}
	http.Handle("/", http.FileServer(http.FS(fsys)))

	// API endpoints
	http.HandleFunc("/api/inspect", func(w http.ResponseWriter, r *http.Request) {
		handleInspect(w, r, global)
	})
	http.HandleFunc("/api/copy", func(w http.ResponseWriter, r *http.Request) {
		handleCopy(w, r, global)
	})
	http.HandleFunc("/api/download", func(w http.ResponseWriter, r *http.Request) {
		handleDownload(w, r, global)
	})
	http.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
		handleDelete(w, r, global)
	})
	http.HandleFunc("/api/list-tags", func(w http.ResponseWriter, r *http.Request) {
		handleListTags(w, r, global)
	})
	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(w, r, global)
	})
	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r, global)
	})

	port := "8080"
	fmt.Printf("Starting Skopeo UI at http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		return err
	}
	return nil
}

func getGlobalOptions(r *http.Request, global *globalOptions) *globalOptions {
	// Create a copy of global options for this request to avoid race conditions
	// and to allow per-request overrides.
	requestGlobal := *global

	// Default to linux/amd64 if not specified, as most images are linux based.
	// This fixes "no image found ... OS windows" error on Windows.
	if requestGlobal.overrideOS == "" {
		requestGlobal.overrideOS = "linux"
	}
	if requestGlobal.overrideArch == "" {
		requestGlobal.overrideArch = "amd64"
	}

	// Allow overrides from query params
	if osParam := r.URL.Query().Get("os"); osParam != "" {
		requestGlobal.overrideOS = osParam
	}
	if archParam := r.URL.Query().Get("arch"); archParam != "" {
		requestGlobal.overrideArch = archParam
	}
	return &requestGlobal
}

func handleInspect(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	imageName := r.URL.Query().Get("image")
	if imageName == "" {
		http.Error(w, "Missing 'image' query parameter", http.StatusBadRequest)
		return
	}

	requestGlobal := getGlobalOptions(r, global)

	// Initialize options
	sharedOpts := &sharedImageOptions{}
	imgOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: requestGlobal,
			shared: sharedOpts,
		},
	}
	retryOpts := &retry.Options{MaxRetry: 3}

	opts := inspectOptions{
		global:    requestGlobal,
		image:     imgOpts,
		retryOpts: retryOpts,
	}

	var buf bytes.Buffer
	err := opts.run([]string{imageName}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
}

func handleCopy(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Source       string `json:"source"`
		Destination  string `json:"destination"`
		OverrideOS   string `json:"os"`
		OverrideArch string `json:"arch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestGlobal := getGlobalOptions(r, global)
	if req.OverrideOS != "" {
		requestGlobal.overrideOS = req.OverrideOS
	}
	if req.OverrideArch != "" {
		requestGlobal.overrideArch = req.OverrideArch
	}

	sharedOpts := &sharedImageOptions{}
	srcOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: requestGlobal,
			shared: sharedOpts,
		},
	}
	destOpts := &imageDestOptions{
		imageOptions: &imageOptions{
			dockerImageOptions: dockerImageOptions{
				global: requestGlobal,
				shared: sharedOpts,
			},
		},
	}
	retryOpts := &retry.Options{MaxRetry: 3}
	copyOpts := &sharedCopyOptions{}
	deprecatedTLSVerify := &deprecatedTLSVerifyOption{}

	opts := copyOptions{
		global:              requestGlobal,
		srcImage:            srcOpts,
		destImage:           destOpts,
		retryOpts:           retryOpts,
		copy:                copyOpts,
		deprecatedTLSVerify: deprecatedTLSVerify,
	}

	var buf bytes.Buffer
	err := opts.run([]string{req.Source, req.Destination}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Copy successful"))
}

func handleDownload(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Source       string `json:"source"`
		Format       string `json:"format"`
		Filename     string `json:"filename"`
		OverrideOS   string `json:"os"`
		OverrideArch string `json:"arch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestGlobal := getGlobalOptions(r, global)
	if req.OverrideOS != "" {
		requestGlobal.overrideOS = req.OverrideOS
	}
	if req.OverrideArch != "" {
		requestGlobal.overrideArch = req.OverrideArch
	}

	// Create temporary directory for the download
	// Use current directory to avoid Windows drive letter colon issues in docker-archive transport parsing
	tempDir, err := os.MkdirTemp(".", "skopeo-download-*")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp directory: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	// Construct destination path
	tempFile := filepath.Join(tempDir, req.Filename)
	// Convert Windows path to Unix-style path for archive formats
	// This ensures we get a relative path like "skopeo-download-123/image.tar" which has no colons
	unixPath := filepath.ToSlash(tempFile)

	// docker-archive and oci-archive require a reference name in the format
	// For docker-archive: docker-archive:path/file.tar:imagename:tag
	var destination string
	if req.Format == "docker-archive" || req.Format == "oci-archive" {
		// Extract a simple reference name from the source (remove transport prefix and normalize)
		sourceRef := req.Source
		if idx := strings.Index(sourceRef, "://"); idx != -1 {
			sourceRef = sourceRef[idx+3:] // Remove transport prefix like "docker://"
		}
		// Convert to lowercase and replace special chars
		sourceRef = strings.ToLower(sourceRef)
		sourceRef = strings.ReplaceAll(sourceRef, "/", "-")
		destination = fmt.Sprintf("%s:%s:%s", req.Format, unixPath, sourceRef)
	} else {
		destination = fmt.Sprintf("%s:%s", req.Format, unixPath)
	}

	sharedOpts := &sharedImageOptions{}
	srcOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: requestGlobal,
			shared: sharedOpts,
		},
	}
	destOpts := &imageDestOptions{
		imageOptions: &imageOptions{
			dockerImageOptions: dockerImageOptions{
				global: requestGlobal,
				shared: sharedOpts,
			},
		},
	}
	retryOpts := &retry.Options{MaxRetry: 3}
	copyOpts := &sharedCopyOptions{}
	deprecatedTLSVerify := &deprecatedTLSVerifyOption{}

	opts := copyOptions{
		global:              requestGlobal,
		srcImage:            srcOpts,
		destImage:           destOpts,
		retryOpts:           retryOpts,
		copy:                copyOpts,
		deprecatedTLSVerify: deprecatedTLSVerify,
	}

	var buf bytes.Buffer
	err = opts.run([]string{req.Source, destination}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Read the file and send it to the browser
	fileData, err := os.ReadFile(tempFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read downloaded file: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Type", "application/x-tar")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", req.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))

	// Write file content to response
	w.Write(fileData)
}

func handleDelete(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	imageName := r.URL.Query().Get("image")
	if imageName == "" {
		http.Error(w, "Missing 'image' query parameter", http.StatusBadRequest)
		return
	}

	requestGlobal := getGlobalOptions(r, global)
	sharedOpts := &sharedImageOptions{}
	imgOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: requestGlobal,
			shared: sharedOpts,
		},
	}
	retryOpts := &retry.Options{MaxRetry: 3}

	opts := deleteOptions{
		global:    requestGlobal,
		image:     imgOpts,
		retryOpts: retryOpts,
	}

	var buf bytes.Buffer
	err := opts.run([]string{imageName}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Delete successful"))
}

// ensurePolicyFile creates a default policy.json file if it doesn't exist
func ensurePolicyFile() error {
	// Check if policy file already exists in user's config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".config", "containers")
	policyPath := filepath.Join(configDir, "policy.json")

	// Check if file already exists
	if _, err := os.Stat(policyPath); err == nil {
		return nil // File already exists
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Create a permissive default policy
	defaultPolicy := `{
    "default": [
        {
            "type": "insecureAcceptAnything"
        }
    ],
    "transports": {
        "docker-daemon": {
            "": [{"type":"insecureAcceptAnything"}]
        }
    }
}`

	// Write policy file
	if err := os.WriteFile(policyPath, []byte(defaultPolicy), 0644); err != nil {
		return err
	}

	fmt.Printf("Created default policy file at: %s\n", policyPath)
	return nil
}
func handleListTags(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	imageName := r.URL.Query().Get("image")
	if imageName == "" {
		http.Error(w, "Missing 'image' query parameter", http.StatusBadRequest)
		return
	}

	requestGlobal := getGlobalOptions(r, global)
	sharedOpts := &sharedImageOptions{}
	imgOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: requestGlobal,
			shared: sharedOpts,
		},
	}
	retryOpts := &retry.Options{MaxRetry: 3}

	opts := tagsOptions{
		global:    requestGlobal,
		image:     imgOpts,
		retryOpts: retryOpts,
	}

	var buf bytes.Buffer
	err := opts.run([]string{imageName}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
}

func handleLogin(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Registry string `json:"registry"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	opts := loginOptions{
		global: global,
		loginOpts: auth.LoginOptions{
			Username:           req.Username,
			Password:           req.Password,
			Stdin:              os.Stdin,
			Stdout:             io.Discard,
			AcceptRepositories: true,
		},
	}

	var buf bytes.Buffer
	err := opts.run([]string{req.Registry}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Login successful"))
}

func handleLogout(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	registry := r.URL.Query().Get("registry")
	if registry == "" {
		http.Error(w, "Missing 'registry' query parameter", http.StatusBadRequest)
		return
	}

	opts := logoutOptions{
		global: global,
		logoutOpts: auth.LogoutOptions{
			Stdout:             io.Discard,
			AcceptRepositories: true,
		},
	}

	var buf bytes.Buffer
	err := opts.run([]string{registry}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Logout successful"))
}
