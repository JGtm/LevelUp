package replaychild

// raison.go — LE VOCABULAIRE DES REFUS DE L'ENFANT DE CUISSON (lot J2.12, constat OPS-5, decision
// DT-5, 2026-09-26).
//
// # LE DEFAUT
//
// Le code de sortie ne dit que « ecarte ». L'enfant classait son erreur en cherchant du TEXTE
// (`strings.Contains(err.Error(), ...)`), ne reconnaissait pas `filmcache.ErrFilmNonFinalise`
// (compte en ECHEC), et le parent traduisait TOUT refus en `ErrMapNotInCatalog` : une cle de
// film inconnue se journalisait « carte hors catalogue ».
//
// # LA REGLE
//
// L'enfant classe par `errors.Is` et rend un JETON par la ligne de protocole de `filmproc`
// ([filmproc.EmitRaison]) ; le parent traduit le jeton en l'erreur typee ENVELOPPEE, que ses
// appelants classent a leur tour par `errors.Is`. Le vocabulaire vit ICI, des deux cotes du
// tube, en UNE table : `filmproc` ne connait que des jetons opaques.

import (
	"errors"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/replaybuild"
)

// refusVoulus : les refus VOULUS de la cuisson, jeton et erreur typee. L'ordre est celui du
// classement : le premier `errors.Is` qui mord l'emporte.
var refusVoulus = []struct {
	jeton string
	err   error
}{
	// Carte hors catalogue de bornes (Forge) : le parent le journalise en DEBUG.
	{"carte_hors_catalogue", replaybuild.ErrMapNotInCatalog},
	// Cle ecrite dans le film absente de la table de profil (lot 3.1.1, D-4 d'ADR 0034) : le
	// film est lisible, c'est le depot qui n'a pas encore sa ligne.
	{"cle_du_film_inconnue", replaybuild.ErrUnknownFilmKey},
	// Film au cache que la cuisson refuse comme NON FINALISE (lot L3) : il sera complet plus tard.
	{"film_non_finalise", filmcache.ErrFilmNonFinalise},
}

// raisonDuRefus rend le jeton de protocole d'un refus VOULU, ou "" quand `err` est un echec.
// Appelee par l'ENFANT.
func raisonDuRefus(err error) string {
	for _, r := range refusVoulus {
		if errors.Is(err, r.err) {
			return r.jeton
		}
	}
	return ""
}

// erreurDuRefus rend l'erreur typee d'un refus annonce par l'enfant, ENVELOPPEE. Appelee par le
// PARENT. Un jeton absent ou inconnu est une RUPTURE DU PROTOCOLE : l'erreur n'enveloppe alors
// aucun refus voulu, et l'appelant la compte en echec plutot que de deviner.
func erreurDuRefus(jeton string, matchID string) error {
	for _, r := range refusVoulus {
		if r.jeton == jeton {
			return fmt.Errorf("cuisson de %s ecartee par l'enfant (%s) : %w", matchID, jeton, r.err)
		}
	}
	return fmt.Errorf("cuisson de %s : l'enfant a refuse le film sans raison connue (jeton %q)",
		matchID, jeton)
}
