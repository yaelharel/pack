package layer

import (
	"archive/tar"
	"io"

	"github.com/buildpacks/imgutil"
)

type TarWriter interface {
	WriteHeader(hdr *tar.Header) error
	Write(b []byte) (int, error)
	Close() error
}

// TODO: Move to method on `imgutil.Image`
func NewWriterForImage(image imgutil.Image, fileWriter io.Writer) (TarWriter, error) {
	os, err := image.OS()
	if err != nil {
		return nil, err
	}
	if os == "windows" {
		return NewWindowsWriter(fileWriter), nil
	}
	return tar.NewWriter(fileWriter), nil
}

/*
imgutil      lifecycle
    ^          ^
     \        /
        pack


 */
