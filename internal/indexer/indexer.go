package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// IndexResult summarises how many files were processed in a run.
type IndexResult struct {
	Indexed int
	Skipped int
}

// supportedExtensions are the file types IndexRepo will process.
var supportedExtensions = map[string]bool{
	".go":  true,
	".ts":  true,
	".tsx": true,
	".js":  true,
	".jsx": true,
	".py":  true,
}

// ignoredDirs are directory names that IndexRepo will not descend into.
var ignoredDirs = map[string]bool{
	"vendor": true, "node_modules": true, ".git": true,
	"dist": true, "build": true, ".next": true, "__pycache__": true,
}

// IndexRepo walks repoPath, extracts symbols from supported source files, and
// writes one markdown brain file per source file under
// brainPath/knowledge/repos/<repo-name>/<relative-path>.md.
// Returns a summary of how many files were indexed vs. skipped (unchanged).
func IndexRepo(repoPath, brainPath string) (IndexResult, error) {
	repoName := filepath.Base(repoPath)
	repoRoot := filepath.Join(brainPath, "knowledge", "repos", repoName)

	manifestPath := filepath.Join(repoRoot, ".manifest.json")
	manifest, err := LoadRepoManifest(manifestPath)
	if err != nil {
		return IndexResult{}, fmt.Errorf("load repo manifest: %w", err)
	}

	var result IndexResult

	walkErr := filepath.Walk(repoPath, func(absPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if info.IsDir() {
			if ignoredDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(absPath))
		if !supportedExtensions[ext] {
			return nil
		}

		relPath, _ := filepath.Rel(repoPath, absPath)

		hash, err := FileHash(absPath)
		if err != nil {
			return nil // skip unreadable files
		}

		if manifest.Files[relPath] == hash {
			result.Skipped++
			return nil
		}

		src, err := os.ReadFile(absPath)
		if err != nil {
			return nil
		}

		symbols := ExtractSymbols(filepath.Base(absPath), src)

		brainFile := filepath.Join(repoRoot, strings.TrimSuffix(relPath, ext)+".md")
		if err := writeBrainFile(brainFile, repoName, relPath, symbols); err != nil {
			return fmt.Errorf("write brain file for %s: %w", relPath, err)
		}

		manifest.Files[relPath] = hash
		result.Indexed++
		return nil
	})

	if walkErr != nil {
		return result, walkErr
	}

	if err := SaveRepoManifest(manifestPath, manifest); err != nil {
		return result, fmt.Errorf("save repo manifest: %w", err)
	}

	return result, nil
}

// writeBrainFile writes a markdown file for a single source file.
func writeBrainFile(destPath, repoName, relPath string, symbols []Symbol) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	var sb strings.Builder
	today := time.Now().Format("2006-01-02")

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("tags: [\"repo:%s\", \"file:%s\"]\n", repoName, relPath))
	sb.WriteString("confidence: medium\n")
	sb.WriteString("source: index-repo\n")
	sb.WriteString(fmt.Sprintf("updated: %s\n", today))
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# %s\n\n", relPath))
	sb.WriteString(fmt.Sprintf("Repository: `%s`\n\n", repoName))

	if len(symbols) == 0 {
		sb.WriteString("_(no top-level symbols found)_\n")
	} else {
		for _, sym := range symbols {
			sb.WriteString(fmt.Sprintf("## %s (%s)\n\n", sym.Name, sym.Kind))
			sb.WriteString("```\n")
			sb.WriteString(sym.Code)
			sb.WriteString("\n```\n\n")
		}
	}

	return os.WriteFile(destPath, []byte(sb.String()), 0644)
}
