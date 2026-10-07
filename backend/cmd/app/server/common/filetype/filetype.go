// Package filetype 按文件开头的字节识别 MIME 类型，不采信文件名和客户端声明的类型。
package filetype

import (
	"bytes"
	"mime"
	"net/http"
)

// SniffLength 是识别类型需要读取的文件开头字节数。
const SniffLength = 512

// ftypBrands 是 ISO 基础媒体文件（ftyp 盒）主品牌到 MIME 类型的对应，补充标准库识别不了的格式。
var ftypBrands = map[string]string{
	"avif": "image/avif",
	"avis": "image/avif",
	"heic": "image/heic",
	"heix": "image/heic",
	"mif1": "image/heif",
	"msf1": "image/heif",
	"M4A ": "audio/mp4",
	"M4B ": "audio/mp4",
	"qt  ": "video/quicktime",
	"isom": "video/mp4",
	"iso2": "video/mp4",
	"mp41": "video/mp4",
	"mp42": "video/mp4",
	"avc1": "video/mp4",
	"M4V ": "video/mp4",
	"3gp4": "video/3gpp",
	"3gp5": "video/3gpp",
}

// Detect 返回文件的 MIME 类型（不带参数），header 是文件开头最多 SniffLength 字节。
// 认不出的二进制内容返回 application/octet-stream，纯文本返回 text/plain。
func Detect(header []byte) string {
	if len(header) >= 12 && string(header[4:8]) == "ftyp" {
		if kind, ok := ftypBrands[string(header[8:12])]; ok {
			return kind
		}
	}
	switch {
	case bytes.HasPrefix(header, []byte("II*\x00")), bytes.HasPrefix(header, []byte("MM\x00*")):
		return "image/tiff"
	case bytes.HasPrefix(header, []byte("fLaC")):
		return "audio/flac"
	}
	detected := http.DetectContentType(header)
	if mediaType, _, err := mime.ParseMediaType(detected); err == nil {
		return mediaType
	}
	return "application/octet-stream"
}
