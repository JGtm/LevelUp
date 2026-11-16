package service

// tactical_service_reapparitions_adverses_test.go — LA LECTURE « REAPPARITIONS ADVERSES » du service, sur
// des sidecars poses a la main : la porte (capability), la regle de ce qu'est une
// reapparition adverse, la zone tenue, les planchers et les raisons d'un etat vide.
//
// Le corpus : une carte, deux zones nommees — « Hall » (la zone choisie, carre de 0 a 10 m) et « Sortie »
// (carre de 20 a 30 m, ou les adversaires reapparaissent). Moi et un ami (equipe 0) contre un adversaire
// (equipe 1).

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/i18n"
)

const (
	volHall   = 1
	volSortie = 2
)

func carreZone(x0, y0, x1, y1 float64) [][2]float64 {
	return [][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

// nomSortie : le nom de la zone « Sortie », un libelle par langue.
var nomSortie = i18n.FromColumns("Sortie", "Exit")

func zonesAdverses() *mockCallouts {
	return &mockCallouts{zones: []domain.ZoneNommee{
		{Nom: i18n.FromColumns("Hall", "Hall"), VolumeIndex: volHall, Polygone: carreZone(0, 0, 10, 10), ZBas: -1, ZHaut: 3, Big: true},
		{Nom: nomSortie, VolumeIndex: volSortie, Polygone: carreZone(20, 0, 30, 10), ZBas: -1, ZHaut: 3, Big: true},
	}}
}

// reappAdverse : une reapparition de l'adversaire au point (x, y), avec `equipiers` joueurs de l'equipe 0
// dans la zone « Hall » a cet instant (plus un joueur de l'equipe 1 ailleurs).
func reappAdverse(frame int, x, y float64, equipiers int) domain.TacticalRasterSpawn {
	var inst []domain.TacticalRasterVivant
	for i := 0; i < equipiers; i++ {
		inst = append(inst, domain.TacticalRasterVivant{XUID: tsMoi, Equipe: 0, X: 2 + float64(i), Y: 2})
	}
	inst = append(inst, domain.TacticalRasterVivant{Equipe: 1, X: 60, Y: 60})
	return domain.TacticalRasterSpawn{
		Frame: frame, X: x, Y: y, Reapparition: true,
		Mort: &domain.TacticalRasterMort{Frame: frame - 100}, Instantane: inst,
	}
}

// sidecarAdverse : le match `id` ou l'adversaire reapparait `tenues` fois a Sortie quand Hall est tenu et
// `libres` fois quand il ne l'est pas.
func sidecarAdverse(id string, tenues, libres int) *domain.TacticalRasterSidecar {
	adv := domain.TacticalRasterJoueur{XUID: tsAdv, Equipe: 1,
		Spawns: []domain.TacticalRasterSpawn{{Frame: 0, X: 25, Y: 5, PremiereVie: true}}}
	frame := 1000
	for i := 0; i < tenues; i++ {
		adv.Spawns = append(adv.Spawns, reappAdverse(frame, 25, 5, 2))
		frame += 1000
	}
	for i := 0; i < libres; i++ {
		adv.Spawns = append(adv.Spawns, reappAdverse(frame, 25, 5, 1))
		frame += 1000
	}
	sc := sidecarPose(id)
	sc.Joueurs = []domain.TacticalRasterJoueur{
		{XUID: tsMoi, Equipe: 0}, {XUID: tsAmi, Equipe: 0}, adv,
	}
	return sc
}

func idsDe(n int) []string {
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, fmt.Sprintf("m%d", i))
	}
	return out
}

// svcAdverses monte le service : titre qui porte les reapparitions, zones.
func svcAdverses(t *testing.T, univ domain.TacticalUnivers, store *mockRasterStore) (*TacticalService, *mockTacticalRepo) {
	t.Helper()
	repo := &mockTacticalRepo{univ: univ}
	svc := NewTacticalService(repo, capsAvecReapparitions(), tsMoi).
		WithRasterStore(store).WithCalloutsStore(zonesAdverses())
	return svc, repo
}

// matchsAssassin : n matchs d'assassin au sidecar donne.
func matchsAssassin(n, tenues, libres int) (domain.TacticalUnivers, *mockRasterStore) {
	variantes := map[string]string{}
	sc := map[string]*domain.TacticalRasterSidecar{}
	for _, id := range idsDe(n) {
		variantes[id] = "Slayer:Arena"
		sc[id] = sidecarAdverse(id, tenues, libres)
	}
	return universVariantes(variantes), &mockRasterStore{sidecars: sc}
}

