package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// TestBuildAssistPairs_BlocAbsentSansFilm : aucune ligne de film pour ce match (ou titre
// sans décodeur) → AUCUN bloc. C'est la porte qui évite qu'un message « non mesuré »
// s'affiche sur tous les matchs d'un titre qui n'a jamais eu de décodeur.
func TestBuildAssistPairs_BlocAbsentSansFilm(t *testing.T) {
	got := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{}, nil)
	if got != nil {
		t.Fatalf("buildAssistPairs = %+v, attendu nil", got)
	}
	// Même verdict si la portée est vide alors que des paires seraient présentes :
	// c'est un état incohérent, et le bloc ne doit pas se fabriquer une portée.
	got = buildAssistPairs(
		context.Background(),
		[]domain.MatchAssistPairRaw{{AssistXUID: "A", KillerXUID: "K", AssistCount: 1}},
		domain.MatchAssistScopeRaw{},
		nil,
	)
	if got != nil {
		t.Fatalf("buildAssistPairs (portée vide) = %+v, attendu nil", got)
	}
}

// TestBuildAssistPairs_NonMesureVsZero : LES DEUX ÉTATS QUE LE BLOC EXISTE POUR
// DISTINGUER. Le film est là dans les deux cas ; ce qui change est MeasuredDeaths.
func TestBuildAssistPairs_NonMesureVsZero(t *testing.T) {
	// (a) « non mesuré » : des morts, aucune assistance mesurée.
	nonMesure := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{MatchDeaths: 40}, nil)
	if nonMesure == nil {
		t.Fatal("bloc attendu (le match a un film), obtenu nil")
	}
	if nonMesure.MeasuredDeaths != 0 {
		t.Errorf("MeasuredDeaths = %d, attendu 0", nonMesure.MeasuredDeaths)
	}
	if len(nonMesure.Pairs) != 0 {
		t.Errorf("Pairs = %+v, attendu vide", nonMesure.Pairs)
	}

	// (b) « mesuré, zéro assistance » : mêmes paires vides, portée DIFFÉRENTE.
	mesureZero := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{MatchDeaths: 40, MeasuredDeaths: 38}, nil)
	if mesureZero == nil {
		t.Fatal("bloc attendu, obtenu nil")
	}
	if mesureZero.MeasuredDeaths != 38 {
		t.Errorf("MeasuredDeaths = %d, attendu 38", mesureZero.MeasuredDeaths)
	}
	if len(mesureZero.Pairs) != 0 {
		t.Errorf("Pairs = %+v, attendu vide", mesureZero.Pairs)
	}
}

// TestBuildAssistPairs_GamertagTueurDepuisScoreboard : le nom du tueur vient du
// scoreboard et de lui seul. Un tueur qui n'y figure pas garde son xuid et un gamertag
// VIDE — jamais un nom inventé, jamais le xuid recopié dans un champ de nom.
func TestBuildAssistPairs_GamertagTueurDepuisScoreboard(t *testing.T) {
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "K1", Gamertag: "Kilo"},
		{XUID: "A1", Gamertag: "Alpha"},
		{XUID: "K3", Gamertag: ""}, // présent mais anonyme : ne compte pas
	}
	avg := 45
	raw := []domain.MatchAssistPairRaw{
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K1", AssistCount: 3, StolenCount: 2, AvgAssistPct: &avg},
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K2", AssistCount: 1},
		{AssistXUID: "A1", AssistGamertag: "Alpha", KillerXUID: "K3", AssistCount: 1},
	}
	got := buildAssistPairs(context.Background(), raw, domain.MatchAssistScopeRaw{MatchDeaths: 50, MeasuredDeaths: 44}, scoreboard)
	if got == nil || len(got.Pairs) != 3 {
		t.Fatalf("bloc = %+v, attendu 3 paires", got)
	}
	if got.Pairs[0].KillerGamertag != "Kilo" {
		t.Errorf("tueur au scoreboard : gamertag = %q, attendu %q", got.Pairs[0].KillerGamertag, "Kilo")
	}
	if got.Pairs[1].KillerGamertag != "" {
		t.Errorf("tueur ABSENT du scoreboard : gamertag = %q, attendu vide", got.Pairs[1].KillerGamertag)
	}
	if got.Pairs[1].KillerXUID != "K2" {
		t.Errorf("le xuid du tueur doit survivre : %q", got.Pairs[1].KillerXUID)
	}
	if got.Pairs[2].KillerGamertag != "" {
		t.Errorf("tueur au scoreboard SANS nom : gamertag = %q, attendu vide", got.Pairs[2].KillerGamertag)
	}
	// Les compteurs traversent sans être recalculés — Q21d les a déjà comptés.
	if got.Pairs[0].AssistCount != 3 || got.Pairs[0].StolenCount != 2 {
		t.Errorf("compteurs = %d/%d, attendus 3/2", got.Pairs[0].AssistCount, got.Pairs[0].StolenCount)
	}
	// La part moyenne traverse telle quelle — et son ABSENCE aussi (nil, jamais 0).
	if got.Pairs[0].AvgAssistPct == nil || *got.Pairs[0].AvgAssistPct != 45 {
		t.Errorf("avg = %v, attendu 45", got.Pairs[0].AvgAssistPct)
	}
	if got.Pairs[1].AvgAssistPct != nil {
		t.Errorf("avg de la paire sans part = %v, attendu nil", *got.Pairs[1].AvgAssistPct)
	}
}

