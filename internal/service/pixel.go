// Package service 提供业务逻辑层。
package service

import (
	"bytes"
	"fmt"
	"image"
	"image/png"

	_ "image/gif"  // 注册 GIF 解码器
	_ "image/jpeg" // 注册 JPEG 解码器
)

// ConvertToPixel 将原始图片按「最近邻采样」缩放为 size×size 的像素点图，
// 返回 PNG 编码后的字节流。最近邻采样不插值，能保留像素边缘，适合像素画。
func ConvertToPixel(data []byte, size int) ([]byte, error) {
	if size <= 0 || size > 512 {
		return nil, fmt.Errorf("invalid size: %d", size)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	srcW, srcH := bounds.Dx(), bounds.Dy()
	if srcW == 0 || srcH == 0 {
		return nil, fmt.Errorf("empty image")
	}

	// 输出图带 alpha 通道，保留原图透明区域
	out := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// 最近邻：映射到原图对应像素
			sx := x * srcW / size
			sy := y * srcH / size
			if sx >= srcW {
				sx = srcW - 1
			}
			if sy >= srcH {
				sy = srcH - 1
			}
			out.Set(x, y, img.At(sx, sy))
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return buf.Bytes(), nil
}
