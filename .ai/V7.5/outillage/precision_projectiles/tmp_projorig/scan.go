package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

// scanFires releve les records de TIR (type 105 porteurs d'un identifiant d'arme).
// Grammaire reprise de `tmp_pjcnt` : un record se reconnait a son premier octet
// (`pay[0]>>1 == 105`), pas par un marqueur cherche au bit pres.
func scanFires(dir string) []fire {
	n := filmdec.CountFilmChunks(dir)
	var out []fire
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
				continue
			}
			pay := p.Payload(chunk)
			if int(pay[0]>>1) != recType {
				continue
			}
			bits := len(pay) * 8
			if bits < minBitsHead || pay[0]&1 != 0 || bits < minBitsWeap {
				continue // record court : pas d'identifiant d'arme
			}
			w := filmdec.PeekBits(pay, bitWeaponHi, widthWeapon)<<32 |
				filmdec.PeekBits(pay, bitWeaponLo, widthWeapon)
			if uint32(w) != commonWeaponSuffix {
				continue // hors population de TIR
			}
			// L'identifiant de `weapon_labels` est le mot de 64 bits COMPLET (sa moitie basse
			// est le suffixe partage) : ne pas decaler, sinon plus rien ne se nomme.
			out = append(out, fire{
				tsUS:    p.TimestampUS,
				weapon:  w,
				shooter: int(filmdec.PeekBits(pay, bitShooter, widthShoot)),
			})
		}
	}
	return out
}

// grenadeWeaponID est une cle SYNTHETIQUE (hors espace des identifiants d'arme) sous laquelle
// on range les LANCERS DE GRENADE. Ils sont le CONTROLE POSITIF de tout ce programme : la
// production a mesure 70 appariements lancer -> naissance sur 70. Si MON detecteur de
// naissances ne les retrouve pas, c'est lui qui est en cause, et le 0,08 des armes ne vaut
// rien. S'il les retrouve, le 0,08 est un vrai negatif.
const grenadeWeaponID = ^uint64(0)

// scanThrows releve les lancers de grenade avec le decodeur de PRODUCTION.
func scanThrows(dir string) []fire {
	th, err := filmdec.ScanFilmGrenadeThrows(dir)
	if err != nil {
		return nil
	}
	out := make([]fire, 0, len(th))
	for _, g := range th {
		out = append(out, fire{tsUS: g.TimestampUS, weapon: grenadeWeaponID, shooter: g.FilmIndex})
	}
	return out
}

// scanBirthsProd releve la PREMIERE position de chaque vie, avec le DECODEUR DE PRODUCTION.
//
// POURQUOI CE DETOUR, ET C'EST LA LECON DE CE PROGRAMME. Ma premiere version reecrivait la
// grammaire en laissant tomber deux filtres de `filmdec/projectiles.go` : la validite de la
// position decodee (porte a zero + quanta non satures) et le minimum de trois points par vie.
// Resultat : 3 fois trop de naissances, un taux de coincidence qui noie tout, et un CONTROLE
// POSITIF A 0,06 la ou la production mesure 0,93. **C'etait un negatif sur l'instrument.**
// On appelle donc la production directement — la population devient exactement celle qui a
// donne les 70 appariements sur 70.
func scanBirthsProd(dir string, wr *filmdec.Vec3Range) []birth {
	tracks, err := filmdec.ScanFilmProjectiles(dir, wr)
	if err != nil {
		return nil
	}
	out := make([]birth, 0, len(tracks))
	for _, t := range tracks {
		if len(t.Pts) == 0 {
			continue
		}
		out = append(out, birth{tsUS: t.Pts[0].TimestampUS, slot: t.Slot})
	}
	return out
}

// scanBirths releve la PREMIERE position de chaque vie d'entite `ti=41`. La grammaire de
// record est celle de `filmdec/projectiles.go` — celle qui porte les 70 lancers sur 70.
func scanBirths(dir string) []birth {
	n := filmdec.CountFilmChunks(dir)
	if n == 0 {
		return nil
	}
	band := slotBand(dir, n)
	if len(band) == 0 {
		return nil
	}
	type key struct{ slot, gen uint32 }
	seen := map[key][]uint64{}
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta {
				continue
			}
			for _, s := range scanProjRecords(p.Payload(chunk), band) {
				k := key{s.slot, s.gen}
				seen[k] = append(seen[k], p.TimestampUS)
			}
		}
	}
	// Le pool de slots reboucle et la generation ne fait que 2 bits : un trou de plus de
	// 250 ms est une frontiere de VIE, pas une lacune. Chaque segment donne UNE naissance.
	var out []birth
	for k, ts := range seen {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		out = append(out, birth{tsUS: ts[0], slot: k.slot})
		for i := 1; i < len(ts); i++ {
			if ts[i]-ts[i-1] > lifeGapUS {
				out = append(out, birth{tsUS: ts[i], slot: k.slot})
			}
		}
	}
	return out
}

