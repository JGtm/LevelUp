package service

import (
	"context"
	"os"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

// objectifsFixture pose, sous une racine neuve, tout ce que le calque d'objectifs
// exige : un artefact de rejeu minimal pour le match "m1", la table de rôles du titre et
// un catalogue d'objectifs v2 à une carte ("map-ctf") portant les trois natures d'objet
// mesurées sur Catalyst : des points drapeau, une livraison EN CYLINDRE (le rôle est
// MIXTE : 2 points + 2 cylindres sur la vraie carte) et une zone de Bastion en boîte.
func objectifsFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, res.ReplayArtifactPath(title.DefaultSlug, "m1"),
		`{"schemaVersion":3,"matchId":"m1","titleSlug":"halo_infinite","frameCount":10,`+
			`"bounds":{"minX":0,"minY":0,"maxX":10,"maxY":10},"tracks":[]}`)
	ecrire(t, res.TitleMappingsDir(title.DefaultSlug)+"/objective_roles.toml", `
[meta]
title_slug = "halo_infinite"
schema_version = 1

[[modes]]
match = ["CTF"]
roles = ["flag_spawn", "flag_delivery"]

[[modes]]
match = ["Strongholds"]
roles = ["strongholds_zone"]
neutral = true
`)
	ecrire(t, res.MapObjectivesPath(title.DefaultSlug),
		`{"schema_version":2,"title_slug":"halo_infinite","generated_at":"2026-08-13T00:00:00Z","maps":{`+
			`"map-ctf":{"map_id":"map-ctf","version_id":"v1","public_name":"Testmap","module":"testmod","objects_n":5,"objectives":[`+
			`{"role":"flag_spawn","type_id":1,"pos":{"x":0,"y":21,"z":26},"forward":{"x":1,"y":0,"z":0},"team_index":0,"instance_id":11,"labels":["flag_spawn"],"object_index":0},`+
			`{"role":"flag_spawn","type_id":1,"pos":{"x":0,"y":-21,"z":26},"forward":{"x":1,"y":0,"z":0},"team_index":1,"instance_id":12,"labels":["flag_spawn"],"object_index":1},`+
			`{"role":"flag_delivery","type_id":2,"pos":{"x":0,"y":20,"z":26},"forward":{"x":1,"y":0,"z":0},"team_index":0,"instance_id":13,"labels":["flag_delivery"],"object_index":2},`+
			`{"role":"flag_delivery","type_id":2,"pos":{"x":0,"y":-20,"z":26},"forward":{"x":0,"y":1,"z":0},"team_index":1,"instance_id":14,"labels":["flag_delivery"],"object_index":3,`+
			`"shape":{"family":"cylinder","radius":4.8,"up_z":2,"down_z":1,"forward":{"x":0,"y":1,"z":0},"up":{"x":0,"y":0,"z":1},"raw":{"family":2,"s5":314572,"s6":0,"s7":131072,"s8":65536}}},`+
			`{"role":"strongholds_zone","type_id":3,"pos":{"x":-15,"y":0,"z":22},"forward":{"x":0,"y":-1,"z":0},"team_index":1,"instance_id":15,"labels":["strongholds_zone"],"object_index":4,`+
			`"shape":{"family":"box","half_x":2.5,"half_y":3,"up_z":4,"down_z":1,"forward":{"x":0,"y":-1,"z":0},"up":{"x":0,"y":0,"z":1},"raw":{"family":3,"s5":327680,"s6":393216,"s7":262144,"s8":65536}}}`+
			`]}}}`)
	return root
}

