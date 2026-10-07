package thumbnail

import (
	"bytes"
	"encoding/binary"
	"image"
	"io"
)

// exifReadLimit 是查找 EXIF 方向时最多读取的文件开头字节数。
const exifReadLimit = 256 << 10

// readOrientation 读取 JPEG（APP1 段）或 TIFF 文件中的 EXIF 方向，取值 1–8；没有或无法解析时返回 1。
func readOrientation(r io.Reader, format string) int {
	data, err := io.ReadAll(io.LimitReader(r, exifReadLimit))
	if err != nil {
		return 1
	}
	if format == "tiff" {
		return tiffOrientation(data)
	}
	return jpegOrientation(data)
}

// jpegOrientation 在 JPEG 的段中查找以 "Exif\0\0" 开头的 APP1 段，遇到图像数据（SOS）即停止。
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return 1
		}
		marker := data[pos+1]
		if marker == 0xDA || marker == 0xD9 {
			return 1
		}
		length := int(binary.BigEndian.Uint16(data[pos+2:]))
		if length < 2 || pos+2+length > len(data) {
			return 1
		}
		segment := data[pos+4 : pos+2+length]
		if marker == 0xE1 && bytes.HasPrefix(segment, []byte("Exif\x00\x00")) {
			return tiffOrientation(segment[6:])
		}
		pos += 2 + length
	}
	return 1
}

// tiffOrientation 解析 TIFF 结构的第一个 IFD，取方向标签（0x0112）。
func tiffOrientation(data []byte) int {
	if len(data) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(data[2:]) != 42 {
		return 1
	}
	offset := int(order.Uint32(data[4:]))
	if offset < 8 || offset+2 > len(data) {
		return 1
	}
	count := int(order.Uint16(data[offset:]))
	for i := 0; i < count; i++ {
		entry := offset + 2 + i*12
		if entry+12 > len(data) {
			return 1
		}
		if order.Uint16(data[entry:]) == 0x0112 && order.Uint16(data[entry+2:]) == 3 {
			if value := int(order.Uint16(data[entry+8:])); value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}

// orient 按 EXIF 方向把图片变换为正常显示的方向。方向 5–8 时宽高互换。
func orient(src *image.RGBA, orientation int) image.Image {
	if orientation <= 1 || orientation > 8 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dw, dh := w, h
	if orientation >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var sx, sy int
			switch orientation {
			case 2: // 水平翻转
				sx, sy = w-1-x, y
			case 3: // 旋转 180°
				sx, sy = w-1-x, h-1-y
			case 4: // 垂直翻转
				sx, sy = x, h-1-y
			case 5: // 沿主对角线翻转
				sx, sy = y, x
			case 6: // 顺时针旋转 90°
				sx, sy = y, h-1-x
			case 7: // 沿副对角线翻转
				sx, sy = w-1-y, h-1-x
			case 8: // 逆时针旋转 90°
				sx, sy = w-1-y, x
			}
			si := src.PixOffset(sx, sy)
			di := dst.PixOffset(x, y)
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}
