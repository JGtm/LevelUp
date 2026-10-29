package haloapi

import (
	"errors"
	"testing"
)

// TestNonRetryable : l'erreur enveloppée est reconnue comme définitive ET comme elle-même, son
// message ne change pas, `nil` reste `nil`.
func TestNonRetryable(t *testing.T) {
	base := errors.New("demo mode: external fetch skipped")
	err := NonRetryable(base)
	if !errors.Is(err, ErrDefinitif) {
		t.Errorf("errors.Is(err, ErrDefinitif) = false")
	}
	if !errors.Is(err, base) {
		t.Errorf("errors.Is(err, base) = false : l'erreur d'origine doit rester visible")
	}
	if err.Error() != base.Error() {
		t.Errorf("message = %q, attendu %q (inchangé)", err.Error(), base.Error())
	}
	if NonRetryable(nil) != nil {
		t.Errorf("NonRetryable(nil) doit rendre nil")
	}
	if errors.Is(base, ErrDefinitif) {
		t.Errorf("une erreur non enveloppée ne doit pas être définitive")
	}
}
