package offlinejwt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

const AlgorithmEd25519 = "Ed25519"

type GeneratedKeyPair struct {
	Kid        string
	Algorithm  string
	PrivatePEM []byte
	PublicPEM  string
}

type TokenParams struct {
	Issuer      string
	Audience    string
	Subject     string
	Slug        string
	Fingerprint string
	ExpiresAt   time.Time
	IssuedAt    time.Time
}

func GenerateEd25519KeyPair() (GeneratedKeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return GeneratedKeyPair{}, fmt.Errorf("generate ed25519 key pair: %w", err)
	}

	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return GeneratedKeyPair{}, fmt.Errorf("marshal private key: %w", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return GeneratedKeyPair{}, fmt.Errorf("marshal public key: %w", err)
	}

	kidBytes := make([]byte, 12)
	if _, err := rand.Read(kidBytes); err != nil {
		return GeneratedKeyPair{}, fmt.Errorf("generate key id: %w", err)
	}

	return GeneratedKeyPair{
		Kid:       hex.EncodeToString(kidBytes),
		Algorithm: AlgorithmEd25519,
		PrivatePEM: pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateDER,
		}),
		PublicPEM: string(pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicDER,
		})),
	}, nil
}

func EncryptPrivateKey(privatePEM []byte, encryptionSecret string) (string, error) {
	if len(privatePEM) == 0 {
		return "", fmt.Errorf("private key is required")
	}

	aead, err := encryptionAEAD(encryptionSecret)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate encryption nonce: %w", err)
	}

	sealed := aead.Seal(nil, nonce, privatePEM, nil)
	blob := append(nonce, sealed...)
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func DecryptPrivateKey(encryptedPrivateKey, encryptionSecret string) ([]byte, error) {
	aead, err := encryptionAEAD(encryptionSecret)
	if err != nil {
		return nil, err
	}

	blob, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encryptedPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("decode encrypted private key: %w", err)
	}
	if len(blob) <= aead.NonceSize() {
		return nil, fmt.Errorf("encrypted private key is malformed")
	}

	nonce := blob[:aead.NonceSize()]
	ciphertext := blob[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt private key: %w", err)
	}

	return plain, nil
}

func SignEd25519JWT(encryptedPrivateKey, encryptionSecret, kid string, params TokenParams) (string, error) {
	privatePEM, err := DecryptPrivateKey(encryptedPrivateKey, encryptionSecret)
	if err != nil {
		return "", err
	}

	privateKey, err := parseEd25519PrivateKey(privatePEM)
	if err != nil {
		return "", err
	}

	now := params.IssuedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	expiresAt := params.ExpiresAt.UTC()
	if !expiresAt.After(now) {
		return "", fmt.Errorf("token expiration must be after issued time")
	}

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", fmt.Errorf("generate token id: %w", err)
	}

	header := map[string]any{
		"alg": "EdDSA",
		"kid": strings.TrimSpace(kid),
		"typ": "JWT",
	}
	claims := map[string]any{
		"iss":         strings.TrimSpace(params.Issuer),
		"sub":         strings.TrimSpace(params.Subject),
		"slug":        strings.TrimSpace(params.Slug),
		"fingerprint": strings.TrimSpace(params.Fingerprint),
		"iat":         now.Unix(),
		"nbf":         now.Unix(),
		"exp":         expiresAt.Unix(),
		"jti":         hex.EncodeToString(jtiBytes),
	}
	if aud := strings.TrimSpace(params.Audience); aud != "" {
		claims["aud"] = aud
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	signature := ed25519.Sign(privateKey, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func encryptionAEAD(secret string) (cipher.AEAD, error) {
	trimmed := strings.TrimSpace(secret)
	if len(trimmed) < 32 {
		return nil, fmt.Errorf("OFFLINE_SIGNING_ENCRYPTION_KEY must be at least 32 characters")
	}

	key := sha256.Sum256([]byte(trimmed))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("create encryption cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm cipher: %w", err)
	}

	return aead, nil
}

func parseEd25519PrivateKey(privatePEM []byte) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode(privatePEM)
	if block == nil {
		return nil, fmt.Errorf("private key PEM is malformed")
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not Ed25519")
	}

	return privateKey, nil
}
