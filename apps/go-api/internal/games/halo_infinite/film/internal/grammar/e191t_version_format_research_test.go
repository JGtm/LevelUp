//go:build research

package grammar

// e191t_version_format_research_test.go — LOT 1.9.1 ter, PAS 1 : LA VERSION DE FORMAT DU FILM.
//
// # CE QUE LE CHARGEUR DIT, ET QUI ETAIT MANQUANT
//
// Releve chez l ECRIVAIN le 2026-09-15 (Ghidra HTTP, `HaloInfinite.exe`, base `140000000`,
// LECTURE SEULE). Le serialiseur de `chunk_00` a DEUX faces et elles sont symetriques :
//
//	ecriture : FUN_14299b198 — W(0x20) base+0 · W(0x20) base+4 · W(0x659000) base+8 ·
//	           W(0xF60) base+0xCB208 · ...   (largeurs LITTERALES du build courant)
//	lecture  : FUN_14299ab50 — R(0x20) base+0 · R(0x20) base+4 ·
//	           R(FUN_141cfff30(base+4)) base+8       <- LARGEUR DU REGISTRE, PAR LA VERSION
//	           R(FUN_141cffe20(base+4)) base+0xCB208 <- LARGEUR DE LA TABLE, PAR LA VERSION
//
// Le SECOND u32 (`base+4`) n est donc pas « un second entier » : c est LA VERSION DE FORMAT DE
// `chunk_00`, et l executable courant porte la grammaire de TOUTES les versions anterieures dans
// deux `std::map` construites au demarrage (`FUN_140268ec0` / `FUN_140268f40`, litteraux relus
// sur le desassemblage) :
//
//	blocs de registre  DAT_1450fb368 : {13:47, 17:48, 18:49, 25:25}   defaut 50 (0x659000 bits)
//	entrees de table   DAT_1450fb378 : {13:110, 16:113, 17:114, 21:117, 22:118, 24:121, 25:122}
//	                                                                   defaut 123 (0xF60 bits)
//
// C est la forme MECANIQUE du fait utilisateur du 2026-09-16 : « le film est autoportant, la
// grammaire des anciens formats est dans l executable courant ». Aucun executable ancien n est
// necessaire, et aucun profil « par build » n est la bonne cle — la cle est CE u32.
//
// # CE QUE CET INSTRUMENT MESURE
//
// Par bobine : la version majeure (`base+0`), la VERSION DE FORMAT (`base+4`), le cardinal de la
// table par type mesure, le cardinal PREDIT par la map de l ecrivain, et la largeur MPP mesuree
// au lot 1.9.1 bis. Il repond a une seule question : la version de format SEPARE-T-ELLE les cinq
// bobines `8/3` des deux bobines `9/5` ?
//
// LECTURE SEULE, sans garde d environnement (les sept bobines sont versionnees).
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191tVersionDeFormat' -v -count=1

import (
	"encoding/binary"
	"path/filepath"
	"testing"
)

// e191tBlocsParVersion : `DAT_1450fb368`, relue sur `FUN_140268ec0` (litteraux du desassemblage).
var e191tBlocsParVersion = []struct{ Version, Valeur int }{
	{13, 47}, {17, 48}, {18, 49}, {25, 25},
}

// e191tTypesParVersion : `DAT_1450fb378`, relue sur `FUN_140268f40`.
var e191tTypesParVersion = []struct{ Version, Valeur int }{
	{13, 110}, {16, 113}, {17, 114}, {21, 117}, {22, 118}, {24, 121}, {25, 122},
}

// e191tBlocsDefaut / e191tTypesDefaut : les valeurs rendues hors table (`0x659000` bits de
// registre = 50 blocs, `0xF60` bits de table = 123 entrees) — celles du build courant.
const (
	e191tBlocsDefaut = 50
	e191tTypesDefaut = 123
)

// e191tPlusGrandeCleInferieure rejoue la recherche de l ecrivain : la valeur de la plus grande
// cle INFERIEURE OU EGALE a la version ; le defaut quand aucune cle ne l est.
func e191tPlusGrandeCleInferieure(table []struct{ Version, Valeur int }, version int, defaut int) int {
	out, meilleure := defaut, -1
	for _, e := range table {
		if e.Version <= version && e.Version > meilleure {
			out, meilleure = e.Valeur, e.Version
		}
	}
	return out
}

