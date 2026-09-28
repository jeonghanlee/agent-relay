package security

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	ErrIntegrityMismatch = errors.New("security: file SHA-256 checksum mismatch")
	ErrInvalidChecksum   = errors.New("security: expected checksum must be 64-character hex")
)

// ComputeFileSHA256 calculates the hex-encoded SHA-256 digest of the specified file.
func ComputeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("security: failed to open file for hashing: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("security: failed to hash file content: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyFileSHA256 checks that the target file matches expectedSHA256.
// Uses constant-time comparison to avoid side-channel leaks.
func VerifyFileSHA256(filePath string, expectedSHA256 string) error {
	normalizedExpected := strings.ToLower(strings.TrimSpace(expectedSHA256))
	if len(normalizedExpected) != 64 {
		return fmt.Errorf("%w: got length %d", ErrInvalidChecksum, len(normalizedExpected))
	}

	actualSHA256, err := ComputeFileSHA256(filePath)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(actualSHA256), []byte(normalizedExpected)) != 1 {
		return fmt.Errorf("%w: expected %s, got %s", ErrIntegrityMismatch, normalizedExpected, actualSHA256)
	}

	return nil
}
