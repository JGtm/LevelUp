package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// TestBuildMatchHeader_DominanceBadge verifie le branchement Phase 1 méta-plan
// entre l'enrichissement DB (dominance_flag) et le badge narratif typé exposé
// dans le DTO header.
func TestBuildMatchHeader_DominanceBadge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		flag           int
		wantBadge      bool
		wantLabelKey   string
		wantColorToken string
	}{
		{
			name:           "domination -> badge typé",
			flag:           int(canonical.DominanceDomination),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.domination",
			wantColorToken: "narrative.dominance.win.strong",
		},
		{
			name:           "humiliation -> badge inversé",
			flag:           int(canonical.DominanceHumiliation),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.humiliation",
			wantColorToken: "narrative.dominance.loss.strong",
		},
		{
			name:           "remontada -> badge comeback",
			flag:           int(canonical.DominanceRemontada),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.remontada",
			wantColorToken: "narrative.dominance.win.comeback",
		},
		{
			name:      "none -> pas de badge",
			flag:      int(canonical.DominanceNone),
			wantBadge: false,
		},
		{
			name:      "flag inconnu -> pas de badge (dégradation gracieuse)",
			flag:      99,
			wantBadge: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			enrich := &domain.MatchEnrichmentRaw{DominanceFlag: tc.flag}
			h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, nil, enrich, nil, nil, false)

			if tc.wantBadge {
				if h.DominanceBadge == nil {
					t.Fatalf("DominanceBadge want non-nil for flag %d", tc.flag)
				}
				if h.DominanceBadge.LabelKey != tc.wantLabelKey {
					t.Errorf("LabelKey want %s, got %s", tc.wantLabelKey, h.DominanceBadge.LabelKey)
				}
				if h.DominanceBadge.ColorToken != tc.wantColorToken {
					t.Errorf("ColorToken want %s, got %s", tc.wantColorToken, h.DominanceBadge.ColorToken)
				}
				if h.DominanceBadge.Flag != tc.flag {
					t.Errorf("Flag want %d, got %d", tc.flag, h.DominanceBadge.Flag)
				}
				if !h.DominanceFlag {
					t.Error("DominanceFlag bool legacy want true (badge présent)")
				}
			} else {
				if h.DominanceBadge != nil {
					t.Errorf("DominanceBadge want nil for flag %d, got %+v", tc.flag, h.DominanceBadge)
				}
				if h.DominanceFlag {
					t.Error("DominanceFlag bool legacy want false (pas de badge)")
				}
			}
		})
	}
}

// TestBuildMatchHeader_NilEnrichment verifie qu'une enrichissement absent ne
// casse pas le header (cas legacy ou capability gap).
func TestBuildMatchHeader_NilEnrichment(t *testing.T) {
	t.Parallel()
	h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, nil, nil, nil, nil, false)
	if h.DominanceBadge != nil {
		t.Errorf("nil enrich: DominanceBadge want nil, got %+v", h.DominanceBadge)
	}
	if h.DominanceFlag {
		t.Error("nil enrich: DominanceFlag bool want false")
	}
}

// TestOutcomeColorToken_MapsAllCodes verifie le mapping outcome -> token
// (Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup hex codes).
func TestOutcomeColorToken_MapsAllCodes(t *testing.T) {
	t.Parallel()
	cases := map[int]string{
		1: "outcome-draw",
		2: "outcome-win",
		3: "outcome-loss",
		4: "outcome-dnf",
		0: "",
		9: "",
	}
	for code, want := range cases {
		got := outcomeColorToken(code)
		if got != want {
			t.Errorf("outcomeColorToken(%d) want %q, got %q", code, want, got)
		}
	}
}

// TestPerfColorToken_MapsScoreToTier verifie le mapping score -> perf-tier-N.
func TestPerfColorToken_MapsScoreToTier(t *testing.T) {
	t.Parallel()
	cases := map[float64]string{
		95.0: "perf-tier-1", // excellent
		75.0: "perf-tier-2", // bon
		55.0: "perf-tier-3", // moyen
		25.0: "perf-tier-4", // faible
		10.0: "perf-tier-5", // très faible
		0.0:  "perf-tier-5",
	}
	for score, want := range cases {
		got := perfColorToken(score)
		if got != want {
			t.Errorf("perfColorToken(%g) want %q, got %q", score, want, got)
		}
	}
}

