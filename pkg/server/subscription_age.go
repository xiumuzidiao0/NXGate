package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

type AgeGenerateResponse struct {
	OK        bool   `json:"ok"`
	Type      string `json:"type"`
	SecretKey string `json:"secret_key"`
	PublicKey string `json:"public_key"`
	Error     string `json:"error,omitempty"`
}

type AgeDeriveRequest struct {
	SecretKey string `json:"secret_key"`
}

type AgeDeriveResponse struct {
	OK        bool   `json:"ok"`
	Type      string `json:"type"`
	PublicKey string `json:"public_key"`
	Error     string `json:"error,omitempty"`
}

// GenerateAgeKeyPair generates a new native X25519 or MLKEM768-X25519 key pair.
func GenerateAgeKeyPair(keyType string) (secretKey, publicKey, resolvedType string, err error) {
	norm := strings.ToLower(strings.TrimSpace(keyType))
	if norm == "mlkem768-x25519" || norm == "mlkem768x25519" || norm == "pq" || norm == "hybrid" {
		id, err := age.GenerateHybridIdentity()
		if err != nil {
			return "", "", "", fmt.Errorf("failed to generate hybrid identity: %w", err)
		}
		return id.String(), id.Recipient().String(), "MLKEM768-X25519", nil
	}

	// Default to native X25519
	id, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate X25519 identity: %w", err)
	}
	return id.String(), id.Recipient().String(), "X25519", nil
}

// DeriveAgePublicKey derives the public recipient key from an age private secret key.
func DeriveAgePublicKey(secretKey string) (publicKey, resolvedType string, err error) {
	trimmed := strings.TrimSpace(secretKey)
	if trimmed == "" {
		return "", "", fmt.Errorf("私钥内容不能为空")
	}

	if strings.HasPrefix(trimmed, "AGE-SECRET-KEY-PQ-") {
		id, err := age.ParseHybridIdentity(trimmed)
		if err != nil {
			return "", "", fmt.Errorf("解析 MLKEM768-X25519 私钥失败: %w", err)
		}
		return id.Recipient().String(), "MLKEM768-X25519", nil
	}

	if strings.HasPrefix(trimmed, "AGE-SECRET-KEY-1") {
		id, err := age.ParseX25519Identity(trimmed)
		if err != nil {
			return "", "", fmt.Errorf("解析 X25519 私钥失败: %w", err)
		}
		return id.Recipient().String(), "X25519", nil
	}

	// Fallback to ParseIdentities
	ids, err := age.ParseIdentities(strings.NewReader(trimmed))
	if err != nil || len(ids) == 0 {
		return "", "", fmt.Errorf("无法识别的 age 私钥格式，请确认以 AGE-SECRET-KEY- 开头")
	}

	switch v := ids[0].(type) {
	case *age.HybridIdentity:
		return v.Recipient().String(), "MLKEM768-X25519", nil
	case *age.X25519Identity:
		return v.Recipient().String(), "X25519", nil
	default:
		return "", "", fmt.Errorf("暂不支持此私钥类型的公钥推导")
	}
}

// EncryptWithAge encrypts plaintext with the given age recipient public key(s),
// formatted as standard ASCII Armor (PEM-like block).
func EncryptWithAge(plaintext []byte, recipientStr string) ([]byte, error) {
	trimmed := strings.TrimSpace(recipientStr)
	if trimmed == "" {
		return nil, fmt.Errorf("age 加密公钥为空")
	}

	var recipients []age.Recipient

	// If starts with age1pq, parse as HybridRecipient
	if strings.HasPrefix(trimmed, "age1pq") {
		r, err := age.ParseHybridRecipient(trimmed)
		if err != nil {
			return nil, fmt.Errorf("解析 MLKEM768-X25519 公钥失败: %w", err)
		}
		recipients = append(recipients, r)
	} else if strings.HasPrefix(trimmed, "age1") {
		r, err := age.ParseX25519Recipient(trimmed)
		if err != nil {
			return nil, fmt.Errorf("解析 X25519 公钥失败: %w", err)
		}
		recipients = append(recipients, r)
	} else {
		parsed, err := age.ParseRecipients(strings.NewReader(trimmed))
		if err != nil || len(parsed) == 0 {
			return nil, fmt.Errorf("无效的 age 公钥: %v", err)
		}
		recipients = parsed
	}

	var out bytes.Buffer
	armorWriter := armor.NewWriter(&out)
	encWriter, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return nil, fmt.Errorf("初始化 age 加密失败: %w", err)
	}

	if _, err := encWriter.Write(plaintext); err != nil {
		_ = encWriter.Close()
		_ = armorWriter.Close()
		return nil, fmt.Errorf("写入 age 加密流失败: %w", err)
	}

	if err := encWriter.Close(); err != nil {
		_ = armorWriter.Close()
		return nil, fmt.Errorf("完成 age 加密失败: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return nil, fmt.Errorf("关闭 age armor 流失败: %w", err)
	}

	return out.Bytes(), nil
}

// DecryptWithAge decrypts an armored or binary age ciphertext using the given secret key.
func DecryptWithAge(ciphertext []byte, secretKey string) ([]byte, error) {
	ids, err := age.ParseIdentities(strings.NewReader(strings.TrimSpace(secretKey)))
	if err != nil || len(ids) == 0 {
		return nil, fmt.Errorf("解析 age 私钥失败: %w", err)
	}

	var r io.Reader = bytes.NewReader(ciphertext)
	if bytes.HasPrefix(bytes.TrimSpace(ciphertext), []byte("-----BEGIN AGE ENCRYPTED FILE-----")) {
		r = armor.NewReader(r)
	}

	decReader, err := age.Decrypt(r, ids...)
	if err != nil {
		return nil, fmt.Errorf("age 解密失败: %w", err)
	}

	return io.ReadAll(decReader)
}

// HTTP API: POST /api/singbox/subscription/age/generate
func (s *Server) handleAgeGenerate(w http.ResponseWriter, r *http.Request) {
	keyType := r.URL.Query().Get("type")
	if keyType == "" {
		keyType = "x25519"
	}

	sec, pub, resType, err := GenerateAgeKeyPair(keyType)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(AgeGenerateResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(AgeGenerateResponse{
		OK:        true,
		Type:      resType,
		SecretKey: sec,
		PublicKey: pub,
	})
}

// HTTP API: POST /api/singbox/subscription/age/derive
func (s *Server) handleAgeDerive(w http.ResponseWriter, r *http.Request) {
	var req AgeDeriveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(AgeDeriveResponse{
			OK:    false,
			Error: "无效的请求格式",
		})
		return
	}

	pub, resType, err := DeriveAgePublicKey(req.SecretKey)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(AgeDeriveResponse{
			OK:    false,
			Error: err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(AgeDeriveResponse{
		OK:        true,
		Type:      resType,
		PublicKey: pub,
	})
}
