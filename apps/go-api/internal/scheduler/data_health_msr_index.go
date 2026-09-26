package scheduler

// data_health_msr_index.go — garde périodique « index ART de match_skill_rank
// désynchronisé » (G.3 des finitions v7.5, 2026-09-13).
//
// CONTEXTE, MESURÉ. Le 2026-09-13, le dry-run de la purge des chaînes LUSR
// étrangères a révélé sur la player DB de JGtm un `idx_msr_playlist` désynchronisé :
// le lookup `WHERE playlist_group = ?` rendait 22 lignes là où le scan forcé en
// comptait 1 826. Même famille que le défaut `personal_score_awards` du volet 2
// (duckdb/duckdb#23645, TOUJOURS ouvert en 1.5.5), sur une AUTRE table — et cette
// fois sur une table que TOUS les lecteurs de notation interrogent. Tout lecteur
// applicatif qui filtre `match_skill_rank` par un prédicat indexé a pu servir des
// lignes amputées, en silence.
//
// CE QUE CETTE GARDE FAIT — ET NE FAIT PAS. Strictement la doctrine posée par la
// garde jumelle de personal_score_awards (retirée le 2026-09-20 en même temps que
// les index qu'elle surveillait) :
//   - DÉTECTE et ALERTE (slog + compteurs + jauge expvar) ; le signal remonte au
//     panneau monitoring via DataHealthCheckResult.WarningsTotal ;
//   - NE RÉPARE JAMAIS. La réparation d'un index sur une base dont la corruption
//     n'est pas comprise reste MANUELLE et tracée :
//     `go run ./cmd/repair_msr_index -db <chemin> -repair`, serveur arrêté.
//
// LA RÈGLE DE COMPARAISON N'EST PAS RECOPIÉE ICI : elle vient de
// `internal/platform/duckdb/indexcheck`, avec la carte des axes indexés — la même
// que celle de `cmd/repair_msr_index`. Deux cartes séparées auraient divergé dès
// la première migration d'index ; garde-rail
// `internal/archlint/no_local_msr_axes_test.go`.
//
// COÛT BORNÉ. msrIndexSampleKeys clés distinctes PAR AXE (tirage réservoir,
// différent à chaque cycle → la couverture s'accumule) : 3 scans groupés + 3 × N
// lookups indexés par player DB. Mesuré sur fixture aux migrations réelles :
// cf. TestScanMSRIndexDesyncCout.
//
// TITLE-AGNOSTIC. Aucun `slug == "..."` : la garde boucle sur les titres du
// registre et sonde la PRÉSENCE de la table dans chaque player DB — la présence de
// la table est le prédicat structurel correct (aucune clé de `capabilities.toml` ne
// décrit `match_skill_rank` ; un titre qui ne note pas ses matchs est skippé).

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"time"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/indexcheck"
)

const (
	// msrIndexSampleKeys : clés distinctes contrôlées PAR AXE et par player DB.
	// Borne le coût ; le tirage étant aléatoire, les cycles successifs balaient
	// progressivement le corpus.
	msrIndexSampleKeys = 200
	// msrIndexProbeTimeout : garde-fou par player DB.
	msrIndexProbeTimeout = 60 * time.Second
	// msrIndexGauge : jauge expvar publiée à chaque cycle COMPLET (clés en écart).
	msrIndexGauge = "data_health_msr_index_desync_keys"
)

// errMSRTableAbsent : la player DB ne porte pas match_skill_rank (titre sans
// notation, DB neuve). Skip silencieux — ce n'est pas une anomalie.
var errMSRTableAbsent = errors.New("match_skill_rank absente")

// msrIndexReport : résultat AGRÉGÉ sur les trois axes d'UNE player DB.
type msrIndexReport struct {
	KeysSampled   int      // clés effectivement comparées, tous axes confondus
	KeysDiverging int      // clés dont le lookup indexé ne rend pas le compte du scan
	RowsMissing   int      // total de lignes invisibles au lookup indexé
	AxesDiverging []string // noms des axes en écart (log + remédiation ciblée)
}

// scanMSRIndexDesync compare, sur un échantillon borné de clés et pour chacun des
// trois axes indexés, le comptage par SCAN FORCÉ au comptage par LOOKUP INDEXÉ.
func scanMSRIndexDesync(ctx context.Context, db *sql.DB, sampleKeys int) (msrIndexReport, error) {
	var rep msrIndexReport
	present, err := indexcheck.TableExists(ctx, db, indexcheck.MatchSkillRankTable)
	if err != nil {
		return rep, err
	}
	if !present {
		return rep, errMSRTableAbsent
	}

	reports, err := indexcheck.RunAll(ctx, db, indexcheck.MatchSkillRankAxes(), indexcheck.Options{
		Table:   indexcheck.MatchSkillRankTable,
		MaxKeys: sampleKeys,
		Sample:  true,
	})
	if err != nil {
		return rep, err
	}
	for _, r := range reports {
		rep.KeysSampled += r.Keys
		rep.KeysDiverging += len(r.Divergences)
		rep.RowsMissing += r.RowsMissing()
		if !r.OK() {
			rep.AxesDiverging = append(rep.AxesDiverging, r.Axis)
		}
	}
	return rep, nil
}

