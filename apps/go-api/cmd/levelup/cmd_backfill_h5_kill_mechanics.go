package main

// cmd_backfill_h5_kill_mechanics.go — sous-commande one-shot
// `levelup backfill-h5-kill-mechanics`.
//
// CONTEXTE (diagnostic 2026-07-24, parquets) : les 3 colonnes de mecaniques de kill
// natives Halo 5 de shared.match_participants (assassination_kills, ground_pound_kills,
// shoulder_bash_kills) ont ete ecrites A ZERO (pas NULL) pour TOUS les matchs ingeres
// AVANT l'activation du mapper carnage (created_at < 2026-06-23T13:00Z), plus un batch
// residuel le 2026-06-26 (14:00->15:00). ~11 200 lignes concernees (dont l'historique
// complet de JGtm : 1 970 matchs a 0/0/0). Les ingestions posterieures portent les vraies
// valeurs. Ces 3 compteurs sont DISJOINTS de melee_kills (mecaniques natives h5).
//
// STRATEGIE : re-fetch le carnage h5 (SOURCE DE VERITE, MEME client + DTO que le sync
// live — halo5.NewCaptureSource + GetMatchCarnage) de chaque match candidat, puis UPDATE
// cible row-by-row (WHERE match_id=? AND xuid=?, placeholders lies -> ART-safe, meme forme
// que cmd/backfill_kda_accuracy) des lignes dont la valeur stockee differe de la verite
// carnage. match_participants N'EST PAS append-only (absente de tablesProtegees / ADR 0026) :
// l'UPDATE cible single-writer serialise est le pattern sanctionne (PostSyncEnrichment).
// Colonnes NON indexees + WHERE sur la PK non modifiee -> aucune suppression d'index (le
// declencheur ART #23046). Outil dans cmd/ -> hors perimetre des garde-rails (no_art_patterns,
// shared_write_guard excluent /cmd/ : one-shot mono-process, serveur arrete) : aucune
// entree d'allowlist requise.
//
// SELECTION (par MOTIF + fenetre, robuste) : un match est candidat s'il porte AU MOINS une
// ligne participant a 0/0/0 dont created_at < cutoff. Le motif 0/0/0 capte la corruption ;
// le cutoff (defaut 2026-06-26T15:00Z, APRES le 2e batch) borne le re-fetch aux ingestions
// de la fenetre de corruption (les matchs correctement ingeres ensuite sont hors scope). Le
// diff par-ligne garantit qu'un match partiellement complete (topup roster, cf.
// h5-roster-refetch) voit quand meme ses lignes 0/0/0 d'origine corrigees. cutoff vide =
// pur motif (re-fetch de TOUS les matchs a 0/0/0, y compris les vrais-zeros).
//
// IDEMPOTENT : apres correction une ligne porte la vraie valeur -> le match sort du motif
// des qu'au moins un joueur a une mecanique. Un match ou PERSONNE n'a de mecanique (vrai
// 0/0/0 partout) reste selectionne mais n'ecrit RIEN (stored == truth par-ligne) : re-fetch
// no-op assume pour un one-shot (documente).
//
// Usage (SERVEUR ARRETE — OpenReadWrite echoue si le lock est tenu) :
//
//	levelup backfill-h5-kill-mechanics --dry-run --limit 20     # selection + fetch echantillon, aucune ecriture
//	levelup backfill-h5-kill-mechanics                          # ecrit (tous les candidats)
//	levelup backfill-h5-kill-mechanics --auth-as JGtm --cutoff 2026-06-26T15:00:00Z
//	levelup backfill-h5-kill-mechanics --cutoff ""              # pur motif (aucune borne temporelle)
//
// LEVELUP_H5_AUTH_AS surcharge --auth-as (parite avec les autres outils h5). Le carnage h5
// (/h5/{mode}/matches/{id}, header Spartan v4 sans clearance ni xuid) sert l'historique de
// n'importe quel gamertag avec n'importe quel token v4 valide (JGtm = seul RT vivant).

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/duckdb"
)

// defaultH5MechCutoff : borne created_at par defaut. Fixee APRES le 2e batch corrompu
// (2026-06-26 14:00->15:00) pour capter les DEUX fenetres de corruption via le motif 0/0/0,
// tout en excluant les ingestions correctes posterieures. Cf. godoc du fichier.
const defaultH5MechCutoff = "2026-06-26T15:00:00Z"

// h5MechCarnageMode : segment d'URL du carnage. "arena" couvre l'Arena classe ET social
// (GameMode 1) — le seul mode h5 ingere (Campagne/Warzone exclus a la collecte). Un match
// residuel d'un autre mode echoue au fetch -> skip best-effort (compte). Meme constante que
// h5-roster-refetch / livesync/csr_match.go.
const h5MechCarnageMode = "arena"

