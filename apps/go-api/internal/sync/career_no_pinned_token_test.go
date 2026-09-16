package sync

// career_no_pinned_token_test.go — le rang de carrière se dégrade SEUL quand le joueur n'a
// pas son propre token, et le sync reste en succès.
//
// POURQUOI (2026-09-16). Un profil suivi sans refresh token propre se synchronise par le pool
// (D1) : historique, stats, films et CSR sont des endpoints publics. Le rang de carrière est
// le SEUL endpoint privacy-gated du sync (PolicyPinnedPlayer). Il doit donc être sauté avec un
// WARN — jamais faire échouer la passe, jamais disparaître en silence.
//
// VÉRIFIÉ SUR PIÈCES le 2026-09-16 : l'étape carrière est DÉCOUPLÉE du post-sync depuis le
// 2026-05-14 (cf. engine_postsync.go, section 3) — le flux XP + Spartan ID est servi par
// service.CareerLiveService. `domain.PostSyncResult.CareerSynced` reste dans le struct mais
// n'est plus jamais positionné à true par le post-sync. Le seam carrière du paquet est
// syncCareerRank : c'est lui qui porte la dégradation.

import (
	"context"
	"errors"
	"testing"
)

// clientSansTokenPropre : un HaloClient dont le seul comportement utile est de rendre
// ErrNoPinnedToken sur la carrière, comme le fait PooledHaloClient pour un joueur hors pool.
type clientSansTokenPropre struct {
	*mockHaloClient
	appels int
}

func (c *clientSansTokenPropre) GetCareerRank(_ context.Context, _ string) (*CareerRankData, error) {
	c.appels++
	return nil, ErrNoPinnedToken
}

// TestSyncCareerRank_SansTokenPropre_DegradeSansEchouer — (nil, nil) et AUCUNE erreur remontée.
func TestSyncCareerRank_SansTokenPropre_DegradeSansEchouer(t *testing.T) {
	client := &clientSansTokenPropre{mockHaloClient: &mockHaloClient{}}

	rank, err := syncCareerRank(context.Background(), client, "2533274800000000")
	if err != nil {
		t.Fatalf("syncCareerRank a rendu une erreur : %v — le sync doit rester en succès "+
			"quand seul le rang de carrière est inaccessible (D1, plan 2026-09-16)", err)
	}
	if rank != nil {
		t.Errorf("rang = %v, attendu nil", rank)
	}
	if client.appels != 1 {
		t.Errorf("appels à GetCareerRank = %d, attendu 1", client.appels)
	}
}

// TestSyncCareerRank_AutreErreur_Remontee — toute AUTRE erreur remonte : la dégradation est
// réservée à l'absence de token propre, elle n'avale pas les pannes.
func TestSyncCareerRank_AutreErreur_Remontee(t *testing.T) {
	panne := errors.New("economy 500")
	client := &mockHaloClient{getCareerErr: panne}

	if _, err := syncCareerRank(context.Background(), client, "2533274800000000"); !errors.Is(err, panne) {
		t.Errorf("erreur = %v, attendu la panne d'origine (pas d'avalement)", err)
	}
}
