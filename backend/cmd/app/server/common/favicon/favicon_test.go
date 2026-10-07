package favicon

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func pngIcon(t *testing.T, size int, c color.Color) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for i := range size * size {
		img.Set(i%size, i/size, c)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// icoWith32BitDIB 构造只含一张 16×16、32 位 DIB 的 ICO。
func icoWith32BitDIB(c color.NRGBA) []byte {
	const size = 16
	var dib bytes.Buffer
	header := make([]byte, 40)
	binary.LittleEndian.PutUint32(header[0:], 40)
	binary.LittleEndian.PutUint32(header[4:], size)
	binary.LittleEndian.PutUint32(header[8:], size*2)
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], 32)
	dib.Write(header)
	for range size * size {
		dib.Write([]byte{c.B, c.G, c.R, c.A})
	}
	dib.Write(make([]byte, 4*size)) // AND 掩码
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := make([]byte, 16)
	entry[0], entry[1] = size, size
	binary.LittleEndian.PutUint32(entry[8:], uint32(dib.Len()))
	binary.LittleEndian.PutUint32(entry[12:], 22)
	ico.Write(entry)
	ico.Write(dib.Bytes())
	return ico.Bytes()
}

// hugePNG 构造一张 1×1 的 PNG，再把头部声明的尺寸改为 width×height 并重算校验和。
// 图像数据仍只有 1 个像素，按声明尺寸解码会申请巨量内存。
func hugePNG(t *testing.T, width, height uint32) []byte {
	t.Helper()
	data := pngIcon(t, 1, color.NRGBA{255, 0, 0, 255})
	// 8 字节签名之后是 IHDR：4 字节长度、4 字节类型、13 字节数据、4 字节 CRC。
	binary.BigEndian.PutUint32(data[16:], width)
	binary.BigEndian.PutUint32(data[20:], height)
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	return data
}

// icoWithPNG 构造只含一张内嵌 PNG 的 ICO。
func icoWithPNG(entryPNG []byte) []byte {
	var ico bytes.Buffer
	ico.Write([]byte{0, 0, 1, 0, 1, 0})
	entry := make([]byte, 16)
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(entryPNG)))
	binary.LittleEndian.PutUint32(entry[12:], 22)
	ico.Write(entry)
	ico.Write(entryPNG)
	return ico.Bytes()
}

func center(t *testing.T, data []byte) color.NRGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != Size || b.Dy() != Size {
		t.Fatalf("图标尺寸 = %v", b)
	}
	return color.NRGBAModel.Convert(img.At(16, 16)).(color.NRGBA)
}

func TestFetchPrefersLargeDeclaredIcon(t *testing.T) {
	red := pngIcon(t, 64, color.NRGBA{255, 0, 0, 255})
	blue := pngIcon(t, 16, color.NRGBA{0, 0, 255, 255})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<html><head><link rel='icon' href='/small.png' sizes='16x16'><link href="/big.png" rel="shortcut icon" sizes="64x64"></head></html>`))
		case "/small.png":
			_, _ = w.Write(blue)
		case "/big.png":
			_, _ = w.Write(red)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data, err := Fetch(context.Background(), server.Client(), server.URL+"/some/page?x=1")
	if err != nil {
		t.Fatal(err)
	}
	if c := center(t, data); c.R != 255 || c.B != 0 {
		t.Fatalf("选中的图标颜色 = %+v", c)
	}
}

func TestFetchFallsBackToFaviconICO(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<link rel="icon" href="/missing.svg">`))
		case "/favicon.ico":
			_, _ = w.Write(icoWith32BitDIB(color.NRGBA{0, 200, 0, 255}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data, err := Fetch(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if c := center(t, data); c.G != 200 || c.A != 255 {
		t.Fatalf("ICO 图标颜色 = %+v", c)
	}
}

func TestDecodeRejectsHugeDeclaredSize(t *testing.T) {
	huge := hugePNG(t, 1<<30, 1<<29)
	if _, err := decodeIcon(huge); !errors.Is(err, errIconTooLarge) {
		t.Fatalf("PNG 声明超大尺寸时 err = %v", err)
	}
	if _, err := decodeIcon(icoWithPNG(huge)); !errors.Is(err, errIconTooLarge) {
		t.Fatalf("ICO 内嵌 PNG 声明超大尺寸时 err = %v", err)
	}
	if _, err := decodeIcon(icoWithPNG(pngIcon(t, 64, color.NRGBA{0, 0, 255, 255}))); err != nil {
		t.Fatalf("ICO 内嵌正常 PNG 时 err = %v", err)
	}
}

func TestFetchSkipsHugeIcon(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			_, _ = w.Write([]byte(`<link rel="icon" href="/huge.png" sizes="64x64">`))
		case "/huge.png":
			_, _ = w.Write(hugePNG(t, 1<<30, 1<<29))
		case "/favicon.ico":
			_, _ = w.Write(icoWith32BitDIB(color.NRGBA{0, 200, 0, 255}))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data, err := Fetch(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if c := center(t, data); c.G != 200 {
		t.Fatalf("超大图标应被跳过，实际颜色 = %+v", c)
	}
}

func TestFetchFailsWithoutIcon(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	if _, err := Fetch(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("没有图标时应返回错误")
	}
}
