package grammar

// instruments_partages_geo_test.go — la geometrie des tirs et des morts (`geo*`) des
// instruments explosifs, dont se sert la garde `weapon_index_groundtruth_test.go`.
// Deplaces tels quels au J12.7 bis depuis les fichiers tagues `research` (decision DU-5 : le
// tag cache les instruments, jamais une garde) ; chaque declaration garde le corps et le
// commentaire de son fichier d'origine.

import (
	"fmt"
	"strings"
	"testing"

	"levelup/go-api/internal/games/weapons/filmshell"
)

// geoActiveBase : base bipede DETECTEE du film courant (ref dom1 -> slot absolu = base+idx).
// Fixee par TestGeoExplosifs apres le sweep geoDetectBase ; 512 par defaut. Le paquet serialise
// le decodage, donc un etat de test est sur : la production, elle, n en a plus.
var geoActiveBase = geoBase

const (
	geoBase      = lot1chReferenceBase // base bipede par defaut (512, calibree 000d5950)
	geoFlightW   = uint64(2_000_000)   // 2 s : fenetre de vol tir lourd -> impact
	geoMatchWin  = uint64(400_000)     // 400 ms : touche fatale <-> mort (dead-state)
	geoPosTolUS  = sondePosTolUS       // 120 ms : ecart max evenement <-> echantillon position
	geoMinDtUS   = uint64(30_000)      // 30 ms : dt de vol minimal pour calibrer une vitesse
	geoDefSpeedU = 45.0                // vitesse projectile par defaut (unites monde/s) si non calibree
)

// geoShot : un tir 0xD2 t36 LONG horodate. att = attaquant (ref0 dom1 brut, MEME espace que
// ref1 d'un damage_aftermath -> slot via geoBase) ; film = FilmIndex (index participant, MEME
// espace que EnumB du dead-state) ; wid = WeaponID ; heavy/direct classent l'arme.
type geoShot struct {
	ts     uint64
	att    uint64
	film   int
	wid    uint64
	name   string
	heavy  bool
	direct bool // projectile a trajectoire DIRECTE (roquette/empaleur/ravageur...) vs traqueur/cloche
}

// geoKill : un dead-state de bipede mort, oracle. killer = EnumB (roster), victim = EnumA.
type geoKill struct {
	victSlot uint32
	victRost int32
	killer   int32
	ts       uint64
}

// geoIsDirect : l'arme lourde tire-t-elle un projectile a trajectoire DIRECTE (l'alignement de
// visee est net) ? Faux pour les traqueurs (Hydra en verrouillage) et les tirs en cloche
// (Fuel Rod), ou la visee ne pointe pas la victime.
func geoIsDirect(name string) bool {
	for _, k := range []string{"SPNKr", "Skewer", "Shock", "Mangler", "Stalker", "Bulldog"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false // Hydra (traqueur), Ravager/Fuel Rod/Rod (cloche) -> non direct
}

// geoWeaponName : nom d'arme par WeaponID (table statique) ou l'hexa.
func geoWeaponName(wid uint64) string {
	if nm, ok := filmshell.WeaponIDToName[wid]; ok {
		return nm
	}
	return attribWeaponName(wid)
}

// geoCollectDamageKills fait UNE passe par chunk : bind keyframes, rejoue la trame (qui capture
// les dead-states = oracle), puis decode les damage_aftermath 0xC0 contre le monde de fin de
// chunk en resolvant sous TOUTES les bases candidates. Rend les degats bruts et les morts.
func geoCollectDamageKills(t *testing.T, dir string, reg *Registry, n int) ([]geoRawDmg, []geoKill) {
	t.Helper()
	cfg := DefaultFrameConfig()
	var raws []geoRawDmg
	var kills []geoKill
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 == 0 {
				br := LecteurSur(pay)
				recs, _ := DecodeFrameRecords(br, w, cfg)
				kills = geoHarvestKills(recs, pk.TimestampUS, kills)
			}
		}
		raws = geoRawDamageInChunk(pks, data, w, raws)
	}
	return raws, kills
}

// geoDetectBase rend la base bipede du film : celle qui resout le PLUS de refs de degat en
// bipede (comme sondeBaseSweep). geoBase (512) en repli si le sweep est vide.
func geoDetectBase(raws []geoRawDmg) int {
	bestBase, bestHits := geoBase, -1
	for bi, b := range lot1chBases {
		if b < 400 {
			continue // seule la bande bipede est pertinente
		}
		hits := 0
		for _, e := range raws {
			if e.bip0[bi] || e.bip1[bi] {
				hits++
			}
		}
		if hits > bestHits {
			bestBase, bestHits = b, hits
		}
	}
	return bestBase
}

// geoMaxChunks borne le balayage (RAM : un film BTB est gros). 16 = compromis arene/BTB.
const geoMaxChunks = 16

