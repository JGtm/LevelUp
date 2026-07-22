package scheduler

// data_health_lusr.go — volet « garde-fou trous LUSR » du HealthScheduler :
// scan read-only par joueur (skill.ScanLUSRGaps) + auto-heal borné OFF par défaut.
// Extrait de data_health_check.go pour tenir la limite de taille de fichier
// (CLAUDE.md n°5) ; le champ `lusrAutoHeal` reste sur le struct HealthScheduler.

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/skill"
)

// lusrAutoHealMinGaps : nombre minimal de trous d'intérieur d'un joueur pour
// déclencher un auto-heal (replay). En dessous, on laisse le prochain cycle de
// sync résorber (les trous permanents ne bougent pas seuls, mais un seul trou ne
// justifie pas un replay complet coûteux). Un seul joueur (le plus impacté) est
// soigné par cycle.
const lusrAutoHealMinGaps = 3

// lusrHealCandidate : le joueur le plus impacté d'un cycle, candidat à l'auto-heal.
type lusrHealCandidate struct {
	titleSlug    string
	gamertag     string
	interiorGaps int
}

// isLUSRAutoHealEnabled : kill-switch de l'auto-heal LUSR.
//
// Défaut OFF (basculé le 2026-07-21) : le garde-fou trous LUSR démarre en ALERTE
// SEULE. Passer ON après observation de la jauge `levelup.lusr_v2.interior_gaps`
// et validation d'un replay manuel (bouton « Recalculer » du panneau monitoring).
// Retrait cible du flag : une fois l'auto-heal validé en prod (≥ 2 semaines
// stables, gauge → 0 au cycle suivant chaque heal, sans hausse de
// `canonical_write_held_watermark_total`), câbler ON par défaut et supprimer ce
// flag. Critère mesurable : interior_gaps retombe à 0 après chaque heal.
func isLUSRAutoHealEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LEVELUP_LUSR_AUTOHEAL_ENABLED")))
	return v == "1" || v == "true" || v == "yes"
}

// WithLUSRAutoHeal injecte le hook de remédiation LUSR (replay d'un joueur).
// Câblé par main.go APRÈS NewRouter (le hook pointe vers un runner du registry).
// Sans ce câblage OU sans le kill-switch ON, l'auto-heal ne fire jamais.
func (s *HealthScheduler) WithLUSRAutoHeal(fn func(ctx context.Context, titleSlug, gamertag string) error) *HealthScheduler {
	s.lusrAutoHeal = fn
	return s
}