// mechTriple : les 3 compteurs de mecaniques de kill natives h5 (verite carnage).
type mechTriple struct {
	assassination int
	groundPound   int
	shoulderBash  int
}

// storedMech : l'etat persiste d'une ligne match_participants (identite + 3 compteurs).
type storedMech struct {
	xuid     string
	gamertag string
	values   mechTriple
}

// mechUpdate : une correction a appliquer sur une ligne (match_id, xuid).
type mechUpdate struct {
	xuid   string
	values mechTriple
}

// mechBackfillStats : bilan chiffre d'une passe (dry-run ou reel).
type mechBackfillStats struct {
	candidates     int // matchs selectionnes (motif 0/0/0 + fenetre)
	fetched        int // carnage fetches OK
	skipped        int // matchs sautes (lecture DB ou fetch carnage KO, ou commit KO)
	matchesUpdated int // matchs ayant au moins une ligne ecrite (0 en dry-run)
	rowsUpdated    int // lignes corrigees (ou qui SERAIENT corrigees en dry-run)
	rowsUnchanged  int // lignes deja a la verite carnage (aucune ecriture)
	rowsUnmatched  int // lignes dont le gamertag est absent du carnage (non corrigeables)
}

func runBackfillH5KillMechanics(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-h5-kill-mechanics", flag.ExitOnError)
	titleSlug := fs.String("title", halo5.TitleSlug, "slug du titre (H5-only par construction)")
	authGT := fs.String("auth-as", "JGtm", "gamertag dont les tokens authentifient l'API carnage (LEVELUP_H5_AUTH_AS surcharge)")
	dryRun := fs.Bool("dry-run", false, "selection + fetch sans ecriture (--limit borne l'echantillon)")
	limit := fs.Int("limit", 0, "borne le nombre de matchs candidats traites (0 = tous)")
	cutoff := fs.String("cutoff", defaultH5MechCutoff, "created_at max des lignes 0/0/0 (RFC3339 ; vide = aucune borne)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if v := os.Getenv("LEVELUP_H5_AUTH_AS"); v != "" {
		*authGT = v
	}
	ctx := context.Background()

	cutoffTime, err := parseH5MechCutoff(*cutoff)
	if err != nil {
		return err
	}

	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(*titleSlug)
	if _, statErr := os.Stat(sharedPath); statErr != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, statErr)
	}

	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	authCtx, src, err := setupH5CarnageSource(ctx, cfg, pr.WatcherTokensDir(), *authGT)
	if err != nil {
		return err
	}

	matchIDs, err := loadZeroMechanicMatches(ctx, db, *titleSlug, cutoffTime, *limit)
	if err != nil {
		return fmt.Errorf("selection candidats: %w", err)
	}
	fmt.Printf("backfill-h5-kill-mechanics : title=%s auth_as=%s cutoff=%s dry_run=%v candidats=%d\n",
		*titleSlug, *authGT, *cutoff, *dryRun, len(matchIDs))
	if len(matchIDs) == 0 {
		fmt.Println("Rien a faire.")
		return nil
	}

	stats := runMechBackfill(authCtx, db, src, matchIDs, !*dryRun)
	if !*dryRun {
		// Flush explicite du WAL vers le fichier (durabilite immediate ; Close le fait
		// aussi mais on ne prend pas le risque d'un WAL non checkpointe, cf. ADR 0022).
		if _, cpErr := db.ExecContext(ctx, "CHECKPOINT"); cpErr != nil {
			slog.WarnContext(ctx, "backfill-h5-kill-mechanics: CHECKPOINT final KO (Close flushera)", "err", cpErr)
		}
	}
	printMechSummary(stats, *dryRun)
	return nil
}

// parseH5MechCutoff parse le cutoff RFC3339. Vide -> time.Time zero (pas de borne).
func parseH5MechCutoff(s string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("cutoff invalide %q (attendu RFC3339, ex. 2026-06-26T15:00:00Z): %w", s, err)
	}
	return t, nil
}

