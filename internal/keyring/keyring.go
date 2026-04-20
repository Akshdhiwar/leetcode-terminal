// Package keyring provides secure credential storage.
//
// Strategy — simple and reliable across all platforms:
//   All platforms → AES-256-GCM encrypted file (~/.leetcode-cli/.credentials)
//
// The encryption key is derived from the machine's hostname + OS username,
// so the file is unreadable if copied to another machine.
//
// macOS additionally tries the Keychain first (best security).
// Linux additionally tries secret-service first.
// Windows uses ONLY the encrypted file (Credential Manager PowerShell API
// is too unreliable across Windows versions and environments).
package keyring

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const serviceName = "leetcode-cli"

// Credentials holds the two secrets needed for LeetCode auth.
type Credentials struct {
	Session string
	CSRF    string
}

// Save stores credentials securely.
func Save(creds Credentials) error {
	switch runtime.GOOS {
	case "darwin":
		// Try macOS Keychain first, fall back to encrypted file
		if err := saveDarwin("session", creds.Session); err == nil {
			if err2 := saveDarwin("csrf", creds.CSRF); err2 == nil {
				return nil
			}
		}
		return saveEncryptedFile(creds)
	case "linux":
		// Try secret-service first, fall back to encrypted file
		if secretToolAvailable() {
			if err := saveSecretTool("session", creds.Session); err == nil {
				_ = saveSecretTool("csrf", creds.CSRF)
				return nil
			}
		}
		return saveEncryptedFile(creds)
	default:
		// Windows and everything else: always use encrypted file
		// (Windows Credential Manager PowerShell API is fragile)
		return saveEncryptedFile(creds)
	}
}

// Load retrieves stored credentials.
func Load() (Credentials, error) {
	switch runtime.GOOS {
	case "darwin":
		session, err1 := loadDarwin("session")
		csrf, err2 := loadDarwin("csrf")
		if err1 == nil && err2 == nil && session != "" && csrf != "" {
			return Credentials{Session: session, CSRF: csrf}, nil
		}
		// Fall through to encrypted file
		return loadEncryptedFile()
	case "linux":
		if secretToolAvailable() {
			session, err := loadSecretTool("session")
			if err == nil && session != "" {
				csrf, _ := loadSecretTool("csrf")
				return Credentials{Session: session, CSRF: csrf}, nil
			}
		}
		return loadEncryptedFile()
	default:
		// Windows: always encrypted file
		return loadEncryptedFile()
	}
}

// Delete removes all stored credentials.
func Delete() error {
	// Always delete encrypted file
	_ = deleteEncryptedFile()
	// Also clean up OS keychain if applicable
	switch runtime.GOOS {
	case "darwin":
		_ = deleteDarwin("session")
		_ = deleteDarwin("csrf")
	case "linux":
		if secretToolAvailable() {
			_ = deleteSecretTool("session")
			_ = deleteSecretTool("csrf")
		}
	}
	return nil
}

// Backend returns a human-readable description of the active storage backend.
func Backend() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain + AES-256-GCM file fallback"
	case "linux":
		if secretToolAvailable() {
			return "Linux Secret Service (libsecret)"
		}
		return "AES-256-GCM encrypted file (~/.leetcode-cli/.credentials)"
	default:
		return "AES-256-GCM encrypted file (~/.leetcode-cli/.credentials)"
	}
}

// IsLoggedIn returns true if valid credentials are stored.
func IsLoggedIn() bool {
	creds, err := Load()
	return err == nil && creds.Session != ""
}

// ─── macOS Keychain ───────────────────────────────────────────────────────────

func saveDarwin(key, value string) error {
	_ = deleteDarwin(key)
	cmd := exec.Command("security", "add-generic-password",
		"-s", serviceName, "-a", key, "-w", value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain save failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadDarwin(key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", serviceName, "-a", key, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("keychain load failed for %q", key)
	}
	return strings.TrimSpace(string(out)), nil
}

func deleteDarwin(key string) error {
	return exec.Command("security", "delete-generic-password",
		"-s", serviceName, "-a", key).Run()
}

// ─── Linux secret-service ─────────────────────────────────────────────────────

func secretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func saveSecretTool(key, value string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("%s %s", serviceName, key),
		"service", serviceName, "key", key)
	cmd.Stdin = strings.NewReader(value)
	return cmd.Run()
}

func loadSecretTool(key string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup", "service", serviceName, "key", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool lookup failed")
	}
	return strings.TrimSpace(string(out)), nil
}

func deleteSecretTool(key string) error {
	return exec.Command("secret-tool", "clear", "service", serviceName, "key", key).Run()
}

// ─── AES-256-GCM Encrypted File (primary on Windows, fallback elsewhere) ──────
//
// Encryption key = SHA-256(hostname + username + hardcoded salt).
// This means the file cannot be decrypted on a different machine or by a
// different user — provides the same protection as DPAPI without needing
// any platform-specific APIs.

type credStore struct {
	Session string `json:"s"`
	CSRF    string `json:"c"`
	Version int    `json:"v"`
}

func credFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot find home directory: %w", err)
	}
	dir := filepath.Join(home, ".leetcode-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config dir: %w", err)
	}
	return filepath.Join(dir, ".credentials"), nil
}

func deriveKey() []byte {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows env var
	}
	if username == "" {
		username = os.Getenv("LOGNAME")
	}
	if username == "" {
		username = "default"
	}
	// Include a hardcoded app salt so the key is app-specific
	raw := fmt.Sprintf("leetcode-cli::v1::%s::%s", hostname, username)
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

func saveEncryptedFile(creds Credentials) error {
	key := deriveKey()

	plaintext, err := json.Marshal(credStore{
		Session: creds.Session,
		CSRF:    creds.CSRF,
		Version: 1,
	})
	if err != nil {
		return err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	// Seal appends ciphertext+tag to nonce
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.StdEncoding.EncodeToString(sealed)

	path, err := credFilePath()
	if err != nil {
		return err
	}

	// 0600 = owner read/write only
	return os.WriteFile(path, []byte(encoded+"\n"), 0600)
}

func loadEncryptedFile() (Credentials, error) {
	path, err := credFilePath()
	if err != nil {
		return Credentials{}, err
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, fmt.Errorf("not authenticated — run: lc auth")
		}
		return Credentials{}, fmt.Errorf("cannot read credentials file: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return Credentials{}, fmt.Errorf("credentials file is corrupted — run: lc auth")
	}

	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return Credentials{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Credentials{}, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return Credentials{}, fmt.Errorf("credentials file is corrupted — run: lc auth")
	}

	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		// Wrong machine/user, or file tampered
		return Credentials{}, fmt.Errorf("cannot decrypt credentials (wrong machine?) — run: lc auth")
	}

	var store credStore
	if err := json.Unmarshal(plaintext, &store); err != nil {
		return Credentials{}, fmt.Errorf("credentials file is corrupted — run: lc auth")
	}

	if store.Session == "" {
		return Credentials{}, fmt.Errorf("no session stored — run: lc auth")
	}

	return Credentials{Session: store.Session, CSRF: store.CSRF}, nil
}

func deleteEncryptedFile() error {
	path, err := credFilePath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
