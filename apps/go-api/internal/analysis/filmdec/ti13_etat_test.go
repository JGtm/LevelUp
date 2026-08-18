package filmdec

// ti13_etat_test.go — LOT C-bis PHASE 1, item CB.1.2 : LA MESURE D'ETAT de `ti=13`.
//
// CE QUI CHANGE PAR RAPPORT A LA PHASE 0. La phase 0 lisait les valeurs a la main, sur les seuls
// records a masque SINGLETON. Les deux composants sont maintenant PORTES (item CB.1.1), donc on
// rejoue la BOUCLE DE PRODUCTION composant par composant sur tout record ancre, dans l'ordre du
// masque, et les valeurs viennent du code de production — ce que cet instrument mesure est
// exactement ce que la phase 2 publierait. Meme voie qu'au lot C (`zone_state_measure_test.go`).
//
// L'INDEX DE JOUEUR SE RECONSTITUE DEPUIS LE MASQUE. `consumeByName` ne recoit pas l'index du
// composant ; le hook, lui, est appele DANS L'ORDRE DU MASQUE. Un curseur suit donc la position
// dans `rec.Idx` et rend l'index reel, que `ManagedPropertyFilmIndex` traduit en numero de
// joueur. C'est le partage des roles decrit dans `components_managed_property.go`.
//
// LES SEUILS SONT ECRITS AVANT LA MESURE (constantes ci-dessous) et ne sont pas ajustables ici.
// Chaque taux est publie avec SON DENOMINATEUR et, quand il s'agit d'une coincidence temporelle,
// avec le NIVEAU DU HASARD — la lecon du lot C, dont le temoin a 20 % etait inatteignable pour un
// canal qui produit 50 rampes par match.
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	$env:ZONE_OBJTYPE="zone"   # zone | flag | none
//	go test -count=1 -run TestTi13EtatLotCbis -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// SEUILS DU GATE 1, ecrits AVANT la mesure (plan `PLAN_EXPLOITATION_REGISTRE_FILM.md`, CB.1.2).
const (
	// ti13FenetreMS : demi-fenetre autour d'une capture. ti13DecalageMS : le temoin decale.
	ti13FenetreMS  = 2000
	ti13DecalageMS = 20000
	// (a) la carte slot -> identifiant de chaine doit etre coherente a ce taux.
	ti13SeuilCoherence = 0.90
	// (b) et (c) : part des captures couvertes.
	ti13SeuilCaptures = 0.80
	// (c) un ETAT est enumerable : au plus 8 valeurs distinctes par slot.
	ti13MaxValeursEtat = 8
	// (d) KOTH : part du temps ou une seule zone est active.
	ti13SeuilUnique = 0.90
	// Ce qui compte comme une RAMPE du tag 3. Trois echantillons croissants, et une amplitude
	// d'au moins 4 096 quanta sur 24 bits — soit ~0,049 unite sur [-100, +100], environ trois
	// fois le pas inter-echantillon mesure en phase 0 (~1 199 quanta). Ecrit avant la mesure.
	ti13RampMinSamples   = 3
	ti13RampMinAmplitude = 4096
	// Volume minimal pour qu'un slot soit juge : en dessous, un taux n'a pas de sens.
	ti13MinParSlot = 10
)

// ti13Ech est une valeur publiee par le hook, datee, attribuee a son slot et a son composant.
type ti13Ech struct {
	tMS    int
	slot   uint32
	idx    int // index du composant dans l'archetype (1 = scalaire, 2..33 = par joueur)
	tag    int
	pay    uint64
	hasPay bool
}

// ti13Col porte ce qu'une passe a recolte.
type ti13Col struct {
	scal []ti13Ech // i1, mode A
	joue []ti13Ech // i2..i33, mode B
	// temoins d'ancrage, publies a chaque passe
	records, walked            int
	ghostRecords               int
	ti4Records, ti4HorsGrammar int
}

// TestTi13EtatLotCbis joue la mesure d'etat CB.1.2 sur UN film.
func TestTi13EtatLotCbis(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	_, reg := zcLoadGrammar(t, dir)
	c := zcKeyframeCensus(dir)
	bands := zcBuildBands(c)
	oracle := zcLoadOracle(t, dir)
	clk := zcLoadClock(t, dir)

	col := ti13EtatScan(t, c, bands, reg, clk)
	t.Logf("FILM %s — %d records ti=13 ancres, %d entierement consommes · ORACLE %q : %d"+
		" evenements d'objectif", short, col.records, col.walked, oracle.family, len(oracle.times))
	t.Logf("  TEMOINS D'ANCRAGE : bande fantome (vide) %d records · purete ti=4 %d records dont"+
		" %.2f %% hors grammaire", col.ghostRecords, col.ti4Records,
		100*zcRate(col.ti4HorsGrammar, col.ti4Records))
	t.Logf("  VALEURS RECOLTEES : i1 scalaire %d · i2..i33 par joueur %d",
		len(col.scal), len(col.joue))

	var sb strings.Builder
	ti13RapportIdentifiants(t, &sb, short, col)  // (a)
	ti13RapportRampe(t, &sb, short, col, oracle) // (b)
	ti13RapportEtat(t, &sb, short, col, oracle)  // (c)
	ti13RapportUnicite(t, &sb, short, col)       // (d)
	ti13RapportParJoueur(t, &sb, short, col)     // (d) volet par joueur, (e)
	zcWriteFile(t, filepath.Join(out, short+"_ti13_etat.tsv"), sb.String())
}

