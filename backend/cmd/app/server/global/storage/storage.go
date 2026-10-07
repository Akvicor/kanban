// Package storage 管理附件文件在磁盘上的存放：全局文件的原文件、缩略图，以及未完成的上传。
//
// 目录结构（根目录来自配置 storage.path）：
//
//	blobs/<sha256 前两位>/<sha256>                       原文件
//	thumbnails/<sha256 前两位>/<sha256>-<规格>.<扩展名>   缩略图
//	uploads/<会话 ID>                                   未完成的上传
//
// 这里只处理文件本身；哪些文件应当存在由服务层按数据库记录决定。
package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"kanban/cmd/config"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	blobsDir      = "blobs"
	thumbnailsDir = "thumbnails"
	uploadsDir    = "uploads"
)

// shaPattern 是合法的 sha256 文本：64 位小写十六进制。用它校验后再拼接路径，防止路径穿越。
var shaPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidSHA256 判断是否为 64 位小写十六进制的 sha256。
func ValidSHA256(sha string) bool {
	return shaPattern.MatchString(sha)
}

// Store 是一个存储根目录。
type Store struct {
	root string
}

var (
	current *Store
	lock    sync.RWMutex
)

// Load 按配置打开存储目录，目录不存在时创建。
func Load() error {
	store, err := Open(config.Global.Storage.Path)
	if err != nil {
		return err
	}
	lock.Lock()
	current = store
	lock.Unlock()
	return nil
}

// Get 返回当前存储，Load 成功前为 nil。
func Get() *Store {
	lock.RLock()
	defer lock.RUnlock()
	return current
}

// Set 替换当前存储，供测试使用临时目录。
func Set(store *Store) {
	lock.Lock()
	current = store
	lock.Unlock()
}

// Open 打开存储根目录并创建三个子目录。
func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("存储目录不能为空")
	}
	for _, dir := range []string{blobsDir, thumbnailsDir, uploadsDir} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o750); err != nil {
			return nil, fmt.Errorf("创建存储目录失败: %w", err)
		}
	}
	return &Store{root: root}, nil
}

// BlobPath 返回原文件路径。sha 必须先经过 ValidSHA256 校验。
func (s *Store) BlobPath(sha string) string {
	return filepath.Join(s.root, blobsDir, sha[:2], sha)
}

// ThumbnailPath 返回缩略图路径，size 为短边像素（360 或 720），ext 不带点。
func (s *Store) ThumbnailPath(sha string, size int, ext string) string {
	return filepath.Join(s.root, thumbnailsDir, sha[:2], fmt.Sprintf("%s-%d.%s", sha, size, ext))
}

// UploadPath 返回上传会话的临时文件路径。
func (s *Store) UploadPath(sessionID int64) string {
	return filepath.Join(s.root, uploadsDir, strconv.FormatInt(sessionID, 10))
}

// UploadSize 返回上传临时文件当前的大小，文件不存在时为 0。
func (s *Store) UploadSize(sessionID int64) (int64, error) {
	info, err := os.Stat(s.UploadPath(sessionID))
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// ErrOffsetMismatch 表示分片的偏移与已收到的字节数不一致。
var ErrOffsetMismatch = errors.New("分片偏移与已收到的大小不一致")

// AppendUpload 把一个分片追加到上传临时文件末尾。offset 必须等于文件当前大小，最多写入 limit 字节。
// 写入中途出错时把文件截回 offset，保证已收到的部分始终完整。返回追加后的大小。
func (s *Store) AppendUpload(sessionID, offset int64, data io.Reader, limit int64) (int64, error) {
	file, err := os.OpenFile(s.UploadPath(sessionID), os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if info.Size() != offset {
		return info.Size(), ErrOffsetMismatch
	}
	if _, err = file.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	written, err := io.Copy(file, io.LimitReader(data, limit))
	if err == nil {
		err = file.Sync()
	}
	if err != nil {
		_ = file.Truncate(offset)
		return offset, err
	}
	return offset + written, nil
}

// HashUpload 计算上传临时文件的 sha256。
func (s *Store) HashUpload(sessionID int64) (string, error) {
	file, err := os.Open(s.UploadPath(sessionID))
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// AdoptUpload 把完成的上传临时文件移动为 sha 对应的原文件（同一文件系统内的原子重命名）。
func (s *Store) AdoptUpload(sessionID int64, sha string) error {
	target := s.BlobPath(sha)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	return os.Rename(s.UploadPath(sessionID), target)
}

// RemoveUpload 删除上传临时文件，文件不存在时忽略。
func (s *Store) RemoveUpload(sessionID int64) error {
	return removeIfExists(s.UploadPath(sessionID))
}

// WriteThumbnail 原子写入一张缩略图：先写临时文件再重命名。
func (s *Store) WriteThumbnail(sha string, size int, ext string, data []byte) error {
	target := s.ThumbnailPath(sha, size, ext)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temp.Name(), target)
}

// RemoveBlob 删除原文件和它的全部缩略图，文件不存在时忽略。
func (s *Store) RemoveBlob(sha string) error {
	err := removeIfExists(s.BlobPath(sha))
	thumbnails, globErr := filepath.Glob(filepath.Join(s.root, thumbnailsDir, sha[:2], sha+"-*"))
	if globErr != nil {
		return errors.Join(err, globErr)
	}
	for _, path := range thumbnails {
		err = errors.Join(err, removeIfExists(path))
	}
	return err
}

// Entries 列出存储中的全部文件，用于启动时清理没有对应记录的孤立文件。
type Entries struct {
	// Blobs 是原文件的 sha256，Thumbnails 是有缩略图的 sha256（可能重复），Uploads 是上传会话 ID。
	Blobs      []string
	Thumbnails []string
	Uploads    []int64
}

// List 列出存储中的原文件、缩略图和上传临时文件。不认识的文件名忽略。
func (s *Store) List() (Entries, error) {
	var entries Entries
	err := walkFiles(filepath.Join(s.root, blobsDir), func(name string) {
		if ValidSHA256(name) {
			entries.Blobs = append(entries.Blobs, name)
		}
	})
	if err != nil {
		return entries, err
	}
	err = walkFiles(filepath.Join(s.root, thumbnailsDir), func(name string) {
		if sha, _, ok := strings.Cut(name, "-"); ok && ValidSHA256(sha) {
			entries.Thumbnails = append(entries.Thumbnails, sha)
		}
	})
	if err != nil {
		return entries, err
	}
	err = walkFiles(filepath.Join(s.root, uploadsDir), func(name string) {
		if id, err := strconv.ParseInt(name, 10, 64); err == nil {
			entries.Uploads = append(entries.Uploads, id)
		}
	})
	return entries, err
}

func walkFiles(root string, visit func(name string)) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type().IsRegular() {
			visit(entry.Name())
		}
		return nil
	})
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
