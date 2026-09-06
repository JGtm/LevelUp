package teammates

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// ─── DECOR ─────────────────────────────────────────────────────────────────────────────

// mockTacticalRepo sert un journal des morts pose a la main. Seul KillEvents est
// exerce ici : la page Escouade ne lit ni les cartes ni les positions.
type mockTacticalRepo struct {
	lecture domain.TacticalKillEvents
	err     error
	vues    []domain.TacticalQuery
}

func (m *mockTacticalRepo) MapsPlayed(context.Context, domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	return nil, errors.New("non appele")
}

func (m *mockTacticalRepo) KillPositions(context.Context, domain.TacticalQuery) (domain.TacticalPositions, error) {
	return domain.TacticalPositions{}, errors.New("non appele")
}

func (m *mockTacticalRepo) KillEvents(_ context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	m.vues = append(m.vues, q)
	return m.lecture, m.err
}

// capsFiables : la porte data-level ouverte par la provenance « film » (Halo Infinite).
func capsFiables() games.CapabilityMap {
	return games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}
}

func echangeRows(ids ...string) []domain.SquadMatchRow {
	start := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	out := make([]domain.SquadMatchRow, 0, len(ids))
	for i, id := range ids {
		out = append(out, domain.SquadMatchRow{
			MatchID:   id,
			StartTime: start.Add(time.Duration(i) * time.Hour),
		})
	}
	return out
}

func echangeMates(gts ...string) []domain.TeammateRow {
	out := make([]domain.TeammateRow, 0, len(gts))
	for _, gt := range gts {
		x := "x_" + gt
		out = append(out, domain.TeammateRow{Gamertag: gt, XUID: &x})
	}
	return out
}

// equipesDeuxContreDeux : moi + Ami en equipe 0, Adv1 + Adv2 en equipe 1, sur chaque match.
func equipesDeuxContreDeux(matchs ...string) domain.EquipesParMatch {
	out := domain.EquipesParMatch{}
	for _, id := range matchs {
		out[id] = map[string]int{
			"x_main": 0, "x_Ami": 0,
			"x_adv1": 1, "x_adv2": 1,
		}
	}
	return out
}

func universDe(matchs ...string) domain.TacticalUnivers {
	u := domain.TacticalUnivers{Equipes: equipesDeuxContreDeux(matchs...)}
	for _, id := range matchs {
		u.Matchs = append(u.Matchs, domain.TacticalMatch{MatchID: id, Outcome: domain.OutcomeWin})
	}
	return u
}

func svcEchange(repo *mockTacticalRepo, caps games.CapabilityMap) *TeammatesService {
	return &TeammatesService{
		titleSlug:    "halo_infinite",
		gamertag:     "main",
		tacticalRepo: repo,
		caps:         caps,
	}
}

// ─── LA MATRICE ────────────────────────────────────────────────────────────────────────

// TestBuildSquadEchange_Matrice : comptes EXACTS sur un journal pose a la main, et
// ORIENTATION vengeur -> venge.
//
// Le scenario de m1 (toutes les valeurs en ms) :
//
//	1000  adv1 tue main            -> vengeable
//	2000  Ami  tue adv1            -> VENGE main (delai 1 000)  [Ami venge main]
//	5000  adv2 tue Ami             -> vengeable
//	6500  main tue adv2            -> VENGE Ami  (delai 1 500)  [main venge Ami]
//	9000  adv1 tue Ami             -> vengeable, JAMAIS vengee
//
// Soit 3 morts vengeables du camp, 2 vengees, et DEUX cases de matrice distinctes,
// symetriques mais non confondues : (Ami -> main) et (main -> Ami).
func TestBuildSquadEchange_Matrice(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			{MatchID: "m1", KillerXUID: "x_adv2", VictimXUID: "x_Ami", TimeMs: 5000},
			{MatchID: "m1", KillerXUID: "x_main", VictimXUID: "x_adv2", TimeMs: 6500},
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_Ami", TimeMs: 9000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}

	// Le lecteur est interroge SANS carte : la page Escouade mesure une composition.
	if len(repo.vues) != 1 || repo.vues[0].MapID != "" || repo.vues[0].PlayerXUID != "x_main" {
		t.Fatalf("requete = %+v, attendue {PlayerXUID: x_main, MapID: \"\"}", repo.vues)
	}

	if got.Couverture.Brut != 2 || got.Couverture.N != 3 {
		t.Errorf("couverture = %d/%d, attendue 2/3", got.Couverture.Brut, got.Couverture.N)
	}
	if len(got.Cellules) != 2 {
		t.Fatalf("cellules = %+v, attendu 2", got.Cellules)
	}
	// L'orientation : LIGNE = celui qui venge, COLONNE = celui qui est venge.
	parCle := map[string]domain.SquadEchangeCell{}
	for _, c := range got.Cellules {
		parCle[c.VengeurGamertag+"->"+c.VengeGamertag] = c
	}
	if c, ok := parCle["Ami->main"]; !ok || c.Nombre != 1 {
		t.Errorf("case (Ami venge main) = %+v, attendue Nombre=1", c)
	}
	if c, ok := parCle["main->Ami"]; !ok || c.Nombre != 1 {
		t.Errorf("case (main venge Ami) = %+v, attendue Nombre=1", c)
	}
	// ParMatch : un compte brut sans denominateur ne se compare pas d'un filtre a l'autre.
	if got.Cellules[0].ParMatch != 1 {
		t.Errorf("par_match = %v sur 1 match, attendu 1", got.Cellules[0].ParMatch)
	}
	// Les axes de la matrice sont le ROSTER, joueur principal d'abord.
	if len(got.Joueurs) != 2 || got.Joueurs[0].Gamertag != "main" || got.Joueurs[1].Gamertag != "Ami" {
		t.Errorf("axes = %+v, attendus [main, Ami]", got.Joueurs)
	}
}

