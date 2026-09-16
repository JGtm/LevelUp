package halo_infinite

// weapon_tiers_categories_guard_test.go — LA REGLE DES MODES A DEPARTS ALEATOIRES, EPROUVEE SUR
// LES FORMES REELLES DU REGISTRE.
//
// # POURQUOI CE GARDE-RAIL, ET POURQUOI IL A ETE REECRIT DEUX FOIS
//
// Version 1 : la liste declarait des PREFIXES de `pair_name`, compares a la partie gauche du
// « : ». Version 2 (prescrite par une revue) : des CATEGORIES resolues par la taxonomie du
// titre. LES DEUX ETAIENT FAUSSES sur les memes quatre formes, et c est mesure ici meme
// (TestWeaponTiers_LaTaxonomieNeSuffitPas) :
//
//	"Slayer:Arena Super Fiesta"   prefixe "Slayer"   categorie "Other"
//	"Slayer:Arena Fiesta"         prefixe "Slayer"   categorie "Other"
//	"BTB:Fiesta Slayer"           prefixe "BTB"      categorie "BTB"
//	"BTB:Fiesta CTF"              prefixe "BTB"      categorie "BTB"
//
// Ces formes sont les plus nombreuses du parc (417 matchs Super Fiesta au seul plateau de
// score) : sur elles, un niveau « arme de base » etait ecrit sur des equipements TIRES AU SORT,
// et sans la note qui l aurait avoue.
//
// La regle retenue est un JETON cherche comme un MOT dans le `pair_name` ENTIER. Ce fichier
// verrouille trois choses : les formes reelles, la coherence avec la taxonomie (tout mode dont
// la CATEGORIE est a departs aleatoires doit aussi etre reconnu), et le faux positif de
// prefixe.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// categoriesADepartsAleatoires : les categories de mode dont l equipement de debut de vie est
// TIRE AU SORT. Mesure du 2026-09-14 (76 artefacts) : en mode regulier les trois armes de
// depart pesent 94,4 % (Assassin) et 92,6 % (BTB) des equipements de premiere emission ; en
// Super Fiesta la distribution est PLATE sur 22 cles, entre 3,2 % et 7,3 %.
var categoriesADepartsAleatoires = []string{
	ModeCategoryFiesta, ModeCategorySuperFiesta, ModeCategoryHuskyRaid,
}

func reglementDuTitre(t *testing.T) *mappings.RegulationSet {
	t.Helper()
	reg, err := mappings.LoadRegulationFromFile(filepath.Join(
		"..", "..", "..", "..", "..", "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation.toml illisible : %v", err)
	}
	return reg
}

// TestWeaponTiers_LesFormesReellesDuRegistre — LE TEST QUI AURAIT ATTRAPE LES DEUX DEFAUTS.
//
// Chaque `pair_name` de ce tableau existe dans `regulation.toml` (tables de temps reglementaire
// et de cible de victoire) ou a ete releve au registre.
func TestWeaponTiers_LesFormesReellesDuRegistre(t *testing.T) {
	reg := reglementDuTitre(t)
	for _, cas := range []struct {
		pair string
		want bool
	}{
		// LES QUATRE FORMES QUE LES DEUX VERSIONS PRECEDENTES RATAIENT.
		{"Slayer:Arena Super Fiesta", true},
		{"Slayer:Arena Fiesta", true},
		{"BTB:Fiesta Slayer", true},
		{"BTB:Fiesta CTF", true},
		{"BTB:Fiesta Total Control", true},
		// Les formes directes.
		{"Super Fiesta:Slayer", true},
		{"Fiesta:Slayer", true},
		{"Slayer:Fiesta", true},
		{"CTF:Husky Raid", true},
		{"Husky Raid:CTF", true},
		{"Husky Raid:Super CTF", true},
		{"Super Husky Raid:CTF", true},
		{"Castle Wars:Slayer", true},
		// Suffixe de carte, sans separateur.
		{"Husky Raid on Streets", true},
		// Et les modes REGULIERS, qui doivent garder leur niveau « arme de base ».
		{"Arena:Slayer", false},
		{"CTF:Arena", false},
		{"CTF:Arena Neutral Flag", false},
		{"Slayer:Arena", false},
		{"Slayer:Arena Tactical", false},
		{"BTB:Total Control", false},
		{"Strongholds:Arena", false},
		{"Team Slayer:Arena", false},
		{"", false},
	} {
		if got := reg.HasRandomStarts(cas.pair); got != cas.want {
			t.Errorf("HasRandomStarts(%q) = %v, attendu %v", cas.pair, got, cas.want)
		}
	}
}

// TestWeaponTiers_PasDeFauxPositifDePrefixe — « comme un MOT », pas « commence par ».
func TestWeaponTiers_PasDeFauxPositifDePrefixe(t *testing.T) {
	reg := reglementDuTitre(t)
	for _, pair := range []string{"Fiestaval:Slayer", "Slayer:Fiestaval", "Huskyland:CTF"} {
		if reg.HasRandomStarts(pair) {
			t.Errorf("HasRandomStarts(%q) = true : le jeton doit etre borne par des frontieres "+
				"de mot, sinon tout mode qui commence par un jeton bascule", pair)
		}
	}
}

// TestWeaponTiers_CoherenceAvecLaTaxonomie — la regle couvre TOUT ce que la taxonomie sait déjà
// classer comme aleatoire.
//
// Elle en couvre STRICTEMENT PLUS (c est le point), mais jamais moins : un mode que la
// taxonomie range en Fiesta et que la regle raterait serait une regression silencieuse.
func TestWeaponTiers_CoherenceAvecLaTaxonomie(t *testing.T) {
	reg := reglementDuTitre(t)
	for _, cat := range categoriesADepartsAleatoires {
		for _, prefixe := range PairNamePrefixesForCategory(cat) {
			for _, pair := range []string{prefixe, prefixe + ":Slayer", prefixe + ":CTF"} {
				if !reg.HasRandomStarts(pair) {
					t.Errorf("la categorie %q (prefixe %q) est a departs aleatoires, mais "+
						"HasRandomStarts(%q) = false — ajouter le jeton manquant au TOML",
						cat, prefixe, pair)
				}
			}
		}
	}
}

// TestWeaponTiers_LaTaxonomieNeSuffitPas — LA MESURE QUI JUSTIFIE LA REGLE, figee.
//
// Elle documente un NEGATIF : la taxonomie de modes du titre, qui est la bonne source partout
// ailleurs, ne reconnait PAS ces quatre formes. Le jour ou elle les reconnaitrait, ce test
// rougirait — et la regle pourrait alors se simplifier en une comparaison de categories.
func TestWeaponTiers_LaTaxonomieNeSuffitPas(t *testing.T) {
	for _, cas := range []struct {
		pair, categorie string
	}{
		{"Slayer:Arena Super Fiesta", "Other"},
		{"Slayer:Arena Fiesta", "Other"},
		{"BTB:Fiesta Slayer", "BTB"},
		{"BTB:Fiesta CTF", "BTB"},
	} {
		if got := InferModeCategoryFromPairName(cas.pair); got != cas.categorie {
			t.Errorf("InferModeCategoryFromPairName(%q) = %q, la mesure du 2026-09-14 disait %q "+
				"— si la taxonomie reconnait desormais ces formes, la regle des jetons peut "+
				"redevenir une comparaison de categories", cas.pair, got, cas.categorie)
		}
	}
}
