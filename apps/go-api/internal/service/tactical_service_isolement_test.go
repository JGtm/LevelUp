package service

// tactical_service_isolement_test.go — LA LECTURE « OU JE MEURS ISOLE », de bout en bout.
//
// Ce que ces tests couvrent et que les tests purs de `coordination` ne peuvent pas : la JOINTURE
// DES EQUIPES (le film ne porte aucun camp), la RESOLUTION DU RAYON par la variante du match, et
// le fait que les matchs sans rayon sortent de l'UNIVERS et pas seulement du numerateur.

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
	out, err := svc.Raster(context.Background(), domain.TacticalRasterRequest{
		MapID: "streets", Question: domain.TacticalQuestionIsole, Qui: domain.TacticalQuiMoi,
		Scope: domain.TacticalScope{MatchIDs: ids},
	})
	if err != nil {
		t.Fatalf("lecture isole: %v", err)
	}
	return out
}

// TestIsole_RayonDeLaVarianteDuMatch — LA MEME MORT, a 19 m d'un coequipier, est ISOLEE en
// Arene (18 m) et NE L'EST PAS en BTB (24 m).
//
// Le rayon vient de la VARIANTE du match, resolue par `regulation.toml`. Un rayon unique
// melangerait deux regles de jeu sous une seule mesure — et un filtre qui contient les deux
// formats est le cas normal.
func TestIsole_RayonDeLaVarianteDuMatch(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{
		"arene": "Slayer:Arena", "btb": "BTB:Slayer",
	}),
		mortContexte("arene", tsMoi, 2, 3, m(19), 1, 0),
		mortContexte("btb", tsMoi, 2, 3, m(19), 1, 0))

	out := lireIsole(t, svc, "arene", "btb")
	if out.Isolement == nil {
		t.Fatal("la lecture isole ne publie aucune couverture")
	}
	if out.Isolement.N != 2 {
		t.Fatalf("denominateur = %d, attendu 2 morts examinees", out.Isolement.N)
	}
	if out.Isolement.Brut != 1 {
		t.Fatalf("morts isolees = %d, attendu 1 : 19 m depasse les 18 m de l'Arene mais pas "+
			"les 24 m du BTB", out.Isolement.Brut)
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
	if out.Isolement.N != 1 || out.Isolement.Brut != 1 {
		t.Fatalf("couverture = %+v, attendu 1 isolee sur 1", out.Isolement)
	}
}

// TestIsole_LesMortsDesAutresNEntrentPas — l'axe « qui » se tranche sur les EQUIPES.
//
// Le collecteur mesure le contexte de CHAQUE mort du match, sans savoir laquelle interesse la
// page. C'est la lecture qui joint les equipes et ne garde que la cible — sinon « ou JE meurs
// isole » compterait aussi les morts des adversaires.
func TestIsole_LesMortsDesAutresNEntrentPas(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{"m1": "Slayer:Arena"}),
		mortContexte("m1", tsMoi, 2, 3, m(40), 1, 0),
		mortContexte("m1", tsAdv, 9, 9, m(40), 1, 0),
		mortContexte("m1", tsAdv2, 8, 8, m(40), 1, 0))

	out := lireIsole(t, svc, "m1")
	if out.Isolement.N != 1 {
		t.Fatalf("denominateur = %d, attendu 1 : seules MES morts entrent sous l'axe « moi »",
			out.Isolement.N)
	}
}

// TestIsole_EquipeATerre_ExclueEtPubliee — personne ne pouvait accompagner.
func TestIsole_EquipeATerre_ExclueEtPubliee(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{"m1": "Slayer:Arena"}),
		mortContexte("m1", tsMoi, 2, 3, nil, 0, 0),
		mortContexte("m1", tsMoi, 4, 5, m(40), 1, 0))

	out := lireIsole(t, svc, "m1")
	if out.MortsEquipeATerre != 1 {
		t.Fatalf("morts_equipe_a_terre = %d, attendu 1", out.MortsEquipeATerre)
	}
	if out.Isolement.N != 1 {
		t.Fatalf("denominateur = %d, attendu 1 : la mort sans personne pour accompagner est "+
			"ECARTEE", out.Isolement.N)
	}
}

