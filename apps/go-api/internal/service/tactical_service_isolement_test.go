package service

// tactical_service_isolement_test.go — LA LECTURE « OU JE MEURS ISOLE », de bout en bout.
//
// Ce que ces tests couvrent et que les tests purs de `coordination` ne peuvent pas : la JOINTURE
// DES EQUIPES (le film ne porte aucun camp), la RESOLUTION DU RAYON par la variante du match, le
// fait que les matchs sans rayon sortent de l'UNIVERS et pas seulement du numerateur, et les
// portees de radar publiees pour l'aide du plan.
//
// LA LECTURE SE LIT SUR SES CELLULES : une cellule porte les morts isolees qui y sont tombees
// (`Brut`) et leur nombre par match de l'univers (`Valeur`). Le plancher de 3 matchs distincts par
// cellule s'applique ; les corpus posent donc chaque mort a mesurer dans trois matchs.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

// mortContexte pose une mort localisee avec son voisinage, tel que le collecteur l'a mesure.
func mortContexte(matchID, victime string, x, y float64, proche *float64,
	visibles, horsDeVue int,
) domain.MortContexte {
	return domain.MortContexte{
		MatchID: matchID, VictimXUID: victime, X: x, Y: y,
		PlusProcheM: proche, Visibles: visibles, HorsDeVue: horsDeVue,
	}
}

func m(v float64) *float64 { return &v }

// svcIsole monte le service avec la table des rayons d'Arene et de BTB.
func svcIsole(univ domain.TacticalUnivers, morts ...domain.MortContexte) (*TacticalService, *mockTacticalRepo) {
	repo := &mockTacticalRepo{
		univ:  univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: morts},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).
		WithRadarRange(map[string]int{"Slayer:Arena": 18, "BTB:Slayer": 24})
	return svc, repo
}

func lireIsole(t *testing.T, svc *TacticalService, ids ...string) domain.TacticalRaster {
	t.Helper()
	return lireQuestion(t, svc, domain.TacticalQuestionIsole, ids...)
}

func lireQuestion(t *testing.T, svc *TacticalService, question string, ids ...string) domain.TacticalRaster {
	t.Helper()
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: question, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids},
	})
	if err != nil {
		t.Fatalf("lecture %s: %v", question, err)
	}
	return out
}

// svcDeuxSubstrats monte le service avec les memes morts servies aux DEUX substrats : le
// journal des positions (lectures morts/kills/gagne) et le contexte des morts (isole).
func svcDeuxSubstrats(univ domain.TacticalUnivers, morts ...domain.MortContexte) *TacticalService {
	svc, repo := svcIsole(univ, morts...)
	repo.pos.Univers = univ
	for _, mc := range morts {
		repo.pos.Points = append(repo.pos.Points, domain.TacticalKillPosition{
			MatchID: mc.MatchID, KillerXUID: tsAdv, VictimXUID: mc.VictimXUID,
			KillerX: 9, KillerY: 9, VictimX: mc.X, VictimY: mc.Y, TimeMs: 1000,
		})
	}
	return svc
}

// mortsDansTroisMatchs pose la MEME mort (meme victime, meme lieu, meme voisinage) dans chacun
// des matchs donnes : c'est ce qui fait franchir le plancher de 3 matchs distincts a sa cellule.
func mortsDansTroisMatchs(ids []string, victime string, x, y float64, proche *float64,
	visibles int) []domain.MortContexte {
	out := make([]domain.MortContexte, 0, len(ids))
	for _, id := range ids {
		out = append(out, mortContexte(id, victime, x, y, proche, visibles, 0))
	}
	return out
}

// variantes rend la table match -> variante pour une liste de matchs d'une meme variante.
func variantes(table map[string]string, variante string, ids ...string) map[string]string {
	for _, id := range ids {
		table[id] = variante
	}
	return table
}

// TestIsole_RayonDeLaVarianteDuMatch — LA MEME MORT, a 19 m d'un coequipier, est ISOLEE en
// Arene (18 m) et NE L'EST PAS en BTB (24 m).
//
// Le rayon vient de la VARIANTE du match, resolue par `regulation.toml`. Un rayon unique
// melangerait deux regles de jeu sous une seule mesure — et un filtre qui contient les deux
// formats est le cas normal.
func TestIsole_RayonDeLaVarianteDuMatch(t *testing.T) {
	arene, btb := []string{"a1", "a2", "a3"}, []string{"b1", "b2", "b3"}
	table := variantes(variantes(map[string]string{}, "Slayer:Arena", arene...), "BTB:Slayer", btb...)
	morts := append(mortsDansTroisMatchs(arene, tsMoi, 2, 3, m(19), 1),
		mortsDansTroisMatchs(btb, tsMoi, 2, 3, m(19), 1)...)
	svc, _ := svcIsole(universVariantes(table), morts...)

	out := lireIsole(t, svc, append(arene, btb...)...)
	c := celluleEn(out.Cellules, 2, 3)
	if c == nil {
		t.Fatalf("cellules = %+v : les trois morts d'Arene a 19 m sont isolees", out.Cellules)
	}
	if c.Brut != 3 || c.Matchs != 3 {
		t.Fatalf("cellule = %+v, attendu 3 morts isolees sur 3 matchs : 19 m depasse les 18 m de "+
			"l'Arene mais pas les 24 m du BTB", *c)
	}
	if c.Valeur != 0.5 {
		t.Fatalf("valeur = %v, attendu 0,5 (3 isolees / 6 matchs ayant un rayon)", c.Valeur)
	}
	if out.MatchsSansRayon != 0 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 0", out.MatchsSansRayon)
	}
}

