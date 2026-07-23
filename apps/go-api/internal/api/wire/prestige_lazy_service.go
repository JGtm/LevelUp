// Package api — adaptateur "lazy" qui résout un prestige.Service
// par requête en s'appuyant sur le PrestigeBundle.
//
// Ce wrapper permet de passer une instance unique au PrestigeHandler tout en
// résolvant le PlayerDB à la demande — chaque méthode reçoit le user_id en
// paramètre, qui sert de clé pour ouvrir/réutiliser la connexion player.

package wire

import (
	"context"
	"errors"
	"net/http"
	"time"

	"levelup/go-api/internal/platform/dblease"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/prestige"
)

// playerSlugCtxKey extrait le player_slug d'un context HTTP.
//
// Phase 4 : on s'appuie sur le user_id passé en query/body. À terme,
// un middleware Auth pourra le mettre dans le context (clé identique à
// celle utilisée par TitleExtractor pour title_slug).
type playerSlugCtxKey struct{}

// WithPlayerSlug attache le player_slug au context (utilisable par middleware).
func WithPlayerSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, playerSlugCtxKey{}, slug)
}

// PlayerSlugFromContext extrait le player_slug, retourne "" si absent.
func PlayerSlugFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(playerSlugCtxKey{}).(string); ok {
		return v
	}
	return ""
}

// LazyPrestigeService implémente prestige.Service en résolvant le Service
// à chaque appel via le bundle.
//
// Le player_slug est extrait depuis :
//  1. la fonction extractFn fournie (lit body/query/context)
//  2. en fallback, retourne ErrPlayerNotResolved
type LazyPrestigeService struct {
	bundle    *PrestigeBundle
	extractFn func(ctx context.Context) string
	// demoMode : en démo le DemoPlayer n'a aucune activité PP réelle → GetUserPrestige
	// sert une fixture (rang Prestige) au lieu d'un prestige vide. Cf. demoUserPrestige.
	demoMode bool
}

// NewLazyPrestigeService construit un wrapper.
//
// extractFn est typiquement une closure qui lit le request en cours via
// un middleware (chi context value). Si elle retourne "", les méthodes
// retournent ErrPlayerNotResolved.
func NewLazyPrestigeService(bundle *PrestigeBundle, extractFn func(ctx context.Context) string, demoMode bool) *LazyPrestigeService {
	if extractFn == nil {
		extractFn = PlayerSlugFromContext
	}
	return &LazyPrestigeService{bundle: bundle, extractFn: extractFn, demoMode: demoMode}
}

// ErrPlayerNotResolved est retournée quand le player_slug ne peut pas être
// extrait du context — l'appelant doit le fournir via middleware ou body.
var ErrPlayerNotResolved = errors.New("prestige: player_slug missing from context")

// resolve charge le Service pour le player courant ; retourne err si introuvable.
func (l *LazyPrestigeService) resolve(ctx context.Context) (prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		return nil, ErrPlayerNotResolved
	}
	return l.bundle.ServiceForPlayer(ctx, slug)
}

// resolveByUserID est un fallback : si le contexte ne contient pas de slug
// mais qu'un userID est fourni explicitement (ex. depuis le body), on l'utilise.
func (l *LazyPrestigeService) resolveByUserID(ctx context.Context, userID string) (prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		slug = userID
	}
	if slug == "" {
		return nil, ErrPlayerNotResolved
	}
	return l.bundle.ServiceForPlayer(ctx, slug)
}

// resolveWithPlayerDB retourne le service ET le *PlayerDB associé — utile pour
// les méthodes write qui doivent acquérir un *LeasedWriter avant délégation.
// Les méthodes read continuent d'utiliser resolve() / resolveByUserID().
func (l *LazyPrestigeService) resolveWithPlayerDB(ctx context.Context) (*platform_duckdb.PlayerDB, prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		return nil, nil, ErrPlayerNotResolved
	}
	return l.bundle.serviceAndPlayerDB(ctx, slug)
}

