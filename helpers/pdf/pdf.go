package pdf

import (
	"io"
	"path/filepath"

	"github.com/ledongthuc/pdf"
)

func PDFLoad(pathToPDF string) (content, title string, err error) {
	f, p, err := pdf.Open(pathToPDF)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	var (
		reader io.Reader
		b      []byte
	)

	reader, err = p.GetPlainText()
	if err != nil {
		return
	}

	b, err = io.ReadAll(reader)
	if err != nil {
		return
	}

	return string(b), filepath.Base(pathToPDF), nil
}
