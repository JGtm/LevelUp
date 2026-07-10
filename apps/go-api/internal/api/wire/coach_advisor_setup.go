// Package api — coach_advisor_setup.go : boot-time bundle pour le coach_advisor.
//
// Charge la grammaire de synthèse (TOML) une seule fois au démarrage, puis
// fournit une factory ServiceForPlayer pour construire un coach_advisor.Service
// par joueur à chaque sync.
//
// La grammaire vide (refuse toute synthèse) est utilisée comme fallback si le
// TOML n'est pas trouvé — le matching catalogue reste fonctionnel mais sans
// extension dynamique. Cf. ADR 0028.
package wire

import (
	"log/slog"
	"path/filepath"

	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
	"levelup/go-api/internal/progression/coach_advisor"
)

// CoachAdvisorBundle détient les ressources partagées process-level du
// coach_advisor : la grammaire de synthèse chargée une fois au boot.
//
// Le Service par joueur est construit à la demande via ServiceForPlayer
// (stateless, peu coûteux).
type CoachAdvisorBundle struct {
	grammar coach_advisor.SynthesisGrammar
	cfg     coach_advisor.SynthesisConfig
}

// NewCoachAdvisorBundle charge la grammaire depuis le path standard
// config/coach_advisor/synthesis_grammar.toml. Si le fichier est absent ou
// invalide, fallback sur une grammaire vide (synthèse désactivée, matcher
// catalog uniquement) — log warn, jamais d'échec bloquant.
func NewCoachAdvisorBundle(repoRoot string) *CoachAdvisorBundle {
	path := filepath.Join(repoRoot, "config", "coach_advisor", "synthesis_grammar.toml")
	grammar, err := coach_advisor.LoadSynthesisGrammar(path)
	if err != nil {
		slog.Warn("coach_advisor: grammar not loaded, synthesis disabled",
			"path", path, "err", err)
		grammar = coach_advisor.DefaultSynthesisGrammar()
	} else {
		slog.Info("coach_advisor: grammar loaded",
			"path", path, "metrics", len(grammar.Metrics()))
	}
	return &CoachAdvisorBundle{
		grammar: grammar,
		cfg:     coach_advisor.DefaultSynthesisConfig(),
	}
}

// ServiceForPlayer construit un Service prêt à l'emploi pour le joueur dont
// la stats.duckdb est attachée à pdb. Les deps (Templates, Prestige) sont
// passées explicitement — c'est l'orchestrateur post-sync qui les résout.
//
// Retourne nil si pdb.Player == nil — la persistance des proposals est
// indispensable pour les opérations List/Generate.
func (b *CoachAdvisorBundle) ServiceForPlayer(
	pdb *platform_duckdb.PlayerDB,
	templates prestige.TemplateRepo,
	prestigeWriter coach_advisor.PrestigeWriter,
) coach_advisor.Service {
	if b == nil || pdb == nil || pdb.Player == nil {
		return nil
	}
	return coach_advisor.NewService(coach_advisor.ServiceDeps{
		Repo:        platform_duckdb.NewCoachProposalRepo(pdb.Player),
		Templates:   templates,
		Synthesizer: coach_advisor.NewSynthesizer(b.grammar, b.cfg),
		Prestige:    prestigeWriter,
	})
}
