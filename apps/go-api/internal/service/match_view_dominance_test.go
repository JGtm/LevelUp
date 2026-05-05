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
func (s *stubAssetURL) CSRRankImageURLOnyx() string { return s.onyxImg }

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
			name: "LUSR (custom) → pas de badge officiel",
			raw: &domain.SkillRankRaw{
				RatingType: "LUSR",
				Tier:       &tDiamond,
				SubTier:    &subTier3,
			},
			assetURL: asset,
			want:     "",
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
