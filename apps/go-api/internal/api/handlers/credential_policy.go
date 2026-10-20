package handlers

import (
	"errors"

	"levelup/go-api/internal/platform/userstore"
)

// Codes des refus de la politique d'identifiants, lus par les écrans d'inscription et de
// changement de mot de passe (qui écrivent eux-mêmes le texte, FR et EN).
const (
	codeInvalidUsername  = "invalid_username"
	codePasswordTooShort = "password_too_short"
	codePasswordTooLong  = "password_too_long"
)

// credentialPolicyCode traduit un refus de la politique d'identifiants (forme du nom,
// longueur du mot de passe) en code stable ; ok est faux pour toute autre erreur.
func credentialPolicyCode(err error) (code string, ok bool) {
	switch {
	case errors.Is(err, userstore.ErrInvalidUsername):
		return codeInvalidUsername, true
	case errors.Is(err, userstore.ErrPasswordTooShort):
		return codePasswordTooShort, true
	case errors.Is(err, userstore.ErrPasswordTooLong):
		return codePasswordTooLong, true
	}
	return "", false
}
