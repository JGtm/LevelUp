package service

// home_service_fournisseur_test.go — le fournisseur de lecture en direct de la passe de combat et des
// défis est injecté au constructeur : un titre qui n'en a pas (Halo 5) se sert du cache, sans panique,
// et le fournisseur injecté est bien celui qu'on appelle.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// fournisseurDirectCompte est un HaloLiveProvider qui rend « live indisponible » et compte ses appels.
type fournisseurDirectCompte struct{ bp, defis, pistes int }

func (f *fournisseurDirectCompte) GetBattlePassWithRaw(context.Context) (domain.BattlePassResponse, []byte) {
	f.bp++
	return domain.BattlePassResponse{}, nil
}

func (f *fournisseurDirectCompte) GetChallengesWithRaw(context.Context) (domain.ChallengesResponse, []byte) {
	f.defis++
	return domain.ChallengesResponse{}, nil
}

func (f *fournisseurDirectCompte) FetchAndWarmTrack(context.Context, string) { f.pistes++ }

// TestHomeService_Fournisseur_EstCeluiDuConstructeur : le service appelle le fournisseur qu'on lui
// donne, et lui seul.
func TestHomeService_Fournisseur_EstCeluiDuConstructeur(t *testing.T) {
	f := &fournisseurDirectCompte{}
	svc := NewHomeService(&mockHomeRepo{}, f).WithCacheRepo(&mockBattlePassCacheRepo{})
	ctx := context.Background()

	svc.GetBattlePass(ctx)
	svc.GetChallenges(ctx)
	svc.RefreshTrack(ctx, "RewardTracks/Operations/Season1")

	if f.bp != 1 || f.defis != 1 || f.pistes != 1 {
		t.Errorf("appels au fournisseur : passe %d, défis %d, pistes %d ; attendu 1, 1, 1", f.bp, f.defis, f.pistes)
	}
}

// TestHomeService_Halo5SansLectureEnDirect : un titre sans lecture en direct reçoit un fournisseur nil.
// La passe de combat et les défis se servent depuis le cache quand il existe, sinon « indisponible » — ni
// panique, ni donnée d'un autre titre ; RefreshTrack est sans effet.
func TestHomeService_Halo5SansLectureEnDirect(t *testing.T) {
	rank, track := 7, "RewardTracks/Operations/Cache"
	total := 3
	avecCache := &mockBattlePassCacheRepo{
		bpResp: &domain.BattlePassResponse{Available: true, Rank: &rank, RewardTrack: &track, FromCache: true},
		bpHit:  true,
		chResp: &domain.ChallengesResponse{Available: true, Total: &total, FromCache: true},
		chHit:  true,
	}
	ctx := context.Background()

	svc := NewHomeService(&mockHomeRepo{}, nil).WithCacheRepo(avecCache)
	if bp := svc.GetBattlePass(ctx); !bp.Available || !bp.FromCache || bp.Rank == nil || *bp.Rank != 7 {
		t.Errorf("passe de combat avec cache : %+v, attendu le cache", bp)
	}
	if ch := svc.GetChallenges(ctx); !ch.Available || !ch.FromCache || ch.Total == nil || *ch.Total != 3 {
		t.Errorf("défis avec cache : %+v, attendu le cache", ch)
	}

	sansCache := NewHomeService(&mockHomeRepo{}, nil).WithCacheRepo(&mockBattlePassCacheRepo{})
	if bp := sansCache.GetBattlePass(ctx); bp.Available {
		t.Errorf("passe de combat sans cache : %+v, attendu indisponible", bp)
	}
	if ch := sansCache.GetChallenges(ctx); ch.Available {
		t.Errorf("défis sans cache : %+v, attendu indisponible", ch)
	}
	sansCache.RefreshTrack(ctx, "RewardTracks/Operations/Season1") // sans fournisseur : sans effet
}
