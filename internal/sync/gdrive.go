package sync

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GDriveBackend syncs brain files to a Google Drive folder.
type GDriveBackend struct {
	svc      *drive.Service
	folderID string
}

// NewGDriveBackend creates a Google Drive sync backend using a service account.
func NewGDriveBackend(ctx context.Context, serviceAccountPath, folderID string) (*GDriveBackend, error) {
	svc, err := drive.NewService(ctx, option.WithCredentialsFile(serviceAccountPath))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	return &GDriveBackend{svc: svc, folderID: folderID}, nil
}

func (g *GDriveBackend) Name() string { return "gdrive" }

// Init validates that the target folder is accessible.
func (g *GDriveBackend) Init(ctx context.Context) error {
	if g.folderID == "" {
		return fmt.Errorf("gdrive folder_id is required")
	}
	_, err := g.svc.Files.Get(g.folderID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("cannot access Drive folder %s: %w", g.folderID, err)
	}
	fmt.Printf("Google Drive folder verified: %s\n", g.folderID)
	return nil
}

// Push uploads all brain .md files to the Drive folder.
func (g *GDriveBackend) Push(ctx context.Context, brainPath string) error {
	// List existing files in folder to determine create vs update.
	existing, err := g.listRemoteFiles(ctx)
	if err != nil {
		return err
	}

	// Walk local brain files.
	var count int
	err = filepath.Walk(brainPath, func(path string, info os.FileInfo, err error) error {
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

		if fileID, ok := existing[rel]; ok {
			// Update existing file.
			_, err = g.svc.Files.Update(fileID, nil).
				Media(bytes.NewReader(data)).
				Context(ctx).
				Do()
		} else {
			// Create new file.
			f := &drive.File{
				Name:    rel,
				Parents: []string{g.folderID},
			}
			_, err = g.svc.Files.Create(f).
				Media(bytes.NewReader(data)).
				Context(ctx).
				Do()
		}
		if err != nil {
			return fmt.Errorf("upload %s: %w", rel, err)
		}
		count++
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("  Uploaded %d file(s) to Google Drive.\n", count)
	return nil
}

// Pull downloads all files from the Drive folder to the local brain.
func (g *GDriveBackend) Pull(ctx context.Context, brainPath string) error {
	existing, err := g.listRemoteFiles(ctx)
	if err != nil {
		return err
	}

	var count int
	for name, fileID := range existing {
		resp, err := g.svc.Files.Get(fileID).Download()
		if err != nil {
			return fmt.Errorf("download %s: %w", name, err)
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		localPath := filepath.Join(brainPath, name)
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", name, err)
		}
		if err := os.WriteFile(localPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
		count++
	}
	fmt.Printf("  Downloaded %d file(s) from Google Drive.\n", count)
	return nil
}

// Status compares local files with the Drive folder.
func (g *GDriveBackend) Status(ctx context.Context, brainPath string) (*Status, error) {
	remote, err := g.listRemoteFiles(ctx)
	if err != nil {
		return nil, err
	}

	// Collect local .md files.
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
		if _, ok := remote[name]; ok {
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

// listRemoteFiles returns a map of filename -> fileID for all files in the folder.
func (g *GDriveBackend) listRemoteFiles(ctx context.Context) (map[string]string, error) {
	files := make(map[string]string)
	query := fmt.Sprintf("'%s' in parents and trashed = false", g.folderID)
	pageToken := ""

	for {
		call := g.svc.Files.List().
			Q(query).
			Fields("nextPageToken, files(id, name)").
			Context(ctx)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list drive files: %w", err)
		}
		for _, f := range resp.Files {
			files[f.Name] = f.Id
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return files, nil
}
