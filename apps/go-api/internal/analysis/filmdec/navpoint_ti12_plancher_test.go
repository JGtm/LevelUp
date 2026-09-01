package filmdec

// navpoint_ti12_plancher_test.go — LE PLANCHER DE LA SEULE OBSERVATION SURVIVANTE DU LOT.
//
// # CE QUI EST EN JEU
//
// Le lot du 2026-09-01 a laisse UNE observation sans plancher : sur les films d'Assaut, l'anneau
// du marqueur (`ti=12 i14`) atteint son quantum plein ~5 s avant chaque explosion. Si cette
// constante survit a un tirage nul, l'anneau EST la jauge d'armement et le chantier est clos par
// un positif. Si elle ne lui survit pas, la derniere piste film tombe, et l'armement se deduira
// hors film (meche constante).
//
// # LE PROTOCOLE, fixe AVANT la mesure (spec de la synthese du lot, appliquee sans retouche)
//
//	A — LA VRAIE. Les 13 explosions des 5 films Neutral Bomb REELS (35b75a31, ce083875,
//	    69b16f5d, 3d58eb37, 34bb3bc8). `1c01e34f` est RETIRE : Husky Raid, carte Forge, rampe
//	    six fois plus rapide (partition ANTERIEURE a la mesure, PLAN_ASSAUT_LOT_A §0).
//	    Montee redefinie avec CONTIGUITE : trou entre echantillons <= 500 ms — c'est ce qui
//	    separe les rampes reelles des ramassages de queue.
//	B — LA NULLE. 1 000 tirages de 13 instants uniformes (graine FIXE = 1), memes effectifs
//	    par film, dans l'etendue des lectures du film. Meme statistique.
//	C — LES DECALAGES. Les vraies cibles decalees de +45 s, -45 s, +120 s.
//
// STATISTIQUE, identique pour tous : delai = cible moins la FIN de la derniere montee contigue
// avant la cible (tous slots), retenu si 0 < delai <= 120 s ; couverture = cibles couvertes ;
// dispersion = ecart-type / mediane des delais (le CV du chantier).
//
// # LA REGLE DE DECISION, ecrite ici et appliquee telle quelle
//
//	RETENU      ssi A couvre 13/13 avec CV <= 0,20, MOINS DE 1 % des tirages nuls font aussi
//	            bien (couverture pleine ET CV <= CV reel), et les decalages echouent.
//	ARTEFACT    si la nulle fait souvent aussi bien : la statistique « dernier evenement avant
//	            t » est degeneree, les ~5 s sont un artefact d'argmax, et le negatif CLOT la
//	            derniere piste film.
//
// REGIME : garde `ASSAUT_CACHE`. Aucune base, aucun reseau, sentinelle memoire armee, un seul
// decodage a la fois (verrou process).
//
//	$env:ASSAUT_CACHE="C:/.../data/cache"
//	go test ./internal/analysis/filmdec/ -run NavpointTi12Plancher -v -timeout 60m

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

// tpFilms : les cinq films Neutral Bomb reels et leurs explosions (copie gardee par
// TestNavpointTi12OracleFige via ti12Explosions — ici seulement le sous-ensemble).
var tpFilms = []struct {
	id   string
	exps []int32
}{
	{"34bb3bc8", []int32{427120}},
	{"35b75a31", []int32{304013, 541270, 787051}},
	{"3d58eb37", []int32{203065, 342196, 386280}},
	{"69b16f5d", []int32{154305, 278617, 310215}},
	{"ce083875", []int32{512505, 686401, 947537}},
}

// La contiguite (trou <= 500 ms) vit desormais en PRODUCTION (`NavpointRiseMaxGapMS`,
// navpoint_radial_rises.go) — TestNavpointTi12ProtocoleFige garde la valeur du protocole.
const (
	tpSensMaxMS  = 120000 // fenetre de sens du chantier
	tpTirages    = 1000
	tpGraine     = 1
	tpCVSeuil    = 0.20
	tpNulSeuilPc = 1.0 // % de tirages nuls autorises a faire aussi bien
)

// tpFilmDonnees : les fins de montees contigues et l'etendue des lectures d'un film.
type tpFilmDonnees struct {
	fins     []int32
	min, max int32
	exps     []int32
}

