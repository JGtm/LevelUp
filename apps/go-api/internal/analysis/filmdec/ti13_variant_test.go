package filmdec

// ti13_variant_test.go — LOT C-bis PHASE 0 : l'instrument qui confronte aux octets de film la
// grammaire du variant ti=13 lue dans HaloInfinite.exe (journal `LOTCBIS_PHASE0.md`).
//
// CE QU'IL N'EST PAS. Il ne PORTE rien : aucun `case` de `traverse.go`, aucun hook, aucune ligne
// de table ECS. Il lit les bits a la position que l'ancrage donne et applique la grammaire ecrite
// a la main ci-dessous. Le port est la phase 1, sur go du superviseur.
//
// POURQUOI LE MASQUE SINGLETON. Meme raison qu'au lot C (`zone_vectors_test.go`) : la traversee
// consomme les composants annonces DANS L'ORDRE, donc la position de la charge utile de X depend
// de la largeur de tous les composants annonces avant lui. Un record dont le masque vaut
// EXACTEMENT {X} n'a pas ce probleme — la charge utile de X commence a `rec.After`, le thunk
// +0x28 ne consommant rien (lot C phase 1a §2).
//
// LE TEMOIN EST LE COEUR DE LA MESURE, pas un ornement. La bande ti=13 est contaminee de 35 a
// 77 % (sonde F5) et tombe SOUS son fantome sur le temoin Slayer. Un tag de 4 bits lu sur du
// bruit d'ancrage est UNIFORME (6,25 % par valeur) ; lu sur des records reels il est CONCENTRE
// (une propriete a UN type, qui ne change pas). La meme passe tourne donc sur la bande reelle ET
// sur la bande fantome (`zcClassVide`), et les deux distributions sont publiees cote a cote.
//
// USAGE (depuis apps/go-api, UN film par processus, avant-plan) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/7344d24f"
//	go test -count=1 -run TestTi13VariantLotCbis -v ./internal/analysis/filmdec/

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// -------------------------------------------------------------------------------------
// LA GRAMMAIRE, lue dans le binaire (lecture seule Ghidra). Voir LOTCBIS_PHASE0.md §2.
// -------------------------------------------------------------------------------------

// LA TABLE DE LARGEURS A MIGRE DANS LE CODE DE PRODUCTION (phase 1, `components_managed_property.go`,
// `managedPropertyPayloadBits`) : ce fichier l APPELLE au lieu d en garder une copie, de sorte que
// les vecteurs figes testent la table PORTEE et non un double qui pourrait diverger.
//
// ti13ModeA / ti13ModeB : le variant `FUN_140ce59bc` lit R(4) puis dispatche sur `FUN_140ce5aa4`,
// qui consulte l'INDEX DE CHAMP porte par le contexte (ctx+0x10) :
//
//	i1   `managed-object-property-component`         -> index = 0xFFFFFFFF (mode A, scalaire)
//	i2.. `managed-object-player-masked-property-...` -> index = *(int*)(descripteur+8) = 0..31
//	                                                    (mode B, element #k d'un tableau)
//
// Les branches 1..6 ne lisent QUE si l'index est hors bornes (`>= 0x20`), les branches 7..11 QUE
// s'il est dans les bornes (`< 0x20`). Les deux modes sont donc disjoints, et la largeur totale
// est entierement determinee par le tag : la grammaire ne peut pas desynchroniser.
const (
	ti13ModeA = true
	ti13ModeB = false
)

// ti13Val est une valeur de variant lue dans le flux.
type ti13Val struct {
	tag     int
	payBits int
	payload uint64
}

// ti13Decode lit un variant a la position `bit` et rend la position juste apres.
func ti13Decode(pay []byte, bit int, modeA bool) (ti13Val, int, bool) {
	total := len(pay) * 8
	if bit+4 > total {
		return ti13Val{}, bit, false
	}
	v := ti13Val{tag: int(PeekBits(pay, bit, 4))}
	v.payBits = managedPropertyPayloadBits(v.tag, modeA)
	if bit+4+v.payBits > total {
		return ti13Val{}, bit, false
	}
	if v.payBits > 0 {
		v.payload = PeekBits(pay, bit+4, v.payBits)
	}
	return v, bit + 4 + v.payBits, true
}

// -------------------------------------------------------------------------------------
// L'INSTRUMENT
// -------------------------------------------------------------------------------------

// ti13Vec est un vecteur retenu : de quoi rejouer la lecture sans le film.
type ti13Vec struct {
	chunk, pkt, bitPay int
	slot, gen          uint32
	raw64              uint64 // les 64 bits bruts a partir de la charge utile (tag inclus)
	val                ti13Val
	// chained : un en-tete de record valide commence exactement au bit de fin calcule par la
	// grammaire (`ti13_chainage_test.go`). Un vecteur chaine porte sa propre preuve de largeur :
	// le flux lui-meme dit ou la valeur s'arrete. Les vecteurs figes sont choisis parmi eux.
	chained bool
}

