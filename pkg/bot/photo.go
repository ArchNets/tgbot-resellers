package bot

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
)

func (b *Bot) processReceiptPhoto(fileID string) (string, error) {
	fileURL, err := b.api.GetFileDirectURL(fileID)
	if err != nil {
		return "", fmt.Errorf("failed to get file URL from telegram: %w", err)
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("failed to download photo from telegram: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram photo download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read photo bytes: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		if len(data) < 1500000 {
			return base64.StdEncoding.EncodeToString(data), nil
		}
		return "", fmt.Errorf("failed to decode photo image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxDim := 1280
	var outImg image.Image = img

	if width > maxDim || height > maxDim {
		var newW, newH int
		if width > height {
			newW = maxDim
			newH = (height * maxDim) / width
		} else {
			newH = maxDim
			newW = (width * maxDim) / height
		}

		dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
		for y := 0; y < newH; y++ {
			for x := 0; x < newW; x++ {
				srcX := (x * width) / newW
				srcY := (y * height) / newH
				dst.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
			}
		}
		outImg = dst
	}

	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: 80}
	if err := jpeg.Encode(&buf, outImg, opts); err != nil {
		return "", fmt.Errorf("failed to encode jpeg: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
