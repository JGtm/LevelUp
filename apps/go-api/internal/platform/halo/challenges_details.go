package halo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
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
	lang := "fr-FR"

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
	ctx context.Context,
	tokens *domain.HaloTokens,
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
		imageURL = challengeBadgeAPIURL(ch.Path, def.Category, def.Difficulty)
	} else {
		imageURL = challengeBadgeAPIURL(ch.Path, "", "")
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

func (p *HaloProvider) fetchChallengeDefinition(ctx context.Context, tokens *domain.HaloTokens, challengePath string) (*challengeDefinitionRaw, error) {
	trimmed := strings.TrimSpace(challengePath)
	if trimmed == "" {
		return nil, nil
	}

	// Branche P4/P5 : déléguer au resolver unifié.
	if p.assetResolver != nil {
		ref := assets.Ref{
			Kind:    assets.KindChallengeDefinition,
			TitleID: "halo_infinite",
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

	// Branche legacy : metadata.duckdb direct.
	if cached, err := p.loadChallengeDefinitionFromMetadata(ctx, trimmed); err == nil && cached != nil {
		slog.DebugContext(ctx, "halo_provider: challenge definition served from metadata cache",
			"path", trimmed)
		return cached, nil
	} else if err != nil {
		slog.DebugContext(ctx, "halo_provider: challenge definition metadata cache read failed",
			"path", trimmed, "err", err)
	}
	base := p.gameCMSBaseURL
	if base == "" {
		base = defaultGameCMSHost
	}
	url := fmt.Sprintf("%s/hi/Progression/file/%s", strings.TrimRight(base, "/"), strings.TrimLeft(trimmed, "/"))
	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		return nil, err
	}
	var def challengeDefinitionRaw
	if err := json.Unmarshal(body, &def); err != nil {
		return nil, fmt.Errorf("challenge definition decode: %w", err)
	}
	if err := p.storeChallengeDefinitionInMetadata(ctx, trimmed, body, &def); err != nil {
		slog.DebugContext(ctx, "halo_provider: challenge definition metadata cache write failed",
			"path", trimmed, "err", err)
	}
	return &def, nil
}

// challengeBadgeAPIURL construit l'URL relative de l'image de badge d'un défi.
// Retourne nil si aucun stem candidat n'est trouvé.
// La résolution locale/distante est gérée par le DefaultResolver (endpoint /assets/challenge-badge/).
func challengeBadgeAPIURL(challengePath, category, difficulty string) *string {
	stems := buildChallengeBadgeCandidates(challengePath, category, difficulty)
	if len(stems) == 0 {
		return nil
	}
	url := "/api/v1/assets/challenge-badge/halo_infinite/" + stems[0]
	return &url
}

func (p *HaloProvider) loadChallengeDefinitionFromMetadata(
	ctx context.Context,
	challengePath string,
) (*challengeDefinitionRaw, error) {
	metaPath := strings.TrimSpace(p.challengeMetaPath)
	if metaPath == "" {
		return nil, nil
	}
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	row := db.QueryRow(ctx, `
		SELECT d.category,
		       d.difficulty,
		       d.threshold_for_success,
		       d.reward_xp,
		       d.secondary_reward_xp,
		       COALESCE(t_fr.title, t_en.title) AS title,
		       COALESCE(t_fr.description, t_en.description) AS description
		FROM challenge_definitions d
		LEFT JOIN challenge_translations t_fr
		       ON t_fr.challenge_path = d.challenge_path
		      AND t_fr.content_hash = d.content_hash
		      AND t_fr.lang = 'fr-FR'
		LEFT JOIN challenge_translations t_en
		       ON t_en.challenge_path = d.challenge_path
		      AND t_en.content_hash = d.content_hash
		      AND t_en.lang = 'en-US'
		WHERE d.challenge_path = ? AND d.is_current = TRUE
		ORDER BY d.last_seen_at DESC
		LIMIT 1`, challengePath)

	var category sql.NullString
	var difficulty sql.NullString
	var threshold sql.NullInt64
	var rewardXP sql.NullInt64
	var secondaryXP sql.NullInt64
	var title sql.NullString
	var description sql.NullString
	if err := row.Scan(&category, &difficulty, &threshold, &rewardXP, &secondaryXP, &title, &description); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	def := &challengeDefinitionRaw{
		Category:    category.String,
		Difficulty:  difficulty.String,
		Title:       title.String,
		Description: description.String,
	}
	if threshold.Valid {
		def.ThresholdForSuccess = int(threshold.Int64)
	}
	if rewardXP.Valid {
		def.Reward.SoftExperience = int(rewardXP.Int64)
	}
	if secondaryXP.Valid {
		def.SecondaryReward.SoftExperience = int(secondaryXP.Int64)
	}
	return def, nil
}

func (p *HaloProvider) storeChallengeDefinitionInMetadata(
	ctx context.Context,
	challengePath string,
	body []byte,
	def *challengeDefinitionRaw,
) error {
	metaPath := strings.TrimSpace(p.challengeMetaPath)
	if metaPath == "" || def == nil || len(body) == 0 {
		return nil
	}
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		return err
	}
	defer db.Close()

	contentHash := challengeDefinitionContentHash(body)
	threshold, _ := coerceChallengeInt(def.ThresholdForSuccess)
	now := time.Now()

	if _, err := db.Exec(ctx, `
		UPDATE challenge_definitions
		SET is_current = FALSE,
		    last_seen_at = ?
		WHERE challenge_path = ?
		  AND content_hash <> ?
		  AND is_current = TRUE`, now, challengePath, contentHash); err != nil {
		return err
	}

	if _, err := db.Exec(ctx, `
		INSERT INTO challenge_definitions
			(challenge_path, content_hash, category, difficulty, threshold_for_success,
			 reward_xp, secondary_reward_xp, first_seen_at, last_seen_at, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE)
		ON CONFLICT (challenge_path, content_hash) DO UPDATE SET
			category = excluded.category,
			difficulty = excluded.difficulty,
			threshold_for_success = excluded.threshold_for_success,
			reward_xp = excluded.reward_xp,
			secondary_reward_xp = excluded.secondary_reward_xp,
			last_seen_at = excluded.last_seen_at,
			is_current = TRUE`,
		challengePath,
		contentHash,
		nullableChallengeString(def.Category),
		nullableChallengeString(def.Difficulty),
		nullableChallengeInt(threshold),
		nullableChallengeInt(def.Reward.SoftExperience),
		nullableChallengeInt(def.SecondaryReward.SoftExperience),
		now,
		now,
	); err != nil {
		return err
	}

	titleTranslations := collectChallengeTranslations(def.Title)
	descriptionTranslations := collectChallengeTranslations(def.Description)
	langs := make(map[string]struct{}, len(titleTranslations)+len(descriptionTranslations))
	for lang := range titleTranslations {
		langs[lang] = struct{}{}
	}
	for lang := range descriptionTranslations {
		langs[lang] = struct{}{}
	}
	for lang := range langs {
		if _, err := db.Exec(ctx, `
			INSERT INTO challenge_translations
				(challenge_path, content_hash, lang, title, description, first_seen_at, last_seen_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (challenge_path, content_hash, lang) DO UPDATE SET
				title = excluded.title,
				description = excluded.description,
				last_seen_at = excluded.last_seen_at`,
			challengePath,
			contentHash,
			lang,
			nullableChallengeString(titleTranslations[lang]),
			nullableChallengeString(descriptionTranslations[lang]),
			now,
			now,
		); err != nil {
			return err
		}
	}

	return nil
}

func challengeDefinitionContentHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

func collectChallengeTranslations(raw any) map[string]string {
	translations := make(map[string]string)
	switch value := raw.(type) {
	case string:
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			translations["en-US"] = trimmed
		}
	case map[string]any:
		if fallback, ok := value["value"].(string); ok {
			if trimmed := strings.TrimSpace(fallback); trimmed != "" {
				translations["en-US"] = trimmed
			}
		}
		if nested, ok := value["translations"].(map[string]any); ok {
			for lang, localized := range nested {
				if text, ok := localized.(string); ok {
					if trimmed := strings.TrimSpace(text); trimmed != "" {
						translations[lang] = trimmed
					}
				}
			}
		}
	}
	return translations
}

