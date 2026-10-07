// Package favicon 抓取网站的站点图标，转换为 32×32 PNG。
//
// 先读取网站首页，找页面中声明的图标（rel 含 icon），优先选声明尺寸不小于 32×32 的；
// 找不到或抓取失败时取 /favicon.ico。页面和图标各最多读取 1MB，图标宽、高各不超过 1024。
// 访问外部网址使用调用方提供的客户端，地址限制由客户端负责（见 safehttp）。
package favicon

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // 注册 GIF 解码
	_ "image/jpeg" // 注册 JPEG 解码
	"image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	_ "golang.org/x/image/bmp" // 注册 BMP 解码
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 WebP 解码
)

const (
	// Size 是输出图标的边长。
	Size = 32
	// maxBodySize 是页面和图标各自最多读取的字节数。
	maxBodySize = 1 << 20
	// maxIconSide 是候选图标宽、高的上限。解码前先读取头部声明的尺寸，超过时视为不可用：
	// 解码器按声明的尺寸分配内存，几十字节的图片就能声明出耗尽内存的尺寸。
	maxIconSide = 1024
	userAgent   = "Mozilla/5.0 (compatible; KanbanFavicon/1.0)"
)

// errIconTooLarge 表示图标头部声明的尺寸超过 maxIconSide。
var errIconTooLarge = errors.New("图标尺寸过大")

var (
	linkTag   = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	attribute = regexp.MustCompile(`(?is)([a-z-]+)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	sizeValue = regexp.MustCompile(`(?i)^(\d+)x(\d+)$`)
)

// Fetch 抓取 pageURL 所在网站的图标，返回 32×32 PNG 数据。
func Fetch(ctx context.Context, client *http.Client, pageURL string) ([]byte, error) {
	page, err := url.Parse(pageURL)
	if err != nil || (page.Scheme != "http" && page.Scheme != "https") || page.Host == "" {
		return nil, errors.New("网址不正确")
	}
	origin := &url.URL{Scheme: page.Scheme, Host: page.Host, Path: "/"}
	fallback := origin.ResolveReference(&url.URL{Path: "/favicon.ico"}).String()

	var candidates []string
	if body, finalURL, err := get(ctx, client, origin.String()); err == nil {
		if href := chooseIcon(string(body)); href != "" {
			if ref, err := url.Parse(href); err == nil {
				candidates = append(candidates, finalURL.ResolveReference(ref).String())
			}
		}
	}
	if len(candidates) == 0 || candidates[0] != fallback {
		candidates = append(candidates, fallback)
	}

	var lastErr error
	for _, candidate := range candidates {
		data, _, err := get(ctx, client, candidate)
		if err != nil {
			lastErr = err
			continue
		}
		icon, err := decodeIcon(data)
		if err != nil {
			lastErr = err
			continue
		}
		return encode(icon)
	}
	return nil, fmt.Errorf("没有可用的站点图标: %w", lastErr)
}

func get(ctx context.Context, client *http.Client, target string) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("响应状态 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	return body, resp.Request.URL, err
}

// chooseIcon 从页面中选出图标地址：rel 含 icon 的 link，优先选声明尺寸不小于 32×32 的，否则取第一个。
func chooseIcon(html string) string {
	first := ""
	for _, tag := range linkTag.FindAllString(html, -1) {
		attrs := map[string]string{}
		for _, m := range attribute.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(m[1])] = m[2] + m[3] + m[4]
		}
		href := strings.TrimSpace(attrs["href"])
		if href == "" || !hasIconRel(attrs["rel"]) {
			continue
		}
		if first == "" {
			first = href
		}
		if largeEnough(attrs["sizes"]) {
			return href
		}
	}
	return first
}

func hasIconRel(rel string) bool {
	for _, part := range strings.Fields(strings.ToLower(rel)) {
		if part == "icon" || part == "apple-touch-icon" {
			return true
		}
	}
	return false
}

func largeEnough(sizes string) bool {
	for _, size := range strings.Fields(sizes) {
		if strings.EqualFold(size, "any") {
			return true
		}
		if m := sizeValue.FindStringSubmatch(size); m != nil {
			w, _ := strconv.Atoi(m[1])
			h, _ := strconv.Atoi(m[2])
			if w >= Size && h >= Size {
				return true
			}
		}
	}
	return false
}

// decodeIcon 解码常见图片格式和 ICO。常见格式先读取尺寸，超过上限时返回 errIconTooLarge。
func decodeIcon(data []byte) (image.Image, error) {
	if config, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		if !withinSizeLimit(config) {
			return nil, errIconTooLarge
		}
		if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
			return img, nil
		}
	}
	return decodeICO(data)
}

// withinSizeLimit 判断图片头部声明的宽、高是否都不超过 maxIconSide。
func withinSizeLimit(config image.Config) bool {
	return config.Width <= maxIconSide && config.Height <= maxIconSide
}

// encode 把图标缩放为 32×32 并编码为 PNG。比 32 小的图标用最近邻放大，保持像素风格清晰。
func encode(icon image.Image) ([]byte, error) {
	dst := image.NewNRGBA(image.Rect(0, 0, Size, Size))
	scaler := draw.Interpolator(draw.CatmullRom)
	if b := icon.Bounds(); b.Dx() < Size || b.Dy() < Size {
		scaler = draw.NearestNeighbor
	}
	scaler.Scale(dst, dst.Bounds(), icon, icon.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
