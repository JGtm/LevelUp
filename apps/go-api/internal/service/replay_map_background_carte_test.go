package service

// replay_map_background_carte_test.go — LE MEME FOND, DEMANDE PAR CARTE.
//
// La grille de l'onglet Tactique liste des cartes : elle n'a aucun match_id sous la main.
// Ces tests verifient que l'entree par map_id emprunte EXACTEMENT la cascade du chemin par
// match (cle map_id d'abord, index des noms ensuite) et rend les memes sentinelles — sans
// quoi une meme carte aurait deux fonds possibles selon la page qui la demande.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// TestMapBackgroundForMap_ParNom — carte NATIVE : le fond est keye par MODULE installe, et
// seul l'index des noms y mene. La carte est designee par son map_id ; c'est donc bien le
// nom candidat, resolu depuis ce map_id, qui doit faire le lien.
func TestMapBackgroundForMap_ParNom(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	bg, err := svc.MapBackgroundForMap(context.Background(), "asset-cliffhanger")
	if err != nil {
		t.Fatalf("MapBackgroundForMap: %v", err)
	}
	if bg.Module != "ridgeline" {
		t.Errorf("module = %q, attendu ridgeline", bg.Module)
	}
	if bg.Calibration.MetersPerPixel != 0.092 {
		t.Errorf("calage non transmis : %+v", bg.Calibration)
	}
	// La carte, et rien qu'elle, a ete demandee au registre : aucun match n'entre ici.
	if len(repo.vu) == 0 || repo.vu[0] != "asset-cliffhanger" {
		t.Errorf("la carte n'a pas ete demandee au registre : %v", repo.vu)
	}

	blob, err := svc.MapBackgroundImageForMap(context.Background(), "asset-cliffhanger")
	if err != nil {
		t.Fatalf("MapBackgroundImageForMap: %v", err)
	}
	if len(blob) < 4 || string(blob[:4]) != "\x89PNG" {
		t.Errorf("les octets servis ne sont pas ceux du fichier : %q", blob)
	}
}

// TestMapBackgroundForMap_ParMapID — carte FORGE : le fond vit sous la cle map_id, et la
// PRESENCE du sidecar sous cette cle decide, avant tout repli par nom.
func TestMapBackgroundForMap_ParMapID(t *testing.T) {
	const mapID = "105f5d84-8de1-4908-af3a-1c4f3bf9d642"
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	fondSousCle(t, root, title.DefaultSlug, mapID)
	// Le stub rend AUSSI un nom qui menerait a « ridgeline » : si la cle map_id ne passait
	// pas en premier, le test servirait le fond de l'autre carte sans que rien ne le dise.
	repo := &mapNamesStub{mapID: mapID, names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	bg, err := svc.MapBackgroundForMap(context.Background(), mapID)
	if err != nil {
		t.Fatalf("MapBackgroundForMap: %v", err)
	}
	if bg.Module != mapID {
		t.Errorf("module = %q, attendu la cle map_id %q", bg.Module, mapID)
	}
}

// TestMapBackgroundForMap_SansFond — carte sans image figee : sentinelle d'absence, jamais
// une erreur de panne. C'est le cas NORMAL d'une bonne partie des cartes.
func TestMapBackgroundForMap_SansFond(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{mapID: "asset-inconnue", names: []string{"CarteSansFond"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	if _, err := svc.MapBackgroundForMap(context.Background(), "asset-inconnue"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
	if _, err := svc.MapBackgroundImageForMap(context.Background(), "asset-inconnue"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err image = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}

// TestMapBackgroundImageForMap_ImageManquante — LE CALAGE SEUL NE SUFFIT PAS. Un sidecar
// present mais aucune image : l'image n'est pas servie (et l'inverse non plus, cf. le
// chemin par match). Une image sans calage ne se superpose a rien.
func TestMapBackgroundImageForMap_ImageManquante(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", false)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	if _, err := svc.MapBackgroundForMap(context.Background(), "asset-cliffhanger"); err != nil {
		t.Fatalf("le calage doit rester lisible : %v", err)
	}
	if _, err := svc.MapBackgroundImageForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}

// TestMapBackgroundForMap_RegistreEnEchec — carte non resolue : degradation propre, jamais
// une devinette sur le nom.
func TestMapBackgroundForMap_RegistreEnEchec(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{err: errors.New("registre illisible")}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	if _, err := svc.MapBackgroundForMap(context.Background(), "asset-x"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Fatalf("err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}
