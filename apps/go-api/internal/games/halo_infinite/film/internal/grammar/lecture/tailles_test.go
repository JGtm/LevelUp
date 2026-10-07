package lecture_test

// tailles_test.go — LES TAILLES DES TYPES DE LA STRUCTURE SONT GELÉES (ADR 0037 IR-8).
//
// La structure se parcourt en flux, dans une arène réutilisée d'un paquet à l'autre. Quand un
// test ou un outil la matérialise, sa mémoire est le nombre de records du film fois leur taille,
// et un film en porte des centaines de milliers. Ces tailles sont donc un CONTRAT : les changer
// est une décision, prise en modifiant la valeur gelée ici, dans le commit qui change le type.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : ajouter un champ `uint8` à [lecture.Record] (la taille
// passe de 40 à 48 octets par l'alignement de `Masque`).

import (
	"testing"
	"unsafe"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
)

// TestTaillesDesTypesSontGelees : la taille de chaque type que la structure multiplie.
func TestTaillesDesTypesSontGelees(t *testing.T) {
	cas := []struct {
		nom          string
		vue, gelee   uintptr
		multipliePar string
	}{
		{"Record", unsafe.Sizeof(lecture.Record{}), 40, "record lu"},
		{"Composant", unsafe.Sizeof(lecture.Composant{}), 12, "occurrence de composant lue"},
		{"EntreeVueC", unsafe.Sizeof(lecture.EntreeVueC{}), 12, "tour de la vue C lu"},
		{"Etendue", unsafe.Sizeof(lecture.Etendue{}), 16, "lecture citée par un fait"},
		{"MessageDeKill", unsafe.Sizeof(lecture.MessageDeKill{}), 16, "message de kill lu"},
	}
	for _, c := range cas {
		if c.vue != c.gelee {
			t.Errorf("lecture.%s fait %d octets, gelé à %d : il est multiplié par chaque %s. "+
				"Changer sa taille est une décision de mémoire (ADR 0037 IR-8), prise en mettant "+
				"à jour la valeur gelée dans le commit qui change le type.",
				c.nom, c.vue, c.gelee, c.multipliePar)
		}
	}
}

// TestLesValeursZeroSontLesSentinelles : pour une énumération que la marche renseigne à chaque
// occurrence, la valeur zéro est la sentinelle « non renseigné » ; pour les autres, la valeur zéro
// a le sens que sa constante écrit. Réordonner les constantes déplacerait ce sens en silence.
func TestLesValeursZeroSontLesSentinelles(t *testing.T) {
	var (
		r lecture.Record
		c lecture.Composant
		p lecture.Paquet
	)
	cas := []struct {
		nom  string
		zero bool
	}{
		{"Genre", r.Genre == lecture.GenreNonRenseigne},
		{"Preuve", r.Preuve == lecture.PreuveNonRenseignee},
		{"Liaison", r.Liaison == lecture.LiaisonAucune},
		{"Etat", c.Etat == lecture.EtatNonRenseigne},
		{"ProvenanceLargeur", c.Prov == lecture.LargeurNonRenseignee},
		{"DebutDeVueB", p.Debut == lecture.DebutNonRenseigne},
		{"EtatDeVue", p.VueA.Etat == lecture.VueNonLue},
		{"SortieVueB", p.VueB.Sortie == lecture.SortieNonAtteinte},
		{"Verdict", p.Fermeture.Verdict == lecture.VerdictNonRendu},
		{"CauseDeQueue", p.Fermeture.Queue.Cause == lecture.CauseAucune},
	}
	for _, c := range cas {
		if !c.zero {
			t.Errorf("la valeur zéro de lecture.%s n'est plus sa sentinelle : une constante a été "+
				"réordonnée", c.nom)
		}
	}
}
