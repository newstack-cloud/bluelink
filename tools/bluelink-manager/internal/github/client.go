package github

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/paths"
	"github.com/newstack-cloud/bluelink/tools/bluelink-manager/internal/ui"
)

const (
	repoOwner = "newstack-cloud"
	repoName  = "bluelink"
)

// Client handles GitHub API interactions.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new GitHub client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

// Release represents a GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
}

// GetLatestVersion fetches the latest version for a component.
func (c *Client) GetLatestVersion(tagPrefix string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", repoOwner, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse releases: %w", err)
	}

	// Find the latest release matching our tag prefix
	prefix := tagPrefix + "/v"
	for _, release := range releases {
		version, found := strings.CutPrefix(release.TagName, prefix)
		if found {
			return version, nil
		}
	}

	return "", fmt.Errorf("no release found for %s", tagPrefix)
}

// DownloadComponent downloads and installs a component.
func (c *Client) DownloadComponent(name, tagPrefix, version, archiveName, binaryName string, platform paths.Platform) error {
	ui.Info("Downloading %s v%s...", name, version)

	tag := fmt.Sprintf("%s/v%s", tagPrefix, version)

	// Windows uses .zip, others use .tar.gz
	ext := ".tar.gz"
	if platform.OS == "windows" {
		ext = ".zip"
	}
	archive := fmt.Sprintf("%s_%s_%s%s", archiveName, version, platform.String(), ext)

	url := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner,
		repoName,
		tag,
		archive,
	)
	checksumsURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/checksums.txt",
		repoOwner,
		repoName,
		tag,
	)

	// Download archive to temp file
	tmpFile, err := os.CreateTemp("", "bluelink-*"+ext)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := c.downloadFile(url, tmpFile); err != nil {
		return fmt.Errorf("failed to download %s: %w", name, err)
	}

	// Verify checksum
	if err := c.verifyChecksum(tmpFile.Name(), archive, checksumsURL); err != nil {
		ui.Warn("Checksum verification: %v", err)
	}

	// Extract binary
	if err := extractBinary(tmpFile.Name(), binaryName, paths.BinDir(), platform.OS == "windows"); err != nil {
		return fmt.Errorf("failed to extract %s: %w", name, err)
	}

	ui.Success("Installed %s to %s", binaryName, paths.BinDir())
	return nil
}

func (c *Client) downloadFile(url string, dest *os.File) error {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(dest, resp.Body)
	return err
}

func (c *Client) verifyChecksum(filePath, archiveName, checksumsURL string) error {
	// Download checksums
	resp, err := c.httpClient.Get(checksumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums not available")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Find expected checksum
	var expectedHash string
	for line := range strings.SplitSeq(string(body), "\n") {
		if strings.Contains(line, archiveName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expectedHash = parts[0]
				break
			}
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", archiveName)
	}

	// Compute actual checksum
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actualHash)) != 1 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	ui.Info("Checksum verified for %s", archiveName)
	return nil
}

func extractBinary(archivePath, binaryName, destDir string, isZip bool) error {
	if isZip {
		return extractBinaryFromZip(archivePath, binaryName, destDir)
	}
	return extractBinaryFromTarGz(archivePath, binaryName, destDir)
}

func extractBinaryFromZip(archivePath, binaryName, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	// On Windows, the binary has .exe extension
	binaryWithExt := binaryName + ".exe"

	for _, f := range r.File {
		name := filepath.Base(f.Name)
		if name == binaryWithExt && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}

			destPath := filepath.Join(destDir, binaryWithExt)
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return err
			}

			if _, err := io.Copy(outFile, rc); err != nil {
				outFile.Close()
				rc.Close()
				return err
			}
			outFile.Close()
			rc.Close()
			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binaryWithExt)
}

func extractBinaryFromTarGz(archivePath, binaryName, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the binary (could be at root or in a subdirectory)
		name := filepath.Base(header.Name)
		if name == binaryName && header.Typeflag == tar.TypeReg {
			destPath := filepath.Join(destDir, binaryName)
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("binary %s not found in archive", binaryName)
}
