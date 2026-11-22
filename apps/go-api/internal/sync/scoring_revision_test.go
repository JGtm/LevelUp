package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/analysis/skill_v2/ep"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/sync/skill"
)

// Empreintes des paramètres de calcul, par révision. Une révision ne change jamais de valeur
// d'empreinte : changer un paramètre impose de changer la constante de révision (scoring_revision.go)
// ET d'ajouter ici une entrée pour la nouvelle révision. Les entrées anciennes restent, elles
// documentent l'historique des formules.
var (
	performanceParametersByRevision = map[string]string{
		"perf-2026-11-08": "858ec5dd81c1ce3e36adb02fe0578f388abc983152eb67e56e60547b19fae43a",
		// Note à objectif (V7, objectiveblend) : profil de combat des chaînes objectif sans ospm, réglages
		// de la note à objectif, poids des frags de soutien, vie effective 229.
		"perf-2026-11-15": "f361041a0e31392d3d269dd2d5ec0ab23a5aa3ef046c5950b555ba897be22c35",
		// Même empreinte : la logique change, pas un paramètre haché. En repli sur les autres familles,
		// O_sout ne classe plus le débit de soutien par minute, seulement la part du soutien.
		"perf-2026-11-15b": "f361041a0e31392d3d269dd2d5ec0ab23a5aa3ef046c5950b555ba897be22c35",
		// Le temps de jeu que la note lit se calcule sur le coup d'envoi du registre (ADR 0050) ; l'empreinte
		// hache désormais timeline.KickoffRev, qui lie les deux révisions.
		"perf-2026-11-18": "2c6301e1c657bc65dea1f009d674e6a478e0d3f4289874428dc1f2ad6402be98",
	}
	lusrParametersByRevision = map[string]string{
		"lusr-2026-11-08": "ef84e6e155f63198d9a7387d11af1f1dfd77473d4e12bad220e667ef5fd3443a",
		// Le repli du solveur sans comptes (solveWithCountFallback) est de la logique : aucun paramètre haché ne change.
		"lusr-2026-11-09": "ef84e6e155f63198d9a7387d11af1f1dfd77473d4e12bad220e667ef5fd3443a",
		// Vie effective du TOML d'Infinite (229) : Conversion offensive et Résistance défensive du composite.
		"lusr-2026-11-15": "f5c143ce41819332a9a06a5943c9b09017dba175cbe19b966dfd58801d3bc047",
		// Coup d'envoi et temps de jeu, que le poids d'un joueur lit, viennent de la résolution unique
		// (ADR 0050) ; l'empreinte hache désormais timeline.KickoffRev, qui lie les deux révisions.
		"lusr-2026-11-18": "31e129714c2ee40a72f92fdad2218ebac7788f0c07f38099cf708752c120427e",
	}
)

// fingerprintBuilder accumule des lignes `clé=valeur` puis les hache triées : l'ordre
// d'insertion et l'ordre d'itération des maps n'influencent pas l'empreinte.
type fingerprintBuilder struct{ lines []string }

func (b *fingerprintBuilder) float(key string, v float64) {
	b.lines = append(b.lines, key+"="+strconv.FormatFloat(v, 'g', -1, 64))
}

func (b *fingerprintBuilder) text(key, v string) {
	b.lines = append(b.lines, key+"="+v)
}

func (b *fingerprintBuilder) integer(key string, v int) {
	b.lines = append(b.lines, key+"="+strconv.Itoa(v))
}

func (b *fingerprintBuilder) weights(prefix string, w map[string]float64) {
	for k, v := range w {
		b.float(prefix+"."+k, v)
	}
}

func (b *fingerprintBuilder) sum() (string, string) {
	sort.Strings(b.lines)
	body := strings.Join(b.lines, "\n")
	h := sha256.Sum256([]byte(body))
	return hex.EncodeToString(h[:]), body
}

// infiniteEffectiveHpToKill lit la vie effective de Halo Infinite dans sa config réelle
// (config/titles/halo_infinite/constants.toml [damage_model]), la source unique que servent le
// resolver du titre et /bootstrap : les empreintes hachent CETTE valeur, pas une constante Go.
func infiniteEffectiveHpToKill(t *testing.T) float64 {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "..")
	reg := mappings.NewRegistry()
	if errs := reg.LoadFromConfigDir(root, []string{"halo_infinite"}, nil); len(errs) != 0 {
		t.Fatalf("config de halo_infinite illisible : %v", errs)
	}
	dm, ok := games.NewMappingsEndpointResolver(reg, "halo_infinite").DamageModelFor("halo_infinite")
	if !ok || dm.EffectiveHpToKill <= 0 {
		t.Fatalf("halo_infinite ne déclare pas effective_hp_to_kill : %+v", dm)
	}
	return dm.EffectiveHpToKill
}

