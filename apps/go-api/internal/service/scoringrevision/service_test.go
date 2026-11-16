package scoringrevision

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// faux : les profils et les deux étapes du rejeu, sans base. Par défaut chaque étape rend « à jour » ; les
// issues d'une étape viennent de kpiAppliquee / kpiErrs et campagnesAppliquee / campagnesErrs (par xuid).
type faux struct {
	joueurs  map[string][]domain.PlayerSummary // par titre
	listeErr map[string]error                  // par titre
	sansBase map[string]bool                   // par xuid : profil jamais synchronisé
	baseErr  map[string]error                  // par xuid : base joueur illisible

	kpiAppliquee, campagnesAppliquee map[string]bool  // par xuid
	kpiErrs, campagnesErrs           map[string]error // par xuid

	appels     []string // "titre/xuid", un par passage de l'étape des notes
	journal    []string // "kpi" / "campagnes", dans l'ordre des appels
	apresAppel func()
}

func (f *faux) FollowedPlayers(slug string) ([]domain.PlayerSummary, error) {
	return f.joueurs[slug], f.listeErr[slug]
}

func (f *faux) HasPlayerDB(_ string, p domain.PlayerSummary) (bool, error) {
	if err := f.baseErr[p.XUID]; err != nil {
		return false, err
	}
	return !f.sansBase[p.XUID], nil
}

// etape : une étape dont l'issue vient des tables du faux.
type etape struct {
	f         *faux
	nom       string
	appliquee map[string]bool
	errs      map[string]error
}

func (e etape) Apply(_ context.Context, slug string, p domain.PlayerSummary) (bool, error) {
	e.f.journal = append(e.f.journal, e.nom)
	if e.nom == "kpi" {
		e.f.appels = append(e.f.appels, slug+"/"+p.XUID)
		if e.f.apresAppel != nil {
			e.f.apresAppel()
		}
	}
	return e.appliquee[p.XUID], e.errs[p.XUID]
}

func joueur(xuid string) domain.PlayerSummary {
	return domain.PlayerSummary{Gamertag: "gt-" + xuid, XUID: xuid}
}

func nouveau(f *faux, titres ...string) *Service {
	return &Service{
		players:   f,
		kpi:       etape{f: f, nom: "kpi", appliquee: f.kpiAppliquee, errs: f.kpiErrs},
		campaigns: etape{f: f, nom: "campagnes", appliquee: f.campagnesAppliquee, errs: f.campagnesErrs},
		titres:    func() []string { return titres },
	}
}

