package filmdec

// r9_i22_signal_research_test.go — LE SIGNAL DE COMMANDE DE L'EQUIPEMENT et LE RECENSEMENT DU
// MASQUE ti=37 (par. 7 du RAPPORT_R9_REPULSEUR_2026-09-03).
//
// LA DECOUVERTE QUI OUVRE CETTE MESURE (Ghidra, ce lot). L'archetype ti=37 compte 31
// composants ; `equipment_state.go` n'en PUBLIE que six (i20 deployed, i21 activated,
// i23 creator, i24 energy, i26 energy-delay, i27 charges). Les cinq autres sont lus puis
// JETES par `consumeByName` — exactement le defaut corrige pour i48 le 2026-08-14 et pour les
// quatre champs de ti=37 le 2026-08-15. Parmi eux :
//
//	i22 equipment-control-signal-component   FUN_14101cd94 : R(4) + R(1)[+quantStat]
//	i28 equipment-tracked-object-handles-stack-component   FUN_140f72dec
//	i29 equipment-command-tick-component     FUN_140e0a564
//
// **« control signal » est le nom que le moteur donne a la COMMANDE**, et sa charge est un
// `R(4)` — seize valeurs, la forme d'un enumere de commande. C'est le seul champ de
// l'archetype dont le NOM annonce un ordre plutot qu'un etat, et personne ne l'a jamais lu.
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	bavardage      le composant doit etre annonce sur assez de records pour porter un signal
//	               (>= 100 annonces par film) ; sous ce seuil, aucun verdict ne sera prononce.
//	concentration  une valeur du R(4) « est » un usage si sa distribution par IDENTITE
//	               d'objet s'ecarte du bruit : >= 75 % de ses occurrences sur une famille.
//	temoin positif les entites de famille `grapple` et `thruster` — dont les instants d'usage
//	               sont connus — doivent emettre ce signal. S'ils ne l'emettent pas, le canal
//	               est disqualifie comme l'a ete celui des charges (par. 3 du rapport).
//
// LIMITE CONNUE ET DITE : le reconnaisseur de records ti=37 exige `Idx[0] == 0` (la position
// ouvre le masque). Un equipement EN MAIN qui ne replique pas sa propre position est donc
// invisible a ce balayage, comme il l'etait au balayage des charges. La mesure porte sur ce
// que le canal LAISSE VOIR, et son denominateur est publie.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c go test ./internal/analysis/filmdec/ \
//	  -run '^TestR9I22Signal$' -count=1 -timeout 180m -v

import (
	"fmt"
	"path/filepath"
	"sort"
	"testing"
)

const (
	r9CompControlSignal = "equipment-control-signal-component"
	r9CompTrackedStack  = "equipment-tracked-object-handles-stack-component"
)

// r9Signal est UNE lecture d'i22 sur un record ti=37.
type r9Signal struct {
	Ent  r9Ent
	TSUS uint64
	Sig  uint32 // le R(4) : la commande
	Bit  bool   // la porte R(1) qui suit
}

// r9SigScan balaye les records ti=37 et lit i22 a la main. C'est un MIROIR de
// `equipmentWalk.walk` (equipment_state.go) : meme marche, memes desers de production, mais il
// s'arrete AVANT i22 pour en lire la charge au lieu de la jeter. Aucune largeur ne change.
func r9SigScan(t *testing.T, dir string) ([]r9Signal, map[int]int, int) {
	t.Helper()
	n := CountFilmChunks(dir)
	band := worldObjectSlotBand(dir, n, EquipmentTypeIndex)
	arch, err := EquipmentArchetype(dir)
	if err != nil {
		t.Fatalf("archetype ti=37 illisible : %v", err)
	}
	idx22 := -1
	if ids := arch.indicesOf(r9CompControlSignal); len(ids) > 0 {
		idx22 = ids[0]
	}
	if idx22 < 0 {
		t.Fatalf("%s absent de l'archetype ti=37 de %s", r9CompControlSignal, dir)
	}
	var out []r9Signal
	census := map[int]int{}
	records := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			out, records = r9SigPayload(pay, band, arch, idx22, pk, census, out, &records)
		}
	}
	return out, census, records
}

