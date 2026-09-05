package service

// replay_vehicle_labels_test.go — les REGLES de la table des sprites de vehicule, posee a la
// requete. Chaque test correspond a une decision ecrite dans l en-tete du fichier de production.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// replayVehicleTracks fabrique des vies ne portant QUE leur famille — la seule entree de
// `vehicleFamiliesUsed`.
func replayVehicleTracks(familles ...string) []replay.VehicleTrack {
	out := make([]replay.VehicleTrack, 0, len(familles))
	for i, f := range familles {
		out = append(out, replay.VehicleTrack{Slot: uint32(700 + i), Family: f})
	}
	return out
}

// artefactVehicules pose un artefact minimal dont les vies de vehicule portent les trois cas qui
// comptent : deux vies d une MEME famille resolue, une vie d une autre famille, et une vie dont
// le chassis n a pas ete resolu (famille vide).
func artefactVehicules(t *testing.T, root, titleSlug, matchID string) {
	t.Helper()
	doc := `{"schemaVersion":29,"matchId":"` + matchID + `","titleSlug":"` + titleSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"vehicles":[` +
		`{"slot":700,"gen":0,"chassis":"5b80c406","family":"ghost","t0":0,"t1":1,"t1max":1,"end":"inconnue"},` +
		`{"slot":701,"gen":0,"chassis":"5b80c406","family":"ghost","t0":0,"t1":1,"t1max":1,"end":"inconnue"},` +
		`{"slot":702,"gen":0,"chassis":"af31ab1a","family":"mongoose","t0":0,"t1":1,"t1max":1,"end":"inconnue"},` +
		`{"slot":703,"gen":0,"chassis":"fe32c0f4","t0":0,"t1":1,"t1max":1,"end":"inconnue"}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(titleSlug, matchID), doc)
}

// TestVehicleLabels_PoseLUrlDuSprite : une entree par FAMILLE employee, pas une par vie — et
// l URL est composee sous le dossier d assets du TITRE.
func TestVehicleLabels_PoseLUrlDuSprite(t *testing.T) {
	root := t.TempDir()
	artefactVehicules(t, root, title.DefaultSlug, "match-vehicules")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-vehicules")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(doc.VehicleLabels) != 2 {
		t.Fatalf("%d libelles de vehicule servis, attendu 2 (deux familles pour quatre vies) : "+
			"la table est keyee par FAMILLE, repeter l URL par vie n apprendrait rien",
			len(doc.VehicleLabels))
	}
	got := doc.VehicleLabels["ghost"]
	if got.Img != "/static/vehicles-assets/"+title.DefaultSlug+"/replay/ghost.png" {
		t.Errorf("URL du sprite ghost = %q : elle doit etre composee sous le dossier d assets du "+
			"titre, sous-dossier `replay`", got.Img)
	}
	if !got.Tinted {
		t.Error("Tinted faux : le sprite se teint a la couleur de l equipe qui occupe le vehicule")
	}
	if !strings.Contains(doc.VehicleLabels["mongoose"].Img, "/mongoose.png") {
		t.Errorf("URL du sprite mongoose = %q", doc.VehicleLabels["mongoose"].Img)
	}
}

// TestVehicleLabels_FamilleVideNAPasDEntree : une vie dont le chassis n est pas resolu ne cree
// AUCUNE entree — ni sous une cle vide, ni sous celle d une voisine.
func TestVehicleLabels_FamilleVideNAPasDEntree(t *testing.T) {
	root := t.TempDir()
	artefactVehicules(t, root, title.DefaultSlug, "match-vehicules")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-vehicules")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if _, ok := doc.VehicleLabels[""]; ok {
		t.Error("entree sous une cle VIDE : une vie sans famille n a pas de sprite, elle n a pas " +
			"non plus d entree")
	}
	// La vie 703 garde son chassis brut a l ecran, et rien d autre.
	for _, v := range doc.Vehicles {
		if v.Slot == 703 && v.Family != "" {
			t.Errorf("vie 703 : famille = %q, attendue vide", v.Family)
		}
	}
}

// TestVehicleLabels_DocumentSansVehiculeNaPasDeTable : l ABSENCE de table dit « rien a
// dessiner ». Une table vide dirait « regarde, il n y a rien » — le client ne peut pas
// distinguer les deux, et l artefact ne doit pas grossir pour rien.
func TestVehicleLabels_DocumentSansVehiculeNaPasDeTable(t *testing.T) {
	root := t.TempDir()
	artefactArmes(t, root, title.DefaultSlug, "match-armes")

	doc, err := NewReplayService(title.DefaultSlug, root, nil).
		GetReplay(context.Background(), "match-armes")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if doc.VehicleLabels != nil {
		t.Errorf("table servie sur un document sans vehicule : %v", doc.VehicleLabels)
	}
}

// TestVehicleLabels_ToutesLesFamillesInconnuesNeServentRien : un document dont AUCUNE vie ne
// resout de famille sert le rejeu ENTIER, sans table. C est une degradation, pas une erreur.
func TestVehicleLabels_ToutesLesFamillesInconnuesNeServentRien(t *testing.T) {
	root := t.TempDir()
	doc := `{"schemaVersion":29,"matchId":"m","titleSlug":"` + title.DefaultSlug + `",` +
		`"frameCount":1,"bounds":{"minX":0,"minY":0,"maxX":1,"maxY":1},"tracks":[],` +
		`"vehicles":[{"slot":700,"gen":0,"chassis":"fe32c0f4","t0":0,"t1":1,"t1max":1,` +
		`"end":"inconnue"}]}`
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "m"), doc)

	got, err := NewReplayService(title.DefaultSlug, root, nil).GetReplay(context.Background(), "m")
	if err != nil {
		t.Fatalf("lecture du rejeu : %v", err)
	}
	if len(got.Vehicles) != 1 {
		t.Fatalf("%d vies servies, attendu 1 — le document doit rester entier", len(got.Vehicles))
	}
	if got.VehicleLabels != nil {
		t.Errorf("table servie alors qu aucune famille n est resolue : %v", got.VehicleLabels)
	}
}

// TestVehicleFamiliesUsed_TrieEtCompte : l ordre d une map ne se publie pas, et les vies sans
// famille se comptent (c est le denominateur du journal de degradation).
func TestVehicleFamiliesUsed_TrieEtCompte(t *testing.T) {
	tracks := replayVehicleTracks("wraith", "ghost", "", "ghost", "")
	got, unnamed := vehicleFamiliesUsed(tracks)
	if len(got) != 2 || got[0] != "ghost" || got[1] != "wraith" {
		t.Errorf("familles = %v, attendu [ghost wraith] triees", got)
	}
	if unnamed != 2 {
		t.Errorf("vies sans famille = %d, attendu 2", unnamed)
	}
}