func lireAdverses(svc *TacticalService, ids []string, vol int) (domain.TacticalReapparitionsAdverses, error) {
	return svc.ReapparitionsAdverses(i18n.WithContext(context.Background(), i18n.Default),
		domain.TacticalReapparitionsAdversesRequest{
			MapID: "streets", VolumeIndex: vol, Scope: domain.TacticalScope{MatchIDs: ids},
		})
}

// Les comptes de matchs de la lecture (journalises, plus servis) : un match sans sidecar n'est dans aucun.
func TestObservationsAdverses_ComptesDeMatchs(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 3)
	svc, _ := svcAdverses(t, univ, store)
	if obs, matchs := svc.observationsAdverses(univ, store.sidecars); matchs.retenus != 5 || matchs.sansEquipe != 0 || len(obs) != 35 {
		t.Errorf("comptes = %+v (%d observations), attendu 5 retenus, 0 sans equipe, 35 observations", matchs, len(obs))
	}
	delete(store.sidecars, "m1")
	if _, matchs := svc.observationsAdverses(univ, store.sidecars); matchs.retenus != 4 {
		t.Errorf("un match sans sidecar n'est pas mesure : %+v", matchs)
	}
}

func TestReapparitionsAdverses_LectureServie(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 3) // par match : 4 reapparitions « Hall tenu », 3 sans
	svc, _ := svcAdverses(t, univ, store)
	out, err := lireAdverses(svc, idsDe(5), volHall)
	if err != nil {
		t.Fatal(err)
	}
	if out.Etat != domain.TacticalEtatServie || out.Raison != "" {
		t.Fatalf("etat = %q (%q), attendu servie : %+v", out.Etat, out.Raison, out)
	}
	if out.Zone.Nom != "Hall" {
		t.Errorf("zone = %+v", out.Zone)
	}
	if out.Reference.Reapparitions != 35 || !out.Reference.Suffisante {
		t.Errorf("reference = %+v, attendu 35 reapparitions (la premiere vie n'en est pas une)", out.Reference)
	}
	if out.Tenue.Reapparitions != 20 || !out.Tenue.Suffisante || len(out.Tenue.Zones) != 1 ||
		out.Tenue.Zones[0].VolumeIndex != volSortie || out.Tenue.Zones[0].Nom != nomSortie.Pick(i18n.Default) ||
		out.Tenue.Zones[0].Part != 1 {
		t.Errorf("tenue = %+v, attendu 20 reapparitions, toutes a Sortie", out.Tenue)
	}
	if out.Plancher.Matchs != 3 || out.Plancher.Reapparitions != 20 || out.Plancher.JoueursPourTenir != 2 {
		t.Errorf("plancher = %+v", out.Plancher)
	}
}

// Le nom de la zone suit la langue de la requete, dans chacune des langues du produit.
func TestReapparitionsAdverses_NomDansLaLangueDeLaRequete(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 0)
	svc, _ := svcAdverses(t, univ, store)
	for _, loc := range i18n.Locales() {
		out, err := svc.ReapparitionsAdverses(i18n.WithContext(context.Background(), loc),
			domain.TacticalReapparitionsAdversesRequest{MapID: "streets", VolumeIndex: volSortie,
				Scope: domain.TacticalScope{MatchIDs: idsDe(5)}})
		if err != nil {
			t.Fatal(err)
		}
		if out.Zone.Nom != nomSortie.Pick(loc) || len(out.Reference.Zones) != 1 || out.Reference.Zones[0].Nom != nomSortie.Pick(loc) {
			t.Errorf("%s : zone %q, reference %+v, attendu %q", loc, out.Zone.Nom, out.Reference.Zones, nomSortie.Pick(loc))
		}
	}
}