// TestBuildMatchHeader_OutcomeColorToken verifie que le token sémantique
// est exposé dans le header pour chaque code outcome.
func TestBuildMatchHeader_OutcomeColorToken(t *testing.T) {
	t.Parallel()
	stats := &domain.PlayerMatchStatsRaw{OutcomeCode: 2} // win
	h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, stats, nil, nil, nil, false)
	if h.OutcomeColorToken != "outcome-win" {
		t.Errorf("OutcomeColorToken want outcome-win, got %q", h.OutcomeColorToken)
	}
	// Le hex legacy reste exposé pour rétrocompat
	if h.OutcomeColor == "" {
		t.Error("OutcomeColor (hex legacy) want non-empty")
	}
}

// stubAssetURL : implémentation minimale de games.TitleAssetURLAdapter
// pour les tests buildMatchHeader / buildRankBlock. mapImg = chaîne fixe
// retournée pour tout name non vide ; csrFn = formateur tier+sub. Onyx fixe.
type stubAssetURL struct {
	mapImg     string
	csrPattern string
	onyxImg    string
}

func (s *stubAssetURL) TitleSlug() string { return "halo_infinite" }
func (s *stubAssetURL) MapImageURL(name string) string {
	if name == "" {
		return ""
	}
	return s.mapImg
}
func (s *stubAssetURL) MedalImageURL(_ uint64) string { return "" }
func (s *stubAssetURL) CSRRankImageURL(tier string, subTier int) string {
	if tier == "" {
		return ""
	}
	return s.csrPattern + tier + ":" + itoa(subTier)
}
func (s *stubAssetURL) CSRRankImageURLOnyx() string          { return s.onyxImg }
func (s *stubAssetURL) WeaponImageURL(_ string) string       { return "" }
func (s *stubAssetURL) MatchWebURL(_ string) string          { return "" }
func (s *stubAssetURL) PlayerMatchWebURL(_, _ string) string { return "" }

