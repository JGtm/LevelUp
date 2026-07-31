// tmp_vitals — TEMOINS du lot L1/L2/L9 : sante (i4), bouclier (i5), compteur de
// reapparition (ti=5 i1) et horloge de manche (ti=0 i5).
//
// THROWAWAY : sert a MESURER, pas a decoder. Le decodage vit dans internal/analysis/filmdec.
//
// Chaque chiffre publie ici vient avec son TEMOIN et la valeur attendue PAR HASARD :
//
//	A. Sante/bouclier — RAPPORT P(sante==0 | juste avant une mort connue)
//	   / P(sante==0 | instant quelconque ou le joueur est vivant). Les morts viennent des
//	   FINS DE VIE des trajectoires (derniere position transmise par slot), pas du decodage
//	   de la sante : les deux sources sont independantes. Un taux brut ne prouverait rien.
//	B. Reapparition — ecart entre le demarrage du compteur et une mort connue, compare a un
//	   TEMOIN ou les morts sont retirees au hasard sur la meme plage temporelle.
//	C. Horloge — regression lineaire sur l'horodatage des paquets. Une pente de -1 s/s est
//	   la signature d'un chrono ; toute autre pente dit que le champ n'est pas un chrono.
//	D. Non-regression du cap de visee, mesuree : l'ajout d'i3 au balayage change la
//	   COUVERTURE de la visee, il ne doit pas degrader sa QUALITE (ecart median au sens du
//	   deplacement, contre un temoin aleatoire).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_vitals [--map Cliffhanger] [--match 000d5950]
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
)

// preDeathMS : fenetre « juste avant la mort ». 500 ms = ~30 trames a 60 Hz, assez large
// pour contenir la sequence de degats terminale, assez etroite pour ne pas avaler du jeu
// normal.
const preDeathMS = 500

// aliveGuardMS : marge autour d'une fin de vie exclue du groupe « vivant ». Sans elle, le
// denominateur du rapport contiendrait les instants de mort eux-memes et le rapport
// tendrait vers 1 par construction — c'est exactement la tautologie a eviter.
const aliveGuardMS = 5000

// minLifeSamples : une vie de moins de 10 echantillons n'a pas de « juste avant ».
const minLifeSamples = 10

// seedChunk : le chunk dont le keyframe seme les bindings du monde (protocole publie de
// cmd/tmp_l0witness). Le parcours sequentiel commence au chunk suivant.
const seedChunk = 2

type sample struct {
	slot      uint32
	tsUS      uint64
	hasBody   bool
	healthQ   uint8
	health    float32
	hasShield bool
	shieldQ   uint8
	shield    float32
	hasYaw    bool
	x, y      float32
}

func main() {
	mapName := flag.String("map", "Cliffhanger", "carte du match (porte les bornes de dequantification)")
	matchID := flag.String("match", "000d5950", "identifiant du match")
	dumpCSV := flag.String("csv", "", "chemin d'un CSV de sortie (sante/bouclier par slot et par instant)")
	// Le catalogue de bornes vit dans le worktree, le cache de films dans l'arbre principal :
	// les deux racines different, d'ou ce chemin explicite (le defaut reste le repoRoot).
	filmFlag := flag.String("film", "", "repertoire des chunks du film (defaut : <repo>/data/cache/film_chunks/<match>)")
	flag.Parse()

	repoRoot, err := title.FindRepoRoot()
	if err != nil {
		fmt.Println("racine repo:", err)
		os.Exit(1)
	}
	cat, err := filmdec.LoadMapQuantCatalog(title.NewPathResolver(repoRoot).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		fmt.Println("catalogue de bornes:", err)
		os.Exit(1)
	}
	entry, err := cat.Lookup(*mapName)
	if err != nil {
		fmt.Println("carte:", err)
		os.Exit(1)
	}
	rng := entry.Range()
	filmDir := filepath.Join(repoRoot, "data", "cache", "film_chunks", *matchID)
	if *filmFlag != "" {
		filmDir = *filmFlag
	}
	filmRoot = filepath.Dir(filmDir)

	opt := filmdec.DefaultScanFilmOptions()
	opt.CaptureDirs = true
	opt.WorldRange = &rng
	pos, err := filmdec.ScanFilmBipedPositions(filmDir, opt)
	if err != nil {
		fmt.Println("balayage du film:", err)
		os.Exit(1)
	}
	samples := make([]sample, 0, len(pos))
	for _, p := range pos {
		s := sample{slot: p.Slot, tsUS: p.TimestampUS, x: p.X, y: p.Y}
		if h, ok := p.HealthAt(); ok {
			s.hasBody, s.health, s.healthQ = true, h, p.Body.Q
		}
		if sh, ok := p.ShieldAt(); ok {
			s.hasShield, s.shield, s.shieldQ = true, sh, p.Shield.Q
		}
		_, s.hasYaw = p.AimHeadingDeg()
		samples = append(samples, s)
	}
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].tsUS < samples[j].tsUS })

	fmt.Printf("=== VALEURS — film %s / carte %s ===\n", *matchID, *mapName)
	reportCoverage(samples)
	lifeEnd := lifeEnds(samples)
	reportShieldWitness(samples, lifeEnd)
	reportShieldRegen(samples)
	reportRespawn(repoRoot, *matchID, lifeEnd)
	reportClock(repoRoot, *matchID)
	reportAimQuality(pos)
	reportWalkReach(repoRoot, *matchID)
	if *dumpCSV != "" {
		if err := writeCSV(*dumpCSV, samples); err != nil {
			fmt.Println("CSV:", err)
		} else {
			fmt.Printf("\nCSV ecrit -> %s (%d lignes)\n", *dumpCSV, len(samples))
		}
	}
}