// TestMapObjectives_CTF — le contrat du lot 4.2 : pair_name CTF -> rôles drapeau ->
// objets joints par map_id, points en marqueurs et volumes en zones, équipes CONSERVÉES
// (le drapeau appartient réellement à un camp).
func TestMapObjectives_CTF(t *testing.T) {
	root := objectifsFixture(t)
	repo := &mapNamesStub{mapID: "map-ctf", pairName: "Arena:CTF on Testmap"}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	doc, err := svc.GetReplay(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	mo := doc.MapObjectives
	if mo == nil {
		t.Fatal("MapObjectives absent, attendu servi")
	}
	// 3 marqueurs (2 spawns + 1 livraison ponctuelle), 1 zone (livraison cylindre) —
	// la zone de Bastion N'EST PAS servie : son rôle n'appartient pas au mode CTF.
	if len(mo.Markers) != 3 || len(mo.Zones) != 1 {
		t.Fatalf("markers=%d zones=%d, attendu 3/1 (%+v)", len(mo.Markers), len(mo.Zones), mo)
	}
	z := mo.Zones[0]
	if z.Role != "flag_delivery" || z.Family != "cylinder" || z.Radius != 4.8 || z.Team != 1 {
		t.Errorf("zone livraison inattendue: %+v", z)
	}
	teams := map[int]bool{}
	for _, m := range mo.Markers {
		teams[m.Team] = true
		if m.Role != "flag_spawn" && m.Role != "flag_delivery" {
			t.Errorf("rôle de marqueur hors mode: %+v", m)
		}
	}
	if !teams[0] || !teams[1] {
		t.Errorf("les équipes du drapeau doivent être conservées: %+v", mo.Markers)
	}
}

// TestMapObjectives_StrongholdsNeutre — la règle produit : la table marque le mode
// `neutral`, la zone s'affiche SANS camp même si le fichier lui donne team_index=1
// (possession dynamique non décodée — mesuré : 95/158 zones de Bastion du catalogue
// portent un camp de fichier).
func TestMapObjectives_StrongholdsNeutre(t *testing.T) {
	root := objectifsFixture(t)
	repo := &mapNamesStub{mapID: "map-ctf", pairName: "Arena:Strongholds on Testmap"}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	doc, err := svc.GetReplay(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	mo := doc.MapObjectives
	if mo == nil || len(mo.Zones) != 1 || len(mo.Markers) != 0 {
		t.Fatalf("attendu 1 zone / 0 marqueur, reçu %+v", mo)
	}
	z := mo.Zones[0]
	if z.Team != replay.TeamNeutral {
		t.Errorf("team = %d, attendu neutre (%d) — la possession est dynamique", z.Team, replay.TeamNeutral)
	}
	if z.Family != "box" || z.HalfX != 2.5 || z.HalfY != 3 || z.FwdY != -1 {
		t.Errorf("géométrie de boîte inattendue: %+v", z)
	}
}

// TestMapObjectives_Absences — CHAQUE maillon manquant rend un champ ABSENT sur un
// document ENTIER : jamais d'erreur, jamais de rejeu perdu (règle du lot : « map_id
// vide / carte inconnue = champ absent »).
func TestMapObjectives_Absences(t *testing.T) {
	cas := []struct {
		nom  string
		stub *mapNamesStub
	}{
		{"mode sans objectifs (Slayer)", &mapNamesStub{mapID: "map-ctf", pairName: "Super Fiesta:Slayer on Testmap - Forge"}},
		{"mode inconnu de la table (KOTH)", &mapNamesStub{mapID: "map-ctf", pairName: "Arena:King of the Hill on Testmap"}},
		{"map_id vide", &mapNamesStub{pairName: "Arena:CTF on Testmap"}},
		{"pair_name vide", &mapNamesStub{mapID: "map-ctf"}},
		{"pair_name UUID brut", &mapNamesStub{mapID: "map-ctf", pairName: "100e12e4-402a-4163-8073-9d0cf5f658ec"}},
		{"carte hors catalogue", &mapNamesStub{mapID: "map-inconnue", pairName: "Arena:CTF on Testmap"}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, objectifsFixture(t), c.stub)
			doc, err := svc.GetReplay(context.Background(), "m1")
			if err != nil {
				t.Fatalf("GetReplay doit servir le document: %v", err)
			}
			if doc.MapObjectives != nil {
				t.Errorf("MapObjectives = %+v, attendu absent", doc.MapObjectives)
			}
			if doc.MatchID != "m1" {
				t.Errorf("le document lui-même doit rester entier: %+v", doc.MatchID)
			}
		})
	}
}

// TestMapObjectives_TitreSansTable — un titre sans objective_roles.toml n'a pas de
// calque, silencieusement (cas nominal d'un second titre) ; et un repo absent (maps nil)
// ne casse pas GetReplay.
func TestMapObjectives_TitreSansTable(t *testing.T) {
	root := objectifsFixture(t)
	res := title.NewPathResolver(root)
	if err := os.Remove(res.TitleMappingsDir(title.DefaultSlug) + "/objective_roles.toml"); err != nil {
		t.Fatal(err)
	}
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: "map-ctf", pairName: "Arena:CTF on Testmap"})
	doc, err := svc.GetReplay(context.Background(), "m1")
	if err != nil || doc.MapObjectives != nil {
		t.Errorf("attendu document entier sans calque, reçu err=%v mo=%+v", err, doc.MapObjectives)
	}

	sansRepo := NewReplayService(title.DefaultSlug, root, nil)
	doc, err = sansRepo.GetReplay(context.Background(), "m1")
	if err != nil || doc.MapObjectives != nil {
		t.Errorf("maps nil : attendu document entier sans calque, reçu err=%v mo=%+v", err, doc.MapObjectives)
	}
}

