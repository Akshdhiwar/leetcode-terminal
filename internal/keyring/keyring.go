// Package keyring provides secure credential storage using the OS keychain
// where available, with an AES-256-GCM encrypted file fallback.
//
// Storage backend per platform:
//   Windows  → Windows Credential Manager via PowerShell (DPAPI-encrypted by OS)
//   macOS    → Keychain via `security` CLI  (Keychain-encrypted by OS)
//   Linux    → secret-service via `secret-tool` if available,
//               else AES-256-GCM encrypted file (~/.leetcode-cli/.credentials)
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

const (
	serviceName = "leetcode-cli"
	keySession  = "session"
	keyCSRF     = "csrf"
)

// Credentials holds the two secrets needed for LeetCode auth.
type Credentials struct {
	Session string
	CSRF    string
}

// Save stores credentials in the OS keychain (or encrypted file on Linux).
func Save(creds Credentials) error {
	switch runtime.GOOS {
	case "windows":
		if err := saveWindows(keySession, creds.Session); err != nil {
			// fallback to encrypted file if Credential Manager fails
			return saveEncryptedFile(creds)
		}
		return saveWindows(keyCSRF, creds.CSRF)
	case "darwin":
		if err := saveDarwin(keySession, creds.Session); err != nil {
			return err
		}
		return saveDarwin(keyCSRF, creds.CSRF)
	default:
		if secretToolAvailable() {
			if err := saveSecretTool(keySession, creds.Session); err == nil {
				_ = saveSecretTool(keyCSRF, creds.CSRF)
				return nil
			}
		}
		return saveEncryptedFile(creds)
	}
}

// Load retrieves credentials from the OS keychain (or encrypted file).
func Load() (Credentials, error) {
	switch runtime.GOOS {
	case "windows":
		session, err := loadWindows(keySession)
		if err != nil || session == "" {
			// fallback to encrypted file
			return loadEncryptedFile()
		}
		csrf, err := loadWindows(keyCSRF)
		if err != nil || csrf == "" {
			return loadEncryptedFile()
		}
		return Credentials{Session: session, CSRF: csrf}, nil
	case "darwin":
		session, err := loadDarwin(keySession)
		if err != nil {
			return Credentials{}, err
		}
		csrf, err := loadDarwin(keyCSRF)
		if err != nil {
			return Credentials{}, err
		}
		return Credentials{Session: session, CSRF: csrf}, nil
	default:
		if secretToolAvailable() {
			session, err := loadSecretTool(keySession)
			if err == nil && session != "" {
				csrf, _ := loadSecretTool(keyCSRF)
				return Credentials{Session: session, CSRF: csrf}, nil
			}
		}
		return loadEncryptedFile()
	}
}

// Delete removes stored credentials from all possible backends.
func Delete() error {
	switch runtime.GOOS {
	case "windows":
		_ = deleteWindows(keySession)
		_ = deleteWindows(keyCSRF)
	case "darwin":
		_ = deleteDarwin(keySession)
		_ = deleteDarwin(keyCSRF)
	default:
		if secretToolAvailable() {
			_ = deleteSecretTool(keySession)
			_ = deleteSecretTool(keyCSRF)
		}
	}
	// Always also delete encrypted file (covers fallback saves)
	_ = deleteEncryptedFile()
	return nil
}

// Backend returns a human-readable description of the storage backend in use.
func Backend() string {
	switch runtime.GOOS {
	case "windows":
		return "Windows Credential Manager (DPAPI encrypted)"
	case "darwin":
		return "macOS Keychain (Keychain encrypted)"
	default:
		if secretToolAvailable() {
			return "Linux Secret Service (libsecret)"
		}
		return "AES-256-GCM encrypted file (~/.leetcode-cli/.credentials)"
	}
}

// IsLoggedIn quickly checks if credentials are stored without loading them fully.
func IsLoggedIn() bool {
	creds, err := Load()
	return err == nil && creds.Session != ""
}

// ─── macOS Keychain ───────────────────────────────────────────────────────────

