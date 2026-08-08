package replay

// ctf_closure_research_test.go — INSTRUMENT DE RECHERCHE #3 (v7.5 voie B).
//
// # CE QU'IL MESURE, ET POURQUOI CE N'EST PAS LE VOTE QU'ON A RETIRÉ
//
// Les instruments #1 et #2 ont établi que les tirs perdus tombent dans des vies que le fil des
// morts ne nomme pas. Celui-ci chiffre une FERMETURE : à tout instant le film porte huit bipèdes
// vivants ; quand un joueur tire alors qu'aucune de ses vies NOMMÉES ne le couvre, son bipède est
// forcément l'une des vies non nommées vivantes à cet instant — **et s'il n'y en a qu'une, elle
// est la sienne par élimination.**
//
// La différence avec le repli voté supprimé le 2026-07-28 est de nature, pas de degré :
//
//	le vote      plusieurs candidats, on garde le mieux placé      -> un CHOIX
//	la fermeture un seul candidat possible, les autres sont exclus -> une DÉDUCTION
//
// Un tir dont deux vies non nommées peuvent porter l'auteur n'est JAMAIS attribué ici.
//
// # LE CONTRÔLE QUI PEUT LA RÉFUTER, POSÉ AVANT LA MESURE
//
// Un joueur n'a qu'un corps. Si une vie attribuée par fermeture RECOUVRE dans le temps une vie
// déjà nommée du même joueur, l'attribution est impossible et elle est REJETÉE, comptée à part.
// Un taux de rejet élevé dirait que la fermeture attrape autre chose que des bipèdes de joueur
// (ragdolls, acteurs) et que la méthode ne vaut rien. C'est le pendant du critère « huit entités
// distinctes » qui a réfuté la piste i19.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_CLOSURE_FILMS="64e8adfa:Catalyst,..." \
//	  go test ./internal/analysis/replay/ -run CTFExclusionClosure -timeout 60m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

const ctfClosureFilmsEnv = "CTF_CLOSURE_FILMS"

// ctfRespawnHalfWidthMS : demi-largeur de la fenêtre de réapparition SERRÉE, en millisecondes.
// La réapparition est déterministe — sur les sept films, l'écart entre le centile 5 et la
// médiane vaut de 2 ms à 202 ms. 750 ms couvre donc largement la dispersion réelle tout en
// restant vingt fois plus étroit que l'intervalle entre deux morts consécutives d'un match.
const ctfRespawnHalfWidthMS = 750

// closureStats porte le bilan d'une fermeture.
type closureStats struct {
	unnamedLives   int // vies non nommées dont le slot n'est pas déjà au pont
	candidates     int // vies pour lesquelles au moins un tir a désigné un joueur
	assigned       int // vies attribuées (un seul joueur désigné, contrôle passé)
	contested      int // vies que deux joueurs se disputent : jamais attribuées
	rejectedByTime int // vies rejetées par le contrôle de recouvrement
	forcedShots    int // tirs orphelins dont l'instant ne portait qu'UNE vie non nommée
	orphanShots    int // tirs orphelins au total
}