// performanceParameters rend l'empreinte des paramètres de la note de performance relative ; hp est la
// vie effective du titre, lue de sa config.
func performanceParameters(hp float64) (digest, body string) {
	var b fingerprintBuilder
	b.integer("perfWindowSize", perfWindowSize)
	b.float("perfFallbackDurationSeconds", perfFallbackDurationSeconds)
	b.integer("MinMatchesForRelative", skill.MinMatchesForRelative)
	b.integer("MinMatchesPerChainForRelative", skill.MinMatchesPerChainForRelative)
	b.float("games.DefaultEffectiveHpToKill", hp) // clé historique des empreintes ; valeur = TOML du titre
	blend := objectiveBlendParams()
	b.float("objectiveblend.AlphaMin", blend.AlphaMin)
	b.float("objectiveblend.AlphaMax", blend.AlphaMax)
	b.float("objectiveblend.SupportSlope", blend.SupportSlope)
	b.integer("objectiveblend.MinFamilyMatches", blend.MinFamilyMatches)
	b.float("narrative.ObjectiveSupportFragWeight", narrative.ObjectiveSupportFragWeight)
	// Le temps de jeu que la note lit vient de la résolution du coup d'envoi (ADR 0050) : changer sa
	// révision change cette empreinte, donc impose de monter PerformanceScoreRev.
	b.text("timeline.KickoffRev", timeline.KickoffRev)
	b.weights("RelativeWeights", skill.RelativeWeights)
	for _, chain := range knownPerformanceChains() {
		b.weights("WeightsForChain."+chain, skill.WeightsForChain(chain))
	}
	return b.sum()
}

// knownPerformanceChains liste les chaînes de performance (LUSR et superset du score de performance).
func knownPerformanceChains() []string {
	return []string{
		skill.LUSRChainArenaSlayer, skill.LUSRChainArenaObjectif, skill.LUSRChainBTB, skill.LUSRChainChaos,
		skill.PerfChainRankedSlayer, skill.PerfChainRankedObjectif, skill.PerfChainFirefight,
	}
}

// lusrParameters rend l'empreinte des paramètres du LUSR : constantes et tables de
// skill_config.go, pénalités d'abandon, et valeurs par défaut du moteur TrueSkill 2.
func lusrParameters(hp float64) (digest, body string) {
	var b fingerprintBuilder
	addSkillConfigParameters(&b)
	// Le LUSR lit le coup d'envoi et le temps de jeu de la résolution unique (ADR 0050).
	b.text("timeline.KickoffRev", timeline.KickoffRev)
	b.float("skill.QuitSecondaryFactor", skill.QuitSecondaryFactor)
	b.float("games.DefaultEffectiveHpToKill", hp) // clé historique des empreintes ; valeur = TOML du titre
	addSkillV2Parameters(&b)
	return b.sum()
}

// addSkillConfigParameters ajoute les valeurs numériques de skill_config.go propres au LUSR.
func addSkillConfigParameters(b *fingerprintBuilder) {
	b.float("InitialMU", skill.InitialMU)
	b.float("InitialSigma", skill.InitialSigma)
	b.float("MinSigma", skill.MinSigma)
	b.float("MaxSigma", skill.MaxSigma)
	b.float("Beta", skill.Beta)
	b.float("Tau", skill.Tau)
	b.float("DrawProbability", skill.DrawProbability)
	b.integer("MinMatchesForRating", skill.MinMatchesForRating)
	b.integer("CSRPlacementThresholdDefault", skill.CSRPlacementThresholdDefault)
	b.integer("LUSRPlacementThreshold", skill.LUSRPlacementThreshold)
	b.float("MinRating", skill.MinRating)
	b.float("KElo", skill.KElo)
	b.float("InactivitySigmaPerDay", skill.InactivitySigmaPerDay)
	b.integer("MaxInactivityDays", skill.MaxInactivityDays)
	b.float("InactivityThresholdDay", skill.InactivityThresholdDay)
	b.weights("CompositeWeights", skill.CompositeWeights)
	b.integer("AccuracyHistorySize", skill.AccuracyHistorySize)
	b.integer("MinMatchesForAccuracyDelta", skill.MinMatchesForAccuracyDelta)
	b.float("IndividualMUAlpha", skill.IndividualMUAlpha)
	b.float("DefaultOpponentSigma", skill.DefaultOpponentSigma)
	b.float("LUSRMaxDelta", skill.LUSRMaxDelta)
	for _, t := range skill.SkillTiers {
		b.float("SkillTiers."+t.Name+".MinRating", t.MinRating)
		b.float("SkillTiers."+t.Name+".MaxRating", t.MaxRating)
		b.integer("SkillTiers."+t.Name+".SubTiers", t.SubTiers)
	}
}

