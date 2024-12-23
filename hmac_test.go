package h2go

import (
	"testing"
)

func TestGenHMACSHA1(t *testing.T) {
	got := GenHMACSHA1("123456", "123456")
	if got != "74b55b6ab2b8e438ac810435e369e3047b3951d0" {
		t.Errorf("GenHMACSHA1() = %v, want %v", got, "74b55b6ab2b8e438ac810435e369e3047b3951d0")
	}
}

func TestVerifyHMACSHA1(t *testing.T) {
	got := VerifyHMACSHA1("123456", "123456", "74b55b6ab2b8e438ac810435e369e3047b3951d0")
	if got != true {
		t.Errorf("VerifyHMACSHA1() = %v, want %v", got, true)
	}
}

func TestVerifyHMACSHA12(t *testing.T) {

	got := VerifyHMACSHA1("123456", "123456", "74b55b6ab2b8e438ac810435e369e304")
	if got != false {
		t.Errorf("VerifyHMACSHA1() = %v, want %v", got, false)
	}
}
