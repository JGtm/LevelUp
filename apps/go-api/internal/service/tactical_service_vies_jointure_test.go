package service

// tactical_service_vies_jointure_test.go — ce que `tactical_service_vies_test.go` ne tient pas :
//   - `toutes` EXIGE `film.spawn_forecast` (le titre synthetique, qui a `film.replay_artifact`
//     sans elle, le refuse ; halo_infinite l'accepte) ;
//   - la jointure route / vie (`DebutFrame` = `Frame`) est PAR MATCH : deux matchs dont les vies
//     n'ont pas les memes instants ne s'echangent rien ;
//   - deux vies d'un meme match parties de la meme grappe comptent toutes les deux.

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// capsAvecReapparitions : l'occupation, plus la capacite qui dit que les sidecars portent les
// reapparitions.
func capsAvecReapparitions() games.CapabilityMap {
	c := capsOccupation()
	c[games.CapFilmSpawnForecast] = games.CapSupported
	return c
}

// svcVies monte le service des tests `vies` : un titre qui sait ce que `toutes` demande.
func svcVies(univ domain.TacticalUnivers, store *mockRasterStore) *TacticalService {
	repo := &mockTacticalRepo{univ: univ}
	return NewTacticalService(repo, capsAvecReapparitions(), tsMoi).WithRasterStore(store)
}

// capsDuTitre lit la carte de capacites REELLE d'un titre livre.
func capsDuTitre(t *testing.T, slug string) games.CapabilityMap {
	t.Helper()
	_, ce, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	racine := filepath.Join(filepath.Dir(ce), "..", "..", "..", "..")
	caps, err := games.LoadCapabilityMap(racine, slug)
	if err != nil {
		t.Fatalf("LoadCapabilityMap(%s): %v", slug, err)
	}
	return caps
}

// TestVies_ToutesExigeSpawnForecast — le titre synthetique declare `film.replay_artifact` mais pas
// `film.spawn_forecast` : ses sidecars ne portent que la premiere vie, `toutes` est refuse (503
// cote HTTP) pour la lecture comme pour le detail ; `depart` reste servi. halo_infinite accepte.
func TestVies_ToutesExigeSpawnForecast(t *testing.T) {
	synthetique := capsDuTitre(t, "synthetic_title_b")
	if !synthetique.Has(games.CapFilmReplayArtifact) || synthetique.Has(games.CapFilmSpawnForecast) {
		t.Fatalf("le titre synthetique a change de capacites : replay_artifact=%v spawn_forecast=%v",
			synthetique.Has(games.CapFilmReplayArtifact), synthetique.Has(games.CapFilmSpawnForecast))
	}
	store, univ, ids := viesFixture()
	svcDe := func(caps games.CapabilityMap) *TacticalService {
		return NewTacticalService(&mockTacticalRepo{univ: univ}, caps, tsMoi).WithRasterStore(store)
	}
	raster := func(svc *TacticalService, vies string) error {
		_, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
			MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
			Scope: domain.TacticalScope{MatchIDs: ids, Vies: vies},
		})
		return err
	}
	cellule := func(svc *TacticalService, vies string) error {
		_, err := svc.Cellule(context.Background(), domain.TacticalCelluleRequest{
			MapID: "streets", Question: domain.TacticalQuestionRoutes, Qui: domain.TacticalQuiMoi,
			Scope: domain.TacticalScope{MatchIDs: ids, Vies: vies}, Col: 10, Lig: 10,
		})
		return err
	}
	for nom, appel := range map[string]func(*TacticalService, string) error{"raster": raster, "cellule": cellule} {
		if err := appel(svcDe(synthetique), domain.TacticalViesToutes); !errors.Is(err, games.ErrCapabilityNotSupported) {
			t.Fatalf("%s, titre synthetique, toutes : err = %v, attendu ErrCapabilityNotSupported", nom, err)
		}
		if err := appel(svcDe(synthetique), domain.TacticalViesDepart); err != nil {
			t.Fatalf("%s, titre synthetique, depart : %v, attendu une lecture servie", nom, err)
		}
		if err := appel(svcDe(capsDuTitre(t, "halo_infinite")), domain.TacticalViesToutes); err != nil {
			t.Fatalf("%s, halo_infinite, toutes : %v, attendu une lecture servie", nom, err)
		}
	}
}

// vieDeTest decrit une vie posee a la main : son instant, ou elle part, le drapeau qui en fait une
// reapparition (la premiere vie est toujours la premiere ; sans drapeau, une piste n'est pas une
// apparition) et la cellule (col = lig) que sa route traverse.
type vieDeTest struct {
	frame        int
	x, y         float64
	reapparition bool
	cellule      int
}

// sidecarDeVies pose un joueur aux vies donnees ; la premiere est la premiere vie.
func sidecarDeVies(matchID string, vies ...vieDeTest) *domain.TacticalRasterSidecar {
	j := domain.TacticalRasterJoueur{XUID: tsMoi}
	for i, v := range vies {
		j.Spawns = append(j.Spawns, domain.TacticalRasterSpawn{
			Frame: v.frame, X: v.x, Y: v.y, PremiereVie: i == 0, Reapparition: i > 0 && v.reapparition,
		})
		j.Routes = append(j.Routes, domain.TacticalRasterRoute{
			DebutFrame: v.frame, Cases: []domain.TacticalRasterCase{{Col: v.cellule, Lig: v.cellule}},
		})
	}
	sc := sidecarPose(matchID)
	sc.Joueurs = []domain.TacticalRasterJoueur{j}
	return sc
}

