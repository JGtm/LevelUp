package sync

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/analysis/skill_v2/ep"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/skill"
)

// Empreintes des paramètres de calcul, par révision. Une révision ne change jamais de valeur
// d'empreinte : changer un paramètre impose de changer la constante de révision (scoring_revision.go)
// ET d'ajouter ici une entrée pour la nouvelle révision. Les entrées anciennes restent, elles
// documentent l'historique des formules.
var (
	performanceParametersByRevision = map[string]string{
		"perf-2026-11-08": "858ec5dd81c1ce3e36adb02fe0578f388abc983152eb67e56e60547b19fae43a",
	}
	lusrParametersByRevision = map[string]string{
		"lusr-2026-11-08": "ef84e6e155f63198d9a7387d11af1f1dfd77473d4e12bad220e667ef5fd3443a",
		// Le repli du solveur sans comptes (solveWithCountFallback) est de la logique : aucun paramètre haché ne change.
		"lusr-2026-11-09": "ef84e6e155f63198d9a7387d11af1f1dfd77473d4e12bad220e667ef5fd3443a",
	}
)

// fingerprintBuilder accumule des lignes `clé=valeur` puis les hache triées : l'ordre
// d'insertion et l'ordre d'itération des maps n'influencent pas l'empreinte.
type fingerprintBuilder struct{ lines []string }

func (b *fingerprintBuilder) float(key string, v float64) {
	b.lines = append(b.lines, key+"="+strconv.FormatFloat(v, 'g', -1, 64))
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

// performanceParameters rend l'empreinte des paramètres de la note de performance relative.
func performanceParameters() (digest, body string) {
	var b fingerprintBuilder
	b.integer("perfWindowSize", perfWindowSize)
	b.float("perfFallbackDurationSeconds", perfFallbackDurationSeconds)
	b.integer("MinMatchesForRelative", skill.MinMatchesForRelative)
	b.integer("MinMatchesPerChainForRelative", skill.MinMatchesPerChainForRelative)
	b.float("games.DefaultEffectiveHpToKill", games.DefaultEffectiveHpToKill)
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
func lusrParameters() (digest, body string) {
	var b fingerprintBuilder
	addSkillConfigParameters(&b)
	b.float("skill.QuitSecondaryFactor", skill.QuitSecondaryFactor)
	b.float("games.DefaultEffectiveHpToKill", games.DefaultEffectiveHpToKill)
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
		digest, body := performanceParameters()
		requireRevisionFingerprint(t, "skill.PerformanceScoreRev", skill.PerformanceScoreRev,
			performanceParametersByRevision, digest, body)
	})
	t.Run("lusr", func(t *testing.T) {
		digest, body := lusrParameters()
		requireRevisionFingerprint(t, "skill.LUSRRev", skill.LUSRRev, lusrParametersByRevision, digest, body)
	})
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
