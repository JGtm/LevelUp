// cmd/backfill-team-scores — corrige `match_registry.team_0_score` / `team_1_score`
// (Halo Infinite) sur les matchs dont la base porte autre chose que le score AFFICHÉ.
//
// # CE QU'IL RÉPARE, ET POURQUOI UN RE-SYNC NE SUFFIT PAS
//
// Mesure du 2026-08-24 sur les 1 934 matchs du corpus
// (`.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md`) : 80 lignes portent une valeur qui
// n'est pas le score affiché — 69 d'entre elles portent, au tick près, le compteur brut du
// mode (`Teams[].Stats.ZonesStats.StrongholdScoringTicks`). Héritage d'une période où l'API
// 343 servait ce compteur dans `CoreStats.Score` ; elle a été corrigée entre avril et mai
// 2026, et les 396 matchs entrés depuis sont justes à 100 %. Le défaut est donc CLOS À LA
// SOURCE : il ne reste qu'un résidu.
//
// Une resynchronisation ne le répare PAS : `persistMatchRegistry`
// (`internal/persist/shared_persister.go:154`) est un INSERT NU, sans `ON CONFLICT` — un
// match déjà présent n'est jamais réécrit. D'où cet outil.
//
// # DEUX PHASES, ET LA RAISON EST LE VERROU
//
// Phase A — SANS aucun droit d'écriture : lecture des lignes visées, téléchargement des
// payloads, décisions. C'est la phase longue (un appel API par match, de l'ordre de la
// minute pour 80 matchs, bien davantage si l'API traîne).
//
// Phase B — n'existe qu'avec `--apply`, et elle est COURTE : ouverture en écriture, puis
// par ligne une re-lecture de la valeur courante suivie de l'écriture, seulement si elle
// diffère toujours.
//
// Découper ainsi n'est pas cosmétique. Prendre le droit d'écriture avant la boucle de
// téléchargement tiendrait la base fermée pendant toute la durée des appels réseau —
// des heures si l'API répond mal — alors qu'aucune écriture n'a encore lieu. La re-lecture
// en phase B rend la séquence sûre même si quelque chose a bougé entre les deux phases.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne fait JAMAIS confiance au fichier de liste pour les VALEURS : le TSV ne fournit que
// des `match_id`, et chaque score est re-téléchargé à l'exécution. Il n'écrit que si la
// valeur diffère, jamais un NULL, jamais un négatif, jamais hors bornes du SMALLINT de la
// colonne. Un match sans camps 0 et 1 (FFA) est sauté et loggé.
//
// # QUI GARANTIT L'UNICITÉ DU WRITER
//
// C'est le VERROU FICHIER de DuckDB, pas autre chose : `OpenReadWrite` ping la base à
// l'ouverture (`platform/duckdb/db.go:461`), donc `--apply` échoue immédiatement, avec un
// « Could not set lock », si le serveur LevelUp tient déjà la base. Le serveur doit donc
// être arrêté (ADR 0013, un seul writer par base).
//
// Le lease `dblease` pris en phase B ne joue AUCUN rôle là-dedans : c'est un mutex
// INTRA-process (`platform/dblease/lease.go:45`), sans effet entre deux process. Il est
// conservé pour la discipline et la métrique, pas comme garantie d'exclusion.
//
// # USAGE
//
//	# 1. répétition à blanc (aucune écriture, aucun droit d'écriture demandé)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> \
//	  go run ./cmd/backfill-team-scores --gamertag JGtm
//
//	# 2. application (SERVEUR ARRÊTÉ)
//	LEVELUP_REPO_ROOT=<repo qui porte data/> \
//	  go run ./cmd/backfill-team-scores --gamertag JGtm --apply
//
//	# un seul match
//	go run ./cmd/backfill-team-scores --match 7344d24f-0154-4949-80ad-e2b781c122f1 --apply
//
// Un `--ids-file` relatif est résolu depuis la racine du dépôt (`LEVELUP_REPO_ROOT`), pas
// depuis le répertoire courant. Auth : `MultiUserTokenStore` (ADR 0023), aucune re-capture.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	auth_platform "levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/dblease"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// defaultIDsFile : la liste nominative produite par la phase 1 du lot, versionnée.
const defaultIDsFile = ".ai/V7.5/replay2d/registre_film/score_equipe_ecarts_2026-08-24.tsv"

