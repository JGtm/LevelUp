// cmd/backfill-team-rounds — renseigne `match_registry.team_0_rounds_won`,
// `team_1_rounds_won` et `rounds_total` (Halo Infinite) sur les matchs déjà en base.
//
// # CE QU'IL REMPLIT, ET POURQUOI UN RE-SYNC NE SUFFIT PAS
//
// Les trois colonnes sont nées le 2026-08-29 : toutes les lignes antérieures les portent à
// NULL. Or `persistMatchRegistry` (`internal/persist/shared_persister.go`) est un INSERT NU,
// sans `ON CONFLICT` — un match déjà présent n'est JAMAIS réécrit, quel que soit le nombre
// de resynchronisations. Aucune requête SQL ne peut non plus les dériver : la donnée n'existe
// que dans le payload de l'API (`Teams[].Stats.CoreStats.RoundsWon/RoundsLost/RoundsTied`).
// D'où cet outil, qui re-télécharge et écrit ligne à ligne.
//
// # POURQUOI ÇA COMPTE
//
// Sur un mode qui se décide aux manches, le score d'équipe est un CUMUL DE POINTS qui ne dit
// pas le résultat. Mesure du 2026-08-29 sur les 1 942 matchs à score du corpus
// (`.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`) : 4 matchs Oddball donnent la victoire à
// l'équipe qui affiche le MOINS de points. Sans ces colonnes, l'app présente ces victoires
// comme des défaites au score.
//
// # DEUX PHASES, ET LA RAISON EST LE VERROU
//
// Phase A — SANS aucun droit d'écriture : liste de travail, téléchargement des payloads,
// décisions. C'est la phase longue (un appel API par match, ~10 min pour 1 942 à `--rps 4`).
//
// Phase B — n'existe qu'avec `--apply`, et elle est COURTE : ouverture en écriture, puis par
// ligne une re-lecture de la valeur courante suivie de l'écriture, seulement si elle diffère
// toujours.
//
// Découper ainsi n'est pas cosmétique : prendre le droit d'écriture avant la boucle de
// téléchargement tiendrait la base fermée pendant toute la durée des appels réseau, alors
// qu'aucune écriture n'a encore lieu.
//
// # CE QU'IL NE FAIT PAS
//
// Il n'écrit jamais un NULL, jamais une valeur négative ou hors bornes du SMALLINT, jamais
// un total de manches inférieur aux manches gagnées. Un match dont l'API ne publie pas les
// deux camps (FFA, payload partiel) est SAUTÉ et journalisé : ses colonnes restent NULL, et
// l'affichage retombe sur les points — la dégradation prévue.
//
// # QUI GARANTIT L'UNICITÉ DU WRITER
//
// Le VERROU FICHIER de DuckDB : `OpenReadWrite` ping la base à l'ouverture, donc `--apply`
// échoue immédiatement (« Could not set lock ») si le serveur LevelUp tient déjà la base. Le
// serveur doit être arrêté (ADR 0013). Le lease `dblease` pris en phase B est un mutex
// INTRA-process : discipline et métrique, jamais la garantie d'exclusion.
//
// # USAGE
//
//	# 1. répétition à blanc (aucune écriture, aucun droit d'écriture demandé)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> go run ./cmd/backfill-team-rounds --gamertag JGtm
//
//	# 2. application (SERVEUR ARRÊTÉ)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> go run ./cmd/backfill-team-rounds --gamertag JGtm --apply
//
//	# un seul match, ou un échantillon
//	go run ./cmd/backfill-team-rounds --match 293a763e-... --apply
//	go run ./cmd/backfill-team-rounds --limit 50
//
// La liste de travail est lue DANS LA BASE (`rounds_total IS NULL`) et RESTREINTE PAR DÉFAUT
// aux variantes déclarées dans `regulation.toml [rounds_decide]` — celles, et seulement
// celles, dont l'affichage change. Rattraper le corpus entier coûterait ~1 900 appels d'API
// pour une poignée de lignes utiles ; les matchs FUTURS, eux, sont renseignés à la sync pour
// toutes les variantes. `--all` force le corpus entier. L'outil est reprenable après
// interruption, et le jour où une variante est ajoutée à la table, un second passage rattrape
// son historique. Auth : `MultiUserTokenStore` (ADR 0023), aucune re-capture.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// writerLeaseTimeout borne l'attente du lease intra-process. Court : en CLI mono-process il
// est toujours libre, un dépassement signalerait un bug d'appel, pas une contention.
const writerLeaseTimeout = 30 * time.Second