// TestE191tVersionDeFormat colle, bobine par bobine, la version de format et ce qu elle predit.
func TestE191tVersionDeFormat(t *testing.T) {
	t.Logf("######## LOT 1.9.1 ter — LA VERSION DE FORMAT DE chunk_00 (base+4) ########")
	t.Logf("  %-10s %-12s %6s %6s %8s %8s %6s", "bobine", "build", "maj", "FORMAT", "types", "predit", "MPP")
	formats := map[string]int{}
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		if len(d0) < 8 {
			t.Fatalf("%s : chunk_00 trop court", court)
		}
		maj := int(binary.LittleEndian.Uint32(d0[0:]))
		format := int(binary.LittleEndian.Uint32(d0[4:]))
		formats[court] = format
		id, err := ReadFilmIdentity(d0)
		types := -1
		build := "?"
		if err == nil {
			types, build = len(id.TypeVersions), id.Build
		}
		t.Logf("  %-10s %-12s %6d %6d %8d %8d %6s", court, build, maj, format, types,
			e191tPlusGrandeCleInferieure(e191tTypesParVersion, format, e191tTypesDefaut),
			e191cLibelleMPP(e191cLargeMPP[court]))
	}
	e191tVerdictSeparation(t, formats)
}

// e191tVerdictSeparation dit si la version de format separe exactement les deux groupes MPP.
func e191tVerdictSeparation(t *testing.T, formats map[string]int) {
	t.Helper()
	petit, grand := map[int]bool{}, map[int]bool{}
	for court, f := range formats {
		if e191cLargeMPP[court] {
			grand[f] = true
			continue
		}
		petit[f] = true
	}
	maxPetit, minGrand := -1, 1<<30
	for f := range petit {
		if f > maxPetit {
			maxPetit = f
		}
	}
	for f := range grand {
		if f < minGrand {
			minGrand = f
		}
	}
	t.Logf("")
	t.Logf("  versions 8/3 = %v · versions 9/5 = %v", e191tTrier(petit), e191tTrier(grand))
	if maxPetit < minGrand {
		t.Logf("  VERDICT : la version de format SEPARE les deux groupes — seuil dans ]%d, %d]",
			maxPetit, minGrand)
		return
	}
	t.Logf("  VERDICT : la version de format NE SEPARE PAS les deux groupes (recouvrement)")
}

