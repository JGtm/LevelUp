package replay

// e191_composants_research_test.go — LOT 1.9.1 : LE VOCABULAIRE DU JEU, MESURE SUR LES OCTETS.
//
// # LA QUESTION, POSEE PAR L UTILISATEUR LE 2026-09-15
//
// « Comment c est appele dans le code du jeu ? » Le jeu n ecrit PAS un evenement « equipment
// drop » — il n en existe aucun (seul `weapon_drop`, type 46, existe, pour les armes). Ce qu il
// ecrit sur un objet d equipement, ce sont des COMPOSANTS d etat (table ECS de l archetype 37,
// `filmdec/testdata/ecs_table.tsv`) :
//
//	i10  object-parent-state-component     le PORTEUR de l objet
//	i11  object-dead-state-component       l objet est detruit
//	i18  item-at-rest-component            l objet est POSE, immobile
//	i20  equipment-deployed-component      l objet est DEPLOYE
//	i21  equipment-activated-component     l objet est ACTIF
//	i23  equipment-creator-component       qui l a cree
//
// SI `i20` EST ECRIT pour les panneaux de mur ET pour les appareils portes deployes au sol
// (capteur pose, champ pose...), et ABSENT des objets laches, alors LA grammaire de l origine
// est ce composant — et le 103, la mort ecrite, le `taken` du porteur et la fenetre de 200 ms
// deviennent tous des CONTROLES comptes. S il n est pas discriminant, la mesure le dit avec ses
// chiffres et la regle precedente tient.
//
// # CE QUE CETTE MESURE LIT, ET OU
//
// Le RECORD DE CREATION de l objet (`filmdec.EquipmentCreation`) porte son MASQUE DE COMPOSANTS
// (`Mask`, la liste des index presents) : c est l etat de l objet AU MOMENT DE LA POSE, et c est
// la question. La chaine de production le balaie deja
// ([filmdec.ScanFilmEquipmentCreations]) ; cette mesure ne fait que joindre ce masque aux poses
// PUBLIEES par la meme cuisson, clef `(slot, gen)` et instant.
//
// RESERVE ECRITE : ce masque est celui du record de CREATION. Un composant ECRIT PLUS TARD dans
// la vie de l objet (un capteur qu on deploie une seconde apres l avoir lache) n y figure pas.
// La table [8] le mesure a part, sur les records DELTA de la meme vie.
//
// LECTURE SEULE, skip par defaut. Memes gardes que `e191_origine_mesure_research_test.go`.
//
//	CGO_ENABLED=0 E191_ROOT=<depot>/data/cache/film_chunks \
//	  go test ./internal/games/halo_infinite/film/replay/ \
//	  -run '^TestE191Composants$' -count=1 -timeout 180m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// Les composants d etat de l archetype 37 que cette mesure regarde, avec le nom que LE JEU leur
// donne. L index est celui de la table ECS ; le nom est celui du code du jeu.
var e191Composants = []struct {
	i   int
	nom string
}{
	{10, "i10 object-parent-state"},
	{11, "i11 object-dead-state"},
	{18, "i18 item-at-rest"},
	{20, "i20 equipment-deployed"},
	{21, "i21 equipment-activated"},
	{23, "i23 equipment-creator"},
}

// e191CleCreation identifie le record de creation d une pose : la vie de l objet et son instant.
type e191CleCreation struct {
	Life filmdec.EquipmentLifeKey
	T0US uint64
}

func TestE191Composants(t *testing.T) {
	root := os.Getenv(e191RootEnv)
	if root == "" {
		t.Skipf("mesure 1.9.1 : definir %s (racine des chunks de film, lecture seule)", e191RootEnv)
	}
	parc := map[string]int{}
	parcFam := map[string]int{}
	films, hors := 0, 0
	for _, f := range e191FilmsDemandes() {
		n, m, ok := e191ComposantsDUnFilm(t, root, f)
		if !ok {
			hors++
			continue
		}
		films++
		for k, v := range n {
			parc[k] += v
		}
		for k, v := range m {
			parcFam[k] += v
		}
	}
	t.Logf("")
	t.Logf("######## PARC — %d films mesures, %d hors mesure ########", films, hors)
	t.Logf("  [7] composant ECS present au RECORD DE CREATION, par famille x origine publiee")
	e191LogTable(t, "     ", parcFam)
	t.Logf("  [7 bis] presence brute par composant (toutes poses appariees)")
	e191LogTable(t, "     ", parc)
}

