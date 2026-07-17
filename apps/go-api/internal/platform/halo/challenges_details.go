package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
)

// Codes de langue Waypoint utilisés pour l'i18n des challenges et BP.
const (
	langFR = "fr-FR"
	langEN = "en-US"
)

type challengeDeckRaw struct {
	Expiration struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
	ActiveChallenges    []challengeDeckItemRaw `json:"ActiveChallenges"`
	CompletedChallenges []json.RawMessage      `json:"CompletedChallenges"`
}

type challengeDeckItemRaw struct {
	Path            string `json:"Path"`
	TrackingID      string `json:"TrackingId"`
	XPReward        *int   `json:"XPReward"`
	Threshold       *int   `json:"Threshold"`
	Progress        *int   `json:"Progress"`
	CurrentProgress *int   `json:"CurrentProgress"`
	CanReroll       *bool  `json:"CanReroll"`
	Expiration      struct {
		ISO8601Date string `json:"ISO8601Date"`
	} `json:"Expiration"`
}

type challengeDefinitionRaw struct {
	Title               any    `json:"Title"`
	Description         any    `json:"Description"`
	Category            string `json:"Category"`
	Difficulty          string `json:"Difficulty"`
	ThresholdForSuccess any    `json:"ThresholdForSuccess"`
	Reward              struct {
		SoftExperience int `json:"SoftExperience"`
	} `json:"Reward"`
	SecondaryReward struct {
		SoftExperience int `json:"SoftExperience"`
	} `json:"SecondaryReward"`
}

func (p *HaloProvider) buildActiveChallengeItems(ctx context.Context, tokens *domain.HaloTokens, decks []challengeDeckRaw) []domain.ChallengeItem {
	seen := make(map[string]struct{})
	items := make([]domain.ChallengeItem, 0)
	// Langue de résolution des titres/descriptions de défis = locale de requête
	// (header X-LevelUp-Locale → ctxkeys ; défaut FR hors requête HTTP, ex. watcher).
	// Auparavant figée FR → « les défis non plus » [traduits] sous UI EN.
	lang := normalizeChallengeLang(ctxkeys.Locale(ctx))

	for _, deck := range decks {
		for _, ch := range deck.ActiveChallenges {
			key := ch.Path
			if key == "" {
				key = ch.TrackingID
			}
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}

			var def *challengeDefinitionRaw
			if ch.Path != "" {
				loaded, err := p.fetchChallengeDefinition(ctx, tokens, ch.Path)
				if err != nil {
					slog.DebugContext(ctx, "halo_provider: challenge definition unavailable",
						"path", ch.Path, "err", err)
				} else {
					def = loaded
				}
			}

			items = append(items, p.buildChallengeItem(ctx, tokens, ch, def, lang))
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		left := challengeSortScore(items[i])
		right := challengeSortScore(items[j])
		if left != right {
			return left > right
		}
		leftCurrent := derefInt(items[i].ProgressCurrent)
		rightCurrent := derefInt(items[j].ProgressCurrent)
		if leftCurrent != rightCurrent {
			return leftCurrent > rightCurrent
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})

	return items
}

func (p *HaloProvider) buildChallengeItem(
	_ context.Context,
	_ *domain.HaloTokens,
	ch challengeDeckItemRaw,
	def *challengeDefinitionRaw,
	lang string,
) domain.ChallengeItem {
	current := resolveChallengeCurrentProgress(ch)
	target := resolveChallengeTarget(ch, def)
	xpReward := resolveChallengeXP(ch, def)
	title := fallbackChallengeTitle(ch.Path)
	var description *string
	var imageURL *string
	var progressPct *float64

	if def != nil {
		if localizedTitle := resolveChallengeLocalizedValue(def.Title, lang); localizedTitle != "" {
			title = localizedTitle
		}
		if localizedDescription := resolveChallengeLocalizedValue(def.Description, lang); localizedDescription != "" {
			description = stringPtr(localizedDescription)
		}
		imageURL = challengeBadgeAPIURL(ch.Path, def.Category, def.Difficulty, p.titleID())
	} else {
		imageURL = challengeBadgeAPIURL(ch.Path, "", "", p.titleID())
	}

	if current != nil && target != nil && *target > 0 {
		pct := float64(*current) / float64(*target) * 100.0
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		progressPct = &pct
	}

	item := domain.ChallengeItem{
		ChallengePath:   challengePathOrFallback(ch.Path, ch.TrackingID),
		Title:           title,
		Description:     description,
		ImageURL:        imageURL,
		ProgressCurrent: current,
		ProgressTarget:  target,
		ProgressPercent: progressPct,
		XPReward:        xpReward,
	}
	if ch.TrackingID != "" {
		item.TrackingID = stringPtr(ch.TrackingID)
	}
	return item
}

