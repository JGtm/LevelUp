package replay

import (
	"math"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// flag_film_bases_test.go — LES BASES DU DRAPEAU LUES DANS LE FILM (flag_film_bases.go).
//
// Le banc : le joueur "1" (equipe 0) vole au point (2, 2), le joueur "2" (equipe 1) vole au point
// (98, 98). Les vols de l'equipe 0 tombent a la base de l'equipe 1, et reciproquement.

// flagFilmBasesScan rend le balayage du banc, sur les socles et les vies libres donnes.
func flagFilmBasesScan(spawns []FlagSpawn, teams map[string]int, free []flagFreeLife) FlagCarryScan {
	return FlagCarryScan{
		Scanned: true, Signals: flagTestSignals(),
		Events: []objectives.NamedEvent{
			{TimeMS: 1000, Slot: 12, Stat: objectives.StatFlagSteals},
			{TimeMS: 3000, Slot: 14, Stat: objectives.StatFlagSteals},
		},
		Identity: objectives.FlatRoundIdentity(map[int]string{12: "1", 14: "2"}),
		Spawns:   spawns, TeamOf: teams, Free: free,
	}
}

// flagFilmBasesCtx rend le contexte du banc : les deux voleurs immobiles a leur point de vol.
func flagFilmBasesCtx(a, b [2]float32) flagCarryCtx {
	ctx := flagTestCtx([]Track{flagTestTrack(10, "1", 0, 99, a[0], a[1]),
		flagTestTrack(11, "2", 0, 99, b[0], b[1])}, nil, 100)
	ctx.fb = fallback.NouveauCompteur()
	return ctx
}

// flagFilmBasesRebirths : deux renaissances de l'objet au point exact de chaque socle.
func flagFilmBasesRebirths() []flagFreeLife {
	return []flagFreeLife{
		flagTestLife(0, [2]float32{0, 0}), flagTestLife(1, [2]float32{0.004, 0}),
		flagTestLife(0, [2]float32{100, 100}), flagTestLife(1, [2]float32{100, 100.003}),
	}
}

var flagFilmBasesTeams = map[string]int{"1": 0, "2": 1}

// flagHasHome dit que la vie du drapeau publie au moins un etat `home`.
func flagHasHome(c FlagCarry) bool {
	for _, s := range c.Spans {
		if s.State == FlagStateHome {
			return true
		}
	}
	return false
}

// TestBasesDuFilmHorsCatalogue — carte hors catalogue : deux drapeaux d'equipe, chacun a la base
// ou l'AUTRE camp vole, poses au point de la renaissance de l'objet, avec leurs etats `home`.
func TestBasesDuFilmHorsCatalogue(t *testing.T) {
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, flagFilmBasesRebirths()), ctx)
	if cov.FilmBases != 2 || cov.SpawnsFromFilm != 2 || cov.Spawns != 2 || cov.CarrierTeamUnknown != 0 {
		t.Fatalf("couverture %+v : deux bases lues, deux socles du film, aucun porteur sans drapeau attendus", cov)
	}
	if len(got) != 2 {
		t.Fatalf("%d drapeaux publies, attendu 2", len(got))
	}
	for _, c := range got {
		if !flagHasHome(c) {
			t.Errorf("drapeau d'equipe %d sans etat home", c.Team)
		}
	}
	if got[0].Team != 0 || got[0].Spans[0].X != 100 || got[1].Team != 1 || got[1].Spans[0].X >= 0.01 {
		t.Errorf("drapeaux %+v / %+v : l'equipe 0 a sa base en (100,100), l'equipe 1 en (0,0)", got[0], got[1])
	}
	for _, nom := range []fallback.Nom{fallback.NomIndexDrapeauZeroPourTous,
		fallback.NomNombreDrapeauxHorsCatalogueSansPassage, fallback.NomSocleDuFilmAuCentreDesVols} {
		if n := ctx.fb.Compte(nom); n != 0 {
			t.Errorf("repli %s declenche %d fois, attendu 0", nom, n)
		}
	}
}