// setupH5CarnageSource resout l'xuid d'auth (db_profiles), rafraichit les tokens
// store-first (ADR 0023) et construit la CaptureSource live. Le SpartanToken est fige
// dans le *Client a la construction -> le ctx retourne (porteur de l'auth) suffit pour
// tous les GetMatchCarnage. Cablage identique a cmd/h5-kill-kind-backfill.
func setupH5CarnageSource(ctx context.Context, cfg *config.AppConfig, tokensDir, authGT string) (context.Context, halo5.CaptureSource, error) {
	authXUID := resolveH5AuthXUID(cfg, authGT)
	if authXUID == "" {
		return nil, nil, fmt.Errorf("xuid auth introuvable pour %q dans db_profiles", authGT)
	}
	store := auth.NewMultiUserTokenStore(tokensDir)
	res, err := auth.RefreshHaloTokensViaStoreFirst(ctx, store, auth.NewSISUProvider(), authXUID, authGT, auth.LegacyAuthInputs{})
	if err != nil || res == nil || res.Tokens == nil {
		return nil, nil, fmt.Errorf("refresh tokens auth_as=%s: %w", authGT, err)
	}
	authCtx := ctxkeys.WithHaloAuth(ctx, res.Tokens, authXUID)
	src, err := halo5.NewCaptureSource(authCtx)
	if err != nil {
		return nil, nil, fmt.Errorf("NewCaptureSource: %w", err)
	}
	return authCtx, src, nil
}

// resolveH5AuthXUID mappe le gamertag d'auth -> xuid via db_profiles (titre h5 puis
// global). "" si introuvable.
func resolveH5AuthXUID(cfg *config.AppConfig, authGT string) string {
	for _, slug := range []string{halo5.TitleSlug, ""} {
		ps, err := cfg.LoadPlayers(slug)
		if err != nil {
			continue
		}
		for i := range ps {
			if ps[i].Gamertag == authGT {
				return ps[i].XUID
			}
		}
	}
	return ""
}