// addSkillV2Parameters ajoute les valeurs par défaut du moteur TrueSkill 2 utilisées par le LUSR.
func addSkillV2Parameters(b *fingerprintBuilder) {
	p := skillv2.DefaultPriors()
	b.float("skillv2.Priors.Mu0", p.Mu0)
	b.float("skillv2.Priors.Sigma0", p.Sigma0)
	b.float("skillv2.Priors.Beta", p.Beta)
	b.float("skillv2.Priors.Tau", p.Tau)
	b.float("skillv2.Priors.DrawProbability", p.DrawProbability)
	for t, h := range skillv2.DefaultCountHyperparamsMap() {
		prefix := "skillv2.CountHyperparams." + strconv.Itoa(int(t))
		b.float(prefix+".Bias", h.Bias)
		b.float(prefix+".WeightPlayer", h.WeightPlayer)
		b.float(prefix+".WeightOpponent", h.WeightOpponent)
		b.float(prefix+".ObservationVar", h.ObservationVar)
	}
	for _, t := range skillv2.DefaultTierBoundaries() {
		b.float("skillv2.TierBoundaries."+t.Name+".MinMu", t.MinMu)
		b.integer("skillv2.TierBoundaries."+t.Name+".SubTiers", t.SubTiers)
	}
	b.float("skillv2.DefaultQuitDeltaRelated", skillv2.DefaultQuitDeltaRelated)
	b.float("skillv2.DefaultQuitDeltaUnrelated", skillv2.DefaultQuitDeltaUnrelated)
	b.float("skillv2.QuitPenaltyMuCap", skillv2.QuitPenaltyMuCap)
	b.float("skillv2.SquadOffsetCap", skillv2.SquadOffsetCap)
	b.float("skillv2.DefaultModeCouplingWeight", skillv2.DefaultModeCouplingWeight)
	b.float("skillv2.Phase4ModeCouplingMaxWeight", skillv2.Phase4ModeCouplingMaxWeight)
	b.integer("skillv2.PlacementMatches", skillv2.PlacementMatches)
	cfg := ep.DefaultMatch2TeamConfig()
	b.float("ep.Match2TeamConfig.Tolerance", cfg.Tolerance)
	b.integer("ep.Match2TeamConfig.MaxIters", cfg.MaxIters)
}

// requireRevisionFingerprint compare l'empreinte courante à celle enregistrée pour la révision
// courante, et dit quoi faire dans chacun des deux cas d'échec.
func requireRevisionFingerprint(t *testing.T, revConst, rev string, table map[string]string, digest, body string) {
	t.Helper()
	want, known := table[rev]
	switch {
	case !known:
		t.Fatalf("la révision %q n'a pas d'empreinte enregistrée.\n"+
			"Ajouter à la table de scoring_revision_test.go : %q: %q\nParamètres hachés :\n%s",
			rev, rev, digest, body)
	case want != digest:
		t.Fatalf("un paramètre de calcul a changé sans que la révision change : changer la constante %s "+
			"(scoring_revision.go), puis ajouter à la table de scoring_revision_test.go : "+
			"%q: %q (l'entrée de %q reste en place).\nParamètres hachés :\n%s",
			revConst, "<nouvelle révision>", digest, rev, body)
	}
}

// TestScoringParametersMatchRevision garantit qu'une modification d'un paramètre de la note de
// performance ou du LUSR change la révision correspondante, donc relance le calcul complet.
func TestScoringParametersMatchRevision(t *testing.T) {
	t.Run("performance", func(t *testing.T) {
		digest, body := performanceParameters(infiniteEffectiveHpToKill(t))
		requireRevisionFingerprint(t, "skill.PerformanceScoreRev", skill.PerformanceScoreRev,
			performanceParametersByRevision, digest, body)
	})
	t.Run("lusr", func(t *testing.T) {
		digest, body := lusrParameters(infiniteEffectiveHpToKill(t))
		requireRevisionFingerprint(t, "skill.LUSRRev", skill.LUSRRev, lusrParametersByRevision, digest, body)
	})
}

// L'empreinte hache la valeur CHARGÉE du TOML du titre : une autre vie change les deux empreintes,
// donc impose de changer les révisions ; la constante Go de repli n'y entre pas.
func TestScoringParameters_LEmpreinteSuitLaValeurDuTOML(t *testing.T) {
	hp := infiniteEffectiveHpToKill(t)
	for name, digestOf := range map[string]func(float64) (string, string){
		"performance": performanceParameters,
		"lusr":        lusrParameters,
	} {
		here, body := digestOf(hp)
		other, _ := digestOf(hp + 1)
		if here == other {
			t.Errorf("%s : une autre vie effective ne change pas l'empreinte", name)
		}
		if !strings.Contains(body, "games.DefaultEffectiveHpToKill="+strconv.FormatFloat(hp, 'g', -1, 64)) {
			t.Errorf("%s : la vie du TOML (%v) n'est pas dans les paramètres hachés : %s", name, hp, body)
		}
	}
}

func TestScoringRevisionComposition(t *testing.T) {
	want := "perf=" + skill.PerformanceScoreRev + ";lusr=" + skill.LUSRRev
	if got := skill.ScoringRevision(); got != want {
		t.Fatalf("skill.ScoringRevision() = %q, attendu %q", got, want)
	}
	if skill.SyncMetaScoringRevision != "scoring_revision" {
		t.Fatalf("clé sync_meta inattendue : %q", skill.SyncMetaScoringRevision)
	}
}
