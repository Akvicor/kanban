package favicon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/png"
)

var errICO = errors.New("无法解析 ICO 图标")

// decodeICO 解码 ICO 文件中最大的一张图标。图标数据可以是 PNG，也可以是不带文件头的 DIB；
// DIB 支持 32 位（带透明通道）、24 位和 8 位调色板，24 位和 8 位使用 AND 掩码表示透明。
func decodeICO(data []byte) (image.Image, error) {
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:]) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil, errICO
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	best, bestWidth := -1, 0
	for i := 0; i < count; i++ {
		entry := 6 + i*16
		if entry+16 > len(data) {
			break
		}
		width := int(data[entry])
		if width == 0 {
			width = 256
		}
		if width > bestWidth {
			best, bestWidth = entry, width
		}
	}
	if best < 0 {
		return nil, errICO
	}
	size := int(binary.LittleEndian.Uint32(data[best+8:]))
	offset := int(binary.LittleEndian.Uint32(data[best+12:]))
	if offset < 0 || size <= 0 || offset+size > len(data) {
		return nil, errICO
	}
	entryData := data[offset : offset+size]
	if bytes.HasPrefix(entryData, []byte("\x89PNG")) {
		config, err := png.DecodeConfig(bytes.NewReader(entryData))
		if err != nil {
			return nil, errICO
		}
		if !withinSizeLimit(config) {
			return nil, errIconTooLarge
		}
		return png.Decode(bytes.NewReader(entryData))
	}
	return decodeDIB(entryData)
}

func decodeDIB(data []byte) (image.Image, error) {
	if len(data) < 40 {
		return nil, errICO
	}
	headerSize := int(binary.LittleEndian.Uint32(data[0:]))
	width := int(int32(binary.LittleEndian.Uint32(data[4:])))
	height := int(int32(binary.LittleEndian.Uint32(data[8:]))) / 2 // ICO 中的高度包含 AND 掩码
	bpp := int(binary.LittleEndian.Uint16(data[14:]))
	if headerSize < 40 || width <= 0 || height <= 0 || width > 256 || height > 256 {
		return nil, errICO
	}
	pos := headerSize
	var palette []color.NRGBA
	if bpp == 8 {
		colors := int(binary.LittleEndian.Uint32(data[32:]))
		if colors == 0 {
			colors = 256
		}
		if pos+colors*4 > len(data) {
			return nil, errICO
		}
		for i := 0; i < colors; i++ {
			p := data[pos+i*4:]
			palette = append(palette, color.NRGBA{R: p[2], G: p[1], B: p[0], A: 255})
		}
		pos += colors * 4
	} else if bpp != 24 && bpp != 32 {
		return nil, errICO
	}

	rowSize := (width*bpp + 31) / 32 * 4
	maskRowSize := (width + 31) / 32 * 4
	maskStart := pos + rowSize*height
	if maskStart > len(data) {
		return nil, errICO
	}
	hasMask := maskStart+maskRowSize*height <= len(data)
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		row := data[pos+(height-1-y)*rowSize:] // DIB 自下而上存储
		for x := 0; x < width; x++ {
			var c color.NRGBA
			switch bpp {
			case 32:
				c = color.NRGBA{R: row[x*4+2], G: row[x*4+1], B: row[x*4], A: row[x*4+3]}
			case 24:
				c = color.NRGBA{R: row[x*3+2], G: row[x*3+1], B: row[x*3], A: 255}
			case 8:
				index := int(row[x])
				if index >= len(palette) {
					return nil, errICO
				}
				c = palette[index]
			}
			if bpp != 32 && hasMask {
				mask := data[maskStart+(height-1-y)*maskRowSize+x/8]
				if mask&(0x80>>(x%8)) != 0 {
					c.A = 0
				}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img, nil
}
