package main

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/spf13/cobra"
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

	port := "8080"
	fmt.Printf("Starting Skopeo UI at http://localhost:%s\n", port)

	// Open browser?
	// openBrowser("http://localhost:" + port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
		return err
	}
	return nil
}

func handleInspect(w http.ResponseWriter, r *http.Request, global *globalOptions) {
	imageName := r.URL.Query().Get("image")
	if imageName == "" {
		http.Error(w, "Missing 'image' query parameter", http.StatusBadRequest)
		return
	}

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

	// Initialize options
	sharedOpts := &sharedImageOptions{}
	// We could load defaults from env vars here if we wanted to be thorough
	// e.g. sharedOpts.authFilePath = os.Getenv("REGISTRY_AUTH_FILE")

	imgOpts := &imageOptions{
		dockerImageOptions: dockerImageOptions{
			global: &requestGlobal,
			shared: sharedOpts,
		},
	}

	retryOpts := &retry.Options{
		MaxRetry: 3,
	}

	opts := inspectOptions{
		global:    &requestGlobal,
		image:     imgOpts,
		retryOpts: retryOpts,
	}

	var buf bytes.Buffer
	// opts.run expects []string{imageName} and writes to the writer
	err := opts.run([]string{imageName}, &buf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
}
