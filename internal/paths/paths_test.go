package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpack/pack/testhelpers"
)

func TestPaths(t *testing.T) {
	spec.Run(t, "Paths", testPaths, spec.Report(report.Terminal{}))
}

func testPaths(t *testing.T, when spec.G, it spec.S) {
	when("#FilePathToUri", func() {
		when("is windows", func() {
			when("path is absolute", func() {
				it("returns uri", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					uri, err := FilePathToUri(`C:\some\file.txt`)
					h.AssertNil(t, err)
					h.AssertEq(t, uri, "file:///C:/some/file.txt")
				})
			})

			when("path is relative", func() {
				it("returns uri", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					uri, err := FilePathToUri(`some\file.tgz`)
					h.AssertNil(t, err)

					uri, err = replaceCwdUriPathWith(uri, "absolute/path")
					h.AssertNil(t, err)

					h.AssertEq(t, uri, "file://absolute/path/some/file.tgz")
				})
			})
		})

		when("is *nix", func() {
			when("path is absolute", func() {
				it("returns uri", func() {
					h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")

					uri, err := FilePathToUri("/tmp/file.tgz")
					h.AssertNil(t, err)
					h.AssertEq(t, uri, "file:///tmp/file.tgz")
				})
			})

			when("path is relative", func() {
				it("returns uri", func() {
					h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")

					uri, err := FilePathToUri("some/file.tgz")
					h.AssertNil(t, err)

					uri, err = replaceCwdUriPathWith(uri, "absolute/path")
					h.AssertNil(t, err)

					h.AssertEq(t, uri, "file://absolute/path/some/file.tgz")
				})
			})
		})
	})

	when("#UriToFilePath", func() {
		when("is windows", func() {
			when("uri is drive", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					path, err := UriToFilePath(`file:///c:/laptop/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `c:\laptop\file.tgz`)
				})
			})

			when("uri is network share", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS != "windows", "Skipped on non-windows")

					path, err := UriToFilePath(`file://laptop/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `\\laptop\file.tgz`)
				})
			})
		})

		when("is *nix", func() {
			when("uri is valid", func() {
				it("returns path", func() {
					h.SkipIf(t, runtime.GOOS == "windows", "Skipped on windows")

					path, err := UriToFilePath(`file:///tmp/file.tgz`)
					h.AssertNil(t, err)

					h.AssertEq(t, path, `/tmp/file.tgz`)
				})
			})
		})
	})
}

func replaceCwdUriPathWith(uri, replacement string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return strings.Replace(uri, filepath.ToSlash(strings.TrimPrefix(cwd, `\\`)), replacement, 1), nil
}