// auditTitleMSRIndex contrôle les player DB d'UN titre et AGRÈGE dans res.
// Best-effort, sémantique « unmeasured ≠ sain » alignée sur auditTitleLUSRGaps.
func (s *HealthScheduler) auditTitleMSRIndex(ctx context.Context, pr *titlePkg.PathResolver, slug string, res *DataHealthCheckResult) {
	entries, err := os.ReadDir(pr.PlayersRootDir(slug))
	if err != nil {
		slog.WarnContext(ctx, "data_health: répertoire joueurs illisible — index match_skill_rank non mesuré",
			"titleSlug", slug, "err", err)
		res.MSRIndexPlayersUnmeasured++
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		playerPath := pr.PlayerDBPath(slug, e.Name())
		if _, statErr := os.Stat(playerPath); statErr != nil {
			continue
		}
		s.auditPlayerMSRIndex(ctx, slug, e.Name(), playerPath, res)
	}
}

// auditPlayerMSRIndex contrôle UNE player DB. Lecture via OpenReadForQuery
// (jamais OpenReadOnly forcé) : si un sync tient déjà le fichier en RW dans ce
// process, le handle en cache est réutilisé au lieu d'échouer sur
// « different configuration » (ADR 0013/0016).
func (s *HealthScheduler) auditPlayerMSRIndex(ctx context.Context, slug, gamertag, playerPath string, res *DataHealthCheckResult) {
	sqlDB, release, err := duckdb.OpenReadForQuery(playerPath)
	if err != nil {
		slog.WarnContext(ctx, "data_health: player DB inouvrable pour contrôle index match_skill_rank (non mesuré)",
			"titleSlug", slug, "gamertag", gamertag, "err", err)
		res.MSRIndexPlayersUnmeasured++
		return
	}
	probeCtx, cancel := context.WithTimeout(ctx, msrIndexProbeTimeout)
	rep, probeErr := scanMSRIndexDesync(probeCtx, sqlDB, msrIndexSampleKeys)
	cancel()
	release()

	if errors.Is(probeErr, errMSRTableAbsent) {
		slog.DebugContext(ctx, "data_health: match_skill_rank absente — skip contrôle index",
			"titleSlug", slug, "gamertag", gamertag)
		return
	}
	if probeErr != nil {
		slog.WarnContext(ctx, "data_health: contrôle index match_skill_rank échoué",
			"titleSlug", slug, "gamertag", gamertag, "err", probeErr)
		res.ProbeErrors++
		res.MSRIndexPlayersUnmeasured++
		return
	}

	res.MSRIndexPlayersScanned++
	if rep.KeysDiverging == 0 {
		return
	}
	res.MSRIndexDesyncPlayers++
	res.MSRIndexDesyncKeys += rep.KeysDiverging
	res.MSRIndexRowsMissing += rep.RowsMissing
	// ALERTE, PAS DE RÉPARATION (cf. en-tête) : la remédiation est manuelle et
	// tracée. Le chemin de la base est nommé parce que l'outil le demande.
	slog.ErrorContext(ctx, "data_health: index match_skill_rank DÉSYNCHRONISÉ (lookup indexé ≠ scan) — réparation MANUELLE requise",
		"titleSlug", slug, "gamertag", gamertag,
		"keys_sampled", rep.KeysSampled,
		"keys_diverging", rep.KeysDiverging,
		"rows_missing", rep.RowsMissing,
		"axes_diverging", rep.AxesDiverging,
		"remediation", "go run ./cmd/repair_msr_index -db "+playerPath+" -repair (serveur arrêté)")
}

// publishMSRIndexGaugeIfComplete republie la jauge expvar — SEULEMENT si le
// contrôle a été COMPLET. Un contrôle partiel (player DB tenue RW, sonde en
// échec) sous-compte : republier éteindrait le signal à tort (« unmeasured ≠
// sain », même invariant que la jauge LUSR).
func publishMSRIndexGaugeIfComplete(ctx context.Context, res *DataHealthCheckResult) {
	if res.MSRIndexPlayersUnmeasured == 0 {
		observability.SetInt(msrIndexGauge, int64(res.MSRIndexDesyncKeys))
		return
	}
	slog.WarnContext(ctx, "data_health: jauge index match_skill_rank non mise à jour (contrôle partiel)",
		"msr_index_players_unmeasured", res.MSRIndexPlayersUnmeasured,
		"msr_index_desync_keys_scanned", res.MSRIndexDesyncKeys)
}
