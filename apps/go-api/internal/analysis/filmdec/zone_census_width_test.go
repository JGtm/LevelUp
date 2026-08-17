package filmdec

// zone_census_width_test.go — ITEM C.1a.0 : la largeur du champ de slot de l'ancrage n'est pas
// une constante du format, et le prouver change les chiffres de la phase 0.
//
// LE FAIT QUI OUVRE CET ITEM. `matchWorldObjectRecord` (projectiles.go:305) lit le slot sur 13
// bits, et `equipment_creation.go:88` cable la meme valeur en la nommant `FrameConfig.IDLowBits`.
// Or cette largeur est une valeur de RUNTIME, peuplee au chargement de carte
// (`frame_records.go:39-44` : 11 sur `000d5950`, 14 sur le film de la capture live) : elle
// DIFFERE d'un film a l'autre. Un ancrage a 13 bits fixes lit donc, sur une partie du corpus, un
// mauvais numero de slot — il rejette de vrais records et accepte des coincidences. C'est le
// candidat n1 pour expliquer les 22 a 39 % de records hors grammaire mesures en phase 0.
//
// CE QUE CET INSTRUMENT FAIT, ET CE QU'IL NE FAIT PAS. Il rejoue le recensement des annonces
// sous PLUSIEURS largeurs de slot, dans une seule passe disque, et publie la part hors grammaire
// de chacune. Il ne modifie AUCUN code de production : `matchWorldObjectRecord` reste tel quel,
// et la variante parametree ci-dessous vit dans ce fichier de test. C'est la seule forme possible
// ici — la phase 1a est en lecture seule et la plomberie partagee appartient au lot 0.
//
// DEUX JUGES INDEPENDANTS, et c est voulu. (1) Le balayage de cadre DEJA present au depot
// (`BestVariant` + `KFQFrameVariants`, keyframe_entity_queue.go) : il choisit la combinaison qui
// consomme le plus de bits d un paquet sans desynchroniser ni deborder — il ne connait rien aux
// bandes de slots. (2) La part HORS GRAMMAIRE de l ancrage, archetype par archetype, dont le
// meilleur juge est `ti=4` : UN slot, UN composant declare, donc un record de ti=4 ne peut
// annoncer QUE i0 — toute autre annonce est une faute de largeur. Quand les deux juges
// concordent, la largeur est etablie ; quand ils divergent, c est le second qui tranche, parce
// qu il porte sur les records reellement recenses.
//
// USAGE (depuis apps/go-api) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestZoneWidthLotC1a -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// zcWidthSweep sont les largeurs de champ de slot probees. 13 est la valeur cablee en
// production (le AVANT) ; 11 et 14 sont les deux valeurs de runtime deja mesurees au depot.
var zcWidthSweep = []int{11, 12, 13, 14, 15}

// zcWidthRef est la largeur cablee dans `matchWorldObjectRecord` — la reference AVANT.
const zcWidthRef = 13

// zcHeaderBitsFor rend la largeur de l'en-tete d'un record delta pour une largeur de slot
// donnee : R(1) prefixe + slot + R(2) tag + R(1) selecteur de base + R(1) porte de masque +
// R(3) compte. Avec w = 13 on retrouve `worldObjectHeaderBits` = 21, et c'est ce controle qui
// dit que cette variante est bien la meme grammaire que celle de production.
func zcHeaderBitsFor(w int) int { return w + 8 }

