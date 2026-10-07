package passwd

import (
	"errors"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// 密码长度按字节计算。上限 72 是 bcrypt 能参与计算的最大长度，更长的部分会被忽略，因此直接拒绝。
const (
	MinLength = 8
	MaxLength = 72
)

// Validate 检查密码长度和编码返回的错误，服务层按它确定错误码。
var (
	ErrCharset  = errors.New("密码包含无效字符")
	ErrTooShort = errors.New("密码至少 8 个字符")
	ErrTooLong  = errors.New("密码不能超过 72 字节")
)

// Validate 检查密码长度和编码。
func Validate(password string) error {
	if !utf8.ValidString(password) {
		return ErrCharset
	}
	if len(password) < MinLength {
		return ErrTooShort
	}
	if len(password) > MaxLength {
		return ErrTooLong
	}
	return nil
}

// Hash 生成密码哈希。
func Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Match 判断明文密码与哈希是否匹配。
func Match(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
