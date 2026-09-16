//go:build research

package grammar

// e191c_n2_oracle_research_test.go — LOT 1.9.1 bis, PAS 2 QUATER : `n2` COMME ORACLE DES
// LARGEURS DE L ETAT PAR DEFAUT, APPLIQUE AUX ARCHETYPES QUI ECHOUENT.
//
// # CE QUE LE PAS 2 QUATER A ETABLI
//
// `n1` se lit a 108 bits du debut du record, `n2` APRES l etat par defaut. Les deux sont des
// tailles de tampon, donc CONSTANTES par archetype et par build. Mesure : les archetypes qui
// FERMENT ont `n1` ET `n2` constants (ti=14 4/28, ti=17 4/432, ti=22 12/12, ti=29 1/256,
// ti=6 4/7896) ; ceux qui echouent ont `n1` constant et `n2` du BRUIT (ti=13 136/bruit,
// ti=37 100/bruit, ti=38 100/bruit). Le premier bit faux est donc DANS L ETAT PAR DEFAUT.
//
// # CE QUE CET INSTRUMENT CHERCHE
//
// L etat par defaut de ti=36, 37, 38, 39, 42 et 43 contient le BLOC MPP (`FUN_14080cfe8`), dont
// le portage garde DEUX largeurs en globales — `mppLeadBits` et `mppIndexBits`. Chez l ecrivain
// elles sont LITTERALES : R(9) (`141fd72de : ADD [RCX+0x2c],0x9`) et R(5) (inline). Si une
// autre paire rendait `n2` CONSTANT, ce serait la preuve que le bloc porte autre chose ; si
// AUCUNE ne le rend constant, le defaut n est pas dans ces deux largeurs et il faut chercher
// ailleurs dans le bloc.
//
// LE CRITERE : la part des records dont le `n2` prend la valeur MODALE. A 9/5 elle est basse
// (bruit) ; une paire juste la mettrait proche de 1.
//
// LECTURE SEULE, sans garde d environnement (bobines versionnees). Aucun composant n est
// deroule : seul l etat par defaut est joue, donc le balayage est bon marche.
//
//	go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestE191cOracleN2$' -v -count=1

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/source"
)

// e191cN2Max borne le balayage de chaque largeur MPP.
const e191cN2Max = 16

// TestE191cOracleN2 balaye les deux largeurs MPP et publie la part modale de `n2`.
func TestE191cOracleN2(t *testing.T) {
	t.Logf("######## PAS 2 QUATER — `n2` CONTRE LES LARGEURS DU BLOC MPP ########")
	t.Logf("  largeurs de l ecrivain : lead=9 (141fd72de), index=5 (inline FUN_14080cfe8)")
	parTI := map[int][]e191cAncre{}
	for _, court := range closureMiniFilms()[:2] {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", dir, err)
		}
		for _, p := range e191cPayloads(NewFilmContext(film)) {
			for _, b := range keyframeBornes(p) {
				parTI[b.TI] = append(parTI[b.TI], e191cAncre{Pay: p, Bit: b.Bit})
			}
		}
	}
	for _, ti := range []int{37, 38, 42} {
		e191cBalayerMPP(t, parTI[ti], ti)
	}
}

// e191cResultatMPP est une paire de largeurs et la part modale qu elle obtient.
type e191cResultatMPP struct {
	Lead, Index int
	Part        float64
	Modal       uint64
}

// e191cBalayerMPP balaye les deux largeurs et colle les dix meilleures paires.
func e191cBalayerMPP(t *testing.T, ancres []e191cAncre, ti int) {
	t.Helper()
	var res []e191cResultatMPP
	total := 0
	for l := 1; l <= e191cN2Max; l++ {
		for i := 1; i <= e191cN2Max; i++ {
			bal := contexteDInstrument()
			bal.Profil.MPP = MPPWidths{Lead: l, Index: i}
			part, modal, n := e191cN2Part(ancres, ti, bal)
			total = n
			res = append(res, e191cResultatMPP{Lead: l, Index: i, Part: part, Modal: modal})
		}
	}
	sort.Slice(res, func(a, b int) bool { return res[a].Part > res[b].Part })
	t.Logf("")
	t.Logf("  ==== ti=%d (%d records) — dix meilleures paires ====", ti, total)
	for k, r := range res {
		if k >= 10 {
			break
		}
		marque := ""
		if r.Lead == 9 && r.Index == 5 {
			marque = "  <- les largeurs de l ecrivain"
		}
		t.Logf("     lead=%-3d index=%-3d part modale=%.3f (n2=%d)%s", r.Lead, r.Index, r.Part, r.Modal, marque)
	}
	for _, r := range res {
		if r.Lead == 9 && r.Index == 5 {
			t.Logf("     RAPPEL lead=9 index=5 : part modale=%.3f (n2=%d)", r.Part, r.Modal)
		}
	}
}

