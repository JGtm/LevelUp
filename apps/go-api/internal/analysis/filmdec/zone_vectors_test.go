package filmdec

// zone_vectors_test.go — ITEM C.1a.1, volet VECTEURS : sortir des octets de film reels les
// vecteurs de test des grammaires lues dans le binaire (phase 1a, lecture seule).
//
// POURQUOI LE MASQUE SINGLETON EST LA CLE. La traversee consomme les composants annonces DANS
// L'ORDRE DU MASQUE : pour savoir ou commence la charge utile du composant X, il faut connaitre
// la largeur de tous les composants annonces AVANT lui — or c'est precisement ce qu'on ne sait
// pas encore. Un record dont le masque vaut EXACTEMENT {X} n'a pas ce probleme : la charge utile
// de X commence au premier bit apres le masque, c'est-a-dire a `rec.After`. La phase 0 a montre
// que ce cas est le cas DOMINANT (95,9 % des records ti=12 de `696a9d7c` annoncent un seul
// composant), donc il y a de quoi choisir.
//
// CE QUE CE FICHIER NE FAIT PAS : il ne porte rien. Il lit les bits a la position que l'ancrage
// donne, applique la grammaire lue dans le binaire, et publie le quantum BRUT a cote de la valeur
// interpretee — pour qu'un desaccord sur la dequantification ne contamine pas le vecteur.
//
// USAGE (depuis apps/go-api) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestZoneVectorsLotC1a -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// zcVecTarget est un composant dont on veut des vecteurs, et la largeur totale de sa charge
// utile telle que le deserialiseur du jeu la lit (grammaire etablie en phase 1a).
type zcVecTarget struct {
	ti, i int
	nom   string
	// payloadBits est la largeur LUE dans le binaire. 0 = grammaire a largeur variable : on
	// publie alors les bits bruts sans les decouper.
	payloadBits int
	// fields decrit le decoupage de la charge utile, en bits, dans l'ordre de lecture.
	fields []zcVecField
}

// zcVecField est un champ de la charge utile : sa largeur en bits et sa plage de dequantification.
type zcVecField struct {
	nom      string
	bits     int
	min, max float64
}

// zcVecTargets sont les cibles de l'item C.1a.1, avec la grammaire lue dans HaloInfinite.exe.
// Les adresses sont au journal `LOTC_PHASE1A.md` ; ici on ne garde que la forme.
var zcVecTargets = []zcVecTarget{
	{ti: 12, i: 14, nom: "managed-navpoint-radial-progress", payloadBits: 8, fields: []zcVecField{
		{nom: "progress", bits: 8, min: -1, max: 1},
	}},
	{ti: 10, i: 1, nom: "managed-object-boundary-color-component", payloadBits: 32, fields: []zcVecField{
		{nom: "r", bits: 8, min: 0, max: 1},
		{nom: "g", bits: 8, min: 0, max: 1},
		{nom: "b", bits: 8, min: 0, max: 1},
		{nom: "a", bits: 8, min: 0, max: 1},
	}},
	// rtpc : R(32) identifiant puis, SI l'identifiant est non nul, R(22) quantifie sur
	// [-10000, +10000]. Largeur donc DATA-DEPENDANTE : 32 ou 54.
	{ti: 10, i: 26, nom: "managed-object-rtpc-component (i26)", payloadBits: 0},
	{ti: 10, i: 27, nom: "managed-object-rtpc-component (i27)", payloadBits: 0},
}

// zcVecMax est le nombre de vecteurs publies par cible : le plan en demande 2 a 3.
const zcVecMax = 3

// zcDequant rend la valeur flottante d'un quantum sur `bits` bits dans [min, max], par la
// convention du MILIEU d'intervalle (celle de `decodeWorldObjectPos`, projectiles.go:358).
//
// RESERVE ECRITE : la convention exacte du jeu (milieu d'intervalle contre bornes incluses)
// n'est PAS etablie par cette passe — le calcul flottant de `FUN_1406d84b4` sort en XMM0 et la
// decompilation ne le rend pas. L'ecart entre les deux conventions vaut un demi-quantum
// (0,4 % ici) : sans consequence pour la FORME, a trancher en phase 1b si une valeur exacte est
// exigee. Le quantum BRUT est publie a cote, lui n'est pas discutable.
func zcDequant(q uint64, bits int, min, max float64) float64 {
	if bits <= 0 {
		return 0
	}
	steps := float64(uint64(1) << uint(bits))
	return min + (float64(q)+0.5)*(max-min)/steps
}

// TestZoneVectorsLotC1a publie, par cible, les vecteurs de test et la distribution du champ.
func TestZoneVectorsLotC1a(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	c := zcKeyframeCensus(dir)
	bands := zcBuildBands(c)
	var sb strings.Builder
	sb.WriteString("ti\ti\tcomposant\tchunk\tpaquet\tbit_record\tbit_charge\tslot\tgen\t" +
		"masque\tbits_bruts_hex\tchamps\n")
	for _, tg := range zcVecTargets {
		zcVectorsFor(t, &sb, c, bands, tg)
	}
	zcWriteFile(t, filepath.Join(out, short+"_vecteurs.tsv"), sb.String())
}