// TestBuildSquadEchange_MatriceEcarteHorsRoster : un vengeur de passage compte au KPI
// (il est de MON CAMP) mais n'a AUCUNE ligne dans la matrice — la page ne sait pas le
// nommer, et un xuid nu vaut moins qu'une ligne absente.
func TestBuildSquadEchange_MatriceEcarteHorsRoster(t *testing.T) {
	equipes := domain.EquipesParMatch{"m1": {
		"x_main": 0, "x_passant": 0, "x_adv1": 1,
	}}
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: domain.TacticalUnivers{
			Matchs:  []domain.TacticalMatch{{MatchID: "m1", Outcome: domain.OutcomeWin}},
			Equipes: equipes,
		},
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_passant", VictimXUID: "x_adv1", TimeMs: 3000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", nil)
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if got.Couverture.Brut != 1 || got.Couverture.N != 1 {
		t.Errorf("couverture = %d/%d, attendue 1/1 (le passant venge bien mon camp)",
			got.Couverture.Brut, got.Couverture.N)
	}
	if len(got.Cellules) != 0 {
		t.Errorf("cellules = %+v, attendues vides (le passant n'est pas au roster)", got.Cellules)
	}
}

// ─── LA FENETRE, AU MILLIER PRES ───────────────────────────────────────────────────────

// TestBuildSquadEchange_BucketsDelai : une vengeance a 4 999 ms est DANS la fenetre, a
// 5 001 ms elle est HORS. C'est la borne qui decide du taux, et l'intervalle qui la porte
// doit la respecter — un `delai / 1000` rangerait 5 000 ms hors fenetre.
func TestBuildSquadEchange_BucketsDelai(t *testing.T) {
	cas := []struct {
		nom           string
		delai         int64
		bucket        int
		dansLaFenetre bool
	}{
		{"immediat", 0, 0, true},
		{"juste sous la seconde", 999, 0, true},
		{"quatrieme seconde", 3200, 3, true},
		{"4 999 ms : DANS la fenetre", 4999, 4, true},
		{"5 000 ms : borne INCLUSE", 5000, 4, true},
		{"5 001 ms : HORS fenetre", 5001, 5, false},
		{"6 999 ms : dernier intervalle borne", 6999, 5, false},
		// Les intervalles sont SEMI-OUVERTS ([debut, fin[), sauf celui qui ferme la
		// fenetre : 7 000 ms ouvre donc l'intervalle non borne, comme son etiquette
		// (« au-dela de 7 s ») l'annonce.
		{"7 000 ms : ouvre l'intervalle non borne", 7000, 6, false},
		{"7 001 ms : intervalle ouvert", 7001, 6, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
				Univers: universDe("m1"),
				Events: []domain.KillEvent{
					{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 10_000},
					{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 10_000 + c.delai},
				},
			}}
			svc := svcEchange(repo, capsFiables())
			got := svc.buildSquadEchange(context.Background(),
				echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
			if got == nil {
				t.Fatal("section attendue, obtenu nil")
			}
			if len(got.Delais) != 7 {
				t.Fatalf("intervalles = %d, attendus 7 (5 dans la fenetre + 2 hors)", len(got.Delais))
			}
			for i, b := range got.Delais {
				attendu := 0
				if i == c.bucket {
					attendu = 1
				}
				if b.Nombre != attendu {
					t.Errorf("intervalle %d (%d-%d ms) = %d, attendu %d",
						i, b.DebutMs, b.FinMs, b.Nombre, attendu)
				}
			}
			if got.Delais[c.bucket].HorsFenetre == c.dansLaFenetre {
				t.Errorf("intervalle %d hors_fenetre = %v, attendu %v",
					c.bucket, got.Delais[c.bucket].HorsFenetre, !c.dansLaFenetre)
			}
			// Le taux, lui, ne compte QUE la fenetre : une riposte a 5 001 ms est
			// montree et n'est vengee de rien.
			vengees := 0
			if c.dansLaFenetre {
				vengees = 1
			}
			if got.Couverture.Brut != vengees {
				t.Errorf("morts vengees = %d, attendu %d (delai %d ms)",
					got.Couverture.Brut, vengees, c.delai)
			}
		})
	}
}

