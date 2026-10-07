package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestUploadAppendHashAndAdopt(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	size, err := store.AppendUpload(7, 0, strings.NewReader("hello "), 1024)
	if err != nil || size != 6 {
		t.Fatalf("第一片 size = %d, err = %v", size, err)
	}
	// 偏移不等于已收到的大小时拒绝，返回当前大小供客户端续传。
	if size, err = store.AppendUpload(7, 3, strings.NewReader("x"), 1024); !errors.Is(err, ErrOffsetMismatch) || size != 6 {
		t.Fatalf("错误偏移 size = %d, err = %v", size, err)
	}
	// 单片超过上限的部分不写入。
	if size, err = store.AppendUpload(7, 6, strings.NewReader("world!!!"), 5); err != nil || size != 11 {
		t.Fatalf("第二片 size = %d, err = %v", size, err)
	}

	sum := sha256.Sum256([]byte("hello world"))
	want := hex.EncodeToString(sum[:])
	if got, err := store.HashUpload(7); err != nil || got != want {
		t.Fatalf("HashUpload = %s, %v", got, err)
	}
	if err = store.AdoptUpload(7, want); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(store.BlobPath(want)); err != nil || string(data) != "hello world" {
		t.Fatalf("原文件 = %q, %v", data, err)
	}
	if err = store.WriteThumbnail(want, 360, "jpg", []byte("thumb")); err != nil {
		t.Fatal(err)
	}

	entries, err := store.List()
	if err != nil || !slices.Equal(entries.Blobs, []string{want}) || !slices.Equal(entries.Thumbnails, []string{want}) || len(entries.Uploads) != 0 {
		t.Fatalf("List = %+v, %v", entries, err)
	}
	if err = store.RemoveBlob(want); err != nil {
		t.Fatal(err)
	}
	if entries, _ = store.List(); len(entries.Blobs)+len(entries.Thumbnails) != 0 {
		t.Fatalf("删除后 List = %+v", entries)
	}
}

func TestValidSHA256(t *testing.T) {
	if !ValidSHA256(strings.Repeat("a", 64)) || ValidSHA256("../"+strings.Repeat("a", 61)) || ValidSHA256(strings.Repeat("A", 64)) {
		t.Fatal("ValidSHA256 判断错误")
	}
}
