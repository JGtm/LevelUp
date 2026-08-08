package replay

// ctf_fatal_research_test.go — INSTRUMENT DE RECHERCHE #4 (v7.5 voie B).
//
// # LA QUESTION, POSÉE PAR L'UTILISATEUR LE 2026-08-08
//
// « Moi en priorité je veux les tirs fatals — quelle couverture on a sur les tirs qui ont causé
// des morts ? » Les instruments précédents mesurent TOUS les tirs indistinctement. Or un tir
// fatal ne vaut pas un tir de suppression : c'est celui qu'on veut voir sur la carte.
//
// # CE QUI EST MESURÉ, ET LA CHAÎNE QU'IL FAUT TENIR
//
// Un tir fatal posé sur la carte exige QUATRE maillons, et le plus faible commande :
//
//	1. la mort est connue         événement DEATH du fil (type-3), daté à la ms
//	2. le tueur est connu          événement KILL du même fil, apparié à la mort
//	3. un tir du tueur est daté    un record de tir (type 105) porte l'index du tueur
//	4. la position du tueur existe le pont slot -> joueur couvre ce tir
//
// Ce fichier chiffre chaque maillon SÉPARÉMENT : dire « 80 % » sans dire lequel des quatre
// manque n'oriente aucun chantier.
//
// # LA FENÊTRE N'EST PAS SUPPOSÉE
//
// L'écart entre le dernier tir du tueur et la mort de sa victime est MESURÉ (centiles publiés)
// avant d'être employé comme critère. Une fenêtre choisie a priori ferait dire au chiffre ce
// qu'on veut.
//
//	CGO_ENABLED=0 FILM_CACHE_ROOT=<mainrepo>/data/cache CTF_RESEARCH_OUT=<dir> \
//	  CTF_FATAL_FILMS="64e8adfa:Catalyst,..." \
//	  go test ./internal/analysis/replay/ -run CTFFatalShots -timeout 90m

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const ctfFatalFilmsEnv = "CTF_FATAL_FILMS"

// ctfFatalWindowMS : fenêtre amont retenue pour dire qu'un tir a pu causer une mort. Elle est
// large À DESSEIN — l'objet de la mesure est la COUVERTURE (le tir existe-t-il et sait-on le
// placer), pas l'imputation exacte du coup mortel, qui relève du chantier arme-par-kill.
const ctfFatalWindowMS = 1500

// ctfKillPair est une mort appariée à son tueur, sur l'horloge du film.
type ctfKillPair struct {
	killerXUID, victimXUID uint64
	filmUS                 uint64
	// timeMS est l'instant sur l'horloge du MATCH, conservé pour joindre la ligne de
	// `match_kill_events_latest` — c'est elle qui porte la CLASSE de la source (arme, mêlée,
	// grenade, véhicule). Sans cette jointure, on compte comme un échec de couverture une mort
	// à la mêlée, qui ne peut par construction porter aucun record de tir.
	timeMS int
}

func TestCTFFatalShots(t *testing.T) {
	spec := os.Getenv(ctfFatalFilmsEnv)
	if spec == "" {
		t.Skipf("couverture des tirs fatals non demandée : %s vide", ctfFatalFilmsEnv)
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
			b := ctfFatalReport(t, cat, filepath.Join(cache, "film_chunks", short), short, mapName)
			if err := os.WriteFile(filepath.Join(outDir, short+"_fatals.txt"), []byte(b), 0o644); err != nil {
				t.Fatalf("écriture : %v", err)
			}
			t.Logf("\n%s", b)
		})
	}
}

