package skill

// skill_v2_shared_access.go — seam d'accès à la DB partagée pour le shadow LUSR
// v2 : sélection sous Read (skill_v2_watermark.go), persistance sous des rafales
// Write BORNÉES (au plus 50 matchs ou 2 s chacune), aucune en régime stationnaire.
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
	"time"

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

// lusrBurstLimits borne une rafale d'écrivain du shadow LUSR v2 (lot perf L9-go,
// 2026-09-23, revue adversariale B, P1). Une rafale unique pour tout le rattrapage d'un
// joueur (lot L6) tenait l'écrivain partagé d'un seul tenant : 2,5 s pour 300 candidats
// sur la base de test, bien davantage au premier cycle après un onboarding ou une longue
// absence — les lecteurs HTTP attendaient tout ce temps.
type lusrBurstLimits struct {
	maxMatches int              // au plus K matchs traités par rafale
	maxHold    time.Duration    // rafale rendue après le match au cours duquel ce seuil est atteint
	now        func() time.Time // horloge (time.Now hors tests)
}

// defaultLUSRBurstLimits : K = 50 matchs, ou 2 s de détention — le seuil du chien de garde
// du provider partagé (sharedprovider.defaultRWHoldWatchdog : au-delà, un writer encore
// tenu déclenche un WARN), dont une rafale ne doit donc pas faire son régime normal.
// Surchargé par les tests du paquet seulement (aucun test parallèle).
var defaultLUSRBurstLimits = lusrBurstLimits{maxMatches: 50, maxHold: 2 * time.Second, now: time.Now}

// burstQueue : la file d'un cycle (ordre chronologique ASC) et ce qui traverse ses rafales.
//
// heldGroups : groupes dont un match a échoué son écriture canonical dans ce CYCLE. Les
// matchs plus RÉCENTS de ces groupes sont sautés pour ne pas avancer le filigrane de
// groupe par-dessus un match non écrit (anti-gap, fix 2026-06-07) — y compris quand ils
// tombent dans une rafale suivante.
type burstQueue struct {
	pending    []shadowMatch
	heldGroups map[string]bool
}

// runWriterBursts persiste la file du cycle sous des rafales d'écrivain BORNÉES. Le
// régime stationnaire n'en prend aucune (décidé avant, sur le lecteur : lot L6, D6.1).
// Quand il y a du nouveau, l'écrivain est rendu tous les K matchs ou dès que la rafale a
// tenu maxHold, puis repris pour le reste de la file, dans le même cycle. Aucune
// transaction ne couvre plusieurs matchs — chaque état v2 est un INSERT autonome, la
// ligne canonique a sa propre transaction côté player DB — donc la découpe ne change pas
// les écritures (TestLUSRV2Shadow_ParityWithLegacyOrchestration). Une ligne INFO par
// rafale ; rend le nombre de rafales prises. Écrivain indisponible → le reste de la file
// est reporté au prochain cycle (filigrane non avancé), comme avant.
func runWriterBursts(ctx context.Context, shared SharedAccessor, base shadowRunContext, w shadowWork, s *shadowRunStats) int {
	q := &burstQueue{pending: w.pending, heldGroups: make(map[string]bool)}
	bursts := 0
	for len(q.pending) > 0 {
		if err := ctx.Err(); bursts > 0 && err != nil {
			slog.InfoContext(ctx, "lusr_v2: cycle interrompu entre deux rafales — reste reporté au prochain cycle",
				"xuid", base.xuid, "bursts", bursts, "remaining", len(q.pending), "err", err)
			return bursts
		}
		matches, held, ok := runOneWriterBurst(ctx, shared, base, q, s)
		if !ok {
			return bursts
		}
		bursts++
		slog.InfoContext(ctx, "lusr_v2: rafale bornée",
			"xuid", base.xuid, "candidates", w.candidates, "new", len(w.pending), "bursts", bursts,
			"matches", matches, "held_ms", held.Milliseconds(), "remaining", len(q.pending))
	}
	return bursts
}

// runOneWriterBurst prend UNE rafale Write("lusr") et y traite la tête de la file dans les
// bornes (processShadowBurst). ok=false : écrivain indisponible.
func runOneWriterBurst(ctx context.Context, shared SharedAccessor, base shadowRunContext, q *burstQueue, s *shadowRunStats) (int, time.Duration, bool) {
	burstDB, release, err := shared.Write(ctx, "lusr")
	if err != nil {
		slog.ErrorContext(ctx, "LUSR v2 shadow: rafale Write shared indisponible — travail reporté au prochain cycle",
			"xuid", base.xuid, "new", len(q.pending), "err", err)
		return 0, 0, false
	}
	defer release()
	if burstDB == nil {
		slog.ErrorContext(ctx, "LUSR v2 shadow: rafale Write shared indisponible — travail reporté au prochain cycle",
			"xuid", base.xuid, "new", len(q.pending), "err", errNilBurstHandle)
		return 0, 0, false
	}
	matches, held := processShadowBurst(ctx, base, burstDB, q, s)
	return matches, held, true
}

// processShadowBurst traite la tête de la file SOUS le handle de la rafale Write (RW),
// jusqu'à base.burst.maxMatches matchs ou base.burst.maxHold de détention, et la retire
// de la file. Les repos SkillV2/SquadOffset et TOUTES les lectures per-match (filigrane
// relu, états, rosters, quit timeline) passent par ce handle : un writer lit aussi, et la
// persistance UpsertState va donc sur un handle inscriptible (fix read-only 2026-07-03).
// `base` est copié par valeur, ses caches (maps) sont des références partagées. Rend le
// nombre de matchs traités et la durée de détention mesurée au dernier.
func processShadowBurst(ctx context.Context, base shadowRunContext, burstDB *sql.DB,
	q *burstQueue, s *shadowRunStats) (int, time.Duration) {
	c := base
	c.sharedDB = burstDB
	c.repo = duckdb.NewSkillV2Repo(burstDB)
	// squadRepo seulement si le flag est actif (sinon interface nil → offsets nuls,
	// comportement strictement inchangé ; un typed-nil casserait le garde de
	// computeTeamSquadOffsets).
	if c.squadEnabled {
		c.squadRepo = duckdb.NewSquadOffsetRepo(burstDB)
	}
	start := c.burst.now()
	n, held := 0, time.Duration(0)
	for n < len(q.pending) {
		processOneShadowMatch(ctx, c, q.pending[n], s, q.heldGroups)
		n++
		held = c.burst.now().Sub(start)
		if n >= c.burst.maxMatches || held >= c.burst.maxHold {
			break
		}
	}
	q.pending = q.pending[n:]
	return n, held
}