// TestBasesDuFilmSansRenaissanceAuCentreDesVols — sans renaissance de l'objet, la base est le
// centre des vols, et le repli nomme se compte une fois par base.
func TestBasesDuFilmSansRenaissanceAuCentreDesVols(t *testing.T) {
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, nil), ctx)
	if cov.Spawns != 2 || len(got) != 2 {
		t.Fatalf("couverture %+v : deux socles attendus", cov)
	}
	if n := ctx.fb.Compte(fallback.NomSocleDuFilmAuCentreDesVols); n != 2 {
		t.Errorf("repli du centre des vols declenche %d fois, attendu 2", n)
	}
	if got[0].Spans[0].X != 98 || got[1].Spans[0].X != 2 {
		t.Errorf("bases (%v, %v) : attendu le centre des vols (98 puis 2)", got[0].Spans[0].X, got[1].Spans[0].X)
	}
}

// TestBasesDuFilmVariantesNeutre — les deux camps volent au meme point : UN drapeau neutre.
func TestBasesDuFilmVariantesNeutre(t *testing.T) {
	ctx := flagFilmBasesCtx([2]float32{50, 50}, [2]float32{51, 50})
	got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, nil), ctx)
	if !cov.NeutralFlag || cov.FilmBases != 1 || cov.Spawns != 1 || len(got) != 1 || got[0].Team != TeamNeutral {
		t.Fatalf("couverture %+v, drapeaux %+v : un drapeau neutre attendu", cov, got)
	}
}

// TestBasesDuFilmSeuilsDeSeparation — la distance entre les points de vol des deux camps
// tranche : jusqu'au rayon de groupe (3 m) une base neutre, a partir du double (6 m) deux bases
// d'equipe — les socles d'Isolation sont a 8,46 m, ses centres de vols a 9,2 m —, entre les
// deux rien n'est lu et le repli de l'index zero tient.
func TestBasesDuFilmSeuilsDeSeparation(t *testing.T) {
	for _, cas := range []struct {
		nom    string
		dx     float32
		bases  int
		repli0 int
	}{
		{"neutre a 2,9 m", 2.9, 1, 0},
		{"bande morte a 3,2 m", 3.2, 0, 2},
		{"bande morte a 5,8 m", 5.8, 0, 2},
		{"deux bases a 6,2 m", 6.2, 2, 0},
		{"deux bases a 9 m (Isolation)", 9, 2, 0},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			ctx := flagFilmBasesCtx([2]float32{0, 0}, [2]float32{cas.dx, 0})
			got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, nil), ctx)
			if cov.FilmBases != cas.bases || cov.Spawns != cas.bases || len(got) != max(cas.bases, 1) {
				t.Fatalf("couverture %+v, %d drapeaux : %d base(s) attendue(s)", cov, len(got), cas.bases)
			}
			if n := ctx.fb.Compte(fallback.NomIndexDrapeauZeroPourTous); n != cas.repli0 {
				t.Errorf("repli de l'index zero declenche %d fois, attendu %d", n, cas.repli0)
			}
		})
	}
}

// TestBasesDuFilmSurLeCatalogue — le catalogue reste la source quand il nomme le camp ; le film
// choisit les socles en jeu, nomme ceux qui n'ont pas de camp, et compte accord et contradiction.
func TestBasesDuFilmSurLeCatalogue(t *testing.T) {
	for _, cas := range []struct {
		nom                         string
		spawns                      []FlagSpawn
		accord, contra, film, teamA int
	}{
		{"deux socles nommes", flagInvariantSpawns(), 2, 0, 0, 1},
		{"socle central sans camp, hors jeu", []FlagSpawn{{Team: 1, X: 0, Y: 0},
			{Team: TeamNeutral, X: 50, Y: 50}, {Team: 0, X: 100, Y: 100}}, 2, 0, 0, 1},
		{"socles sans camp", []FlagSpawn{{Team: TeamNeutral, X: 0, Y: 0},
			{Team: TeamNeutral, X: 100, Y: 100}}, 0, 0, 2, 1},
		{"catalogue contredit", []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}}, 0, 2, 0, 0},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
			got, cov := buildFlagCarries(flagFilmBasesScan(cas.spawns, flagFilmBasesTeams, nil), ctx)
			if cov.FilmBaseAgree != cas.accord || cov.FilmBaseContradict != cas.contra ||
				cov.SpawnsFromFilm != cas.film || cov.Spawns != 2 || len(got) != 2 {
				t.Fatalf("couverture %+v : accord %d, contradiction %d, du film %d, deux socles attendus",
					cov, cas.accord, cas.contra, cas.film)
			}
			// Le premier drapeau publie est celui du socle (0,0), dans l'ordre du catalogue.
			if got[0].Team != cas.teamA || got[0].Spans[0].X != 0 {
				t.Errorf("premier drapeau %+v : equipe %d au socle (0,0) attendue", got[0], cas.teamA)
			}
			if n := ctx.fb.Compte(fallback.NomSocleDuFilmAuCentreDesVols); n != 0 {
				t.Errorf("repli du centre des vols declenche %d fois sur des socles du catalogue", n)
			}
		})
	}
}

