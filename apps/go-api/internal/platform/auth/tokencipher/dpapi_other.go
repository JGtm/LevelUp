//go:build !windows

package tokencipher

// dpapi_other.go : hors Windows, le coffre n'existe pas. Le profil desktop est Windows seul
// (ADR 0045 D-2) : le constructeur le dit, il ne dégrade pas en silence.

func newDPAPI() (Cipher, error) { return nil, ErrUnavailable }
