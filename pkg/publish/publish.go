// Package publish uploads Python artefacts to a PyPI-compatible registry.
package publish

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ErrAlreadyExists is returned when the registry reports the file was
// already uploaded (HTTP 400 with "File already exists").
var ErrAlreadyExists = errors.New("publish: file already exists on registry")

// ErrUnauthorized is returned when the registry rejects the token
// (HTTP 403).
var ErrUnauthorized = errors.New("publish: unauthorized (check your API token)")

// UploadRequest describes a publish job.
type UploadRequest struct {
	Files    []string // absolute paths to .whl or .tar.gz files
	Registry string   // upload endpoint URL
	Token    string   // PyPI API token (pypi-... format)
	DryRun   bool
}

// UploadResult is the per-file outcome.
type UploadResult struct {
	File string
	URL  string // canonical registry URL for the uploaded release, or "" on dry-run
}

// Upload posts each file to the registry. Returns per-file results.
// If DryRun is true, files are inspected but not posted.
func Upload(req UploadRequest) ([]UploadResult, error) {
	registry := req.Registry
	if registry == "" {
		registry = "https://upload.pypi.org/legacy/"
	}

	var results []UploadResult
	for _, path := range req.Files {
		result, err := uploadOne(path, registry, req.Token, req.DryRun)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}
	return results, nil
}

func uploadOne(path, registry, token string, dryRun bool) (UploadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return UploadResult{}, fmt.Errorf("publish: read %s: %w", path, err)
	}

	name, version, filetype, pyversion := parseArtefact(filepath.Base(path), data)
	h := sha256.Sum256(data)
	sha256hex := fmt.Sprintf("%x", h)

	result := UploadResult{File: path}

	if dryRun {
		return result, nil
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fields := map[string]string{
		":action":          "file_upload",
		"protocol_version": "1",
		"name":             name,
		"version":          version,
		"filetype":         filetype,
		"pyversion":        pyversion,
		"sha256_digest":    sha256hex,
	}
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return UploadResult{}, fmt.Errorf("publish: write field %s: %w", k, err)
		}
	}
	fw, err := mw.CreateFormFile("content", filepath.Base(path))
	if err != nil {
		return UploadResult{}, fmt.Errorf("publish: create form file: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return UploadResult{}, fmt.Errorf("publish: write file data: %w", err)
	}
	mw.Close()

	req, err := http.NewRequest("POST", registry, &body)
	if err != nil {
		return UploadResult{}, fmt.Errorf("publish: build request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	creds := base64.StdEncoding.EncodeToString([]byte("__token__:" + token))
	req.Header.Set("Authorization", "Basic "+creds)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UploadResult{}, fmt.Errorf("publish: post to %s: %w", registry, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case 200, 201:
		result.URL = fmt.Sprintf("https://pypi.org/project/%s/%s/", name, version)
		return result, nil
	case 400:
		msg := string(raw)
		if strings.Contains(msg, "already exists") || strings.Contains(msg, "already been") {
			return UploadResult{}, ErrAlreadyExists
		}
		return UploadResult{}, fmt.Errorf("publish: registry returned 400: %s", msg)
	case 403:
		return UploadResult{}, ErrUnauthorized
	default:
		return UploadResult{}, fmt.Errorf("publish: registry returned %s: %s", resp.Status, string(raw))
	}
}

// parseArtefact extracts name, version, filetype, pyversion from a
// wheel filename (PEP 427) or sdist filename.
func parseArtefact(filename string, data []byte) (name, version, filetype, pyversion string) {
	if strings.HasSuffix(filename, ".whl") {
		// PEP 427: {distribution}-{version}(-{build})?-{python}-{abi}-{platform}.whl
		base := strings.TrimSuffix(filename, ".whl")
		parts := strings.Split(base, "-")
		if len(parts) >= 5 {
			name = strings.ReplaceAll(parts[0], "_", "-")
			version = parts[1]
			pyversion = parts[2]
			filetype = "bdist_wheel"
			return
		}
	}
	if strings.HasSuffix(filename, ".tar.gz") {
		// sdist: {name}-{version}.tar.gz
		base := strings.TrimSuffix(filename, ".tar.gz")
		idx := strings.LastIndex(base, "-")
		if idx > 0 {
			name = strings.ReplaceAll(base[:idx], "_", "-")
			version = base[idx+1:]
			filetype = "sdist"
			pyversion = "source"
			return
		}
	}
	// Fall back to reading METADATA from wheel zip.
	if strings.HasSuffix(filename, ".whl") {
		if n, v := metadataFromWheel(data); n != "" {
			name = n
			version = v
			filetype = "bdist_wheel"
			pyversion = "py3"
			return
		}
	}
	name = strings.TrimSuffix(filename, filepath.Ext(filename))
	filetype = "bdist_wheel"
	pyversion = "py3"
	return
}

func metadataFromWheel(data []byte) (name, version string) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", ""
	}
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, ".dist-info/METADATA") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", ""
		}
		raw, _ := io.ReadAll(rc)
		rc.Close()
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name: ") {
				name = strings.TrimPrefix(line, "Name: ")
			}
			if strings.HasPrefix(line, "Version: ") {
				version = strings.TrimPrefix(line, "Version: ")
			}
			if name != "" && version != "" {
				return
			}
		}
	}
	return "", ""
}
