package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const vercelBlobAPI = "https://blob.vercel-storage.com"

// VercelBlobBackend syncs brain files to Vercel Blob storage.
type VercelBlobBackend struct {
	token  string
	client *http.Client
}

// NewVercelBlobBackend creates a Vercel Blob sync backend.
func NewVercelBlobBackend(token string) *VercelBlobBackend {
	return &VercelBlobBackend{token: token, client: &http.Client{}}
}

func (v *VercelBlobBackend) Name() string { return "vercel-blob" }

// Init validates credentials by listing blobs.
func (v *VercelBlobBackend) Init(ctx context.Context) error {
	_, err := v.listBlobs(ctx)
	if err != nil {
		return fmt.Errorf("vercel blob auth failed: %w", err)
	}
	fmt.Println("Vercel Blob credentials verified.")
	return nil
}

// Push uploads all brain .md files to Vercel Blob.
func (v *VercelBlobBackend) Push(ctx context.Context, brainPath string) error {
	var count int
	err := filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(brainPath, path)

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}

		pathname := "kai-brain/" + rel
		if err := v.putBlob(ctx, pathname, data); err != nil {
			return fmt.Errorf("upload %s: %w", rel, err)
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("  Uploaded %d file(s) to Vercel Blob.\n", count)
	return nil
}

// Pull downloads all brain blobs to the local brain directory.
func (v *VercelBlobBackend) Pull(ctx context.Context, brainPath string) error {
	blobs, err := v.listBlobs(ctx)
	if err != nil {
		return err
	}

	var count int
	for _, b := range blobs {
		if !strings.HasPrefix(b.Pathname, "kai-brain/") {
			continue
		}
		rel := strings.TrimPrefix(b.Pathname, "kai-brain/")
		if rel == "" || !strings.HasSuffix(rel, ".md") {
			continue
		}

		data, err := v.getBlob(ctx, b.URL)
		if err != nil {
			return fmt.Errorf("download %s: %w", rel, err)
		}

		localPath := filepath.Join(brainPath, rel)
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", rel, err)
		}
		if err := os.WriteFile(localPath, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		count++
	}
	fmt.Printf("  Downloaded %d file(s) from Vercel Blob.\n", count)
	return nil
}

// Status compares local files with Vercel Blob.
func (v *VercelBlobBackend) Status(ctx context.Context, brainPath string) (*Status, error) {
	blobs, err := v.listBlobs(ctx)
	if err != nil {
		return nil, err
	}

	remote := make(map[string]bool)
	for _, b := range blobs {
		if strings.HasPrefix(b.Pathname, "kai-brain/") {
			rel := strings.TrimPrefix(b.Pathname, "kai-brain/")
			if strings.HasSuffix(rel, ".md") {
				remote[rel] = true
			}
		}
	}

	local := make(map[string]bool)
	filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return err
		}
		rel, _ := filepath.Rel(brainPath, path)
		local[rel] = true
		return nil
	})

	s := &Status{}
	for name := range local {
		if remote[name] {
			s.InSync = append(s.InSync, name)
		} else {
			s.LocalOnly = append(s.LocalOnly, name)
		}
	}
	for name := range remote {
		if !local[name] {
			s.RemoteOnly = append(s.RemoteOnly, name)
		}
	}
	return s, nil
}

type blobEntry struct {
	URL      string `json:"url"`
	Pathname string `json:"pathname"`
}

type listResponse struct {
	Blobs  []blobEntry `json:"blobs"`
	Cursor string      `json:"cursor"`
}

func (v *VercelBlobBackend) listBlobs(ctx context.Context) ([]blobEntry, error) {
	var all []blobEntry
	cursor := ""

	for {
		url := vercelBlobAPI + "?prefix=kai-brain/"
		if cursor != "" {
			url += "&cursor=" + cursor
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+v.token)

		resp, err := v.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list blobs: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("list blobs: %s — %s", resp.Status, string(body))
		}

		var lr listResponse
		if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
			return nil, fmt.Errorf("decode list response: %w", err)
		}
		all = append(all, lr.Blobs...)
		if lr.Cursor == "" {
			break
		}
		cursor = lr.Cursor
	}
	return all, nil
}

func (v *VercelBlobBackend) putBlob(ctx context.Context, pathname string, data []byte) error {
	req, err := http.NewRequestWithContext(ctx, "PUT", vercelBlobAPI+"/"+pathname, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+v.token)
	req.Header.Set("x-api-blob-no-suffix", "1")
	req.Header.Set("Content-Type", "text/markdown")

	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("put blob %s: %s — %s", pathname, resp.Status, string(body))
	}
	return nil
}

func (v *VercelBlobBackend) getBlob(ctx context.Context, blobURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", blobURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get blob: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