// TestNavpointTi12Plancher applique le protocole A / B / C.
func TestNavpointTi12Plancher(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer tpSentinelle(t)()
	release := LockProcessDecode()
	defer release()

	var films []tpFilmDonnees
	for _, f := range tpFilms {
		d, ok := tpCharger(t, cache, f.id)
		if !ok {
			t.Fatalf("%s : film indispensable absent — le protocole exige les cinq", f.id)
		}
		d.exps = f.exps
		films = append(films, d)
		t.Logf("%-9s : %3d montee(s) contigue(s), lectures de %d a %d ms",
			f.id, len(d.fins), d.min, d.max)
	}

	// A — LA VRAIE.
	couv, cv, med := tpStat(films, nil)
	t.Logf("########## A (reel)      : couverture %2d/13, delai median %6.1f s, CV %5.3f",
		couv, med/1000, cv)

	// C — LES DECALAGES.
	for _, dec := range []int32{45000, -45000, 120000} {
		c, v, m := tpStat(films, func(e int32) int32 { return e + dec })
		t.Logf("########## C (%+6d ms) : couverture %2d/13, delai median %6.1f s, CV %5.3f",
			dec, c, m/1000, v)
	}

	// B — LA NULLE.
	rng := rand.New(rand.NewSource(tpGraine))
	aussiBien, pleins := 0, 0
	cvs := make([]float64, 0, tpTirages)
	for i := 0; i < tpTirages; i++ {
		c, v, _ := tpStatNulle(films, rng)
		if c == 13 {
			pleins++
			cvs = append(cvs, v)
			if v <= cv {
				aussiBien++
			}
		}
	}
	sort.Float64s(cvs)
	t.Logf("########## B (nulle, %d tirages, graine %d) :", tpTirages, tpGraine)
	t.Logf("  couverture pleine : %d/%d tirages (%.1f %%)", pleins, tpTirages,
		100*float64(pleins)/float64(tpTirages))
	if len(cvs) > 0 {
		t.Logf("  CV des tirages pleins : p5 %5.3f, mediane %5.3f, p95 %5.3f",
			cvs[len(cvs)/20], cvs[len(cvs)/2], cvs[len(cvs)*19/20])
	}
	t.Logf("  tirages faisant AUSSI BIEN que le reel (pleins ET CV <= %5.3f) : %d (%.1f %%)",
		cv, aussiBien, 100*float64(aussiBien)/float64(tpTirages))

	// LA REGLE, appliquee telle qu'ecrite.
	reelPasse := couv == 13 && cv <= tpCVSeuil
	nulRare := 100*float64(aussiBien)/float64(tpTirages) < tpNulSeuilPc
	if reelPasse && nulRare {
		t.Logf("VERDICT : RETENU sous la regle — l'anneau ti=12 est la jauge d'armement candidate.")
	} else {
		t.Logf("VERDICT : ARTEFACT sous la regle (reel passe=%v, nulle rare=%v) — la derniere "+
			"piste film tombe.", reelPasse, nulRare)
	}
}

// tpCharger balaie UN film et rend les fins de montees CONTIGUES, tous slots.
func tpCharger(t *testing.T, cache, id string) (tpFilmDonnees, bool) {
	t.Helper()
	dir := filepath.Join(cache, "film_chunks", id)
	if CountFilmChunks(dir) == 0 {
		return tpFilmDonnees{}, false
	}
	clk, ok := ti12Horloge(dir)
	if !ok {
		return tpFilmDonnees{}, false
	}
	sc, err := ScanFilmNavpointRadial(dir, clk.startMS)
	if err != nil {
		return tpFilmDonnees{}, false
	}
	d := tpFilmDonnees{min: math.MaxInt32, max: math.MinInt32}
	for _, r := range sc.Reads {
		if r.TMS < d.min {
			d.min = r.TMS
		}
		if r.TMS > d.max {
			d.max = r.TMS
		}
	}
	// LA DETECTION EST CELLE DE PRODUCTION (`NavpointContiguousRises`, portee le 2026-09-01
	// depuis cet instrument) : le plancher mesure ainsi le code qui publie, pas une copie.
	// Les fins sortent triees (EndMS puis Slot) — l'ordre qu'exige tpDelai.
	for _, m := range NavpointContiguousRises(sc.Reads) {
		d.fins = append(d.fins, m.EndMS)
	}
	return d, len(d.fins) > 0
}

// TestNavpointTi12ProtocoleFige garde les seuils de production contre une derive silencieuse :
// le verdict 0/1000 du plancher n'a de sens que sous LA definition du protocole (trou 500 ms,
// 3 echantillons, 16 quanta). Changer un seuil de production invalide le protocole — ce test
// force a le dire.
func TestNavpointTi12ProtocoleFige(t *testing.T) {
	if NavpointRiseMaxGapMS != 500 || NavpointRiseMinSamples != 3 || NavpointRiseMinQuanta != 16 {
		t.Fatalf("seuils de production hors protocole du 2026-09-01 : trou %d (attendu 500), "+
			"echantillons %d (attendu 3), quanta %d (attendu 16) — re-passer le plancher avant de bouger",
			NavpointRiseMaxGapMS, NavpointRiseMinSamples, NavpointRiseMinQuanta)
	}
}

