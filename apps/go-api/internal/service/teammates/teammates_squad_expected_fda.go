// Package teammates — teammates_squad_expected_fda.go : écart au FDA attendu PAR
// MEMBRE (teammates.16). Résout les assists attendus de chaque membre via SA player
// DB (chaîne D8 : personnel → populationnel → nil), avec cache par (membre, mode) et
// par mode. L'attendu K/D vient de r.Self.KillsExpected/DeathsExpected (canonical).
package teammates

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/service/squadagg"
)

// titleHasExpectedStats indique si le titre déclare CapExpectedStats (écart au FDA
// attendu). Gate title-agnostic (jamais slug== — ratchet no_slug_comparison_test.go)
// de la résolution des assists attendus par membre : évite les requêtes DB et le
// bruit de logs sur les titres sans stats attendues (Halo 5).
func titleHasExpectedStats(slug string) bool {
	d := titlePkg.DefaultRegistry().Get(slug)
	return d != nil && d.HasCapability(titlePkg.CapExpectedStats)
}

// squadExpectedAssistsResolver résout et met en cache les assists attendus des
// membres de l'escouade. Chaîne D8 : modèle personnel du membre (player DB, clé
// gt|mode) → fallback populationnel (metadata title-level, clé mode) → nil. Chaque
// résolution DB est faite UNE fois (pas par match). Échec loggé, jamais silencieux.
type squadExpectedAssistsResolver struct {
	ctx      context.Context
	loader   squadagg.SquadV2Loader
	slug     string
	personal map[string]*domain.PlayerAssistsModel // clé "gt\x00mode" (présence = résolu)
	popCoef  map[string]popAssistsCoef             // clé mode (présence = résolu)
}

type popAssistsCoef struct {
	slope, intercept float64
	ok               bool
}

func newSquadExpectedAssistsResolver(ctx context.Context, loader squadagg.SquadV2Loader, slug string) *squadExpectedAssistsResolver {
	return &squadExpectedAssistsResolver{
		ctx:      ctx,
		loader:   loader,
		slug:     slug,
		personal: make(map[string]*domain.PlayerAssistsModel),
		popCoef:  make(map[string]popAssistsCoef),
	}
}

// assistsFor retourne les assists attendus du membre gt pour la row r (mode =
// r.Summary.GameVariant). nil si aucun modèle résoluble → terme 0 dans ExpectedFDA.
func (z *squadExpectedAssistsResolver) assistsFor(gt string, r *canonical.PlayerMatchRow) *float64 {
	if z == nil || z.loader == nil {
		return nil
	}
	mode := ""
	if r.Summary.GameVariant != nil {
		mode = r.Summary.GameVariant.DefaultLabel
	}
	if m := z.personalModel(gt, mode); m != nil {
		v := analysis.ApplyPersonalAssistsModel(m,
			canonicalIntF(r.Self.Kills), canonicalIntF(r.Self.Deaths),
			canonicalIntF(r.Self.DamageDealt), canonicalIntF(r.Self.DamageTaken),
			memberMMRDelta(r))
		return &v
	}
	if pc := z.populational(mode); pc.ok {
		v := analysis.ApplyPopulationalAssists(pc.slope, pc.intercept,
			canonicalIntF(r.Self.PersonalScore), canonicalIntF(r.Self.ShotsHit))
		return &v
	}
	return nil
}

func (z *squadExpectedAssistsResolver) personalModel(gt, mode string) *domain.PlayerAssistsModel {
	key := gt + "\x00" + mode
	if m, done := z.personal[key]; done {
		return m
	}
	m, err := z.loader.LoadPlayerAssistsModel(z.ctx, z.slug, gt, mode)
	if err != nil {
		slog.WarnContext(z.ctx, "squad_expected_assists_personal_failed", "gamertag", gt, "mode", mode, "err", err)
	}
	z.personal[key] = m // m peut être nil (membre sans modèle) — marque résolu
	return m
}

func (z *squadExpectedAssistsResolver) populational(mode string) popAssistsCoef {
	if pc, done := z.popCoef[mode]; done {
		return pc
	}
	slope, intercept, ok, err := z.loader.LoadPopulationalAssistsCoef(z.ctx, z.slug, mode)
	if err != nil {
		slog.WarnContext(z.ctx, "squad_expected_assists_pop_failed", "mode", mode, "err", err)
	}
	pc := popAssistsCoef{slope: slope, intercept: intercept, ok: ok}
	z.popCoef[mode] = pc
	return pc
}

// canonicalIntF déréférence un *int canonical en float64 (0 si nil).
func canonicalIntF(p *int) float64 {
	if p == nil {
		return 0
	}
	return float64(*p)
}

// memberMMRDelta retourne team_mmr − enemy_mmr du membre (0 si l'un manque).
func memberMMRDelta(r *canonical.PlayerMatchRow) float64 {
	if r.Enrichment.TeamMMR != nil && r.Enrichment.EnemyMMR != nil {
		return *r.Enrichment.TeamMMR - *r.Enrichment.EnemyMMR
	}
	return 0
}