func ctfFatalReport(t *testing.T, cat *filmdec.MapQuantCatalog, dir, short, mapName string) string {
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
	evs := ctfHighlightEvents(t, dir)
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

	// Le pont AVANT et APRÈS les deux fermetures : la question « les tirs fatals » se juge sur
	// les deux, sinon on ne sait pas ce que le lot du §7.2 apporterait à ce qui compte le plus.
	augA, _ := ctfCloseByExclusion(tracks, own.Owner, lives, fire)
	_, p50, _, _ := ctfRespawnCalib(lives, deaths, off)
	augB, _ := ctfCloseByRespawn(tracks, augA, lives, deaths, off, table.ByXUID,
		p50-ctfRespawnHalfWidthMS, p50+ctfRespawnHalfWidthMS)

	pairs, nKills, nUnpaired := ctfPairKills(evs, off)
	var b strings.Builder
	fmt.Fprintf(&b, "film\t%s\ncarte\t%s\n", short, mapName)
	fmt.Fprintf(&b, "morts_du_fil\t%d\nevenements_kill\t%d\ncouples_tueur_victime\t%d\nkills_sans_mort_appariee\t%d\n",
		len(deaths), nKills, len(pairs), nUnpaired)
	ctfWriteFatalDelays(&b, pairs, fire, table)
	fmt.Fprintf(&b, "\n# maillon 3 et 4 — le tir du tueur existe-t-il, et sait-on le placer\n")
	ctfWriteFatalCoverage(&b, "pont_actuel", pairs, fire, tracks, own.Owner, table)
	ctfWriteFatalCoverage(&b, "pont_ferme", pairs, fire, tracks, augB, table)
	// Détail par mort, pour la jointure hors ligne avec la CLASSE de la source. Les colonnes
	// sont volontairement plates : ce bloc est fait pour être relu par DuckDB, pas par un humain.
	fmt.Fprintf(&b, "\n# detail par mort : match\tvictime\ttime_ms\tstatut\n")
	for _, p := range pairs {
		fmt.Fprintf(&b, "mort\t%s\t%d\t%d\t%s\n", short, p.victimXUID, p.timeMS,
			ctfFatalStatus(p, fire, tracks, augB, table))
	}
	return b.String()
}

// ctfHighlightEvents lit TOUS les événements du chunk highlight (morts ET frags), là où
// ScanFilmDeaths ne garde que les morts.
func ctfHighlightEvents(t *testing.T, filmDir string) []analysis.HighlightEvent {
	t.Helper()
	n := filmdec.CountFilmChunks(filmDir)
	raw, err := os.ReadFile(filepath.Join(filmDir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		t.Fatalf("chunk highlight : %v", err)
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		t.Fatalf("parse highlight : %v", err)
	}
	return evs
}

// ctfPairKills apparie chaque FRAG à la MORT du même instant — la reconstruction établie par le
// chantier voisin (93/93 sur le film de référence, 0 erreur). Rend aussi le nombre de frags
// qu'aucune mort n'accompagne, parce qu'un appariement sans son reste ne se juge pas.
func ctfPairKills(evs []analysis.HighlightEvent, off int64) ([]ctfKillPair, int, int) {
	var kills, deaths []analysis.HighlightEvent
	for _, e := range evs {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, e)
		case analysis.EventTypeDeath:
			deaths = append(deaths, e)
		}
	}
	used := make([]bool, len(deaths))
	var out []ctfKillPair
	unpaired := 0
	for _, k := range kills {
		bi, bd := -1, 2 // la reconstruction établie tient à 1 ms près
		for i, d := range deaths {
			if used[i] || d.XUID == k.XUID {
				continue // un joueur ne se tue pas lui-même dans ce chemin
			}
			if delta := k.TimeMS - d.TimeMS; delta < bd && delta > -bd {
				if delta < 0 {
					delta = -delta
				}
				bd, bi = delta, i
			}
		}
		if bi < 0 {
			unpaired++
			continue
		}
		used[bi] = true
		out = append(out, ctfKillPair{killerXUID: k.XUID, victimXUID: deaths[bi].XUID,
			filmUS: uint64((int64(k.TimeMS) + off) * 1000), timeMS: k.TimeMS})
	}
	return out, len(kills), unpaired
}