// TestMapObjectives_DonneesReelles — L'ORACLE SUR LE CATALOGUE VERSIONNÉ (même règle que
// les callouts : pas de SKIP quand le catalogue manque, il est versionné).
//
// Catalyst en CTF (le match de vérification du gate, 64e8adfa) : 3 apparitions de drapeau,
// 2 livraisons ponctuelles et 2 livraisons EN CYLINDRE = **7 marqueurs, 0 zone**. La zone de
// Bastion, les zones d'Extraction et les apparitions d'Oddball de la même carte ne sortent PAS.
//
// CE TEST FIGEAIT LE DÉFAUT JUSQU'AU 2026-08-26 : il attendait 5 marqueurs et 2 ZONES, et
// c'étaient ces deux zones que l'utilisateur voyait s'afficher comme des bases sur un CTF. Il
// passait sa propre liste de rôles, sans le drapeau `points_only` de la table du titre — donc
// il attestait d'un comportement que la production n'a jamais eu à avoir. Il lit désormais la
// TABLE VERSIONNÉE, comme le service, et c'est la seule façon qu'il avait d'attraper ça.
func TestMapObjectives_DonneesReelles(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	if _, statErr := os.Stat(res.MapObjectivesPath(title.DefaultSlug)); statErr != nil {
		t.Fatalf("catalogue d'objectifs versionné absent : %v", statErr)
	}
	cat, err := replay.LoadMapObjectives(res.MapObjectivesPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue versionné illisible : %v", err)
	}
	entry, err := cat.Lookup("f7e8cde9-0c0a-487c-94a3-61bfa0f20465") // Catalyst
	if err != nil {
		t.Fatalf("Catalyst absent du catalogue : %v", err)
	}
	svc := &replayService{titleSlug: title.DefaultSlug, repoRoot: root}
	mo := replay.BuildMapObjectives(entry, svc.objectiveRoleSpecs(context.Background(), "Arena:CTF on Catalyst"))
	if mo == nil || len(mo.Markers) != 7 || len(mo.Zones) != 0 {
		t.Fatalf("Catalyst CTF : markers=%d zones=%d, attendu 7/0",
			markersCount(mo), zonesCount(mo))
	}
	livraisons := 0
	for _, m := range mo.Markers {
		if m.Role == "flag_delivery" {
			livraisons++
		}
	}
	// LES QUATRE LIVRAISONS SORTENT, pas seulement les deux ponctuelles : le correctif
	// REDESSINE les objets à forme, il ne les écarte pas.
	if livraisons != 4 {
		t.Errorf("livraisons servies = %d, attendu 4 (2 ponctuelles + 2 cylindres redessinés)", livraisons)
	}
	// Le spawn NEUTRE (drapeau du milieu, team -1) est servi tel quel : c'est le
	// variant Neutral Flag qui l'emploie, et l'absence de camp est une donnée.
	neutres := 0
	for _, m := range mo.Markers {
		if m.Team == replay.TeamNeutral {
			neutres++
		}
	}
	if neutres != 1 {
		t.Errorf("marqueurs neutres = %d, attendu 1 (le drapeau central)", neutres)
	}
}

