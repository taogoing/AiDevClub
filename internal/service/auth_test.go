package service

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	h, err := hashPassword("secret123")
	if err != nil {
		t.Fatal(err)
	}
	if h == "secret123" {
		t.Fatal("password not hashed")
	}
	if err := checkPassword(h, "secret123"); err != nil {
		t.Fatalf("check = %v, want nil", err)
	}
	if err := checkPassword(h, "wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
}
