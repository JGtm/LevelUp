package wire

// prestige_lazy_service_squad.go — les méthodes ESCOUADE de LazyPrestigeService (défis
// d'escouade, membres, contextes usuels, orientation). Scindé de prestige_lazy_service.go
// sous le seuil des 500 lignes (CLAUDE.md n 5) ; même contrat de délégation : résoudre le
// service du joueur, prendre le writer requis par l'écriture, déléguer.

import (
	"context"

	"levelup/go-api/internal/prestige"
)

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

func (l *LazyPrestigeService) AbandonSquadChallenge(ctx context.Context, challengeID, requestedBy string) error {
	pdb, svc, err := l.resolveWithPlayerDBByUserID(ctx, requestedBy)
	if err != nil {
		return err
	}
	// Écrit dans shared_social (UPDATE squad_challenge.archived_at) → writer requis.
	w, err := acquireSharedSocialWriter(pdb)
	if err != nil {
		return err
	}
	defer w.Release()
	return svc.AbandonSquadChallenge(ctx, challengeID, requestedBy)
}

func (l *LazyPrestigeService) GetSquadChallenge(ctx context.Context, id string) (prestige.SquadChallenge, error) {
	svc, err := l.resolve(ctx)
	if err != nil {
		return prestige.SquadChallenge{}, err
	}
	return svc.GetSquadChallenge(ctx, id)
}

func (l *LazyPrestigeService) ListSquadChallenges(ctx context.Context, squadID, requestedBy string) ([]prestige.SquadChallengeView, error) {
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

// SquadUsualContexts lit l'indice d'escouade dans la base partagée du PlayerDB résolu : le
// titre est donc CELUI DE CE PlayerDB (pdb.TitleSlug), jamais celui de l'appelant (lot
// B-C8 du backlog 2026-09-26). Le web appelle GET /squads sans title_slug : le handler
// transmettait "", et l'exclusion de la Campagne du titre (D-5) ne s'appliquait pas. Un
// titre d'appelant différent de celui de la base lue appliquerait, lui, la mauvaise
// exclusion : il n'a pas voix ici.
func (l *LazyPrestigeService) SquadUsualContexts(ctx context.Context, rosterXUIDs []string, _ string) ([]string, []string, error) {
	pdb, svc, err := l.resolveWithPlayerDB(ctx)
	if err != nil {
		return nil, nil, err
	}
	return svc.SquadUsualContexts(ctx, rosterXUIDs, pdb.TitleSlug)
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