// resolveWithPlayerDBByUserID idem, avec fallback userID.
func (l *LazyPrestigeService) resolveWithPlayerDBByUserID(ctx context.Context, userID string) (*platform_duckdb.PlayerDB, prestige.Service, error) {
	slug := l.extractFn(ctx)
	if slug == "" {
		slug = userID
	}
	if slug == "" {
		return nil, nil, ErrPlayerNotResolved
	}
	return l.bundle.serviceAndPlayerDB(ctx, slug)
}

// acquirePlayerWriter encapsule le pattern d'acquisition pour les méthodes
// write Prestige côté player DB. Retourne ErrDBLocked si le sync engine ou
// un autre handler tient déjà le lease — les handlers HTTP mappent en 503.
func acquirePlayerWriter(pdb *platform_duckdb.PlayerDB) (*dblease.LeasedWriter, error) {
	return pdb.AcquirePlayerWriterTimeout(dblease.PlayerLeaseTimeout)
}

// acquireSharedSocialWriter encapsule l'acquisition du lease sur
// shared_social.duckdb (squad challenges, prestige events / user_prestige).
// La DB étant partagée entre tous les joueurs, le lease sérialise les writes
// inter-joueurs aussi.
func acquireSharedSocialWriter(pdb *platform_duckdb.PlayerDB) (*dblease.LeasedWriter, error) {
	return pdb.AcquireSharedSocialWriterTimeout(dblease.SharedLeaseTimeout)
}

// ─── Compile-time assertion ───
var _ prestige.Service = (*LazyPrestigeService)(nil)

// ─── Méthodes Service (délégation) ───

func (l *LazyPrestigeService) CreateChallenge(ctx context.Context, req prestige.CreateChallengeRequest) (prestige.Challenge, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, req.UserID)
	if err != nil {
		return prestige.Challenge{}, err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return prestige.Challenge{}, err
	}
	defer w.Release()
	return svc.CreateChallenge(ctx, req)
}

func (l *LazyPrestigeService) UpdateChallenge(ctx context.Context, id string, patch prestige.UpdateChallengePatch) (prestige.Challenge, error) {
	pdb, svc, err := l.resolveWithPlayerDB(ctx)
	if err != nil {
		return prestige.Challenge{}, err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return prestige.Challenge{}, err
	}
	defer w.Release()
	return svc.UpdateChallenge(ctx, id, patch)
}

func (l *LazyPrestigeService) AbandonChallenge(ctx context.Context, id string) error {
	pdb, svc, err := l.resolveWithPlayerDB(ctx)
	if err != nil {
		return err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.AbandonChallenge(ctx, id)
}

func (l *LazyPrestigeService) GetChallenge(ctx context.Context, id string) (prestige.Challenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Challenge{}, err
	}
	return svc.GetChallenge(ctx, id)
}

