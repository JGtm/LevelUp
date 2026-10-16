// Package service — openspartan_import_scope.go : ce qu'un import OpenSpartan a le
// droit d'écrire (décision utilisateur D-B du 2026-10-15).
//
// L'import est ouvert à tout compte lié, mais il n'écrit que les matchs auxquels le
// joueur qui importe (xuid de sa session) a participé, et aucune donnée propre à un
// autre joueur :
//   - un match dont l'identifiant n'a pas la forme Halo (title.IsValidMatchID) est
//     écarté : il servirait plus tard de composant de chemin ;
//   - un match sans l'importeur parmi ses participants est écarté, avec ses
//     événements marquants ;
//   - l'alias de l'importeur est mis à jour (UpsertXUIDAlias) ; celui d'un autre
//     joueur n'est que PROPOSÉ (INSERT OR IGNORE, par le batch du match), et
//     seulement pour un participant d'un match écrit : un nom déjà connu n'est
//     jamais réécrit.
//
// Risque résiduel accepté par D-B : l'importeur peut falsifier un match auquel il a
// participé (y compris les lignes des autres participants) et devancer sa version
// réelle, que l'écriture INSERT-only ne remplacera pas.
package service

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/openspartan"
	"levelup/go-api/internal/openspartan/mapper"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync"
)

// importScope porte le périmètre d'un import : l'importeur, les options de
// projection, et les matchs retenus (ceux de l'importeur, à identifiant valide).
type importScope struct {
	owner   string
	mapOpts mapper.MapOptions
	alias   func(xuid string) string
	dryRun  bool
	matches map[string]bool
}

func newImportScope(owner string, mapOpts mapper.MapOptions, dryRun bool) *importScope {
	alias := mapOpts.AliasResolver
	if alias == nil {
		alias = func(string) string { return "" }
	}
	return &importScope{owner: owner, mapOpts: mapOpts, alias: alias, dryRun: dryRun, matches: map[string]bool{}}
}

// admit dit si le match entre dans le périmètre de l'import, et le retient alors
// (ses événements marquants suivront). Un refus est compté et journalisé.
func (sc *importScope) admit(ctx context.Context, pm *openspartan.ParsedMatch, mm *mapper.MappedMatch, result *ImportResult) bool {
	for _, p := range mm.Participants {
		if p.XUID == sc.owner {
			sc.matches[pm.MatchID] = true
			return true
		}
	}
	result.SkippedForeignMatches++
	slog.WarnContext(ctx, "openspartan_import: match sans l'importeur écarté",
		"match_id", pm.MatchID, "owner_xuid", sc.owner)
	return false
}

// thirdPartyAliases rend les alias PROPOSÉS pour les autres participants humains
// d'un match retenu (INSERT OR IGNORE via le batch : jamais de réécriture).
func (sc *importScope) thirdPartyAliases(parts []mapper.MatchParticipantRow) []persist.XUIDAliasInsert {
	now := time.Now().UTC()
	var out []persist.XUIDAliasInsert
	for _, p := range parts {
		if p.XUID == "" || p.XUID == sc.owner || analysis.IsBot(p.XUID) {
			continue
		}
		if gt := sc.alias(p.XUID); gt != "" {
			out = append(out, persist.XUIDAliasInsert{XUID: p.XUID, Gamertag: gt, LastSeen: now})
		}
	}
	return out
}

// rejectMatchID écarte un match à l'identifiant malformé (compté en erreur d'étape).
func rejectMatchID(ctx context.Context, matchID string, result *ImportResult) bool {
	if title.IsValidMatchID(matchID) {
		return false
	}
	result.Errors = append(result.Errors, ImportError{MatchID: matchID, Stage: "invalid_match_id",
		Err: "match id is not a Halo match id"})
	slog.WarnContext(ctx, "openspartan_import: identifiant de match invalide écarté", "match_id", matchID)
	return true
}

// importOwnerAlias met à jour l'alias de l'importeur seul ; ceux des autres joueurs
// sont proposés par le batch de chaque match retenu (thirdPartyAliases).
func (s *OpenSpartanImportService) importOwnerAlias(
	ctx context.Context,
	sharedDB *sql.DB,
	r *openspartan.Reader,
	scope *importScope,
	result *ImportResult,
) {
	rows, err := r.LoadXuidAliases(ctx)
	if err != nil {
		result.Errors = append(result.Errors, ImportError{Stage: "load_aliases_for_write", Err: err.Error()})
		return
	}
	for _, a := range rows {
		if a.XUID != scope.owner {
			continue
		}
		if scope.dryRun {
			result.InsertedAliases++
			return
		}
		if err := sync.UpsertXUIDAlias(ctx, sharedDB, a.XUID, a.Gamertag); err != nil {
			result.Errors = append(result.Errors, ImportError{Stage: "upsert_alias", Err: err.Error()})
			return
		}
		result.InsertedAliases++
		return
	}
}