// ctfWriteFatalDelays mesure l'écart entre la mort et le tir du tueur le plus proche en amont.
// Publié AVANT tout critère : la fenêtre doit se lire dans la donnée.
func ctfWriteFatalDelays(b *strings.Builder, pairs []ctfKillPair, fire []filmdec.FireEvent,
	tbl PlayerIndexTable) {
	var d []int64
	for _, p := range pairs {
		pi, ok := tbl.ByXUID[p.killerXUID]
		if !ok {
			continue
		}
		best := int64(-1)
		for _, e := range fire {
			if e.FilmIndex != pi || e.TimestampUS > p.filmUS {
				continue
			}
			if delta := int64(p.filmUS-e.TimestampUS) / 1000; best < 0 || delta < best {
				best = delta
			}
		}
		if best >= 0 {
			d = append(d, best)
		}
	}
	fmt.Fprintf(b, "\n# ecart mort <-> tir du tueur le plus proche en amont (ms)\n")
	if len(d) == 0 {
		fmt.Fprintf(b, "aucun\n")
		return
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	fmt.Fprintf(b, "n\t%d\tp25\t%d\tmediane\t%d\tp75\t%d\tp90\t%d\n",
		len(d), d[len(d)/4], d[len(d)/2], d[len(d)*3/4], d[len(d)*9/10])
}

// ctfWriteFatalCoverage chiffre les maillons 3 et 4 pour un pont donné.
func ctfWriteFatalCoverage(b *strings.Builder, label string, pairs []ctfKillPair,
	fire []filmdec.FireEvent, tracks map[uint32]slotTrack, owner map[uint32]int, tbl PlayerIndexTable) {
	var noIndex, noShot, placed, unplaced int
	for _, p := range pairs {
		pi, ok := tbl.ByXUID[p.killerXUID]
		if !ok {
			noIndex++
			continue
		}
		found, ok2 := ctfLastShotBefore(fire, pi, p.filmUS)
		if !ok2 {
			noShot++
			continue
		}
		if _, r := slotFor(tracks, owner, pi, found.TimestampUS); r == reasonAttached {
			placed++
			continue
		}
		unplaced++
	}
	tot := len(pairs)
	fmt.Fprintf(b, "%s\tcouples\t%d\ttueur_hors_table\t%d\taucun_tir_dans_la_fenetre\t%d\ttir_non_place\t%d\ttir_PLACE\t%d\ttaux\t%.4f\n",
		label, tot, noIndex, noShot, unplaced, placed, ratio(placed, tot))
}

// ctfFatalStatus rend le statut d'une mort pour le pont donné, en un mot joignable.
func ctfFatalStatus(p ctfKillPair, fire []filmdec.FireEvent, tracks map[uint32]slotTrack,
	owner map[uint32]int, tbl PlayerIndexTable) string {
	pi, ok := tbl.ByXUID[p.killerXUID]
	if !ok {
		return "tueur_hors_table"
	}
	e, ok2 := ctfLastShotBefore(fire, pi, p.filmUS)
	if !ok2 {
		return "aucun_tir"
	}
	if _, r := slotFor(tracks, owner, pi, e.TimestampUS); r == reasonAttached {
		return "PLACE"
	}
	return "non_place"
}

// ctfLastShotBefore rend le tir du joueur pi le plus proche en amont de tUS, dans la fenêtre.
func ctfLastShotBefore(fire []filmdec.FireEvent, pi int, tUS uint64) (filmdec.FireEvent, bool) {
	var best filmdec.FireEvent
	found := false
	for _, e := range fire {
		if e.FilmIndex != pi || e.TimestampUS > tUS {
			continue
		}
		if tUS-e.TimestampUS > ctfFatalWindowMS*1000 {
			continue
		}
		if !found || e.TimestampUS > best.TimestampUS {
			best, found = e, true
		}
	}
	return best, found
}