// loadZeroMechanicMatches selectionne les match_id (recents d'abord) ayant AU MOINS une
// ligne participant a 0/0/0 dont created_at < cutoff (Campagne exclue read-side canonique).
// cutoff zero -> aucune borne temporelle. Meme forme DISTINCT + tri temporel canonique que
// cmd/backfill_kda_accuracy (proven sur DuckDB).
func loadZeroMechanicMatches(ctx context.Context, db *sql.DB, titleSlug string, cutoff time.Time, limit int) ([]string, error) {
	where := `COALESCE(mp.assassination_kills,0)=0 AND COALESCE(mp.ground_pound_kills,0)=0 AND COALESCE(mp.shoulder_bash_kills,0)=0`
	var qargs []any
	if !cutoff.IsZero() {
		where += ` AND mp.created_at < ?`
		qargs = append(qargs, cutoff)
	}
	q := `SELECT DISTINCT mp.match_id
	      FROM match_participants mp
	      JOIN match_registry mr ON mr.match_id = mp.match_id
	      WHERE ` + where +
		analysis.SQLExcludeCampaignVariants(titleSlug, "mr") +
		` ORDER BY ` + analysis.SQLStartTimeCanonical("mr") + ` DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q, qargs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// runMechBackfill traite chaque candidat best-effort (un fetch/commit KO saute le match,
// sans bloquer les autres) et retourne le bilan. commit=false -> dry-run (compte sans ecrire).
func runMechBackfill(ctx context.Context, db *sql.DB, src halo5.CaptureSource, matchIDs []string, commit bool) mechBackfillStats {
	stats := mechBackfillStats{candidates: len(matchIDs)}
	for i, mid := range matchIDs {
		stored, err := loadMatchMechanics(ctx, db, mid)
		if err != nil {
			stats.skipped++
			slog.WarnContext(ctx, "backfill-h5-kill-mechanics: lecture participants KO (skip)", "match_id", mid, "err", err)
			continue
		}
		carnage, cerr := src.GetMatchCarnage(ctx, mid, h5MechCarnageMode)
		if cerr != nil {
			stats.skipped++
			slog.WarnContext(ctx, "backfill-h5-kill-mechanics: carnage indisponible (skip)", "match_id", mid, "err", cerr)
			continue
		}
		stats.fetched++
		updates, unchanged, unmatched := planMechanicUpdates(stored, carnageMechByGamertag(carnage))
		stats.rowsUnchanged += unchanged
		stats.rowsUnmatched += unmatched
		if len(updates) == 0 {
			continue
		}
		if commit {
			if err := applyMechanicUpdates(ctx, db, mid, updates); err != nil {
				stats.skipped++
				slog.WarnContext(ctx, "backfill-h5-kill-mechanics: UPDATE KO (match saute)", "match_id", mid, "err", err)
				continue
			}
			stats.matchesUpdated++
		}
		stats.rowsUpdated += len(updates)
		if (i+1)%200 == 0 {
			slog.InfoContext(ctx, "backfill-h5-kill-mechanics: progression",
				"fait", i+1, "total", len(matchIDs), "lignes_maj", stats.rowsUpdated, "sautes", stats.skipped)
		}
	}
	return stats
}

// loadMatchMechanics charge l'identite + les 3 compteurs stockes de tous les participants
// (xuid non vide) d'un match. La map gamertag->xuid n'est PAS reconstruite : on corrige les
// lignes EXISTANTES par leur (match_id, xuid), le gamertag stocke (issu du carnage d'origine)
// sert de cle de jointure vers le carnage re-fetche.
func loadMatchMechanics(ctx context.Context, db *sql.DB, matchID string) ([]storedMech, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT xuid, COALESCE(gamertag, ''),
		       COALESCE(assassination_kills, 0), COALESCE(ground_pound_kills, 0), COALESCE(shoulder_bash_kills, 0)
		FROM match_participants
		WHERE match_id = ? AND xuid IS NOT NULL AND xuid <> ''`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []storedMech
	for rows.Next() {
		var s storedMech
		if err := rows.Scan(&s.xuid, &s.gamertag, &s.values.assassination, &s.values.groundPound, &s.values.shoulderBash); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// carnageMechByGamertag projette le carnage -> map gamertag -> 3 mecaniques (verite). PURE
// (aucun DB/reseau) -> unit-testable. Un gamertag vide est ignore (jamais cle de jointure).
func carnageMechByGamertag(carnage *halo5.H5CarnageResponse) map[string]mechTriple {
	out := map[string]mechTriple{}
	if carnage == nil {
		return out
	}
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		if p.Player.Gamertag == "" {
			continue
		}
		out[p.Player.Gamertag] = mechTriple{
			assassination: p.TotalAssassinations,
			groundPound:   p.TotalGroundPoundKills,
			shoulderBash:  p.TotalShoulderBashKills,
		}
	}
	return out
}

// planMechanicUpdates decide, par ligne stockee, s'il faut corriger : le gamertag absent du
// carnage est non-corrigeable (unmatched) ; une ligne deja a la verite est inchangee ; sinon
// on planifie l'UPDATE avec la verite carnage. PURE (aucun DB/reseau) -> unit-testable ; porte
// l'idempotence (une 2e passe sur des lignes deja corrigees ne planifie rien).
func planMechanicUpdates(stored []storedMech, truth map[string]mechTriple) (updates []mechUpdate, unchanged, unmatched int) {
	for _, s := range stored {
		t, ok := truth[s.gamertag]
		if !ok {
			unmatched++
			continue
		}
		if s.values == t {
			unchanged++
			continue
		}
		updates = append(updates, mechUpdate{xuid: s.xuid, values: t})
	}
	return updates, unchanged, unmatched
}

// applyMechanicUpdates ecrit les corrections d'un match en UNE transaction : N UPDATE
// row-by-row `WHERE match_id=? AND xuid=?` (placeholders lies -> ART-safe, colonnes non
// indexees, PK non modifiee). Serialise, serveur arrete (single-writer). Rollback si KO.
func applyMechanicUpdates(ctx context.Context, db *sql.DB, matchID string, updates []mechUpdate) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit reussi
	for _, u := range updates {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match_participants
			 SET assassination_kills = ?, ground_pound_kills = ?, shoulder_bash_kills = ?
			 WHERE match_id = ? AND xuid = ?`,
			u.values.assassination, u.values.groundPound, u.values.shoulderBash, matchID, u.xuid); err != nil {
			return fmt.Errorf("UPDATE %s/%s: %w", matchID, u.xuid, err)
		}
	}
	return tx.Commit()
}

// printMechSummary imprime le bilan final chiffre (dry-run vs reel).
func printMechSummary(stats mechBackfillStats, dryRun bool) {
	fmt.Printf("\n=== BILAN ===\n")
	fmt.Printf("Candidats           : %d\n", stats.candidates)
	fmt.Printf("Carnage fetches     : %d\n", stats.fetched)
	fmt.Printf("Sautes (DB/API/KO)  : %d\n", stats.skipped)
	fmt.Printf("Lignes inchangees   : %d (deja a la verite carnage)\n", stats.rowsUnchanged)
	fmt.Printf("Lignes non mappees  : %d (gamertag absent du carnage)\n", stats.rowsUnmatched)
	if dryRun {
		fmt.Printf("[DRY-RUN] %d ligne(s) SERAIENT corrigee(s). Relancer sans --dry-run pour ecrire.\n", stats.rowsUpdated)
		return
	}
	fmt.Printf("CORRIGEES           : %d ligne(s) sur %d match(s) (UPDATE row-by-row, ART-safe)\n",
		stats.rowsUpdated, stats.matchesUpdated)
}
