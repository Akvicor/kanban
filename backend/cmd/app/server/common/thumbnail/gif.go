package thumbnail

import (
	"bufio"
	"errors"
	"image"
	"image/gif"
	"io"
)

// GIF 块结构中的引导字节。
const (
	gifExtensionIntroducer = 0x21
	gifImageSeparator      = 0x2C
	gifTrailer             = 0x3B
)

// errGIFStructure 表示 GIF 块结构不合法。
var errGIFStructure = errors.New("GIF 块结构不合法")

// generateGIF 生成 GIF 的缩略图。先按块结构计帧：帧数大于 1 且「帧数 × 画布像素数」不超过 maxAnimationWork 时
// 解码全部帧生成动图缩略图，否则只解码第一帧生成静态缩略图。file 位于文件开头。
func generateGIF(file io.ReadSeeker, config image.Config) (*Result, error) {
	frames, err := countGIFFrames(file)
	if err != nil || frames == 0 {
		return nil, ErrNotImage
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if frames > 1 && frames*config.Width*config.Height <= maxAnimationWork {
		all, err := gif.DecodeAll(file)
		if err != nil || len(all.Image) == 0 {
			return nil, ErrNotImage
		}
		return animated(all, config)
	}
	first, err := gif.Decode(file)
	if err != nil {
		return nil, ErrNotImage
	}
	return static(first, 1)
}

// countGIFFrames 只扫描 GIF 的块结构，返回图像帧数。
// 它跳过调色板、扩展块和图像数据子块，不做 LZW 解压，耗时与文件大小成正比且几乎不分配内存，
// 用于在解码之前判断动图的解码开销：高压缩比的数据可以在很小的文件里声明大量大尺寸帧。
// 结构不完整、缺少结束符或出现未知块时返回错误，与标准库解码全部帧时的要求一致。
func countGIFFrames(r io.Reader) (int, error) {
	br := bufio.NewReader(r)
	// 6 字节签名和版本，之后是 7 字节逻辑屏幕描述，第 11 字节是全局调色板标志。
	var header [13]byte
	if _, err := io.ReadFull(br, header[:]); err != nil {
		return 0, err
	}
	if string(header[:3]) != "GIF" {
		return 0, errGIFStructure
	}
	if err := skipColorTable(br, header[10]); err != nil {
		return 0, err
	}

	frames := 0
	for {
		introducer, err := br.ReadByte()
		if err != nil {
			return 0, err
		}
		switch introducer {
		case gifExtensionIntroducer:
			// 1 字节扩展标签，之后是数据子块。
			if _, err = br.Discard(1); err != nil {
				return 0, err
			}
			if err = skipSubBlocks(br); err != nil {
				return 0, err
			}
		case gifImageSeparator:
			// 9 字节图像描述（最后 1 字节是局部调色板标志），可选局部调色板，
			// 1 字节 LZW 最小码长，之后是图像数据子块。
			var descriptor [9]byte
			if _, err = io.ReadFull(br, descriptor[:]); err != nil {
				return 0, err
			}
			if err = skipColorTable(br, descriptor[8]); err != nil {
				return 0, err
			}
			if _, err = br.Discard(1); err != nil {
				return 0, err
			}
			if err = skipSubBlocks(br); err != nil {
				return 0, err
			}
			frames++
		case gifTrailer:
			return frames, nil
		default:
			return 0, errGIFStructure
		}
	}
}

// skipColorTable 按标志字节跳过调色板：最高位表示存在调色板，低 3 位 n 表示 2^(n+1) 种颜色，每种 3 字节。
func skipColorTable(br *bufio.Reader, flags byte) error {
	if flags&0x80 == 0 {
		return nil
	}
	_, err := br.Discard(3 << ((flags & 0x07) + 1))
	return err
}

// skipSubBlocks 跳过一串数据子块：每个子块以 1 字节长度开头，长度为 0 的子块表示结束。
func skipSubBlocks(br *bufio.Reader) error {
	for {
		size, err := br.ReadByte()
		if err != nil {
			return err
		}
		if size == 0 {
			return nil
		}
		if _, err = br.Discard(int(size)); err != nil {
			return err
		}
	}
}
