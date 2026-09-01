package filmdec

// bombe_desamorcage_research_test.go — LES POSES SANS EXPLOSION : le desamorcage, s'il existe
// au corpus.
//
// # CE QUI EST ACQUIS (2026-09-01, chantier bombe-portee)
//
// L'anneau `ti=12 i14` est la jauge d'armement : fin de montee contigue = bombe ARMEE,
// +4,93 s = explosion (plancher 0/1000, `navpoint_ti12_plancher_test.go`). La bombe est la
// famille `0x3fee4fcf` du canal des armes tenues (B1 : unique candidate des 9 films ; atlas
// HUD du jeu : sprite `contour-34` nomme « ball | bomb »).
//
// # PROTOCOLE, ecrit avant la mesure
//
//	D1  Une POSE COMPLETE est une montee contigue (memes regles que le plancher : trous
//	    <= 500 ms, >= 3 echantillons, amplitude >= 16) dont le quantum FINAL atteint le
//	    quantum MAXIMAL observe sur le film (la valeur pleine de l'anneau).
//	D2  CONTROLE : chaque explosion de l'oracle doit etre precedee (<= 10 s) d'une pose
//	    complete — publie par film, avec les delais. Un trou ici borne la sensibilite de D1,
//	    il ne casse pas D3.
//	D3  Une pose complete SANS explosion de l'oracle dans les 10 s qui suivent est une
//	    CANDIDATE DESAMORCAGE. Pour chacune : publier ce que fait la famille bombe dans les
//	    30 s suivantes (transitions VERS = un joueur reprend la bombe ; DEPUIS = il la
//	    lache), le slot et le delai. AUCUNE candidate au corpus = « pas d'occurrence
//	    oracle », et c'est un resultat, pas un echec.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, verrou
// process (un seul decodage a la fois).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run BombeDesamorcage -v -timeout 60m

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// bdBombe est la famille de la bombe (B1 + atlas HUD).
	bdBombe = uint32(0x3fee4fcf)
	// bdFenetreExplosionMS : une pose complete est ARMEE si une explosion la suit dans
	// cette fenetre (la meche mesuree vaut ~4,93 s ; 10 s la couvre avec marge).
	bdFenetreExplosionMS = 10000
	// bdFenetreSuiteMS : fenetre d'observation du canal bombe apres une candidate.
	bdFenetreSuiteMS = 30000
)

// bdMontee est une montee contigue avec ses quanta de depart et de fin.
type bdMontee struct {
	slot           uint32
	debutMS, finMS int32
	qDebut, qFin   uint8
}

// bdMonteesContigues decoupe une serie triee avec les regles du plancher, en gardant les
// quanta (tpMonteesContigues ne rend que les fins — ici la COMPLETUDE se juge au quantum).
func bdMonteesContigues(slot uint32, s []ti12Ech) []bdMontee {
	var out []bdMontee
	for i := 0; i < len(s); {
		j := i
		for j+1 < len(s) && s[j+1].q >= s[j].q && s[j+1].tMS-s[j].tMS <= NavpointRiseMaxGapMS {
			j++
		}
		n := j - i + 1
		if n >= ti12MonteeMinEch && int(s[j].q)-int(s[i].q) >= ti12MonteeMinAmpl {
			out = append(out, bdMontee{
				slot: slot, debutMS: s[i].tMS, finMS: s[j].tMS, qDebut: s[i].q, qFin: s[j].q,
			})
		}
		if j == i {
			i++
			continue
		}
		i = j
	}
	return out
}

// bdTransition est une transition de la famille bombe, datee sur l'horloge du match.
type bdTransition struct {
	tMS   int
	slot  uint32
	prise bool
}

// bdTransitionsBombe lit les transitions de la famille bombe et les date par la MEME horloge
// que le scan ti=12 (startMS du manifeste + premier paquet delta du chunk, patron zcScanChunk).
func bdTransitionsBombe(t *testing.T, dir string, clk zcClock) []bdTransition {
	t.Helper()
	changes, _, err := ScanFilmHeldWeaponChanges(dir, nil)
	if err != nil {
		t.Fatalf("%s : canal des armes tenues illisible : %v", dir, err)
	}
	base := map[int]uint64{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type == PacketTypeDelta {
				base[c] = pk.TimestampUS
				break
			}
		}
	}
	var out []bdTransition
	for _, ch := range changes {
		start, okS := clk.startMS[ch.Chunk]
		b, okB := base[ch.Chunk]
		if !okS || !okB {
			continue
		}
		tMS := start + int((ch.TimestampUS-b)/1000)
		if ch.Family == bdBombe {
			out = append(out, bdTransition{tMS: tMS, slot: ch.Slot, prise: true})
		}
		if ch.Previous == bdBombe && ch.Family != bdBombe {
			out = append(out, bdTransition{tMS: tMS, slot: ch.Slot, prise: false})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].tMS < out[j].tMS })
	return out
}