// ti13EtatScan balaye le film et rejoue la boucle de production sur les records ti=13.
func ti13EtatScan(t *testing.T, c zcCensus, b zcBands, reg *Registry, clk zcClock) *ti13Col {
	t.Helper()
	col := &ti13Col{}
	var curT int
	var curSlot uint32
	var curIdx []int
	var curPos int

	prev := managedPropertyHook
	SetManagedPropertyHook(func(f ManagedPropertyField, values []uint64) {
		e := ti13Ech{tMS: curT, slot: curSlot, idx: -1, tag: int(values[0])}
		if curPos < len(curIdx) {
			e.idx = curIdx[curPos]
		}
		curPos++
		if len(values) > 1 {
			e.pay, e.hasPay = values[1], true
		}
		if f == ManagedPropertyScalar {
			col.scal = append(col.scal, e)
			return
		}
		col.joue = append(col.joue, e)
	})
	defer SetManagedPropertyHook(prev)

	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		startMS, hasStart := clk.startMS[ch]
		var base uint64
		haveBase := false
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			if !haveBase {
				base, haveBase = pk.TimestampUS, true
			}
			if !hasStart {
				continue
			}
			curT = startMS + int((pk.TimestampUS-base)/1000)
			ti13EtatPayload(pk.Payload(data), b, reg, col, &curSlot, &curIdx, &curPos)
		}
	}
	return col
}

// ti13EtatPayload ancre les records d'un payload et rejoue leurs composants annonces.
func ti13EtatPayload(pay []byte, b zcBands, reg *Registry, col *ti13Col,
	curSlot *uint32, curIdx *[]int, curPos *int,
) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, b.all)
		if !ok {
			continue
		}
		p = rec.After
		switch cl := b.class[rec.Slot]; {
		case cl == zcClassVide:
			col.ghostRecords++
			continue
		case cl == 4:
			col.ti4Records++
			for _, i := range rec.Idx {
				if i != 0 {
					col.ti4HorsGrammar++
					break
				}
			}
			continue
		case cl != 13:
			continue
		}
		arch, ok := reg.Archetype(13)
		if !ok {
			continue
		}
		col.records++
		*curSlot, *curIdx, *curPos = rec.Slot, rec.Idx, 0
		if zsReplay(pay, rec, arch, 13) {
			col.walked++
		}
	}
}

// -------------------------------------------------------------------------------------
// (a) TAG 5 — l'identifiant de chaine, cle d'appariement slot -> zone
// -------------------------------------------------------------------------------------

// ti13RapportIdentifiants mesure la coherence de la carte slot -> identifiant de chaine.
//
// CE QUE CE VOLET FAIT, ET CE QU'IL NE FAIT PAS. Il etablit la CLE d'appariement : un slot
// ti=13 porte-t-il UN identifiant de chaine, stable sur tout le match ? L'appariement de cette
// cle a une zone du CATALOGUE (`replay.AttributeZones`) n'est PAS fait ici, et pour une raison
// dure : `internal/analysis/replay` importe `filmdec`, donc `filmdec` ne peut pas importer
// `replay` — le pont geometrique appartient au paquet `replay`, c'est-a-dire a la phase 2.
func ti13RapportIdentifiants(t *testing.T, sb *strings.Builder, short string, col *ti13Col) {
	t.Helper()
	parSlot := map[uint32]map[uint64]int{}
	for _, e := range col.scal {
		if e.tag != ManagedPropertyTagStringID || !e.hasPay {
			continue
		}
		if parSlot[e.slot] == nil {
			parSlot[e.slot] = map[uint64]int{}
		}
		parSlot[e.slot][e.pay]++
	}
	t.Logf("")
	t.Logf("=== (a) TAG 5 identifiant de chaine — %d slots en portent", len(parSlot))
	if len(parSlot) == 0 {
		t.Logf("  aucun identifiant sur ce film : volet (a) SANS OBJET ici")
		fmt.Fprintf(sb, "# (a) %s : aucun identifiant de chaine\n", short)
		return
	}
	slots := ti13SlotsTries(parSlot)
	coherents, juges := 0, 0
	for _, s := range slots {
		tot, best, bestV := 0, 0, uint64(0)
		for v, n := range parSlot[s] {
			tot += n
			if n > best {
				best, bestV = n, v
			}
		}
		part := float64(best) / float64(tot)
		if tot >= ti13MinParSlot {
			juges++
			if part >= ti13SeuilCoherence {
				coherents++
			}
		}
		t.Logf("  slot %d : identifiant dominant 0x%08X sur %d emissions (%.1f %%), %d valeurs"+
			" distinctes", s, bestV, tot, 100*part, len(parSlot[s]))
		fmt.Fprintf(sb, "a_identifiant\t%s\t%d\t%d\t%d\t%.3f\t%d\n", short, s, bestV, tot,
			part, len(parSlot[s]))
	}
	ti13VerdictA(t, sb, short, coherents, juges)
}

