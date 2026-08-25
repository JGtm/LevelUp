// Package sync — transforms_helpers.go : helpers privés pour la transformation
// JSON API Halo → structs DB.
//
// Ce fichier contient les expressions régulières, les parseurs et les accésseurs
// primitifs utilisés par transforms.go. Toutes les fonctions sont stateless.
package sync

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ──────────────────────────────────────────────────────────────────────────────
// Expressions régulières
// ──────────────────────────────────────────────────────────────────────────────

var (
	xuidRE = regexp.MustCompile(`xuid\((\d+)\)`)
	ptRE   = regexp.MustCompile(`(?i)PT(?:(\d+)H)?(?:(\d+)M)?(?:([\d.]+)S)?`)
)

// ──────────────────────────────────────────────────────────────────────────────
// Helpers privés — extraction JSON
// ──────────────────────────────────────────────────────────────────────────────

// extractXUID extrait le XUID depuis un PlayerId Halo Infinite.
// Humains : "xuid(1234567890)" → "1234567890"
// Bots    : "bid(3.0)" → "bid(3.0)" / "bid(3.0" (API sans paren) → "bid(3.0)"
func extractXUID(playerID string) string {
	if m := xuidRE.FindStringSubmatch(playerID); len(m) == 2 {
		return m[1]
	}
	if strings.HasPrefix(playerID, "bid(") {
		if !strings.HasSuffix(playerID, ")") {
			return playerID + ")"
		}
		return playerID
	}
	return ""
}

// extractAssetID extrait AssetId depuis un sous-objet JSON (ex: "Playlist").
func extractAssetID(matchInfo map[string]any, key string) string {
	obj, _ := matchInfo[key].(map[string]any)
	if obj == nil {
		return ""
	}
	id, _ := obj["AssetId"].(string)
	return id
}

// extractPublicName extrait PublicName depuis un sous-objet JSON.
func extractPublicName(matchInfo map[string]any, key string) string {
	obj, _ := matchInfo[key].(map[string]any)
	if obj == nil {
		return ""
	}
	name, _ := obj["PublicName"].(string)
	return name
}

// extractVersionID extrait VersionId depuis un sous-objet JSON (ex: "Playlist").
// Phase B du plan catalogue : permet de tracker les versions d'assets par match
// pour détecter les rotations Ranked / mises à jour de weights / changements d'assets UGC.
func extractVersionID(matchInfo map[string]any, key string) string {
	obj, _ := matchInfo[key].(map[string]any)
	if obj == nil {
		return ""
	}
	id, _ := obj["VersionId"].(string)
	return id
}

// findCoreStats retourne le dict CoreStats du premier PlayerTeamStats du joueur.
func findCoreStats(player map[string]any) map[string]any {
	pts, _ := player["PlayerTeamStats"].([]any)
	for _, ts := range pts {
		teamStats, ok := ts.(map[string]any)
		if !ok {
			continue
		}
		stats, _ := teamStats["Stats"].(map[string]any)
		if stats == nil {
			continue
		}
		core, _ := stats["CoreStats"].(map[string]any)
		if core != nil {
			return core
		}
	}
	return nil
}

// isRankedPlaylist détermine si le match est un match classé.
// Portage de _is_ranked_playlist() Python.
func isRankedPlaylist(matchInfo map[string]any) bool {
	playlist, _ := matchInfo["Playlist"].(map[string]any)
	if playlist == nil {
		return false
	}
	name, _ := playlist["PublicName"].(string)
	if strings.Contains(strings.ToLower(name), PerfChainRanked) {
		return true
	}
	if tags, ok := playlist["Tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok && strings.ToLower(s) == PerfChainRanked {
				return true
			}
		}
	}
	return false
}

// isFirefightMatch détermine si le match est un mode Firefight/PvE.
// Portage de _is_firefight_match() Python.
func isFirefightMatch(matchInfo map[string]any) bool {
	// GameVariantCategory (22 = Firefight Arcade, 32 = Firefight Heroic)
	if cat, ok := matchInfo["GameVariantCategory"].(float64); ok {
		firefightCats := map[int]bool{22: true, 32: true, 40: true, 41: true, 42: true}
		if firefightCats[int(cat)] {
			return true
		}
	}
	if gv, ok := matchInfo["UgcGameVariant"].(map[string]any); ok {
		name, _ := gv["PublicName"].(string)
		if strings.Contains(strings.ToLower(name), "firefight") {
			return true
		}
	}
	return false
}

// determineModeCategory déduit la catégorie custom depuis pair_name.
// Portage simplifié de infer_custom_category_from_pair_name() Python.
func determineModeCategory(pairName string) string {
	lower := strings.ToLower(pairName)
	switch {
	case strings.Contains(lower, PerfChainRanked):
		return modeCategoryRanked
	case strings.Contains(lower, PerfChainFirefight):
		return modeCategoryFirefight
	case strings.Contains(lower, "btb") || strings.Contains(lower, "big team") || strings.Contains(lower, "big-team"):
		return modeCategoryBTB
	case strings.Contains(lower, "fiesta"):
		return modeCategoryFiesta
	case strings.Contains(lower, "assassin"):
		return modeCategoryAssassin
	default:
		return modeCategoryOther
	}
}

