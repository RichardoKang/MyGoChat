package util

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 将明文密码哈希，返回可直接存入数据库的字符串
func HashPassword(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// ComparePassword 将明文密码与数据库中的哈希比较，密码匹配返回 true
func ComparePassword(hashed string, plain string) bool {
	if hashed == "" || plain == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}
