package skill

// skill_v2_shared_access.go — seam d'accès à la DB partagée pour le shadow LUSR
// v2 : sélection sous Read (skill_v2_watermark.go), persistance sous UNE rafale
// Write par joueur et par cycle.
//
// Régression prod 2026-07-03 (fix hotfix/lusr-shadow-ro) : le shadow recevait un
// *sql.DB unique classé « segment lecture » (en mode burst, un handle RO) et
// tentait un INSERT dessus → « Cannot execute statement of type INSERT ... which
// is attached in read-only mode ». Cause : erreur de CLASSIFICATION du refactor
// contention (le v2 shadow écrit player_skill_state_v2 CÔTÉ SHARED, ce n'est pas
// un pur lecteur comme le v1). Le seam ci-dessous force la sélection sous Read
// (RO) et la persistance sous une rafale Write (RW).

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"levelup/go-api/internal/platform/duckdb"
)

// SharedAccessor est le seam d'accès shared du shadow LUSR v2.
//
// Le sous-package skill NE PEUT PAS importer internal/sync (cycle sync→skill) :
// l'interface est déclarée ici et satisfaite STRUCTURELLEMENT par
// *sync.SharedAccess (mêmes signatures Read/Write). Contrat :
//   - Read  : handle de LECTURE RO (ne gate personne) — sélection, filigrane et
//     éligibilité (selectShadowWorkUnderRead).
//   - Write : rafale RW labellisée (persistance) ; REFUSE si un Read du même accès
//     est encore en vol (garde anti-deadlock) → toujours release le Read AVANT de
//     demander un Write.
type SharedAccessor interface {
	Read(ctx context.Context) (*sql.DB, func(), error)
	Write(ctx context.Context, step string) (*sql.DB, func(), error)
}

// errNilBurstHandle : l'accès a rendu une rafale Write sans handle (shared
// indisponible) — même dégradation qu'un échec d'acquisition.
var errNilBurstHandle = errors.New("rafale Write sans handle (shared indisponible)")

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

// runSingleWriterBurst prend UNE rafale Write("lusr") pour tout le travail du
// joueur sur ce cycle (lot perf L6, D6.2). Aucune transaction ne couvre plusieurs
// matchs — chaque état v2 est un INSERT autonome, la ligne canonique a sa propre
// transaction côté player DB — donc rien n'exige de relâcher l'écrivain entre deux
// matchs. Les anciens lots de 3 (un écrivain par triplet de candidats, le filigrane
// testé DANS la rafale) coûtaient une bascule RO→RW→RO par triplet, y compris
// quand rien n'était à écrire. Échec d'acquisition → tout le travail est reporté au
// prochain cycle (filigrane non avancé), comme avant.
func runSingleWriterBurst(ctx context.Context, shared SharedAccessor, base shadowRunContext, pending []shadowMatch, s *shadowRunStats) {
	burstDB, release, err := shared.Write(ctx, "lusr")
	if err != nil {
		slog.ErrorContext(ctx, "LUSR v2 shadow: rafale Write shared indisponible — travail reporté au prochain cycle",
			"xuid", base.xuid, "new", len(pending), "err", err)
		return
	}
	defer release()
	if burstDB == nil {
		slog.ErrorContext(ctx, "LUSR v2 shadow: rafale Write shared indisponible — travail reporté au prochain cycle",
			"xuid", base.xuid, "new", len(pending), "err", errNilBurstHandle)
		return
	}
	processShadowBurst(ctx, base, burstDB, pending, s)
}

// processShadowBurst traite tout le travail du cycle SOUS le handle de l'unique
// rafale Write (RW). Les repos SkillV2/SquadOffset et TOUTES les lectures
// per-match (filigrane relu, états, rosters, quit timeline) passent par ce handle :
// un writer lit aussi, et la persistance UpsertState va donc sur un handle
// inscriptible (fix read-only 2026-07-03). `base` est copié par valeur, ses caches
// (maps) sont des références partagées.
//
// heldGroups : groupes dont un match a échoué son écriture canonical dans cette
// rafale. Les matchs plus RÉCENTS de ces groupes sont sautés pour ne pas avancer le
// filigrane de groupe par-dessus un match non écrit (anti-gap, fix 2026-06-07) —
// pending est en ordre chronologique ASC.
func processShadowBurst(ctx context.Context, base shadowRunContext, burstDB *sql.DB,
	pending []shadowMatch, s *shadowRunStats) {
	c := base
	c.sharedDB = burstDB
	c.repo = duckdb.NewSkillV2Repo(burstDB)
	// squadRepo seulement si le flag est actif (sinon interface nil → offsets nuls,
	// comportement strictement inchangé ; un typed-nil casserait le garde de
	// computeTeamSquadOffsets).
	if c.squadEnabled {
		c.squadRepo = duckdb.NewSquadOffsetRepo(burstDB)
	}
	heldGroups := make(map[string]bool)
	for _, m := range pending {
		processOneShadowMatch(ctx, c, m, s, heldGroups)
	}
}