func TestReapparitionsAdverses_SousLePlancherDEffectif(t *testing.T) {
	univ, store := matchsAssassin(5, 3, 3) // « Hall tenu » : 15 reapparitions, sous les 20 ; reference : 30
	svc, _ := svcAdverses(t, univ, store)
	out, err := lireAdverses(svc, idsDe(5), volHall)
	if err != nil {
		t.Fatal(err)
	}
	if out.Etat != domain.TacticalEtatVide || out.Raison != domain.TacticalRaisonSousPlancherReapparitions {
		t.Fatalf("etat %q raison %q, attendu vide / sous_plancher_reapparitions", out.Etat, out.Raison)
	}
	if out.Tenue.Reapparitions != 15 || out.Tenue.Suffisante || len(out.Tenue.Zones) != 0 {
		t.Errorf("tenue = %+v : l'effectif se lit, les zones ne sont pas servies", out.Tenue)
	}
	if !out.Reference.Suffisante || out.Reference.Reapparitions != 30 {
		t.Errorf("reference = %+v : elle est suffisante a elle seule", out.Reference)
	}
}

func TestReapparitionsAdverses_SousLePlancherDeMatchs(t *testing.T) {
	univ, store := matchsAssassin(2, 20, 0)
	svc, _ := svcAdverses(t, univ, store)
	out, err := lireAdverses(svc, idsDe(2), volHall)
	if err != nil {
		t.Fatal(err)
	}
	if out.Etat != domain.TacticalEtatVide || out.Raison != domain.TacticalRaisonSousPlancherMatchs {
		t.Fatalf("etat %q raison %q, attendu vide / sous_plancher_matchs", out.Etat, out.Raison)
	}
}

// Aucun filtre par mode : tous les modes se melangent et tous les matchs mesures nourrissent la lecture,
// l'oddball et le BTB comme l'assassin. Seuls les planchers de donnees decident.
func TestReapparitionsAdverses_AucunFiltreParMode(t *testing.T) {
	univ, store := matchsAssassin(6, 4, 3)
	univ.Matchs[0].GameVariantName = "Oddball:Arena"
	univ.Matchs[1].GameVariantName = "BTB:Heavies"
	svc, _ := svcAdverses(t, univ, store)
	out, err := lireAdverses(svc, idsDe(6), volHall)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reference.Reapparitions != 42 || out.Etat != domain.TacticalEtatServie {
		t.Errorf("reponse = %+v", out)
	}
}

// Capability absente : refus propre, sans rien lire.
func TestReapparitionsAdverses_CapabilityAbsente(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 3)
	for nom, caps := range map[string]games.CapabilityMap{
		"sans film.spawn_forecast":  capsOccupation(),
		"sans film.replay_artifact": {games.CapFilmSpawnForecast: games.CapSupported},
		"carte nulle":               nil,
	} {
		repo := &mockTacticalRepo{univ: univ}
		svc := NewTacticalService(repo, caps, tsMoi).WithRasterStore(store).
			WithCalloutsStore(zonesAdverses())
		_, err := lireAdverses(svc, idsDe(5), volHall)
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			t.Errorf("%s : erreur %v, attendu ErrCapabilityNotSupported", nom, err)
		}
		if repo.vuUniv.PlayerXUID != "" {
			t.Errorf("%s : l'univers ne doit pas etre lu quand la porte refuse", nom)
		}
	}
	// Lecteur de sidecars non cable : defaut de cablage, meme refus propre.
	svc := NewTacticalService(&mockTacticalRepo{univ: univ}, capsAvecReapparitions(), tsMoi)
	if _, err := lireAdverses(svc, idsDe(5), volHall); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("sans lecteur de sidecars : %v", err)
	}
}

func TestReapparitionsAdverses_Refus(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 3)
	svc, _ := svcAdverses(t, univ, store)
	if _, err := lireAdverses(svc, idsDe(5), 99); !errors.Is(err, domain.ErrTacticalZoneInconnue) {
		t.Errorf("zone inconnue : %v", err)
	}
	if _, err := lireAdverses(svc, nil, volHall); !errors.Is(err, domain.ErrTacticalCarteInconnue) {
		t.Errorf("aucun match retenu : %v", err)
	}
	_, err := svc.ReapparitionsAdverses(context.Background(), domain.TacticalReapparitionsAdversesRequest{
		MapID: "streets", VolumeIndex: volHall,
		Scope: domain.TacticalScope{MatchIDs: idsDe(5), Coequipiers: []string{"a", "b", "c", "d"}},
	})
	if !errors.Is(err, domain.ErrTacticalCompositionInvalide) {
		t.Errorf("composition hors bornes : %v", err)
	}
}