// Un joueur à jour, un joueur recalculé : un appel chacun, chaque issue comptée sur son titre.
func TestRun_AJourEtAppliquee(t *testing.T) {
	f := &faux{
		joueurs:      map[string][]domain.PlayerSummary{"sr-a": {joueur("x1"), joueur("x2")}},
		kpiAppliquee: map[string]bool{"x2": true},
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
	if got := observability.LoadCounterT("sr-a", CompteurAJour) - aJourAvant; got != 3 {
		t.Errorf("à jour : +%d, attendu +3 (deux étapes du premier joueur, l'étape des campagnes du second)", got)
	}
}

// Une erreur est comptée et journalisée, la boucle passe au joueur suivant ; une liste de joueurs
// illisible saute le titre sans arrêter les autres.
func TestRun_UneErreurNArretePasLaBoucle(t *testing.T) {
	f := &faux{
		joueurs: map[string][]domain.PlayerSummary{
			"sr-b": {joueur("x1"), joueur("x2")},
			"sr-d": {joueur("x3")},
		},
		listeErr:     map[string]error{"sr-c": errors.New("profils illisibles")},
		kpiErrs:      map[string]error{"x1": errors.New("rejeu refusé")},
		kpiAppliquee: map[string]bool{"x2": true, "x3": true},
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
	f := &faux{joueurs: map[string][]domain.PlayerSummary{"sr-e": {joueur("x1"), jeton}}}
	nouveau(f, "sr-e").Run(context.Background())
	if len(f.appels) != 1 || f.appels[0] != "sr-e/x1" {
		t.Fatalf("appels = %v, attendu le seul profil à base joueur", f.appels)
	}
}

// Un profil sans xuid n'est jamais rejoué : le recalcul viderait ses notes. Il est compté en erreur
// et ne retient pas le joueur suivant.
func TestRun_ProfilSansXuidIgnore(t *testing.T) {
	sansXuid := joueur("")
	f := &faux{joueurs: map[string][]domain.PlayerSummary{"sr-f": {sansXuid, joueur("x1")}}}
	erreursAvant := observability.LoadCounterT("sr-f", CompteurErreurs)

	nouveau(f, "sr-f").Run(context.Background())

	if len(f.appels) != 1 || f.appels[0] != "sr-f/x1" {
		t.Fatalf("appels = %v, attendu le seul profil avec xuid", f.appels)
	}
	if got := observability.LoadCounterT("sr-f", CompteurErreurs) - erreursAvant; got != 1 {
		t.Errorf("erreurs : +%d, attendu +1", got)
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
	for _, slug := range New(&faux{}, nil, nil).titres() {
		d := title.DefaultRegistry().Get(slug)
		if d == nil || !d.HasCapability(title.CapLUSR) {
			t.Errorf("titre %q passé en revue sans capacité LUSR", slug)
		}
	}
}

// L'annulation du contexte arrête la boucle entre deux joueurs, et empêche les titres suivants.
func TestRun_AnnulationArreteEntreDeuxJoueurs(t *testing.T) {
	ctx, annuler := context.WithCancel(context.Background())
	f := &faux{
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

// capteurJournal remplace le journal par défaut le temps d'un test et rend ce qui y a été écrit.
func capteurJournal(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// Échec partiel : les notes réussissent (révision rangée), les campagnes échouent. L'étape réussie compte
// comme appliquée, l'échec compte en erreur et le journal nomme l'étape qui sera retentée : les notes ne
// seront pas rejouées, seules les campagnes le seront.
func TestRun_EchecPartielNommeLEtapeEnEchec(t *testing.T) {
	f := &faux{
		joueurs:       map[string][]domain.PlayerSummary{"sr-h": {joueur("x1")}},
		kpiAppliquee:  map[string]bool{"x1": true},
		campagnesErrs: map[string]error{"x1": errors.New("lease refusé")},
	}
	journal := capteurJournal(t)
	appliqueeAvant := observability.LoadCounterT("sr-h", CompteurAppliquee)
	erreursAvant := observability.LoadCounterT("sr-h", CompteurErreurs)

	nouveau(f, "sr-h").Run(context.Background())

	if got := observability.LoadCounterT("sr-h", CompteurAppliquee) - appliqueeAvant; got != 1 {
		t.Errorf("appliquées : +%d, attendu +1 (l'étape des notes a réussi)", got)
	}
	if got := observability.LoadCounterT("sr-h", CompteurErreurs) - erreursAvant; got != 1 {
		t.Errorf("erreurs : +%d, attendu +1 (l'étape des campagnes a échoué)", got)
	}
	logs := journal.String()
	if !strings.Contains(logs, "step=campagnes") || !strings.Contains(logs, "étape échouée") {
		t.Errorf("le journal ne nomme pas l'étape en échec :\n%s", logs)
	}
	if !strings.Contains(logs, "step=kpi") || !strings.Contains(logs, "étape appliquée") {
		t.Errorf("le journal ne nomme pas l'étape réussie :\n%s", logs)
	}
}

// Les deux étapes sont jouées dans l'ordre : notes, puis campagnes, et chaque issue porte son étape.
func TestEtapesDuJoueur_NotesPuisCampagnes(t *testing.T) {
	f := &faux{kpiAppliquee: map[string]bool{"x1": true}, campagnesAppliquee: map[string]bool{"x1": false}}
	got := nouveau(f, "sr-i").etapesDuJoueur(context.Background(), "sr-i", joueur("x1"))

	if len(f.journal) != 2 || f.journal[0] != "kpi" || f.journal[1] != "campagnes" {
		t.Fatalf("étapes jouées = %v, attendu kpi puis campagnes", f.journal)
	}
	attendreIssues(t, got, []Outcome{{Step: StepKPI, Applied: true}, {Step: StepCampaigns}})
}

// L'étape des notes échoue : celle des campagnes tourne quand même et son issue est rendue à part.
func TestEtapesDuJoueur_EchecDesNotesNeRetientPasLesCampagnes(t *testing.T) {
	boom := errors.New("rejeu refusé")
	f := &faux{kpiErrs: map[string]error{"x1": boom}, campagnesAppliquee: map[string]bool{"x1": true}}
	got := nouveau(f, "sr-i").etapesDuJoueur(context.Background(), "sr-i", joueur("x1"))

	if len(f.journal) != 2 {
		t.Fatalf("étapes jouées = %v, attendu les deux", f.journal)
	}
	attendreIssues(t, got, []Outcome{{Step: StepKPI, Err: boom}, {Step: StepCampaigns, Applied: true}})
}

// L'étape des campagnes échoue : celle des notes a tourné et reste appliquée.
func TestEtapesDuJoueur_EchecDesCampagnesNeDefaitPasLesNotes(t *testing.T) {
	boom := errors.New("lease refusé")
	f := &faux{kpiAppliquee: map[string]bool{"x1": true}, campagnesErrs: map[string]error{"x1": boom}}
	got := nouveau(f, "sr-i").etapesDuJoueur(context.Background(), "sr-i", joueur("x1"))

	if len(f.journal) != 2 || f.journal[0] != "kpi" || f.journal[1] != "campagnes" {
		t.Fatalf("étapes jouées = %v, attendu kpi puis campagnes", f.journal)
	}
	attendreIssues(t, got, []Outcome{{Step: StepKPI, Applied: true}, {Step: StepCampaigns, Err: boom}})
}

// Un profil sans base joueur n'a rien à recalculer : aucune étape, aucune issue.
func TestEtapesDuJoueur_ProfilSansBaseJoueurSaute(t *testing.T) {
	f := &faux{sansBase: map[string]bool{"x1": true}}
	got := nouveau(f, "sr-i").etapesDuJoueur(context.Background(), "sr-i", joueur("x1"))
	if len(got) != 0 || len(f.journal) != 0 {
		t.Fatalf("issues = %+v, étapes jouées = %v, attendu ni l'un ni l'autre", got, f.journal)
	}
}

// Une base joueur illisible rend une seule issue en échec, avant toute étape.
func TestEtapesDuJoueur_BaseIllisible(t *testing.T) {
	boom := errors.New("accès refusé")
	f := &faux{baseErr: map[string]error{"x1": boom}}
	got := nouveau(f, "sr-i").etapesDuJoueur(context.Background(), "sr-i", joueur("x1"))
	if len(f.journal) != 0 {
		t.Fatalf("étapes jouées = %v, attendu aucune", f.journal)
	}
	attendreIssues(t, got, []Outcome{{Step: StepPlayerDB, Err: boom}})
}

func attendreIssues(t *testing.T, got []Outcome, want []Outcome) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("issues = %+v, attendu %+v", got, want)
	}
	for i := range want {
		if got[i].Step != want[i].Step || got[i].Applied != want[i].Applied || !errors.Is(got[i].Err, want[i].Err) {
			t.Fatalf("issue %d = %+v, attendu %+v", i, got[i], want[i])
		}
	}
}
