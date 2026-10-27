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
// Le point d'entrée EXPORTÉ de la couche reste SquadRepo.LoadModeTranslations
// (port.SquadRepository / port.CareerRepository) — les callers hors package et
// l'injection prestige (WithModeTranslator) passent par lui, inchangés.
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

	"levelup/go-api/internal/i18n"
)

// selectModeNameTr — requête canonique de résolution des noms de mode dans une langue (premier
// argument : le code de langue, `lang`). `%s` = liste de placeholders. L'affichage EN utilise le
// mode_en canonique tel quel, sans lookup.
const selectModeNameTr = `SELECT mode_en, name FROM mode_name_tr WHERE lang = ? AND mode_en IN (%s)`

// selectDistinctModeEN — inventaire des modes canoniques connus (clés mode_en),
// toutes langues confondues.
const selectDistinctModeEN = `SELECT DISTINCT mode_en FROM mode_name_tr`

// modeNameTrQueryTimeout — plafond des lectures best-effort de mode_name_tr : la
// traduction d'un libellé ne doit jamais retenir une page (dégradation = EN).
const modeNameTrQueryTimeout = 3 * time.Second

// queryModeNameTr retourne le mapping mode_en → nom dans la langue loc pour les modes EN
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
func queryModeNameTr(ctx context.Context, meta *DB, loc i18n.Locale, modeENs []string) (map[string]string, error) {
	if meta == nil || len(modeENs) == 0 {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(modeENs)), ",")
	args := make([]any, 0, len(modeENs)+1)
	args = append(args, string(loc))
	for _, n := range modeENs {
		args = append(args, n)
	}
	rows, err := meta.QueryRecovered(ctx, fmt.Sprintf(selectModeNameTr, placeholders), args...)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string, len(modeENs))
	for rows.Next() {
		var modeEN, name string
		if scanErr := rows.Scan(&modeEN, &name); scanErr != nil {
			continue
		}
		if strings.TrimSpace(name) != "" {
			out[modeEN] = name
		}
	}
	return out, rows.Err()
}

// loadModeNamesForKeys — variante BEST-EFFORT de queryModeNameTr : plafonne
// la requête (modeNameTrQueryTimeout) et absorbe l'erreur en la loguant et en la
// consignant (noteDegraded : un chargement mis en cache qui l'appelle n'est pas
// caché), pour les chemins d'enrichissement qui ne doivent jamais échouer
// (match_history, filters, media). Retourne nil si rien n'est résolu.
func loadModeNamesForKeys(ctx context.Context, meta *DB, loc i18n.Locale, enKeys []string) map[string]string {
	if meta == nil || len(enKeys) == 0 {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, modeNameTrQueryTimeout)
	defer cancel()
	out, err := queryModeNameTr(ctx2, meta, loc, enKeys)
	if err != nil {
		slog.WarnContext(ctx, "fr_translations: loadModeNamesFRForKeys failed", "err", err)
		noteDegraded(ctx, "mode_names_fr")
		return nil
	}
	return out
}

// loadKnownModes charge la liste DISTINCTE des mode_en de mode_name_tr — les
// modes canoniques connus — pour rattraper les variantes non canoniques
// ("Legacy Slayer BR" → "Slayer") via analysis.ExtractKnownMode. Best-effort :
// nil si meta absent ou table absente.
func loadKnownModes(ctx context.Context, meta *DB) []string {
	if meta == nil {
		return nil
	}
	ctx2, cancel := context.WithTimeout(ctx, modeNameTrQueryTimeout)
	defer cancel()
	rows, err := meta.QueryRecovered(ctx2, selectDistinctModeEN)
	if err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			"fr_translations: loadKnownModes failed, variantes de mode non rattrapées", "err", err)
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			slog.Log(ctx, degradedLookupLevel(ctx, err), "fr_translations: loadKnownModes ligne illisible ignorée", "err", err)
			continue
		}
		if strings.TrimSpace(m) != "" {
			out = append(out, m)
		}
	}
	if err := rows.Err(); err != nil {
		slog.Log(ctx, degradedLookupLevel(ctx, err),
			"fr_translations: loadKnownModes itération interrompue, liste partielle", "err", err, "modes", len(out))
	}
	return out
}