// ---------------------------------------------------------------- couverture

func reportCoverage(s []sample) {
	nb, ns, ny := 0, 0, 0
	hq := map[uint8]int{}
	sq := map[uint8]int{}
	var hmin, hmax float32 = 9, -9
	var smin, smax float32 = 9, -9
	for _, x := range s {
		if x.hasYaw {
			ny++
		}
		if x.hasBody {
			nb++
			hq[x.healthQ]++
			if x.health < hmin {
				hmin = x.health
			}
			if x.health > hmax {
				hmax = x.health
			}
		}
		if x.hasShield {
			ns++
			sq[x.shieldQ]++
			if x.shield < smin {
				smin = x.shield
			}
			if x.shield > smax {
				smax = x.shield
			}
		}
	}
	fmt.Printf("\n-- COUVERTURE (%d positions decodees) --\n", len(s))
	fmt.Printf("  sante i4    : %6d (%.1f %%)  plage affichee [%.3f, %.3f]  quanta distincts %d\n",
		nb, pct(nb, len(s)), hmin, hmax, len(hq))
	fmt.Printf("  bouclier i5 : %6d (%.1f %%)  plage affichee [%.3f, %.3f]  quanta distincts %d\n",
		ns, pct(ns, len(s)), smin, smax, len(sq))
	fmt.Printf("  cap de visee: %6d (%.1f %%)  [TEMOIN de non-regression : cette couverture CHANGE\n", ny, pct(ny, len(s)))
	fmt.Printf("                               avec l'ajout d'i3 au balayage — cf. rapport]\n")
	fmt.Printf("  quanta de sante les plus frequents   : %s\n", topQ(hq, 6))
	fmt.Printf("  quanta de bouclier les plus frequents: %s\n", topQ(sq, 6))
	reportQuantumSupport("sante i4", hq, nb, 128, 254)
	reportQuantumSupport("bouclier i5", sq, ns, 0, 64)
}

// reportQuantumSupport — TEMOIN DE FORME, celui qu'on peut opposer au hasard sans rien
// supposer du jeu : un champ de 8 bits lu au mauvais endroit est du bruit, donc uniforme
// sur 0..255. Si les quanta observes se concentrent EXACTEMENT sur l'intervalle qui
// correspond a la grandeur physique attendue, ce n'est pas une coincidence.
//
//	sante   : l'intervalle utile est [128, 254] = la moitie POSITIVE de [-1, +1].
//	          Sous bruit uniforme, environ la moitie des tirages tomberait sous 128.
//	bouclier: l'intervalle utile est [0, 64] = la plage [0, 1] d'un bouclier standard
//	          dans une serialisation qui va jusqu'a 4. Sous bruit, ~75 % tomberait au-dela.
func reportQuantumSupport(name string, m map[uint8]int, total int, lo, hi uint8) {
	in, minQ, maxQ := 0, uint8(255), uint8(0)
	for q, c := range m {
		if q >= lo && q <= hi {
			in += c
		}
		if q < minQ {
			minQ = q
		}
		if q > maxQ {
			maxQ = q
		}
	}
	if total == 0 {
		return
	}
	expected := 100 * float64(int(hi)-int(lo)+1) / 256
	fmt.Printf("  TEMOIN DE FORME %-12s : quanta observes dans [%d, %d] ; %d / %d = %.2f %% dans l'intervalle\n",
		name, minQ, maxQ, in, total, pct(in, total))
	fmt.Printf("                                attendu par hasard (champ uniforme 0..255) : %.1f %%\n", expected)
}