func nullableChallengeString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableChallengeInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func (p *HaloProvider) doGetWithAccept(ctx context.Context, rawURL string, tokens *domain.HaloTokens, accept string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if err := p.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("doGetWithAccept new request: %w", err)
		}
		req.Header.Set("Accept", accept)
		req.Header.Set("x-343-authorization-spartan", tokens.SpartanToken)
		if tokens.ClearanceToken != "" {
			req.Header.Set("343-clearance", tokens.ClearanceToken)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if readErr != nil {
				lastErr = readErr
			} else if resp.StatusCode == http.StatusOK {
				return body, nil
			} else if resp.StatusCode == http.StatusNotFound {
				return nil, fmt.Errorf("doGetWithAccept %s: 404", rawURL)
			} else if resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("doGetWithAccept %s: %d", rawURL, resp.StatusCode)
			} else {
				return nil, fmt.Errorf("doGetWithAccept %s: %d", rawURL, resp.StatusCode)
			}
		}

		if attempt < p.maxRetries-1 {
			backoff := providerRetryBase * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
	return nil, lastErr
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
		return "fr-FR"
	case "en", "en-us":
		return "en-US"
	default:
		return "fr-FR"
	}
}

func challengeLanguageCandidates(lang string) []string {
	if lang == "fr-FR" {
		return []string{"fr-FR", "fr"}
	}
	if lang == "en-US" {
		return []string{"en-US", "en-GB", "en"}
	}
	short := strings.Split(lang, "-")[0]
	return []string{lang, short}
}

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
