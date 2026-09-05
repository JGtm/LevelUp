package main

// sweep.go — le BALAYAGE : parent qui planifie, enfant qui decode UN film sous plafond.
//
// LE PARENT NE DECODE RIEN : pour chaque film il re-execute son propre binaire avec
// `-child -match <id>`, attend sa mort et compte son issue (le patron de
// `cmd/zone-attribution/passe.go`). L'ENFANT arme la sentinelle de MESURE (2 Gio), balaye
// le statborg, imprime le TSV et rend un code du protocole filmproc.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// sweepMaxComp : le dernier index de composant balaye. L'archetype 6 declare 28
// emplacements `statborg-current-round-value-stat-component` (named.go) : 0..27.
const sweepMaxComp = 27

// runParent lance un enfant par film et imprime le recap de la passe.
func runParent(ctx context.Context, repoRoot string, a runArgs) error {
	runner, err := filmproc.NewRunner(repoRoot, os.Stdout)
	if err != nil {
		return err
	}
	fmt.Printf("mode parent/enfant : 1 processus par film, plafond %d Gio, priorite basse\n",
		filmproc.MeasureLimitGiB)
	recap := map[filmproc.Issue]int{}
	for i, id := range a.films {
		fmt.Printf("===== FILM %d/%d — %s =====\n", i+1, len(a.films), id)
		res := runner.Run(ctx, []string{"-child", "-match", id, "-cache", a.cache})
		recap[res.Issue]++
		fmt.Printf("ISSUE %s : %s code=%d duree=%s pic=%.2f Gio\n",
			id, res.Issue, res.Code, res.Dur.Round(1e6), float64(res.Peak)/(1<<30))
		if res.Err != nil {
			fmt.Printf("ISSUE %s : erreur de lancement : %v\n", id, res.Err)
		}
	}
	fmt.Printf("PASSE — %d film(s) :", len(a.films))
	for _, i := range []filmproc.Issue{filmproc.IssueOK, filmproc.IssueSkipped,
		filmproc.IssueFailed, filmproc.IssuePreparation, filmproc.IssueMemory,
		filmproc.IssueSuddenDeath} {
		if n := recap[i]; n > 0 {
			fmt.Printf(" %s=%d", i, n)
		}
	}
	fmt.Println()
	if n := recap[filmproc.IssueMemory] + recap[filmproc.IssueSuddenDeath]; n > 0 {
		fmt.Printf("ATTENTION : %d film(s) NON balayes — le corpus de la confrontation est "+
			"incomplet et cela se dit\n", n)
	}
	return nil
}

// runChild balaye UN film sous la sentinelle memoire et rend le code de protocole.
//
// LA SENTINELLE N'A DROIT DE CITE QUE DANS UN PROCESSUS SANS ECRITURE : c'est le cas, ce
// binaire n'ouvre aucune base et n'ecrit que sur stdout.
func runChild(cache, id string) int {
	if id == "" {
		fmt.Fprintln(os.Stderr, "enfant : -match obligatoire")
		return filmproc.CodePreparation
	}
	g := filmproc.Arm("statnames-sweep", filmproc.MeasureLimitGiB, func(peak uint64) {
		filmproc.EmitPeak(peak)
		fmt.Fprintf(os.Stderr, "enfant : plafond memoire depasse (%d octets) — film abandonne\n", peak)
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()
	if err := sweepFilm(cache, id); err != nil {
		fmt.Fprintf(os.Stderr, "enfant %s : %v\n", id, err)
		return filmproc.CodeFailed
	}
	return filmproc.CodeOK
}

// sweepFilm decode le statborg d'un film et imprime le TSV : identite des slots, puis la
// valeur finale de chaque emplacement par slot.
//
// UN SEUL CHARGEMENT DU FILM pour les deux lectures de ce balayage (les enregistrements
// d'entite et le fil des morts) : jamais un `*Film` d'un cote et une enveloppe `dir` de
// l'autre, ce serait deux decompressions du meme film (item 1.6 de PLAN_CUISSON_PERF).
func sweepFilm(cache, id string) error {
	film, ok, err := filmcache.LoadFilm(cache, id)
	if err != nil {
		return fmt.Errorf("chargement du film : %w", err)
	}
	if !ok {
		return fmt.Errorf("film absent du cache (%s)", cache)
	}
	recs, truncated := objectiveevents.StatRecordsCtx(context.Background(), film, id)
	deaths, err := replay.ScanDeaths(film)
	if err != nil {
		return fmt.Errorf("fil des morts : %w", err)
	}
	identity := objectiveevents.SlotIdentityByDeaths(recs, deathInstants(deaths))
	fmt.Printf("IDENTITE\t%s\tslots_nommes=%d\tenregistrements=%d\ttronque=%v\n",
		id, len(identity), len(recs), truncated)
	for _, slot := range sortedSlots(identity) {
		fmt.Printf("JOUEUR\t%s\t%d\t%s\n", id, slot, identity[slot])
	}
	sweepEmplacements(id, recs, identity)
	return nil
}

// sweepEmplacements imprime la valeur finale de chaque emplacement, par slot nomme.
//
// LES SLOTS NON NOMMES SORTENT AUSSI (xuid « - ») : la confrontation ne s'en sert pas,
// mais un balayage qui les tairait cacherait son propre denominateur.
func sweepEmplacements(id string, recs []objectiveevents.StatRecord, identity map[int]string) {
	for comp := 0; comp <= sweepMaxComp; comp++ {
		for _, sideB := range []bool{false, true} {
			c := objectiveevents.StatComponent{Comp: comp, SideB: sideB}
			series := objectiveevents.SeriesTotal(recs, c, false)
			for _, slot := range sortedSeriesSlots(series) {
				pts := series[slot]
				if len(pts) == 0 {
					continue
				}
				xuid, ok := identity[slot]
				if !ok {
					xuid = "-"
				}
				fmt.Printf("SWEEP\t%s\t%d\t%s\t%d\t%s\t%d\n",
					id, slot, xuid, comp, sideLabel(sideB), pts[len(pts)-1].Value)
			}
		}
	}
}

// deathInstants traduit le fil des morts dans la forme du pont d'identite — meme
// conversion que `deathInstantsOf` du rejeu (xuid decimal, horloge du match).
func deathInstants(deaths []replay.Death) []objectiveevents.DeathInstant {
	out := make([]objectiveevents.DeathInstant, 0, len(deaths))
	for _, d := range deaths {
		out = append(out, objectiveevents.DeathInstant{
			XUID: strconv.FormatUint(d.XUID, 10), TimeMS: int(d.TimeMS),
		})
	}
	return out
}

// sideLabel rend le cote d'une paire de valeurs.
func sideLabel(sideB bool) string {
	if sideB {
		return "B"
	}
	return "A"
}

// sortedSlots / sortedSeriesSlots : ordres totaux, sans eux la sortie changerait a chaque
// execution (parcours de map).
func sortedSlots(m map[int]string) []int {
	out := make([]int, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Ints(out)
	return out
}

func sortedSeriesSlots(m map[int][]objectiveevents.ScorePoint) []int {
	out := make([]int, 0, len(m))
	for s := range m {
		out = append(out, s)
	}
	sort.Ints(out)
	return out
}