func topQ(m map[uint8]int, n int) string {
	type kv struct {
		q uint8
		n int
	}
	var l []kv
	tot := 0
	for q, c := range m {
		l = append(l, kv{q, c})
		tot += c
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	out := ""
	for i := 0; i < n && i < len(l); i++ {
		out += fmt.Sprintf("q=%d:%.1f%% ", l[i].q, pct(l[i].n, tot))
	}
	return out
}

// ---------------------------------------------------------------- A. temoin bouclier/sante

// lifeEnds rend, par slot, l'horodatage de la DERNIERE position transmise = la fin de vie.
// C'est la source INDEPENDANTE du decodage de la vitalite (c'est aussi celle qui a produit
// la correlation 92/93 a 22 ms avec les morts horodatees du match).
func lifeEnds(s []sample) map[uint32]uint64 {
	end := map[uint32]uint64{}
	cnt := map[uint32]int{}
	for _, x := range s {
		cnt[x.slot]++
		if x.tsUS > end[x.slot] {
			end[x.slot] = x.tsUS
		}
	}
	for slot, n := range cnt {
		if n < minLifeSamples {
			delete(end, slot)
		}
	}
	return end
}

func reportShieldWitness(s []sample, end map[uint32]uint64) {
	// DEUX POPULATIONS SEPAREES, une par grandeur. Les compter sur un denominateur commun
	// (les echantillons qui portent i4) serait une erreur : i4 n'est repliqué que 0,6 % du
	// temps, le bouclier serait alors juge sur 1/240e de ses mesures, et couple a la
	// presence d'un AUTRE composant.
	preN, preZeroH := 0, 0
	aliveN, aliveZeroH := 0, 0
	preNS, preZeroS := 0, 0
	aliveNS, aliveZeroS := 0, 0
	for _, x := range s {
		te, ok := end[x.slot]
		if !ok {
			continue
		}
		dtMS := int64(te/1000) - int64(x.tsUS/1000)
		pre := dtMS >= 0 && dtMS <= preDeathMS
		alive := dtMS > aliveGuardMS
		if x.hasBody {
			if pre {
				preN++
				if x.healthQ == 0 {
					preZeroH++
				}
			} else if alive {
				aliveN++
				if x.healthQ == 0 {
					aliveZeroH++
				}
			}
		}
		if x.hasShield {
			if pre {
				preNS++
				if x.shieldQ == 0 {
					preZeroS++
				}
			} else if alive {
				aliveNS++
				if x.shieldQ == 0 {
					aliveZeroS++
				}
			}
		}
	}
	fmt.Printf("\n-- A. TEMOIN NON TAUTOLOGIQUE : sante nulle avant la mort --\n")
	fmt.Printf("  Les instants de mort viennent des FINS DE VIE des trajectoires, PAS du\n")
	fmt.Printf("  decodage de la sante. %d vies retenues (>= %d echantillons).\n", len(end), minLifeSamples)
	pH := pct(preZeroH, preN)
	aH := pct(aliveZeroH, aliveN)
	fmt.Printf("  P(sante q=0 | dans les %d ms avant une mort) = %.2f %%  (%d / %d)\n", preDeathMS, pH, preZeroH, preN)
	fmt.Printf("  P(sante q=0 | vivant, > %d s d'une mort)     = %.2f %%  (%d / %d)  <-- attendu par hasard\n",
		aliveGuardMS/1000, aH, aliveZeroH, aliveN)
	fmt.Printf("  RAPPORT = %s\n", ratio(pH, aH))
	pS := pct(preZeroS, preNS)
	aS := pct(aliveZeroS, aliveNS)
	fmt.Printf("  P(bouclier q=0 | avant la mort) = %.2f %% (%d / %d)\n", pS, preZeroS, preNS)
	fmt.Printf("  P(bouclier q=0 | vivant)        = %.2f %% (%d / %d) ; RAPPORT %s\n",
		aS, aliveZeroS, aliveNS, ratio(pS, aS))

	// La sante ne descend JAMAIS a q=0 sur ce film : le rapport demande est donc 0/0, et
	// un 0/0 ne dit rien. On publie a la place le MEME rapport sur le seul evenement de
	// sante qui existe reellement dans le corpus — « le corps a pris des degats » — avec
	// exactement les memes deux populations, donc la meme protection anti-tautologie.
	preDmg, aliveDmg := 0, 0
	var preH, aliveH, preSh, aliveSh []float64
	for _, x := range s {
		te, ok := end[x.slot]
		if !ok {
			continue
		}
		dtMS := int64(te/1000) - int64(x.tsUS/1000)
		pre := dtMS >= 0 && dtMS <= preDeathMS
		alive := dtMS > aliveGuardMS
		if x.hasBody {
			if pre {
				preH = append(preH, float64(x.health))
				if x.healthQ < 254 {
					preDmg++
				}
			} else if alive {
				aliveH = append(aliveH, float64(x.health))
				if x.healthQ < 254 {
					aliveDmg++
				}
			}
		}
		if x.hasShield {
			if pre {
				preSh = append(preSh, float64(x.shield))
			} else if alive {
				aliveSh = append(aliveSh, float64(x.shield))
			}
		}
	}
	pD := pct(preDmg, len(preH))
	aD := pct(aliveDmg, len(aliveH))
	fmt.Printf("  [substitut] P(corps ENDOMMAGE, q<254 | avant la mort) = %.2f %% (%d / %d)\n", pD, preDmg, len(preH))
	fmt.Printf("  [substitut] P(corps ENDOMMAGE, q<254 | vivant)        = %.2f %% (%d / %d) ; rapport %s\n",
		aD, aliveDmg, len(aliveH), ratio(pD, aD))
	fmt.Printf("  medianes : sante %.3f (avant la mort) contre %.3f (vivant) | bouclier %.3f contre %.3f\n",
		medianF(preH), medianF(aliveH), medianF(preSh), medianF(aliveSh))
	permutationWitness("sante", preH, aliveH)
	permutationWitness("bouclier", preSh, aliveSh)
}

// permutationWitness — TEMOIN PAR PERMUTATION. Un ecart de medianes n'est pas une preuve
// tant qu'on ne sait pas ce que le HASARD produit sur les memes donnees. On melange donc
// les etiquettes « avant la mort » / « vivant » sur l'union des deux echantillons, en
// gardant les MEMES effectifs, et on compte combien de tirages atteignent l'ecart observe.
// C'est la version non parametrique du controle negatif exige.
func permutationWitness(name string, pre, alive []float64) {
	if len(pre) < 10 || len(alive) < 10 {
		fmt.Printf("  [permutation %s] echantillon trop petit (%d / %d) — AUCUN temoin publie.\n",
			name, len(pre), len(alive))
		return
	}
	all := append(append([]float64(nil), pre...), alive...)
	obs := math.Abs(medianF(alive) - medianF(pre))
	const draws = 10000
	seed := uint64(987654321)
	hits := 0
	buf := make([]float64, len(all))
	for d := 0; d < draws; d++ {
		copy(buf, all)
		for i := len(buf) - 1; i > 0; i-- {
			seed = seed*6364136223846793005 + 1442695040888963407
			j := int(seed >> 33 % uint64(i+1))
			buf[i], buf[j] = buf[j], buf[i]
		}
		if math.Abs(medianF(buf[len(pre):])-medianF(buf[:len(pre)])) >= obs {
			hits++
		}
	}
	fmt.Printf("  [permutation %s] ecart de medianes observe %.3f ; atteint ou depasse par %d tirages sur %d (p = %.4f)\n",
		name, obs, hits, draws, float64(hits)/draws)
}

func ratio(a, b float64) string {
	if b <= 0 {
		if a <= 0 {
			return "indefini (0 / 0)"
		}
		return fmt.Sprintf("infini (%.2f %% contre 0 %%)", a)
	}
	return fmt.Sprintf("%.2f x", a/b)
}

// reportShieldRegen : SECOND temoin, independant du premier. Apres la derniere prise de
// degat, le bouclier doit remonter de facon MONOTONE puis plafonner. Une suite monotone
// croissante longue est hors de portee d'un flux desynchronise ; on la compare donc au
// nombre de suites monotones qu'une permutation aleatoire des memes valeurs produirait.
func reportShieldRegen(s []sample) {
	bySlot := map[uint32][]sample{}
	for _, x := range s {
		if x.hasShield {
			bySlot[x.slot] = append(bySlot[x.slot], x)
		}
	}
	// TEMOIN A PARITE DE DISTRIBUTION : on ne compare PAS a un melange global (les slots
	// n'ont pas la meme distribution de bouclier, ce serait un temoin trop facile). Pour
	// chaque slot, on melange l'ORDRE de SES PROPRES valeurs : meme multi-ensemble, meme
	// longueur, seul l'ordre temporel est detruit. Tout ecart mesure vient donc de l'ordre.
	const minRun = 5
	runs, total, longest := 0, 0, 0
	witness := 0
	seed := uint64(20260726)
	for _, l := range bySlot {
		q := make([]uint8, len(l))
		for i, x := range l {
			q[i] = x.shieldQ
		}
		r, lg := countRuns(q, minRun)
		runs += r
		total += len(q) - 1
		if lg > longest {
			longest = lg
		}
		sh := append([]uint8(nil), q...)
		for i := len(sh) - 1; i > 0; i-- {
			seed = seed*6364136223846793005 + 1442695040888963407
			j := int(seed >> 33 % uint64(i+1))
			sh[i], sh[j] = sh[j], sh[i]
		}
		w, _ := countRuns(sh, minRun)
		witness += w
	}
	fmt.Printf("\n-- A-bis. TEMOIN de la RECHARGE du bouclier (independant du precedent) --\n")
	fmt.Printf("  suites strictement croissantes de >= %d echantillons : %d (sur %d transitions)\n", minRun, runs, total)
	fmt.Printf("  la plus longue : %d echantillons consecutifs\n", longest)
	fmt.Printf("  TEMOIN a parite de distribution (ordre melange DANS chaque slot) : %d\n", witness)
	fmt.Printf("  Un temoin >= au reel signifie que l'ordre temporel ne porte PAS de recharge lisible.\n")
}

// countRuns compte les suites strictement croissantes d'au moins minRun elements, et rend
// la plus longue.
func countRuns(q []uint8, minRun int) (int, int) {
	runs, longest, run := 0, 1, 1
	for i := 1; i < len(q); i++ {
		if q[i] > q[i-1] {
			run++
			if run > longest {
				longest = run
			}
			if run == minRun {
				runs++
			}
			continue
		}
		run = 1
	}
	return runs, longest
}

// ---------------------------------------------------------------- B. reapparition, C. horloge

// walkFrames parcourt les paquets delta du film en decodage SEQUENTIEL et appelle fn sur
// chaque record. C'est le seul chemin qui atteint les entites ti=5 (joueur) et ti=0 (moteur
// de jeu) : le balayage bit a bit d'i0 est specifique aux bipedes.
// filmRoot est le repertoire PARENT des dossiers de films, fixe par le flag --film.
var filmRoot string

func walkFrames(repoRoot, matchID string, fn func(ts uint64, r filmdec.FrameRecord)) error {
	dir := filepath.Join(filmRoot, matchID)
	reg, err := filmdec.ParseRegistryChunk(mustChunk(dir, 0))
	if err != nil {
		return err
	}
	// SEMENCE DU MONDE : les bindings slot -> archetype viennent du keyframe du chunk 02,
	// et le parcours commence au chunk 03. C'est EXACTEMENT le protocole de cmd/tmp_l0witness
	// (deja publie) : un binding pris ailleurs, ou un delta decode avant la semence, lie les
	// slots au mauvais archetype et fait desyncher tous les records de la trame.
	w := filmdec.NewWorld(reg)
	kfChunk, err := filmdec.ReadFilmChunk(dir, seedChunk)
	if err != nil {
		return fmt.Errorf("chunk de semence %d : %w", seedChunk, err)
	}
	bound := 0
	for _, pk := range filmdec.WalkPackets(kfChunk) {
		if pk.Type != filmdec.PacketTypeKeyframe {
			continue
		}
		for _, b := range filmdec.WalkKeyframeWorld(pk.Payload(kfChunk)) {
			if b.Slot >= 0 {
				w.BindFull(uint32((b.Gen<<30)|b.Slot), uint32(b.TI))
				bound++
			}
		}
		break
	}
	if bound == 0 {
		return fmt.Errorf("aucun binding dans le keyframe du chunk %d", seedChunk)
	}
	filmdec.SetRecordStateParam(2)
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	n := filmdec.CountFilmChunks(dir)
	for c := seedChunk + 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			br := filmdec.NewBitReader(pk.Payload(data))
			recs, _ := filmdec.DecodeFrameRecords(br, w, cfg)
			for _, r := range recs {
				fn(pk.TimestampUS, r)
			}
		}
	}
	return nil
}

