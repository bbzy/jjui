package context

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/idursun/jjui/internal/jj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAppContextCapturesWorkingDirectoryAndWorkspaceChangePreservesIt(t *testing.T) {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)
	workingDirectory, err = filepath.EvalSymlinks(workingDirectory)
	require.NoError(t, err)
	ctx := NewAppContext("/repo", nil)
	assert.Equal(t, workingDirectory, ctx.WorkingDirectory)

	ctx.ChangeWorkspace("/other/repo")
	assert.Equal(t, "/other/repo", ctx.Location)
	assert.Equal(t, workingDirectory, ctx.WorkingDirectory)
}

func TestFileDisplayFromSymlinkWorkingDirectory(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	repo := filepath.Join(root, "repo")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "src"), 0o755))
	link := filepath.Join(root, "link")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	for _, subdir := range []string{"", "src"} {
		t.Run(subdir, func(t *testing.T) {
			cwd := filepath.Join(link, subdir)
			t.Chdir(cwd)
			t.Setenv("PWD", cwd)
			ctx := NewAppContext(repo, nil)
			want := "src/main.go"
			if subdir != "" {
				want = "main.go"
			}
			// The file need not exist: deleted and historical files also display
			// relative to the directory where the application was started.
			file := jj.NewFileName("src/main.go")
			assert.Equal(t, want, file.Display(ctx.Location, ctx.WorkingDirectory))
			assert.Equal(t, "src/main.go", file.Path())
			if subdir != "" {
				assert.Equal(t, "../docs/readme.md", jj.NewFileName("docs/readme.md").Display(ctx.Location, ctx.WorkingDirectory))
			}
		})
	}
}

func TestChangeWorkspace(t *testing.T) {
	runner := &MainCommandRunner{Location: "/old/path"}
	ctx := &MainContext{
		Location:      "/old/path",
		CommandRunner: runner,
	}

	ctx.ChangeWorkspace("/new/path")

	assert.Equal(t, "/new/path", ctx.Location)
	assert.Equal(t, "/new/path", runner.Location)
}

func TestChangeWorkspace_UpdatesBothLocations(t *testing.T) {
	runner := &MainCommandRunner{Location: "/a"}
	ctx := &MainContext{
		Location:      "/a",
		CommandRunner: runner,
	}

	ctx.ChangeWorkspace("/b")
	assert.Equal(t, "/b", ctx.Location)
	assert.Equal(t, "/b", runner.Location)

	ctx.ChangeWorkspace("/c")
	assert.Equal(t, "/c", ctx.Location)
	assert.Equal(t, "/c", runner.Location)
}

func TestChangeWorkspace_NonMainCommandRunner(t *testing.T) {
	// If the runner is not a *MainCommandRunner, ctx.Location still updates
	// but the runner's location is unaffected (no panic).
	ctx := &MainContext{
		Location:      "/old",
		CommandRunner: nil,
	}

	require.NotPanics(t, func() {
		ctx.ChangeWorkspace("/new")
	})
	assert.Equal(t, "/new", ctx.Location)
}