// ti13Acc accumule ce qu'une passe observe sur UN composant d'UNE bande.
type ti13Acc struct {
	records int
	tags    [16]int
	perSlot map[uint32]map[int]int
	vecs    map[int][]ti13Vec
	vals    map[uint64]int
	// raws compte les 64 bits BRUTS lus a partir de la charge utile. C'est le detecteur de
	// MOTIF LITTERAL : un canal reel varie (chaque record porte un etat different), alors
	// qu'un faux positif d'ancrage rematche les MEMES octets a chaque passage. Un rapport
	// « valeurs distinctes / records » proche de 0 denonce l'artefact ; proche de 1, un canal.
	raws map[uint64]int
}

func newTi13Acc() *ti13Acc {
	return &ti13Acc{
		perSlot: map[uint32]map[int]int{}, vecs: map[int][]ti13Vec{},
		vals: map[uint64]int{}, raws: map[uint64]int{},
	}
}

// ti13VecMax est le nombre de vecteurs gardes par tag (le plan en demande 2 a 3).
const ti13VecMax = 3

func (a *ti13Acc) add(v ti13Vec) {
	a.records++
	a.tags[v.val.tag]++
	if a.perSlot[v.slot] == nil {
		a.perSlot[v.slot] = map[int]int{}
	}
	a.perSlot[v.slot][v.val.tag]++
	a.raws[v.raw64]++
	if v.val.payBits > 0 {
		a.vals[v.val.payload]++
	}
	lst := a.vecs[v.val.tag]
	if len(lst) < ti13VecMax {
		a.vecs[v.val.tag] = append(lst, v)
		return
	}
	if !v.chained {
		return
	}
	for k := range lst { // un vecteur CHAINE chasse un vecteur qui ne l'est pas
		if !lst[k].chained {
			lst[k] = v
			return
		}
	}
}

// ti13Cible designe un composant a mesurer.
type ti13Cible struct {
	i     int
	nom   string
	modeA bool
}

// ti13Cibles couvre TOUT l'archetype : i0 le NOM (R(32) fixe, deja porte), i1 le variant
// scalaire (mode A), i2..i33 les 32 elements par joueur (mode B). Cibler la liste a la main
// aurait fait manquer les composants qui parlent en KOTH (i2..i9) au profit de ceux qui parlent
// en Strongholds (i13, i17, i21) — or ce ne sont pas les memes, et l'un des deux groupes est
// de la contamination. Le rapport filtre ensuite sur le volume observe.
var ti13Cibles = func() []ti13Cible {
	l := []ti13Cible{{i: 1, nom: "managed-object-property-component", modeA: ti13ModeA}}
	for i := 2; i <= ti13MaxComp; i++ {
		l = append(l, ti13Cible{
			i: i, nom: fmt.Sprintf("player-masked-property (i%d)", i), modeA: ti13ModeB,
		})
	}
	return l
}()

// ti13MinRecords est le volume minimal pour qu'un composant soit detaille au rapport. En
// dessous, la distribution des tags n'a pas de sens : 10 records sur 16 tags ne mesurent rien.
const ti13MinRecords = 20

// TestTi13VariantLotCbis joue la grammaire sur la bande reelle ti=13 et sur la bande fantome.
func TestTi13VariantLotCbis(t *testing.T) {
	dir := zcDir(t)
	out := zcOutDir(t)
	short := filepath.Base(dir)
	release := LockProcessDecode()
	defer release()

	c := zcKeyframeCensus(dir)
	b := zcBuildBands(c)
	reelle := b.perTI[13]
	fantome := map[uint32]bool{}
	for slot, cl := range b.class {
		if cl == zcClassVide {
			fantome[slot] = true
		}
	}
	t.Logf("FILM %s — bande ti=13 : %d slots ; bande FANTOME (temoin) : %d slots",
		short, len(reelle), len(fantome))
	if len(reelle) == 0 {
		t.Skipf("aucun slot ti=13 en image-cle sur %s", short)
	}

	var sb strings.Builder
	sb.WriteString("bande\ti\tcomposant\ttag\tchunk\tpaquet\tbit_charge\tslot\tgen\t" +
		"bits_bruts_hex\tpay_bits\tpayload\n")
	reel := ti13Scan(c, reelle)
	faux := ti13Scan(c, fantome)
	ti13Report(t, &sb, short, reel, faux)
	ti13ReportNom(t, &sb, short, reel, faux)
	zcWriteFile(t, filepath.Join(out, short+"_ti13_variant.tsv"), sb.String())
}