func mustChunk(dir string, i int) []byte {
	b, err := filmdec.ReadFilmChunk(dir, i)
	if err != nil {
		fmt.Println("chunk", i, ":", err)
		os.Exit(1)
	}
	return b
}

type respawnObs struct {
	ts     uint64
	slot   uint32
	active bool
	t0, t1 uint16
}

func reportRespawn(repoRoot, matchID string, end map[uint32]uint64) {
	var obs []respawnObs
	seen := 0
	err := walkFrames(repoRoot, matchID, func(ts uint64, r filmdec.FrameRecord) {
		for _, c := range r.Trace.Comps {
			if rt, ok := c.RespawnOf(); ok {
				seen++
				obs = append(obs, respawnObs{ts, r.Slot, rt.Active, rt.T0, rt.T1})
			}
		}
	})
	fmt.Printf("\n-- B. COMPTEUR DE REAPPARITION (ti=5 i1) --\n")
	if err != nil {
		fmt.Println("  parcours sequentiel impossible :", err)
		return
	}
	if seen == 0 {
		fmt.Printf("  AUCUNE occurrence atteinte dans le film. Le composant n'est jamais repliqué,\n")
		fmt.Printf("  ou le walk sequentiel desynchronise avant de l'atteindre. RIEN N'EST PUBLIE.\n")
		return
	}
	sort.SliceStable(obs, func(i, j int) bool { return obs[i].ts < obs[j].ts })
	act, t0max, t0nz := 0, uint16(0), 0
	for _, o := range obs {
		if o.active {
			act++
		}
		if o.t0 > t0max {
			t0max = o.t0
		}
		if o.t0 != 0 {
			t0nz++
		}
	}
	fmt.Printf("  occurrences atteintes : %d ; actif=true : %d ; t0 non nul : %d ; t0 max : %d\n",
		seen, act, t0nz, t0max)
	fmt.Printf("  unite candidate d'apres t0max : ticks 60 Hz -> %.2f s | centisecondes -> %.2f s | decisecondes -> %.1f s\n",
		float64(t0max)/60, float64(t0max)/100, float64(t0max)/10)

	// TEMOIN : le compteur doit demarrer a l'instant d'une mort CONNUE. On mesure l'ecart
	// entre chaque demarrage (passage a une valeur haute) et la fin de vie la plus proche,
	// puis on compare a un temoin ou les morts sont retirees au hasard sur la meme plage.
	starts := respawnStarts(obs)
	if len(starts) == 0 {
		fmt.Printf("  aucun DEMARRAGE de compteur detecte -> le temoin ne peut pas etre construit.\n")
		return
	}
	deaths := make([]uint64, 0, len(end))
	for _, te := range end {
		deaths = append(deaths, te)
	}
	sort.Slice(deaths, func(i, j int) bool { return deaths[i] < deaths[j] })
	real := nearestGaps(starts, deaths)
	lo, hi := deaths[0], deaths[len(deaths)-1]
	fake := make([]uint64, len(deaths))
	seed := uint64(424242)
	for i := range fake {
		seed = seed*6364136223846793005 + 1442695040888963407
		fake[i] = lo + seed>>11%(hi-lo+1)
	}
	sort.Slice(fake, func(i, j int) bool { return fake[i] < fake[j] })
	witness := nearestGaps(starts, fake)
	fmt.Printf("  demarrages de compteur detectes : %d ; morts connues : %d\n", len(starts), len(deaths))
	fmt.Printf("  ecart MEDIAN demarrage <-> mort la plus proche : %.0f ms\n", medianF(real))
	fmt.Printf("  TEMOIN, memes demarrages contre des morts TIREES AU HASARD sur la meme plage : %.0f ms\n", medianF(witness))
}

