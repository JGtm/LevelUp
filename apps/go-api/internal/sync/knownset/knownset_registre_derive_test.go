package knownset

// knownset_registre_derive_test.go — la vérification au registre en échec (dérive de type de
// match_registry.match_id). Fichier à part : sa DDL est volontairement fausse, et le marqueur
// d'exemption du ratchet archlint vaut pour tout le fichier.

import (
	"context"
	"errors"
	"testing"
)

// TestLoad_RegistreIllisibleALaVerificationEstFatal : la première requête (participants du xuid
// au registre) réussit, la vérification au registre des enrichissements hors participants
// échoue (dérive de type de match_registry.match_id : un identifiant texte n'y est pas
// convertible). L'erreur remonte en ErrSharedUnreadable — jamais un ensemble où les
// enrichissements non vérifiés seraient traités comme connus (l'ancienne union).
func TestLoad_RegistreIllisibleALaVerificationEstFatal(t *testing.T) {
	sharedDB := openMem(t)
	// match_registry-ddl: legacy — dérive de type volontaire : la requête IN (...) doit échouer seule.
	exec(t, sharedDB, `CREATE TABLE match_registry (match_id INTEGER PRIMARY KEY)`)
	exec(t, sharedDB, `CREATE TABLE match_participants (match_id INTEGER, xuid VARCHAR)`)
	exec(t, sharedDB, `INSERT INTO match_registry VALUES (1)`)
	exec(t, sharedDB, `INSERT INTO match_participants VALUES (1, ?)`, xuidJoueur)
	if got, err := queryIDs(context.Background(), sharedDB, sharedKnownSQL, xuidJoueur); err != nil || len(got) != 1 {
		t.Fatalf("pré-condition : la première requête doit réussir (got %v, err %v)", keys(got), err)
	}

	set, err := Load(context.Background(), newPlayerDB(t, "enrichi-hors-participants"), sharedDB, xuidJoueur)
	if !errors.Is(err, ErrSharedUnreadable) {
		t.Fatalf("err = %v, attendu ErrSharedUnreadable (vérification au registre en échec)", err)
	}
	if set.Known != nil {
		t.Errorf("connus = %v, attendu nil (aucun ensemble partiel)", keys(set.Known))
	}
}