// ExtractTeamScoresByID extrait les scores de team_0 et team_1 depuis le payload
// GetMatchStats, en les indexant par `Teams[].TeamId` (jamais par position dans le
// tableau : l'ordre y suit le rang, pas l'identifiant de camp).
//
// Portage de _extract_team_scores_by_id() Python.
//
// SOURCE UNIQUE DU SCORE D'ÉQUIPE, ET C'EST POUR ÇA QU'ELLE EST EXPORTÉE. Le champ lu,
// `Teams[].Stats.CoreStats.Score`, est le score AFFICHÉ par le jeu — mesuré sur les 1 934
// matchs du corpus le 2026-08-24 (rapport `.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md`).
// Le bloc voisin `Stats.ZonesStats.StrongholdScoringTicks` porte, lui, le compteur brut du
// mode : 69 lignes de `match_registry` le contiennent par erreur, héritage d'une période où
// l'API 343 servait ce compteur dans `CoreStats.Score` (corrigée entre avril et mai 2026).
//
// Toute relecture de ce score — sync ou backfill — passe par ICI. Une seconde
// implémentation re-divergerait le jour où l'un des deux appelants suivrait le mauvais
// champ, et c'est exactement le défaut que le backfill répare.
// Appelants : `ExtractRegistry` (`transforms.go:128`, la sync) et `cmd/backfill-team-scores`.
//
// Retourne (nil, nil) si aucun bloc `Teams` exploitable ; un pointeur nil par camp absent
// (FFA, équipes au-delà de 0/1) — l'appelant ne doit JAMAIS substituer un zéro à un nil.
func ExtractTeamScoresByID(matchJSON map[string]any) (*int, *int) {
	teams, _ := matchJSON["Teams"].([]any)
	scores := map[int]int{}
	for _, t := range teams {
		team, ok := t.(map[string]any)
		if !ok {
			continue
		}
		id := intFrom(team, "TeamId")
		stats, _ := team["Stats"].(map[string]any)
		if stats == nil {
			continue
		}
		core, _ := stats["CoreStats"].(map[string]any)
		if core == nil {
			continue
		}
		score := intFrom(core, "Score")
		scores[id] = score
	}
	if len(scores) == 0 {
		return nil, nil
	}
	t0, ok0 := scores[0]
	t1, ok1 := scores[1]
	var p0, p1 *int
	if ok0 {
		p0 = &t0
	}
	if ok1 {
		p1 = &t1
	}
	return p0, p1
}

// parsePTDuration convertit une durée ISO 8601 "PT1H2M3.456S" en secondes.
// Portage de _parse_duration_to_seconds() Python.
func parsePTDuration(s string) *int {
	if s == "" {
		return nil
	}
	m := ptRE.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	total := 0
	if m[1] != "" {
		h, _ := strconv.Atoi(m[1])
		total += h * 3600
	}
	if m[2] != "" {
		min, _ := strconv.Atoi(m[2])
		total += min * 60
	}
	if m[3] != "" {
		f, _ := strconv.ParseFloat(m[3], 64)
		total += int(f)
	}
	return &total
}

// parseISO parse une date ISO 8601 UTC.
func parseISO(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999Z",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parseISO: impossible de parser %q", s)
}

// ──────────────────────────────────────────────────────────────────────────────
// Helpers primitifs
// ──────────────────────────────────────────────────────────────────────────────

func asString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func strPtrNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// coalesceStrPtr conserve un CONTRAT DISTINCT de service.coalesceStr : il retourne
// un *string (préserve nil vs "" pour l'aval du pipeline de transforms), là où
// coalesceStr aplatit vers string. Volontairement non fusionné (F4, revue 2026-07-17).
func coalesceStrPtr(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}

func intPtrFrom(m map[string]any, key string) *int {
	v, ok := m[key].(float64)
	if !ok {
		return nil
	}
	n := int(v)
	return &n
}

// jsonBoolPtr lit une clé booléenne d'un objet JSON décodé. Retourne nil si la
// clé est absente, NULL côté JSON, ou pas un booléen.
func jsonBoolPtr(m map[string]any, key string) *bool {
	v, ok := m[key].(bool)
	if !ok {
		return nil
	}
	return &v
}

func floatPtrFrom(m map[string]any, key string) *float64 {
	v, ok := m[key].(float64)
	if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
		return nil
	}
	return &v
}

func intFrom(m map[string]any, key string) int {
	v, _ := m[key].(float64)
	return int(v)
}

func int64From(m map[string]any, key string) int64 {
	v, _ := m[key].(float64)
	return int64(v)
}

// ErrMissingField crée une erreur de champ manquant.
func ErrMissingField(field string) error {
	return fmt.Errorf("champ manquant: %s", field)
}
