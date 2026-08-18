package filmdec

// ti13_chainage_test.go — LOT C-bis PHASE 0, volet 2 : DEPARTAGER LES DEUX MODES sur les octets.
//
// LE PROBLEME QUE CE FICHIER RESOUT. Le variant ti=13 a deux modes (`ti13_variant_test.go`) :
// mode A (scalaire, index de champ = 0xFFFFFFFF) ou les tags 1..6 lisent et 7..15 ne lisent rien,
// mode B (element #k, index < 0x20) ou c'est l'inverse. Le desassemblage dit que i1 est en mode A
// (`OR dword [RSP+0x30],0xffffffff`) et que i2..i33 prennent leur index dans le DESCRIPTEUR
// (`MOV EAX,[R9+8]`). Mais l'index exact que le registre pose dans le descripteur (0..31 ou 2..33)
// ne se lit pas dans le binaire — il est ecrit a la construction de l'archetype. Or il decide de
// la LARGEUR : sous l'une des hypotheses un record fait 4 bits, sous l'autre 36.
//
// LA MESURE QUI TRANCHE, sans rien porter. Une largeur juste fait tomber la fin du record sur le
// DEBUT DU SUIVANT ; une largeur fausse tombe n'importe ou. On decode donc le record ENTIER
// (tous les composants annonces, dans l'ordre du masque) sous chaque hypothese, et on regarde si
// un en-tete de record valide commence au bit de fin. Trois temoins bornent la lecture :
//
//	FANTOME    la meme passe sur des slots vides -> le taux du bruit pur ;
//	DECALE     le meme test au bit de fin + 3 -> le NIVEAU DU HASARD de l'en-tete lui-meme ;
//	ti=4       la bande la plus pure du corpus (1 slot, 1 composant), temoin POSITIF.
//
// AUCUN PORT : rien n'est ajoute a `traverse.go`, aucun hook, aucune ligne de table ECS.
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestTi13ChainageLotCbis -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ti13MaxComp est le plus grand index de composant que l'archetype ti=13 declare (34 composants :
// i0 nom, i1 propriete scalaire, i2..i33 propriete par joueur). Un record qui annonce au-dela
// n'est pas un record ti=13 — c'est la contamination que la sonde F5 a chiffree a 35-77 %.
const ti13MaxComp = 33

// ti13Hyp est une hypothese de mode pour les composants i2..i33.
type ti13Hyp struct {
	nom         string
	maskedModeA bool
}

var ti13Hyps = []ti13Hyp{
	{nom: "B (element #k, index < 0x20)", maskedModeA: false},
	{nom: "A (scalaire, index hors bornes)", maskedModeA: true},
}

// ti13RecordEnd decode tous les composants annonces par `rec` et rend le bit de fin.
// i0 = R(32) fixe (FUN_142ed69d8) ; i1 = variant mode A ; i2..i33 = variant selon l'hypothese.
func ti13RecordEnd(pay []byte, rec WorldObjectRecord, maskedModeA bool) (int, bool) {
	p := rec.After
	total := len(pay) * 8
	for _, i := range rec.Idx {
		switch {
		case i > ti13MaxComp:
			return 0, false // hors grammaire de l'archetype
		case i == 0:
			if p+32 > total {
				return 0, false
			}
			p += 32
		default:
			modeA := i == 1 || maskedModeA
			_, next, ok := ti13Decode(pay, p, modeA)
			if !ok {
				return 0, false
			}
			p = next
		}
	}
	return p, true
}