// e191tTrier rend les versions d un groupe, triees.
func e191tTrier(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// e191tIndexPortes : les index de type que l ECRIVAIN interroge reellement. Releve exhaustif du
// 2026-09-15 : `FUN_1428e1c64` est le SEUL lecteur indexe de la table (`+0xCB208`, quatre sites
// d instruction sur 13,6 M), son seul appelant est `FUN_141102ed0` (film -> table du film, hors
// film -> table native `DAT_14474cd90`), et `FUN_141102ed0` a QUINZE appelants. Les index
// LITTERAUX de leurs sites d appel :
//
//	0x23 FUN_142f17500 · 0x24 FUN_14080c1f8 (x2) · 0x30 FUN_142f183f0 · 0x59 FUN_142ef6ec0,
//	FUN_142ef80fc · 0x5a FUN_142ef6fdc, FUN_142ef8138 (x2) · 0x5b FUN_142ef7044,
//	FUN_142ef8334 (x2) · 0x5d FUN_142ef8d94 · 0x61 FUN_142f16544 (x4) · 0x72 FUN_142ef87f0
//
// Trois sites de plus calculent l index (`[RDX+0x5a]`, `[RDX+0x5b]`, `[RBP+0x27]`) : ils ne sont
// pas litteraux et ne sont pas retenus ici.
var e191tIndexPortes = []int{0x23, 0x24, 0x30, 0x59, 0x5a, 0x5b, 0x5d, 0x61, 0x72}

// TestE191tIndexQuiVarient colle TOUT index dont la version n est pas la meme sur les sept
// bobines — critere plus large que la separation exacte en deux groupes, pour qu un index qui
// varie a l interieur d un groupe ne passe pas inapercu.
func TestE191tIndexQuiVarient(t *testing.T) {
	ordre := closureMiniFilms()
	vers := map[string][]uint32{}
	for _, court := range ordre {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		_, d0 := readChunk00(t, dir)
		id, err := ReadFilmIdentity(d0)
		if err == nil {
			vers[court] = id.TypeVersions
		}
	}
	mini := 1 << 30
	for _, v := range vers {
		if len(v) < mini {
			mini = len(v)
		}
	}
	t.Logf("######## INDEX DE TYPE QUI VARIENT ENTRE BOBINES (%d positions communes) ########", mini)
	varient := 0
	for i := 0; i < mini; i++ {
		vus := map[uint32]bool{}
		for _, court := range ordre {
			if v, ok := vers[court]; ok {
				vus[v[i]] = true
			}
		}
		if len(vus) > 1 {
			varient++
			t.Logf("  type[%3d] VARIE : %v", i, e191tTrierU32(vus))
		}
	}
	t.Logf("  BILAN : %d index varient sur %d communs", varient, mini)
	t.Logf("")
	t.Logf("######## LES NEUF INDEX QUE L ECRIVAIN INTERROGE, BOBINE PAR BOBINE ########")
	for _, i := range e191tIndexPortes {
		ligne := ""
		for _, court := range ordre {
			v, ok := vers[court]
			if !ok || i >= len(v) {
				ligne += "  --"
				continue
			}
			ligne += " " + e191tPad(int(v[i]))
		}
		t.Logf("  type[0x%02x] %s", i, ligne)
	}
	t.Logf("  (ordre : a521164d 60ae07c4 11de8353 111fa685 e5adf7b2 bcb6d393 fb1a1a72)")
}

// e191tTrierU32 rend les valeurs vues, triees.
func e191tTrierU32(m map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// e191tPad cadre un entier sur trois colonnes.
func e191tPad(v int) string {
	s := ""
	for _, c := range []byte{byte('0' + (v/100)%10), byte('0' + (v/10)%10), byte('0' + v%10)} {
		s += string(c)
	}
	return s
}

// e191tContradictions : les DEUX lignes des tables de l ECRIVAIN que la mesure contredit.
// Elles sont figees ici pour qu un futur passage ne les « corrige » pas en silence dans un sens
// ou dans l autre : la derivation structurelle de `ReadFilmIdentity` est validee sur les
// 1 351 chunks du cache, la table de l executable l est sur un seul executable.
var e191tContradictions = []struct {
	Format           int
	Quoi             string
	Ecrivain, Mesure int
	Temoin           string
}{
	{21, "entrees de table par type", 117, 116, "a521164d : chaine de build a 815 864, fin de registre a 815 368"},
	{25, "blocs de registre", 25, 49, "e5adf7b2 : chaine de build a 815 888 = 49 blocs + 122 entrees"},
}

// TestE191tContradictionsDeLEcrivain colle les deux contradictions, sans en trancher aucune.
func TestE191tContradictionsDeLEcrivain(t *testing.T) {
	t.Logf("######## LES DEUX LIGNES DE L ECRIVAIN QUE LA MESURE CONTREDIT ########")
	for _, c := range e191tContradictions {
		t.Logf("  format %2d · %-26s ecrivain=%-4d mesure=%-4d · %s",
			c.Format, c.Quoi, c.Ecrivain, c.Mesure, c.Temoin)
	}
	t.Logf("  Ces deux lignes ne sont PAS portees : le depot derive ces deux cardinaux des")
	t.Logf("  octets (ancrage sur la chaine de build), et cette derivation ferme a l unite sur")
	t.Logf("  les 1 351 chunk_00 du cache. Consignees au §4 du PLAN_DECODEUR_FILM.")
}

// TestE191tVersionDeFormatDuCache colle la distribution (version majeure, version de format) sur
// un corpus de films, et NOMME ceux qui n ont pas de section d identification.
//
// Garde `CHUNK00_FILMS` (liste separee par `;` de repertoires de film), comme les autres
// instruments de corpus du paquet. Mesure du 2026-09-15 sur les 1 351 `chunk_00` du cache :
//
//	(41, 27) x 1123 · (40, 27) x 146 · (40, 25) x 39 · (39, 24) x 26 · (37, 24) x 10 ·
//	(31, 20) x 3   · (33, 20) x 2   · (38, 24) x 1  · (33, 21) x 1
//
// LES CINQ FILMS DE FORMAT 20 SONT EXACTEMENT LES CINQ SANS SECTION D IDENTIFICATION
// (`03af54c3`, `13b00e35`, `47d20b5d`, `50247b26`, `a349fea8`) : la section apparait au format
// 21. Et les quatre autres classes reproduisent a l unite la partition que `film_identity.go`
// derivait par arithmetique (1 269 / 39 / 37 / 1).
func TestE191tVersionDeFormatDuCache(t *testing.T) {
	dirs := chunk00Films(t, "CHUNK00_FILMS")
	type cle struct{ Maj, Format int }
	compte := map[cle]int{}
	sansSection := map[int]int{}
	for _, dir := range dirs {
		_, d0 := readChunk00(t, dir)
		maj, okM := FilmMajorVersionFromHeader(d0)
		format, okF := FilmFormatVersionFromHeader(d0)
		if !okM || !okF {
			t.Errorf("%s : en-tete trop court", dir)
			continue
		}
		compte[cle{maj, format}]++
		if _, err := ReadFilmIdentity(d0); err != nil {
			sansSection[format]++
		}
	}
	t.Logf("######## (VERSION MAJEURE, VERSION DE FORMAT) SUR %d FILMS ########", len(dirs))
	for k, n := range compte {
		t.Logf("  (maj %2d, format %2d) x %d", k.Maj, k.Format, n)
	}
	t.Logf("  ---- films SANS section d identification, par format ----")
	for f, n := range sansSection {
		t.Logf("  format %2d : %d films sans section", f, n)
	}
}
