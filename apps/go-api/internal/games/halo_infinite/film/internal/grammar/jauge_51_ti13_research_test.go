//go:build research

package grammar

// jauge_51_ti13_research_test.go — LOT 5.1.6 : LA JAUGE DE RETOUR DU DRAPEAU, CHERCHEE DANS
// `ti=13` TAG 3.
//
// # LE VERDICT, MESURE LE 2026-09-18 : LA JAUGE EST LA, ET ELLE EST BINAIREMENT SEPARANTE
//
// LA JAUGE DE RETOUR DU DRAPEAU EST `ti=13 i1` TAG 3, mode A, voie DELTA. Un slot par drapeau,
// et la correspondance est TOTALE :
//
//	film        slot   drapeau   echantillons dans un lacher DE CE drapeau
//	bcb6d393    1490      0       111 / 111  = 100,0 %
//	fb1a1a72    1614      0       453 / 453  = 100,0 %
//	fb1a1a72    1619      1       320 / 320  = 100,0 %
//
// 884 echantillons, tous a l interieur d un lacher de leur propre drapeau, zero en dehors. Le
// slot 1495 de `bcb6d393`, lui, ne tombe dans les lachers qu a 15,8 % : ce n est pas une jauge de
// drapeau, et le contraste le dit.
//
// L ORACLE EST BINAIRE ET IL NE SOUFFRE AUCUNE EXCEPTION. Par lacher, selon CE QUI LE TERMINE :
//
//	retour automatique (`home`)        13 / 13 atteignent le plein (>= 0,99)
//	repris par un joueur avant la fin    0 / 14 atteignent le plein
//
// Separation totale sur 27 lachers instrumentes, trois jauges, deux films. Exemple lisible
// (`bcb6d393`, lacher 83604..99104) : la jauge part de 0,0208 a 83882 ms, monte SANS REDESCENDRE
// sur 45 echantillons — 0,05 · 0,13 · 0,25 · 0,50 · 0,76 · 0,98 — et vaut EXACTEMENT 1,0000 a
// 99298 ms, quand le calque du drapeau le rend a sa base a 99204 ms.
//
// L ECHELLE EST 0 -> 1, PAS 0 -> 100 : [-100, +100] est la plage de SERIALISATION du tag 3, la
// valeur portee est une fraction normalisee. Un port qui publierait « 100 » lirait l unite pour
// un pourcentage.
//
// LE TAUX N EST PAS CONSTANT, et c est la mecanique Lua qui l explique : deux lachers de meme
// duree ne montent pas pareil (18,2 s -> 1,0000 ; 26,9 s -> 0,3209), parce que le remplissage
// suit `1/reset + H(n)/solo` ou `n` est le nombre de defenseurs dans le rayon. Les REDESCENTES
// observees (jusqu a 20 sur un lacher) sont le vidage quand plus personne n y est. Rien de tout
// cela n a besoin d etre modelise pour publier la jauge : elle se lit, elle ne se calcule pas.
//
// # POURQUOI CE CANAL, APRES DEUX NEGATIFS
//
// Le retour d un drapeau de CTF n est pas un MINUTEUR, c est une JAUGE : le script du mode
// (`parcel_deliver_object.lua`) remplit `flagReturnTimer` au taux `1/reset + H(n)/solo`, ou `n`
// est le nombre de defenseurs dans le rayon et `H` la serie harmonique — le jeu nomme lui-meme la
// fonction `CalculateReturnRateHarmonic`. Les deux canaux de MINUTEUR ont ete refutes par la
// mesure : le bassin du moteur (`ti=11 i0` a « aucun minuteur » sur 446 records sur 446, lot 3.7
// § 6 bis.3) et le minuteur manuel du navpoint (`ti=12 i11`/`i12` nul sur 588 lectures sur 588,
// lot 5.1.2). Reste le canal des PROPRIETES RESEAU NOMMEES PAR LE SCRIPT — `ti=13`, dont le
// tag 3 est un `R(24)` quantifie sur [-100, +100] et dont le meme canal porte deja la jauge de
// capture des zones en production (note 3.7 § 9.3).
//
// # CE QUE CET INSTRUMENT AJOUTE, ET CE QU IL NE REFAIT PAS
//
// Le balayage de `ti=13` par la voie DELTA existe deja et il est eprouve : `ti13EtatScan`
// (`ti13_etat_test.go`, lot C-bis). Il ancre les records, rejoue la boucle de PRODUCTION
// composant par composant et recolte ce que le hook publie. Cet instrument NE LE RECOPIE PAS :
// il l appelle, et se contente d ecrire les echantillons du tag 3 (mode A) dates par slot, pour
// qu ils se confrontent aux intervalles `dropped` / `returned` que `flagCarries` publie.
//
// LA VOIE IMAGE-CLE N EST PAS UTILISABLE ICI, et c est mesure ailleurs : l etat par defaut de
// `ti=13` est desaligne en image-cle (oracle `n2`, note 3.7 § 9.3). Seule la voie DELTA parle.
//
// UN SEUL FILM PAR INVOCATION.
//
//	ZONE_FILM=<cache>/film_chunks/bcb6d393 ZONE_OBJTYPE=flag ZONE_OUT=<tmp>
//	go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/
//	  -run Jauge51Ti13 -v -timeout 60m

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestJauge51Ti13SurFilm(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	court := filepath.Base(dir)

	_, reg := zcLoadGrammar(t, dir)
	c := zcKeyframeCensus(dir)
	col := ti13EtatScan(t, c, zcBuildBands(c), reg, zcLoadClock(t, dir))

	ech := make([]ti13Ech, 0, len(col.scal))
	for _, e := range col.scal {
		if e.tag == ManagedPropertyTagQuant && e.hasPay {
			ech = append(ech, e)
		}
	}
	sort.SliceStable(ech, func(i, j int) bool { return ech[i].tMS < ech[j].tMS })

	parSlot := map[uint32]int{}
	for _, e := range ech {
		parSlot[e.slot]++
	}
	slots := make([]int, 0, len(parSlot))
	for s := range parSlot {
		slots = append(slots, int(s))
	}
	sort.Ints(slots)

	var b strings.Builder
	fmt.Fprintf(&b, "# jauge ti=13 tag 3 (mode A, R(24) sur [-100, +100]) — film %s\n", court)
	fmt.Fprintf(&b, "# %d echantillons sur %d slot(s) ; records ancres=%d consommes=%d\n",
		len(ech), len(parSlot), col.records, col.walked)
	b.WriteString("tms\tslot\tquantum\tvaleur\n")
	for _, e := range ech {
		fmt.Fprintf(&b, "%d\t%d\t%d\t%.6f\n", e.tMS, e.slot, e.pay, ManagedPropertyQuantValue(e.pay))
	}
	zcWriteFile(t, filepath.Join(out, court+"_ti13_tag3.tsv"), b.String())

	t.Logf("FILM %s — %d echantillons de tag 3 sur %d slot(s) ; records ti=13 ancres=%d consommes=%d",
		court, len(ech), len(parSlot), col.records, col.walked)
	for _, s := range slots {
		t.Logf("FILM %s — slot %d : %d echantillons", court, s, parSlot[uint32(s)]) //nolint:gosec // slot de 13 bits
	}
}