// TestMapObjectives_ModesPonctuels_AucuneZoneSurTOUTLeCatalogue — LA GARDE DU CORRECTIF
// « des bases s'affichent sur un CTF » (2026-08-26).
//
// Elle ne vise pas une carte : elle balaie les 73 entrées du catalogue VERSIONNÉ pour les
// quatre modes dont l'objectif se TOUCHE au lieu de se TENIR. Aucune ne doit produire une
// seule zone — y compris les 14 cartes dont le fichier donne une FORME à leurs livraisons de
// drapeau, les 16 dont tous les navpoints de Stockpile en portent une, et les 2 de la bombe
// d'Assaut.
//
// POURQUOI SUR TOUT LE CATALOGUE ET PAS SUR CATALYST SEUL : le défaut n'était pas propre à une
// carte, et une garde posée sur un seul exemplaire laisserait passer la prochaine carte
// extraite dont le fichier donnerait une forme à un objectif ponctuel. Le compte des cartes
// RÉELLEMENT porteuses de formes est vérifié au passage — sans lui, le test resterait vert le
// jour où le catalogue perdrait ces formes, et il ne prouverait plus rien.
func TestMapObjectives_ModesPonctuels_AucuneZoneSurTOUTLeCatalogue(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	cat, err := replay.LoadMapObjectives(res.MapObjectivesPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue versionné illisible : %v", err)
	}
	svc := &replayService{titleSlug: title.DefaultSlug, repoRoot: root}
	ctx := context.Background()
	pairs := map[string]string{
		"CTF":       "Arena:CTF on X",
		"Oddball":   "Arena:Oddball on X",
		"Stockpile": "BTB:Stockpile on X",
		"Assault":   "Arena:Assault on X",
	}
	// avecForme compte les cartes qui portent AU MOINS un objet à forme sur un rôle du mode :
	// c'est le dénominateur qui rend la garde non tautologique.
	avecForme := map[string]int{}
	for mode, pair := range pairs {
		specs := svc.objectiveRoleSpecs(ctx, pair)
		if len(specs) == 0 {
			t.Fatalf("%s : la table du titre ne sert aucun rôle — le mode n'est plus reconnu", mode)
		}
		for _, spec := range specs {
			if !spec.PointsOnly {
				t.Errorf("%s : le rôle %q n'est pas points_only dans la table versionnée", mode, spec.Role)
			}
		}
		for mapID, entry := range cat.Maps {
			formes := 0
			for _, spec := range specs {
				formes += len(entry.ZonesOfRole(spec.Role).Zones)
			}
			if formes > 0 {
				avecForme[mode]++
			}
			mo := replay.BuildMapObjectives(entry, specs)
			if zonesCount(mo) != 0 {
				t.Errorf("%s sur %s : %d zone(s) servie(s) — un mode ponctuel ne dessine JAMAIS de base",
					mode, mapID, zonesCount(mo))
			}
			// L'objet à forme n'est pas perdu : il est redessiné en marqueur.
			if formes > 0 && markersCount(mo) < formes {
				t.Errorf("%s sur %s : %d objet(s) à forme pour seulement %d marqueur(s) — des objectifs ont disparu",
					mode, mapID, formes, markersCount(mo))
			}
		}
	}
	// Relevé du 2026-08-26 sur le catalogue versionné. Ces comptes SONT la preuve que la garde
	// a quelque chose à attraper : à zéro, elle ne testerait plus rien.
	for mode, attendu := range map[string]int{"CTF": 14, "Stockpile": 16, "Assault": 2} {
		if avecForme[mode] != attendu {
			t.Errorf("%s : %d carte(s) à forme au catalogue, relevé %d le 2026-08-26 — "+
				"le catalogue a bougé, revérifier le correctif avant d'ajuster ce compte",
				mode, avecForme[mode], attendu)
		}
	}
}

// TestMapObjectives_KOTH_DonneesReelles — lot C-ter volet 2 : la table VERSIONNÉE reconnaît
// King of the Hill sous ses deux libellés du registre (« Arena:King of the Hill on X »
// normalisé, « Ranked:King of the Hill on X ») et sert le rôle `hill`, NEUTRE ; sur Catalyst
// le catalogue versionné porte 6 collines (toutes des volumes : aucun marqueur), et le même
// pair_name en CTF n'en sert aucune.
func TestMapObjectives_KOTH_DonneesReelles(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	svc := &replayService{titleSlug: title.DefaultSlug, repoRoot: root}
	ctx := context.Background()
	for _, pair := range []string{"Arena:King of the Hill on Catalyst", "Ranked:King of the Hill on Solitude"} {
		specs := svc.objectiveRoleSpecs(ctx, pair)
		if len(specs) != 1 || specs[0].Role != "hill" || !specs[0].Neutral {
			t.Fatalf("%q : specs = %+v, attendu [{hill neutre}]", pair, specs)
		}
	}
	for _, spec := range svc.objectiveRoleSpecs(ctx, "Arena:CTF on Catalyst") {
		if spec.Role == "hill" {
			t.Fatalf("CTF sert le rôle hill — la table déborde de son mode")
		}
	}
	res := title.NewPathResolver(root)
	cat, err := replay.LoadMapObjectives(res.MapObjectivesPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue versionné illisible : %v", err)
	}
	entry, err := cat.Lookup("f7e8cde9-0c0a-487c-94a3-61bfa0f20465") // Catalyst
	if err != nil {
		t.Fatalf("Catalyst absent du catalogue : %v", err)
	}
	mo := replay.BuildMapObjectives(entry, svc.objectiveRoleSpecs(ctx, "Arena:King of the Hill on Catalyst"))
	if mo == nil || len(mo.Zones) != 6 || len(mo.Markers) != 0 {
		t.Fatalf("Catalyst KOTH : zones=%d markers=%d, attendu 6/0", zonesCount(mo), markersCount(mo))
	}
	for _, z := range mo.Zones {
		if z.Role != "hill" || z.Team != replay.TeamNeutral || (z.Family != "box" && z.Family != "cylinder") {
			t.Errorf("colline inattendue : %+v", z)
		}
	}
}

func zonesCount(mo *replay.MapObjectives) int {
	if mo == nil {
		return 0
	}
	return len(mo.Zones)
}

func markersCount(mo *replay.MapObjectives) int {
	if mo == nil {
		return 0
	}
	return len(mo.Markers)
}