// writerLeaseTimeout borne l'attente du lease intra-process. Court : en CLI mono-process
// il est toujours libre, un dépassement signalerait un bug d'appel, pas une contention.
const writerLeaseTimeout = 30 * time.Second

// tally compte les issues. Publié tel quel en fin de course.
//
// `planned` et `fixed` sont DEUX compteurs distincts, et le découpage en deux phases rend
// la distinction nécessaire : la phase A décide (elle incrémente `planned`), la phase B
// écrit (elle incrémente `fixed`). En répétition à blanc la phase B n'existe pas — les
// confondre ferait afficher « corriges=0 » alors que N corrections sont prêtes, et rendrait
// l'avertissement de fin inatteignable. C'est le défaut P1-a de la ronde 2 de revue.
type tally struct {
	read, identical, planned, fixed, skipped, failed int
}

// plannedFix est une correction décidée en phase A, en attente de la phase B.
type plannedFix struct {
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

	matchIDs, err := resolveMatchIDs(opts.single, opts.idsFile, cfg.RepoRoot)
	if err != nil {
		fatal("%v", err)
	}
	client, err := newAPIClient(ctx, cfg, pr, opts.gamertag, opts.rps)
	if err != nil {
		fatal("%v", err)
	}

	slog.InfoContext(ctx, "backfill_team_scores: démarrage",
		"matchs", len(matchIDs), "apply", opts.apply, "shared_db", sharedPath)

	// --- Phase A : aucun droit d'écriture n'est demandé ici. ---
	plans, t, err := planPhase(ctx, client, sharedPath, matchIDs)
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
	gamertag, idsFile, single string
	apply                     bool
	rps                       int
}

func parseFlags() options {
	var o options
	flag.StringVar(&o.gamertag, "gamertag", "JGtm", "Gamertag dont les tokens servent à l'auth API (db_profiles.json)")
	flag.StringVar(&o.idsFile, "ids-file", defaultIDsFile, "Liste de match_id (TSV à en-tête `match_id`, ou une colonne nue). Relatif = depuis la racine du dépôt")
	flag.StringVar(&o.single, "match", "", "Traiter CE seul match_id (ignore --ids-file)")
	flag.BoolVar(&o.apply, "apply", false, "ÉCRIRE réellement. Sans ce drapeau : répétition à blanc, aucune écriture")
	flag.IntVar(&o.rps, "rps", 4, "Requêtes API par seconde")
	flag.Parse()
	return o
}

// reportTally publie le résumé. `planifiees` et `corriges` sont affichés dans LES DEUX
// modes : en répétition à blanc on lit `planifiees=N corriges=0`, après application
// `planifiees=N corriges=N`. Un écart entre les deux en mode `--apply` se voit donc d'un
// coup d'oeil, au lieu d'être noyé.
func reportTally(ctx context.Context, apply bool, t tally) {
	slog.InfoContext(ctx, "backfill_team_scores: terminé",
		"apply", apply, "lus", t.read, "identiques", t.identical,
		"planifiees", t.planned, "corriges", t.fixed,
		"skippes", t.skipped, "echecs", t.failed)
	if !apply && t.planned > 0 {
		slog.WarnContext(ctx, "backfill_team_scores: répétition à blanc — RIEN n'a été écrit ; relancer avec --apply (serveur arrêté) pour appliquer",
			"a_corriger", t.planned)
	}
}

// planPhase ouvre la base en LECTURE, décide pour chaque match, et rend le plan.
//
// `OpenReadForQuery` dégénère en ouverture read-only depuis un process séparé : la lecture
// cohabite avec un serveur qui tient la base EN LECTURE, mais avorte proprement (« Could
// not set lock ») s'il est en train d'y écrire. C'est une limite acceptée : la répétition à
// blanc n'a rien à réparer, on la relance.
func planPhase(ctx context.Context, f matchFetcher, path string, ids []string) ([]plannedFix, tally, error) {
	var t tally
	db, closeDB, err := duckdbpkg.OpenReadForQuery(path)
	if err != nil {
		return nil, t, fmt.Errorf("ouverture lecture de %s : %w", path, err)
	}
	defer closeDB()

	reg := sqlRegistry{q: db}
	var plans []plannedFix
	for _, id := range ids {
		if p, ok := planMatch(ctx, f, reg, id, &t); ok {
			plans = append(plans, p)
		}
	}
	return plans, t, nil
}

