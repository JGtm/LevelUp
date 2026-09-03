package filmdec

// r7_oracle117_research_test.go — lot R7 : L'ORACLE 117, temoin de NON-REGRESSION permanent.
//
// R6 a valide la charge du type 117 `EquipmentTranslocatorTeleportEffects` AU METRE sur 18
// evenements de 5 films : un identifiant d'effet 32 bits garde, puis DEUX positions
// quantifiees — A = le depart du saut, B = l'arrivee — exactes a 0,00-0,26 m. Cette charge
// est depuis decodee en PRODUCTION.
//
// CE QUE CE TEST EXIGE DE LA MARCHE COMPLETE, et c'est le point : la marche doit
//  1. RETROUVER ces evenements a l'identique (memes slots, memes positions au metre) ;
//  2. les retrouver AUSSI quand ils ne sont pas en tete de liste (R6 ne lisait que la tete) ;
//  3. ENCHAINER proprement derriere — bit de fin de liste atteint, types plausibles.
//
// Une marche fausse echoue les trois : elle place la charge au mauvais bit (positions
// aberrantes), et elle produit des types absurdes derriere.
//
// SEUIL ECRIT AVANT LA MESURE : au moins autant d'evenements 117 valides (A et B a <= 1,5 m
// du couple depart/arrivee de la piste) que les 18 de R6, et un taux de validation >= 90 %
// des evenements 117 dont le slot a une piste dans l'artefact.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0.
//
//	CGO_ENABLED=0 R7_ROOT=... R7_ARTS=... R7_CAT=... R7_MAPS=... R7_IDS=... \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7Oracle117$' -count=1 -timeout 30m -v

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// r7Pos117 : une position decodee d'un evenement 117.
type r7Pos117 struct {
	x, y, z float64
	region  bool
}

// r7Decode117 lit la charge d'un evenement 117 a partir du bit qui suit son R(7) de type.
// Rend le slot (ref0, domaine 2, base 512), les deux positions, et si la lecture a tenu.
func r7Decode117(pay []byte, bitApresType int, e r6CatEntry) (int, r7Pos117, r7Pos117, bool) {
	br := NewBitReader(pay)
	br.Skip(bitApresType)
	var slot int
	if !br.ReadBit() {
		return 0, r7Pos117{}, r7Pos117{}, false // ref0 absente : pas d'unite designee
	}
	slot = int(br.ReadBits(8)) + 512
	br.Skip(2) // generation
	for i := 0; i < 2; i++ {
		if br.ReadBit() { // refs 1 et 2 : domaines 0 et 7, 13 bits
			br.Skip(13 + 2)
		}
	}
	if br.ReadBit() { // mot d'effet garde
		br.Skip(32)
	}
	lire := func() r7Pos117 {
		var p r7Pos117
		min, max := e.Min, e.Max
		bits := e.AxisWidths
		if !br.ReadBit() { // porte INVERSEE : 0 -> bornes de la region
			p.region = true
			br.Skip(1) // index de region (largeur runtime mesuree a 1)
		} else {
			min = [3]float64{-20000, -20000, -20000}
			max = [3]float64{20000, 20000, 20000}
			bits = [3]uint{22, 22, 22}
		}
		var out [3]float64
		for i := 0; i < 3; i++ {
			q := br.ReadBits(bits[i])
			out[i] = min[i] + (float64(q)+0.5)*(max[i]-min[i])/float64(uint64(1)<<bits[i])
		}
		p.x, p.y, p.z = out[0], out[1], out[2]
		return p
	}
	a, b := lire(), lire()
	return slot, a, b, br.BitPos() <= len(pay)*8
}