// e191ComposantsDUnFilm decode un film et joint le masque de composants du record de creation
// aux poses que la production publie.
func e191ComposantsDUnFilm(t *testing.T, root string, f e191Film) (map[string]int, map[string]int, bool) {
	t.Helper()
	entry, err := (goldenBuild{Short8: f.Short8, Map: f.Carte}).mapQuant()
	if err != nil {
		t.Logf("film %s : carte %q hors catalogue (%v) — hors mesure", f.Short8, f.Carte, err)
		return nil, nil, false
	}
	dir := filepath.Join(root, f.Short8)
	g, err := decodeFilmInputsForEntry(f.Short8, dir, entry)
	if err != nil {
		t.Logf("film %s : balayage impossible (%v) — hors mesure", f.Short8, err)
		return nil, nil, false
	}
	cre, cst, ok := e191Creations(dir, entry, g)
	if !ok {
		t.Logf("film %s : creations ti=37 illisibles — hors mesure", f.Short8)
		return nil, nil, false
	}
	parMasque := map[e191CleCreation][]int{}
	for _, c := range cre {
		parMasque[e191CleCreation{filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}, c.TimestampUS}] = c.Mask
	}
	ctx := e191Contexte(g)
	ctx.familles = goldenCatalog(t).EquipmentFamilies
	portes := e191ObjetsPortes(t)
	spawns, _, err := e191Spawns(dir)
	if err != nil {
		t.Logf("film %s : evenements 103 illisibles (%v) — hors mesure", f.Short8, err)
		return nil, nil, false
	}
	designees := map[filmdec.EquipmentLifeKey][]uint64{}
	for _, e := range spawns {
		if e.SpawnedValid {
			designees[e.Spawned] = append(designees[e.Spawned], e.TimestampUS)
		}
	}

	brut, parFam := map[string]int{}, map[string]int{}
	apparies, orphelines := 0, 0
	for _, p := range g.Placements {
		q := e191UnePose(p, f.Short8, designees, portes, ctx)
		lu, _ := e191Cascade(q)
		masque, vu := parMasque[e191CleCreation{p.Life, p.T0US}]
		if !vu {
			orphelines++
			continue
		}
		apparies++
		nature := "deployable"
		if q.Portee {
			nature = "portee"
		}
		parFam[fmt.Sprintf("%-20s %-9s %-9s %s", q.Famille, nature, lu, e191Signature(masque))]++
		for _, c := range e191Composants {
			if e191MasqueContient(masque, c.i) {
				brut[c.nom]++
			}
		}
	}
	t.Logf("")
	t.Logf("######## FILM %s (%s) — %d poses, %d creations ti=37 (%d ancres, %d acceptees) ########",
		f.Short8, f.Build, len(g.Placements), len(cre), cst.Anchors, cst.Accepted)
	t.Logf("  jointure pose <-> record de creation : %d appariees, %d orphelines", apparies, orphelines)
	e191LogTable(t, "     ", parFam)
	return brut, parFam, true
}

// e191Creations balaie les records de CREATION ti=37 du film, aux memes largeurs MPP que la
// chaine de production vient de mesurer — sans elles, aucune identite ne se resout.
func e191Creations(dir string, e filmdec.MapQuantEntry, g *goldenInputs,
) ([]filmdec.EquipmentCreation, filmdec.EquipmentCreationStats, bool) {
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(e, g.Film, nil)()
	prev := filmdec.SetMPPWidths(g.PlacementStats.Calibration.Widths)
	defer filmdec.SetMPPWidths(prev)
	wr := e.Range()
	cre, st, err := filmdec.ScanFilmEquipmentCreations(dir, &wr)
	if err != nil {
		return nil, st, false
	}
	return cre, st, true
}

// e191MasqueContient dit si l index de composant figure au masque (liste croissante).
func e191MasqueContient(masque []int, i int) bool {
	k := sort.SearchInts(masque, i)
	return k < len(masque) && masque[k] == i
}

// e191Signature rend la signature COURTE des composants regardes : `i10+ i18- i20+ ...`.
func e191Signature(masque []int) string {
	out := ""
	for _, c := range e191Composants {
		marque := "-"
		if e191MasqueContient(masque, c.i) {
			marque = "+"
		}
		out += fmt.Sprintf("i%d%s ", c.i, marque)
	}
	return out
}