// ti13HeaderAt dit si un en-tete de record delta plausible commence au bit `p`. On NE contraint
// PAS le slot a la bande : un paquet melange tous les archetypes, donc le successeur d'un record
// ti=13 est le plus souvent d'un autre archetype. La contrainte porte sur la FORME de l'en-tete
// (prefixe, porte de masque, cardinalite, indices strictement croissants), ce qui laisse au
// hasard une probabilite non nulle — c'est exactement ce que le temoin DECALE mesure.
func ti13HeaderAt(pay []byte, p int) bool {
	total := len(pay) * 8
	if p+worldObjectHeaderBits+worldObjectIndexBits > total {
		return false
	}
	if PeekBits(pay, p, 1) != 1 {
		return false
	}
	if PeekBits(pay, p+16, 2) != 0 {
		return false
	}
	mc := int(PeekBits(pay, p+18, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt {
		return false
	}
	if p+worldObjectHeaderBits+worldObjectIndexBits*mc > total {
		return false
	}
	_, ok := ascendingComponents(pay, p+worldObjectHeaderBits, mc)
	return ok
}

// ti13ChainAcc compte les issues d'une passe de chainage.
type ti13ChainAcc struct {
	records   int
	chained   int
	decale    int
	parMasque map[int]int // cardinalite du masque -> records
	parComp   map[int]int // composant annonce -> records chaines
	compTot   map[int]int // composant annonce -> records total
}

func newTi13ChainAcc() *ti13ChainAcc {
	return &ti13ChainAcc{parMasque: map[int]int{}, parComp: map[int]int{}, compTot: map[int]int{}}
}

func (a *ti13ChainAcc) taux(n int) float64 {
	if a.records == 0 {
		return 0
	}
	return 100 * float64(n) / float64(a.records)
}

// TestTi13ChainageLotCbis mesure le chainage des records ti=13 sous les deux hypotheses.
func TestTi13ChainageLotCbis(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	c := zcKeyframeCensus(dir)
	b := zcBuildBands(c)
	reelle := b.perTI[13]
	if len(reelle) == 0 {
		t.Skipf("aucun slot ti=13 en image-cle sur %s", short)
	}
	fantome := map[uint32]bool{}
	for slot, cl := range b.class {
		if cl == zcClassVide {
			fantome[slot] = true
		}
	}
	t.Logf("FILM %s — ti=13 : %d slots · FANTOME : %d slots · ti=4 (temoin positif) : %d slots",
		short, len(reelle), len(fantome), len(b.perTI[4]))

	var sb strings.Builder
	sb.WriteString("film\tbande\thypothese\trecords\tchaines\ttaux\tdecale\ttaux_decale\n")
	for _, h := range ti13Hyps {
		ti13ChainReport(t, &sb, short, "REELLE", h, ti13Chain(c, reelle, h))
		ti13ChainReport(t, &sb, short, "FANTOME", h, ti13Chain(c, fantome, h))
	}
	ti13ChainTemoin(t, &sb, short, c, b)
	zcWriteFile(t, filepath.Join(out, short+"_ti13_chainage.tsv"), sb.String())
}

// ti13Chain balaye le film et mesure le chainage sous une hypothese.
func ti13Chain(c zcCensus, band map[uint32]bool, h ti13Hyp) *ti13ChainAcc {
	a := newTi13ChainAcc()
	if len(band) == 0 {
		return a
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
			ti13ChainPacket(a, pk.Payload(data), band, h)
		}
	}
	return a
}

// ti13ChainPacket traite un paquet delta.
func ti13ChainPacket(a *ti13ChainAcc, pay []byte, band map[uint32]bool, h ti13Hyp) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok {
			continue
		}
		p = rec.After
		end, ok := ti13RecordEnd(pay, rec, h.maskedModeA)
		if !ok {
			continue // hors grammaire de l'archetype : ecarte, et le denominateur le dit
		}
		a.records++
		a.parMasque[len(rec.Idx)]++
		ch := ti13HeaderAt(pay, end)
		if ch {
			a.chained++
		}
		if ti13HeaderAt(pay, end+3) {
			a.decale++
		}
		for _, i := range rec.Idx {
			a.compTot[i]++
			if ch {
				a.parComp[i]++
			}
		}
	}
}