// TestIsole_VarianteSansRayon — LE MATCH SORT DE L'UNIVERS, PAS SEULEMENT DU NUMERATEUR.
//
// Le laisser au denominateur diviserait la mesure par des matchs qu'on a refuse de lire : deux
// matchs dont un Husky Raid rendraient 0,5 mort isolee par match au lieu de 1.
func TestIsole_VarianteSansRayon(t *testing.T) {
	svc, _ := svcIsole(universVariantes(map[string]string{
		"connu": "Slayer:Arena", "inconnu": "Husky Raid:CTF",
	}), mortContexte("connu", tsMoi, 2, 3, m(40), 1, 0))

	out := lireIsole(t, svc, "connu", "inconnu")
	if out.MatchsSansRayon != 1 {
		t.Fatalf("matchs_sans_rayon = %d, attendu 1", out.MatchsSansRayon)
	}
	if out.MatchsFiltres != 2 {
		t.Fatalf("matchs_filtres = %d, attendu 2 : les deux matchs restent dans l'univers du filtre",
			out.MatchsFiltres)
	}
	if out.MatchsRetenus != 1 {
		t.Fatalf("matchs_retenus = %d, attendu 1 : l'univers de la lecture est « mesure ET ayant "+
			"un rayon »", out.MatchsRetenus)
	}
	if out.Isolement.ParMatch != 1 {
		t.Fatalf("par match = %v, attendu 1 (1 isolee / 1 match ayant un rayon) — diviser par 2 "+
			"ferait varier la mesure avec les matchs qu'on refuse de lire", out.Isolement.ParMatch)
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
	if out.MatchsSansRayon != 1 || out.Isolement.N != 0 || len(out.Cellules) != 0 {
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
	if out.Isolement.N != 1 {
		t.Fatalf("denominateur = %d, attendu 1 : le perimetre ne retient que m1", out.Isolement.N)
	}
}

// TestIsole_LaCouvertureNeCompteQueMesMorts — la face VICTIME, jamais les deux.
//
// « Isole » mesure la part de MES morts survenues sans coequipier a portee. Compter aussi mes
// kills au denominateur de couverture ferait annoncer au pied de carte « N morts, M
// localisees » sur un N deux fois trop grand — la couverture ne decrirait plus la mesure
// affichee.
func TestIsole_LaCouvertureNeCompteQueMesMorts(t *testing.T) {
	univ := universVariantes(map[string]string{"m1": "Slayer:Arena"})
	repo := &mockTacticalRepo{
		univ: univ,
		morts: domain.TacticalMortsContexte{Univers: univ, Morts: []domain.MortContexte{
			mortContexte("m1", tsMoi, 2, 3, m(40), 1, 0),
		}},
		ev: domain.TacticalKillEvents{Univers: univ, Events: []domain.KillEvent{
			{MatchID: "m1", VictimXUID: tsMoi, KillerXUID: tsAdv, TimeMs: 1000},
			{MatchID: "m1", VictimXUID: tsAdv, KillerXUID: tsMoi, TimeMs: 2000},
			{MatchID: "m1", VictimXUID: tsAdv2, KillerXUID: tsMoi, TimeMs: 3000},
		}},
	}
	svc := NewTacticalService(repo, capsCompletes(), tsMoi).
		WithRadarRange(map[string]int{"Slayer:Arena": 18})

	out := lireIsole(t, svc, "m1")
	if out.EvenementsJournal != 1 {
		t.Fatalf("evenements_journal = %d, attendu 1 : une seule de MES morts, mes DEUX kills "+
			"n'entrent pas dans la couverture de cette lecture", out.EvenementsJournal)
	}
}
