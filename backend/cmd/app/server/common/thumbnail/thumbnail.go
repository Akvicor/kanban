// Package thumbnail 为图片生成两种规格的缩略图：短边 360 和 720，不放大。
//
// 支持 JPEG、PNG、GIF（动图逐帧缩放，保留动画）、WebP（动图只取第一帧）、TIFF、BMP、AVIF。
// JPEG 和 TIFF 按 EXIF 方向矫正全部 8 种情况；AVIF 的方向由解码库处理。
// 输出格式：GIF 动图输出 GIF，带透明通道输出 PNG，其余输出质量 80 的 JPEG。
package thumbnail

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"sync"

	_ "github.com/gen2brain/avif" // 注册 AVIF 解码
	_ "golang.org/x/image/bmp"    // 注册 BMP 解码
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff" // 注册 TIFF 解码
	_ "golang.org/x/image/webp" // 注册 WebP 解码
)

const (
	// MaxFileSize 是生成缩略图的文件大小上限，与参考项目一致。
	MaxFileSize = 64 << 20
	// MaxPixels 是原图像素数上限，防止解压后占用过多内存。
	MaxPixels = 100_000_000
	// maxAnimationWork 是 GIF 动图「帧数 × 像素数」的上限，超过时只取第一帧生成静态缩略图。
	maxAnimationWork = 500_000_000
	jpegQuality      = 80
)

// Sizes 是缩略图的规格（短边像素），从大到小。
var Sizes = []int{720, 360}

// ErrNotImage 表示文件不是支持的图片，或超过大小、像素上限。这样的文件不算图片。
var ErrNotImage = errors.New("不是可以生成缩略图的图片")

// Result 是生成结果。Width、Height 是原图按方向矫正后的尺寸；Ext 是缩略图扩展名（jpg、png、gif）。
type Result struct {
	Width      int
	Height     int
	Ext        string
	Thumbnails map[int][]byte
}

// mu 保证同一时间只处理一张图片，限制内存占用。
var mu sync.Mutex

// Generate 读取 path 指向的图片并生成缩略图。不是支持的图片或超过上限时返回 ErrNotImage。
func Generate(path string) (*Result, error) {
	mu.Lock()
	defer mu.Unlock()

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxFileSize {
		return nil, ErrNotImage
	}
	config, format, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width*config.Height > MaxPixels {
		return nil, ErrNotImage
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	if format == "gif" {
		return generateGIF(file, config)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, ErrNotImage
	}
	orientation := 1
	if format == "jpeg" || format == "tiff" {
		if _, err = file.Seek(0, io.SeekStart); err == nil {
			orientation = readOrientation(file, format)
		}
	}
	return static(img, orientation)
}

// targetSize 返回把 w×h 缩到短边 size 时的尺寸，不放大。方向矫正只交换宽高，不改变短边，因此可以在原方向上计算。
func targetSize(w, h, size int) (int, int) {
	short := min(w, h)
	if short <= size {
		return w, h
	}
	return max(1, w*size/short), max(1, h*size/short)
}

func resize(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)
	return dst
}

// static 生成静态缩略图：先在原方向上缩放，再按 EXIF 方向矫正，避免对大图整体做一次方向变换。
func static(img image.Image, orientation int) (*Result, error) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	opaque := isOpaque(img)
	ext := "jpg"
	if !opaque {
		ext = "png"
	}
	result := &Result{Width: w, Height: h, Ext: ext, Thumbnails: map[int][]byte{}}
	if orientation >= 5 {
		result.Width, result.Height = h, w
	}

	source := img
	for _, size := range Sizes {
		tw, th := targetSize(w, h, size)
		scaled := resize(source, tw, th)
		source = scaled
		var buf bytes.Buffer
		var err error
		oriented := orient(scaled, orientation)
		if opaque {
			err = jpeg.Encode(&buf, oriented, &jpeg.Options{Quality: jpegQuality})
		} else {
			err = png.Encode(&buf, oriented)
		}
		if err != nil {
			return nil, fmt.Errorf("编码缩略图失败: %w", err)
		}
		result.Thumbnails[size] = buf.Bytes()
	}
	return result, nil
}

// isOpaque 判断图片是否没有透明像素。图片类型自带判断时直接使用，否则逐像素检查。
func isOpaque(img image.Image) bool {
	if o, ok := img.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}
	return true
}

// animated 生成 GIF 动图缩略图：逐帧按处置方式合成到画布，再把整帧缩放并按该帧的调色板抖动量化。
func animated(all *gif.GIF, config image.Config) (*Result, error) {
	canvas := image.NewRGBA(image.Rect(0, 0, config.Width, config.Height))
	outputs := map[int]*gif.GIF{}
	for _, size := range Sizes {
		outputs[size] = &gif.GIF{LoopCount: all.LoopCount}
	}
	var previous *image.RGBA
	for i, frame := range all.Image {
		disposal := byte(0)
		if i < len(all.Disposal) {
			disposal = all.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			previous = image.NewRGBA(canvas.Bounds())
			copy(previous.Pix, canvas.Pix)
		}
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		source := image.Image(canvas)
		for _, size := range Sizes {
			tw, th := targetSize(config.Width, config.Height, size)
			scaled := resize(source, tw, th)
			source = scaled
			palette := frame.Palette
			if len(palette) == 0 {
				palette = color.Palette{color.Black, color.White}
			}
			paletted := image.NewPaletted(scaled.Bounds(), palette)
			draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), scaled, image.Point{})
			out := outputs[size]
			out.Image = append(out.Image, paletted)
			delay := 0
			if i < len(all.Delay) {
				delay = all.Delay[i]
			}
			out.Delay = append(out.Delay, delay)
		}

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			copy(canvas.Pix, previous.Pix)
		}
	}

	result := &Result{Width: config.Width, Height: config.Height, Ext: "gif", Thumbnails: map[int][]byte{}}
	for size, out := range outputs {
		var buf bytes.Buffer
		if err := gif.EncodeAll(&buf, out); err != nil {
			return nil, fmt.Errorf("编码缩略图失败: %w", err)
		}
		result.Thumbnails[size] = buf.Bytes()
	}
	return result, nil
}