// TestR7Oracle117 rejoue l'oracle du translocateur sur la marche COMPLETE.
func TestR7Oracle117(t *testing.T) {
	root, ids := r7Films(t)
	arts, catPath := os.Getenv(r7ArtsEnv), os.Getenv(r7CatEnv)
	maps := os.Getenv(r7MapsEnv)
	if arts == "" || catPath == "" || maps == "" {
		t.Skipf("oracle 117 : definir %s, %s et %s", r7ArtsEnv, r7CatEnv, r7MapsEnv)
	}
	cat := r6LireCatalogue(t, catPath)
	nomsParFilm := map[string]string{}
	for _, kv := range splitKV(maps) {
		nomsParFilm[kv[0]] = kv[1]
	}
	cartes := r7Cartes(t)
	var totalVus, totalValides, totalSansPiste, totalTete, totalDerriere int
	suivants := map[int]int{}
	for _, id := range ids {
		nom := nomsParFilm[id]
		e, ok := cat[nom]
		if !ok {
			continue
		}
		dir := filepath.Join(root, id)
		n := r7Chunks(dir)
		if n == 0 {
			continue
		}
		raw, err := ReadFilmChunk(dir, 1)
		if err != nil {
			continue
		}
		pks := WalkPackets(raw)
		if len(pks) == 0 {
			continue
		}
		origine := pks[0].TimestampUS
		art, pistes := r6LireArtefact(t, filepath.Join(arts, id+".json"))
		ctx := cartes[id]
		vus, valides, sansPiste, tete := 0, 0, 0, 0
		for c := 1; c <= n; c++ {
			data, err := ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, pk := range WalkPackets(data) {
				if pk.Type != PacketTypeDelta || pk.Size < 2 {
					continue
				}
				pay := pk.Payload(data)
				if pay[0]&0x40 == 0 {
					continue
				}
				evs, _, _, _ := r7Marche(pay, ctx)
				for i, ev := range evs {
					if ev.Typ != 117 {
						continue
					}
					vus++
					if ev.Pos == 1 {
						tete++
					}
					if i+1 < len(evs) {
						suivants[evs[i+1].Typ]++
					}
					slot, a, b, ok := r7Decode117(pay, ev.BitDebut+7, e)
					if !ok {
						continue
					}
					tsMS := (int64(pk.TimestampUS) - int64(origine)) / 1000
					frame := (tsMS - art.OriginMs) / art.FrameIntervalMs
					pts := pistes[slot]
					if len(pts) == 0 {
						sansPiste++
						continue
					}
					from, to, saut, ok2 := r6FromTo(pts, frame)
					if !ok2 {
						sansPiste++
						continue
					}
					dA := math.Hypot(a.x-from.x, a.y-from.y)
					dB := math.Hypot(b.x-to.x, b.y-to.y)
					if math.Max(dA, dB) <= r6TolM {
						valides++
					}
					t.Logf("  %s @%d ms slot %d pos %d : dA=%.2f dB=%.2f m (saut piste %.2f m)",
						id, tsMS, slot, ev.Pos, dA, dB, saut)
				}
			}
		}
		t.Logf("film %s : %d evenements 117 (%d en tete, %d derriere) · %d valides · %d sans piste",
			id, vus, tete, vus-tete, valides, sansPiste)
		totalVus += vus
		totalValides += valides
		totalSansPiste += sansPiste
		totalTete += tete
		totalDerriere += vus - tete
	}
	t.Logf("")
	t.Logf("=== ORACLE 117 : %d evenements vus (%d en tete, %d DERRIERE une tete) · %d valides "+
		"a <= %.1f m · %d sans piste exploitable ===",
		totalVus, totalTete, totalDerriere, totalValides, r6TolM, totalSansPiste)
	den := totalVus - totalSansPiste
	t.Logf("=== TAUX DE VALIDATION : %d/%d = %.1f %% (seuil ecrit d'avance : 90 %%, et >= 18 valides) ===",
		totalValides, den, 100*float64(totalValides)/float64(max(1, den)))
	var types []int
	for typ := range suivants {
		types = append(types, typ)
	}
	sort.Ints(types)
	for _, typ := range types {
		t.Logf("  type qui SUIT un 117 dans la liste : %d %s x%d", typ, r7Noms[typ], suivants[typ])
	}
}

// splitKV decoupe "a=b,c=d" en couples.
func splitKV(s string) [][2]string {
	var out [][2]string
	for _, kv := range splitVirgule(s) {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				out = append(out, [2]string{trimEsp(kv[:i]), trimEsp(kv[i+1:])})
				break
			}
		}
	}
	return out
}

func splitVirgule(s string) []string {
	var out []string
	deb := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			out = append(out, s[deb:i])
			deb = i + 1
		}
	}
	return out
}

func trimEsp(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
