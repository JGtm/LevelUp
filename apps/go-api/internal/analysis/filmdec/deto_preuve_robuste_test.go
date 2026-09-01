package filmdec

// deto_preuve_robuste_test.go — LA PREUVE : l'attribution DETONATION->tireur des touches
// explosives confrontee au VRAI TUEUR (dead-state), avec un scan de kills ROBUSTE.
//
// TestDetoAttribution/M5 mesurait deja l'accord DETONATION->tireur == tueur dead-state, mais sur
// une recolte de morts data-starved (`geoCollectDamageKills`, 0-5 morts, 0 kill explosif). Ce
// test rejoue EXACTEMENT la meme mesure M5 en changeant la SEULE chose qui manquait : la source
// des morts devient `robustCollectKills` (marche + localisateur + 8 vues, cf.
// deto_preuve_robuste_helpers_test.go). Il ajoute un TEMOIN tireur ALEATOIRE pour chiffrer la
// part de hasard de l'accord.
//
// Garde LOT1_TRAME_FILM (+ LOT1_MAXCHUNKS pour moissonner) et LOT1_SONDE_MAP. Un film par
// process, verrou pris ici.

import (
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// detoProofRandomWitness : accord d'une attribution ALEATOIRE au vrai tueur, sur les MEMES kills
// explosifs relies que la these. Pour chaque kill relie, on tire un FilmIndex au hasard dans le
// roster observe (les valeurs distinctes de la table roster->FilmIndex) et on regarde s'il tombe
// sur le vrai tueur. Moyenne sur `trials` graines. C'est le plancher de hasard = ~1/|roster|.
func detoProofRandomWitness(c detoGTCtx, roster []int, trials int) (okRand, evalRand int) {
	if len(roster) == 0 {
		return 0, 0
	}
	rng := rand.New(rand.NewSource(20260901))
	for _, k := range c.kills {
		trueFilm, ok := c.table[k.killer]
		if !ok {
			continue
		}
		vp, okv := geoLookup(c.tracks[k.victSlot], k.ts, detoPosTol)
		if !okv {
			continue
		}
		d, ok := detoLinkFatalDeton(vp, k.ts, c.detons, detoGTRadius)
		if !ok {
			continue
		}
		// le kill est explosif (relie a une detonation source) : on l'evalue.
		_ = d
		for tr := 0; tr < trials; tr++ {
			evalRand++
			if roster[rng.Intn(len(roster))] == trueFilm {
				okRand++
			}
		}
	}
	return okRand, evalRand
}

// TestDetoPreuveRobuste produit la preuve terrain de l'attribution detonation->tireur.
func TestDetoPreuveRobuste(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : preuve sautee", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	maxCh := geoMaxChunks
	if v := os.Getenv("LOT1_MAXCHUNKS"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			maxCh = k
		}
	}
	if n > maxCh {
		n = maxCh
	}
	wr := sondeWorldRange(t, dir)
	if wr == nil {
		t.Skipf("bornes monde absentes : la preuve exige des positions (renseigner %s)", sondeMapEnv)
	}

	shots := geoCollectShots(t, dir, n)
	raws, grossKills := geoCollectDamageKills(t, dir, reg, n)
	robKills := robustCollectKills(t, dir, reg, n)
	tracks := geoTracks(t, dir, wr, n)
	detons := detoScanProjectiles(t, dir, wr, n)

	geoActiveBase = geoDetectBase(raws)
	defer func() { geoActiveBase = geoBase }()
	touch := geoBuildTouches(raws, geoActiveBase)
	geoMatchFatal(touch, robKills)

	var heavy []geoShot
	for _, s := range shots {
		if s.heavy {
			heavy = append(heavy, s)
		}
	}
	sort.Slice(heavy, func(i, j int) bool { return heavy[i].ts < heavy[j].ts })
	speedByWid := detoM2Speed(t, detons, heavy, tracks)

	table, card, inj := geoBuildIdentity(shots, robKills)
	roster := rosterFilmIndices(table)

	t.Logf("== PREUVE film %s · %d chunks · base %d ==", filepath.Base(dir), n, geoActiveBase)
	t.Logf("SCAN DE KILLS : grossier (geoCollectDamageKills) %d morts  VS  robuste (marche+localisateur) %d morts",
		len(grossKills), len(robKills))
	t.Logf("identite roster<->FilmIndex : %d mappes (injective %v) · roster observe %d joueurs",
		card, inj, len(roster))

	ctx := detoGTCtx{detons: detons, touch: touch, kills: robKills, heavy: heavy, tracks: tracks, speed: speedByWid, table: table}
	detoM5GroundTruth(t, ctx)

	okRand, evalRand := detoProofRandomWitness(ctx, roster, 50)
	t.Logf("TEMOIN TIREUR ALEATOIRE (50 tirages/kill, plancher de hasard ~ 1/%d) : %d/%d (%.1f %%)",
		len(roster), okRand, evalRand, lot1Pct(okRand, evalRand))
	t.Logf("LECTURE : la these est PROUVEE si DETONATION->tireur bat NETTEMENT et le temoin aleatoire")
	t.Logf("et la voie VICTIME->tireur ; NON CONCLUANTE si l'echantillon de kills explosifs relies < 3.")
}

// rosterFilmIndices : les FilmIndex distincts de la table roster->FilmIndex, tries.
func rosterFilmIndices(table map[int32]int) []int {
	seen := map[int]bool{}
	var out []int
	for _, f := range table {
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	sort.Ints(out)
	return out
}
