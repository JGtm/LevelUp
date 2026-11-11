package grammar

// equipement_porte_identite_test.go — LA RELECTURE DE L IDENTITE d une creation ti=37, sans film : un payload ecrit
// bit a bit, lu par le vrai chemin ([lecteurDIdentite.identite], le positionnement a `HeaderBit` + [woNewHeaderBits]
// et le deserialiseur [consumeDefaultStateTI37]). Les goldens lisent des identites deja cuites : sans ce test, un
// decalage de l en-tete ou un crochet pose sur un autre champ du bloc passerait.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

const (
	mppMotAttendu   = uint64(0xCAFE1234) // le mot inconditionnel du bloc : l identite
	mppMotDeTete    = uint64(0x1A5)      // premier champ du bloc, largeur `Lead`
	mppNomDeVariant = uint64(0xDEADBEEF) // « variant-name », derriere une porte
	mppNomDeQueue   = uint64(0xBAADF00D) // champ de la queue G3, derriere une porte
)

// payloadDeCreationTI37 : `rembourrage` bits avant le record, l en-tete NEW (champs non lus par la relecture), puis
// l etat par defaut d un ti=37 dont le bloc MPP porte quatre mots de 32 bits ou de `Lead` bits tous distincts.
func payloadDeCreationTI37(rembourrage uint64) []byte {
	mpp := profile.MPPParDefaut()
	return bitsDePayload(
		[2]uint64{0x15555 & (1<<rembourrage - 1), rembourrage},
		[2]uint64{0xABCDEF, woNewHeaderBits},
		[2]uint64{0, 1},                           // prefixe de version ti=37
		[2]uint64{0, 1},                           // prefixe de version ti=36
		[2]uint64{mppMotDeTete, uint64(mpp.Lead)}, // FUN_141fd72c0
		[2]uint64{mppMotAttendu, 32},              // FUN_14080d6f0 : l identite
		[2]uint64{0, 1}, [2]uint64{mppNomDeVariant, 32},
		[2]uint64{0, 1}, // porte R(18) fermee
		[2]uint64{0, 1}, // FUN_14080d524 : porte fermee
		[2]uint64{0, 2}, [2]uint64{0, uint64(mpp.Index)},
		[2]uint64{0, 3}, // aucune entree de liste
		[2]uint64{0, 1}, // FUN_14080d4d0 : porte fermee
		[2]uint64{1, 1}, [2]uint64{mppNomDeQueue, 32}, [2]uint64{0, 1}, [2]uint64{0, 14},
		[2]uint64{1, 1}, // reference d entite absente
		[2]uint64{0, 1}, // identifiant d aptitude absent
	)
}

// TestLecteurDIdentiteRendLeMotInconditionnel : quelle que soit la position du record dans le payload, l identite est
// le mot de 32 bits inconditionnel du bloc, et non l un des autres champs du bloc.
func TestLecteurDIdentiteRendLeMotInconditionnel(t *testing.T) {
	for _, rembourrage := range []uint64{0, 5, 13} {
		pay := payloadDeCreationTI37(rembourrage)
		l := nouveauLecteurDIdentite(nil)
		got, ok := l.identite(&lecture.Paquet{Payload: pay}, &FrameRecord{HeaderBit: int(rembourrage)})
		if !ok || uint64(got) != mppMotAttendu {
			t.Fatalf("rembourrage %d bits : identite %#x (lue %v), attendu %#x", rembourrage, got, ok, mppMotAttendu)
		}
	}
}