// zcMatchRecordW est `matchWorldObjectRecord` avec la largeur de slot en parametre. Meme
// grammaire, memes deux ecarts assumes (aucun filtre sur le tag, compte de masque minimal a 1).
func zcMatchRecordW(pay []byte, p int, band map[uint32]bool, w int) (WorldObjectRecord, bool) {
	var rec WorldObjectRecord
	if PeekBits(pay, p, 1) != 1 { // prefixe de record DELTA
		return rec, false
	}
	slot := uint32(PeekBits(pay, p+1, w))
	if !band[slot] {
		return rec, false
	}
	if PeekBits(pay, p+w+3, 2) != 0 { // selecteur de base ET porte de masque nuls
		return rec, false
	}
	mc := int(PeekBits(pay, p+w+5, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt {
		return rec, false
	}
	hdr := zcHeaderBitsFor(w)
	idx, ok := ascendingComponents(pay, p+hdr, mc)
	if !ok {
		return rec, false
	}
	rec.Slot, rec.Gen = slot, uint32(PeekBits(pay, p+w+1, 2))
	rec.Idx, rec.After = idx, p+hdr+worldObjectIndexBits*mc
	return rec, true
}

// zcInferIDLowBits rend la largeur de champ d'id que le CADRE du film impose, mesuree par le
// balayage deja present au depot, et le detail de la mesure.
//
// Le vote porte sur plusieurs paquets delta et non sur un seul : le premier paquet d'un film est
// un cas particulier (etat complet) et un film peut porter plusieurs sessions
// (`AllPacketsOfType` le dit). La largeur retenue est celle de meilleure COUVERTURE MOYENNE.
func zcInferIDLowBits(t *testing.T, dir string, reg *Registry, maxPk int) (int, string) {
	t.Helper()
	chunks, pkts, _ := AllPacketsOfType(dir, PacketTypeDelta)
	votes := map[int]int{}
	cover := map[int]float64{}
	lo, hi := zcWidthSweep[0], zcWidthSweep[len(zcWidthSweep)-1]
	n := 0
	for i := range pkts {
		if n >= maxPk {
			break
		}
		pay := pkts[i].Payload(chunks[i])
		if len(pay) < 256 {
			continue
		}
		best, _ := BestVariant(reg, pay, KFQFrameVariants(lo, hi), nil)
		if best.TotalBits == 0 {
			continue
		}
		votes[best.Variant.IDLowBits]++
		cover[best.Variant.IDLowBits] += best.Coverage()
		n++
	}
	if len(votes) == 0 {
		return zcWidthRef, fmt.Sprintf("aucun paquet exploitable : repli sur %d", zcWidthRef)
	}
	// LE CRITERE EST LA COUVERTURE MOYENNE, PAS LE NOMBRE DE VOTES. Mesure du 2026-08-17 sur
	// `7344d24f` : les 8 paquets votants se repartissent sur les cinq largeurs (2-2-2-1-1), donc
	// la modale est un ex aequo que l ordre de parcours tranche au hasard — elle avait designe 12.
	// La couverture, elle, separe nettement (0,380 pour w=13 contre 0,003 a 0,132 ailleurs), et
	// c est la grandeur que `BestVariant` optimise deja paquet par paquet.
	keys := zcSortedKeysInt(votes)
	sort.SliceStable(keys, func(a, b int) bool {
		ma := cover[keys[a]] / float64(votes[keys[a]])
		mb := cover[keys[b]] / float64(votes[keys[b]])
		return ma > mb
	})
	var sb strings.Builder
	for _, k := range zcSortedKeysInt(votes) {
		fmt.Fprintf(&sb, "w=%d:%d vote(s), couverture moyenne %.3f ; ",
			k, votes[k], cover[k]/float64(votes[k]))
	}
	return keys[0], fmt.Sprintf("%d paquets votants — %s", n, sb.String())
}

// TestZoneWidthLotC1a rejoue le recensement des annonces sous plusieurs largeurs de slot et
// publie la part hors grammaire de chacune. Un film par processus.
func TestZoneWidthLotC1a(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	gram, reg := zcLoadGrammar(t, dir)
	grammarLen := map[int]int{}
	for ti, g := range gram {
		grammarLen[ti] = len(g.components)
	}
	c := zcKeyframeCensus(dir)
	bands := zcBuildBands(c)

	inferred, detail := zcInferIDLowBits(t, dir, reg, 8)
	t.Logf("FILM %s — largeur de champ d'id inferee par le balayage de cadre : %d"+
		" (reference cablee en production : %d)", short, inferred, zcWidthRef)
	t.Logf("  detail du vote : %s", detail)

	res := zcScanWidths(c, bands, grammarLen)
	zcReportWidths(t, res, bands, gram, inferred, out, short)
}

// zcWidthStats porte, pour UNE largeur, les stats par classe de slot.
type zcWidthStats map[int]*zcDeltaStats

// zcScanWidths balaye les paquets delta UNE fois et agrege pour toutes les largeurs probees.
// Une position de bit est testee sous chaque largeur : ce sont des grammaires differentes, donc
// des positions de fin differentes, donc chaque largeur avance a son propre pas.
func zcScanWidths(c zcCensus, b zcBands, grammarLen map[int]int) map[int]zcWidthStats {
	res := map[int]zcWidthStats{}
	for _, w := range zcWidthSweep {
		st := zcWidthStats{}
		for _, ti := range zcTargetTIs {
			st[ti] = newZCDeltaStats()
		}
		for _, cl := range []int{zcClassInconnu, zcClassOccupe, zcClassVide} {
			st[cl] = newZCDeltaStats()
		}
		res[w] = st
	}
	noWin := zcPacketWindow{} // l'item C.1a.0 ne mesure pas la densite : aucune fenetre
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
			for _, w := range zcWidthSweep {
				zcScanPayloadW(pay, res[w], b, grammarLen, w, noWin)
			}
		}
	}
	return res
}