// TestBuildAssistPairs_AssistantAussiNommeParLeScoreboard : LE CORRECTIF F14.
//
// L'assistant arrivait nommé PAR LE FILM (gamertag capté à l'enregistrement) tandis que
// le tueur était résolu au scoreboard (nom d'API, alias compris). Un joueur qui a changé
// de gamertag depuis pouvait donc apparaître DEUX FOIS dans le même graphe, sous deux
// orthographes : une fois en barre (assistant) et une fois en segment (tueur).
func TestBuildAssistPairs_AssistantAussiNommeParLeScoreboard(t *testing.T) {
	scoreboard := []domain.ScoreboardRaw{
		{XUID: "A1", Gamertag: "NouveauNom"},
		{XUID: "K1", Gamertag: "Kilo"},
	}
	raw := []domain.MatchAssistPairRaw{
		// Le film porte l'ANCIEN nom de A1.
		{AssistXUID: "A1", AssistGamertag: "AncienNom", KillerXUID: "K1", AssistCount: 2},
		// A2 n'est pas au scoreboard (parti avant la fin) : son nom de film survit.
		{AssistXUID: "A2", AssistGamertag: "Bravo", KillerXUID: "K1", AssistCount: 1},
		// A3 est au scoreboard mais anonyme : on ne remplace pas un nom par du vide.
		{AssistXUID: "A3", AssistGamertag: "Charlie", KillerXUID: "K1", AssistCount: 1},
	}
	scoreboard = append(scoreboard, domain.ScoreboardRaw{XUID: "A3", Gamertag: ""})

	got := buildAssistPairs(context.Background(), raw, domain.MatchAssistScopeRaw{MatchDeaths: 20, MeasuredDeaths: 18}, scoreboard)
	if got == nil || len(got.Pairs) != 3 {
		t.Fatalf("bloc = %+v, attendu 3 paires", got)
	}
	if got.Pairs[0].AssistGamertag != "NouveauNom" {
		t.Errorf("assistant au scoreboard : gamertag = %q, attendu %q (le nom du scoreboard gagne)",
			got.Pairs[0].AssistGamertag, "NouveauNom")
	}
	if got.Pairs[1].AssistGamertag != "Bravo" {
		t.Errorf("assistant ABSENT du scoreboard : gamertag = %q, attendu le repli du film %q",
			got.Pairs[1].AssistGamertag, "Bravo")
	}
	if got.Pairs[2].AssistGamertag != "Charlie" {
		t.Errorf("assistant anonyme au scoreboard : gamertag = %q, attendu le repli du film %q",
			got.Pairs[2].AssistGamertag, "Charlie")
	}
	// Les deux côtés de la paire sortent du MÊME dictionnaire quand il les connaît.
	if got.Pairs[0].KillerGamertag != "Kilo" {
		t.Errorf("tueur = %q, attendu %q", got.Pairs[0].KillerGamertag, "Kilo")
	}
}

