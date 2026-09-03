package filmdec

// r7_repulseur_research_test.go — lot R7 : LE REPULSEUR CONFRONTE A LA TRAJECTOIRE.
//
// L'EVENEMENT EST LA LECTURE, LA TRAJECTOIRE N'EST QU'UN ORACLE DE CONTROLE. On ne cherche
// pas les poussees dans les pistes pour les apparier ensuite : on lit les evenements 104
// `EquipmentKnockbackPlayer` dans le film, on resout leur unite, et on DEMANDE a la piste de
// cette unite si elle bouge anormalement a cet instant. Le sens de la fleche compte : un
// detecteur de trajectoire n'a jamais ete construit ici.
//
// CE QUE PORTE LA CHARGE (lecteur 0x14116c344 -> FUN_14076d528, sourcee de l'exe) :
//
//	R(1) nul ; si nul==0 : R(19) direction unitaire + R(10) magnitude
//	magnitude : echelle LOGARITHMIQUE entre 0,05 (DAT_143cd8648) et 20,0 (DAT_143cd8f60)
//
// L'unite poussee est la reference 0 de l'en-tete (domaine 0, 13 bits). Sa base est CALIBREE
// comme celle du zoom l'a ete : la base qui fait atterrir le plus d'index sur un slot ayant
// reellement une piste dans l'artefact. Une mauvaise base n'en place presque aucun.
//
// SEUILS ECRITS AVANT LA MESURE :
//  1. FERMETURE DE LA BASE : la meilleure base doit placer >= 80 % des index sur un slot a
//     piste, et au moins deux fois plus que la deuxieme meilleure base. Sinon : non concluant.
//  2. POUSSEE : pour chaque evenement, on prend la vitesse 2D de pointe de l'unite designee
//     dans [t, t+600 ms] et on la compare au 90e centile de la vitesse de CE slot sur tout le
//     film. Le taux de depassement doit valoir au moins 3 fois celui du TEMOIN — le meme
//     calcul aux memes instants sur un slot TIRE AU HASARD parmi les autres joueurs.
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0.
//
//	CGO_ENABLED=0 R7_ROOT=... R7_ARTS=... R7_CAT=... R7_MAPS=... R7_IDS=... \
//	  go test ./internal/analysis/filmdec/ -run '^TestR7Repulseur$' -count=1 -timeout 60m -v

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// r7Knock : un evenement 104 decode.
type r7Knock struct {
	tsMS    int64
	idx     uint64
	present bool
	nul     bool
	dir     uint64
	magRaw  uint64
	pos     int
}

// r7DecodeKnock lit l'en-tete et la charge d'un evenement 104 a partir du bit du R(7) de type.
func r7DecodeKnock(pay []byte, bitType int) (r7Knock, bool) {
	br := NewBitReader(pay)
	br.Skip(bitType + 7)
	var k r7Knock
	// refs : domaines {0, 0, 7} — 13 bits chacune.
	for i := 0; i < 3; i++ {
		if !br.ReadBit() {
			continue
		}
		v := br.ReadBits(13)
		br.Skip(2)
		if i == 0 {
			k.idx, k.present = v, true
		}
	}
	k.nul = br.ReadBit()
	if !k.nul {
		k.dir = br.ReadBits(19)
		k.magRaw = br.ReadBits(10)
	}
	return k, br.BitPos() <= len(pay)*8
}

// r7Magnitude deploie l'echelle logarithmique de la magnitude (bornes 0,05 et 20,0).
func r7Magnitude(raw uint64) float64 {
	const lo, hi = 0.05, 20.0
	f := float64(raw) / float64((1<<10)-1)
	return lo * math.Pow(hi/lo, f)
}

// r7Vitesses rend, par slot, la suite des vitesses 2D (m/s) entre frames consecutives.
func r7Vitesses(pistes map[int][]r6Point, intervalMs int64) map[int][]float64 {
	out := map[int][]float64{}
	dt := float64(intervalMs) / 1000.0
	for slot, pts := range pistes {
		v := make([]float64, 0, len(pts))
		for i := 0; i+1 < len(pts); i++ {
			if pts[i+1].t != pts[i].t+1 {
				v = append(v, 0)
				continue
			}
			v = append(v, math.Hypot(pts[i+1].x-pts[i].x, pts[i+1].y-pts[i].y)/dt)
		}
		out[slot] = v
	}
	return out
}

// r7Centile rend le centile p (0..1) d'une serie (copie triee).
func r7Centile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	i := int(p * float64(len(c)-1))
	return c[i]
}

// r7Pointe rend la vitesse de pointe du slot dans [frame, frame+n].
func r7Pointe(vit []float64, frame int64, n int) float64 {
	best := 0.0
	for i := frame; i <= frame+int64(n); i++ {
		if i < 0 || int(i) >= len(vit) {
			continue
		}
		if vit[i] > best {
			best = vit[i]
		}
	}
	return best
}

