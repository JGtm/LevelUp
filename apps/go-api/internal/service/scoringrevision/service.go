// Package scoringrevision rejoue, une fois au démarrage, notes de performance et LUSR de chaque
// joueur dont la révision de formule rangée diffère de celle du binaire (sync/skill.ScoringRevision),
// et recale le départ de ses campagnes sur l'axe Objectif quand la révision de cette mesure diffère
// (campaign.ObjectiveAxisRev) ; le port décide de ce que fait un passage par joueur.
//
// AVANT les joueurs, la base PARTAGÉE de chaque titre rattrape coup d'envoi et temps de jeu quand sa
// révision diffère (timeline.KickoffRev, ADR 0050) : les notes lisent ce que ce rattrapage écrit. Un
// rattrapage PARTIEL (des matchs en échec, port.ErrSharedRevisionPartial) laisse jouer les joueurs ; un
// rattrapage qui n'a pas pu tourner (base ou writer indisponibles, lot illisible) retient les joueurs du
// titre à ce démarrage (leur révision reste, le démarrage suivant reprend les deux).
//
// Une mise à jour qui change une formule n'a besoin d'aucun outil manuel : le premier démarrage
// qui suit recalcule l'historique de chaque joueur, une fois. Une étape qui échoue garde son
// ancienne révision et reste à rejouer au démarrage suivant (les étapes réussies du même joueur ne le
// sont pas) ; son échec ne retient ni les autres étapes ni les autres joueurs. Les compteurs et le
// journal sont par étape.
//
// Ce paquet décide de l'ordre des étapes (notes, puis campagnes) et de leur indépendance : le câblage
// du serveur ne fait qu'injecter les étapes.
package scoringrevision

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// Compteurs expvar (ADR 0009), par titre ; une issue par étape et par joueur.
const (
	CompteurAppliquee = "scoring_revision_appliquee_total"
	CompteurAJour     = "scoring_revision_a_jour_total"
	CompteurErreurs   = "scoring_revision_erreurs_total"
	// Issues de l'étape de la base partagée, une par titre et par démarrage.
	CompteurPartageeAppliquee = "scoring_revision_base_partagee_appliquee_total"
	CompteurPartageeAJour     = "scoring_revision_base_partagee_a_jour_total"
	CompteurPartageeErreurs   = "scoring_revision_base_partagee_erreurs_total"
	CompteurPartageePartielle = "scoring_revision_base_partagee_partielle_total"
)

// Step nomme une étape du passage par joueur ; chaque étape a sa propre révision rangée dans `sync_meta`,
// réussit ou échoue indépendamment des autres.
type Step string

const (
	// StepShared : le rattrapage de la base partagée du titre (coup d'envoi et temps de jeu).
	StepShared Step = "base_partagee"
	// StepPlayerDB : l'accès à la base du joueur, avant toute étape (fichier illisible).
	StepPlayerDB Step = "base_joueur"
	// StepKPI : rejeu des notes de performance et du LUSR après un changement de formule.
	StepKPI Step = "kpi"
	// StepCampaigns : recalage du départ des campagnes de l'axe Objectif après un changement de sa mesure.
	StepCampaigns Step = "campagnes"
)

// Outcome est l'issue d'une étape pour un joueur : recalcul effectué (Applied), à jour (ni Applied ni
// Err) ou échec (Err). Une étape en échec garde sa révision rangée et est retentée au démarrage suivant ;
// les étapes réussies du même joueur ne le sont pas.
type Outcome struct {
	Step    Step
	Applied bool
	Err     error
}

// Service : le passage unique de rejeu.
type Service struct {
	players   port.ScoringRevisionPlayers
	shared    port.SharedRevisionStep
	kpi       port.ScoringRevisionStep
	campaigns port.ScoringRevisionStep
	titres    func() []string
}

// New construit le service ; [Service.Run] le fait travailler sur les titres du registre qui
// portent la capacité LUSR. `shared` est l'étape de la base partagée du titre, jouée d'abord ; `kpi` et
// `campaigns` sont les deux étapes par joueur, jouées ensuite dans cet ordre.
func New(players port.ScoringRevisionPlayers, shared port.SharedRevisionStep, kpi, campaigns port.ScoringRevisionStep) *Service {
	return &Service{
		players: players, shared: shared, kpi: kpi, campaigns: campaigns,
		titres: func() []string { return titresLUSR(title.DefaultRegistry().All()) },
	}
}

// titresLUSR rend les titres qui portent la capacité LUSR, dans l'ordre du registre.
func titresLUSR(tous []*title.TitleDescriptor) []string {
	var slugs []string
	for _, t := range tous {
		if t.HasCapability(title.CapLUSR) {
			slugs = append(slugs, t.Slug)
		}
	}
	return slugs
}