// TestBasesDuFilmSoclesSuperposes — une carte declare deux socles de camps opposes au meme point,
// a quelques millimetres (cas `ee43d273`), et le socle contradictoire vient en premier et au plus
// pres : la base retient le socle dont le camp CONCORDE avec le film, puis un socle sans camp,
// et seulement a defaut un socle contradictoire.
func TestBasesDuFilmSoclesSuperposes(t *testing.T) {
	for _, cas := range []struct {
		nom                  string
		spawns               []FlagSpawn
		accord, contra, film int
	}{
		{"camp concordant plutot que le plus proche", []FlagSpawn{
			{Team: 1, X: 100, Y: 100}, {Team: 0, X: 100.003, Y: 100.003},
			{Team: 0, X: 0, Y: 0}, {Team: 1, X: 0.003, Y: 0.003}}, 2, 0, 0},
		{"sans camp plutot que contradictoire", []FlagSpawn{
			{Team: 1, X: 100, Y: 100}, {Team: TeamNeutral, X: 100.003, Y: 100.003},
			{Team: 0, X: 0, Y: 0}, {Team: TeamNeutral, X: 0.003, Y: 0.003}}, 0, 0, 2},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
			got, cov := buildFlagCarries(flagFilmBasesScan(cas.spawns, flagFilmBasesTeams, flagFilmBasesRebirths()), ctx)
			if cov.FilmBaseAgree != cas.accord || cov.FilmBaseContradict != cas.contra ||
				cov.SpawnsFromFilm != cas.film || cov.Spawns != 2 || len(got) != 2 {
				t.Fatalf("couverture %+v : accord %d, contradiction %d, du film %d, deux socles attendus",
					cov, cas.accord, cas.contra, cas.film)
			}
			// Ordre du catalogue : le socle d'index 1 (l'equipe 0, en (100,100)) puis celui d'index 3.
			if got[0].Team != 0 || got[0].Spans[0].X < 100 || got[1].Team != 1 || got[1].Spans[0].X > 0.01 {
				t.Errorf("drapeaux %+v / %+v : equipe 0 au socle (100,100), equipe 1 au socle (0,0)", got[0], got[1])
			}
		})
	}
}

// flagFilmBasesNeutralBirths : trois naissances de l'objet au point (x, y) — assez pour que
// l'objet tranche « drapeau neutre » quand un socle neutre du catalogue s'y trouve.
func flagFilmBasesNeutralBirths(x, y float32) []flagFreeLife {
	return []flagFreeLife{flagTestLife(0, [2]float32{x, y}), flagTestLife(10, [2]float32{x, y}),
		flagTestLife(20, [2]float32{x, y})}
}