// respawnStarts detecte les fronts montants du compteur : une occurrence dont t0 depasse
// nettement la precedente du meme slot (le compteur decroit ensuite).
func respawnStarts(obs []respawnObs) []uint64 {
	prev := map[uint32]uint16{}
	var out []uint64
	for _, o := range obs {
		p, had := prev[o.slot]
		if o.t0 > p+50 || (!had && o.t0 > 50) {
			out = append(out, o.ts)
		}
		prev[o.slot] = o.t0
	}
	return out
}

func nearestGaps(starts, refs []uint64) []float64 {
	if len(refs) == 0 {
		return nil
	}
	out := make([]float64, 0, len(starts))
	for _, s := range starts {
		best := math.MaxFloat64
		for _, r := range refs {
			d := math.Abs(float64(int64(s)-int64(r))) / 1000
			if d < best {
				best = d
			}
		}
		out = append(out, best)
	}
	return out
}

type clockObs struct {
	ts     uint64
	a, b   float32
	qa, qb uint16
}

func reportClock(repoRoot, matchID string) {
	var obs []clockObs
	err := walkFrames(repoRoot, matchID, func(ts uint64, r filmdec.FrameRecord) {
		for _, c := range r.Trace.Comps {
			if rt, ok := c.RoundTimerOf(); ok {
				obs = append(obs, clockObs{ts, rt.A, rt.B, rt.QA, rt.QB})
			}
		}
	})
	fmt.Printf("\n-- C. HORLOGE DE MANCHE (ti=0 i5) --\n")
	if err != nil {
		fmt.Println("  parcours sequentiel impossible :", err)
		return
	}
	if len(obs) == 0 {
		fmt.Printf("  AUCUNE occurrence atteinte. Le composant n'est jamais repliqué dans ce film,\n")
		fmt.Printf("  ou le walk desynchronise avant de l'atteindre. RIEN N'EST PUBLIE.\n")
		return
	}
	sort.SliceStable(obs, func(i, j int) bool { return obs[i].ts < obs[j].ts })
	t0 := obs[0].ts
	var xs, ya, yb []float64
	for _, o := range obs {
		xs = append(xs, float64(o.ts-t0)/1e6)
		ya = append(ya, float64(o.a))
		yb = append(yb, float64(o.b))
	}
	sa, ra := slope(xs, ya)
	sb, rb := slope(xs, yb)
	fmt.Printf("  occurrences : %d ; duree couverte : %.1f s\n", len(obs), xs[len(xs)-1])
	fmt.Printf("  champ A : %.3f -> %.3f s ; pente %.4f s/s ; R2 %.4f\n", ya[0], ya[len(ya)-1], sa, ra)
	fmt.Printf("  champ B : %.3f -> %.3f s ; pente %.4f s/s ; R2 %.4f\n", yb[0], yb[len(yb)-1], sb, rb)
	fmt.Printf("  ATTENDU pour un chrono : pente -1 (restant) ou +1 (ecoule), R2 proche de 1.\n")
	fmt.Printf("  Une pente quelconque avec un R2 bas = le champ N'EST PAS un chrono.\n")
	// Second controle, independant de la regression : un chrono de match vit dans une
	// plage etroite (quelques centaines de secondes). La serialisation, elle, va jusqu'a
	// 36000 s. Mesurer la part des valeurs plausibles, contre la part attendue d'un champ
	// uniforme sur toute la plage serialisee.
	plaus := 0
	for _, v := range ya {
		if v <= 1200 {
			plaus++
		}
	}
	fmt.Printf("  champ A dans [0, 1200] s (plage plausible d'un match) : %.1f %% ; attendu d'un champ uniforme sur [0, 36000] : 3.3 %%\n",
		pct(plaus, len(ya)))
	fmt.Printf("  duree reelle couverte par le film : %.1f s — a recouper avec duration_seconds en base.\n", xs[len(xs)-1])
}