func (p *HaloProvider) fetchChallengeDefinition(ctx context.Context, _ *domain.HaloTokens, challengePath string) (*challengeDefinitionRaw, error) {
	trimmed := strings.TrimSpace(challengePath)
	if trimmed == "" || p.assetResolver == nil {
		return nil, nil
	}

	ref := assets.Ref{
		Kind:    assets.KindChallengeDefinition,
		TitleID: p.titleID(),
		ID:      trimmed,
	}
	resolved, err := p.assetResolver.Get(ctx, ref)
	if err != nil {
		slog.DebugContext(ctx, "halo_provider: challenge definition resolver miss",
			"path", trimmed, "err", err)
		return nil, err
	}
	jp, ok := resolved.Payload.(assets.JSONPayload)
	if !ok {
		return nil, fmt.Errorf("challenge definition unexpected payload type for %s", trimmed)
	}
	var def challengeDefinitionRaw
	if err := json.Unmarshal(jp.RawJSON, &def); err != nil {
		return nil, fmt.Errorf("challenge definition decode: %w", err)
	}
	return &def, nil
}

// challengeBadgeAPIURL construit l'URL relative de l'image de badge d'un défi.
// Retourne nil si aucun stem candidat n'est trouvé.
// La résolution locale/distante est gérée par le DefaultResolver (endpoint /assets/challenge-badge/).
func challengeBadgeAPIURL(challengePath, category, difficulty, titleID string) *string {
	stems := buildChallengeBadgeCandidates(challengePath, category, difficulty)
	if len(stems) == 0 {
		return nil
	}
	url := "/api/v1/assets/challenge-badge/" + titleID + "/" + stems[0]
	return &url
}

func resolveChallengeCurrentProgress(ch challengeDeckItemRaw) *int {
	if ch.Progress != nil {
		return intPtr(*ch.Progress)
	}
	if ch.CurrentProgress != nil {
		return intPtr(*ch.CurrentProgress)
	}
	return nil
}

func resolveChallengeTarget(ch challengeDeckItemRaw, def *challengeDefinitionRaw) *int {
	if ch.Threshold != nil {
		return intPtr(*ch.Threshold)
	}
	if def == nil {
		return nil
	}
	if value, ok := coerceChallengeInt(def.ThresholdForSuccess); ok {
		return intPtr(value)
	}
	return nil
}

func resolveChallengeXP(ch challengeDeckItemRaw, def *challengeDefinitionRaw) *int {
	if ch.XPReward != nil {
		return intPtr(*ch.XPReward)
	}
	if def == nil {
		return nil
	}
	if def.Reward.SoftExperience > 0 {
		return intPtr(def.Reward.SoftExperience)
	}
	if def.SecondaryReward.SoftExperience > 0 {
		return intPtr(def.SecondaryReward.SoftExperience)
	}
	return nil
}