// TestBasesDuFilmBaseNeutreSurLeCatalogue — les deux camps volent au meme point (50,50). Le
// catalogue reste source quand il NOMME le camp ou quand l'objet a tranche ; un socle qui ne nomme
// aucun camp devient le socle neutre quand l'objet n'a pas tranche (cas Cliffside `4bffd021` :
// socle central sans camp ni label).
func TestBasesDuFilmBaseNeutreSurLeCatalogue(t *testing.T) {
	equipes := []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 1, X: 100, Y: 100}}
	avec := func(s ...FlagSpawn) []FlagSpawn { return append(s, equipes...) }
	for _, cas := range []struct {
		nom                  string
		spawns               []FlagSpawn
		free                 []flagFreeLife
		accord, contra, film int
		socles               int
		// x : abscisse du drapeau neutre publie ; 0 = aucun drapeau neutre attendu.
		x float32
	}{
		{"accord : l'objet a tranche neutre au meme point",
			avec(FlagSpawn{Team: TeamNeutral, Neutral: true, X: 50, Y: 50}), flagFilmBasesNeutralBirths(50, 50),
			1, 0, 0, 1, 50},
		{"contradiction : l'objet a tranche neutre ailleurs, le socle sans camp a portee ne l'emporte pas",
			avec(FlagSpawn{Team: TeamNeutral, Neutral: true, X: 80, Y: 80}, FlagSpawn{Team: TeamNeutral, X: 50, Y: 50}),
			flagFilmBasesNeutralBirths(80, 80), 0, 1, 0, 1, 80},
		{"contradiction : le socle a portee nomme un camp",
			avec(FlagSpawn{Team: 0, X: 50, Y: 50}), nil, 0, 1, 0, 3, 0},
		{"socle sans camp promu neutre",
			avec(FlagSpawn{Team: TeamNeutral, X: 50, Y: 50}), nil, 0, 0, 1, 1, 50},
		{"socle neutre par son label, objet muet",
			avec(FlagSpawn{Team: 0, Neutral: true, X: 50, Y: 50}), nil, 1, 0, 0, 1, 50},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			ctx := flagFilmBasesCtx([2]float32{50, 50}, [2]float32{50.5, 50})
			got, cov := buildFlagCarries(flagFilmBasesScan(cas.spawns, flagFilmBasesTeams, cas.free), ctx)
			neutre := cas.x != 0
			if cov.FilmBases != 1 || cov.FilmBaseAgree != cas.accord || cov.FilmBaseContradict != cas.contra ||
				cov.SpawnsFromFilm != cas.film || cov.NeutralFlag != neutre || cov.Spawns != cas.socles {
				t.Fatalf("couverture %+v : accord %d, contradiction %d, du film %d, neutre %v, %d socle(s) attendus",
					cov, cas.accord, cas.contra, cas.film, neutre, cas.socles)
			}
			if neutre && (len(got) != 1 || got[0].Team != TeamNeutral || got[0].Spans[0].X != cas.x) {
				t.Errorf("drapeaux %+v : un seul drapeau neutre, au socle du catalogue d'abscisse %v", got, cas.x)
			}
		})
	}
}

// TestBasesDuFilmDeuxBasesContreLeVerdictNeutre — le film lit deux bases alors que l'objet a
// tranche « drapeau neutre » : la contradiction se compte, le verdict de l'objet reste.
func TestBasesDuFilmDeuxBasesContreLeVerdictNeutre(t *testing.T) {
	spawns := append(flagInvariantSpawns(), FlagSpawn{Team: TeamNeutral, Neutral: true, X: 50, Y: 50})
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	got, cov := buildFlagCarries(flagFilmBasesScan(spawns, flagFilmBasesTeams, flagFilmBasesNeutralBirths(50, 50)), ctx)
	if cov.FilmBases != 2 || cov.FilmBaseContradict != 1 || cov.FilmBaseAgree != 0 ||
		!cov.NeutralFlag || cov.Spawns != 1 || len(got) != 1 {
		t.Fatalf("couverture %+v : contradiction comptee, verdict neutre et socle unique gardes", cov)
	}
}