// tpStat calcule (couverture, CV, mediane) des delais cible - derniere fin de montee, sur les
// vraies explosions transformees par `f` (nil = identite).
func tpStat(films []tpFilmDonnees, f func(int32) int32) (int, float64, float64) {
	var delais []float64
	couv := 0
	for _, d := range films {
		for _, e := range d.exps {
			c := e
			if f != nil {
				c = f(e)
			}
			if dd, ok := tpDelai(d.fins, c); ok {
				couv++
				delais = append(delais, dd)
			}
		}
	}
	med, cv := tpMedCV(delais)
	return couv, cv, med
}

// tpStatNulle tire les cibles uniformement dans l'etendue de chaque film, memes effectifs.
func tpStatNulle(films []tpFilmDonnees, rng *rand.Rand) (int, float64, float64) {
	var delais []float64
	couv := 0
	for _, d := range films {
		span := d.max - d.min
		for range d.exps {
			c := d.min + int32(rng.Int63n(int64(span)+1))
			if dd, ok := tpDelai(d.fins, c); ok {
				couv++
				delais = append(delais, dd)
			}
		}
	}
	med, cv := tpMedCV(delais)
	return couv, cv, med
}

// tpDelai rend le delai cible - derniere fin de montee AVANT la cible, dans la fenetre de sens.
func tpDelai(fins []int32, cible int32) (float64, bool) {
	i := sort.Search(len(fins), func(k int) bool { return fins[k] >= cible })
	if i == 0 {
		return 0, false
	}
	d := cible - fins[i-1]
	if d <= 0 || d > tpSensMaxMS {
		return 0, false
	}
	return float64(d), true
}

// tpMedCV rend la mediane et le CV (ecart-type sur mediane) d'une serie.
func tpMedCV(xs []float64) (float64, float64) {
	if len(xs) == 0 {
		return 0, math.Inf(1)
	}
	tri := append([]float64(nil), xs...)
	sort.Float64s(tri)
	med := tri[len(tri)/2]
	if med == 0 {
		return med, math.Inf(1)
	}
	var s float64
	for _, x := range xs {
		s += (x - med) * (x - med)
	}
	return med, math.Sqrt(s/float64(len(xs))) / med
}

// tpSentinelle arme le plafond memoire de mesure et rend le desarmement.
func tpSentinelle(t *testing.T) func() {
	t.Helper()
	g := filmproc.Arm("TestNavpointTi12Plancher", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE (%.2f Gio) — plancher interrompu", float64(peak)/(1<<30))
	})
	return func() { g.Disarm() }
}

// TestNavpointTi12PlancherVariantes — LA MEME EPREUVE sur One Bomb et Husky Raid.
//
// La constante de 4,93 s est etablie sur NEUTRAL BOMB. Le lot avait observe ~17 s sur One Bomb
// et des rampes six fois plus rapides sur Husky Raid : soit chaque variante a SA meche (un
// reglage de mode), soit la lecture ne se generalise pas. Meme statistique, meme nulle, groupes
// separes — le verdict est par variante.
//
//	go test ./internal/analysis/filmdec/ -run NavpointTi12PlancherVariantes -v -timeout 60m
func TestNavpointTi12PlancherVariantes(t *testing.T) {
	cache := os.Getenv("ASSAUT_CACHE")
	if cache == "" {
		t.Skip("mesure non demandee : ASSAUT_CACHE requis")
	}
	defer tpSentinelle(t)()
	release := LockProcessDecode()
	defer release()

	groupes := []struct {
		nom   string
		films []struct {
			id   string
			exps []int32
		}
	}{
		{"One Bomb", []struct {
			id   string
			exps []int32
		}{
			{"9f57c612", []int32{83322, 298489, 353160, 469057}},
			{"c75f33b8", []int32{109549, 395724, 450833}},
			{"df8fcbef", []int32{255767, 309284, 485860, 778033}},
		}},
		{"Husky Raid", []struct {
			id   string
			exps []int32
		}{
			{"1c01e34f", []int32{150546, 273787, 335637, 400853}},
		}},
	}

	for _, grp := range groupes {
		var films []tpFilmDonnees
		total := 0
		for _, f := range grp.films {
			d, ok := tpCharger(t, cache, f.id)
			if !ok {
				t.Logf("%s : %s absent — groupe incomplet", grp.nom, f.id)
				continue
			}
			d.exps = f.exps
			films = append(films, d)
			total += len(f.exps)
		}
		couv, cv, med := tpStat(films, nil)
		t.Logf("########## %s — %d explosion(s) : couverture %d/%d, delai median %6.1f s, CV %5.3f",
			grp.nom, total, couv, total, med/1000, cv)
		rng := rand.New(rand.NewSource(tpGraine))
		aussiBien, pleins := 0, 0
		for i := 0; i < tpTirages; i++ {
			c, v, _ := tpStatNulle(films, rng)
			if c == total {
				pleins++
				if v <= cv {
					aussiBien++
				}
			}
		}
		t.Logf("           nulle : %d/%d pleins, %d aussi bien que le reel", pleins, tpTirages, aussiBien)
	}
}