func slope(x, y []float64) (float64, float64) {
	n := float64(len(x))
	if n < 2 {
		return 0, 0
	}
	var sx, sy, sxy, sxx, syy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
		sxy += x[i] * y[i]
		sxx += x[i] * x[i]
		syy += y[i] * y[i]
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0, 0
	}
	s := (n*sxy - sx*sy) / den
	num := n*sxy - sx*sy
	r2den := (n*sxx - sx*sx) * (n*syy - sy*sy)
	if r2den <= 0 {
		return s, 0
	}
	r := num / math.Sqrt(r2den)
	return s, r * r
}

// reportAimQuality — NON-REGRESSION DU CAP DE VISEE, la contrepartie obligatoire de
// l'ajout d'i3 au balayage. Aller plus loin dans le record augmente la COUVERTURE de la
// visee ; il faut prouver que la QUALITE ne se degrade pas, sinon on a simplement ajoute du
// bruit. Metrique deja au dossier : ecart median entre le cap decode et la direction de
// DEPLACEMENT reelle du meme slot, contre un temoin ou le deplacement est tire d'un autre
// echantillon au hasard (attendu ~90 deg).
func reportAimQuality(pos []filmdec.BipedPosition) {
	bySlot := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			bySlot[p.Slot] = append(bySlot[p.Slot], p)
		}
	}
	// Une paire = deux echantillons du MEME slot separes d'au moins 100 ms ET d'au moins
	// 50 cm : en deca, la « direction de deplacement » est du bruit de quantification et la
	// comparaison ne dit rien. Le TEMOIN garde les memes caps et les memes deplacements,
	// mais les reapparie AU HASARD dans le meme slot : meme distribution des deux cotes,
	// seul l'appariement temporel est detruit.
	var real, aims, moves []float64
	for _, l := range bySlot {
		var sa, sm []float64
		for i := 1; i < len(l); i++ {
			j := i - 1
			for j > 0 && (l[i].TimestampUS-l[j].TimestampUS) < 100000 {
				j--
			}
			dx, dy := float64(l[i].X-l[j].X), float64(l[i].Y-l[j].Y)
			if dx*dx+dy*dy < 0.25 {
				continue
			}
			h, ok := l[i].AimHeadingDeg()
			if !ok {
				continue
			}
			mv := math.Atan2(dy, dx) * 180 / math.Pi
			real = append(real, angDiff(float64(h), mv))
			sa, sm = append(sa, float64(h)), append(sm, mv)
		}
		aims, moves = append(aims, sa...), append(moves, sm...)
		_ = sm
	}
	fmt.Printf("\n-- NON-REGRESSION du cap de visee (contrepartie de l'ajout d'i3) --\n")
	if len(real) < 100 {
		fmt.Printf("  echantillon insuffisant (%d) — aucun temoin publie.\n", len(real))
		return
	}
	seed := uint64(13579)
	ctrl := make([]float64, 0, len(aims))
	for i := range aims {
		seed = seed*6364136223846793005 + 1442695040888963407
		ctrl = append(ctrl, angDiff(aims[i], moves[int((seed>>33)%uint64(len(moves)))]))
	}
	fmt.Printf("  paires (cap decode, deplacement reel) : %d\n", len(real))
	fmt.Printf("  ecart MEDIAN cap <-> deplacement : %.1f deg\n", medianF(real))
	fmt.Printf("  TEMOIN, caps et deplacements reapparies au hasard : %.1f deg (attendu ~90)\n", medianF(ctrl))
}

