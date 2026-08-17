package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was stored as plaintext")
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("VerifyPassword() rejected correct password")
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("VerifyPassword() accepted incorrect password")
	}
}

func TestPasswordHashesUseUniqueSalts(t *testing.T) {
	one, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	two, err := HashPassword("same password")
	if err != nil {
		t.Fatal(err)
	}
	if one == two {
		t.Fatal("identical passwords produced identical hashes")
	}
}
