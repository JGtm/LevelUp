package teammates

// composition_stricte_soiree_test.go — non-régression de la soirée du 7 octobre 2026 (ADR 0033,
// décision 1) : huit matchs de JGtm avec Joueur90001 et Joueur90002 du début à la fin, la
// quatrième place tenue par un inconnu différent selon les matchs. L'un d'eux (« Joueur900008 »)
// était dans deux de ces matchs (Aquarius, Houseki), ce qui le plaçait dans le top des
// coéquipiers fréquents : l'option « composition stricte » écartait ces deux matchs. Un
// coéquipier connu, c'est un ami déclaré ou un profil suivi — rien d'autre.

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	soireeJGtm     = "2603225103667623"
	soireeMadina   = "2623439693412443"
	soireeChoco    = "2547408409520575"
	soireeTeaSnack = "2555408283632666"
	soireeSession  = "S_07_10"
)

// soireeDu7Octobre : les huit matchs (heure UTC du registre), la quatrième place de chacun, et
// le registre des profils et des amis de l'instance (JGtm, Joueur90001, Joueur90002,
// Joueur90000000f suivis et amis entre eux ; J900004 suivi ; Joueur9009 auth_only).
func soireeDu7Octobre() *mockSquadRepo {
	type match struct {
		id, carte, quatrieme string
		h, m                 int
	}
	matchs := []match{
		{"1bde099b", "Dredge", "2700955136410675", 19, 23},
		{"3b515f13", "Cliffhanger", "2666953393830752", 19, 37},
		{"ff9d29b8", "Aquarius", soireeTeaSnack, 19, 50},
		{"b20cff37", "Houseki", soireeTeaSnack, 20, 3},
		{"5c38f581", "Prism", "2609704305081750", 20, 10},
		{"6c55be89", "Fortress", "2593890809119174", 20, 19},
		{"760fb768", "Empyrean", "2559048201277236", 20, 33},
		{"0a08d2f2", "Banished Narrows", "2571106636493958", 20, 48},
	}
	var rows []domain.SquadMatchRow
	var allies []domain.AllyParticipant
	for _, m := range matchs {
		at := time.Date(2026, 10, 7, m.h, m.m, 0, 0, time.UTC)
		rows = append(rows, makeSquadRowSess(m.id, m.carte, domain.OutcomeWin, soireeSession, at))
		allies = append(allies,
			ally(m.id, soireeJGtm), ally(m.id, soireeMadina), ally(m.id, soireeChoco), ally(m.id, m.quatrieme))
	}
	return &mockSquadRepo{
		// Le top des coéquipiers fréquents (Q29) tel que mesuré : Joueur900008 y figure, avec
		// deux matchs ensemble — ceux de cette soirée.
		topRows: []domain.TopTeammateRow{
			{XUID: soireeMadina, Gamertag: "Joueur90002", GamesTogether: 587},
			{XUID: soireeChoco, Gamertag: "Joueur90001", GamesTogether: 466},
			{XUID: "2558964982616721", Gamertag: "Joueur90000000f", GamesTogether: 39},
			{XUID: soireeTeaSnack, Gamertag: "Joueur900008", GamesTogether: 2},
		},
		squadRowsByTeammate: map[string][]domain.SquadMatchRow{soireeMadina: rows, soireeChoco: rows},
		allyRows:            allies,
		amis:                []string{"Joueur90000000f", "Joueur90002", "Joueur90001"},
		profils: []domain.PlayerSummary{
			profilSuivi(soireeJGtm, "JGtm"), profilSuivi(soireeChoco, "Joueur90001"),
			profilSuivi(soireeMadina, "Joueur90002"), profilSuivi("2558964982616721", "Joueur90000000f"),
			profilSuivi("2543518745364847", "J900004"),
			{XUID: "2734274180837334", Gamertag: "Joueur9009", AuthOnly: true},
		},
	}
}

