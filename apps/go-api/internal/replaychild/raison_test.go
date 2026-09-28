package replaychild

// raison_test.go — LA RAISON D'UN REFUS, DES DEUX COTES DU TUBE (lot J2.12, constat OPS-5,
// decision DT-5, 2026-09-26).
//
// L'ENFANT classe l'erreur de construction par `errors.Is` et en rend un JETON ; le PARENT
// traduit le jeton en l'erreur typee enveloppee. Aucun classement par texte : le texte d'une
// erreur n'est pas un contrat, et `ErrFilmNonFinalise` n'etait reconnu par personne.

import (
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

// refusTypes : les trois refus voulus du protocole.
var refusTypes = []error{
	replaybuild.ErrMapNotInCatalog,
	replaybuild.ErrUnknownFilmKey,
	filmcache.ErrFilmNonFinalise,
}

func TestEnfant_RaisonParRefusType(t *testing.T) {
	vus := map[string]bool{}
	for _, typee := range refusTypes {
		enveloppee := fmt.Errorf("construction m1 : etape x : %w", typee)
		jeton := raisonDuRefus(enveloppee)
		if jeton == "" {
			t.Errorf("%v enveloppee : aucun jeton — le refus serait compte en echec", typee)
			continue
		}
		if vus[jeton] {
			t.Errorf("jeton %q rendu pour deux refus differents", jeton)
		}
		vus[jeton] = true
	}
	if j := raisonDuRefus(errors.New("panne de decodage")); j != "" {
		t.Errorf("une panne ordinaire rend le jeton %q, attendu aucun", j)
	}
	// LE TEXTE NE SUFFIT PAS : une erreur qui CITE le message d'un refus sans l'envelopper
	// n'en est pas un.
	cite := errors.New("journal : " + replaybuild.ErrMapNotInCatalog.Error())
	if j := raisonDuRefus(cite); j != "" {
		t.Errorf("une erreur qui cite le texte d'un refus rend le jeton %q, attendu aucun", j)
	}
}

func TestSpawn_JetonVersErreurTypee(t *testing.T) {
	for _, typee := range refusTypes {
		jeton := raisonDuRefus(typee)
		if err := erreurDuRefus(jeton, "m1"); !errors.Is(err, typee) {
			t.Errorf("jeton %q : erreur %v, attendu une enveloppe de %v", jeton, err, typee)
		}
	}
	// UN REFUS SANS RAISON CONNUE N'EST AUCUN DES TROIS : l'enfant a rompu le protocole, et le
	// parent le compte en echec plutot que de deviner.
	for _, jeton := range []string{"", "jeton_inconnu"} {
		err := erreurDuRefus(jeton, "m1")
		if err == nil {
			t.Errorf("jeton %q : aucune erreur", jeton)
			continue
		}
		for _, typee := range refusTypes {
			if errors.Is(err, typee) {
				t.Errorf("jeton %q : pris pour %v", jeton, typee)
			}
		}
	}
}