// tally compte les issues. Publié tel quel en fin de course.
//
// `planned` et `written` sont DEUX compteurs distincts : la phase A décide (elle incrémente
// `planned`), la phase B écrit (elle incrémente `written`). En répétition à blanc la phase B
// n'existe pas — les confondre ferait afficher « ecrits=0 » alors que N écritures sont prêtes.
type tally struct {
	read, identical, planned, written, skipped, failed int
}

// plannedWrite est une écriture décidée en phase A, en attente de la phase B.
type plannedWrite struct {
	matchID  string
	decision Decision
}

func main() {
	opts := parseFlags()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fatal("config: %v", err)
	}
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)

	client, err := newAPIClient(ctx, cfg, pr, opts.gamertag, opts.rps)
	if err != nil {
		fatal("%v", err)
	}

	variants, err := declaredVariants(pr)
	if err != nil {
		fatal("%v", err)
	}

	// --- Phase A : aucun droit d'écriture n'est demandé ici. ---
	plans, t, err := planPhase(ctx, client, sharedPath, variants, opts)
	if err != nil {
		fatal("%v", err)
	}

	// --- Phase B : courte, et seulement si on applique. ---
	if opts.apply && len(plans) > 0 {
		if err := applyPhase(ctx, sharedPath, plans, &t); err != nil {
			fatal("%v", err)
		}
	}

	reportTally(ctx, opts.apply, t)
	if t.failed > 0 {
		os.Exit(1)
	}
}

// options porte les drapeaux, pour que main reste lisible et sous les seuils.
type options struct {
	gamertag, single string
	apply, all       bool
	rps, limit       int
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.gamertag, "gamertag", "JGtm", "Gamertag dont les tokens servent à l'auth API (db_profiles.json)")
	flag.StringVar(&o.single, "match", "", "Traiter CE seul match_id (ignore la liste de travail)")
	flag.BoolVar(&o.apply, "apply", false, "ÉCRIRE réellement. Sans ce drapeau : répétition à blanc, aucune écriture")
	flag.IntVar(&o.rps, "rps", 4, "Requêtes API par seconde")
	flag.IntVar(&o.limit, "limit", 0, "Ne traiter que les N premiers matchs de la liste (0 = tous)")
	flag.BoolVar(&o.all, "all", false, "Renseigner TOUT le corpus au lieu des seules variantes declarees dans [rounds_decide] (~1 900 appels d'API pour une poignee de lignes utiles)")
	flag.Parse()
	return o
}

// reportTally publie le résumé. `planifiees` et `ecrits` sont affichés dans LES DEUX modes :
// en répétition à blanc on lit `planifiees=N ecrits=0`, après application `planifiees=N
// ecrits=N`. Un écart en mode `--apply` se voit donc d'un coup d'oeil.
func reportTally(ctx context.Context, apply bool, t tally) {
	slog.InfoContext(ctx, "backfill_team_rounds: terminé",
		"apply", apply, "lus", t.read, "identiques", t.identical,
		"planifiees", t.planned, "ecrits", t.written,
		"skippes", t.skipped, "echecs", t.failed)
	if !apply && t.planned > 0 {
		slog.WarnContext(ctx, "backfill_team_rounds: répétition à blanc — RIEN n'a été écrit ; relancer avec --apply (serveur arrêté) pour appliquer",
			"a_ecrire", t.planned)
	}
}