func resolveChallengeLocalizedValue(data any, lang string) string {
	if value, ok := data.(string); ok {
		return strings.TrimSpace(value)
	}
	obj, ok := data.(map[string]any)
	if !ok {
		return ""
	}
	translations, _ := obj["translations"].(map[string]any)
	for _, candidate := range challengeLanguageCandidates(normalizeChallengeLang(lang)) {
		if raw, ok := translations[candidate]; ok {
			if text, ok := raw.(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}
	if fallback, ok := obj["value"].(string); ok {
		return strings.TrimSpace(fallback)
	}
	return ""
}

func normalizeChallengeLang(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "fr", "fr-fr":
		return langFR
	case "en", "en-us":
		return langEN
	default:
		return langFR
	}
}

func challengeLanguageCandidates(lang string) []string {
	if lang == langFR {
		return []string{langFR, "fr"}
	}
	if lang == langEN {
		return []string{langEN, "en-GB", "en"}
	}
	short := strings.Split(lang, "-")[0]
	return []string{lang, short}
}

//nolint:gocyclo // construction de candidats de badge : N branches d'heuristiques sur path/category/difficulty
func buildChallengeBadgeCandidates(challengePath, category, difficulty string) []string {
	normalizedPath := strings.ToLower(strings.ReplaceAll(challengePath, `\`, "/"))
	cat := slugifyChallengeToken(category)
	diff := slugifyChallengeToken(difficulty)
	weeklyFamily := inferWeeklyFamily(normalizedPath)
	seasonalPath := isSeasonalChallengePath(normalizedPath)
	candidates := make([]string, 0, 6)

	if strings.Contains(normalizedPath, "dailychallenges") && diff != "" {
		candidates = append(candidates, "daily-"+diff)
	}
	if strings.Contains(normalizedPath, "weeklychallenges") && weeklyFamily != "" && diff != "" {
		candidates = append(candidates, "weekly-"+weeklyFamily+"-"+diff)
	}
	if seasonalPath && diff != "" {
		candidates = append(candidates, "weekly-"+diff)
	}
	if strings.Contains(normalizedPath, "ultimate") || strings.Contains(normalizedPath, "capstone") {
		if diff == "" {
			diff = "mythic"
		}
		candidates = append(candidates, "capstone-"+diff)
	}
	if seasonalPath && diff != "" {
		for _, family := range []string{"action", "gametype", "weapon"} {
			candidates = append(candidates, "weekly-"+family+"-"+diff)
		}
	}
	if cat == "daily" && diff != "" {
		candidates = append(candidates, "daily-"+diff)
	}
	if cat == "seasonal" && diff != "" {
		candidates = append(candidates, "weekly-"+diff)
	}
	if cat == "weekly" && weeklyFamily != "" && diff != "" {
		candidates = append(candidates, "weekly-"+weeklyFamily+"-"+diff)
	}
	if cat == "ultimate" || cat == "capstone" {
		if diff == "" {
			diff = "mythic"
		}
		candidates = append(candidates, "capstone-"+diff)
	}
	if diff == "mythic" {
		candidates = append(candidates, "capstone-mythic")
	}
	if cat != "" && diff != "" {
		candidates = append(candidates, cat+"-"+diff)
	}
	return dedupeChallengeCandidates(candidates)
}

func dedupeChallengeCandidates(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func slugifyChallengeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer("_", "-", " ", "-")
	value = replacer.Replace(value)
	b := strings.Builder{}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			b.WriteRune(char)
		}
	}
	return strings.Trim(b.String(), "-")
}

func inferWeeklyFamily(normalizedPath string) string {
	const marker = "/weeklychallenges/"
	index := strings.Index(normalizedPath, marker)
	if index >= 0 {
		remainder := normalizedPath[index+len(marker):]
		segment, _, _ := strings.Cut(remainder, "/")
		if family := slugifyChallengeToken(segment); family != "" {
			return family
		}
	}

	for _, token := range []string{"action", "gametype", "weapon"} {
		if strings.Contains(normalizedPath, "/"+token+"/") {
			return token
		}
	}
	return ""
}

func isSeasonalChallengePath(normalizedPath string) bool {
	if normalizedPath == "" {
		return false
	}
	seasonalTokens := []string{
		"winterchallenges",
		"seasonalchallenges",
		"eventchallenges",
		"operationchallenges",
		"fracturechallenges",
	}
	for _, token := range seasonalTokens {
		if strings.Contains(normalizedPath, token) {
			return true
		}
	}
	return strings.Contains(normalizedPath, "/s") && strings.Contains(normalizedPath, "challenges/")
}

func challengeSortScore(item domain.ChallengeItem) float64 {
	if item.ProgressPercent != nil {
		return *item.ProgressPercent
	}
	if item.ProgressCurrent != nil && *item.ProgressCurrent > 0 {
		return 0.001 + float64(*item.ProgressCurrent)/10000.0
	}
	return 0
}

func fallbackChallengeTitle(challengePath string) string {
	base := path.Base(challengePath)
	base = strings.TrimSuffix(base, path.Ext(base))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.TrimSpace(base)
	if base == "" || strings.EqualFold(base, ".") {
		return "Défi actif"
	}
	return base
}

func challengePathOrFallback(challengePath, trackingID string) string {
	if strings.TrimSpace(challengePath) != "" {
		return strings.TrimSpace(challengePath)
	}
	if trackingID != "" {
		return "Challenges/Tracking/" + trackingID
	}
	return "Challenges/Unknown"
}

func coerceChallengeInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case map[string]any:
		//nolint:goconst // "Threshold"/"Value"/"Count" sont des clés JSON Halo (PascalCase + camelCase variantes), pas des constantes métier à factoriser.
		for _, key := range []string{"value", "Value", "threshold", "Threshold", "count", "Count"} {
			if resolved, ok := coerceChallengeInt(v[key]); ok {
				return resolved, true
			}
		}
	}
	return 0, false
}

func intPtr(value int) *int { return &value }

func stringPtr(value string) *string { return &value }

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