// TestIsole_VarianteAvecBlancs_ResoutQuandMeme — LE NOM VIENT DE LA BASE, et la base porte ce
// que l'API a envoye : des variantes y arrivent avec un blanc de tete ou de queue.
//
// Sans nettoyage, la cle manque la table et le match sort SILENCIEUSEMENT de l'univers : un
// defaut de donnee deguise en trou de referentiel, qui envoie chercher la panne au mauvais
// endroit.
func TestIsole_VarianteAvecBlancs_ResoutQuandMeme(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{"m1": "  Slayer:Arena "}),
		mortContexte("m1", tsMoi, 2, 3, m(19), 1, 0))

	out := lireIsole(t, svc, "m1")
	if out.MatchsSansRayon != 0 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 0 : « %s » est l'Arene, blancs compris",
			out.MatchsSansRayon, "  Slayer:Arena ")
	}
	if out.MatchsRetenus != 1 {
		t.Fatalf("matchs_retenus = %d, attendu 1 : le match resolu entre dans l'univers de la lecture",
			out.MatchsRetenus)
	}
}

// TestIsole_LesMortsDesAutresNEntrentPas — l'axe « qui » se tranche sur les EQUIPES.
//
// Le collecteur mesure le contexte de CHAQUE mort du match, sans savoir laquelle interesse la
// page. C'est la lecture qui joint les equipes et ne garde que la cible — sinon « ou JE meurs
// isole » peindrait aussi les morts des adversaires.
func TestIsole_LesMortsDesAutresNEntrentPas(t *testing.T) {
	ids := []string{"m1", "m2", "m3"}
	morts := append(mortsDansTroisMatchs(ids, tsMoi, 2, 3, m(40), 1),
		mortsDansTroisMatchs(ids, tsAdv, 9, 9, m(40), 1)...)
	svc, _ := svcIsole(universVariantes(variantes(map[string]string{}, "Slayer:Arena", ids...)), morts...)

	out := lireIsole(t, svc, ids...)
	if c := celluleEn(out.Cellules, 2, 3); c == nil || c.Brut != 3 {
		t.Fatalf("cellules = %+v, attendu mes 3 morts isolees en (2,3)", out.Cellules)
	}
	if c := celluleEn(out.Cellules, 9, 9); c != nil {
		t.Fatalf("cellule (9,9) = %+v : seules MES morts entrent sous l'axe « moi »", *c)
	}
}

// TestIsole_EquipeATerre_ExclueEtPubliee — personne ne pouvait accompagner : la mort est
// ECARTEE de la lecture (aucune cellule ne la porte) et comptee a part.
func TestIsole_EquipeATerre_ExclueEtPubliee(t *testing.T) {
	ids := []string{"m1", "m2", "m3"}
	morts := append(mortsDansTroisMatchs(ids, tsMoi, 2, 3, nil, 0),
		mortsDansTroisMatchs(ids, tsMoi, 4, 5, m(40), 1)...)
	svc, _ := svcIsole(universVariantes(variantes(map[string]string{}, "Slayer:Arena", ids...)), morts...)

	out := lireIsole(t, svc, ids...)
	if out.MortsEquipeATerre != 3 {
		t.Fatalf("morts_equipe_a_terre = %d, attendu 3", out.MortsEquipeATerre)
	}
	if c := celluleEn(out.Cellules, 2, 3); c != nil {
		t.Fatalf("cellule (2,3) = %+v : une mort sans personne pour accompagner n'est pas isolee", *c)
	}
	if c := celluleEn(out.Cellules, 4, 5); c == nil || c.Brut != 3 {
		t.Fatalf("cellules = %+v, attendu les 3 morts isolees en (4,5)", out.Cellules)
	}
}

// TestIsole_VarianteSansRayon — LE MATCH SORT DE L'UNIVERS, PAS SEULEMENT DU NUMERATEUR.
//
// Le laisser au denominateur diviserait la mesure par des matchs qu'on a refuse de lire : trois
// matchs connus et un Husky Raid rendraient 0,75 mort isolee par match au lieu de 1.
func TestIsole_VarianteSansRayon(t *testing.T) {
	connus := []string{"c1", "c2", "c3"}
	table := variantes(map[string]string{"inconnu": "Husky Raid:CTF"}, "Slayer:Arena", connus...)
	svc, _ := svcIsole(universVariantes(table), mortsDansTroisMatchs(connus, tsMoi, 2, 3, m(40), 1)...)

	out := lireIsole(t, svc, append(connus, "inconnu")...)
	if out.MatchsSansRayon != 1 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 1", out.MatchsSansRayon)
	}
	if out.MatchsFiltres != 4 {
		t.Fatalf("matchs_filtres = %d, attendu 4 : les quatre matchs restent dans l'univers du filtre",
			out.MatchsFiltres)
	}
	if out.MatchsRetenus != 3 {
		t.Fatalf("matchs_retenus = %d, attendu 3 : l'univers de la lecture est « mesure ET ayant "+
			"un rayon »", out.MatchsRetenus)
	}
	c := celluleEn(out.Cellules, 2, 3)
	if c == nil || c.Valeur != 1 {
		t.Fatalf("cellules = %+v, attendu une valeur de 1 en (2,3) (3 isolees / 3 matchs ayant un "+
			"rayon) — diviser par 4 ferait varier la mesure avec les matchs qu'on refuse de lire", out.Cellules)
	}
}