// zcVecHit est un record retenu comme vecteur.
type zcVecHit struct {
	chunk, pkt, bitRec, bitPay int
	slot, gen                  uint32
	raw                        uint64
	rawBits                    int
}

// zcVectorsFor balaye le film a la recherche de records dont le masque vaut EXACTEMENT {tg.i},
// publie les premiers comme vecteurs et l'histogramme du premier champ sur TOUS les records
// trouves (le denominateur est publie).
func zcVectorsFor(t *testing.T, sb *strings.Builder, c zcCensus, b zcBands, tg zcVecTarget) {
	t.Helper()
	band := b.perTI[tg.ti]
	if len(band) == 0 {
		t.Logf("CIBLE ti=%d i%d %s : aucun slot de cet archetype dans ce film — aucun vecteur",
			tg.ti, tg.i, tg.nom)
		return
	}
	var hits []zcVecHit
	total := 0
	hist := map[uint64]int{}
	width := tg.payloadBits
	if width == 0 {
		width = 64 // grammaire variable : on publie 64 bits bruts et on ne decoupe pas
	}
	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
			for p := 0; p <= limit; p++ {
				rec, ok := matchWorldObjectRecord(pay, p, band)
				if !ok {
					continue
				}
				p = rec.After
				if len(rec.Idx) != 1 || rec.Idx[0] != tg.i {
					continue // masque non singleton : la position de la charge utile est inconnue
				}
				if rec.After+width > len(pay)*8 {
					continue
				}
				total++
				raw := PeekBits(pay, rec.After, width)
				if len(tg.fields) > 0 {
					hist[raw>>uint(width-tg.fields[0].bits)]++
				}
				if len(hits) < zcVecMax {
					hits = append(hits, zcVecHit{
						chunk: ch, pkt: pk.Index, bitRec: zcRecordStartBit(rec), bitPay: rec.After,
						slot: rec.Slot, gen: rec.Gen, raw: raw, rawBits: width,
					})
				}
			}
		}
	}
	zcVecReport(t, sb, tg, hits, total, hist, width)
}

// zcRecordStartBit rend la position du premier bit d un record. `matchWorldObjectRecord` ne la
// rend pas, mais elle se retrouve : la charge utile commence a `After`, et l en-tete plus les
// index du masque la precedent.
func zcRecordStartBit(rec WorldObjectRecord) int {
	return rec.After - worldObjectHeaderBits - worldObjectIndexBits*len(rec.Idx)
}

// zcVecReport journalise et ecrit les vecteurs d'une cible.
func zcVecReport(t *testing.T, sb *strings.Builder, tg zcVecTarget, hits []zcVecHit,
	total int, hist map[uint64]int, width int,
) {
	t.Helper()
	t.Logf("CIBLE ti=%d i%d %s — %d records a masque SINGLETON {i%d} (denominateur des vecteurs)",
		tg.ti, tg.i, tg.nom, total, tg.i)
	if total == 0 {
		return
	}
	for n, h := range hits {
		champs := zcVecFields(tg, h.raw, width)
		t.Logf("  vecteur %d : chunk %d · paquet %d · bit du record %d · bit de la charge %d"+
			" · slot %d gen %d · bruts 0x%0*X · %s",
			n+1, h.chunk, h.pkt, h.bitRec, h.bitPay, h.slot, h.gen, (width+3)/4, h.raw, champs)
		sb.WriteString(fmt.Sprintf("%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t{%d}\t%0*X\t%s\n",
			tg.ti, tg.i, tg.nom, h.chunk, h.pkt, h.bitRec, h.bitPay, h.slot, h.gen, tg.i,
			(width+3)/4, h.raw, champs))
	}
	if len(hist) == 0 {
		return
	}
	keys := make([]uint64, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return hist[keys[i]] > hist[keys[j]] })
	var top strings.Builder
	for i, k := range keys {
		if i >= 8 {
			break
		}
		fmt.Fprintf(&top, "%d:%d (%.1f %%) ", k, hist[k], 100*float64(hist[k])/float64(total))
	}
	t.Logf("  DISTRIBUTION du premier champ (%d valeurs distinctes sur %d records) — les 8 plus"+
		" frequentes : %s", len(hist), total, top.String())
}

// zcVecFields decoupe la charge utile selon la grammaire et rend une chaine lisible.
func zcVecFields(tg zcVecTarget, raw uint64, width int) string {
	if len(tg.fields) == 0 {
		return "grammaire a largeur variable : non decoupee"
	}
	var sb strings.Builder
	off := 0
	for _, f := range tg.fields {
		q := (raw >> uint(width-off-f.bits)) & ((uint64(1) << uint(f.bits)) - 1)
		fmt.Fprintf(&sb, "%s=q%d/%.4f ", f.nom, q, zcDequant(q, f.bits, f.min, f.max))
		off += f.bits
	}
	return strings.TrimSpace(sb.String())
}
