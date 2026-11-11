package lead

import (
	"strings"
	"testing"
)

// 加密解密要能原样还原
func TestEncryptRoundTrip(t *testing.T) {
	cases := []string{
		"你好，鸡蛋批发多少钱？",
		"13800001111",
		"a long message with 中文 and symbols !@#$%^&*()",
		"",
	}

	for _, plain := range cases {
		cipher := EncryptContent(plain)

		if plain != "" && cipher == plain {
			t.Fatalf("明文没被加密: %s", plain)
		}

		got := DecryptContent(cipher)

		if got != plain {
			t.Fatalf("还原失败: got %q want %q", got, plain)
		}
	}
}

// 同一明文两次加密应不同（有随机 nonce），但都能还原
func TestEncryptRandomized(t *testing.T) {
	a := EncryptContent("测试消息")
	b := EncryptContent("测试消息")

	if a == b {
		t.Fatal("两次加密结果相同，缺少随机性")
	}

	if DecryptContent(a) != "测试消息" || DecryptContent(b) != "测试消息" {
		t.Fatal("随机化后无法还原")
	}
}

// 密文被篡改时不能返回原文明文
func TestDecryptTampered(t *testing.T) {
	cipher := EncryptContent("重要内容")

	if len(cipher) < 6 {
		t.Skip("密文太短")
	}

	tampered := cipher[:len(cipher)-2] + "AA"

	if got := DecryptContent(tampered); strings.Contains(got, "重要内容") {
		t.Fatalf("篡改后仍还原出明文: %q", got)
	}
}

// 兼容明文（历史上没加密的数据）
func TestDecryptPlaintextPassthrough(t *testing.T) {
	if got := DecryptContent("这不是密文"); got != "这不是密文" {
		t.Fatalf("明文应原样返回，got %q", got)
	}
}