// TestBasesDuFilmDeuxSoclesDuMemeCamp — les deux socles appries portent le MEME camp : le
// catalogue est contredit sur l'un d'eux, son choix (trois socles) reste, et la contradiction se
// compte.
func TestBasesDuFilmDeuxSoclesDuMemeCamp(t *testing.T) {
	spawns := []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: 0, X: 100, Y: 100}, {Team: 1, X: 50, Y: 50}}
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	_, cov := buildFlagCarries(flagFilmBasesScan(spawns, flagFilmBasesTeams, nil), ctx)
	if cov.FilmBaseAgree != 1 || cov.FilmBaseContradict != 1 || cov.SpawnsFromFilm != 0 || cov.Spawns != 3 {
		t.Fatalf("couverture %+v : accord 1, contradiction 1, les trois socles du catalogue attendus", cov)
	}
}

// TestBasesDuFilmRenaissanceAvecCatalogue — le catalogue ne declare que le socle de l'equipe 1 :
// la base de l'equipe 0, sans socle a portee, est posee au point de la RENAISSANCE de l'objet
// (100, 100), pas au centre des vols (98, 98), et le repli du centre des vols ne se compte pas.
func TestBasesDuFilmRenaissanceAvecCatalogue(t *testing.T) {
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	got, cov := buildFlagCarries(flagFilmBasesScan([]FlagSpawn{{Team: 1, X: 0, Y: 0}}, flagFilmBasesTeams,
		flagFilmBasesRebirths()), ctx)
	if cov.FilmBaseAgree != 1 || cov.SpawnsFromFilm != 1 || cov.Spawns != 2 || len(got) != 2 {
		t.Fatalf("couverture %+v : accord 1, un socle du film, deux socles attendus", cov)
	}
	if got[1].Team != 0 || got[1].Spans[0].X != 100 {
		t.Errorf("drapeau %+v : equipe 0 a la renaissance (100,100) attendue", got[1])
	}
	if n := ctx.fb.Compte(fallback.NomSocleDuFilmAuCentreDesVols); n != 0 {
		t.Errorf("repli du centre des vols declenche %d fois, attendu 0", n)
	}
}

// TestBasesDuFilmPartDuGroupePrincipal — un camp dont les vols se partagent entre deux points ne
// donne aucune base : le groupe principal doit tenir les trois quarts des vols.
func TestBasesDuFilmPartDuGroupePrincipal(t *testing.T) {
	partages := []flagPoint{{0, 0}, {0.5, 0}, {40, 40}, {40.5, 40}}
	if _, ok := flagStealCentre(partages); ok {
		t.Errorf("vols partages moitie-moitie : aucune base attendue")
	}
	groupes := []flagPoint{{0, 0}, {0.6, 0}, {0, 0.3}, {40, 40}}
	c, ok := flagStealCentre(groupes)
	if !ok || math.Abs(float64(c.x)-0.2) > 1e-6 || math.Abs(float64(c.y)-0.1) > 1e-6 {
		t.Errorf("trois vols sur quatre au meme point : centre (0,2 ; 0,1) attendu, rendu %+v (%v)", c, ok)
	}
}

// TestBasesDuFilmNaissanceIsoleeNEstPasUneRenaissance — carte hors catalogue : une naissance
// ISOLEE de l'objet (un lacher) a moins de 3 m du centre des vols ne fait pas une renaissance
// ([flagFilmRebirthMin] = 2). La base de l'equipe 0 reste au centre des vols (98, 98) et le repli
// se compte ; celle de l'equipe 1, qui a sa renaissance repetee en (0, 0), s'y pose.
func TestBasesDuFilmNaissanceIsoleeNEstPasUneRenaissance(t *testing.T) {
	free := []flagFreeLife{
		flagTestLife(0, [2]float32{0, 0}), flagTestLife(1, [2]float32{0.004, 0}),
		flagTestLife(40, [2]float32{99, 99}),
	}
	ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
	got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, free), ctx)
	if cov.FilmBases != 2 || cov.Spawns != 2 || len(got) != 2 {
		t.Fatalf("couverture %+v : deux bases attendues", cov)
	}
	if got[0].Team != 0 || got[0].Spans[0].X != 98 {
		t.Errorf("drapeau %+v : equipe 0 au centre des vols (98, 98) attendue, pas au lacher (99, 99)", got[0])
	}
	if got[1].Team != 1 || got[1].Spans[0].X > 0.01 {
		t.Errorf("drapeau %+v : equipe 1 a sa renaissance (0, 0) attendue", got[1])
	}
	if n := ctx.fb.Compte(fallback.NomSocleDuFilmAuCentreDesVols); n != 1 {
		t.Errorf("repli du centre des vols declenche %d fois, attendu 1 (la base sans renaissance)", n)
	}
}

