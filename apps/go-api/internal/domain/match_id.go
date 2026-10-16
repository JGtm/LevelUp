// Package domain — match_id.go : forme d'un identifiant de match.
package domain

// MaxMatchIDLength borne la longueur d'un identifiant de match accepté d'une
// source externe (un UUID Halo en compte 36).
const MaxMatchIDLength = 64

// IsValidMatchID dit si id a la forme d'un identifiant de match Halo : 1 à
// MaxMatchIDLength caractères hexadécimaux ou tirets. Un identifiant venu d'une
// source externe (fichier importé, requête) qui ne la respecte pas ne doit être ni
// écrit ni transformé en chemin.
func IsValidMatchID(id string) bool {
	if id == "" || len(id) > MaxMatchIDLength {
		return false
	}
	for _, c := range id {
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex && c != '-' {
			return false
		}
	}
	return true
}