// planMatch traite UN match en phase A : lecture, fetch, décision. ok=true si une
// correction est à faire.
func planMatch(ctx context.Context, f matchFetcher, r registryReader, id string, t *tally) (plannedFix, bool) {
	cur, found, err := r.ReadScores(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: lecture match_registry échouée", "match_id", id, "err", err)
		t.failed++
		return plannedFix{}, false
	}
	if !found {
		slog.WarnContext(ctx, "backfill_team_scores: match absent de match_registry — sauté", "match_id", id)
		t.skipped++
		return plannedFix{}, false
	}
	matchJSON, err := f.GetMatchStats(ctx, id)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: GetMatchStats échoué", "match_id", id, "err", err)
		t.failed++
		return plannedFix{}, false
	}
	t.read++

	dec := Decide(matchJSON, cur)
	switch dec.Verdict {
	case VerdictIdentical:
		t.identical++
		slog.DebugContext(ctx, "backfill_team_scores: identique", "match_id", id, "cause", dec.Reason)
		return plannedFix{}, false
	case VerdictSkipNoTeams, VerdictSkipImplausible:
		t.skipped++
		slog.WarnContext(ctx, "backfill_team_scores: sauté",
			"match_id", id, "verdict", string(dec.Verdict), "cause", dec.Reason)
		return plannedFix{}, false
	}
	t.planned++
	slog.InfoContext(ctx, "backfill_team_scores: correction retenue",
		"match_id", id,
		"avant", formatScores(dec.Old), "apres", fmt.Sprintf("%d/%d", dec.NewTeam0, dec.NewTeam1),
		"cause", dec.Reason)
	return plannedFix{matchID: id, decision: dec}, true
}

// applyPhase ouvre la base en ÉCRITURE et applique le plan. Fenêtre volontairement courte :
// plus aucun appel réseau ici.
func applyPhase(ctx context.Context, path string, plans []plannedFix, t *tally) error {
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

// applyOne écrit UNE correction, après re-lecture.
//
// La re-lecture n'est pas une précaution de principe : entre la phase A et la phase B il
// s'est écoulé le temps de tous les appels API, et rien ne garantit que la ligne n'a pas
// bougé. Si elle porte déjà la valeur visée, on ne réécrit pas — l'outil reste idempotent
// et un second passage ne compte rien à tort.
func applyOne(ctx context.Context, r registryReader, w registryWriter, p plannedFix, t *tally) {
	cur, found, err := r.ReadScores(ctx, p.matchID)
	if err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: re-lecture avant écriture échouée", "match_id", p.matchID, "err", err)
		t.failed++
		return
	}
	if !found {
		slog.WarnContext(ctx, "backfill_team_scores: match disparu du registre entre les deux phases — sauté", "match_id", p.matchID)
		t.skipped++
		return
	}
	if cur.Team0 != nil && cur.Team1 != nil &&
		*cur.Team0 == p.decision.NewTeam0 && *cur.Team1 == p.decision.NewTeam1 {
		t.identical++
		slog.InfoContext(ctx, "backfill_team_scores: déjà à jour à la seconde lecture — rien à écrire", "match_id", p.matchID)
		return
	}
	if err := w.WriteScores(ctx, p.matchID, p.decision.NewTeam0, p.decision.NewTeam1); err != nil {
		slog.ErrorContext(ctx, "backfill_team_scores: écriture échouée", "match_id", p.matchID, "err", err)
		t.failed++
		return
	}
	t.fixed++
	slog.InfoContext(ctx, "backfill_team_scores: écrit",
		"match_id", p.matchID,
		"avant", formatScores(cur), "apres", fmt.Sprintf("%d/%d", p.decision.NewTeam0, p.decision.NewTeam1))
}

// resolveMatchIDs rend la liste de travail : le match unique s'il est fourni, sinon le
// fichier. Un chemin relatif est résolu depuis la racine du dépôt.
func resolveMatchIDs(single, idsFile, repoRoot string) ([]string, error) {
	if single != "" {
		return []string{single}, nil
	}
	path := idsFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	return LoadMatchIDs(path)
}

// newAPIClient résout le xuid du porteur puis rafraîchit ses tokens (ADR 0023).
// AUCUNE re-capture : un refresh token valide se rafraîchit ; s'il est mort, on s'arrête
// en le disant plutôt que d'en fabriquer un nouveau.
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