// zcScanPayloadW balaye un payload sous UNE largeur de slot.
func zcScanPayloadW(pay []byte, st zcWidthStats, b zcBands, grammarLen map[int]int,
	w int, noWin zcPacketWindow,
) {
	limit := len(pay)*8 - (zcHeaderBitsFor(w) + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := zcMatchRecordW(pay, p, b.all, w)
		if !ok {
			continue
		}
		cl := b.class[rec.Slot]
		zcAccumulate(st[cl], rec, grammarLen[cl], noWin)
		p = rec.After
	}
}

// zcReportWidths journalise et ecrit la comparaison AVANT / APRES.
func zcReportWidths(t *testing.T, res map[int]zcWidthStats, b zcBands, gram map[int]zcArchInfo,
	inferred int, out, short string,
) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("largeur_slot\tclasse\trecords\tslots_peuples\trecords_par_slot\t" +
		"hors_grammaire_pct\tplancher_bruit\n")
	for _, w := range zcWidthSweep {
		mark := ""
		if w == zcWidthRef {
			mark = "   <- AVANT (cable en production)"
		}
		if w == inferred {
			mark += "   <- APRES (inferee pour ce film)"
		}
		t.Logf("  largeur de slot w=%d%s", w, mark)
		for _, ti := range zcTargetTIs {
			zcWidthRow(t, &sb, res[w][ti], w, fmt.Sprintf("ti=%d", ti), len(b.perTI[ti]))
		}
		for _, cl := range []int{zcClassVide, zcClassInconnu, zcClassOccupe} {
			zcWidthRow(t, &sb, res[w][cl], w, "temoin_"+zcClassName(cl), 0)
		}
	}
	zcVerdictWidth(t, res)
	zcWriteFile(t, filepath.Join(out, short+"_largeur_slot.tsv"), sb.String())
}

// zcVerdictWidth publie la largeur que le SECOND juge designe : celle qui minimise la part hors
// grammaire de `ti=4` (un slot, un composant declare) puis, en appui, celle de `ti=47`.
func zcVerdictWidth(t *testing.T, res map[int]zcWidthStats) {
	t.Helper()
	best, bestRate, line := 0, 101.0, ""
	for _, w := range zcWidthSweep {
		s := res[w][4]
		if s == nil || s.records == 0 {
			line += fmt.Sprintf("w=%d: aucun record ; ", w)
			continue
		}
		r := 100 * zcRate(s.outOfGrammar, s.records)
		line += fmt.Sprintf("w=%d: %.2f %% (%d records) ; ", w, r, s.records)
		if r < bestRate {
			best, bestRate = w, r
		}
	}
	t.Logf("  JUGE DE PURETE ti=4 (un slot, un composant : toute annonce hors i0 est une faute de"+
		" largeur) — %s", line)
	t.Logf("  VERDICT DE LARGEUR par ce juge : w=%d (%.2f %% hors grammaire)", best, bestRate)
}

// zcWidthRow ecrit une ligne (largeur, classe) et la journalise si elle porte quelque chose.
func zcWidthRow(t *testing.T, sb *strings.Builder, s *zcDeltaStats, w int, label string, bandSize int) {
	t.Helper()
	if s == nil {
		return
	}
	oog := 100 * zcRate(s.outOfGrammar, s.records)
	sb.WriteString(fmt.Sprintf("%d\t%s\t%d\t%d\t%.1f\t%.2f\t%.1f\n",
		w, label, s.records, len(s.slots), zcRecordsPerSlot(s), oog, zcNoiseFloor(s)))
	if s.records == 0 {
		return
	}
	extra := ""
	if bandSize > 0 {
		extra = fmt.Sprintf(" (%d/%d slots)", len(s.slots), bandSize)
	}
	t.Logf("      %-16s %8d records%-16s · %7.1f/slot · hors grammaire %6.2f %% · plancher %6.1f",
		label, s.records, extra, zcRecordsPerSlot(s), oog, zcNoiseFloor(s))
}