func (l *LazyPrestigeService) ListActiveChallenges(ctx context.Context, userID, titleSlug string) ([]prestige.Challenge, error) {
	if l.demoMode {
		return demoActiveChallenges(userID, titleSlug), nil
	}
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListActiveChallenges(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) ListChallenges(ctx context.Context, userID, titleSlug string, statuses []prestige.ChallengeStatus) ([]prestige.Challenge, error) {
	if l.demoMode {
		// Le mode démo n'a pas d'historique : ne sert que les défis actifs de
		// démonstration si le filtre inclut le statut actif, sinon rien.
		for _, st := range statuses {
			if st == prestige.StatusActive {
				return demoActiveChallenges(userID, titleSlug), nil
			}
		}
		return nil, nil
	}
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListChallenges(ctx, userID, titleSlug, statuses)
}

func (l *LazyPrestigeService) EvaluateForUser(ctx context.Context, userID, titleSlug string) ([]prestige.EvaluationOutcome, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.EvaluateForUser(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) GetUserPrestige(ctx context.Context, userID, titleSlug string) (prestige.UserPrestige, error) {
	if l.demoMode {
		return demoUserPrestige(userID, titleSlug), nil
	}
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return prestige.UserPrestige{}, err
	}
	return svc.GetUserPrestige(ctx, userID, titleSlug)
}

// demoUserPrestige : rang Prestige fixé pour la démo (le DemoPlayer n'a pas
// d'activité PP réelle). Niveau "Vétéran" (cf. tuning.go Names), progression
// plausible. Read-time → survit reseed + déploiement.
func demoUserPrestige(userID, titleSlug string) prestige.UserPrestige {
	return prestige.UserPrestige{
		UserID:       userID,
		TitleSlug:    titleSlug,
		TotalPP:      2100,
		CurrentLevel: 2,
		UpdatedAt:    time.Now().UTC(),
		Level: &prestige.Level{
			Index:           2,
			Name:            "Vétéran",
			ThresholdPP:     1500,
			NextThresholdPP: 3000,
			ProgressRatio:   0.4,
		},
	}
}

const demoArcID = "demo-arc-ascension"

// demoArcs : un arc Prestige démo (la section Prestige du Home affiche l'arc actif).
//
// ObjectivesPP / CompletionBonusPP sont calculés depuis les objectifs démo (mêmes
// règles que l'enrichissement read-time réel) pour que la démo illustre la
// distinction "PP cumulés des objectifs" vs "bonus de complétion d'arc".
func demoArcs(userID, titleSlug string) []prestige.Arc {
	tuning := prestige.DefaultTuning()
	objectivesPP := 0
	for _, c := range demoActiveChallenges(userID, titleSlug) {
		objectivesPP += c.PPReward
	}
	return []prestige.Arc{{
		ID:                demoArcID,
		UserID:            userID,
		TitleSlug:         titleSlug,
		Title:             "Ascension du Spartan",
		Description:       "Enchaîne les objectifs pour gravir les paliers de prestige.",
		IsPreset:          true,
		PresetID:          "ascension",
		CreatedAt:         time.Now().UTC().Add(-14 * 24 * time.Hour),
		ObjectivesPP:      objectivesPP,
		CompletionBonusPP: prestige.PPForArcCompletion(tuning, objectivesPP),
	}}
}

// demoActiveChallenges : objectifs Prestige démo (rattachés à l'arc). 3 en cours
// (CurrentValue < Target) + 1 complété, pour que l'arc affiche une progression
// (Étape 1/4) sur le Home et la page Ascension. Read-time → survit reseed.
func demoActiveChallenges(userID, titleSlug string) []prestige.Challenge {
	now := time.Now().UTC()
	exp := now.Add(5 * 24 * time.Hour)
	// PP affiché par objectif démo : tous Héroïque + données complètes (même
	// calcul que l'enrichissement réel ListActiveChallenges → PPForCompletion).
	ppReward := prestige.PPForCompletion(prestige.DefaultTuning(), prestige.TierHeroic, false, prestige.DataFull)
	mk := func(id, label, metric string, pos int, target, current float64, status prestige.ChallengeStatus) prestige.Challenge {
		return prestige.Challenge{
			ID: id, UserID: userID, TitleSlug: titleSlug,
			ArcID: demoArcID, Position: pos,
			Metric: metric, Target: target, CurrentValue: current,
			WindowType: prestige.WindowLastNMatches, WindowValue: "10",
			Cadence: prestige.CadenceWeekly, EvalType: prestige.EvalCumulative,
			Mode: prestige.ModeLibre, Tier: prestige.TierHeroic, DataTier: prestige.DataFull,
			Label: label, Status: status, PPReward: ppReward,
			ExpiresAt: &exp, CreatedAt: now.Add(-3 * 24 * time.Hour),
		}
	}
	return []prestige.Challenge{
		// 1re étape complétée → l'arc démo affiche une progression réelle (Étape 1/4).
		mk("demo-ch-warmup", "Mise en jambe — 10 parties jouées", "matches_played", 0, 10, 10, prestige.StatusCompleted),
		mk("demo-ch-kills", "Tueur d'élite — 100 éliminations", "kills", 1, 100, 67, prestige.StatusActive),
		mk("demo-ch-headshots", "Précision létale — 40 tirs à la tête", "headshot_kills", 2, 40, 28, prestige.StatusActive),
		mk("demo-ch-wins", "Série victorieuse — 8 victoires", "wins", 3, 8, 5, prestige.StatusActive),
	}
}

func (l *LazyPrestigeService) SuggestTemplates(ctx context.Context, userID, titleSlug string, count int) ([]prestige.Template, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.SuggestTemplates(ctx, userID, titleSlug, count)
}

func (l *LazyPrestigeService) SuggestNext(ctx context.Context, completedID string) ([]prestige.Template, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return svc.SuggestNext(ctx, completedID)
}

func (l *LazyPrestigeService) CreateArc(ctx context.Context, req prestige.CreateArcRequest) (prestige.Arc, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, req.UserID)
	if err != nil {
		return prestige.Arc{}, err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return prestige.Arc{}, err
	}
	defer w.Release()
	return svc.CreateArc(ctx, req)
}

