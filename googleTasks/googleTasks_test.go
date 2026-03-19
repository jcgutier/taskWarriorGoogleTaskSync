package googletasks

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestSaveTokenAndTokenFromFileRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	tokenPath := filepath.Join(tempDir, "token.json")
	expected := &oauth2.Token{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC),
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	defer os.Chdir(originalWD)

	saveToken(tokenPath, expected)

	actual, err := tokenFromFile(tokenPath)
	if err != nil {
		t.Fatalf("tokenFromFile returned error: %v", err)
	}

	if actual.AccessToken != expected.AccessToken {
		t.Fatalf("expected access token %q, got %q", expected.AccessToken, actual.AccessToken)
	}
	if actual.RefreshToken != expected.RefreshToken {
		t.Fatalf("expected refresh token %q, got %q", expected.RefreshToken, actual.RefreshToken)
	}
	if actual.TokenType != expected.TokenType {
		t.Fatalf("expected token type %q, got %q", expected.TokenType, actual.TokenType)
	}
	if !actual.Expiry.Equal(expected.Expiry) {
		t.Fatalf("expected expiry %v, got %v", expected.Expiry, actual.Expiry)
	}
}

func TestTokenFromFileReturnsErrorForMissingFile(t *testing.T) {
	_, err := tokenFromFile(filepath.Join(t.TempDir(), "missing-token.json"))
	if err == nil {
		t.Fatal("expected error for missing token file")
	}
}