// TestBombeDesamorcage applique D1-D3 aux neuf films d'Assaut.
func TestBombeDesamorcage(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	g := filmproc.Arm("TestBombeDesamorcage", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — mesure interrompue", float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("pic memoire observe : %.2f Gio", float64(g.Peak())/(1<<30))
	}()
	release := LockProcessDecode()
	defer release()

	films := make([]string, 0, len(ti12Explosions))
	for id := range ti12Explosions {
		films = append(films, id)
	}
	sort.Strings(films)

	totalCandidates := 0
	for _, id := range films {
		totalCandidates += bdFilm(t, cache, id)
	}
	if totalCandidates == 0 {
		t.Logf("VERDICT D3 : aucune pose complete sans explosion au corpus — pas d'occurrence oracle de desamorcage")
	} else {
		t.Logf("VERDICT D3 : %d candidate(s) desamorcage au corpus", totalCandidates)
	}
}

// bdFilm applique D1-D3 a UN film et rend le nombre de candidates D3.
func bdFilm(t *testing.T, cache, id string) int {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	clk, ok := ti12Horloge(dir)
	if !ok {
		t.Fatalf("%s : horloge illisible", id)
	}
	sc, err := ScanFilmNavpointRadial(dir, clk.startMS)
	if err != nil {
		t.Fatalf("%s : scan ti=12 impossible : %v", id, err)
	}
	series := map[uint32][]ti12Ech{}
	var qMax uint8
	for _, r := range sc.Reads {
		series[r.Slot] = append(series[r.Slot], ti12Ech{r.TMS, r.Q})
		if r.Q > qMax {
			qMax = r.Q
		}
	}
	var completes []bdMontee
	for slot, s := range series {
		sort.Slice(s, func(i, j int) bool { return s[i].tMS < s[j].tMS })
		for _, m := range bdMonteesContigues(slot, s) {
			if m.qFin == qMax { // D1 : la valeur pleine de l'anneau
				completes = append(completes, m)
			}
		}
	}
	sort.Slice(completes, func(i, j int) bool { return completes[i].finMS < completes[j].finMS })
	bdControleD2(t, id, completes, qMax)
	return bdCandidatesD3(t, id, dir, clk, completes)
}

// bdSuivieDExplosion dit si une montee complete est suivie d'une explosion de l'oracle.
func bdSuivieDExplosion(id string, m bdMontee) bool {
	for _, e := range ti12Explosions[id] {
		d := int32(e) - m.finMS
		if d > 0 && d <= bdFenetreExplosionMS {
			return true
		}
	}
	return false
}

// bdControleD2 publie la couverture des explosions par les poses completes.
func bdControleD2(t *testing.T, id string, completes []bdMontee, qMax uint8) {
	t.Helper()
	couvertes := 0
	for _, e := range ti12Explosions[id] {
		trouve := false
		for _, m := range completes {
			d := int32(e) - m.finMS
			if d > 0 && d <= bdFenetreExplosionMS {
				t.Logf("%s : explosion %d — pose complete a %d (delai %.2f s, q %d->%d, slot %d)",
					id, e, m.finMS, float64(d)/1000, m.qDebut, m.qFin, m.slot)
				trouve = true
				break
			}
		}
		if trouve {
			couvertes++
		} else {
			t.Logf("%s : explosion %d — AUCUNE pose complete dans les 10 s avant (sensibilite D1)", id, e)
		}
	}
	t.Logf("%s : qMax=%d, %d montees completes, D2 %d/%d explosions couvertes",
		id, qMax, len(completes), couvertes, len(ti12Explosions[id]))
}

// bdCandidatesD3 publie les poses completes sans explosion et ce que fait la bombe apres.
func bdCandidatesD3(t *testing.T, id, dir string, clk zcClock, completes []bdMontee) int {
	t.Helper()
	candidates := 0
	var trans []bdTransition
	transChargees := false
	for _, m := range completes {
		if bdSuivieDExplosion(id, m) {
			continue
		}
		candidates++
		if !transChargees {
			trans = bdTransitionsBombe(t, dir, clk)
			transChargees = true
		}
		t.Logf("%s : CANDIDATE DESAMORCAGE — pose complete a %d ms (q %d->%d, slot %d) sans explosion <= 10 s",
			id, m.finMS, m.qDebut, m.qFin, m.slot)
		for _, tr := range trans {
			d := tr.tMS - int(m.finMS)
			if d < 0 || d > bdFenetreSuiteMS {
				continue
			}
			geste := "LACHER"
			if tr.prise {
				geste = "PRISE"
			}
			t.Logf("    +%5.2f s : %s de la bombe par le slot %d", float64(d)/1000, geste, tr.slot)
		}
	}
	return candidates
}