func TestCTFExclusionClosure(t *testing.T) {
	spec := os.Getenv(ctfClosureFilmsEnv)
	if spec == "" {
		t.Skipf("fermeture non demandée : %s vide", ctfClosureFilmsEnv)
	}
	cache, outDir := os.Getenv(ctfCacheEnv), os.Getenv(ctfOutEnv)
	if cache == "" || outDir == "" {
		t.Fatalf("%s et %s sont requis", ctfCacheEnv, ctfOutEnv)
	}
	cat := loadCTFQuantCatalog(t)
	for _, item := range strings.Split(spec, ",") {
		short, mapName, ok := strings.Cut(strings.TrimSpace(item), ":")
		if !ok {
			t.Fatalf("entrée mal formée %q", item)
		}
		t.Run(short, func(t *testing.T) {
			b := ctfClosureReport(t, cat, filepath.Join(cache, "film_chunks", short), short, mapName)
			if err := os.WriteFile(filepath.Join(outDir, short+"_fermeture.txt"), []byte(b), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			t.Logf("\n%s", b)
		})
	}
}

func ctfClosureReport(t *testing.T, cat *filmdec.MapQuantCatalog, dir, short, mapName string) string {
	t.Helper()
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("bornes de %s : %v", mapName, err)
	}
	world := entry.Range()
	scan := filmdec.DefaultScanFilmOptions()
	scan.WorldRange, scan.CaptureDirs = &world, true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	fire, err := filmdec.ScanFilmFireEvents(dir)
	if err != nil {
		t.Fatalf("tirs : %v", err)
	}
	deaths, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("morts : %v", err)
	}
	idx, err := ScanFilmPlayerIndices(dir, rosterFromDeaths(deaths))
	if err != nil {
		t.Fatalf("index : %v", err)
	}
	table, _ := injectiveOrEmpty(idx)
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	tracks := indexBySlot(pos)
	own := buildOwners(tracks, deaths, table)

	lives := buildLifeSpans(tracks)
	off, _ := bestDeathOffset(lives, deaths)
	nameLivesByDeaths(lives, deaths, off)

	// AVANT : la couverture telle que la production la produit aujourd'hui.
	before := ctfCoverageOf(tracks, own.Owner, fire)
	augmented, st := ctfCloseByExclusion(tracks, own.Owner, lives, fire)
	after := ctfCoverageOf(tracks, augmented, fire)
	// FERMETURE B, appliquée SUR le résultat de A : les deux ne se disputent rien (A n'agit que
	// là où une seule vie est vivante, B là où une seule mort précède).
	p05, p50, p95, nCal := ctfRespawnCalib(lives, deaths, off)
	// DEUX FENÊTRES, et la comparaison est le résultat. [p05,p95] paraît prudente mais le p95
	// est pollué par les vies dont la mort précédente du même joueur n'est PAS celle qui les a
	// fait réapparaître (premières vies) : il monte à 51 s et 67 s sur deux films, ce qui rend
	// toute vie contestée. La réapparition étant DÉTERMINISTE (p05 et médiane à 2 ms près), la
	// fenêtre juste est serrée autour de la médiane.
	augB, stB := ctfCloseByRespawn(tracks, augmented, lives, deaths, off, table.ByXUID, p05, p95)
	afterB := ctfCoverageOf(tracks, augB, fire)
	augT, stT := ctfCloseByRespawn(tracks, augmented, lives, deaths, off, table.ByXUID,
		p50-ctfRespawnHalfWidthMS, p50+ctfRespawnHalfWidthMS)
	afterT := ctfCoverageOf(tracks, augT, fire)

	var b strings.Builder
	fmt.Fprintf(&b, "film\t%s\ncarte\t%s\n", short, mapName)
	fmt.Fprintf(&b, "slots_pont_avant\t%d\nslots_pont_apres\t%d\n", len(own.Owner), len(augmented))
	fmt.Fprintf(&b, "vies_non_nommees_candidates\t%d\ndesignees_par_un_tir\t%d\nattribuees\t%d\tcontestees\t%d\trejetees_recouvrement\t%d\n",
		st.unnamedLives, st.candidates, st.assigned, st.contested, st.rejectedByTime)
	fmt.Fprintf(&b, "tirs_orphelins\t%d\tdont_instant_a_une_seule_vie\t%d\n", st.orphanShots, st.forcedShots)
	fmt.Fprintf(&b, "\n# couverture des tirs\n")
	fmt.Fprintf(&b, "avant\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		before.Attached, before.Available, ratio(before.Attached, before.Available), before.NoSlot, before.Ambiguous)
	fmt.Fprintf(&b, "apres\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		after.Attached, after.Available, ratio(after.Attached, after.Available), after.NoSlot, after.Ambiguous)
	fmt.Fprintf(&b, "gain_points_A\t%.2f\n",
		100*(ratio(after.Attached, after.Available)-ratio(before.Attached, before.Available)))
	fmt.Fprintf(&b, "\n# fermeture B — par la reapparition\n")
	fmt.Fprintf(&b, "calibration_vies\t%d\tp05_ms\t%d\tmediane_ms\t%d\tp95_ms\t%d\n", nCal, p05, p50, p95)
	fmt.Fprintf(&b, "vies_restantes\t%d\tdesignees\t%d\tattribuees\t%d\tcontestees\t%d\trejetees_recouvrement\t%d\n",
		stB.unnamedLives, stB.candidates, stB.assigned, stB.contested, stB.rejectedByTime)
	fmt.Fprintf(&b, "apresB\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		afterB.Attached, afterB.Available, ratio(afterB.Attached, afterB.Available), afterB.NoSlot, afterB.Ambiguous)
	fmt.Fprintf(&b, "gain_points_total_fenetre_large\t%.2f\n",
		100*(ratio(afterB.Attached, afterB.Available)-ratio(before.Attached, before.Available)))
	fmt.Fprintf(&b, "\n# fermeture B — fenetre SERREE (mediane +/- %d ms)\n", ctfRespawnHalfWidthMS)
	fmt.Fprintf(&b, "vies_restantes\t%d\tdesignees\t%d\tattribuees\t%d\tcontestees\t%d\trejetees_recouvrement\t%d\n",
		stT.unnamedLives, stT.candidates, stT.assigned, stT.contested, stT.rejectedByTime)
	fmt.Fprintf(&b, "apresT\trattaches\t%d\tsur\t%d\ttaux\t%.4f\tsans_slot\t%d\tambigus\t%d\n",
		afterT.Attached, afterT.Available, ratio(afterT.Attached, afterT.Available), afterT.NoSlot, afterT.Ambiguous)
	fmt.Fprintf(&b, "gain_points_total_fenetre_serree\t%.2f\n",
		100*(ratio(afterT.Attached, afterT.Available)-ratio(before.Attached, before.Available)))
	return b.String()
}

// ctfCoverageOf rend la couverture du calque des tirs pour un pont donné. C'est le MÊME
// enchaînement que buildShots, sans le filtre des trajectoires publiées (mesuré nul sur les
// sept films) : on compare deux ponts, pas deux pipelines.
func ctfCoverageOf(tracks map[uint32]slotTrack, owner map[uint32]int, fire []filmdec.FireEvent) LayerCoverage {
	cov := LayerCoverage{Available: len(fire)}
	for _, e := range fire {
		_, reason := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS)
		cov.count(reason)
	}
	return cov
}

// ctfCloseByExclusion referme la bijection et rend le pont augmenté.
func ctfCloseByExclusion(tracks map[uint32]slotTrack, owner map[uint32]int,
	lives []lifeSpan, fire []filmdec.FireEvent) (map[uint32]int, closureStats) {
	var st closureStats
	free := ctfFreeLives(owner, lives)
	st.unnamedLives = len(free)
	claims := map[uint32]map[int]int{} // slot -> index de joueur -> nombre de tirs
	for _, e := range fire {
		if _, r := slotFor(tracks, owner, e.FilmIndex, e.TimestampUS); r == reasonAttached {
			continue
		}
		st.orphanShots++
		alive := ctfLivesAt(free, tracks, e.TimestampUS)
		if len(alive) != 1 { // deux corps possibles : on ne tranche pas
			continue
		}
		st.forcedShots++
		if claims[alive[0].slot] == nil {
			claims[alive[0].slot] = map[int]int{}
		}
		claims[alive[0].slot][e.FilmIndex]++
	}
	st.candidates = len(claims)
	out := make(map[uint32]int, len(owner)+len(claims))
	for s, pi := range owner {
		out[s] = pi
	}
	for _, slot := range ctfSortedSlots(claims) {
		if len(claims[slot]) != 1 { // deux joueurs se disputent le corps
			st.contested++
			continue
		}
		pi := ctfOnlyKey(claims[slot])
		if ctfOverlapsNamed(tracks, owner, slot, pi) {
			st.rejectedByTime++ // un joueur n'a qu'un corps : impossible
			continue
		}
		out[slot] = pi
		st.assigned++
	}
	return out, st
}

// ctfFreeLives rend les vies sans identité dont le slot n'est pas DÉJÀ au pont : un slot déjà
// nommé par une autre de ses vies n'a rien à déduire.
func ctfFreeLives(owner map[uint32]int, lives []lifeSpan) []lifeSpan {
	var out []lifeSpan
	for _, l := range lives {
		if l.xuid != 0 {
			continue
		}
		if _, known := owner[l.slot]; known {
			continue
		}
		out = append(out, l)
	}
	return out
}

// ctfLivesAt rend les vies libres qui portent une position à moins de la tolérance de tUS —
// « vivantes » au sens du rattachement, pas au sens du jeu.
func ctfLivesAt(free []lifeSpan, tracks map[uint32]slotTrack, tUS uint64) []lifeSpan {
	var out []lifeSpan
	t := int64(tUS)
	for _, l := range free {
		if t < l.from || t > l.to {
			continue
		}
		if _, d := tracks[l.slot].at(tUS); d <= shotPosToleranceUS {
			out = append(out, l)
		}
	}
	return out
}

// ctfOverlapsNamed dit si le slot candidat porte des positions en même temps qu'un slot DÉJÀ
// attribué au même joueur. C'est le contrôle qui peut réfuter la fermeture.
func ctfOverlapsNamed(tracks map[uint32]slotTrack, owner map[uint32]int, slot uint32, pi int) bool {
	cand := tracks[slot].pts
	if len(cand) == 0 {
		return false
	}
	from, to := cand[0].TimestampUS, cand[len(cand)-1].TimestampUS
	for s, p := range owner {
		if p != pi || s == slot {
			continue
		}
		pts := tracks[s].pts
		if len(pts) == 0 {
			continue
		}
		if pts[0].TimestampUS <= to && from <= pts[len(pts)-1].TimestampUS {
			return true
		}
	}
	return false
}

// ============================ FERMETURE B — PAR LA RÉAPPARITION ============================
//
// Une vie ne commence pas de nulle part : elle commence UNE RÉAPPARITION APRÈS LA MORT qui l'a
// causée, et le fil des morts NOMME la victime de cette mort. Si une seule mort du fil tombe dans
// la fenêtre de réapparition qui précède le début d'une vie anonyme, cette vie est celle de sa
// victime — encore une déduction, pas un vote.
//
// LA FENÊTRE N'EST PAS SUPPOSÉE, ELLE EST CALIBRÉE sur les vies DÉJÀ NOMMÉES du même film :
// on mesure l'écart entre le début d'une vie nommée et la mort précédente de son propre joueur,
// et on en prend les centiles. Un réglage importé d'un autre film serait une supposition.

// ctfRespawnCalib rend les centiles (p05, p50, p95) de l'écart début-de-vie <-> mort précédente
// du MÊME joueur, en millisecondes, mesurés sur les vies nommées.
func ctfRespawnCalib(lives []lifeSpan, deaths []Death, off int64) (int64, int64, int64, int) {
	var d []int64
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		best := int64(-1)
		for _, dd := range deaths {
			if dd.XUID != l.xuid {
				continue
			}
			if t := dd.TimeMS + off; t < l.from/1000 && (best < 0 || t > best) {
				best = t
			}
		}
		if best >= 0 {
			d = append(d, l.from/1000-best)
		}
	}
	if len(d) == 0 {
		return 0, 0, 0, 0
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	return d[len(d)*5/100], d[len(d)/2], d[len(d)*95/100], len(d)
}

// ctfCloseByRespawn attribue les vies anonymes dont le début ne suit QU'UNE SEULE mort du fil
// dans la fenêtre calibrée.
func ctfCloseByRespawn(tracks map[uint32]slotTrack, owner map[uint32]int, lives []lifeSpan,
	deaths []Death, off int64, byXUID map[uint64]int, lo, hi int64) (map[uint32]int, closureStats) {
	var st closureStats
	free := ctfFreeLives(owner, lives)
	st.unnamedLives = len(free)
	out := make(map[uint32]int, len(owner)+len(free))
	for s, pi := range owner {
		out[s] = pi
	}
	for _, l := range free {
		var cand []uint64
		for _, dd := range deaths {
			delta := l.from/1000 - (dd.TimeMS + off)
			if delta >= lo && delta <= hi && !ctfHasXUID(cand, dd.XUID) {
				cand = append(cand, dd.XUID)
			}
		}
		if len(cand) == 0 {
			continue
		}
		st.candidates++
		if len(cand) > 1 {
			st.contested++
			continue
		}
		pi, ok := byXUID[cand[0]]
		if !ok {
			continue
		}
		if ctfOverlapsNamed(tracks, out, l.slot, pi) {
			st.rejectedByTime++
			continue
		}
		out[l.slot] = pi
		st.assigned++
	}
	return out, st
}

func ctfHasXUID(xs []uint64, x uint64) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func ctfSortedSlots(m map[uint32]map[int]int) []uint32 {
	out := make([]uint32, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func ctfOnlyKey(m map[int]int) int {
	for k := range m {
		return k
	}
	return -1
}