// planPhase ouvre la base en LECTURE, décide pour chaque match, et rend le plan.
//
// `OpenReadForQuery` dégénère en ouverture read-only depuis un process séparé : la lecture
// cohabite avec un serveur qui tient la base EN LECTURE, mais avorte proprement (« Could not
// set lock ») s'il est en train d'y écrire. Limite acceptée : la répétition à blanc n'a rien
// à réparer, on la relance.
func planPhase(ctx context.Context, f matchFetcher, path string, variants []string, opts options) ([]plannedWrite, tally, error) {
	var t tally
	db, closeDB, err := duckdbpkg.OpenReadForQuery(path)
	if err != nil {
		return nil, t, fmt.Errorf("ouverture lecture de %s : %w", path, err)
	}
	defer closeDB()

	ids, err := workList(ctx, db, variants, opts)
	if err != nil {
		return nil, t, err
	}
	slog.InfoContext(ctx, "backfill_team_rounds: démarrage",
		"matchs", len(ids), "apply", opts.apply, "corpus_entier", opts.all,
		"variantes_declarees", len(variants), "shared_db", path)

	reg := sqlRegistry{q: db}
	var plans []plannedWrite
	for _, id := range ids {
		if p, ok := planMatch(ctx, f, reg, id, &t); ok {
			plans = append(plans, p)
		}
	}
	return plans, t, nil
}

// workList rend les matchs à traiter : le match unique s'il est fourni, sinon ceux dont les
// manches sont encore inconnues — restreints par défaut aux variantes déclarées.
func workList(ctx context.Context, db *sql.DB, variants []string, opts options) ([]string, error) {
	if opts.single != "" {
		return []string{opts.single}, nil
	}
	if !opts.all && len(variants) == 0 {
		return nil, fmt.Errorf("aucune variante déclarée dans [rounds_decide] — rien à rattraper (utiliser --all pour renseigner tout le corpus)")
	}
	filter := variants
	if opts.all {
		filter = nil
	}
	ids, err := pendingMatchIDs(ctx, db, filter, opts.limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		slog.InfoContext(ctx, "backfill_team_rounds: aucun match sans manches — rien à faire")
	}
	return ids, nil
}

// declaredVariants lit les variantes dont le résultat se lit en manches, depuis le
// regulation.toml du titre. C'est la MÊME table que celle qui commande l'affichage : le
// backfill ne peut donc pas rattraper une population différente de celle qui en a besoin.
func declaredVariants(pr *titlePkg.PathResolver) ([]string, error) {
	path := filepath.Join(pr.TitleMappingsDir(titlePkg.DefaultSlug), "regulation.toml")
	set, err := mappings.LoadRegulationFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture de %s : %w", path, err)
	}
	return set.RoundsDecideVariants(), nil
}

// planMatch traite UN match en phase A : lecture, fetch, décision. ok=true si une écriture
// est à faire.
func planMatch(ctx context.Context, f matchFetcher, r registryReader, id string, t *tally) (plannedWrite, bool) {
	cur, found, err := r.ReadRounds(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_rounds: lecture match_registry échouée", "match_id", id, "err", err)
		t.failed++
		return plannedWrite{}, false
	}
	if !found {
		slog.WarnContext(ctx, "backfill_team_rounds: match absent de match_registry — sauté", "match_id", id)
		t.skipped++
		return plannedWrite{}, false
	}
	matchJSON, err := f.GetMatchStats(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_rounds: GetMatchStats échoué", "match_id", id, "err", err)
		t.failed++
		return plannedWrite{}, false
	}
	t.read++

	dec := Decide(matchJSON, cur)
	switch dec.Verdict {
	case VerdictIdentical:
		t.identical++
		return plannedWrite{}, false
	case VerdictSkipNoTeams, VerdictSkipImplausible:
		t.skipped++
		slog.WarnContext(ctx, "backfill_team_rounds: sauté",
			"match_id", id, "verdict", string(dec.Verdict), "cause", dec.Reason)
		return plannedWrite{}, false
	}
	t.planned++
	slog.DebugContext(ctx, "backfill_team_rounds: écriture retenue",
		"match_id", id, "avant", formatRounds(dec.Old),
		"apres", fmt.Sprintf("%d-%d sur %d", dec.NewTeam0Won, dec.NewTeam1Won, dec.NewTotal))
	return plannedWrite{matchID: id, decision: dec}, true
}