func (l *LazyPrestigeService) DeleteArc(ctx context.Context, userID, id string, opts prestige.DeleteArcOptions) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, userID)
	if err != nil {
		return err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.DeleteArc(ctx, userID, id, opts)
}

func (l *LazyPrestigeService) ListArcs(ctx context.Context, userID, titleSlug string) ([]prestige.Arc, error) {
	if l.demoMode {
		return demoArcs(userID, titleSlug), nil
	}
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListArcs(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) GetArc(ctx context.Context, id string) (prestige.Arc, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Arc{}, err
	}
	return svc.GetArc(ctx, id)
}

func (l *LazyPrestigeService) ListArcPresets(ctx context.Context, userID, titleSlug string) ([]prestige.PresetArc, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListArcPresets(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) AdoptPresetArc(ctx context.Context, userID, titleSlug, presetID string) (prestige.Arc, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, userID)
	if err != nil {
		return prestige.Arc{}, err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return prestige.Arc{}, err
	}
	defer w.Release()
	return svc.AdoptPresetArc(ctx, userID, titleSlug, presetID)
}

func (l *LazyPrestigeService) CreateSquadChallenge(ctx context.Context, req prestige.CreateSquadChallengeRequest) (prestige.SquadChallenge, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, req.CreatedBy)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	defer w.Release()
	return svc.CreateSquadChallenge(ctx, req)
}

func (l *LazyPrestigeService) JoinSquadChallenge(ctx context.Context, challengeID, userID string, chosenTier prestige.Tier, isPrivate bool) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, userID)
	if err != nil {
		return err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.JoinSquadChallenge(ctx, challengeID, userID, chosenTier, isPrivate)
}

func (l *LazyPrestigeService) GetSquadChallenge(ctx context.Context, id string) (prestige.SquadChallenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	return svc.GetSquadChallenge(ctx, id)
}

func (l *LazyPrestigeService) ListSquadChallenges(ctx context.Context, squadID, requestedBy string) ([]prestige.SquadChallenge, error) {
	svc, err := l.resolveByUserID(ctx, requestedBy)
	if err != nil {
		return nil, err
	}
	return svc.ListSquadChallenges(ctx, squadID, requestedBy)
}

func (l *LazyPrestigeService) RefreshSquadPool(ctx context.Context, squadID, titleSlug, requestedBy string) ([]prestige.Template, error) {
	svc, err := l.resolveByUserID(ctx, requestedBy)
	if err != nil {
		return nil, err
	}
	return svc.RefreshSquadPool(ctx, squadID, titleSlug, requestedBy)
}