// TestR7Repulseur confronte les evenements 104 aux pistes.
func TestR7Repulseur(t *testing.T) {
	root, ids := r7Films(t)
	arts := os.Getenv(r7ArtsEnv)
	if arts == "" {
		t.Skipf("definir %s (artefacts de rejeu)", r7ArtsEnv)
	}
	cartes := r7Cartes(t)
	bases := []int{0, 128, 256, 384, 448, 480, 500, 508, 510, 512, 514, 516, 520, 544, 576}
	hits := map[int]int{}
	type occ struct {
		film string
		k    r7Knock
	}
	var toutes []occ
	pistesParFilm := map[string]map[int][]r6Point{}
	artParFilm := map[string]r6Artefact{}
	for _, id := range ids {
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
		pistesParFilm[id], artParFilm[id] = pistes, art
		ctx := cartes[id]
		nFilm := 0
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
				for _, ev := range evs {
					if ev.Typ != 104 {
						continue
					}
					k, ok := r7DecodeKnock(pay, ev.BitDebut)
					if !ok {
						continue
					}
					k.tsMS = (int64(pk.TimestampUS) - int64(origine)) / 1000
					k.pos = ev.Pos
					toutes = append(toutes, occ{id, k})
					nFilm++
					if !k.present {
						continue
					}
					for _, b := range bases {
						if len(pistes[b+int(k.idx)]) > 0 {
							hits[b]++
						}
					}
				}
			}
		}
		t.Logf("film %s : %d evenements 104", id, nFilm)
	}
	if len(toutes) == 0 {
		t.Logf("aucun evenement 104 : rien a eprouver")
		return
	}
	// Seuil 1 : fermeture de la base.
	avecRef := 0
	for _, o := range toutes {
		if o.k.present {
			avecRef++
		}
	}
	type bh struct {
		b, n int
	}
	var l []bh
	for _, b := range bases {
		l = append(l, bh{b, hits[b]})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	t.Logf("")
	t.Logf("=== %d evenements 104 · %d avec reference presente ===", len(toutes), avecRef)
	for _, e := range l[:min(5, len(l))] {
		t.Logf("  base %4d : %d/%d index sur un slot a piste (%.1f %%)", e.b, e.n, avecRef,
			100*float64(e.n)/float64(max(1, avecRef)))
	}
	// DIAGNOSTIC : la distribution des index bruts. Un index de BIPEDE reste dans une bande
	// etroite (les 8 joueurs) ; un index d'OBJET du monde derive avec le temps de film.
	var idxs []int
	for _, o := range toutes {
		if o.k.present {
			idxs = append(idxs, int(o.k.idx))
		}
	}
	sort.Ints(idxs)
	if len(idxs) > 0 {
		t.Logf("  index ref0 bruts : min %d · q1 %d · median %d · q3 %d · max %d · %d distincts",
			idxs[0], idxs[len(idxs)/4], idxs[len(idxs)/2], idxs[3*len(idxs)/4],
			idxs[len(idxs)-1], nbDistincts(idxs))
		t.Logf("  echantillon : %v", idxs[:min(30, len(idxs))])
	}
	base := l[0].b
	tauxBase := 100 * float64(l[0].n) / float64(max(1, avecRef))
	second := 0
	if len(l) > 1 {
		second = l[1].n
	}
	t.Logf("SEUIL 1 (fermeture de la base) : base %d a %.1f %% ; deuxieme a %d — %s",
		base, tauxBase, second, verdictBase(tauxBase, l[0].n, second))
	// Seuil 2 : poussee mesuree contre temoin.
	const fenetreFrames = 6 // 600 ms a 100 ms/frame
	nMes, nDep, nTem, nTemDep := 0, 0, 0, 0
	var magnitudes []float64
	for _, o := range toutes {
		if !o.k.present {
			continue
		}
		pistes := pistesParFilm[o.film]
		art := artParFilm[o.film]
		if art.FrameIntervalMs == 0 {
			continue
		}
		vits := r7Vitesses(pistes, art.FrameIntervalMs)
		slot := base + int(o.k.idx)
		v := vits[slot]
		if len(v) < 50 {
			continue
		}
		frame := (o.k.tsMS - art.OriginMs) / art.FrameIntervalMs
		seuil := r7Centile(v, 0.90)
		nMes++
		if r7Pointe(v, frame, fenetreFrames) > seuil {
			nDep++
		}
		if !o.k.nul {
			magnitudes = append(magnitudes, r7Magnitude(o.k.magRaw))
		}
		// TEMOIN : le meme instant, sur les autres slots a piste du meme film.
		var autres []int
		for s := range vits {
			if s != slot && len(vits[s]) >= 50 {
				autres = append(autres, s)
			}
		}
		sort.Ints(autres)
		for _, s := range autres {
			nTem++
			if r7Pointe(vits[s], frame, fenetreFrames) > r7Centile(vits[s], 0.90) {
				nTemDep++
			}
		}
	}
	tx := 100 * float64(nDep) / float64(max(1, nMes))
	tt := 100 * float64(nTemDep) / float64(max(1, nTem))
	t.Logf("")
	t.Logf("SEUIL 2 (poussee) : unite DESIGNEE %d/%d = %.1f %% au-dessus de son 90e centile ; "+
		"TEMOIN (autres slots, memes instants) %d/%d = %.1f %% · facteur %.2f (seuil 3)",
		nDep, nMes, tx, nTemDep, nTem, tt, tx/mathMax(tt, 0.01))
	if len(magnitudes) > 0 {
		sort.Float64s(magnitudes)
		t.Logf("magnitudes decodees (n=%d) : min %.2f · median %.2f · max %.2f",
			len(magnitudes), magnitudes[0], magnitudes[len(magnitudes)/2],
			magnitudes[len(magnitudes)-1])
	}
	var nuls int
	for _, o := range toutes {
		if o.k.nul {
			nuls++
		}
	}
	t.Logf("charges a poussee NULLE (R(1)=1) : %d/%d", nuls, len(toutes))
}

func verdictBase(taux float64, premier, second int) string {
	if taux >= 80 && premier >= 2*second {
		return "FERME"
	}
	return "NON CONCLUANT"
}

func nbDistincts(v []int) int {
	n := 0
	for i := range v {
		if i == 0 || v[i] != v[i-1] {
			n++
		}
	}
	return n
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