// ti13Scan balaye tous les paquets delta et rend, par composant cible, ce que la grammaire lit
// sur les records a masque SINGLETON. `accNom` (cle -1) porte i0 `property-name` (R(32) fixe).
func ti13Scan(c zcCensus, band map[uint32]bool) map[int]*ti13Acc {
	acc := map[int]*ti13Acc{-1: newTi13Acc()}
	for _, cb := range ti13Cibles {
		acc[cb.i] = newTi13Acc()
	}
	if len(band) == 0 {
		return acc
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
			ti13ScanPacket(acc, data, pk, ch, band)
		}
	}
	return acc
}

// ti13ScanPacket traite un paquet delta.
func ti13ScanPacket(acc map[int]*ti13Acc, data []byte, pk FilmPacket, ch int, band map[uint32]bool) {
	pay := pk.Payload(data)
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok {
			continue
		}
		p = rec.After
		if len(rec.Idx) != 1 {
			continue // masque non singleton : la position de la charge utile est inconnue
		}
		ti13ScanRecord(acc, pay, rec, ch, pk.Index)
	}
}

// ti13ScanRecord applique la grammaire au record singleton `rec`.
func ti13ScanRecord(acc map[int]*ti13Acc, pay []byte, rec WorldObjectRecord, ch, pkt int) {
	idx := rec.Idx[0]
	base := ti13Vec{chunk: ch, pkt: pkt, bitPay: rec.After, slot: rec.Slot, gen: rec.Gen}
	if rec.After+64 <= len(pay)*8 {
		base.raw64 = PeekBits(pay, rec.After, 64)
	}
	if idx == 0 { // i0 `property-name` : R(32) fixe, deja porte (FUN_142ed69d8)
		if rec.After+32 > len(pay)*8 {
			return
		}
		base.val = ti13Val{tag: 0, payBits: 32, payload: PeekBits(pay, rec.After, 32)}
		base.chained = ti13HeaderAt(pay, rec.After+32)
		acc[-1].add(base)
		return
	}
	for _, cb := range ti13Cibles {
		if cb.i != idx {
			continue
		}
		v, end, ok := ti13Decode(pay, rec.After, cb.modeA)
		if !ok {
			return
		}
		base.val, base.chained = v, ti13HeaderAt(pay, end)
		acc[idx].add(base)
		return
	}
}

// -------------------------------------------------------------------------------------
// RAPPORT
// -------------------------------------------------------------------------------------

// ti13Report publie, par cible, la distribution des tags sur la bande reelle et sur le fantome,
// la coherence de tag par slot, et les vecteurs.
func ti13Report(t *testing.T, sb *strings.Builder, short string, reel, faux map[int]*ti13Acc) {
	t.Helper()
	for _, cb := range ti13Cibles {
		r, f := reel[cb.i], faux[cb.i]
		if cb.i != 1 && r.records < ti13MinRecords {
			continue // volume trop faible pour que la distribution des tags dise quoi que ce soit
		}
		t.Logf("")
		t.Logf("=== ti=13 i%d %s (mode %s) — %d records singleton (REELLE) / %d (FANTOME)",
			cb.i, cb.nom, ti13ModeNom(cb.modeA), r.records, f.records)
		t.Logf("  tags REELLE  : %s", ti13TagLine(r))
		t.Logf("  tags FANTOME : %s", ti13TagLine(f))
		t.Logf("  concentration du tag dominant : REELLE %.1f %% · FANTOME %.1f %%"+
			"  (uniforme = 6,3 %%)", ti13TopShare(r), ti13TopShare(f))
		t.Logf("  coherence du tag PAR SLOT (>= 5 records) : REELLE %s · FANTOME %s",
			ti13SlotCoherence(r), ti13SlotCoherence(f))
		t.Logf("  64 bits bruts DISTINCTS / records (motif litteral si ~0) : REELLE %d/%d = %s"+
			" · FANTOME %d/%d = %s", len(r.raws), r.records, ti13Ratio(len(r.raws), r.records),
			len(f.raws), f.records, ti13Ratio(len(f.raws), f.records))
		ti13WriteVecs(t, sb, short, cb, r)
	}
}

// ti13ChainMark marque un vecteur dont la largeur est confirmee par le flux lui-meme.
func ti13ChainMark(chained bool) string {
	if chained {
		return "[CHAINE]"
	}
	return "[non chaine]"
}

func ti13ModeNom(modeA bool) string {
	if modeA {
		return "A/scalaire"
	}
	return "B/par-joueur"
}