// geoBuildIdentity apprend la table roster(EnumA/B) -> FilmIndex : chaque mort lie la victime
// (slot -> FilmIndex via ses tirs) a son roster EnumA. Rend (table, cardinalite, injective?).
func geoBuildIdentity(shots []geoShot, kills []geoKill) (map[int32]int, int, bool) {
	slotFilm := map[uint32]map[int]int{}
	for _, s := range shots {
		slot := uint32(geoActiveBase + int(s.att))
		if slotFilm[slot] == nil {
			slotFilm[slot] = map[int]int{}
		}
		slotFilm[slot][s.film]++
	}
	argmax := func(m map[int]int) (int, bool) {
		best, bn, ok := 0, -1, false
		for f, n := range m {
			if n > bn {
				best, bn, ok = f, n, true
			}
		}
		return best, ok
	}
	rosterVotes := map[int32]map[int]int{}
	for _, k := range kills {
		if fm, ok := slotFilm[k.victSlot]; ok {
			if f, ok2 := argmax(fm); ok2 {
				if rosterVotes[k.victRost] == nil {
					rosterVotes[k.victRost] = map[int]int{}
				}
				rosterVotes[k.victRost][f]++
			}
		}
	}
	table := map[int32]int{}
	filmUsed := map[int]int32{}
	injective := true
	for r, m := range rosterVotes {
		if f, ok := argmax(m); ok {
			table[r] = f
			if prev, seen := filmUsed[f]; seen && prev != r {
				injective = false
			}
			filmUsed[f] = r
		}
	}
	return table, len(table), injective
}

// lot1IsHeavy : l'arme est-elle une arme lourde/explosif/faisceau (0 % en type 0) ?
func lot1IsHeavy(name string) bool {
	for _, k := range []string{"SPNKr", "Hydra", "Skewer", "Ravager", "Shock", "Mangler",
		"Stalker", "Bulldog", "Fuel Rod", "Rod"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}

// itoa : petit entier -> chaine (evite d'importer strconv pour deux usages).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// geoRawDmg : un damage_aftermath brut, avec la resolution bipede de ses deux refs sous CHAQUE
// base candidate (la base bipede est propre au film, comme la sonde de precision la calibre).
type geoRawDmg struct {
	ts         uint64
	idx0, idx1 int
	mag        float64
	bip0, bip1 []bool // aligne sur lot1chBases
}

// geoHarvestKills recolte les dead-states de bipede mort (victime slot + roster EnumA + tueur EnumB).
func geoHarvestKills(recs []FrameRecord, ts uint64, kills []geoKill) []geoKill {
	for i := range recs {
		rec := &recs[i]
		if rec.TypeIndex != BipedTypeIndex || rec.Trace.Dead == nil {
			continue
		}
		d := rec.Trace.Dead
		if !d.Mort || d.EnumB < 0 {
			continue
		}
		kills = append(kills, geoKill{victSlot: rec.Slot, victRost: d.EnumA, killer: d.EnumB, ts: ts})
	}
	return kills
}

// geoRawDamageInChunk decode les 0xC0 t0 d'un chunk et resout ref0/ref1 en bipede sous CHAQUE
// base candidate (la selection de base est faite globalement, apres la collecte).
func geoRawDamageInChunk(pks []FilmPacket, data []byte, w *World, out []geoRawDmg) []geoRawDmg {
	for _, pk := range pks {
		if pk.Type != PacketTypeDelta || pk.Size < 2 {
			continue
		}
		pay := pk.Payload(data)
		if pay[0] != 0xC0 {
			continue
		}
		br := LecteurSur(pay)
		br.Skip(2)
		if br.ReadBits(7) != 0 {
			continue
		}
		i0, ok0 := lot1RefDom1(br)
		i1, ok1 := lot1RefDom1(br)
		lot1RefDom(br, 7)
		r := lot1DecodeDamageAftermath(br)
		if !ok1 {
			continue
		}
		e := geoRawDmg{ts: pk.TimestampUS, idx0: -1, idx1: int(i1), mag: r.dmgClear,
			bip0: make([]bool, len(lot1chBases)), bip1: make([]bool, len(lot1chBases))}
		if ok0 {
			e.idx0 = int(i0)
		}
		for bi, b := range lot1chBases {
			e.bip0[bi] = geoResolveBip(w, b, e.idx0)
			e.bip1[bi] = geoResolveBip(w, b, e.idx1)
		}
		out = append(out, e)
	}
	return out
}

// attribWeaponName nomme une arme par son WeaponID (metadata weapon_labels) ; a defaut, l'hexa.
func attribWeaponName(wid uint64) string {
	if n, ok := filmshell.WeaponIDToName[wid]; ok {
		return n
	}
	return fmt.Sprintf("wid#%016x", wid)
}

// lot1chReferenceBase : base de la bande bipede etablie par l'instrument A (calibration par
// la vitalite, base a couverture max = 512 sur les trois films temoins). Sert de reference
// commune pour le AVANT/APRES quand l'argmax "monde" tombe sur une base voisine (bande contigue).
const lot1chReferenceBase = 512

// sondePosTolUS : tolerance temporelle evenement<->position. MEME valeur que
// replay/shots.go shotPosToleranceUS (120 ms) ; recopiee ici pour rester DANS filmdec
// (pas d'import de internal/games/halo_infinite/film/replay depuis un instrument de filmdec).
const sondePosTolUS = uint64(120_000)

// geoResolveBip rend vrai si (base+idx) est un bipede lie dans le monde w.
func geoResolveBip(w *World, base, idx int) bool {
	if idx < 0 {
		return false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}