// auditTitleLUSRGaps scanne les trous LUSR de tous les joueurs d'un titre
// (ScanLUSRGaps : read-only, croise candidats/watermark sur la shared DB avec les
// lignes LUSR notées de chaque player DB). AGRÈGE les compteurs dans res. La chaîne
// LUSR étant title-aware, on injecte le slug dans le ctx. Best-effort, sémantique
// « unmeasured ≠ sain » (ne casse jamais le cycle) :
//   - joueur sans xuid.txt → répertoire non résolu, PAS un joueur mesurable :
//     skip silencieux (Debug), ni ProbeErrors ni Unmeasured ;
//   - player DB présente mais inouvrable (typiquement tenue RW par un sync) OU
//     os.ReadDir du répertoire joueurs KO → WARN + res.LUSRPlayersUnmeasured : le
//     scan devient PARTIEL (la jauge ne sera pas republiée, cf. runCycle) ;
//   - scan SQL en échec → WARN + res.LUSRPlayersUnmeasured ET res.ProbeErrors (une
//     sonde SQL ratée = cron non « sain », en plus du scan partiel).
func (s *HealthScheduler) auditTitleLUSRGaps(ctx context.Context, pr *titlePkg.PathResolver, slug string, sharedDB *duckdb.DB, res *DataHealthCheckResult, healCand *lusrHealCandidate) {
	titleCtx := ctxkeys.WithTitleSlug(ctx, slug)
	sharedSQL := sharedDB.SQLDb()
	entries, err := os.ReadDir(pr.PlayersRootDir(slug))
	if err != nil {
		// Répertoire joueurs illisible → AUCUN joueur du titre n'a pu être scanné. Un
		// seul +1 (et non 1/joueur, nombre inconnu ici) suffit à marquer le scan
		// partiel : il gèle la republication de la jauge (runCycle), l'invariant visé.
		slog.WarnContext(ctx, "data_health: répertoire joueurs illisible — scan LUSR du titre non mesuré",
			"titleSlug", slug, "err", err)
		res.LUSRPlayersUnmeasured++
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		gamertag := e.Name()
		xuid := readXUIDFromDir(pr.PlayerDir(slug, gamertag))
		if xuid == "" {
			continue // pas de xuid.txt → joueur non résolu, skip silencieux
		}
		playerPath := pr.PlayerDBPath(slug, gamertag)
		if _, err := os.Stat(playerPath); err != nil {
			continue
		}
		pdb, err := openDBShared(playerPath)
		if err != nil {
			// Player DB présente mais inouvrable (typiquement tenue RW par un sync en
			// cours) → joueur NON mesuré ce cycle. WARN + compteur : le scan devient
			// partiel (jauge gelée par runCycle) plutôt qu'un sous-comptage muet. Pas
			// de ProbeErrors : une DB tenue RW est transitoire, elle ne fait pas
			// échouer le cron.
			slog.WarnContext(ctx, "data_health: player DB inouvrable pour scan LUSR (non mesuré)",
				"titleSlug", slug, "gamertag", gamertag, "err", err)
			res.LUSRPlayersUnmeasured++
			continue
		}
		scanCtx, cancel := context.WithTimeout(titleCtx, 60*time.Second)
		rep, serr := skill.ScanLUSRGaps(scanCtx, pdb.SQLDb(), sharedSQL, xuid)
		cancel()
		_ = pdb.Close() //nolint:errcheck // ref-count : best-effort
		if serr != nil {
			// Sonde SQL ratée : cron non « sain » (ProbeErrors) ET joueur non mesuré
			// (Unmeasured → jauge gelée). Les deux compteurs sont incrémentés.
			slog.WarnContext(ctx, "data_health: scan trous LUSR échoué",
				"titleSlug", slug, "gamertag", gamertag, "err", serr)
			res.ProbeErrors++
			res.LUSRPlayersUnmeasured++
			continue
		}
		res.LUSRInteriorGaps += rep.TotalInteriorGaps
		res.LUSRPendingRecent += rep.TotalPendingRecent
		res.LUSRPlayersScanned++
		// Retient le joueur le plus impacté (candidat auto-heal, 1/cycle).
		if rep.TotalInteriorGaps > healCand.interiorGaps {
			healCand.titleSlug = slug
			healCand.gamertag = gamertag
			healCand.interiorGaps = rep.TotalInteriorGaps
		}
	}
}

// maybeAutoHealLUSR déclenche le replay du joueur le plus impacté — SEULEMENT si
// le hook est câblé, le kill-switch ON, et l'impact ≥ seuil. Borné à 1 joueur par
// cycle (le replay est complet et coûteux). Best-effort : un échec est loggé, le
// cycle continue.
func (s *HealthScheduler) maybeAutoHealLUSR(ctx context.Context, cand lusrHealCandidate) {
	if s.lusrAutoHeal == nil || !isLUSRAutoHealEnabled() {
		return
	}
	if cand.gamertag == "" || cand.interiorGaps < lusrAutoHealMinGaps {
		return
	}
	slog.WarnContext(ctx, "data_health: auto-heal LUSR déclenché (replay)",
		"titleSlug", cand.titleSlug, "gamertag", cand.gamertag, "interior_gaps", cand.interiorGaps)
	if err := s.lusrAutoHeal(ctx, cand.titleSlug, cand.gamertag); err != nil {
		slog.ErrorContext(ctx, "data_health: auto-heal LUSR échoué",
			"titleSlug", cand.titleSlug, "gamertag", cand.gamertag, "err", err)
	}
}

// readXUIDFromDir lit le xuid.txt d'un répertoire joueur (1 ligne). Vide si absent
// ou illisible — le caller skippe alors le joueur.
func readXUIDFromDir(playerDir string) string {
	data, err := os.ReadFile(filepath.Clean(filepath.Join(playerDir, "xuid.txt")))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
