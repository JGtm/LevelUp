package sync

import (
	"errors"
	"testing"
)

// ── isNotFoundErr ────────────────────────────────────────────────────────────

func TestIsNotFoundErr_Nil(t *testing.T) {
	if isNotFoundErr(nil) {
		t.Fatal("expected false for nil")
	}
}

func TestIsNotFoundErr_404(t *testing.T) {
	if !isNotFoundErr(errors.New("HTTP 404 Not Found")) {
		t.Fatal("expected true for HTTP 404")
	}
}

func TestIsNotFoundErr_410(t *testing.T) {
	if !isNotFoundErr(errors.New("HTTP 410 Gone")) {
		t.Fatal("expected true for HTTP 410")
	}
}

func TestIsNotFoundErr_RessourceAbsente(t *testing.T) {
	if !isNotFoundErr(errors.New("ressource absente")) {
		t.Fatal("expected true for ressource absente")
	}
}

func TestIsNotFoundErr_OtherError(t *testing.T) {
	if isNotFoundErr(errors.New("connection refused")) {
		t.Fatal("expected false for other error")
	}
}
