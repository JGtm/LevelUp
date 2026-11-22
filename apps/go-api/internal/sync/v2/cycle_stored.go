// Package v2 — cycle_stored.go : preuve de rangement des matchs listés pour un joueur, après la
// persistance du cycle.
//
// Un match listé n'est « inséré » que s'il est PROUVÉ rangé : présent dans l'ensemble connu du joueur
// RELU après la persistance (une erreur de lecture du match, un 404 ou un rejet à l'écriture ne
// passent pas pour un rangement). Une seule relecture par joueur sert au compte « matchs insérés »
// (populateOutcome) et au règlement du rattrapage (history_settle.go).
package v2

import (
	"context"
	"log/slog"
	"slices"

	syncpkg "levelup/go-api/internal/sync"
)

// playerStored est la preuve de rangement d'un joueur : son ensemble connu relu après la persistance.
// read=false : la relecture n'a pas eu lieu ou a échoué, rien n'est prouvé.
type playerStored struct {
	known map[string]bool
	read  bool
}

// storedProof donne la preuve de rangement de chaque joueur dont des matchs listés pouvaient être
// rangés (PlayerSlug -> preuve) ; un joueur absent n'avait rien à prouver.
type storedProof map[string]playerStored

// isNotFoundFetch dit si la récupération du match a échoué parce que l'API ne l'a pas (404/410).
func isNotFoundFetch(fetchErrs map[string]error, matchID string) bool {
	err, failed := fetchErrs[matchID]
	return failed && syncpkg.IsNotFound(err)
}

// notFoundListed rend, dans l'ordre, les matchs listés que l'API a rendus introuvables (404/410).
func notFoundListed(listed []string, fetchErrs map[string]error) []string {
	var out []string
	for _, id := range listed {
		if isNotFoundFetch(fetchErrs, id) {
			out = append(out, id)
		}
	}
	return out
}

// proveStored relit l'ensemble connu de chaque joueur qui a listé au moins un match que l'API n'a pas
// rendu introuvable. Un joueur sans match listé, ou dont la découverte a échoué, n'est pas relu.
func (o *CycleOrchestratorImpl) proveStored(
	ctx context.Context,
	players []PlayerProfile,
	disc DiscoveryResult,
	fetched FetchSharedResult,
) storedProof {
	proof := storedProof{}
	for _, p := range players {
		if disc.Errors[p.PlayerSlug] != nil || !hasFetchable(disc.PerPlayer[p.PlayerSlug], fetched.Errors) {
			continue
		}
		state, err := o.loader.LoadKnown(ctx, p)
		if err != nil {
			slog.WarnContext(ctx, "sync.v2: ensemble connu illisible après la persistance — rangement non prouvé",
				"event", "sync.v2.history.reload_failed", "player", p.PlayerSlug, "err", err)
			proof[p.PlayerSlug] = playerStored{}
			continue
		}
		proof[p.PlayerSlug] = playerStored{known: state.Known, read: true}
	}
	return proof
}

// hasFetchable dit si la liste contient au moins un match que l'API n'a pas rendu introuvable.
func hasFetchable(listed []string, fetchErrs map[string]error) bool {
	for _, id := range listed {
		if !isNotFoundFetch(fetchErrs, id) {
			return true
		}
	}
	return false
}

// insertedFor compte les matchs listés pour le joueur dont le rangement est prouvé. Une relecture
// impossible (journalisée par proveStored) retombe sur les matchs que la phase de récupération a
// rendus et que la persistance du cycle a acceptés : mieux qu'un compte nul, la persistance a réussi.
func (st playerStored) insertedFor(listed []string, fetched FetchSharedResult) int {
	n := 0
	for _, id := range listed {
		if _, ok := fetched.Matches[id]; !ok {
			continue
		}
		if st.read && !st.known[id] {
			continue
		}
		n++
	}
	return n
}

// outcomeProof est ce que le cycle sait du rangement des matchs listés au moment de remplir les
// compteurs d'un joueur : les matchs récupérés, ceux encore dans la file d'écriture (attente expirée,
// non comptés insérés : ils le seront quand la file aura fini) et la preuve de rangement par joueur.
type outcomeProof struct {
	fetched FetchSharedResult
	queued  map[string]bool
	stored  storedProof
}

// populateOutcome remplit les compteurs Phase 1-5 d'un joueur, sans écraser un status "failed" déjà
// positionné par markFailed. MatchesUnknown = les matchs listés pour lui ; MatchesInserted = ceux dont
// le rangement est PROUVÉ (cycle_stored.go) : un match introuvable (404/410), en échec de récupération
// ou rejeté à l'écriture n'y entre pas.
func (o *CycleOrchestratorImpl) populateOutcome(
	per map[string]PlayerOutcome,
	p PlayerProfile,
	disc DiscoveryResult,
	dedup DedupResult,
	pr outcomeProof,
) {
	out := per[p.PlayerSlug]
	out.Gamertag = p.Gamertag
	out.XUID = p.XUID
	if out.Status == "" {
		out.Status = "ok"
	}
	listed := disc.PerPlayer[p.PlayerSlug]
	out.MatchesUnknown = len(listed)
	// Un match n'est compté que pour un joueur qui y participe (proxy correct pour V2, où la
	// persistance se fait en lot).
	candidates := make([]string, 0, len(listed))
	for _, mID := range listed {
		if pr.queued[mID] || !slices.Contains(dedup.ParticipantsByMatch[mID], p.PlayerSlug) {
			continue
		}
		candidates = append(candidates, mID)
	}
	out.MatchesInserted += pr.stored[p.PlayerSlug].insertedFor(candidates, pr.fetched)
	per[p.PlayerSlug] = out
}