// TestBuildSquadEchange_DernierIntervalleOuvert : le dernier intervalle n'a pas de borne
// haute, et il est publie comme tel (le client ecrit « au-dela de 7 s », pas « 7-0 s »).
func TestBuildSquadEchange_DernierIntervalleOuvert(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
		},
	}}
	got := svcEchange(repo, capsFiables()).buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	dernier := got.Delais[len(got.Delais)-1]
	if !dernier.Ouvert || dernier.FinMs != 0 {
		t.Errorf("dernier intervalle = %+v, attendu ouvert sans borne haute", dernier)
	}
	if got.FenetreMs != coordination.FenetreEchangeMs {
		t.Errorf("fenetre publiee = %d, attendue %d", got.FenetreMs, coordination.FenetreEchangeMs)
	}
}

// ─── LE CAMP, ET LA REFERENCE HABITUELLE ───────────────────────────────────────────────

// TestBuildSquadEchange_CampEtHabituel : le KPI porte sur MON CAMP ENTIER (moi ET mes
// coequipiers du match), jamais sur mes seules morts ; et l'habituel est la MEME mesure
// sur l'historique complet de la composition, dont le perimetre filtre est un
// sous-ensemble.
//
// m1 (dans le perimetre) : ma mort vengee, celle d'Ami non vengee -> 1/2.
// m2 (hors perimetre filtre, dans l'habituel) : ma mort vengee -> l'habituel fait 2/3.
// Une mort ADVERSE, vengee, ne doit compter NULLE PART.
func TestBuildSquadEchange_CampEtHabituel(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1", "m2"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
			{MatchID: "m1", KillerXUID: "x_adv2", VictimXUID: "x_Ami", TimeMs: 20_000},
			// Une mort adverse, vengee par l'autre adversaire : hors de mon camp.
			{MatchID: "m1", KillerXUID: "x_main", VictimXUID: "x_adv2", TimeMs: 30_000},
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 31_000},

			{MatchID: "m2", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m2", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 3000},
		},
	}}
	svc := svcEchange(repo, capsFiables())

	got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1", "m2"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if got.Couverture.Brut != 1 || got.Couverture.N != 3 {
		t.Errorf("perimetre filtre = %d/%d, attendu 1/3 (mes 2 morts + celle d'Ami)",
			got.Couverture.Brut, got.Couverture.N)
	}
	if got.Habituel.Brut != 2 || got.Habituel.N != 4 {
		t.Errorf("habituel = %d/%d, attendu 2/4", got.Habituel.Brut, got.Habituel.N)
	}
	if got.MatchsTotal != 1 || got.MatchsHabituel != 2 {
		t.Errorf("matchs = %d / habituel %d, attendus 1 / 2", got.MatchsTotal, got.MatchsHabituel)
	}
	// Bandeau de couverture : les films expirent, le manque est definitif.
	if got.MatchsMesures != 1 {
		t.Errorf("matchs mesures = %d, attendu 1", got.MatchsMesures)
	}
	// Sous le plancher de 30 morts, la mesure existe et ne classe personne.
	if !got.Couverture.EchantillonFaible {
		t.Error("echantillon faible attendu a 3 morts (plancher 30)")
	}
}

// TestBuildSquadEchange_MatchMuetAuDenominateur : un match RETENU sans aucune mort
// mesuree reste au denominateur « par match » de la matrice — le deduire des evenements
// l'effacerait (defaut mesure en phase 1 du plan tactique).
func TestBuildSquadEchange_MatchMuetAuDenominateur(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1", "m2"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
		},
	}}
	got := svcEchange(repo, capsFiables()).buildSquadEchange(context.Background(),
		echangeRows("m1", "m2"), echangeRows("m1", "m2"), "main", "x_main", echangeMates("Ami"))
	if got == nil {
		t.Fatal("section attendue, obtenu nil")
	}
	if got.MatchsTotal != 2 || got.MatchsMesures != 1 {
		t.Errorf("couverture = %d/%d, attendue 1 mesure sur 2", got.MatchsMesures, got.MatchsTotal)
	}
	if len(got.Cellules) != 1 || got.Cellules[0].ParMatch != 0.5 {
		t.Errorf("par_match = %+v, attendu 0,5 (1 echange sur 2 matchs RETENUS)", got.Cellules)
	}
	if got.Couverture.ParMatch != 0.5 {
		t.Errorf("couverture par match = %v, attendue 0,5", got.Couverture.ParMatch)
	}
}