func (l *LazyPrestigeService) CreateSquad(ctx context.Context, req prestige.CreateSquadRequest) (prestige.Squad, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, req.CreatedBy)
	if err != nil {
		return prestige.Squad{}, err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return prestige.Squad{}, err
	}
	defer w.Release()
	return svc.CreateSquad(ctx, req)
}

func (l *LazyPrestigeService) ListSquadsForUser(ctx context.Context, userID string) ([]prestige.Squad, error) {
	svc, err := l.resolveByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return svc.ListSquadsForUser(ctx, userID)
}

func (l *LazyPrestigeService) GetSquad(ctx context.Context, id string) (prestige.Squad, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.Squad{}, err
	}
	return svc.GetSquad(ctx, id)
}

func (l *LazyPrestigeService) ListSquadMembers(ctx context.Context, squadID string) ([]prestige.SquadMember, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return nil, err
	}
	return svc.ListSquadMembers(ctx, squadID)
}

func (l *LazyPrestigeService) AddSquadMember(ctx context.Context, squadID string, member prestige.SquadMember, requestedBy string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.AddSquadMember(ctx, squadID, member, requestedBy)
}

func (l *LazyPrestigeService) RemoveSquadMember(ctx context.Context, squadID, xuid, requestedBy string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.RemoveSquadMember(ctx, squadID, xuid, requestedBy)
}

func (l *LazyPrestigeService) RenameSquad(ctx context.Context, squadID, name, requestedBy string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.RenameSquad(ctx, squadID, name, requestedBy)
}

func (l *LazyPrestigeService) DeleteSquad(ctx context.Context, squadID, requestedBy string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.DeleteSquad(ctx, squadID, requestedBy)
}

func (l *LazyPrestigeService) SquadUsualContexts(ctx context.Context, rosterXUIDs []string, titleSlug string) ([]string, []string, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return nil, nil, err
	}
	return svc.SquadUsualContexts(ctx, rosterXUIDs, titleSlug)
}

func (l *LazyPrestigeService) EvaluateSquadChallenge(ctx context.Context, squadChallengeID, requestedBy string) ([]prestige.SquadParticipantProgress, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return nil, err
	}
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return nil, err
	}
	defer w.Release()
	return svc.EvaluateSquadChallenge(ctx, squadChallengeID, requestedBy)
}

func (l *LazyPrestigeService) SquadOrientation(ctx context.Context, squadID, requestedBy string) (string, error) {
	svc, err := l.resolveByUserID(ctx, requestedBy)
	if err != nil {
		return "", err
	}
	return svc.SquadOrientation(ctx, squadID, requestedBy)
}

func (l *LazyPrestigeService) EnablePilotMode(ctx context.Context, userID, titleSlug string) (prestige.PilotModeAttribution, error) {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, userID)
	if err != nil {
		return prestige.PilotModeAttribution{}, err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return prestige.PilotModeAttribution{}, err
	}
	defer w.Release()
	return svc.EnablePilotMode(ctx, userID, titleSlug)
}

func (l *LazyPrestigeService) DisablePilotMode(ctx context.Context, userID, titleSlug string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, userID)
	if err != nil {
		return err
	}
	w, err := acquirePlayerWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.DisablePilotMode(ctx, userID, titleSlug)
}

// ─── Helper HTTP middleware ───

// PlayerSlugFromQueryOrBody extrait player_slug d'une request HTTP via :
//  1. query param "user_id"
//  2. en fallback, le slug "halo_infinite" du DefaultSlug si ?title_slug indique
//
// Pour le body, l'extraction se fait ailleurs (avant de wrapper le ctx).
// Phase 4 : implémentation pragmatique, à étendre avec un middleware Auth
// quand l'identification utilisateur sera robuste.
func PlayerSlugFromQueryOrBody(r *http.Request) string {
	if v := r.URL.Query().Get("user_id"); v != "" {
		return v
	}
	return ""
}
