package thumbnail

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/avif"
)

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// halves 生成左半红、右半蓝的 w×h 图片。
func halves(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{255, 0, 0, 255}
			if x >= w/2 {
				c = color.RGBA{0, 0, 255, 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// withOrientation 在 JPEG 的 SOI 之后插入只含方向标签的 EXIF 段。
func withOrientation(jpegData []byte, orientation uint16) []byte {
	var tiff bytes.Buffer
	tiff.WriteString("MM\x00*")
	_ = binary.Write(&tiff, binary.BigEndian, uint32(8))
	_ = binary.Write(&tiff, binary.BigEndian, uint16(1))
	_ = binary.Write(&tiff, binary.BigEndian, []uint16{0x0112, 3})
	_ = binary.Write(&tiff, binary.BigEndian, uint32(1))
	_ = binary.Write(&tiff, binary.BigEndian, []uint16{orientation, 0})
	_ = binary.Write(&tiff, binary.BigEndian, uint32(0))
	payload := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	segment := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	segment = append(segment, payload...)
	return append(append([]byte{0xFF, 0xD8}, segment...), jpegData[2:]...)
}

func decode(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func isRed(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0xc000 && g < 0x4000 && b < 0x4000
}

func TestJPEGOrientationAndSizes(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, halves(1600, 800), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	// 方向 6：显示时需要顺时针旋转 90°，原图左半（红）转到上方，矫正后宽高互换。
	result, err := Generate(writeFile(t, "a.jpg", withOrientation(buf.Bytes(), 6)))
	if err != nil {
		t.Fatal(err)
	}
	if result.Width != 800 || result.Height != 1600 || result.Ext != "jpg" {
		t.Fatalf("结果 = %dx%d %s", result.Width, result.Height, result.Ext)
	}
	for size, data := range result.Thumbnails {
		img := decode(t, data)
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size*2 {
			t.Fatalf("%d 规格尺寸 = %v", size, b)
		}
		if !isRed(img.At(size/2, size/4)) || isRed(img.At(size/2, size*7/4)) {
			t.Fatalf("%d 规格方向不对", size)
		}
	}
}

func TestSmallImageNotEnlargedAndAlphaUsesPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 50))
	img.Set(1, 1, color.NRGBA{0, 0, 0, 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	result, err := Generate(writeFile(t, "a.png", buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if result.Ext != "png" {
		t.Fatalf("透明图片的缩略图格式 = %s", result.Ext)
	}
	if b := decode(t, result.Thumbnails[360]).Bounds(); b.Dx() != 100 || b.Dy() != 50 {
		t.Fatalf("小图被放大：%v", b)
	}
}

func TestAnimatedGIFKeepsFrames(t *testing.T) {
	palette := color.Palette{color.Black, color.White}
	all := &gif.GIF{LoopCount: 0}
	for i := 0; i < 3; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, 800, 400), palette)
		frame.SetColorIndex(i, i, 1)
		all.Image = append(all.Image, frame)
		all.Delay = append(all.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, all); err != nil {
		t.Fatal(err)
	}
	result, err := Generate(writeFile(t, "a.gif", buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	out, err := gif.DecodeAll(bytes.NewReader(result.Thumbnails[360]))
	if err != nil || result.Ext != "gif" || len(out.Image) != 3 || out.Image[0].Bounds().Dy() != 360 {
		t.Fatalf("动图缩略图 ext = %s, err = %v, 帧数 = %d", result.Ext, err, len(out.Image))
	}
}

// encodeGIF 编码 frames 帧 w×h 的黑白 GIF，canvas 不为空时作为画布尺寸。
func encodeGIF(t *testing.T, frames, w, h int, canvas image.Config) []byte {
	t.Helper()
	palette := color.Palette{color.Black, color.White}
	all := &gif.GIF{Config: canvas}
	if canvas != (image.Config{}) {
		all.Config.ColorModel = palette
	}
	for i := 0; i < frames; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, w, h), palette)
		frame.SetColorIndex(i%w, 0, 1)
		all.Image = append(all.Image, frame)
		all.Delay = append(all.Delay, 10)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, all); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCountGIFFrames(t *testing.T) {
	for _, frames := range []int{1, 5} {
		count, err := countGIFFrames(bytes.NewReader(encodeGIF(t, frames, 16, 8, image.Config{})))
		if err != nil || count != frames {
			t.Fatalf("%d 帧 GIF 计帧 = %d, err = %v", frames, count, err)
		}
	}
	data := encodeGIF(t, 3, 16, 8, image.Config{})
	if _, err := countGIFFrames(bytes.NewReader(data[:len(data)-1])); err == nil {
		t.Fatal("缺少结束符的 GIF 应返回错误")
	}
	if _, err := countGIFFrames(bytes.NewReader([]byte("GIF89a"))); err == nil {
		t.Fatal("被截断的 GIF 应返回错误")
	}
}

func TestGIFOverAnimationWorkUsesFirstFrame(t *testing.T) {
	// 画布 10000×10000，6 帧：帧数 × 画布像素数 = 6 亿，超过 maxAnimationWork，只用第一帧生成静态缩略图。
	data := encodeGIF(t, 6, 16, 8, image.Config{Width: 10000, Height: 10000})
	result, err := Generate(writeFile(t, "many.gif", data))
	if err != nil {
		t.Fatal(err)
	}
	if result.Ext == "gif" || result.Width != 16 || result.Height != 8 {
		t.Fatalf("超过动图上限时结果 = %dx%d %s", result.Width, result.Height, result.Ext)
	}
}

func TestAVIF(t *testing.T) {
	var buf bytes.Buffer
	if err := avif.Encode(&buf, halves(64, 32)); err != nil {
		t.Fatal(err)
	}
	result, err := Generate(writeFile(t, "a.avif", buf.Bytes()))
	if err != nil || result.Width != 64 || result.Height != 32 || result.Ext != "jpg" {
		t.Fatalf("AVIF 结果 = %+v, err = %v", result, err)
	}
}

func TestRejectsNonImagesAndHugeImages(t *testing.T) {
	if _, err := Generate(writeFile(t, "a.txt", []byte("hello"))); !errors.Is(err, ErrNotImage) {
		t.Fatalf("文本 err = %v", err)
	}
	// 只构造声明 20000×6000 的 PNG 头，确认按头部尺寸拒绝，不会真正解码。
	var header bytes.Buffer
	header.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := []byte("IHDR")
	ihdr = binary.BigEndian.AppendUint32(ihdr, 20000)
	ihdr = binary.BigEndian.AppendUint32(ihdr, 6000)
	ihdr = append(ihdr, 8, 6, 0, 0, 0)
	_ = binary.Write(&header, binary.BigEndian, uint32(len(ihdr)-4))
	header.Write(ihdr)
	_ = binary.Write(&header, binary.BigEndian, crc32.ChecksumIEEE(ihdr))
	if _, err := Generate(writeFile(t, "huge.png", header.Bytes())); !errors.Is(err, ErrNotImage) {
		t.Fatalf("超大图片 err = %v", err)
	}
}