// applyPhase ouvre la base en ÉCRITURE et applique le plan. Fenêtre volontairement courte :
// plus aucun appel réseau ici.
func applyPhase(ctx context.Context, path string, plans []plannedWrite, t *tally) error {
	handle, err := duckdbpkg.OpenReadWrite(path)
	if err != nil {
		return fmt.Errorf("ouverture écriture de %s (serveur LevelUp arrêté ?) : %w", path, err)
	}
	defer func() { _ = handle.Close() }()
	db := handle.SQLDb()

	// Mutex intra-process : discipline et métrique, PAS la garantie d'exclusion (celle-ci
	// vient du verrou fichier DuckDB, déjà éprouvé par l'ouverture ci-dessus).
	w, err := dblease.AcquireWriter(db, path, dblease.KindSharedMatches, writerLeaseTimeout)
	if err != nil {
		return fmt.Errorf("lease writer sur %s : %w", path, err)
	}
	defer w.Release()

	reg := sqlRegistry{q: db, ex: w}
	for _, p := range plans {
		applyOne(ctx, reg, reg, p, t)
	}
	return nil
}

// applyOne écrit UNE ligne, après re-lecture.
//
// La re-lecture n'est pas une précaution de principe : entre la phase A et la phase B il
// s'est écoulé le temps de tous les appels API, et rien ne garantit que la ligne n'a pas
// bougé. Si elle porte déjà la valeur visée, on ne réécrit pas — l'outil reste idempotent.
func applyOne(ctx context.Context, r registryReader, w registryWriter, p plannedWrite, t *tally) {
	cur, found, err := r.ReadRounds(ctx, p.matchID)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_rounds: re-lecture avant écriture échouée", "match_id", p.matchID, "err", err)
		t.failed++
		return
	}
	if !found {
		slog.WarnContext(ctx, "backfill_team_rounds: match disparu du registre entre les deux phases — sauté", "match_id", p.matchID)
		t.skipped++
		return
	}
	if cur.Team0Won != nil && cur.Team1Won != nil && cur.Total != nil &&
		*cur.Team0Won == p.decision.NewTeam0Won && *cur.Team1Won == p.decision.NewTeam1Won &&
		*cur.Total == p.decision.NewTotal {
		t.identical++
		return
	}
	if err := w.WriteRounds(ctx, p.matchID, p.decision.NewTeam0Won, p.decision.NewTeam1Won, p.decision.NewTotal); err != nil {
		slog.ErrorContext(ctx, "backfill_team_rounds: écriture échouée", "match_id", p.matchID, "err", err)
		t.failed++
		return
	}
	t.written++
}

// newAPIClient résout le xuid du porteur puis rafraîchit ses tokens (ADR 0023).
// AUCUNE re-capture : un refresh token valide se rafraîchit ; s'il est mort, on s'arrête en
// le disant plutôt que d'en fabriquer un nouveau.
func newAPIClient(ctx context.Context, cfg *config.AppConfig, pr *titlePkg.PathResolver,
	gamertag string, rps int) (*go_sync.HaloAPIClient, error) {
	xuid, err := resolveXUID(cfg, gamertag)
	if err != nil {
		return nil, fmt.Errorf("résolution du xuid de %s : %w", gamertag, err)
	}
	store := auth_platform.NewMultiUserTokenStore(pr.WatcherTokensDir())
	exch, err := auth_platform.RefreshHaloTokensViaStoreFirst(
		ctx, store, auth_platform.NewSISUProvider(), xuid, gamertag)
	if err != nil || exch == nil || exch.Tokens == nil {
		return nil, fmt.Errorf("auth %s (xuid %s) : %w — diagnostiquer (RT tourné perdu, mauvais xuid), NE PAS re-capturer", gamertag, xuid, err)
	}
	return go_sync.NewHaloAPIClient(exch.Tokens.SpartanToken, exch.Tokens.ClearanceToken, rps), nil
}

// resolveXUID lit le xuid d'un gamertag dans db_profiles.json.
func resolveXUID(cfg *config.AppConfig, gamertag string) (string, error) {
	players, err := cfg.LoadPlayers(titlePkg.DefaultSlug)
	if err != nil {
		return "", err
	}
	for i := range players {
		if players[i].Gamertag == gamertag {
			return players[i].XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent de db_profiles.json", gamertag)
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
