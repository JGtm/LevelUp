package teammates

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

func assistTestRows() []domain.SquadMatchRow {
	start := time.Date(2026, 8, 20, 18, 0, 0, 0, time.UTC)
	return []domain.SquadMatchRow{
		{MatchID: "m1", StartTime: start, DurationSeconds: 600},
		{MatchID: "m2", StartTime: start.Add(time.Hour), DurationSeconds: 600},
		{MatchID: "m3", StartTime: start.Add(2 * time.Hour), DurationSeconds: 600},
	}
}

func assistTestMates() []domain.TeammateRow {
	x := "x_mate"
	return []domain.TeammateRow{{Gamertag: "Mate", XUID: &x}}
}

// TestBuildSquadAssistPairs_Nominal : la couverture est en MATCHS (pas en morts), le
// total sert de dénominateur à la colonne « part », et les gamertags viennent du ROSTER
// de la page — pas de ce que le film a écrit.
func TestBuildSquadAssistPairs_Nominal(t *testing.T) {
	repo := &mockSquadRepo{
		assistPairs: []domain.SquadAssistPairRaw{
			{AssistXUID: "x_main", KillerXUID: "x_mate", AssistCount: 7, StolenCount: 3},
			{AssistXUID: "x_mate", KillerXUID: "x_main", AssistCount: 3, StolenCount: 0},
		},
		assistMeasured: 2,
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}

	got := svc.buildSquadAssistPairs(context.Background(), assistTestRows(), "main", "x_main", assistTestMates())
	if got == nil {
		t.Fatal("bloc attendu, obtenu nil")
	}
	if got.MatchesMeasured != 2 || got.MatchesTotal != 3 {
		t.Errorf("couverture = %d/%d, attendue 2/3", got.MatchesMeasured, got.MatchesTotal)
	}
	if got.TotalAssists != 10 {
		t.Errorf("total = %d, attendu 10 (dénominateur de la part)", got.TotalAssists)
	}
	if len(got.Pairs) != 2 {
		t.Fatalf("paires = %+v, attendu 2", got.Pairs)
	}
	if got.Pairs[0].AssistGamertag != "main" || got.Pairs[0].KillerGamertag != "Mate" {
		t.Errorf("noms résolus depuis le roster attendus, obtenu %q -> %q",
			got.Pairs[0].AssistGamertag, got.Pairs[0].KillerGamertag)
	}
	if got.Pairs[0].StolenCount != 3 {
		t.Errorf("éliminations volées = %d, attendu 3", got.Pairs[0].StolenCount)
	}
}

// TestBuildSquadAssistPairs_RienMesure : aucun match de la sélection n'a d'assistance
// mesurée → AUCUN bloc. C'est aussi ce qui arrive sur un titre sans décodeur de film,
// obtenu par la DONNÉE seule — jamais par un test sur le slug.
func TestBuildSquadAssistPairs_RienMesure(t *testing.T) {
	repo := &mockSquadRepo{assistMeasured: 0}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	if got := svc.buildSquadAssistPairs(
		context.Background(), assistTestRows(), "main", "x_main", assistTestMates(),
	); got != nil {
		t.Fatalf("bloc = %+v, attendu nil", got)
	}
}

// TestBuildSquadAssistPairs_RetraitCompte : LE RETRAIT DU BLOC LAISSE UNE TRACE.
//
// Ce test existe parce que son absence a coûté cinq mois : le `return nil` sur
// `measured == 0` a fait disparaître le tableau de la page sans aucun signal, et le défaut
// n'a été trouvé que sur question de l'utilisateur (registre
// `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`). Le compteur est ce qui rend le retrait
// observable ; le pin sur son incrément est ce qui empêche de le supprimer par distraction.
func TestBuildSquadAssistPairs_RetraitCompte(t *testing.T) {
	avant := observability.LoadCounter(compteurAssistPairsSansMesure)
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: &mockSquadRepo{assistMeasured: 0}}

	if got := svc.buildSquadAssistPairs(
		context.Background(), assistTestRows(), "main", "x_main", assistTestMates(),
	); got != nil {
		t.Fatalf("bloc = %+v, attendu nil", got)
	}
	if apres := observability.LoadCounter(compteurAssistPairsSansMesure); apres != avant+1 {
		t.Errorf("%s = %d, attendu %d — un bloc qui se retire sans trace est le défaut qu'on corrige",
			compteurAssistPairsSansMesure, apres, avant+1)
	}

	// Le compteur ne bouge PAS quand le bloc sort : sinon il mesurerait le trafic, pas le trou.
	avant = observability.LoadCounter(compteurAssistPairsSansMesure)
	svcOK := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: &mockSquadRepo{assistMeasured: 3}}
	if got := svcOK.buildSquadAssistPairs(
		context.Background(), assistTestRows(), "main", "x_main", assistTestMates(),
	); got == nil {
		t.Fatal("bloc attendu")
	}
	if apres := observability.LoadCounter(compteurAssistPairsSansMesure); apres != avant {
		t.Errorf("%s a bougé sur un bloc émis (%d -> %d)", compteurAssistPairsSansMesure, avant, apres)
	}
}

