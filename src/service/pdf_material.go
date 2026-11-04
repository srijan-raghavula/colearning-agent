package service

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

const MaxPDFMaterialBytes int64 = 10 << 20

func ExtractPDFText(r io.Reader, size int64) (string, error) {
	if size <= 0 || size > MaxPDFMaterialBytes {
		return "", fmt.Errorf("pdf must be between 1 byte and %d MB", MaxPDFMaterialBytes>>20)
	}
	temp, err := os.CreateTemp("", "colearn-material-*.pdf")
	if err != nil {
		return "", err
	}
	name := temp.Name()
	defer os.Remove(name)
	defer temp.Close()
	if _, err := io.CopyN(temp, io.LimitReader(r, MaxPDFMaterialBytes+1), MaxPDFMaterialBytes+1); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}
	_, reader, err := pdf.Open(name)
	if err != nil {
		return "", fmt.Errorf("could not read pdf: %w", err)
	}
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("could not extract pdf text: %w", err)
	}
	raw, err := io.ReadAll(textReader)
	if err != nil {
		return "", err
	}
	text := strings.Join(strings.Fields(string(raw)), " ")
	if text == "" {
		return "", errors.New("pdf contains no extractable text")
	}
	return text, nil
}

func SafePDFTitle(filename string) string {
	title := strings.TrimSpace(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	if title == "" {
		return "Uploaded PDF material"
	}
	return title
}