// LE RESULTAT, ET IL BORNE LE DEFAUT A TROIS BITS DANS UNE FONCTION NOMMEE (2026-09-15).
//
// Balayage 16 x 16 sur deux bobines, part des records dont `n2` prend la valeur modale :
//
//	            lead=9 index=5 (l ecrivain)     meilleure paire
//	ti=37  1 187 records    0,204 (n2=0)        lead=8 index=3 -> 0,639 (n2=1396)
//	ti=38  3 402 records    0,053 (n2=0)        lead=8 index=3 -> 0,617 (n2=1764)
//	ti=42    951 records    0,059               lead=8 index=3 -> 0,732 (n2=1300)
//
// LA MEME PAIRE GAGNE SUR LES TROIS ARCHETYPES, et les `n2` modaux y deviennent des tailles de
// tampon plausibles (1 396, 1 764, 1 300) au lieu de 0. Ce n est pas du bruit : trois
// populations independantes ne designent pas la meme paire par hasard.
//
// CE QUE CELA NE VEUT PAS DIRE, ET C EST LA REGLE D13. `lead = 8` CONTREDIT l ecrivain, qui
// donne `R(9)` par un litteral (`141fd72de : ADD [RCX+0x2c],0x9`), et `index = 3` contredit le
// `R(5)` inline. La bonne lecture du resultat n est donc PAS « poser 8 et 3 » — ce serait
// accorder un decodeur a une mesure, exactement ce que le chantier interdit — mais :
//
//	LE BLOC MPP (`FUN_14080cfe8`) CONSOMME TROIS BITS DE TROP, et les deux largeurs du
//	balayage ne sont que les seules molettes disponibles pour les absorber.
//
// Le defaut passe donc de « quelque part dans 31 composants » a « trois bits dans une fonction
// nommee ». La part modale plafonne a 0,62-0,73 et non a 1 : le bloc porte encore un element
// variable au-dela de ces trois bits. C est la prochaine relecture chez l ecrivain.

// TestE191cOracleN2ParBuild rejoue le balayage BOBINE PAR BOBINE. Si la paire gagnante differe
// d un build a l autre, la grammaire du bloc MPP est VERSIONNEE et le depot en lit une seule
// pour les sept builds — c est le profil par build de l ADR (D-3). Si la meme paire gagne
// partout, la grammaire est commune et les trois bits sont ailleurs.
func TestE191cOracleN2ParBuild(t *testing.T) {
	t.Logf("######## PAS 2 QUINQUIES — LE BALAYAGE MPP, BOBINE PAR BOBINE ########")
	t.Logf("  %-10s %-5s %7s   %-18s   %-18s", "bobine", "ti", "records", "meilleure paire", "l ecrivain 9/5")
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "replay", "testdata", "minifilm_"+court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", dir, err)
		}
		parTI := map[int][]e191cAncre{}
		for _, p := range e191cPayloads(NewFilmContext(film)) {
			for _, b := range keyframeBornes(p) {
				parTI[b.TI] = append(parTI[b.TI], e191cAncre{Pay: p, Bit: b.Bit})
			}
		}
		_, d0 := readChunk00(t, dir)
		id, errID := ReadFilmIdentity(d0)
		v, okv := FilmMajorVersion(film)
		if errID != nil {
			id.Build = "(sans identification)"
		}
		t.Logf("  %-10s build=%-22s v=%-3v types=%d", court, id.Build, e191cVer(v, okv), len(id.TypeVersions))
		for _, ti := range []int{37, 38, 42} {
			e191cLigneParBuild(t, court, ti, parTI[ti], v, okv)
		}
	}
}

// e191cLigneParBuild colle une ligne : la meilleure paire de cette bobine et le score de 9/5.
func e191cLigneParBuild(t *testing.T, court string, ti int, ancres []e191cAncre, ver int, okv bool) {
	t.Helper()
	if len(ancres) == 0 {
		return
	}
	meilleur := e191cResultatMPP{}
	var ref e191cResultatMPP
	for l := 1; l <= e191cN2Max; l++ {
		for i := 1; i <= e191cN2Max; i++ {
			bal := contexteDInstrument()
			bal.Profil.MPP = MPPWidths{Lead: l, Index: i}
			part, modal, _ := e191cN2Part(ancres, ti, bal)
			r := e191cResultatMPP{Lead: l, Index: i, Part: part, Modal: modal}
			if part > meilleur.Part {
				meilleur = r
			}
			if l == 9 && i == 5 {
				ref = r
			}
		}
	}
	t.Logf("  %-10s v=%-3v ti=%-2d %7d   lead=%-2d index=%-2d %.3f   %.3f (n2=%d)",
		court, e191cVer(ver, okv), ti, len(ancres), meilleur.Lead, meilleur.Index, meilleur.Part,
		ref.Part, ref.Modal)
}