// ti13ChainReport publie le resultat d'une passe.
func ti13ChainReport(t *testing.T, sb *strings.Builder, short, bande string, h ti13Hyp,
	a *ti13ChainAcc,
) {
	t.Helper()
	t.Logf("")
	t.Logf("=== %s · hypothese i2..i33 = mode %s — %d records dans la grammaire",
		bande, h.nom, a.records)
	t.Logf("  CHAINES (un en-tete valide commence au bit de fin) : %d = %.1f %%",
		a.chained, a.taux(a.chained))
	t.Logf("  temoin DECALE (+3 bits, niveau du hasard de l'en-tete) : %d = %.1f %%",
		a.decale, a.taux(a.decale))
	t.Logf("  cardinalite du masque : %s", ti13MaskLine(a))
	if bande == "REELLE" {
		t.Logf("  chainage PAR COMPOSANT annonce : %s", ti13CompLine(a))
	}
	fmt.Fprintf(sb, "%s\t%s\t%s\t%d\t%d\t%.1f\t%d\t%.1f\n", short, bande, h.nom,
		a.records, a.chained, a.taux(a.chained), a.decale, a.taux(a.decale))
}

// ti13MaskLine rend la distribution des cardinalites de masque.
func ti13MaskLine(a *ti13ChainAcc) string {
	if a.records == 0 {
		return "(aucun record)"
	}
	keys := make([]int, 0, len(a.parMasque))
	for k := range a.parMasque {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%d:%d (%.1f %%) ", k, a.parMasque[k],
			100*float64(a.parMasque[k])/float64(a.records))
	}
	return strings.TrimSpace(sb.String())
}

// ti13CompLine rend, pour les composants les plus annonces, leur taux de chainage.
func ti13CompLine(a *ti13ChainAcc) string {
	keys := make([]int, 0, len(a.compTot))
	for k := range a.compTot {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return a.compTot[keys[i]] > a.compTot[keys[j]] })
	var sb strings.Builder
	for n, k := range keys {
		if n >= 6 {
			break
		}
		fmt.Fprintf(&sb, "i%d:%d/%d (%.0f %%) ", k, a.parComp[k], a.compTot[k],
			100*float64(a.parComp[k])/float64(a.compTot[k]))
	}
	return strings.TrimSpace(sb.String())
}

// ti13ChainTemoin joue le temoin POSITIF : ti=4, dont la grammaire est connue et pure
// (`high-frequency`, R(8) en variante FRAME, sonde F3). Un record ti=4 a UN composant declare,
// donc sa charge utile fait 8 bits et sa fin est calculable sans hypothese. Le taux de chainage
// obtenu la est le PLAFOND que ce test peut atteindre sur ce film.
func ti13ChainTemoin(t *testing.T, sb *strings.Builder, short string, c zcCensus, b zcBands) {
	t.Helper()
	band := b.perTI[4]
	if len(band) == 0 {
		t.Logf("")
		t.Logf("=== TEMOIN POSITIF ti=4 : aucun slot sur ce film")
		return
	}
	a := newTi13ChainAcc()
	for ch := 1; ch <= c.chunks; ch++ {
		data, err := ReadFilmChunk(c.dir, ch)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			ti13TemoinPacket(a, pk.Payload(data), band)
		}
	}
	t.Logf("")
	t.Logf("=== TEMOIN POSITIF ti=4 `high-frequency` (R(8), 1 composant) — %d records", a.records)
	t.Logf("  CHAINES : %d = %.1f %%  ·  temoin DECALE (+3) : %d = %.1f %%",
		a.chained, a.taux(a.chained), a.decale, a.taux(a.decale))
	fmt.Fprintf(sb, "%s\tTEMOIN_ti4\tR(8)\t%d\t%d\t%.1f\t%d\t%.1f\n", short,
		a.records, a.chained, a.taux(a.chained), a.decale, a.taux(a.decale))
}

// ti13TemoinPacket applique la grammaire de ti=4 (8 bits) a un paquet.
func ti13TemoinPacket(a *ti13ChainAcc, pay []byte, band map[uint32]bool) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok {
			continue
		}
		p = rec.After
		if len(rec.Idx) != 1 || rec.Idx[0] != 0 {
			continue
		}
		end := rec.After + 8
		if end > len(pay)*8 {
			continue
		}
		a.records++
		if ti13HeaderAt(pay, end) {
			a.chained++
		}
		if ti13HeaderAt(pay, end+3) {
			a.decale++
		}
	}
}
