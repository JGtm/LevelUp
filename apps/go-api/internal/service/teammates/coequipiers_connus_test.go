package teammates

// coequipiers_connus_test.go — la définition des coéquipiers connus (ADR 0033, décision 1) et la
// résolution des amis déclarés en xuids partagée avec la Carrière (ADR 0036 I4 : le registre
// d'abord, puis UNE lecture pour les autres).

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"levelup/go-api/internal/domain"
)

// lecteurCompte : la lecture des amis hors registre, lectures comptées.
type lecteurCompte struct {
	base     map[string]string
	err      error
	lectures [][]string
}

func (l *lecteurCompte) lire(_ context.Context, gts []string) (map[string]string, error) {
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

func TestResolveFriendXUIDs_RegistreDAbordPuisUneLecture(t *testing.T) {
	l := &lecteurCompte{base: map[string]string{"Hors registre": "x_hors"}}
	registre := map[string]string{"Madina97294": "x_madina", "Chocoboflor": "x_choco", "Sans xuid": ""}
	got := ResolveFriendXUIDs(context.Background(), "test",
		[]string{"madina97294", " Chocoboflor ", "Inconnu", "Hors registre", "", "Sans xuid"}, registre, l.lire)
	want := map[string]string{"x_madina": "madina97294", "x_choco": "Chocoboflor", "x_hors": "Hors registre"}
	if !maps.Equal(got, want) {
		t.Errorf("résolus = %v, attendu %v", got, want)
	}
	if len(l.lectures) != 1 || !slices.Equal(l.lectures[0], []string{"Inconnu", "Hors registre", "Sans xuid"}) {
		t.Errorf("lectures = %v, attendu UNE lecture des trois amis hors registre", l.lectures)
	}
}

func TestResolveFriendXUIDs_TousAuRegistre_AucuneLecture(t *testing.T) {
	l := &lecteurCompte{}
	got := ResolveFriendXUIDs(context.Background(), "test", []string{"A", "B"},
		map[string]string{"a": "xa", "B": "xb"}, l.lire)
	if len(got) != 2 || len(l.lectures) != 0 {
		t.Errorf("résolus = %v, lectures = %v — attendu 2 résolus sans lecture", got, l.lectures)
	}
}

func TestResolveFriendXUIDs_LectureEnEchecOuAbsente(t *testing.T) {
	l := &lecteurCompte{err: errors.New("database is locked")}
	registre := map[string]string{"A": "xa"}
	got := ResolveFriendXUIDs(context.Background(), "test", []string{"A", "Autre"}, registre, l.lire)
	if !maps.Equal(got, map[string]string{"xa": "A"}) {
		t.Errorf("lecture en échec : %v, attendu le seul ami du registre", got)
	}
	got = ResolveFriendXUIDs(context.Background(), "test", []string{"A", "Autre"}, registre, nil)
	if !maps.Equal(got, map[string]string{"xa": "A"}) {
		t.Errorf("sans lecture : %v, attendu le seul ami du registre", got)
	}
}

// connusDe : les coéquipiers connus d'un service branché sur ces profils, ces amis et cette
// lecture.
func connusDe(profils []domain.PlayerSummary, amis []string, l *lecteurCompte, errProfils error) map[string]string {
	svc := NewTeammatesService(&mockSquadRepo{}, nil).WithCoequipiersConnus(
		func(context.Context) ([]domain.PlayerSummary, error) { return profils, errProfils }, l.lire)
	return svc.coequipiersConnus(context.Background(), amis)
}

// TestCoequipiersConnus_ProfilsSuivisEtAmisDeclares : les profils suivis (une pause n'y change
// rien) et les amis déclarés, rien d'autre ; un profil auth_only n'en est un que déclaré ami, et
// il garde alors le nom de son profil.
func TestCoequipiersConnus_ProfilsSuivisEtAmisDeclares(t *testing.T) {
	enPause := profilSuivi("x_pause", "EnPause")
	enPause.SyncEnabled = false
	profils := []domain.PlayerSummary{
		profilSuivi("x_moi", "Moi"), profilSuivi("x_suivi", "Suivi"), enPause,
		{XUID: "x_jeton", Gamertag: "Jeton", AuthOnly: true},
		{XUID: "x_jeton_ami", Gamertag: "JetonAmi", AuthOnly: true},
		{XUID: "", Gamertag: "SansXuid"},
	}
	l := &lecteurCompte{base: map[string]string{"AmiHors": "x_ami_hors"}}
	got := connusDe(profils, []string{"jetonami", "AmiHors", "Personne"}, l, nil)
	want := map[string]string{
		"x_moi": "Moi", "x_suivi": "Suivi", "x_pause": "EnPause",
		"x_jeton_ami": "JetonAmi", "x_ami_hors": "AmiHors",
	}
	if !maps.Equal(got, want) {
		t.Errorf("connus = %v, attendu %v", got, want)
	}
}

// TestCoequipiersConnus_RegistreIllisible : les amis se résolvent alors par la seule lecture ;
// sans aucune source, personne n'est connu.
func TestCoequipiersConnus_RegistreIllisible(t *testing.T) {
	l := &lecteurCompte{base: map[string]string{"Ami": "x_ami"}}
	got := connusDe(nil, []string{"Ami"}, l, errors.New("db_profiles.json illisible"))
	if !maps.Equal(got, map[string]string{"x_ami": "Ami"}) {
		t.Errorf("connus = %v, attendu le seul ami lu", got)
	}
	if got := NewTeammatesService(&mockSquadRepo{}, nil).coequipiersConnus(context.Background(), []string{"Ami"}); len(got) != 0 {
		t.Errorf("sans source : %v, attendu aucun connu", got)
	}
}