// ti13TagLine rend la distribution des 16 tags, les plus frequents d'abord.
func ti13TagLine(a *ti13Acc) string {
	if a.records == 0 {
		return "(aucun record)"
	}
	type kv struct{ tag, n int }
	var l []kv
	for tag, n := range a.tags {
		if n > 0 {
			l = append(l, kv{tag, n})
		}
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	var sb strings.Builder
	for i, e := range l {
		if i >= 8 {
			fmt.Fprintf(&sb, "(+%d autres)", len(l)-8)
			break
		}
		fmt.Fprintf(&sb, "t%d:%d (%.1f %%) ", e.tag, e.n, 100*float64(e.n)/float64(a.records))
	}
	return strings.TrimSpace(sb.String())
}

// ti13TopShare rend la part du tag le plus frequent.
func ti13TopShare(a *ti13Acc) float64 {
	if a.records == 0 {
		return 0
	}
	best := 0
	for _, n := range a.tags {
		if n > best {
			best = n
		}
	}
	return 100 * float64(best) / float64(a.records)
}

// ti13SlotCoherence rend la part MEDIANE du tag dominant par slot, sur les slots ayant au moins
// 5 records. C'est le discriminant structurel : une propriete a UN type, donc un slot reel doit
// rendre TOUJOURS le meme tag ; du bruit rend une valeur proche de l'uniforme.
func ti13SlotCoherence(a *ti13Acc) string {
	var parts []float64
	for _, tags := range a.perSlot {
		tot, best := 0, 0
		for _, n := range tags {
			tot += n
			if n > best {
				best = n
			}
		}
		if tot >= 5 {
			parts = append(parts, 100*float64(best)/float64(tot))
		}
	}
	if len(parts) == 0 {
		return "(aucun slot a >= 5 records)"
	}
	sort.Float64s(parts)
	return fmt.Sprintf("mediane %.1f %% sur %d slots", parts[len(parts)/2], len(parts))
}

// ti13WriteVecs journalise et ecrit les vecteurs d'une cible, tag par tag.
func ti13WriteVecs(t *testing.T, sb *strings.Builder, short string, cb ti13Cible, a *ti13Acc) {
	t.Helper()
	tags := make([]int, 0, len(a.vecs))
	for tag := range a.vecs {
		tags = append(tags, tag)
	}
	sort.Ints(tags)
	for _, tag := range tags {
		for _, v := range a.vecs[tag] {
			t.Logf("  VECTEUR i%d tag %d %s : chunk %d · paquet %d · bit %d · slot %d gen %d"+
				" · bruts 0x%016X · charge %d bits = %d",
				cb.i, tag, ti13ChainMark(v.chained), v.chunk, v.pkt, v.bitPay, v.slot, v.gen,
				v.raw64, v.val.payBits, v.val.payload)
			fmt.Fprintf(sb, "reelle\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%016X\t%d\t%d\t%s\n",
				cb.i, cb.nom, tag, v.chunk, v.pkt, v.bitPay, v.slot, v.gen, v.raw64,
				v.val.payBits, v.val.payload, ti13ChainMark(v.chained))
		}
	}
	_ = short
}

// ti13ReportNom publie le verdict sur i0 `property-name` : un identifiant de chaine a un
// VOCABULAIRE (peu de valeurs, tres repetees). Le rapport valeurs distinctes / emissions est
// compare a celui du FANTOME, ou il vaut mecaniquement ~1,0.
func ti13ReportNom(t *testing.T, sb *strings.Builder, short string, reel, faux map[int]*ti13Acc) {
	t.Helper()
	r, f := reel[-1], faux[-1]
	t.Logf("")
	t.Logf("=== ti=13 i0 managed-object-property-name (R(32), porte) — %d emissions (REELLE)"+
		" / %d (FANTOME)", r.records, f.records)
	t.Logf("  valeurs distinctes : REELLE %d (ratio %s) · FANTOME %d (ratio %s)",
		len(r.vals), ti13Ratio(len(r.vals), r.records), len(f.vals), ti13Ratio(len(f.vals), f.records))
	t.Logf("  valeurs les plus frequentes (REELLE) : %s", ti13TopVals(r))
	t.Logf("  valeurs les plus frequentes (FANTOME): %s", ti13TopVals(f))
	for _, v := range r.vecs[0] {
		fmt.Fprintf(sb, "reelle\t0\tproperty-name\t-\t%d\t%d\t%d\t%d\t%d\t%016X\t32\t%d\n",
			v.chunk, v.pkt, v.bitPay, v.slot, v.gen, v.raw64, v.val.payload)
	}
	_ = short
}

func ti13Ratio(distinct, total int) string {
	if total == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", float64(distinct)/float64(total))
}

// ti13TopVals rend les 5 valeurs les plus frequentes.
func ti13TopVals(a *ti13Acc) string {
	if len(a.vals) == 0 {
		return "(aucune)"
	}
	keys := make([]uint64, 0, len(a.vals))
	for k := range a.vals {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return a.vals[keys[i]] > a.vals[keys[j]] })
	var sb strings.Builder
	for i, k := range keys {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "0x%08X:%d (%.1f %%) ", k, a.vals[k],
			100*float64(a.vals[k])/float64(a.records))
	}
	return strings.TrimSpace(sb.String())
}