// TestBasesDuFilmNeutreSansCatalogue — carte hors catalogue, les deux camps volent au meme point :
// sans renaissance la base neutre est le centre des vols et le repli se compte ; avec une
// renaissance repetee a portee, la base s'y pose, sans repli. Les naissances au socle neutre
// retenu se comptent dans la couverture.
func TestBasesDuFilmNeutreSansCatalogue(t *testing.T) {
	for _, cas := range []struct {
		nom              string
		free             []flagFreeLife
		x                float32
		repli, naissance int
	}{
		{"sans renaissance : centre des vols", nil, 50.25, 1, 0},
		{"renaissance repetee", []flagFreeLife{flagTestLife(0, [2]float32{51.5, 50}),
			flagTestLife(10, [2]float32{51.5, 50.004})}, 51.5, 0, 2},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			ctx := flagFilmBasesCtx([2]float32{50, 50}, [2]float32{50.5, 50})
			got, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, cas.free), ctx)
			if !cov.NeutralFlag || cov.Spawns != 1 || len(got) != 1 || got[0].Team != TeamNeutral {
				t.Fatalf("couverture %+v : un drapeau neutre attendu", cov)
			}
			if math.Abs(float64(got[0].Spans[0].X-cas.x)) > 0.01 {
				t.Errorf("socle neutre en x=%v, attendu %v", got[0].Spans[0].X, cas.x)
			}
			if n := ctx.fb.Compte(fallback.NomSocleDuFilmAuCentreDesVols); n != cas.repli {
				t.Errorf("repli du centre des vols declenche %d fois, attendu %d", n, cas.repli)
			}
			if cov.NeutralBirths != cas.naissance {
				t.Errorf("naissances au socle neutre %d, attendu %d", cov.NeutralBirths, cas.naissance)
			}
		})
	}
}

// TestBasesDuFilmRecomptentLesNaissances — quand le film change les socles retenus, les comptes de
// naissances qui fondent la variante sont recalcules SUR CES SOCLES : ceux du catalogue (vides
// hors catalogue) ne disent rien des socles lus dans le film.
func TestBasesDuFilmRecomptentLesNaissances(t *testing.T) {
	t.Run("deux bases d'equipe hors catalogue", func(t *testing.T) {
		ctx := flagFilmBasesCtx([2]float32{2, 2}, [2]float32{98, 98})
		_, cov := buildFlagCarries(flagFilmBasesScan(nil, flagFilmBasesTeams, flagFilmBasesRebirths()), ctx)
		if cov.TeamBirths != 4 || cov.NeutralBirths != 0 {
			t.Errorf("naissances equipe %d / neutre %d, attendu 4 / 0", cov.TeamBirths, cov.NeutralBirths)
		}
	})
	t.Run("socle sans camp promu neutre", func(t *testing.T) {
		spawns := []FlagSpawn{{Team: 0, X: 0, Y: 0}, {Team: TeamNeutral, X: 50, Y: 50}, {Team: 1, X: 100, Y: 100}}
		free := append(flagFilmBasesNeutralBirths(50, 50)[:2], flagTestLife(30, [2]float32{0, 0}))
		ctx := flagFilmBasesCtx([2]float32{50, 50}, [2]float32{50.5, 50})
		_, cov := buildFlagCarries(flagFilmBasesScan(spawns, flagFilmBasesTeams, free), ctx)
		if !cov.NeutralFlag || cov.SpawnsFromFilm != 1 {
			t.Fatalf("couverture %+v : socle sans camp promu neutre attendu", cov)
		}
		if cov.NeutralBirths != 2 || cov.TeamBirths != 1 {
			t.Errorf("naissances neutre %d / equipe %d, attendu 2 / 1", cov.NeutralBirths, cov.TeamBirths)
		}
	})
}
