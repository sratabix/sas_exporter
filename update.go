package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const repo = "sratabix/sas_exporter"

var httpClient = &http.Client{Timeout: 60 * time.Second}

func selfUpdate() error {
	switch runtime.GOARCH {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("self-update not supported on %s", runtime.GOARCH)
	}

	latest, err := latestVersion()
	if err != nil {
		return fmt.Errorf("fetching latest version: %w", err)
	}

	if latest == version {
		fmt.Printf("Already at latest version %s.\n", version)
		return nil
	}

	fmt.Printf("Updating %s -> %s...\n", version, latest)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	url := fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/sas_exporter_linux_%s",
		repo, latest, runtime.GOARCH,
	)

	tmp, err := downloadTemp(url, filepath.Dir(exePath))
	if err != nil {
		return fmt.Errorf("downloading binary: %w", err)
	}
	defer os.Remove(tmp)

	if err := os.Chmod(tmp, 0755); err != nil {
		return err
	}
	// Rename is atomic on Linux when src and dst are on the same filesystem,
	// which is guaranteed here since tmp is created in the same directory.
	if err := os.Rename(tmp, exePath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("Updated to %s. Restarting service...\n", latest)
	restartService()
	return nil
}

func latestVersion() (string, error) {
	resp, err := httpClient.Get(
		fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in API response")
	}
	return release.TagName, nil
}

func downloadTemp(url, dir string) (string, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp(dir, ".sas_exporter_update_*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}
