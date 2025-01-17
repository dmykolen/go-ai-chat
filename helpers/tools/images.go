package tools

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
)

// Helper functions for image processing
func CompressImage(data []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	switch format {
	case "jpeg":
		err = jpeg.Encode(buf, img, &jpeg.Options{Quality: 85})
	case "png":
		encoder := png.Encoder{CompressionLevel: png.BestCompression}
		err = encoder.Encode(buf, img)
	default:
		return data, nil
	}

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Helper functions for image detect type
func DetectImageType(data []byte) (string, error) {
	_, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	return format, nil
}

// ReadImageFromFile read image from file and return bytes
func ReadImageFromFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, file)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