// Run passe en revue chaque joueur de chaque titre, un à la fois, et s'arrête entre deux joueurs
// dès que ctx est annulé. Une liste de joueurs illisible saute le titre ; les profils réservés aux
// jetons (AuthOnly) n'ont pas de base joueur. Un profil sans xuid est sauté et compté en erreur : le
// recalcul se joint sur le xuid, il n'y trouverait aucun match et effacerait toutes les notes.
func (s *Service) Run(ctx context.Context) {
	for _, slug := range s.titres() {
		if ctx.Err() != nil {
			return
		}
		if !s.rattraperBasePartagee(ctx, slug) {
			continue
		}
		players, err := s.players.FollowedPlayers(slug)
		if err != nil {
			observability.IncCounterT(slug, CompteurErreurs)
			slog.WarnContext(ctx, "scoring revision: liste des joueurs illisible, titre sauté",
				"titleSlug", slug, "err", err)
			continue
		}
		for _, p := range players {
			if ctx.Err() != nil {
				return
			}
			if p.AuthOnly {
				continue
			}
			if p.XUID == "" {
				observability.IncCounterT(slug, CompteurErreurs)
				slog.WarnContext(ctx, "scoring revision: profil sans xuid, notes laissées telles quelles",
					"player", p.Gamertag, "titleSlug", slug)
				continue
			}
			s.appliquer(ctx, slug, p)
		}
	}
}

// rattraperBasePartagee joue l'étape de la base partagée du titre et dit si les joueurs peuvent suivre.
func (s *Service) rattraperBasePartagee(ctx context.Context, slug string) bool {
	applied, err := s.shared.Apply(ctx, slug)
	switch {
	case errors.Is(err, port.ErrSharedRevisionPartial):
		observability.IncCounterT(slug, CompteurPartageePartielle)
		slog.ErrorContext(ctx, "scoring revision: shared catch-up partial, revision not stored, player notes still run",
			"step", string(StepShared), "err", err, "titleSlug", slug)
		return true
	case err != nil:
		observability.IncCounterT(slug, CompteurPartageeErreurs)
		slog.WarnContext(ctx, "scoring revision: rattrapage de la base partagée échoué, joueurs du titre retentés au prochain démarrage",
			"step", string(StepShared), "err", err, "titleSlug", slug)
		return false
	case applied:
		observability.IncCounterT(slug, CompteurPartageeAppliquee)
		slog.InfoContext(ctx, "scoring revision: base partagée rattrapée après changement de révision",
			"step", string(StepShared), "titleSlug", slug)
	default:
		observability.IncCounterT(slug, CompteurPartageeAJour)
	}
	return true
}

// appliquer applique les étapes d'un joueur et compte l'issue de CHACUNE : une étape réussie compte
// comme appliquée même si une autre du même joueur échoue (elle a rangé sa révision et ne sera pas
// rejouée), et un échec nomme son étape dans le journal, puisque seule celle-là est retentée.
func (s *Service) appliquer(ctx context.Context, slug string, p domain.PlayerSummary) {
	for _, o := range s.etapesDuJoueur(ctx, slug, p) {
		switch {
		case o.Err != nil:
			observability.IncCounterT(slug, CompteurErreurs)
			slog.WarnContext(ctx, "scoring revision: étape échouée, retentée au prochain démarrage",
				"step", string(o.Step), "err", o.Err, "player", p.Gamertag, "titleSlug", slug)
		case o.Applied:
			observability.IncCounterT(slug, CompteurAppliquee)
			slog.InfoContext(ctx, "scoring revision: étape appliquée après changement de formule ou de mesure",
				"step", string(o.Step), "player", p.Gamertag, "titleSlug", slug)
		default:
			observability.IncCounterT(slug, CompteurAJour)
		}
	}
}

// etapesDuJoueur joue les deux étapes du joueur et rend l'issue de chacune, dans l'ordre : notes (rejeu des
// KPI si sa révision de formule rangée diffère de celle du binaire), puis campagnes (départ de l'axe
// Objectif recalé si sa révision de mesure rangée diffère). Les étapes sont indépendantes : l'échec de
// l'une ne retient pas l'autre, et chacune garde sa révision rangée tant qu'elle n'a pas réussi. Un
// profil sans base joueur (jamais synchronisé) n'a rien à recalculer : il n'est pas créé et n'a aucune
// issue ; une base illisible rend une seule issue en échec, avant toute étape.
func (s *Service) etapesDuJoueur(ctx context.Context, slug string, p domain.PlayerSummary) []Outcome {
	present, err := s.players.HasPlayerDB(slug, p)
	if err != nil {
		return []Outcome{{Step: StepPlayerDB, Err: fmt.Errorf("base joueur illisible: %w", err)}}
	}
	if !present {
		slog.InfoContext(ctx, "scoring revision: profil sans base joueur, rien à recalculer",
			"player", p.Gamertag, "titleSlug", slug)
		return nil
	}
	kpi, kpiErr := s.kpi.Apply(ctx, slug, p)
	campaigns, campaignErr := s.campaigns.Apply(ctx, slug, p)
	return []Outcome{
		{Step: StepKPI, Applied: kpi, Err: kpiErr},
		{Step: StepCampaigns, Applied: campaigns, Err: campaignErr},
	}
}
