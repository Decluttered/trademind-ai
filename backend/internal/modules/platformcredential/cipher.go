package platformcredential

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	AlgorithmAES256GCM = "AES-256-GCM"
	EnvelopeVersionV1  = 1
)

var (
	ErrKeyUnavailable      = errors.New("credential key unavailable")
	ErrCredentialUseDenied = errors.New("credential use denied")
	ErrCredentialConflict  = errors.New("credential revision conflict")
	ErrCredentialNotFound  = errors.New("credential not found")
	ErrOAuthStateInvalid   = errors.New("oauth state invalid")
	ErrRedirectNotAllowed  = errors.New("oauth redirect not allowed")
)

type KeyMaterial struct {
	Reference       string
	Key             []byte
	DevelopmentOnly bool
}

type KeyProvider interface {
	CurrentKey(context.Context) (KeyMaterial, error)
	ResolveKey(context.Context, string) (KeyMaterial, error)
	RotateKey(context.Context) (KeyMaterial, error)
}

type LocalKeyProvider struct {
	reference string
	key       []byte
}

func NewLocalKeyProvider(reference, raw string) (*LocalKeyProvider, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrKeyUnavailable
	}
	sum := sha256.Sum256([]byte(raw))
	return &LocalKeyProvider{reference: strings.TrimSpace(reference), key: sum[:]}, nil
}

func (p *LocalKeyProvider) CurrentKey(context.Context) (KeyMaterial, error) {
	if p == nil || len(p.key) != 32 || p.reference == "" {
		return KeyMaterial{}, ErrKeyUnavailable
	}
	return KeyMaterial{Reference: p.reference, Key: append([]byte(nil), p.key...), DevelopmentOnly: true}, nil
}

func (p *LocalKeyProvider) ResolveKey(ctx context.Context, reference string) (KeyMaterial, error) {
	if p == nil || strings.TrimSpace(reference) != p.reference {
		return KeyMaterial{}, ErrKeyUnavailable
	}
	return p.CurrentKey(ctx)
}

func (p *LocalKeyProvider) RotateKey(context.Context) (KeyMaterial, error) {
	return KeyMaterial{}, errors.New("local key rotation requires explicit configuration replacement")
}

type EncryptedCredential struct {
	Ciphertext   string
	Nonce        string
	Algorithm    string
	KeyReference string
	Version      int
}

type CredentialCipher interface {
	Encrypt(context.Context, []byte, []byte) (EncryptedCredential, error)
	Decrypt(context.Context, EncryptedCredential, []byte) ([]byte, error)
}

type AESGCMCredentialCipher struct{ Keys KeyProvider }

func (c AESGCMCredentialCipher) Encrypt(ctx context.Context, plaintext, aad []byte) (EncryptedCredential, error) {
	key, err := c.Keys.CurrentKey(ctx)
	if err != nil || len(key.Key) != 32 {
		return EncryptedCredential{}, ErrKeyUnavailable
	}
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return EncryptedCredential{}, fmt.Errorf("credential cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedCredential{}, fmt.Errorf("credential cipher: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedCredential{}, fmt.Errorf("credential nonce: %w", err)
	}
	sealed := gcm.Seal(nil, nonce, plaintext, aad)
	return EncryptedCredential{Ciphertext: base64.RawStdEncoding.EncodeToString(sealed), Nonce: base64.RawStdEncoding.EncodeToString(nonce), Algorithm: AlgorithmAES256GCM, KeyReference: key.Reference, Version: EnvelopeVersionV1}, nil
}

func (c AESGCMCredentialCipher) Decrypt(ctx context.Context, envelope EncryptedCredential, aad []byte) ([]byte, error) {
	if envelope.Algorithm != AlgorithmAES256GCM || envelope.Version != EnvelopeVersionV1 {
		return nil, ErrCredentialUseDenied
	}
	key, err := c.Keys.ResolveKey(ctx, envelope.KeyReference)
	if err != nil || len(key.Key) != 32 {
		return nil, ErrKeyUnavailable
	}
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, ErrCredentialUseDenied
	}
	sealed, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, ErrCredentialUseDenied
	}
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, ErrCredentialUseDenied
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != gcm.NonceSize() {
		return nil, ErrCredentialUseDenied
	}
	plain, err := gcm.Open(nil, nonce, sealed, aad)
	if err != nil {
		return nil, ErrCredentialUseDenied
	}
	return plain, nil
}

type RuntimeCredential struct {
	ClientID     string
	ClientSecret []byte
	Bearer       []byte
	Renewal      []byte
	ExpiresAt    int64
	Version      int
}

type secretPayload struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Bearer       string `json:"bearer"`
	Renewal      string `json:"renewal"`
}

func encodeSecretPayload(value RuntimeCredential) ([]byte, error) {
	return json.Marshal(secretPayload{ClientID: value.ClientID, ClientSecret: string(value.ClientSecret), Bearer: string(value.Bearer), Renewal: string(value.Renewal)})
}

func decodeSecretPayload(raw []byte, version int, expiresAt int64) (RuntimeCredential, error) {
	var payload secretPayload
	if err := json.Unmarshal(raw, &payload); err != nil || strings.TrimSpace(payload.Bearer) == "" {
		return RuntimeCredential{}, ErrCredentialUseDenied
	}
	return RuntimeCredential{ClientID: payload.ClientID, ClientSecret: []byte(payload.ClientSecret), Bearer: []byte(payload.Bearer), Renewal: []byte(payload.Renewal), Version: version, ExpiresAt: expiresAt}, nil
}
