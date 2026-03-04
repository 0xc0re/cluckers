package launch

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0xc0re/cluckers/internal/config"
)

func TestExtractSHMLauncher_UsesCluckersTmpDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	path, cleanup, err := ExtractSHMLauncher()
	if err != nil {
		t.Fatalf("ExtractSHMLauncher() error: %v", err)
	}
	defer cleanup()

	expectedPrefix := filepath.Join(tmp, "tmp")
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("ExtractSHMLauncher() path = %q, want prefix %q", path, expectedPrefix)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat extracted file: %v", err)
	}
	if info.Size() == 0 {
		t.Error("extracted shm_launcher.exe is empty")
	}

	// Cleanup should remove the file.
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove file at %q", path)
	}
}

func TestWriteBootstrapFile_UsesCluckersTmpDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	testData := []byte("BPS1" + strings.Repeat("\x00", 132))
	path, cleanup, err := WriteBootstrapFile(testData)
	if err != nil {
		t.Fatalf("WriteBootstrapFile() error: %v", err)
	}
	defer cleanup()

	expectedPrefix := filepath.Join(tmp, "tmp")
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("WriteBootstrapFile() path = %q, want prefix %q", path, expectedPrefix)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading bootstrap file: %v", err)
	}
	if len(data) != len(testData) {
		t.Errorf("bootstrap file size = %d, want %d", len(data), len(testData))
	}

	// Cleanup should remove the file.
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove file at %q", path)
	}
}

func TestWriteOIDCTokenFile_UsesCluckersTmpDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	path, cleanup, err := writeOIDCTokenFile("test-oidc-token-value")
	if err != nil {
		t.Fatalf("writeOIDCTokenFile() error: %v", err)
	}
	defer cleanup()

	expectedPrefix := filepath.Join(tmp, "tmp")
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("writeOIDCTokenFile() path = %q, want prefix %q", path, expectedPrefix)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading OIDC token file: %v", err)
	}
	if string(data) != "test-oidc-token-value" {
		t.Errorf("OIDC token file content = %q, want %q", string(data), "test-oidc-token-value")
	}

	// Cleanup should remove the file.
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("cleanup did not remove file at %q", path)
	}
}

func TestTmpDir_AutoCreated(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CLUCKERS_HOME", tmp)

	tmpDir := config.TmpDir()
	// Verify the tmp directory does NOT exist yet.
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Fatalf("tmp dir %q should not exist before first use, err: %v", tmpDir, err)
	}

	// ExtractSHMLauncher triggers EnsureDir on TmpDir.
	path, cleanup, err := ExtractSHMLauncher()
	if err != nil {
		t.Fatalf("ExtractSHMLauncher() error: %v", err)
	}
	defer cleanup()

	// Verify the tmp directory was created.
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("tmp dir %q was not created: %v", tmpDir, err)
	}
	if !info.IsDir() {
		t.Errorf("tmp dir %q is not a directory", tmpDir)
	}

	// Also verify file is inside the tmp dir.
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("extracted file %q is not under tmp dir %q", path, tmpDir)
	}
}

// TestNoDirectOsTempDir scans all production .go files in the repo for direct
// calls to os.TempDir(). All temp file operations should use config.TmpDir()
// for consistency across platforms and to prevent issues on restricted systems.
func TestNoDirectOsTempDir(t *testing.T) {
	// Find repo root by going up from internal/launch/ (this file's location).
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	fset := token.NewFileSet()
	var violations []string

	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor directory.
		if d.IsDir() && d.Name() == "vendor" {
			return filepath.SkipDir
		}

		// Skip non-Go files and test files.
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			// Skip files that fail to parse (e.g., wrong GOOS).
			return nil
		}

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			ident, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			if ident.Name == "os" && sel.Sel.Name == "TempDir" {
				pos := fset.Position(call.Pos())
				violations = append(violations, pos.String())
			}
			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("walking repo: %v", err)
	}

	for _, v := range violations {
		t.Errorf("found os.TempDir() call at %s — use config.TmpDir() instead", v)
	}
}