// e191cVer formate la version majeure du film, ou "?" quand l en-tete ne la porte pas.
func e191cVer(v int, ok bool) string {
	if !ok {
		return "?"
	}
	return fmt.Sprintf("%d", v)
}

// CORRIGE LE 2026-09-15 (lot 1.9.1 ter) : la grammaire n est PAS versionnee par build, elle
// l est par la VERSION DE FORMAT de `chunk_00` (`+4`) — c est cette valeur que le chargeur du
// jeu consulte (`FUN_14299ab50`, `FUN_1428e1c0c`), et elle separe les deux groupes aussi bien
// que le build tout en couvrant les cinq films sans section d identification. Le titre et le
// tableau ci-dessous restent le RELEVE du pas 2 quinquies, dont la mesure tient : c est son
// interpretation de la cle qui a ete corrigee. Cf. `filmdec/film_format_version.go`.
//
// LE RESULTAT DU PAS 2 QUINQUIES : LA GRAMMAIRE DU BLOC MPP EST VERSIONNEE PAR BUILD, ET LE
// PORTAGE EST JUSTE — POUR LES BUILDS RECENTS SEULEMENT (2026-09-15).
//
// Balayage 16 x 16 rejoue BOBINE PAR BOBINE, part des records dont `n2` prend la valeur modale :
//
//	bobine      build         types   meilleure paire        a 9/5 (l ecrivain)
//	a521164d    HI_1_4_1      116     lead=8 index=3  0,995   0,304
//	60ae07c4    HI_1_8_0      121     lead=8 index=3  0,988   0,522
//	11de8353    HI_1_9_0      121     lead=8 index=3  0,990   0,492
//	111fa685    HI_1_10_0     121     lead=8 index=3  0,993   0,432
//	e5adf7b2    HI_1_11_0     122     lead=8 index=3  0,996   0,472
//	bcb6d393    HI_1_12_0     123     lead=9 index=5  0,949   0,949
//	fb1a1a72    HI_1_13_0     123     lead=9 index=5  1,000   1,000
//
// (ti=37 ; ti=38 et ti=42 donnent la MEME coupure, cf. la sortie du test.)
//
// LA BASCULE EST A `HI_1_12_0`, ET LA VERSION MAJEURE DU FILM NE LA DONNE PAS : `e5adf7b2` et
// `bcb6d393` portent tous deux `v=40` et tombent de part et d autre. Ce qui les separe est le
// BUILD — et, dans `chunk_00`, le CARDINAL DE LA TABLE PAR TYPE : 123 d un cote, 116 a 122 de
// l autre. Ce cardinal est LISIBLE HORS LIGNE (`FilmIdentity.TypeVersions`).
//
// CONSEQUENCE, ET ELLE CORRIGE LA LECTURE DU PAS 2 QUATER : le bloc MPP ne consomme PAS trois
// bits de trop « dans l absolu ». Le portage 9/5 est EXACT sur les builds >= HI_1_12_0 — il y
// ferme 0,95 a 1,000, ce qui est le meilleur score du balayage entier. Ce qui manque est un
// PROFIL PAR BUILD pour les builds <= HI_1_11_0, ou le bloc en lit trois de moins.
//
// ET C EST LA QUE D13 RENCONTRE SA LIMITE, QU IL FAUT NOMMER : l executable ouvert dans Ghidra
// est UN SEUL BUILD, et c est un build recent — il dit `R(9)` (`141fd72de`) et `R(5)` inline,
// sans aucune branche de version dans `FUN_14080cfe8` (le seul `if` runtime du bloc,
// `DAT_145121140 == 1`, ne consomme AUCUN bit : c est une resolution d objet). La grammaire des
// builds anciens n est donc PAS relisible chez cet ecrivain-la. La version sur laquelle le jeu
// branche ailleurs (`FUN_1428e1c0c(&DAT_144c23178)`, qui gouverne le bit `DAT_144706104`) vient
// d une structure RUNTIME attachee au film charge (`*(param_1 + 0x108)`, ou `*(param_1 + 0x120)
// + 0x130` selon `FUN_1428e1e94`) : elle vient donc DU FILM, et le film la porte dans
// `chunk_00`. Lire le profil dans le film n est pas deviner — c est lire le film.
//
// CE QUE CELA CHANGE POUR LA CONVERSION MPP DEMANDEE : retirer la calibration SANS profil par
// build casserait les cinq builds anciens. `CalibrateMPPWidths` existe precisement parce que la
// variabilite avait ete CONSTATEE sans que sa cause soit trouvee : c est le profil manquant,
// ecrit en heuristique. La conversion juste n est donc pas « la grammaire remplace la
// calibration » mais « le PROFIL PAR BUILD remplace la calibration, la calibration devient le
// controle » — avec, conformement a l ADR 0034, un build inconnu qui rend une erreur typee.
