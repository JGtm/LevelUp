package service

// career_service_friends_test.go — résolution des amis des rencontres de la Carrière (lot perf
// L9-go, 2026-09-23, revue adversariale D, P1) : le registre des profils suivis d'abord (xuid
// connu, aucune lecture), puis UNE lecture pour les autres — jamais une lecture par ami (avant :
// ExplorerRepo.ResolveXUIDByGamertag, 1,8 à 2,8 s chacun).

import (
	"context"
	"errors"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

// lecteurDAmis compte ses lectures et rend les xuids connus de `base`.
type lecteurDAmis struct {
	base     map[string]string
	err      error
	lectures [][]string
}

func (l *lecteurDAmis) lire(_ context.Context, gts []string) (map[string]string, error) {
	l.lectures = append(l.lectures, slices.Clone(gts))
	if l.err != nil {
		return nil, l.err
	}
	out := map[string]string{}
	for _, gt := range gts {
		if x, ok := l.base[gt]; ok {
			out[gt] = x
		}
	}
	return out, nil
}

func registreDesSuivis(context.Context) map[string]string {
	return map[string]string{"Madina97294": "x_madina", "Chocoboflor": "x_choco", "JGtm": "x_jgtm"}
}

func careerAvecAmis(amis []string, l *lecteurDAmis) (*CareerService, *mockCareerRepo) {
	repo := &mockCareerRepo{
		topEncountersRows:  []domain.MatchEncounterRow{{XUID: "x42", Gamertag: "Stranger", CountTogether: 5}},
		topEncountersStats: []domain.EncounterStatsRaw{{XUID: "x42", AllyCount: 0, EnemyCount: 5}},
	}
	svc := NewCareerService(repo).
		WithFriendGamertagsResolver(func(context.Context) []string { return amis }).
		WithFriendXUIDSources(registreDesSuivis, l.lire)
	return svc, repo
}

// TestCareerService_ResolveFriendXUIDs_RegistreDAbordPuisUneLecture : les amis suivis se
// résolvent par le registre, sans casse ni espaces parasites ; les autres en UNE lecture ; un
// ami que rien ne connaît n'est pas exclu.
func TestCareerService_ResolveFriendXUIDs_RegistreDAbordPuisUneLecture(t *testing.T) {
	l := &lecteurDAmis{base: map[string]string{"Hors registre": "x_hors"}}
	svc, repo := careerAvecAmis([]string{"madina97294", " Chocoboflor ", "Inconnu", "Hors registre", ""}, l)
	if _, err := svc.GetTopEncounters(context.Background()); err != nil {
		t.Fatalf("GetTopEncounters : %v", err)
	}
	exclus := slices.Sorted(slices.Values(repo.topEncountersExcludeArg))
	if want := []string{"x_choco", "x_hors", "x_madina"}; !slices.Equal(exclus, want) {
		t.Errorf("xuids exclus = %v, want %v", exclus, want)
	}
	if len(l.lectures) != 1 || !slices.Equal(l.lectures[0], []string{"Inconnu", "Hors registre"}) {
		t.Errorf("lectures = %v, want UNE lecture des deux amis hors registre", l.lectures)
	}
}

// TestCareerService_ResolveFriendXUIDs_TousSuivis_AucuneLecture : des amis tous suivis (le cas
// de la base de production) ne coûtent aucune lecture.
func TestCareerService_ResolveFriendXUIDs_TousSuivis_AucuneLecture(t *testing.T) {
	l := &lecteurDAmis{}
	svc, repo := careerAvecAmis([]string{"Madina97294", "Chocoboflor", "JGtm"}, l)
	if _, err := svc.GetTopEncounters(context.Background()); err != nil {
		t.Fatalf("GetTopEncounters : %v", err)
	}
	if len(l.lectures) != 0 || len(repo.topEncountersExcludeArg) != 3 {
		t.Errorf("lectures = %v, exclus = %v — want aucune lecture, 3 exclus", l.lectures, repo.topEncountersExcludeArg)
	}
}

// TestCareerService_ResolveFriendXUIDs_LectureEnEchec : la lecture en échec ne fait pas échouer
// les rencontres (best-effort, journalisé) ; les amis du registre restent exclus.
func TestCareerService_ResolveFriendXUIDs_LectureEnEchec(t *testing.T) {
	l := &lecteurDAmis{err: errors.New("database is locked")}
	svc, repo := careerAvecAmis([]string{"JGtm", "Hors registre"}, l)
	resp, err := svc.GetTopEncounters(context.Background())
	if err != nil || len(resp.Items) != 1 {
		t.Fatalf("GetTopEncounters : err=%v, %d rencontre(s)", err, len(resp.Items))
	}
	if !slices.Equal(repo.topEncountersExcludeArg, []string{"x_jgtm"}) {
		t.Errorf("xuids exclus = %v, want [x_jgtm]", repo.topEncountersExcludeArg)
	}
}