// r9SigPayload traite UN payload (extrait pour tenir la limite de 80 lignes par fonction).
func r9SigPayload(pay []byte, band map[uint32]bool, arch Archetype, idx22 int,
	pk FilmPacket, census map[int]int, out []r9Signal, records *int) ([]r9Signal, int) {
	total := len(pay) * 8
	limit := total - (worldObjectHeaderBits + worldObjectIndexBits + projPosBits())
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok || rec.Idx[0] != 0 {
			continue
		}
		*records++
		for _, id := range rec.Idx {
			census[id]++
		}
		if maskHas(rec.Idx, idx22) {
			if at := r9SigOffset(pay, rec, total, arch, idx22); at >= 0 && at+5 <= total {
				br := NewBitReader(pay)
				br.SetBitPos(at)
				out = append(out, r9Signal{Ent: r9Ent{Slot: rec.Slot, Gen: rec.Gen},
					TSUS: pk.TimestampUS, Sig: uint32(br.ReadBits(4)), Bit: br.ReadBit()})
			}
		}
		p = rec.After - 1
	}
	return out, *records
}

// r9SigOffset marche les composants du masque qui precedent i22 avec les desers de PRODUCTION
// et rend la position de bit ou i22 commence, ou -1.
func r9SigOffset(pay []byte, rec WorldObjectRecord, total int, arch Archetype, target int) int {
	at := rec.After
	for _, id := range rec.Idx {
		if at > total {
			return -1
		}
		if id == target {
			return at
		}
		name := arch.component(id)
		if name == "" {
			return -1
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(EquipmentTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return -1
		}
		at = br.BitPos()
	}
	return -1
}

func TestR9I22Signal(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r9I22OneFilm(t, dir)
	}
}

func r9I22OneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	pl, _, err := ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("poses ti=37 illisibles : %v", err)
	}
	fam := map[r9Ent]string{}
	for _, p := range pl {
		fam[p.Life] = r8FamilyOf(p.GlobalID)
	}
	sigs, census, records := r9SigScan(t, dir)
	arch, _ := EquipmentArchetype(dir)
	t.Logf("%s : records ti=37 = %d | lectures i22 = %d | poses nommees = %d",
		filepath.Base(dir), records, len(sigs), len(fam))
	r9LogCensus(t, census, arch, records)
	r9LogSignals(t, sigs, fam)
}

// r9LogCensus publie LE RECENSEMENT DU MASQUE de ti=37 : un composant jamais annonce ne
// portera jamais rien, quel que soit le soin mis a decoder sa valeur.
func r9LogCensus(t *testing.T, census map[int]int, arch Archetype, records int) {
	t.Helper()
	keys := make([]int, 0, len(census))
	for k := range census {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	t.Logf("  recensement du masque ti=37 (%d records)", records)
	for _, k := range keys {
		t.Logf("    i%-3d %-52s %7d", k, arch.component(k), census[k])
	}
}

// r9LogSignals croise la valeur du R(4) avec l'IDENTITE de l'objet.
func r9LogSignals(t *testing.T, sigs []r9Signal, fam map[r9Ent]string) {
	t.Helper()
	parSig := map[string]map[string]int{}
	ents := map[string]map[r9Ent]bool{}
	for _, s := range sigs {
		k := fmt.Sprintf("sig=%2d bit=%v", s.Sig, s.Bit)
		f := fam[s.Ent]
		if f == "" {
			f = "?"
		}
		if parSig[k] == nil {
			parSig[k] = map[string]int{}
			ents[k] = map[r9Ent]bool{}
		}
		parSig[k][f]++
		ents[k][s.Ent] = true
	}
	keys := make([]string, 0, len(parSig))
	for k := range parSig {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("  i22 `equipment-control-signal` — valeur du R(4) x identite de l'objet")
	for _, k := range keys {
		var parts []string
		fams := make([]string, 0, len(parSig[k]))
		for f := range parSig[k] {
			fams = append(fams, f)
		}
		sort.Strings(fams)
		for _, f := range fams {
			parts = append(parts, fmt.Sprintf("%s x%d", f, parSig[k][f]))
		}
		t.Logf("    %-18s lectures=%-6d entites=%-5d %v", k, r9SumMap(parSig[k]),
			len(ents[k]), parts)
	}
}

func r9SumMap(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