// TestBuildSquadAssistPairs_MesureSansPaire : mesuré, mais l'escouade ne s'est pas
// entraidée. LE BLOC EXISTE QUAND MÊME — « mesuré sur 3 des 3 matchs, aucune assistance
// interne » est un fait ; le taire le ferait passer pour « rien mesuré ».
func TestBuildSquadAssistPairs_MesureSansPaire(t *testing.T) {
	repo := &mockSquadRepo{assistMeasured: 3}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	got := svc.buildSquadAssistPairs(context.Background(), assistTestRows(), "main", "x_main", assistTestMates())
	if got == nil {
		t.Fatal("bloc attendu (la mesure a eu lieu), obtenu nil")
	}
	if len(got.Pairs) != 0 || got.TotalAssists != 0 {
		t.Errorf("paires = %+v, total = %d ; attendu vide et 0", got.Pairs, got.TotalAssists)
	}
	if got.MatchesMeasured != 3 || got.MatchesTotal != 3 {
		t.Errorf("couverture = %d/%d, attendue 3/3", got.MatchesMeasured, got.MatchesTotal)
	}
}

// TestBuildSquadAssistPairs_PaireHorsRoster : impossible par construction (Q32d contraint
// les deux côtés), mais si elle survenait la ligne serait ÉCARTÉE — et surtout exclue du
// TOTAL, sinon les parts affichées ne sommeraient plus à 100 %.
func TestBuildSquadAssistPairs_PaireHorsRoster(t *testing.T) {
	repo := &mockSquadRepo{
		assistPairs: []domain.SquadAssistPairRaw{
			{AssistXUID: "x_main", KillerXUID: "x_mate", AssistCount: 4},
			{AssistXUID: "x_inconnu", KillerXUID: "x_mate", AssistCount: 6},
		},
		assistMeasured: 1,
	}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	got := svc.buildSquadAssistPairs(context.Background(), assistTestRows(), "main", "x_main", assistTestMates())
	if got == nil || len(got.Pairs) != 1 {
		t.Fatalf("bloc = %+v, attendu une seule paire", got)
	}
	if got.TotalAssists != 4 {
		t.Errorf("total = %d, attendu 4 (la ligne écartée ne compte pas)", got.TotalAssists)
	}
}

// TestBuildSquadAssistPairs_PerimetreVide : aucun match, ou aucun joueur résolu → pas de
// lecture, pas de bloc.
func TestBuildSquadAssistPairs_PerimetreVide(t *testing.T) {
	repo := &mockSquadRepo{assistMeasured: 5}
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	if got := svc.buildSquadAssistPairs(context.Background(), nil, "main", "x_main", nil); got != nil {
		t.Errorf("sans match : bloc = %+v, attendu nil", got)
	}
	if got := svc.buildSquadAssistPairs(context.Background(), assistTestRows(), "main", "", nil); got != nil {
		t.Errorf("sans joueur résolu : bloc = %+v, attendu nil", got)
	}
}

// TestBuildSquadAssistPairs_RepoAbsentOuEnErreur : dégradation gracieuse aux deux bouts —
// pas de repo câblé, ou lecture en échec. Aucun bloc, jamais de panique.
func TestBuildSquadAssistPairs_RepoAbsentOuEnErreur(t *testing.T) {
	svc := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main"} // repo nil
	if got := svc.buildSquadAssistPairs(
		context.Background(), assistTestRows(), "main", "x_main", assistTestMates(),
	); got != nil {
		t.Errorf("repo nil : bloc = %+v, attendu nil", got)
	}

	repo := &mockSquadRepo{assistErr: context.DeadlineExceeded, assistMeasured: 3}
	svcErr := &TeammatesService{titleSlug: "halo_infinite", gamertag: "main", repo: repo}
	if got := svcErr.buildSquadAssistPairs(
		context.Background(), assistTestRows(), "main", "x_main", assistTestMates(),
	); got != nil {
		t.Errorf("lecture en échec : bloc = %+v, attendu nil", got)
	}
}