func saveDarwin(key, value string) error {
	_ = deleteDarwin(key) // remove old entry first to avoid duplicate error
	cmd := exec.Command("security", "add-generic-password",
		"-s", serviceName,
		"-a", key,
		"-w", value,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("keychain save failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadDarwin(key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", serviceName,
		"-a", key,
		"-w",
	)
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

// ─── Windows Credential Manager ───────────────────────────────────────────────
// cmdkey stores/retrieves generic credentials.
// Reading back uses PowerShell with the built-in CredentialManager WinAPI.

func saveWindows(key, value string) error {
	target := fmt.Sprintf("%s-%s", serviceName, key)
	// cmdkey /generic stores a credential; /pass is the password field
	cmd := exec.Command("cmdkey",
		fmt.Sprintf("/generic:%s", target),
		"/user:lc",
		fmt.Sprintf("/pass:%s", value),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmdkey save failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadWindows(key string) (string, error) {
	target := fmt.Sprintf("%s-%s", serviceName, key)

	// Use PowerShell + Windows CredentialManager API (no third-party module needed)
	// [System.Net.NetworkCredential] can unwrap the stored password from CredUI
	script := fmt.Sprintf(`
try {
    Add-Type -AssemblyName System.Security
    $sig = '[DllImport("advapi32.dll", EntryPoint = "CredReadW", CharSet = CharSet.Unicode, SetLastError = true)] public static extern bool CredRead(string target, int type, int flags, out IntPtr credential);'
    $credType = Add-Type -MemberDefinition $sig -Name "Creds" -Namespace "ADVAPI32" -PassThru -ErrorAction Stop
    $credPtr = [IntPtr]::Zero
    if ($credType::CredRead("%s", 1, 0, [ref]$credPtr)) {
        $cred = [System.Runtime.InteropServices.Marshal]::PtrToStructure($credPtr, [System.Type]::GetType("System.Object"))
    }
} catch {}

# Simpler approach: use cmdkey list and PowerShell credential vault
$output = cmdkey /list:"%s" 2>$null
if ($output -match "Target:") {
    # credential exists, read via .NET CredentialCache approach
}

# Most reliable: use the credential blob directly
Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
public class WinCred {
    [StructLayout(LayoutKind.Sequential, CharSet=CharSet.Unicode)]
    public struct CREDENTIAL {
        public int Flags; public int Type; public string TargetName;
        public string Comment; public System.Runtime.InteropServices.ComTypes.FILETIME LastWritten;
        public int CredentialBlobSize; public IntPtr CredentialBlob;
        public int Persist; public int AttributeCount; public IntPtr Attributes;
        public string TargetAlias; public string UserName;
    }
    [DllImport("advapi32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    public static extern bool CredRead(string target, uint type, uint flags, out IntPtr credential);
    [DllImport("advapi32.dll")] public static extern void CredFree(IntPtr credential);
    public static string ReadPassword(string target) {
        IntPtr ptr;
        if (!CredRead(target, 1, 0, out ptr)) return "";
        try {
            var cred = (CREDENTIAL)Marshal.PtrToStructure(ptr, typeof(CREDENTIAL));
            if (cred.CredentialBlobSize == 0) return "";
            return Encoding.Unicode.GetString(Marshal.PtrToStringUni(cred.CredentialBlob) != null ?
                System.Text.Encoding.Unicode.GetBytes(Marshal.PtrToStringUni(cred.CredentialBlob)) :
                new byte[0]);
        } finally { CredFree(ptr); }
    }
}
"@ -ErrorAction SilentlyContinue
if (([System.Management.Automation.PSTypeName]'WinCred').Type) {
    $pass = [WinCred]::ReadPassword("%s")
    if ($pass) { Write-Output $pass; exit 0 }
}
exit 1
`, target, target, target)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	result := strings.TrimSpace(string(out))
	if err == nil && result != "" {
		return result, nil
	}

	// Final fallback: encrypted file
	return "", fmt.Errorf("could not read credential %q from Windows Credential Manager", key)
}

func deleteWindows(key string) error {
	target := fmt.Sprintf("%s-%s", serviceName, key)
	return exec.Command("cmdkey", fmt.Sprintf("/delete:%s", target)).Run()
}

// ─── Linux secret-service (libsecret / GNOME Keyring) ────────────────────────

func secretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func saveSecretTool(key, value string) error {
	cmd := exec.Command("secret-tool", "store",
		"--label", fmt.Sprintf("leetcode-cli %s", key),
		"service", serviceName,
		"key", key,
	)
	cmd.Stdin = strings.NewReader(value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("secret-tool store failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadSecretTool(key string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup", "service", serviceName, "key", key)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret-tool lookup failed for %q", key)
	}
	return strings.TrimSpace(string(out)), nil
}

func deleteSecretTool(key string) error {
	return exec.Command("secret-tool", "clear", "service", serviceName, "key", key).Run()
}

// ─── AES-256-GCM Encrypted File (universal fallback) ─────────────────────────
// The encryption key is derived from machine-id + username so the file
// is unreadable if copied to another machine.

type encryptedStore struct {
	Session string `json:"session"`
	CSRF    string `json:"csrf"`
}

func credFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".leetcode-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, ".credentials"), nil
}

func deriveKey() ([]byte, error) {
	machineID := getMachineID()
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // Windows
	}
	if username == "" {
		username = "default"
	}
	raw := fmt.Sprintf("leetcode-cli::%s::%s", machineID, username)
	hash := sha256.Sum256([]byte(raw))
	return hash[:], nil
}

func getMachineID() string {
	// Linux
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}
	// macOS via ioreg would need exec — use hostname as fallback
	if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}
	// Windows — use ComputerName env
	if cn := os.Getenv("COMPUTERNAME"); cn != "" {
		return cn
	}
	hostname, _ := os.Hostname()
	return hostname
}

func saveEncryptedFile(creds Credentials) error {
	key, err := deriveKey()
	if err != nil {
		return err
	}

	plaintext, err := json.Marshal(encryptedStore{Session: creds.Session, CSRF: creds.CSRF})
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

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)

	path, err := credFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(encoded), 0600)
}

func loadEncryptedFile() (Credentials, error) {
	path, err := credFilePath()
	if err != nil {
		return Credentials{}, err
	}

	encoded, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, fmt.Errorf("no credentials found — run `lc auth`")
		}
		return Credentials{}, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return Credentials{}, fmt.Errorf("corrupted credentials file — run `lc auth` again")
	}

	key, err := deriveKey()
	if err != nil {
		return Credentials{}, err
	}

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
		return Credentials{}, fmt.Errorf("corrupted credentials file — run `lc auth` again")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return Credentials{}, fmt.Errorf("failed to decrypt credentials — run `lc auth` again")
	}

	var store encryptedStore
	if err := json.Unmarshal(plaintext, &store); err != nil {
		return Credentials{}, fmt.Errorf("corrupted credentials — run `lc auth` again")
	}

	return Credentials{Session: store.Session, CSRF: store.CSRF}, nil
}

func deleteEncryptedFile() error {
	path, err := credFilePath()
	if err != nil {
		return err
	}
	return os.Remove(path)
}
