package lead

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// 内容加密：跟进记录里的聊天文本加密后入库
const encPrefix = "enc:"

var (
	contentKeyOnce sync.Once
	contentKey     []byte
)

// loadContentKey 读取或生成本机密钥（仅本机可用）
func loadContentKey() []byte {
	contentKeyOnce.Do(func() {
		paths := []string{
			os.Getenv("WX_CONTENT_KEY_FILE"),
			`E:\ai\data\.content_key`,
		}

		for _, path := range paths {
			if path == "" {
				continue
			}

			if data, err := os.ReadFile(path); err == nil {
				raw, err := hex.DecodeString(strings.TrimSpace(string(data)))
				if err == nil && len(raw) == 32 {
					contentKey = raw
					return
				}
			}
		}

		// 生成新密钥
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return
		}

		contentKey = key
		_ = os.MkdirAll(filepath.Dir(`E:\ai\data\.content_key`), 0o755)
		_ = os.WriteFile(`E:\ai\data\.content_key`, []byte(hex.EncodeToString(key)), 0o600)
	})

	return contentKey
}

// EncryptContent 加密（返回带前缀的 base64）
func EncryptContent(plain string) string {
	if plain == "" {
		return ""
	}

	key := loadContentKey()
	if len(key) != 32 {
		return plain
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return plain
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plain
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return plain
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)

	return encPrefix + base64.StdEncoding.EncodeToString(sealed)
}

// DecryptContent 解密（非加密内容原样返回）
func DecryptContent(value string) string {
	if !strings.HasPrefix(value, encPrefix) {
		return value
	}

	key := loadContentKey()
	if len(key) != 32 {
		return value
	}

	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return value
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return value
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return value
	}

	if len(raw) < gcm.NonceSize() {
		return value
	}

	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return value
	}

	return string(plain)
}
