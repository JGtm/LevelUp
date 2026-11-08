package scoringrevision

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// fauxApplier : le port du rejeu, sans base.
type fauxApplier struct {
	joueurs    map[string][]domain.PlayerSummary // par titre
	listeErr   map[string]error                  // par titre
	appliquee  map[string]bool                   // par xuid
	errs       map[string]error                  // par xuid
	appels     []string                          // "titre/xuid"
	apresAppel func()
}

func (f *fauxApplier) FollowedPlayers(slug string) ([]domain.PlayerSummary, error) {
	return f.joueurs[slug], f.listeErr[slug]
}

func (f *fauxApplier) ApplyForPlayer(_ context.Context, slug string, p domain.PlayerSummary) (bool, error) {
	f.appels = append(f.appels, slug+"/"+p.XUID)
	if f.apresAppel != nil {
		f.apresAppel()
	}
	return f.appliquee[p.XUID], f.errs[p.XUID]
}

func joueur(xuid string) domain.PlayerSummary {
	return domain.PlayerSummary{Gamertag: "gt-" + xuid, XUID: xuid}
}

func nouveau(f *fauxApplier, titres ...string) *Service {
	return &Service{a: f, titres: func() []string { return titres }}
}

// Un joueur à jour, un joueur recalculé : un appel chacun, chaque issue comptée sur son titre.
func TestRun_AJourEtAppliquee(t *testing.T) {
	f := &fauxApplier{
		joueurs:   map[string][]domain.PlayerSummary{"sr-a": {joueur("x1"), joueur("x2")}},
		appliquee: map[string]bool{"x2": true},
	}
	appliqueeAvant := observability.LoadCounterT("sr-a", CompteurAppliquee)
	aJourAvant := observability.LoadCounterT("sr-a", CompteurAJour)

	nouveau(f, "sr-a").Run(context.Background())

	if len(f.appels) != 2 || f.appels[0] != "sr-a/x1" || f.appels[1] != "sr-a/x2" {
		t.Fatalf("appels = %v, attendu les deux joueurs dans l'ordre", f.appels)
	}
	if got := observability.LoadCounterT("sr-a", CompteurAppliquee) - appliqueeAvant; got != 1 {
		t.Errorf("appliquées : +%d, attendu +1", got)
	}
	if got := observability.LoadCounterT("sr-a", CompteurAJour) - aJourAvant; got != 1 {
		t.Errorf("à jour : +%d, attendu +1", got)
	}
}

// Une erreur est comptée et journalisée, la boucle passe au joueur suivant ; une liste de joueurs
// illisible saute le titre sans arrêter les autres.
func TestRun_UneErreurNArretePasLaBoucle(t *testing.T) {
	f := &fauxApplier{
		joueurs: map[string][]domain.PlayerSummary{
			"sr-b": {joueur("x1"), joueur("x2")},
			"sr-d": {joueur("x3")},
		},
		listeErr:  map[string]error{"sr-c": errors.New("profils illisibles")},
		errs:      map[string]error{"x1": errors.New("base illisible")},
		appliquee: map[string]bool{"x2": true, "x3": true},
	}
	erreursB := observability.LoadCounterT("sr-b", CompteurErreurs)
	erreursC := observability.LoadCounterT("sr-c", CompteurErreurs)

	nouveau(f, "sr-b", "sr-c", "sr-d").Run(context.Background())

	want := []string{"sr-b/x1", "sr-b/x2", "sr-d/x3"}
	if len(f.appels) != len(want) {
		t.Fatalf("appels = %v, attendu %v", f.appels, want)
	}
	for i := range want {
		if f.appels[i] != want[i] {
			t.Fatalf("appels = %v, attendu %v", f.appels, want)
		}
	}
	if got := observability.LoadCounterT("sr-b", CompteurErreurs) - erreursB; got != 1 {
		t.Errorf("erreurs du titre sr-b : +%d, attendu +1", got)
	}
	if got := observability.LoadCounterT("sr-c", CompteurErreurs) - erreursC; got != 1 {
		t.Errorf("erreurs du titre sr-c (liste illisible) : +%d, attendu +1", got)
	}
}

// Les profils réservés aux jetons n'ont pas de base joueur : ils ne sont pas rejoués.
func TestRun_ProfilAuthOnlyIgnore(t *testing.T) {
	jeton := joueur("x2")
	jeton.AuthOnly = true
	f := &fauxApplier{joueurs: map[string][]domain.PlayerSummary{"sr-e": {joueur("x1"), jeton}}}
	nouveau(f, "sr-e").Run(context.Background())
	if len(f.appels) != 1 || f.appels[0] != "sr-e/x1" {
		t.Fatalf("appels = %v, attendu le seul profil à base joueur", f.appels)
	}
}

// Seuls les titres qui portent la capacité LUSR sont passés en revue.
func TestTitresLUSR_SauteUnTitreSansCapacite(t *testing.T) {
	tous := []*title.TitleDescriptor{
		{Slug: "avec", Capabilities: []title.Capability{title.CapMedia, title.CapLUSR}},
		{Slug: "sans", Capabilities: []title.Capability{title.CapMedia}},
		{Slug: "aussi", Capabilities: []title.Capability{title.CapLUSR}},
	}
	got := titresLUSR(tous)
	if len(got) != 2 || got[0] != "avec" || got[1] != "aussi" {
		t.Fatalf("titresLUSR = %v, attendu [avec aussi]", got)
	}
}

// Le service du registre ne passe en revue que des titres qui portent la capacité LUSR.
func TestNew_TitresDuRegistreOntLaCapaciteLUSR(t *testing.T) {
	for _, slug := range New(&fauxApplier{}).titres() {
		d := title.DefaultRegistry().Get(slug)
		if d == nil || !d.HasCapability(title.CapLUSR) {
			t.Errorf("titre %q passé en revue sans capacité LUSR", slug)
		}
	}
}

// L'annulation du contexte arrête la boucle entre deux joueurs, et empêche les titres suivants.
func TestRun_AnnulationArreteEntreDeuxJoueurs(t *testing.T) {
	ctx, annuler := context.WithCancel(context.Background())
	f := &fauxApplier{
		joueurs: map[string][]domain.PlayerSummary{
			"sr-f": {joueur("x1"), joueur("x2")},
			"sr-g": {joueur("x3")},
		},
		apresAppel: annuler,
	}
	nouveau(f, "sr-f", "sr-g").Run(ctx)
	if len(f.appels) != 1 || f.appels[0] != "sr-f/x1" {
		t.Fatalf("appels = %v, attendu un seul appel avant l'arrêt", f.appels)
	}
}