type projRec struct{ slot, gen uint32 }

// scanProjRecords : la grammaire de `scanProjectileRecords`, sans la deqantification (on ne
// veut ici que l'instant et l'identite de la vie).
func scanProjRecords(pay []byte, band map[uint32]bool) []projRec {
	var out []projRec
	limit := len(pay)*8 - (21 + 6 + hiPosBits)
	for p := 0; p <= limit; p++ {
		if filmdec.PeekBits(pay, p, 1) != 1 {
			continue
		}
		slot := uint32(filmdec.PeekBits(pay, p+1, 13))
		if !band[slot] {
			continue
		}
		gen := uint32(filmdec.PeekBits(pay, p+14, 2))
		if filmdec.PeekBits(pay, p+16, 2) != 0 {
			continue
		}
		mc := int(filmdec.PeekBits(pay, p+18, 3))
		if mc < 1 || mc > 7 {
			continue
		}
		idx, ok := ascending(pay, p+21, mc)
		if !ok || idx[0] != 0 {
			continue
		}
		at := p + 21 + 6*mc
		if filmdec.PeekBits(pay, at, 3) != 0 {
			// Branche opaque ou index de region non nul : on ne sait pas la decoder, mais
			// le record EXISTE — on le compte comme presence de la vie.
			out = append(out, projRec{slot, gen})
			p += hiPosBits
			continue
		}
		out = append(out, projRec{slot, gen})
		p += lowPosBits
	}
	return out
}

func ascending(pay []byte, at, mc int) ([]int, bool) {
	idx := make([]int, mc)
	prev := -1
	for k := 0; k < mc; k++ {
		v := int(filmdec.PeekBits(pay, at+6*k, 6))
		if v <= prev {
			return nil, false
		}
		idx[k], prev = v, v
	}
	return idx, true
}

// slotBand : combler la plage de l'archetype, puis retirer tout slot vu porter un AUTRE
// archetype (regle de `worldObjectSlotBand`).
func slotBand(dir string, n int) map[uint32]bool {
	seen := map[uint32]bool{}
	others := map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == projectileTI {
					seen[uint32(r.Slot)] = true
				} else {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	lo, hi := ^uint32(0), uint32(0)
	for k := range seen {
		if k < lo {
			lo = k
		}
		if k > hi {
			hi = k
		}
	}
	band := make(map[uint32]bool, hi-lo+1)
	for k := lo; k <= hi; k++ {
		band[k] = true
	}
	for s := range others {
		delete(band, s)
	}
	return band
}

// listFilmsWithBounds croise le cache de films, la table matchID -> carte et le catalogue de
// bornes. Un film dont la carte n'a pas de bornes est ECARTE : le decodeur de production
// REFUSE de rendre des coordonnees sans elles, et les lui donner fausses serait pire.
func listFilmsWithBounds(root, csvPath string, cat *filmdec.MapQuantCatalog, limit int) (
	[]string, []filmdec.Vec3Range, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	byID := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		i := strings.Index(line, ",")
		if i > 0 {
			byID[line[:i]] = line[i+1:]
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var dirs []string
	var ranges []filmdec.Vec3Range
	for _, n := range names {
		m, ok := byID[n]
		if !ok {
			continue
		}
		entry, err := cat.Lookup(m)
		if err != nil {
			continue
		}
		dirs = append(dirs, filepath.Join(root, n))
		ranges = append(ranges, entry.Range())
		if limit > 0 && len(dirs) >= limit {
			break
		}
	}
	return dirs, ranges, nil
}

func loadWeaponNames(path string) map[uint64]string {
	out := map[uint64]string{}
	if path == "" {
		return out
	}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			if strings.HasPrefix(strings.ToLower(line), "weapon_id") {
				continue
			}
		}
		i := strings.Index(line, ",")
		if i <= 0 {
			continue
		}
		id, err := strconv.ParseUint(strings.TrimSpace(line[:i]), 10, 64)
		if err != nil {
			continue
		}
		out[id] = strings.TrimSpace(line[i+1:])
	}
	return out
}