// TestBuildAssistPairs_CompteursDesDeuxCauses : les deux états sont COMPTÉS, et ils le
// sont séparément.
//
// « aucune ligne » se corrige en produisant des kill-events pour ce match ; « aucune
// mesure » se corrige en DÉCODANT SON FILM. C'est le second qui a explosé le 2026-04-07
// (registre `.ai/V7.5/REGISTRE_ASSISTANCES_2026-08-29.md`) : les additionner rendrait
// l'alerte inexploitable, et n'en compter aucun est ce qui a laissé le trou invisible.
func TestBuildAssistPairs_CompteursDesDeuxCauses(t *testing.T) {
	sansLigne := observability.LoadCounter(compteurMatchAssistSansLigne)
	sansMesure := observability.LoadCounter(compteurMatchAssistSansMesure)

	if got := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{}, nil); got != nil {
		t.Fatalf("buildAssistPairs = %+v, attendu nil", got)
	}
	if v := observability.LoadCounter(compteurMatchAssistSansLigne); v != sansLigne+1 {
		t.Errorf("%s = %d, attendu %d", compteurMatchAssistSansLigne, v, sansLigne+1)
	}
	if v := observability.LoadCounter(compteurMatchAssistSansMesure); v != sansMesure {
		t.Errorf("%s a bougé sur un match SANS ligne — les deux causes seraient confondues",
			compteurMatchAssistSansMesure)
	}

	// Le film est là, mais aucune mort n'est lisible : le bloc SORT, et l'état est compté.
	sansMesure = observability.LoadCounter(compteurMatchAssistSansMesure)
	if got := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{MatchDeaths: 40}, nil); got == nil {
		t.Fatal("bloc attendu : « non mesuré » est un état publié, pas une absence")
	}
	if v := observability.LoadCounter(compteurMatchAssistSansMesure); v != sansMesure+1 {
		t.Errorf("%s = %d, attendu %d", compteurMatchAssistSansMesure, v, sansMesure+1)
	}

	// Mesuré : aucun des deux ne bouge.
	sansLigne = observability.LoadCounter(compteurMatchAssistSansLigne)
	sansMesure = observability.LoadCounter(compteurMatchAssistSansMesure)
	if got := buildAssistPairs(context.Background(), nil, domain.MatchAssistScopeRaw{MatchDeaths: 40, MeasuredDeaths: 38}, nil); got == nil {
		t.Fatal("bloc attendu")
	}
	if v := observability.LoadCounter(compteurMatchAssistSansLigne); v != sansLigne {
		t.Errorf("%s a bougé sur un match mesuré", compteurMatchAssistSansLigne)
	}
	if v := observability.LoadCounter(compteurMatchAssistSansMesure); v != sansMesure {
		t.Errorf("%s a bougé sur un match mesuré : il compterait le trafic, pas le trou",
			compteurMatchAssistSansMesure)
	}
}

// TestBuildAssistPairs_CompteurSepareParTitre : `halo_5` est ACTIF et ce builder n'est protégé
// par aucune capability — ouvrir une vue match Halo 5 suffit à incrémenter. Sur une clé nue,
// « la passe de film s'est arrêtée » deviendrait indistinguable de « quelqu'un a navigué dans
// Halo 5 », c'est-à-dire l'ambiguïté même que ces compteurs existent pour supprimer.
func TestBuildAssistPairs_CompteurSepareParTitre(t *testing.T) {
	nu := observability.LoadCounter(compteurMatchAssistSansLigne)
	h5 := observability.LoadCounterT("halo_5", compteurMatchAssistSansLigne)

	ctxH5 := ctxkeys.WithTitleSlug(context.Background(), "halo_5")
	if got := buildAssistPairs(ctxH5, nil, domain.MatchAssistScopeRaw{}, nil); got != nil {
		t.Fatalf("buildAssistPairs = %+v, attendu nil", got)
	}
	if v := observability.LoadCounterT("halo_5", compteurMatchAssistSansLigne); v != h5+1 {
		t.Errorf("compteur halo_5 = %d, attendu %d", v, h5+1)
	}
	if v := observability.LoadCounter(compteurMatchAssistSansLigne); v != nu {
		t.Errorf("la clé NUE a bougé (%d -> %d) : la navigation Halo 5 pollue la mesure Halo Infinite",
			nu, v)
	}
}