func universDeMatchs(ids ...string) domain.TacticalUnivers {
	v := map[string]string{}
	for _, id := range ids {
		v[id] = "Slayer:Arena"
	}
	return universVariantes(v)
}

// TestVies_LaJointureEstParMatch — chaque match a ses propres instants. Les reapparitions de la
// grappe B tombent aux frames 1000, 2500 et 4000 ; chaque match porte en plus une piste SANS drapeau
// (route par la cellule 70) dont l'instant est celui de la reapparition d'un AUTRE match. Si la
// jointure melangeait les matchs (l'ensemble des instants de la grappe valable pour tous), cette
// route passerait le filtre ; elle ne doit pas.
func TestVies_LaJointureEstParMatch(t *testing.T) {
	instants := []int{1000, 2500, 4000}
	sc := map[string]*domain.TacticalRasterSidecar{}
	ids := []string{}
	for i, id := range []string{"j1", "j2", "j3"} {
		dx := float64(i) * 0.4
		sc[id] = sidecarDeVies(id,
			vieDeTest{frame: 0, x: 0.25 + dx, y: 0.25, cellule: 10},
			vieDeTest{frame: instants[i], x: 50.25 + dx, y: 50.25, reapparition: true, cellule: 30},
			vieDeTest{frame: instants[(i+1)%3], x: 150.25 + dx, y: 150.25, cellule: 70},
		)
		ids = append(ids, id)
	}
	svc := svcVies(universDeMatchs(ids...), &mockRasterStore{sidecars: sc})
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids).Grappes
	reapparition := grappeAutour(t, grappes, 50, 50)

	out := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, reapparition, ids)
	if !aLaCellule(out, 30) {
		t.Fatalf("la route de la vie partie de la grappe manque : %+v", out.Cellules)
	}
	if aLaCellule(out, 70) {
		t.Fatalf("la route d'une piste sans drapeau, dont l'instant est celui d'une reapparition d'un autre match, passe le filtre : %+v", out.Cellules)
	}
	if aLaCellule(out, 10) {
		t.Fatalf("la route de la premiere vie, partie d'ailleurs, est peinte sous le filtre : %+v", out.Cellules)
	}
	// Le detail d'une cellule suit la meme jointure.
	for cellule, attendu := range map[int]int{30: 3, 70: 0, 10: 0} {
		if n := len(celluleEnVies(t, svc, domain.TacticalViesToutes, reapparition, ids, cellule).Contributions); n != attendu {
			t.Fatalf("detail de la cellule %d : %d contributions, attendu %d", cellule, n, attendu)
		}
	}
}

// TestVies_DeuxViesDUnMemeMatchDansLaMemeGrappe — la premiere vie et une reapparition d'un meme
// match partent de la meme grappe : les deux routes comptent, et le match n'est compte qu'une fois.
func TestVies_DeuxViesDUnMemeMatchDansLaMemeGrappe(t *testing.T) {
	sc := map[string]*domain.TacticalRasterSidecar{}
	ids := []string{}
	for i, id := range []string{"k1", "k2", "k3"} {
		dx := float64(i) * 0.4
		sc[id] = sidecarDeVies(id,
			vieDeTest{frame: 0, x: 0.25 + dx, y: 0.25, cellule: 10},
			vieDeTest{frame: 800, x: 0.25 + dx, y: 0.75, reapparition: true, cellule: 20},
			vieDeTest{frame: 1600, x: 90.25 + dx, y: 90.25, reapparition: true, cellule: 40},
		)
		ids = append(ids, id)
	}
	svc := svcVies(universDeMatchs(ids...), &mockRasterStore{sidecars: sc})
	grappes := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, "", ids).Grappes
	depart := grappeAutour(t, grappes, 0, 0)

	out := lireEnVies(t, svc, domain.TacticalQuestionRoutes, domain.TacticalViesToutes, depart, ids)
	if out.MatchsRetenus != 3 || out.MatchsFiltres != 3 {
		t.Fatalf("matchs retenus/filtres = %d/%d, attendu 3/3 (un match n'est compte qu'une fois)", out.MatchsRetenus, out.MatchsFiltres)
	}
	for _, c := range []int{10, 20} {
		if !aLaCellule(out, c) {
			t.Fatalf("la route de la vie partie de la grappe (cellule %d) manque : %+v", c, out.Cellules)
		}
		if n := len(celluleEnVies(t, svc, domain.TacticalViesToutes, depart, ids, c).Contributions); n != 3 {
			t.Fatalf("detail de la cellule %d : %d contributions, attendu 3", c, n)
		}
	}
	if aLaCellule(out, 40) {
		t.Fatalf("la route de la vie partie d'une autre grappe est peinte : %+v", out.Cellules)
	}
}