// TestIsole_SansTableDeRayon — un titre dont `regulation.toml` ne declare aucune portee ne rend
// AUCUNE lecture d'isolement, et le dit.
func TestIsole_SansTableDeRayon(t *testing.T) {
	univ := universVariantes(map[string]string{"m1": "Slayer:Arena"})
	repo := &mockTacticalRepo{
		univ: univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{
			mortContexte("m1", tsMoi, 2, 3, m(40), 1, 0),
		}},
	}
	// AUCUN `WithRadarRange` : c'est le cas teste.
	svc := NewTacticalService(repo, capsCompletes(), tsMoi)
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionIsole, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: []string{"m1"}},
	})
	if err != nil {
		t.Fatalf("lecture: %v", err)
	}
	if out.MatchsSansRayon != 1 || out.MatchsRetenus != 0 || len(out.Cellules) != 0 {
		t.Fatalf("sortie = %+v : sans table, aucune mort ne doit etre examinee", out)
	}
}

// TestIsole_NeVentilePasLesArtefacts — LA VENTILATION NE CONCERNE PAS CETTE LECTURE.
//
// « Isole » lit la BASE : elle n'attend aucun artefact de rejeu, donc rien n'y est « en
// attente » ni « non cuisable ». Remplir ces compteurs annoncerait un traitement en cours a qui
// a deja toute sa reponse — et l'invariant de somme ne vaut que pour les lectures d'artefact.
func TestIsole_NeVentilePasLesArtefacts(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{"m1": "Slayer:Arena"}),
		mortContexte("m1", tsMoi, 2, 3, m(40), 1, 0))

	out := lireIsole(t, svc, "m1")
	if out.MatchsEnAttente != 0 || out.MatchsNonCuisables != 0 {
		t.Fatalf("en_attente=%d non_cuisables=%d, attendu 0 et 0 : cette lecture n'attend "+
			"aucun artefact", out.MatchsEnAttente, out.MatchsNonCuisables)
	}
}

// TestIsole_LeFiltreDeSpawnSAppliqueAussi — le filtre de grappe est un filtre d'UNIVERS : il
// vaut pour cette lecture comme pour les autres.
func TestIsole_HonoreLaListeBlanche(t *testing.T) {
	svc, repo := svcIsole(universVariantes(map[string]string{
		"m1": "Slayer:Arena", "m2": "Slayer:Arena",
	}),
		mortContexte("m1", tsMoi, 2, 3, m(40), 1, 0),
		mortContexte("m2", tsMoi, 2, 3, m(40), 1, 0))

	out := lireIsole(t, svc, "m1")
	if !repo.vuMorts.Matchs.Restreint() {
		t.Fatal("liste blanche non posee : le lecteur servirait tout l'historique")
	}
	if out.MatchsFiltres != 1 {
		t.Fatalf("matchs_filtres = %d, attendu 1 : le perimetre ne retient que m1", out.MatchsFiltres)
	}
}

// TestIsole_RayonsRadarDistinctsEtTries : la lecture « isole » publie les portées de radar de ses
// matchs, DISTINCTES et triées (deux Arène + un BTB → [18 24]), jamais leur moyenne ; les autres
// lectures n'en publient aucune.
func TestIsole_RayonsRadarDistinctsEtTries(t *testing.T) {
	svc := svcDeuxSubstrats(universVariantes(map[string]string{
		"arene1": "Slayer:Arena", "arene2": "Slayer:Arena", "btb": "BTB:Slayer",
	}),
		mortContexte("arene1", tsMoi, 2, 3, m(30), 1, 0),
		mortContexte("arene2", tsMoi, 2, 3, m(30), 1, 0),
		mortContexte("btb", tsMoi, 2, 3, m(30), 1, 0))

	out := lireQuestion(t, svc, domain.TacticalQuestionIsole, "arene1", "arene2", "btb")
	if got := out.RayonsRadarM; len(got) != 2 || got[0] != 18 || got[1] != 24 {
		t.Fatalf("rayons_radar_m = %v, attendu [18 24] (distincts, triés)", got)
	}
	if autre := lireQuestion(t, svc, domain.TacticalQuestionMorts, "arene1", "arene2", "btb"); len(autre.RayonsRadarM) != 0 {
		t.Errorf("rayons_radar_m publiés hors de « isole » : %v", autre.RayonsRadarM)
	}
}