// ─── LES PORTES ────────────────────────────────────────────────────────────────────────

// TestBuildSquadEchange_PorteFermee : un titre qui ne nomme pas le tueur de chaque mort
// n'a PAS de section — pas une section a zero.
//
// `match.killfeed.per_kill = degraded` est le cas exact que la porte doit refuser : des
// kills simultanes omis se lisent comme des morts « non vengees », et fabriqueraient un
// taux d'echange faux. Obtenu par les CAPABILITIES, jamais par le slug.
func TestBuildSquadEchange_PorteFermee(t *testing.T) {
	journal := domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "x_Ami", VictimXUID: "x_adv1", TimeMs: 2000},
		},
	}
	cas := map[string]games.CapabilityMap{
		"aucune capability":       nil,
		"kill-feed natif degrade": {games.CapMatchKillfeedPerKill: games.CapDegraded},
	}
	for nom, caps := range cas {
		t.Run(nom, func(t *testing.T) {
			repo := &mockTacticalRepo{lecture: journal}
			svc := svcEchange(repo, caps)
			if got := svc.buildSquadEchange(context.Background(),
				echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"),
			); got != nil {
				t.Fatalf("section = %+v, attendue absente", got)
			}
			if len(repo.vues) != 0 {
				t.Error("aucune lecture de base ne doit partir quand la porte est fermee")
			}
		})
	}
}

// TestBuildSquadEchange_PorteOuverteParLeKillfeedNatif : la seconde provenance
// (`match.killfeed.per_kill = supported`, Halo 5) ouvre la porte a elle seule — c'est ce
// qui evite qu'un titre sans decodeur de film soit prive d'une jointure qui marche.
func TestBuildSquadEchange_PorteOuverteParLeKillfeedNatif(t *testing.T) {
	repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{
		Univers: universDe("m1"),
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "x_adv1", VictimXUID: "x_main", TimeMs: 1000},
		},
	}}
	svc := svcEchange(repo, games.CapabilityMap{games.CapMatchKillfeedPerKill: games.CapSupported})
	if got := svc.buildSquadEchange(context.Background(),
		echangeRows("m1"), echangeRows("m1"), "main", "x_main", echangeMates("Ami"),
	); got == nil {
		t.Fatal("section attendue : le kill-feed natif `supported` suffit")
	}
}

// TestBuildSquadEchange_DegradationsSilencieuses : lecteur non cable, journal en echec,
// perimetre vide, ou aucun match mesure -> section absente, jamais des zeros. La page
// garde tous ses autres blocs.
func TestBuildSquadEchange_DegradationsSilencieuses(t *testing.T) {
	rows := echangeRows("m1")
	mates := echangeMates("Ami")

	t.Run("lecteur non cable", func(t *testing.T) {
		svc := &TeammatesService{gamertag: "main", caps: capsFiables()}
		if got := svc.buildSquadEchange(context.Background(), rows, rows, "main", "x_main", mates); got != nil {
			t.Fatalf("section = %+v, attendue absente", got)
		}
	})
	t.Run("journal en echec", func(t *testing.T) {
		repo := &mockTacticalRepo{err: errors.New("shared reader indisponible")}
		if got := svcEchange(repo, capsFiables()).buildSquadEchange(
			context.Background(), rows, rows, "main", "x_main", mates); got != nil {
			t.Fatalf("section = %+v, attendue absente", got)
		}
	})
	t.Run("perimetre vide", func(t *testing.T) {
		repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{Univers: universDe("m1")}}
		if got := svcEchange(repo, capsFiables()).buildSquadEchange(
			context.Background(), nil, nil, "main", "x_main", mates); got != nil {
			t.Fatalf("section = %+v, attendue absente", got)
		}
	})
	t.Run("aucun match mesure", func(t *testing.T) {
		// L'univers connait le match, le journal n'en dit rien : le film a expire.
		repo := &mockTacticalRepo{lecture: domain.TacticalKillEvents{Univers: universDe("m1")}}
		if got := svcEchange(repo, capsFiables()).buildSquadEchange(
			context.Background(), rows, rows, "main", "x_main", mates); got != nil {
			t.Fatalf("section = %+v, attendue absente (aucun journal sur le perimetre)", got)
		}
	})
}
