package spectest

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The wast2json-converted spec test suite. The tag must match the suite
// used by the targeted WasmEdge release (see test/spec in WasmEdge).
const (
	suiteTag = "wasm-core-20260322"
	suiteURL = "https://github.com/WasmEdge/wasmedge-spectest/archive/refs/tags/" +
		suiteTag + ".tar.gz"
)

// suiteRoot returns the extracted suite path, downloading into testdata/ on
// first use. Set WASMEDGE_SPECTEST_PATH to use a pre-downloaded suite.
func suiteRoot(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("WASMEDGE_SPECTEST_PATH"); path != "" {
		return path
	}

	cache := filepath.Join("testdata", "wasmedge-spectest-"+suiteTag)
	marker := filepath.Join(cache, ".complete")
	if _, err := os.Stat(marker); err == nil {
		return cache
	}

	t.Logf("downloading the spec test suite %s ...", suiteTag)
	if err := downloadSuite(cache); err != nil {
		t.Fatalf("failed to download the spec test suite: %v", err)
	}
	if err := os.WriteFile(marker, []byte(suiteURL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return cache
}

func downloadSuite(cache string) error {
	resp, err := http.Get(suiteURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", suiteURL, resp.Status)
	}

	tmp := cache + ".tmp"
	if err := os.RemoveAll(tmp); err != nil {
		return err
	}
	if err := extractTarGz(resp.Body, tmp); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if err := os.RemoveAll(cache); err != nil {
		return err
	}
	return os.Rename(tmp, cache)
}

// extractTarGz extracts a gzipped tarball into destDir, stripping the first
// path component (the "<repo>-<tag>/" prefix of GitHub archives).
func extractTarGz(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		name := hdr.Name
		if idx := strings.IndexByte(name, '/'); idx >= 0 {
			name = name[idx+1:]
		} else {
			continue
		}
		if name == "" {
			continue
		}
		// Guard against path traversal in archive entries.
		path := filepath.Join(destDir, filepath.FromSlash(name))
		if !strings.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry escapes destination: %q", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
	}
}

// listUnits returns the sorted unit names of a suite folder: the
// subdirectories containing a "<unit>.json" test manifest.
func listUnits(t *testing.T, root string, folder string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, folder))
	if err != nil {
		t.Fatalf("failed to list the %q folder: %v", folder, err)
	}
	var units []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, err := os.Stat(filepath.Join(root, folder, name, name+".json")); err == nil {
			units = append(units, name)
		}
	}
	return units
}
