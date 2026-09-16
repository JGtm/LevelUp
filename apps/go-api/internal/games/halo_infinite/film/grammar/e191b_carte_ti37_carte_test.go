//go:build research

package grammar

// e191b_carte_ti37_carte_test.go — LOT 1.9.1 bis, PAS 1 : LE CONTROLE QUI TRANCHE LA CAUSE.
//
// # CE QUE LE PAS 1 A VU, ET LA QUESTION QUE CA POSE
//
// La mesure `TestE191bCarteTI37` etablit deux faits sur les 7 bobines par build :
//
//	AUCUN composant de ti=37 ne desynchronise (les 31 sont consommes) ;
//	les archetypes qui portent `object-position-component` ferment 0,85 % de leurs records
//	  (184/21 698), ceux qui ne le portent pas 44,29 % (12 654/28 573) — 52 fois plus.
//
// Le defaut n est donc PAS propre a l equipement : il frappe toute la famille « objet du
// monde ». Le premier suspect est nomme par le code lui-meme : `object-position-component`
// lit ses trois axes aux largeurs de la CARTE (`WorldObjectPrecision`, traverse.go:154), que
// `replay.BuildFromFilm` installe depuis le catalogue de bornes pour la duree d une cuisson —
// et que la mesure de fermeture n installe PAS. Elle mesure donc toutes les cartes aux
// largeurs de Cliffhanger (13/13/14, index de region 1 bit), le defaut du paquet.
//
// # CE QUE CE CONTROLE FAIT
//
// Il rejoue EXACTEMENT la meme fermeture, bobine par bobine, avec les largeurs que le
// CATALOGUE donne a la carte reellement jouee par cette bobine (`map_quant_bounds.json`,
// versionne, la meme entree que la production). Deux colonnes : avant / apres.
//
//	si la fermeture MONTE, la cause est le decoupage d i0 et la mesure de fermeture est a
//	  corriger AVANT toute relecture de grammaire d equipement ;
//	si elle NE BOUGE PAS, i0 est hors de cause et le defaut est ailleurs dans le prefixe.
//
// Le controle est un test PERMANENT (sans garde d environnement) : bobines et catalogue sont
// versionnes. Il restaure le descripteur global apres chaque bobine.
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191bFermetureAvecCarte$' -v -count=1

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// e191bCarteDeBobine : la carte REELLEMENT jouee par chaque bobine par build (table
// `replay/minifilm_builds_test.go`, colonne Map — recopiee ici parce que les deux paquets ne
// se voient pas ; l ecart se verrait au `Lookup`, qui echoue sur un nom inconnu).
var e191bCarteDeBobine = map[string]string{
	"a521164d": "Fragmentation Heavies",
	"60ae07c4": "Live Fire - Ranked",
	"11de8353": "Thunderhead",
	"111fa685": "Command",
	"e5adf7b2": "Fragmentation",
	"bcb6d393": "Cliffhanger",
	"fb1a1a72": "Banished Narrows",
}

func TestE191bFermetureAvecCarte(t *testing.T) {
	cat, err := LoadMapQuantCatalog(e191bCatalogue())
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	t.Logf("######## PAS 1 — CONTROLE : LA FERMETURE AUX LARGEURS DE LA CARTE JOUEE ########")
	t.Logf("  %-10s %-22s %-14s %14s %14s", "bobine", "carte", "largeurs", "ti=37 defaut", "ti=37 carte")
	var avantF, avantT, apresF, apresT int
	for _, court := range closureMiniFilms() {
		a, b, entry := e191bDeuxMesures(t, cat, court)
		t.Logf("  %-10s %-22s %-14s %10d/%-4d %10d/%-4d", court, e191bCarteDeBobine[court],
			e191bLargeurCarte(entry), a.Closed, a.Total, b.Closed, b.Total)
		avantF, avantT = avantF+a.Closed, avantT+a.Total
		apresF, apresT = apresF+b.Closed, apresT+b.Total
	}
	t.Logf("  TOTAL ti=37 : %d/%d au defaut Cliffhanger, %d/%d aux largeurs de la carte",
		avantF, avantT, apresF, apresT)
}

// e191bLargeurCarte rend la signature lisible du decoupage impose par le catalogue.
func e191bLargeurCarte(e MapQuantEntry) string {
	l := e.Layout()
	return sprintfLargeur(l.AxisW[0], l.AxisW[1], l.AxisW[2], e.EffectiveRegionIndexBits(), e.Region)
}

// sprintfLargeur formate « 13/13/14 r1@0 » : les trois axes, la largeur d index de region et
// la region attendue.
func sprintfLargeur(a, b, c, idx uint, region uint32) string {
	return e191bItoa(a) + "/" + e191bItoa(b) + "/" + e191bItoa(c) + " r" + e191bItoa(idx) + "@" + e191bItoa(uint(region))
}

// e191bItoa evite un import de strconv pour trois entiers positifs.
func e191bItoa(v uint) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

// e191bDeuxMesures mesure la fermeture ti=37 d une bobine deux fois : au descripteur par
// defaut du paquet, puis aux largeurs de la carte jouee. Le descripteur global est restaure.
func e191bDeuxMesures(t *testing.T, cat *MapQuantCatalog, court string) (a, b KeyframeClosureStat, e MapQuantEntry) {
	t.Helper()
	dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	e, err = cat.Lookup(e191bCarteDeBobine[court])
	if err != nil {
		t.Fatalf("carte %q de la bobine %s hors catalogue : %v", e191bCarteDeBobine[court], court, err)
	}
	a = e191bFermetureTI37(t, film, court, nil)
	b = e191bFermetureTI37(t, film, court, &e)
	return a, b, e
}

// e191bFermetureTI37 rend la fermeture de l archetype 37 pour un film charge.
func e191bFermetureTI37(t *testing.T, film *filmsource.Film, court string,
	carte *MapQuantEntry) KeyframeClosureStat {
	t.Helper()
	fc := NewFilmContext(film)
	if carte != nil {
		// LES LARGEURS DE LA CARTE, posees sur CE contexte (lot 2.3) : c est la seule
		// difference entre les deux mesures, et elle ne sort pas d ici.
		fc.PoserLargeursObjetDuMondeDepuisDecoupage(carte.Layout())
	}
	stats, err := KeyframeClosure(fc)
	if err != nil {
		t.Fatalf("KeyframeClosure %s : %v", court, err)
	}
	return stats[e191bTI]
}