// Les raisons d'un etat vide, de la plus generale a la plus precise.
func TestReapparitionsAdverses_RaisonsDEtatVide(t *testing.T) {
	// Aucun sidecar : le film n'est pas decode.
	univ, _ := matchsAssassin(5, 4, 3)
	svc, _ := svcAdverses(t, univ, &mockRasterStore{sidecars: map[string]*domain.TacticalRasterSidecar{}})
	out, err := lireAdverses(svc, idsDe(5), volHall)
	if err != nil || out.Raison != domain.TacticalRaisonAucunMatchMesure {
		t.Errorf("sans sidecar : (%+v, %v)", out, err)
	}

	// Le joueur n'a pas d'equipe nommee dans les sidecars.
	univ2, store2 := matchsAssassin(5, 4, 3)
	for _, sc := range store2.sidecars {
		sc.Joueurs[0].Equipe = -1
	}
	svc2, _ := svcAdverses(t, univ2, store2)
	out2, _ := lireAdverses(svc2, idsDe(5), volHall)
	if out2.Raison != domain.TacticalRaisonEquipeInconnue {
		t.Errorf("equipe inconnue : %+v", out2)
	}
	if _, matchs := svc2.observationsAdverses(univ2, store2.sidecars); matchs.sansEquipe != 5 || matchs.retenus != 0 {
		t.Errorf("comptes de matchs sans equipe = %+v, attendu 5 sans equipe et 0 retenu", matchs)
	}

	// Une zone sans forme ne contient personne.
	univ3, store3 := matchsAssassin(5, 4, 3)
	zones := zonesAdverses()
	zones.zones = append(zones.zones, domain.ZoneNommee{Nom: i18n.FromColumns("Fantome", "Ghost"), VolumeIndex: 7})
	svc3, _ := svcAdverses(t, univ3, store3)
	svc3.WithCalloutsStore(zones)
	out3, err := lireAdverses(svc3, idsDe(5), 7)
	if err != nil || out3.Raison != domain.TacticalRaisonZoneSansForme || out3.Etat != domain.TacticalEtatVide {
		t.Errorf("zone sans forme : (%+v, %v)", out3, err)
	}
}

// Ce qui est une reapparition adverse : une reapparition apres mort d'un joueur de l'AUTRE equipe. Ni un
// debut de manche, ni une premiere vie, ni un equipier, ni un joueur sans equipe.
func TestReapparitionsAdverses_QuiCompte(t *testing.T) {
	sc := sidecarAdverse("m1", 0, 0)
	adv := &sc.Joueurs[2]
	adv.Spawns = append(adv.Spawns,
		reappAdverse(1000, 25, 5, 2),
		domain.TacticalRasterSpawn{Frame: 2000, X: 25, Y: 5, DebutDeManche: true, Instantane: []domain.TacticalRasterVivant{{Equipe: 0, X: 2, Y: 2}}},
		domain.TacticalRasterSpawn{Frame: 3000, X: 25, Y: 5}, // piste sans drapeau
	)
	sc.Joueurs[1].Spawns = []domain.TacticalRasterSpawn{reappAdverse(1500, 25, 5, 2)} // un equipier
	sc.Joueurs = append(sc.Joueurs, domain.TacticalRasterJoueur{XUID: tsAdv2, Equipe: -1,
		Spawns: []domain.TacticalRasterSpawn{reappAdverse(1700, 25, 5, 2)}}) // equipe inconnue

	obs, connue := reapparitionsAdversesDuMatch("m1", sc, tsMoi)
	if !connue || len(obs) != 1 {
		t.Fatalf("(%d observations, equipe connue %v), attendu 1 (la seule reapparition adverse)", len(obs), connue)
	}
	if obs[0].EquipeActive != 0 || len(obs[0].Instantane) != 3 {
		t.Errorf("observation = %+v", obs[0])
	}
	if _, connue := reapparitionsAdversesDuMatch("m1", sc, "inconnu"); connue {
		t.Error("un joueur absent du sidecar n'a pas d'equipe connue")
	}
}

// La liste blanche du perimetre est honoree : un match hors liste n'entre pas dans la lecture.
func TestReapparitionsAdverses_LeperimetreEstUneListeBlanche(t *testing.T) {
	univ, store := matchsAssassin(5, 4, 3)
	svc, _ := svcAdverses(t, univ, store)
	out, err := lireAdverses(svc, idsDe(3), volHall) // trois matchs sur cinq
	if err != nil {
		t.Fatal(err)
	}
	if out.Reference.Reapparitions != 21 {
		t.Errorf("reponse = %+v", out)
	}
}
