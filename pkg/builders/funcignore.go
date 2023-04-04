package builders

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	gitignore "github.com/sabhiram/go-gitignore"

	fn "knative.dev/func/pkg/functions"
)

// funcignoreBuilder wraps the implement of Builder.
// funcignoreBuilder symlinks files filterred by .funcignore to a temporary directory
// then sets function root to the temporary path and calls the inner Builder.
type funcignoreBuilder struct {
	fn.Builder
}

func WrapBuilderWithIgnorer(b fn.Builder) funcignoreBuilder {
	return funcignoreBuilder{Builder: b}
}

func (b funcignoreBuilder) Build(ctx context.Context, f fn.Function) error {
	fi, err := gitignore.CompileIgnoreFile(filepath.Join(f.Root, fn.FuncignoreFile))
	if err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "function-buidler")
	if err != nil {
		return fmt.Errorf("cannot create temporary dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	err = filepath.Walk(f.Root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(f.Root, path)
		if err != nil {
			return err
		}
		if fi.MatchesPath(relPath) {
			return nil
		}
		linkPath := filepath.Join(tmp, relPath)
		if err := os.MkdirAll(filepath.Dir(linkPath), os.ModePerm); err != nil {
			return err
		}
		return os.Symlink(path, linkPath)
	})
	if err != nil {
		return err
	}

	f.Root = tmp
	return b.Builder.Build(ctx, f)
}
