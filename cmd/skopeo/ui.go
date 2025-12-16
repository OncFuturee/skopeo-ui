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

	opts := copyOptions{
		global:    requestGlobal,
		srcImage:  srcOpts,
		destImage: destOpts,
		retryOpts: retryOpts,
		copy:      copyOpts,
	}

	var buf bytes.Buffer
	err := opts.run([]string{req.Source, req.Destination}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Copy successful"))
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
