package sourcecode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LocalProvider struct {
	root string
}

func NewLocalProvider(cfg Config) Provider {
	root := discoverLocalRoot(cfg)
	if root == "" {
		return nil
	}
	return &LocalProvider{root: root}
}

func (p *LocalProvider) FileURL(path string, line int, commitOrBranch string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return filepath.Join(p.root, path)
}

func (p *LocalProvider) GetFile(ctx context.Context, path string, ref string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	fullPath := filepath.Join(p.root, filepath.Clean(path))
	return os.ReadFile(fullPath)
}

func (p *LocalProvider) GetCommitsForFiles(ctx context.Context, files []string, since time.Time) ([]Commit, error) {
	return nil, nil
}

func (p *LocalProvider) ResolveRef(ctx context.Context, ref string) (string, error) {
	return "local", nil
}

func (p *LocalProvider) TestConnection(ctx context.Context) error {
	info, err := os.Stat(p.root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("local source root is not a directory")
	}
	return nil
}

func discoverLocalRoot(cfg Config) string {
	candidates := candidateRoots(cfg)
	for _, root := range candidates {
		if root == "" {
			continue
		}
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			return root
		}
	}
	return ""
}

func candidateRoots(cfg Config) []string {
	name := strings.TrimSpace(cfg.Name)
	owner := strings.TrimSpace(cfg.Owner)
	if name == "" {
		return nil
	}

	seen := map[string]bool{}
	add := func(items *[]string, value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		*items = append(*items, value)
	}

	var roots []string
	if env := strings.TrimSpace(os.Getenv("GOSNAG_SOURCE_ROOT")); env != "" {
		for _, base := range strings.Split(env, string(os.PathListSeparator)) {
			add(&roots, filepath.Join(base, name))
			if owner != "" {
				add(&roots, filepath.Join(base, owner, name))
			}
		}
	}

	cwd, _ := os.Getwd()
	if cwd == "" {
		return roots
	}

	baseDirs := []string{
		cwd,
		filepath.Dir(cwd),
		filepath.Dir(filepath.Dir(cwd)),
	}
	for _, base := range baseDirs {
		add(&roots, filepath.Join(base, name))
		if owner != "" {
			add(&roots, filepath.Join(base, owner, name))
		}
	}
	return roots
}