// ti13VerdictA rend le verdict du volet (a) contre son seuil.
func ti13VerdictA(t *testing.T, sb *strings.Builder, short string, coherents, juges int) {
	t.Helper()
	if juges == 0 {
		t.Logf("  VERDICT (a) : aucun slot n'atteint %d emissions — NON JUGEABLE sur ce film",
			ti13MinParSlot)
		fmt.Fprintf(sb, "# (a) %s : non jugeable (aucun slot a %d emissions)\n", short,
			ti13MinParSlot)
		return
	}
	part := float64(coherents) / float64(juges)
	v := "NON TENU"
	if part >= ti13SeuilCoherence {
		v = "TENU"
	}
	t.Logf("  VERDICT (a) : %d/%d slots juges (>= %d emissions) portent UN identifiant dominant"+
		" a >= %.0f %% = %.1f %% — seuil %.0f %% : %s", coherents, juges, ti13MinParSlot,
		100*ti13SeuilCoherence, 100*part, 100*ti13SeuilCoherence, v)
	fmt.Fprintf(sb, "# (a) %s : %d/%d slots coherents (%.1f %%), seuil %.0f %%, verdict %s\n",
		short, coherents, juges, 100*part, 100*ti13SeuilCoherence, v)
}

// -------------------------------------------------------------------------------------
// (b) TAG 3 — la rampe, confrontee aux captures
// -------------------------------------------------------------------------------------

// ti13Ramp est une montee monotone detectee sur un slot.
type ti13Ramp struct {
	slot         uint32
	t0, tMax     int
	qStart, qMax uint64
	samples      int
}

// ti13RapportRampe detecte les rampes du tag 3 et les confronte aux captures.
func ti13RapportRampe(t *testing.T, sb *strings.Builder, short string, col *ti13Col,
	o zcOracle,
) {
	t.Helper()
	series := map[uint32][]ti13Ech{}
	for _, e := range col.scal {
		if e.tag == ManagedPropertyTagQuant && e.hasPay {
			series[e.slot] = append(series[e.slot], e)
		}
	}
	var ramps []ti13Ramp
	for s, ss := range series {
		sort.Slice(ss, func(i, j int) bool { return ss[i].tMS < ss[j].tMS })
		ramps = append(ramps, ti13FindRamps(s, ss)...)
	}
	t.Logf("")
	t.Logf("=== (b) TAG 3 rampe quantifiee — %d valeurs sur %d slots · %d rampes",
		ti13CountTag(col.scal, ManagedPropertyTagQuant), len(series), len(ramps))
	if len(ramps) == 0 {
		t.Logf("  aucune rampe : volet (b) SANS OBJET sur ce film")
		fmt.Fprintf(sb, "# (b) %s : aucune rampe\n", short)
		return
	}
	t.Logf("  amplitude des rampes (quanta sur 2^24) : %s", ti13RampStats(ramps))
	tops := make([]int, 0, len(ramps))
	for _, r := range ramps {
		tops = append(tops, r.tMax)
	}
	ti13VerdictTemporel(t, sb, short, "b", "sommets de rampe", tops, o, col)
}

// ti13FindRamps decoupe une serie chronologique en montees monotones.
func ti13FindRamps(slot uint32, ss []ti13Ech) []ti13Ramp {
	var out []ti13Ramp
	i := 0
	for i < len(ss) {
		j := i
		for j+1 < len(ss) && ss[j+1].pay >= ss[j].pay {
			j++
		}
		if n := j - i + 1; n >= ti13RampMinSamples &&
			ss[j].pay-ss[i].pay >= ti13RampMinAmplitude {
			out = append(out, ti13Ramp{slot: slot, t0: ss[i].tMS, tMax: ss[j].tMS,
				qStart: ss[i].pay, qMax: ss[j].pay, samples: n})
		}
		if j == i {
			i++
			continue
		}
		i = j + 1
	}
	return out
}

// ti13RampStats resume les amplitudes.
func ti13RampStats(rs []ti13Ramp) string {
	amps := make([]int, 0, len(rs))
	for _, r := range rs {
		amps = append(amps, int(r.qMax-r.qStart))
	}
	sort.Ints(amps)
	return fmt.Sprintf("min %d · mediane %d · max %d (soit %.3f a %.3f unite sur [-100, +100])",
		amps[0], amps[len(amps)/2], amps[len(amps)-1],
		float64(amps[0])*200/(1<<24), float64(amps[len(amps)-1])*200/(1<<24))
}

// ti13CountTag compte les echantillons d'un tag.
func ti13CountTag(es []ti13Ech, tag int) int {
	n := 0
	for _, e := range es {
		if e.tag == tag {
			n++
		}
	}
	return n
}

// ti13SlotsTries rend les slots d'une carte, tries.
func ti13SlotsTries[T any](m map[uint32]T) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
