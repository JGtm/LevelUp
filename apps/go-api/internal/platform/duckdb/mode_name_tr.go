// Package duckdb — mode_name_tr.go : SOURCE UNIQUE du SQL de LECTURE sur
// metadata.mode_name_tr (traductions FR des modes de jeu normalisés).
//
// Contexte (2026-07-25, règle CLAUDE.md n°6 « ≤ 2 copies d'un même pattern → à
// la 3e, centraliser ET poser un garde-rail ») : la requête de résolution FR des
// modes était recopiée dans SIX repos de cette couche —
// career_repo_highlights.go, explorer_repo.go, match_history_fr_translations.go,
// squad_repo_mapstats.go (4 copies au littéral près) plus
// home_repo_translations.go et media_repo_translations.go (mêmes clauses, mise en
// forme différente). Chaque copie portait sa PROPRE gestion du NULL/vide, de la
// table absente et du handle FATAL-invalidated : toute divergence se paie en
// libellés FR incohérents d'une page à l'autre pour la MÊME donnée (le bug
// « CTF » + « Capture du drapeau » côte à côte, thought_log 2026-05-09).
//
// Le point d'entrée EXPORTÉ de la couche reste SquadRepo.LoadModeTranslationsFR
// (port.SquadRepository / port.CareerRepository) — les callers hors package et
// l'injection prestige (WithModeTranslatorFR) passent par lui, inchangés.
//
// Garde-rail : no_mode_name_tr_literal_test.go interdit tout accès SQL
// `FROM mode_name_tr` ailleurs que dans ce fichier, sous internal/platform/duckdb.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// selectModeNameTrFR — requête canonique de résolution FR des modes. `%s` = liste
// de placeholders. `lang = 'fr'` est le seul contrat servi par cette couche :
// l'affichage EN utilise le mode_en canonique tel quel, sans lookup.
const selectModeNameTrFR = `SELECT mode_en, name FROM mode_name_tr WHERE lang = 'fr' AND mode_en IN (%s)`

// selectDistinctModeEN — inventaire des modes canoniques connus (clés mode_en),
// toutes langues confondues.
const selectDistinctModeEN = `SELECT DISTINCT mode_en FROM mode_name_tr`

// modeNameTrQueryTimeout — plafond des lectures best-effort de mode_name_tr : la
// traduction d'un libellé ne doit jamais retenir une page (dégradation = EN).
const modeNameTrQueryTimeout = 3 * time.Second

// queryModeNameTrFR retourne le mapping mode_en → nom FR pour les modes EN
// NORMALISÉS donnés (via analysis.NormalizeModeLabel côté caller). Une clé
// absente du résultat = pas de traduction connue : le caller GARDE l'EN, il ne
// vide jamais le libellé. Les noms FR vides/blancs sont écartés.
//
// Contrat d'erreur : (nil, nil) si metadata ou la table est absente (donnée non
// encore migrée — dégradation normale) ; l'erreur SQL réelle est REMONTÉE, le
// caller décide de sa propre dégradation (log + repli EN pour les best-effort).
//
// QueryRecovered et non Query : auto-réparation si le handle metadata a été
// FATAL-invalidated (bug ART) — sinon la traduction FR retombe en EN jusqu'au
// prochain restart du process. C'était déjà le comportement de home_repo et de
// match_history ; les quatre autres copies l'acquièrent en convergeant ici.
func queryModeNameTrFR(ctx context.Context, meta *DB, modeENs []string) (map[string]string, error) {
	if meta == nil || len(modeENs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENs)), ",")
	args := make([]any, len(modeENs))
	for i, n := range modeENs {
		args[i] = n
	}
	rows, err := meta.QueryRecovered(ctx, fmt.Sprintf(selectModeNameTrFR, placeholders), args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(modeENs))
	for rows.Next() {
		var modeEN, nameFR string
		if scanErr := rows.Scan(&modeEN, &nameFR); scanErr != nil {
			continue
		}
		if strings.TrimSpace(nameFR) != "" {
			out[modeEN] = nameFR
		}
	}
	return out, rows.Err()
}

// loadModeNamesFRForKeys — variante BEST-EFFORT de queryModeNameTrFR : plafonne
// la requête (modeNameTrQueryTimeout) et absorbe l'erreur en la loguant, pour les
// chemins d'enrichissement qui ne doivent jamais échouer (match_history,
// filters, media). Retourne nil si rien n'est résolu.
func loadModeNamesFRForKeys(ctx context.Context, meta *DB, enKeys []string) map[string]string {
	if meta == nil || len(enKeys) == 0 {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, modeNameTrQueryTimeout)
	defer cancel()
	out, err := queryModeNameTrFR(ctx2, meta, enKeys)
	if err != nil {
		slog.WarnContext(ctx, "fr_translations: loadModeNamesFRForKeys failed", "err", err)
		return nil
	}
	return out
}

// loadKnownModesEN charge la liste DISTINCTE des mode_en de mode_name_tr — les
// modes canoniques connus — pour rattraper les variantes non canoniques
// ("Legacy Slayer BR" → "Slayer") via analysis.ExtractKnownMode. Best-effort :
// nil si meta absent ou table absente.
func loadKnownModesEN(ctx context.Context, meta *DB) []string {
	if meta == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, modeNameTrQueryTimeout)
	defer cancel()
	rows, err := meta.QueryRecovered(ctx2, selectDistinctModeEN)
	if err != nil {
		if !isTableNotFoundErr(err) {
			slog.WarnContext(ctx, "fr_translations: loadKnownModesEN failed", "err", err)
		}
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if rows.Scan(&m) == nil && strings.TrimSpace(m) != "" {
			out = append(out, m)
		}
	}
	return out
}
