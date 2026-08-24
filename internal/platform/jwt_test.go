package platform

import (
	"testing"
	"time"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	tok, err := GenerateAccessToken("s", time.Minute, 42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := ParseAccessToken("s", tok)
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("id = %d, want 42", id)
	}
}

func TestParseAccessTokenRejectsBadSecret(t *testing.T) {
	tok, _ := GenerateAccessToken("a", time.Minute, 1)
	if _, err := ParseAccessToken("b", tok); err == nil {
		t.Fatal("bad secret accepted")
	}
}
