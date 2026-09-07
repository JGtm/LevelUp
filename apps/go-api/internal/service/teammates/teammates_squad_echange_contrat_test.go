package teammates

import (
	"reflect"
	"testing"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// ─── LE CONTRAT ────────────────────────────────────────────────────────────────────────

// TestSquadEchange_AucunTauxNu — GARDE-RAIL de contrat.
//
// Le garde-rail `no_naked_rate_test` de `analysis/coordination` protege la FRONTIERE DE CE
// PAQUET-LA ; il ne dit rien de la forme sous laquelle un taux traverse le contrat HTTP.
// Ici : aucun `float64` de `domain.SquadEchange` ni de ses composants n'est un TAUX, hors
// ceux qui vivent dans un `domain.Couverture` (taux + brut + par match + N + echantillon
// faible). La seule exception est `ParMatch`, qui est une QUANTITE par match, sans borne
// haute — et elle est nommee ici pour qu'un futur `Taux float64` nu ne passe pas.
func TestSquadEchange_AucunTauxNu(t *testing.T) {
	// Champs float64 tolerés hors Couverture, par type. Ajouter une entrée ici exige de
	// verifier que le champ n'est PAS un quotient borne a 1.
	tolerés := map[string]bool{"SquadEchangeCell.ParMatch": true}

	verifie := func(nomType string, champs map[string]string) {
		for champ, kind := range champs {
			if kind == "float64" && !tolerés[nomType+"."+champ] {
				t.Errorf("%s.%s est un float64 nu : un taux sort dans domain.Couverture, "+
					"jamais seul (doctrine « jamais un taux seul »)", nomType, champ)
			}
		}
	}
	verifie("SquadEchange", champsFloatDe(domain.SquadEchange{
		Couverture: domain.Couverture{}, Habituel: domain.Couverture{},
	}))
	verifie("SquadEchangeCell", champsFloatDe(domain.SquadEchangeCell{}))
	verifie("SquadEchangeBucket", champsFloatDe(domain.SquadEchangeBucket{}))
	verifie("SquadEchangeJoueur", champsFloatDe(domain.SquadEchangeJoueur{}))
}

// champsFloatDe rend, pour chaque champ de PREMIER NIVEAU, le nom de son type. Les
// composants imbriques (domain.Couverture) ne sont pas deroules : c'est justement la
// forme sous laquelle un taux a le droit de voyager.
func champsFloatDe(v any) map[string]string {
	rt := reflect.TypeOf(v)
	out := make(map[string]string, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		out[f.Name] = f.Type.String()
	}
	return out
}

// TestSquadEchange_GardeRailNonVide : une sentinelle anti-vacuite. Un garde-rail qui
// n'inspecte plus rien passe au vert en silence, et c'est exactement ce qu'un renommage
// de type produirait ci-dessus.
func TestSquadEchange_GardeRailNonVide(t *testing.T) {
	champs := champsFloatDe(domain.SquadEchangeCell{})
	if champs["ParMatch"] != "float64" {
		t.Fatalf("SquadEchangeCell.ParMatch = %q : le garde-rail n'inspecte plus la bonne "+
			"structure", champs["ParMatch"])
	}
}

// TestBucketDelai_IntervalleServiContientLeDelai — PROPRIETE, correction G1.
//
// Le decoupage est derive de `bornesDelaiMs` et de rien d'autre. Ce test verifie sur
// une dizaine de delais poses a la main que le delai tombe TOUJOURS dans l'intervalle
// que le contrat publie pour lui : [DebutMs, FinMs[ partout, sauf l'intervalle qui
// FERME la fenetre, ou la borne haute est incluse (c'est elle qui decide du taux, et
// coordination.chercheVengeur la traite pareil).
//
// Sans lui, changer une borne dans `bornesDelaiMs` laisserait le rangement suivre
// l'ancienne — c'est exactement ce que faisait la version arithmetique (division par
// 1 000, bornage a 4) que cette correction remplace.
func TestBucketDelai_IntervalleServiContientLeDelai(t *testing.T) {
	delais := []int64{0, 1, 999, 1000, 3200, 3999, 4000, 4999, 5000, 5001, 6999, 7000, 40_000}
	fenetre := coordination.FenetreEchangeMs

	for _, d := range delais {
		i := bucketDelai(d)
		if i < 0 || i >= len(bornesDelaiMs) {
			t.Fatalf("delai %d ms : indice %d hors des %d intervalles", d, i, len(bornesDelaiMs))
		}
		debut := bornesDelaiMs[i]
		if d < debut {
			t.Errorf("delai %d ms range dans l'intervalle %d qui commence a %d ms", d, i, debut)
		}
		if i == len(bornesDelaiMs)-1 {
			continue // intervalle OUVERT : aucune borne haute a verifier
		}
		fin := bornesDelaiMs[i+1]
		fermeLaFenetre := fin == fenetre
		if (fermeLaFenetre && d > fin) || (!fermeLaFenetre && d >= fin) {
			t.Errorf("delai %d ms range dans l'intervalle [%d, %d%s : il n'y tombe pas",
				d, debut, fin, map[bool]string{true: "]", false: "["}[fermeLaFenetre])
		}
	}

	// Sentinelle anti-vacuite : le decoupage doit couvrir les deux populations.
	if bucketDelai(fenetre) == bucketDelai(fenetre+1) {
		t.Fatal("la borne de la fenetre ne separe plus rien : le garde ne garde rien")
	}
}