func itoa(n int) string {
	// minimal helper pour éviter strconv import dans le file de test
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// TestBuildMatchHeader_MapImageURL verifie le câblage TitleAssetURLAdapter.
func TestBuildMatchHeader_MapImageURL(t *testing.T) {
	t.Parallel()

	mapName := "Aquarius"
	cases := []struct {
		name     string
		assetURL *stubAssetURL
		mapName  *string
		want     string
	}{
		{
			name:     "asset URL résout → MapImageURL non nil",
			assetURL: &stubAssetURL{mapImg: "/static/maps/halo_infinite/Aquarius.png"},
			mapName:  &mapName,
			want:     "/static/maps/halo_infinite/Aquarius.png",
		},
		{
			name:     "adapter nil → MapImageURL nil (dégradation)",
			assetURL: nil,
			mapName:  &mapName,
			want:     "",
		},
		{
			name:     "adapter retourne vide → MapImageURL nil + warn loggé",
			assetURL: &stubAssetURL{mapImg: ""},
			mapName:  &mapName,
			want:     "",
		},
		{
			name:     "mapName nil → MapImageURL nil (pas de panic)",
			assetURL: &stubAssetURL{mapImg: "/some.png"},
			mapName:  nil,
			want:     "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := &domain.MatchMetaRaw{MapName: tc.mapName}
			var assetURL games.TitleAssetURLAdapter
			if tc.assetURL != nil {
				assetURL = tc.assetURL
			}
			h := buildMatchHeader(context.Background(), "m1", meta, nil, nil, nil, assetURL, false)
			got := ""
			if h.MapImageURL != nil {
				got = *h.MapImageURL
			}
			if got != tc.want {
				t.Errorf("MapImageURL = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestBuildMatchHeader_MapImageURL_PrefersENNameOverRawUUID reproduit
// exactement le scénario observé en prod 2026-05-08 : `match_registry.map_name`
// contient l'UUID brut "2890782c-…", mais `asset_translations` en-US a
// "Shiro". Le registry est vide pour ce map_id (cmd/migrate-static-maps pas
// re-runnée). L'adapter doit recevoir "Shiro" (nom EN résolu), pas l'UUID,
// pour pouvoir retrouver l'image dans son index static dir.
func TestBuildMatchHeader_MapImageURL_PrefersENNameOverRawUUID(t *testing.T) {
	t.Parallel()

	rawUUID := "2890782c-0a33-4f2c-a468-e3a7d6cd6db4"
	enName := "Shiro"

	// Stub qui mimique le vrai AssetURLAdapter : rejette les UUIDs (regex
	// uuidRe), accepte les noms propres.
	stubRejectUUID := &stubAssetURLNameAware{
		validNames: map[string]string{enName: "/static/maps/halo_infinite/Shiro.jpg"},
	}

	// Cas critique : MapName=UUID + MapNameEN="Shiro" → adapter reçoit "Shiro"
	meta := &domain.MatchMetaRaw{
		MapName:   &rawUUID,
		MapNameEN: &enName,
	}
	h := buildMatchHeader(context.Background(), "m1", meta, nil, nil, nil, stubRejectUUID, false)
	if h.MapImageURL == nil || *h.MapImageURL != "/static/maps/halo_infinite/Shiro.jpg" {
		t.Errorf("MapImageURL = %v, want /static/maps/halo_infinite/Shiro.jpg (l'adapter doit recevoir le nom EN, pas l'UUID)", h.MapImageURL)
	}
	if got := stubRejectUUID.lastNameReceived; got != enName {
		t.Errorf("nom passé à l'adapter = %q, want %q (UUID brut envoyé à la place)", got, enName)
	}

	// Cas dégradé : MapNameEN absent → fallback sur MapName brut (qui peut
	// échouer dans l'adapter mais c'est le mieux qu'on puisse faire)
	meta2 := &domain.MatchMetaRaw{MapName: &enName} // ici MapName = nom propre déjà
	h2 := buildMatchHeader(context.Background(), "m1", meta2, nil, nil, nil, stubRejectUUID, false)
	if h2.MapImageURL == nil || *h2.MapImageURL != "/static/maps/halo_infinite/Shiro.jpg" {
		t.Errorf("fallback MapName brut: MapImageURL = %v, want Shiro URL", h2.MapImageURL)
	}
}

// stubAssetURLNameAware mimique le vrai AssetURLAdapter : retourne l'URL
// uniquement pour les noms présents dans validNames (rejette UUID, etc.).
// Mémorise lastNameReceived pour vérifier ce qui a été passé en paramètre.
type stubAssetURLNameAware struct {
	validNames       map[string]string
	lastNameReceived string
}

func (s *stubAssetURLNameAware) TitleSlug() string { return "halo_infinite" }
func (s *stubAssetURLNameAware) MapImageURL(name string) string {
	s.lastNameReceived = name
	if url, ok := s.validNames[name]; ok {
		return url
	}
	return ""
}
func (s *stubAssetURLNameAware) MedalImageURL(_ uint64) string          { return "" }
func (s *stubAssetURLNameAware) CSRRankImageURL(_ string, _ int) string { return "" }
func (s *stubAssetURLNameAware) CSRRankImageURLOnyx() string            { return "" }
func (s *stubAssetURLNameAware) WeaponImageURL(_ string) string         { return "" }
func (s *stubAssetURLNameAware) MatchWebURL(_ string) string            { return "" }
func (s *stubAssetURLNameAware) PlayerMatchWebURL(_, _ string) string   { return "" }

// TestBuildMatchHeader_MapImageRegistry vérifie que meta.MapImageURL (registry)
// a la priorité sur l'adapter name-based, et que l'adapter sert de fallback.
func TestBuildMatchHeader_MapImageRegistry(t *testing.T) {
	t.Parallel()
	mapName := "Forbidden"
	registryURL := "/static/maps/halo_infinite/Forbidden.jpg"

	// Registry résout → priorité sur l'adapter (même si l'adapter renverrait autre chose)
	meta := &domain.MatchMetaRaw{
		MapName:     &mapName,
		MapImageURL: &registryURL,
	}
	h := buildMatchHeader(context.Background(), "m1", meta, nil, nil, nil,
		&stubAssetURL{mapImg: "/other.jpg"}, false)
	if h.MapImageURL == nil || *h.MapImageURL != registryURL {
		t.Errorf("registry priorité: MapImageURL = %v, want %q", h.MapImageURL, registryURL)
	}

	// Registry nil → fallback adapter
	metaNoReg := &domain.MatchMetaRaw{MapName: &mapName}
	adapterURL := "/static/maps/halo_infinite/Forbidden.jpg"
	h2 := buildMatchHeader(context.Background(), "m1", metaNoReg, nil, nil, nil,
		&stubAssetURL{mapImg: adapterURL}, false)
	if h2.MapImageURL == nil || *h2.MapImageURL != adapterURL {
		t.Errorf("fallback adapter: MapImageURL = %v, want %q", h2.MapImageURL, adapterURL)
	}
}

// TestBuildMatchHeader_ModeFallbackNormalized vérifie que le fallback mode
// utilise le label EN normalisé (pas le pair_name brut avec suffixe map).
func TestBuildMatchHeader_ModeFallbackNormalized(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		pairName   string
		modeNameFR *string
		mapNameFR  *string
		want       string
	}{
		{
			name:     "pair_name avec suffixe map → label normalisé EN",
			pairName: "Slayer : Forbidden",
			want:     "Slayer",
		},
		{
			name:     "pair_name technique → label normalisé EN",
			pairName: "Arena:Slayer",
			want:     "Slayer",
		},
		{
			name:       "ModeNameFR prioritaire sur pair_name",
			pairName:   "Slayer : Forbidden",
			modeNameFR: strPtr("Assassin"),
			want:       "Assassin",
		},
		{
			// Régression "Slayer on Forest sur Forêt" : ModeNameFR arrive brut
			// du repo (catalogue incomplet) avec le nom de map EN collé, alors
			// que MapUI est la traduction FR. Le ModeUI doit être re-normalisé.
			name:       "ModeNameFR brut avec map EN collée → re-normalisé",
			pairName:   "Slayer : Forest",
			modeNameFR: strPtr("Slayer on Forest"),
			mapNameFR:  strPtr("Forêt"),
			want:       "Slayer",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			meta := &domain.MatchMetaRaw{
				PairName:   &tc.pairName,
				ModeNameFR: tc.modeNameFR,
				MapNameFR:  tc.mapNameFR,
			}
			h := buildMatchHeader(context.Background(), "m1", meta, nil, nil, nil, nil, false)
			if h.ModeUI != tc.want {
				t.Errorf("ModeUI = %q, want %q", h.ModeUI, tc.want)
			}
		})
	}
}

// TestDetectPartialMatchData_AllReasons vérifie que les 4 codes partial sont
// émis quand toutes les sources secondaires sont vides (RC6 — match_registry
// OK mais sync incomplet, le front rend dégradé au lieu d'un écran d'erreur).
func TestDetectPartialMatchData_AllReasons(t *testing.T) {
	reasons := detectPartialMatchData(nil, nil, nil, nil)
	want := map[string]bool{
		"scoreboard_empty":   false,
		"events_empty":       false,
		"player_stats_empty": false,
		"medals_empty":       false,
	}
	for _, r := range reasons {
		if _, ok := want[r]; !ok {
			t.Errorf("raison inattendue : %q", r)
		}
		want[r] = true
	}
	for code, found := range want {
		if !found {
			t.Errorf("raison manquante : %q", code)
		}
	}
}

// TestDetectPartialMatchData_FullData : aucune raison quand toutes les sources
// sont remplies → IsPartial=false côté response.
func TestDetectPartialMatchData_FullData(t *testing.T) {
	stats := &domain.PlayerMatchStatsRaw{OutcomeCode: 2}
	scoreboard := []domain.ScoreboardRaw{{XUID: "x"}}
	events := []domain.EventRaw{{EventType: "kill"}}
	medals := []domain.MedalRaw{{MedalID: 1, Count: 1}}
	reasons := detectPartialMatchData(stats, scoreboard, events, medals)
	if len(reasons) != 0 {
		t.Errorf("attendu aucune raison, obtenu %v", reasons)
	}
}

// TestDetectPartialMatchData_PartialMix : stats nil mais reste plein → seul
// code "player_stats_empty" émis.
func TestDetectPartialMatchData_PartialMix(t *testing.T) {
	scoreboard := []domain.ScoreboardRaw{{XUID: "x"}}
	events := []domain.EventRaw{{EventType: "kill"}}
	medals := []domain.MedalRaw{{MedalID: 1, Count: 1}}
	reasons := detectPartialMatchData(nil, scoreboard, events, medals)
	if len(reasons) != 1 || reasons[0] != "player_stats_empty" {
		t.Errorf("attendu [player_stats_empty], obtenu %v", reasons)
	}
}

// TestDetectPartialMatchData_MedalsLegitimatelyEmpty : 0 médaille avec PBitMedals
// positionné → pas de "medals_empty" (match sans médailles, donnée complète).
func TestDetectPartialMatchData_MedalsLegitimatelyEmpty(t *testing.T) {
	bits := pBitMedals // 512 — médailles fetchées, résultat = 0
	stats := &domain.PlayerMatchStatsRaw{OutcomeCode: 2, BackfillBits: &bits}
	scoreboard := []domain.ScoreboardRaw{{XUID: "x"}}
	events := []domain.EventRaw{{EventType: "kill"}}
	reasons := detectPartialMatchData(stats, scoreboard, events, nil)
	for _, r := range reasons {
		if r == "medals_empty" {
			t.Errorf("medals_empty ne doit pas être émis quand PBitMedals est positionné")
		}
	}
}

// TestDetectPartialMatchData_MedalsNeverFetched : 0 médaille sans PBitMedals →
// "medals_empty" doit être émis (sync incomplet).
func TestDetectPartialMatchData_MedalsNeverFetched(t *testing.T) {
	bits := 0 // aucun bit positionné
	stats := &domain.PlayerMatchStatsRaw{OutcomeCode: 2, BackfillBits: &bits}
	scoreboard := []domain.ScoreboardRaw{{XUID: "x"}}
	events := []domain.EventRaw{{EventType: "kill"}}
	reasons := detectPartialMatchData(stats, scoreboard, events, nil)
	found := false
	for _, r := range reasons {
		if r == "medals_empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("medals_empty attendu quand PBitMedals n'est pas positionné, obtenu %v", reasons)
	}
}

// TestBuildMatchHeader_IsFavorite verifie que le bool IsFavorite est exposé.
func TestBuildMatchHeader_IsFavorite(t *testing.T) {
	t.Parallel()
	for _, fav := range []bool{true, false} {
		h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, nil, nil, nil, nil, fav)
		if h.IsFavorite != fav {
			t.Errorf("IsFavorite = %v, want %v", h.IsFavorite, fav)
		}
	}
}

// TestBuildMatchHeader_PlaylistFR : la traduction FR a priorité sur le label brut.
func TestBuildMatchHeader_PlaylistFR(t *testing.T) {
	t.Parallel()
	en := "Ranked Arena"
	fr := "Arène classée"
	meta := &domain.MatchMetaRaw{
		PlaylistName:   &en,
		PlaylistNameFR: &fr,
	}
	h := buildMatchHeader(context.Background(), "m1", meta, nil, nil, nil, nil, false)
	if h.PlaylistLabel != fr {
		t.Errorf("PlaylistLabel = %q, want %q (FR prioritaire)", h.PlaylistLabel, fr)
	}

	// Sans FR : fallback brut EN
	meta2 := &domain.MatchMetaRaw{PlaylistName: &en}
	h2 := buildMatchHeader(context.Background(), "m1", meta2, nil, nil, nil, nil, false)
	if h2.PlaylistLabel != en {
		t.Errorf("PlaylistLabel sans FR = %q, want fallback EN %q", h2.PlaylistLabel, en)
	}
}

// TestBuildRankBlock_IconURL verifie la résolution du badge CSR via
// TitleAssetURLAdapter (chemin Onyx + non-Onyx + LUSR + nil adapter).
func TestBuildRankBlock_IconURL(t *testing.T) {
	t.Parallel()

	asset := &stubAssetURL{
		csrPattern: "/csr/",
		onyxImg:    "/onyx.png",
	}

	tDiamond := "Diamond"
	subTier3 := 3
	tOnyx := "Onyx"

	cases := []struct {
		name     string
		raw      *domain.SkillRankRaw
		assetURL games.TitleAssetURLAdapter
		want     string
	}{
		{
			name: "CSR Diamond 3 → URL paramétrée",
			raw: &domain.SkillRankRaw{
				RatingType: "CSR",
				Tier:       &tDiamond,
				SubTier:    &subTier3,
			},
			assetURL: asset,
			want:     "/csr/Diamond:3",
		},
		{
			name: "CSR Onyx → URL Onyx fixe (pas de sub-tier)",
			raw: &domain.SkillRankRaw{
				RatingType: "CSR",
				Tier:       &tOnyx,
			},
			assetURL: asset,
			want:     "/onyx.png",
		},
		{
			name: "LUSR → même badge que CSR (mêmes fichiers static)",
			raw: &domain.SkillRankRaw{
				RatingType: "LUSR",
				Tier:       &tDiamond,
				SubTier:    &subTier3,
			},
			assetURL: asset,
			want:     "/csr/Diamond:3",
		},
		{
			name: "adapter nil → IconURL vide (dégradation)",
			raw: &domain.SkillRankRaw{
				RatingType: "CSR",
				Tier:       &tDiamond,
				SubTier:    &subTier3,
			},
			assetURL: nil,
			want:     "",
		},
		{
			name: "Tier nil → IconURL vide",
			raw: &domain.SkillRankRaw{
				RatingType: "CSR",
				SubTier:    &subTier3,
			},
			assetURL: asset,
			want:     "",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rank := buildRankBlock(tc.raw, tc.assetURL)
			if rank.IconURL != tc.want {
				t.Errorf("IconURL = %q, want %q", rank.IconURL, tc.want)
			}
		})
	}
}

// TestBuildRankBlock_ProgressPct vérifie le calcul de la position dans le tier.
func TestBuildRankBlock_ProgressPct(t *testing.T) {
	t.Parallel()

	ptr := func(f float64) *float64 { return &f }
	tierGold := "Gold"
	tierOnyx := "Onyx"
	sub3 := 3

	cases := []struct {
		name    string
		raw     *domain.SkillRankRaw
		wantPct *float64 // nil = attendu nil
	}{
		{
			name:    "Gold 3 — rating 1112 → 12/50 = 0.24",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", Tier: &tierGold, SubTier: &sub3, RatingValue: ptr(1112)},
			wantPct: ptr(12.0 / 50.0),
		},
		{
			name:    "début de tier exact — rating 1100 → 0/50 = 0.0",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", Tier: &tierGold, SubTier: &sub3, RatingValue: ptr(1100)},
			wantPct: ptr(0.0),
		},
		{
			name:    "fin de tier — rating 1149 → 49/50 = 0.98",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", Tier: &tierGold, SubTier: &sub3, RatingValue: ptr(1149)},
			wantPct: ptr(49.0 / 50.0),
		},
		{
			name:    "Onyx → nil (pas de tier suivant)",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", Tier: &tierOnyx, RatingValue: ptr(1600)},
			wantPct: nil,
		},
		{
			name:    "RatingValue nil → nil",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", Tier: &tierGold, SubTier: &sub3},
			wantPct: nil,
		},
		{
			name:    "Tier nil → nil",
			raw:     &domain.SkillRankRaw{RatingType: "CSR", RatingValue: ptr(1112)},
			wantPct: nil,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rank := buildRankBlock(tc.raw, nil)
			if tc.wantPct == nil {
				if rank.ProgressPct != nil {
					t.Errorf("ProgressPct = %v, want nil", *rank.ProgressPct)
				}
				return
			}
			if rank.ProgressPct == nil {
				t.Fatalf("ProgressPct = nil, want %v", *tc.wantPct)
			}
			const epsilon = 1e-9
			if diff := *rank.ProgressPct - *tc.wantPct; diff < -epsilon || diff > epsilon {
				t.Errorf("ProgressPct = %.10f, want %.10f", *rank.ProgressPct, *tc.wantPct)
			}
		})
	}
}