// pageEtSessionsDeLaSoiree : la page Escouade (option cochée) et la lecture légère des sessions,
// qui doivent dire la même chose.
func pageEtSessionsDeLaSoiree(t *testing.T, repo *mockSquadRepo) domain.CompositionSessionEntry {
	t.Helper()
	gts := []string{"Joueur90001", "Joueur90002"}
	svc := avecConnus(NewTeammatesService(repo, nil), repo).WithPlayerMatchesRepo(
		newSynthMockFromRows(repo.synthRows, repo.synthErr), "halo_infinite", "JGtm")
	page, err := svc.GetPage(context.Background(), soireeJGtm, domain.TeammatesQueryRequest{
		SelectedGamertags: gts, FilterExactComposition: true,
	})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	sessions, _, err := svc.CompositionSessions(context.Background(), soireeJGtm, gts, true)
	if err != nil {
		t.Fatalf("CompositionSessions : %v", err)
	}
	if !reflect.DeepEqual(page.CompositionSessions, sessions) {
		t.Errorf("page et lecture légère divergent :\n page   %+v\n légère %+v", page.CompositionSessions, sessions)
	}
	if len(sessions) != 1 || sessions[0].Label != soireeSession {
		t.Fatalf("sessions : %v, attendu [%s]", labelsDe(sessions), soireeSession)
	}
	if got := len(page.MatchHistory); got != sessions[0].MatchCount {
		t.Errorf("historique de la page : %d matchs, la session en compte %d", got, sessions[0].MatchCount)
	}
	return sessions[0]
}

// TestCompositionStricte_Soiree7Octobre_InconnuFrequentNEcartePas : la configuration de
// l'instance — l'inconnu présent dans deux matchs n'est ni ami ni suivi : les huit matchs restent.
func TestCompositionStricte_Soiree7Octobre_InconnuFrequentNEcartePas(t *testing.T) {
	s := pageEtSessionsDeLaSoiree(t, soireeDu7Octobre())
	if s.MatchCount != 8 || s.MatchCountRoster != 8 || len(s.ExcludedByExactComposition) != 0 {
		t.Errorf("session : %d sur %d, écartés %+v — attendu 8 sur 8, aucun écarté",
			s.MatchCount, s.MatchCountRoster, s.ExcludedByExactComposition)
	}
}

// TestCompositionStricte_Soiree7Octobre_ConnuEcarte : le même joueur, ami déclaré ou profil
// suivi, est un coéquipier connu : ses deux matchs sont écartés et il est nommé. Un profil
// auth_only (compte prêteur de jetons) n'est pas un coéquipier.
func TestCompositionStricte_Soiree7Octobre_ConnuEcarte(t *testing.T) {
	cas := []struct {
		nom       string
		modifier  func(*mockSquadRepo)
		gardes    int
		ecartePar []string
	}{
		{"ami declare", func(r *mockSquadRepo) {
			r.amis = append(r.amis, "Joueur900008")
			r.amisLus = map[string]string{"Joueur900008": soireeTeaSnack}
		}, 6, []string{"Joueur900008"}},
		{"profil suivi", func(r *mockSquadRepo) {
			r.profils = append(r.profils, profilSuivi(soireeTeaSnack, "Joueur900008"))
		}, 6, []string{"Joueur900008"}},
		{"profil suivi en pause", func(r *mockSquadRepo) {
			p := profilSuivi(soireeTeaSnack, "Joueur900008")
			p.SyncEnabled = false
			r.profils = append(r.profils, p)
		}, 6, []string{"Joueur900008"}},
		{"profil auth_only", func(r *mockSquadRepo) {
			r.profils = append(r.profils, domain.PlayerSummary{XUID: soireeTeaSnack, Gamertag: "Joueur900008", AuthOnly: true})
		}, 8, nil},
		{"profil auth_only declare ami", func(r *mockSquadRepo) {
			r.profils = append(r.profils, domain.PlayerSummary{XUID: soireeTeaSnack, Gamertag: "Joueur900008", AuthOnly: true})
			r.amis = append(r.amis, "joueur900008")
		}, 6, []string{"Joueur900008"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo := soireeDu7Octobre()
			c.modifier(repo)
			s := pageEtSessionsDeLaSoiree(t, repo)
			if s.MatchCount != c.gardes || s.MatchCountRoster != 8 {
				t.Fatalf("session : %d sur %d, attendu %d sur 8", s.MatchCount, s.MatchCountRoster, c.gardes)
			}
			var ecartes []string
			for _, ex := range s.ExcludedByExactComposition {
				ecartes = append(ecartes, ex.MatchID)
				if !slices.Equal(ex.ExtraGamertags, c.ecartePar) {
					t.Errorf("%s écarté par %v, attendu %v", ex.MatchID, ex.ExtraGamertags, c.ecartePar)
				}
			}
			slices.Sort(ecartes)
			if want := map[int][]string{6: {"b20cff37", "ff9d29b8"}, 8: nil}[c.gardes]; !slices.Equal(ecartes, want) {
				t.Errorf("matchs écartés : %v, attendu %v (Aquarius et Houseki)", ecartes, want)
			}
		})
	}
}
