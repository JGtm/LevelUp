package skill

// skill_v2_shared_access.go — seam d'accès à la DB partagée pour le shadow LUSR
// v2, et découpage per-match en bursts Write courts.
//
// Régression prod 2026-07-03 (fix hotfix/lusr-shadow-ro) : le shadow recevait un
// *sql.DB unique classé « segment lecture » (en mode burst, un handle RO) et
// tentait un INSERT dessus → « Cannot execute statement of type INSERT ... which
// is attached in read-only mode ». Cause : erreur de CLASSIFICATION du refactor
// contention (le v2 shadow écrit player_skill_state_v2 CÔTÉ SHARED, ce n'est pas
// un pur lecteur comme le v1). Le seam ci-dessous force la sélection sous Read
// (RO) et la persistance sous des bursts Write (RW courts).

import (
	"context"
	"database/sql"
	"fmt"

	"levelup/go-api/internal/platform/duckdb"
)

// SharedAccessor est le seam d'accès shared du shadow LUSR v2.
//
// Le sous-package skill NE PEUT PAS importer internal/sync (cycle sync→skill) :
// l'interface est déclarée ici et satisfaite STRUCTURELLEMENT par
// *sync.SharedAccess (mêmes signatures Read/Write). Contrat :
//   - Read  : handle de LECTURE RO (ne gate personne) — sélection des matchs.
//   - Write : burst RW COURT labellisé (persistance per-match) ; REFUSE si un
//     Read du même accès est encore en vol (garde anti-deadlock) → toujours
//     release le Read AVANT de demander un Write.
type SharedAccessor interface {
	Read(ctx context.Context) (*sql.DB, func(), error)
	Write(ctx context.Context, step string) (*sql.DB, func(), error)
}

// postsyncLUSRBurstChunk borne la fenêtre RW d'un burst Write("lusr") du shadow :
// on relâche/ré-acquiert le writer tous les N matchs, les lecteurs passent entre
// les chunks. Même esprit/valeur que sync.postsyncEventsBurstChunk (=3,
// internal/sync/engine.go) — dupliqué ici car skill ne peut pas importer sync.
const postsyncLUSRBurstChunk = 3

// pinnedSharedAccess : SharedAccessor sur un handle DÉJÀ TENU par le caller
// (CLI/backfill qui possèdent le writer, et tests). Read/Write retournent ce
// handle, release no-op — parité stricte avec l'ancien passage direct d'un
// *sql.DB. Équivalent skill-local de sync.NewPinnedSharedAccess (cycle interdit).
type pinnedSharedAccess struct{ db *sql.DB }

func newPinnedSharedAccessor(db *sql.DB) SharedAccessor { return pinnedSharedAccess{db: db} }

func (p pinnedSharedAccess) Read(context.Context) (*sql.DB, func(), error) {
	return p.db, func() {}, nil
}

func (p pinnedSharedAccess) Write(context.Context, string) (*sql.DB, func(), error) {
	return p.db, func() {}, nil
}

// loadShadowMatchesUnderRead sélectionne les matchs LUSR-éligibles sous un segment
// de LECTURE shared (pur SELECT), et RELÂCHE le Read avant de rendre la main — un
// Read encore en vol ferait échouer le burst Write suivant (garde anti-deadlock).
func loadShadowMatchesUnderRead(ctx context.Context, shared SharedAccessor, xuid string) ([]shadowMatch, error) {
	readDB, release, err := shared.Read(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	if readDB == nil {
		return nil, fmt.Errorf("shared read handle nil")
	}
	return loadShadowMatches(ctx, readDB, xuid)
}

// processShadowChunk traite un chunk de matchs SOUS le handle d'un burst Write
// (RW). Les repos SkillV2/SquadOffset et TOUTES les lectures per-match (états,
// rosters, quit timeline) passent par ce handle : un writer lit aussi, et la
// persistance UpsertState va donc sur un handle inscriptible (fix read-only
// 2026-07-03). `base` est copié par valeur mais ses caches (maps) et heldGroups
// sont des références partagées → l'état de run survit d'un chunk au suivant.
func processShadowChunk(ctx context.Context, base shadowRunContext, burstDB *sql.DB, squadEnabled bool,
	chunk []shadowMatch, s *shadowRunStats, heldGroups map[string]bool, withGameplayDur *int) {
	c := base
	c.sharedDB = burstDB
	c.repo = duckdb.NewSkillV2Repo(burstDB)
	// squadRepo seulement si le flag est actif (sinon interface nil → offsets nuls,
	// comportement strictement inchangé ; un typed-nil casserait le garde de
	// computeTeamSquadOffsets).
	if squadEnabled {
		c.squadRepo = duckdb.NewSquadOffsetRepo(burstDB)
	}
	for _, m := range chunk {
		if m.gameplayDurMs > 0 {
			*withGameplayDur++
		}
		processOneShadowMatch(ctx, c, m, s, heldGroups)
	}
}
