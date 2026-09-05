package main

import (
	"context"
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/platform/duckdb"
)

// processMatch décode + résume (et persiste si cfg.write) un match.
func processMatch(ctx context.Context, c *conn, cfg runConfig, m matchRef) error {
	reg, ok, err := loadRegistry(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("absent de match_registry")
	}
	// LE FILM EST CHARGE UNE FOIS (chunks decompresses, paquets decoupes) : c'est ce que prend
	// desormais `objectiveevents` (item 1.5 de PLAN_CUISSON_PERF).
	film, ok, err := filmcache.LoadFilm(cfg.cacheDir, m.short)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("manifest cache absent (%s)", m.short)
	}
	roster, err := loadRoster(ctx, c.sqlDB, m.full)
	if err != nil {
		return err
	}

	events := objectiveevents.Extract(m.full, reg.variant, film, objectiveevents.MapRoster(roster))
	printSummary(m, reg, events)

	if cfg.write {
		if err := writeEvents(ctx, c, m.full, events); err != nil {
			return err
		}
		fmt.Printf("  [write] %d event(s) persistés sur match_objective_events\n", len(events))
	}
	fmt.Println()
	return nil
}

// writeEvents persiste les events via ObjectiveEventsRepo (PlayerDB minimal :
// seul Shared est requis, SharedReadDB() retombe sur LegacySharedReader).
func writeEvents(ctx context.Context, c *conn, matchID string, events []domain.ObjectiveEvent) error {
	if c.rwDB == nil {
		return fmt.Errorf("connexion non-RW (write impossible)")
	}
	pdb := &duckdb.PlayerDB{Shared: c.rwDB}
	repo := duckdb.NewObjectiveEventsRepo(pdb)
	return repo.WriteMatch(ctx, matchID, events)
}

// printSummary affiche le résumé d'un match : mode décodé, events par
// objective_type/event_type, split par équipe, timeline, + compare CTF vs DB.
func printSummary(m matchRef, reg *registryRow, events []domain.ObjectiveEvent) {
	fmt.Printf("[%s] %s — variant=%q\n", m.short, m.full, reg.variant)
	if len(events) == 0 {
		fmt.Println("  Aucun event objectif décodé (mode non-objectif, footer absent, ou film sans capture).")
		return
	}

	for _, line := range eventTypeBreakdown(events) {
		fmt.Printf("  %s\n", line)
	}
	t0, t1, unknown := teamSplit(events)
	fmt.Printf("  Split équipe : team0=%d team1=%d inconnu=%d\n", t0, t1, unknown)

	printTimeline(events)
	// CTF si au moins un event flag/capture décodé : permet de comparer au score
	// DB sans dépendre du classifieur interne (non exporté) du package.
	if countCaptures(events) > 0 {
		printCTFCompare(reg, t0, t1, events)
	}
}

// printTimeline affiche chaque event ordonné (seq, t en s, type, équipe, scorer).
func printTimeline(events []domain.ObjectiveEvent) {
	fmt.Println("  Timeline :")
	for _, e := range events {
		fmt.Printf("    seq=%-3d t=%-8s %s/%s team=%s %s conf=%s%s\n",
			e.Seq, fmtTimeMS(e.TimeMS), e.ObjectiveType, e.EventType,
			fmtTeam(e.TeamID), fmtScorer(e.Players), e.Confidence, fmtSource(e.Source))
	}
}

// printCTFCompare confronte le COUNT/split de captures décodées au score DB.
func printCTFCompare(reg *registryRow, t0, t1 int, events []domain.ObjectiveEvent) {
	captures := countCaptures(events)
	fmt.Println("  --- Comparaison CTF vs DB ---")
	if !reg.hasScores {
		fmt.Printf("    Captures décodées : %d (split %d-%d) — DB sans score final (NULL)\n",
			captures, t0, t1)
		return
	}
	dbTotal := reg.team0Score + reg.team1Score
	countOK := captures == dbTotal
	splitOK := t0 == reg.team0Score && t1 == reg.team1Score
	fmt.Printf("    Captures décodées : %d (split %d-%d)\n", captures, t0, t1)
	fmt.Printf("    Score DB final    : %d (split %d-%d)\n", dbTotal, reg.team0Score, reg.team1Score)
	fmt.Printf("    COUNT match=%s · SPLIT match=%s\n", okStr(countOK), okStr(splitOK))
}

// eventTypeBreakdown agrège les events par (objective_type, event_type),
// ordonné, en lignes "type/event : N".
func eventTypeBreakdown(events []domain.ObjectiveEvent) []string {
	counts := make(map[string]int)
	for _, e := range events {
		counts[e.ObjectiveType+"/"+e.EventType]++
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%-22s : %d", k, counts[k]))
	}
	return out
}

// teamSplit compte les events par team_id (0/1/inconnu).
func teamSplit(events []domain.ObjectiveEvent) (t0, t1, unknown int) {
	for _, e := range events {
		switch {
		case e.TeamID == nil:
			unknown++
		case *e.TeamID == 0:
			t0++
		case *e.TeamID == 1:
			t1++
		}
	}
	return
}

// countCaptures compte les events CTF capture.
func countCaptures(events []domain.ObjectiveEvent) int {
	n := 0
	for _, e := range events {
		if e.EventType == objectiveevents.EventTypeCapture {
			n++
		}
	}
	return n
}