func angDiff(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// reportWalkReach — DIAGNOSTIC D'ATTEIGNABILITE. Sans lui, « 0 occurrence » est
// ininterpretable : on ne saurait pas distinguer « le composant n'est jamais repliqué » de
// « le walk n'arrive jamais jusqu'a cette entite ». C'est l'artefact d'outil qui a deja
// piege ce projet (un scan a conclu « 0 occurrence » sur un corpus lisible a 5 %).
func reportWalkReach(repoRoot, matchID string) {
	byTI := map[uint32]int{}
	desyncTI := map[uint32]int{}
	desyncAt := map[string]int{}
	compSeen := map[string]int{}
	err := walkFrames(repoRoot, matchID, func(_ uint64, r filmdec.FrameRecord) {
		byTI[r.TypeIndex]++
		if r.DesyncAt >= 0 {
			desyncTI[r.TypeIndex]++
			if r.DesyncAt < len(r.Trace.Comps) {
				c := r.Trace.Comps[len(r.Trace.Comps)-1]
				desyncAt[fmt.Sprintf("ti=%d i%02d %s", r.TypeIndex, c.Index, c.Name)]++
			}
		}
		for _, c := range r.Trace.Comps {
			if c.Ported {
				compSeen[c.Name]++
			}
		}
	})
	fmt.Printf("\n-- DIAGNOSTIC : ce que le parcours SEQUENTIEL atteint reellement --\n")
	if err != nil {
		fmt.Println("  parcours impossible :", err)
		return
	}
	type kv struct {
		k string
		n int
	}
	var tis []kv
	tot := 0
	for ti, n := range byTI {
		tis = append(tis, kv{fmt.Sprintf("ti=%-3d", ti), n})
		tot += n
	}
	sort.Slice(tis, func(i, j int) bool { return tis[i].n > tis[j].n })
	fmt.Printf("  %d records decodes, repartis sur %d archetypes :\n", tot, len(byTI))
	for i := 0; i < 10 && i < len(tis); i++ {
		fmt.Printf("      %s %6d\n", tis[i].k, tis[i].n)
	}
	for _, ti := range []uint32{0, 5} {
		fmt.Printf("  archetype ti=%d (%s) : %d records atteints, dont %d en desync\n",
			ti, map[uint32]string{0: "game-engine, porteur de l'horloge", 5: "player, porteur du compteur de reapparition"}[ti],
			byTI[ti], desyncTI[ti])
	}
	var ds []kv
	for k, n := range desyncAt {
		ds = append(ds, kv{k, n})
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].n > ds[j].n })
	fmt.Printf("  dernier composant consomme avant desync (top 6) :\n")
	for i := 0; i < 6 && i < len(ds); i++ {
		fmt.Printf("      %-56s %d\n", ds[i].k, ds[i].n)
	}
	for _, n := range []string{"player-waypoint-component", "player-respawn-timer-component",
		"game-engine-team-mapping-component", "game-engine-round-timer-component"} {
		fmt.Printf("  composant %-40s consomme %d fois\n", n, compSeen[n])
	}
}

// ---------------------------------------------------------------- utilitaires

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func medianF(v []float64) float64 {
	if len(v) == 0 {
		return math.NaN()
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

func writeCSV(path string, s []sample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"ts_us", "slot", "x", "y", "has_body", "health_q", "health", "has_shield", "shield_q", "shield"})
	for _, x := range s {
		_ = w.Write([]string{
			strconv.FormatUint(x.tsUS, 10), strconv.FormatUint(uint64(x.slot), 10),
			fmt.Sprintf("%.3f", x.x), fmt.Sprintf("%.3f", x.y),
			strconv.FormatBool(x.hasBody), strconv.Itoa(int(x.healthQ)), fmt.Sprintf("%.5f", x.health),
			strconv.FormatBool(x.hasShield), strconv.Itoa(int(x.shieldQ)), fmt.Sprintf("%.5f", x.shield),
		})
	}
	return nil
}
