package layer

import (
	"archive/tar"
	"fmt"
	"io"
	"path"

	"github.com/buildpacks/pack/internal/archive"
)

type Writer struct {
	tarWriter          *tar.Writer
	isWindows          bool
	isInitialized      bool
	existingLayerPaths map[string]bool
	wrote              string
}

func NewWriter(fileWriter io.Writer, imageOS string) *Writer {
	tarWriter := tar.NewWriter(fileWriter)

	isWindows := (imageOS == "windows")

	return &Writer{
		tarWriter:          tarWriter,
		isWindows:          isWindows,
		isInitialized:      false,
		existingLayerPaths: map[string]bool{},
	}
}

func (w *Writer) WriteHeader(tarHeader *tar.Header) error {
	if err := w.initializeLayer(); err != nil {
		return err
	}

	layerPath, err := w.prepareLayerPath(tarHeader.Name)
	if err != nil {
		return err
	}

	tarHeader.Name = layerPath

	return w.tarWriter.WriteHeader(tarHeader)
}

func (w *Writer) Write(content []byte) (int, error) {
	return w.tarWriter.Write(content)
}

func (w *Writer) AddFile(initialLayerPath string, txt string) error {
	if err := w.initializeLayer(); err != nil {
		return err
	}

	destLayerPath, err := w.prepareLayerPath(initialLayerPath)
	if err != nil {
		return err
	}

	return archive.AddFileToTar(w.tarWriter, destLayerPath, txt)
}

func (w *Writer) FromTarReader(tr *tar.Reader) error {
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		//re-write imageOS-appropriate header
		if err := w.WriteHeader(th); err != nil {
			return err
		}
		io.Copy(w.tarWriter, tr)
	}
	return nil
}

func (w *Writer) Close() error {
	if err := w.initializeLayer(); err != nil {
		return err
	}

	return w.tarWriter.Close()
}

func (w *Writer) initializeLayer() error {
	if !w.isWindows || w.isInitialized {
		return nil
	}

	if err := w.tarWriter.WriteHeader(&tar.Header{Name: "Files", Typeflag: tar.TypeDir}); err != nil {
		return err
	}
	w.existingLayerPaths["Files"] = true

	if err := w.tarWriter.WriteHeader(&tar.Header{Name: "Hives", Typeflag: tar.TypeDir}); err != nil {
		return err
	}
	w.existingLayerPaths["Hives"] = true

	w.isInitialized = true

	return nil
}

func (w *Writer) prepareLayerPath(initialLayerPath string) (string, error) {
	destLayerPath := initialLayerPath
	if w.isWindows {
		destLayerPath = path.Join("Files", initialLayerPath)
		err := w.writeParentDirHeaders(destLayerPath)
		if err != nil {
			return "", err
		}
	}

	if w.existingLayerPaths[destLayerPath] {
		return "", fmt.Errorf("attempted write of duplicate entry to layer: %s", destLayerPath)
	}
	w.existingLayerPaths[destLayerPath] = true

	return destLayerPath, nil
}

func (w *Writer) writeParentDirHeaders(childPath string) error {
	var parentPaths []string

	for {
		//loop through child's parent dirs until reaching the top
		parentPath := path.Dir(childPath)

		//break at the top
		if parentPath == "/" || parentPath == "." || parentPath == "" {
			break
		}

		//prepend each path so they are in order, shallowest-to-deepest
		parentPaths = append([]string{parentPath}, parentPaths...)

		//restart with next shallower path
		childPath = parentPath
	}

	for _, path := range parentPaths {
		//avoid duplicates
		if w.existingLayerPaths[path] {
			continue
		}
		w.existingLayerPaths[path] = true

		if err := w.tarWriter.WriteHeader(&tar.Header{Name: path, Typeflag: tar.TypeDir}); err != nil {
			return err
		}
	}

	return nil
}
