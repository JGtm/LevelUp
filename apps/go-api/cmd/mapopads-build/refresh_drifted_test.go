package main

// refresh_drifted_test.go — LES TROIS PROMESSES DE LA RE-VALIDATION.
//
//	1. carte CONCORDANTE -> intacte, byte-identique ;
//	2. carte DERIVEE     -> entree COMPLETE regeneree (socles ET points) ;
//	3. un RAPPORT de diff est produit — sans lui, l'automatique cesse d'etre auditable, ce qui
//	   est la seule condition a laquelle il a ete accepte.

import (
	"context"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
)

func rdSocle(x float64, family string) replay.MapWeaponPadSpot {
	return replay.MapWeaponPadSpot{
		Pos: mapvar.Vec3{X: x}, TypeID: "0x5F379533", Family: family, Objects: 1,
	}
}

// TestRefreshDriftedRegenereLaDeriveeEtLaisseLaConcordante — promesses 1, 2 et 3.
func TestRefreshDriftedRegenereLaDeriveeEtLaisseLaConcordante(t *testing.T) {
	pointsAnciens := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	entree := func(id string, objets int) replay.MapWeaponPadsEntry {
		cp := append([]replay.MapSpawnPointSpot(nil), pointsAnciens...)
		return replay.MapWeaponPadsEntry{
			MapID: id, MvarFile: id + ".mvar", ObjectsN: objets, LevelID: 7,
			Pads:        []replay.MapWeaponPadSpot{rdSocle(1, "power"), rdSocle(5, "rack")},
			SpawnPoints: &cp,
		}
	}
	cat := &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
		Maps: map[string]replay.MapWeaponPadsEntry{
			"concordante": entree("concordante", 400),
			"derivee":     entree("derivee", 462),
		},
	}
	objectifs := &replay.MapObjectivesCatalog{
		SchemaVersion: replay.MapObjectivesSchemaVersion,
		Maps: map[string]replay.MapObjectivesEntry{
			"concordante": {MapID: "concordante", MvarFile: "concordante.mvar"},
			"derivee":     {MapID: "derivee", MvarFile: "derivee.mvar"},
		},
	}
	// L'ingestion factice rend, pour chaque carte, ce que le `.mvar` FRAIS donnerait.
	pointsNeufs := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 3}, TypeID: "0xE42158DF", Kind: "equipment", Objects: 1},
		{Pos: mapvar.Vec3{X: 7}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	aoAvecIngestFactice(t, func(mapID string, _ replay.MapObjectivesEntry, _, base string,
	) (replay.MapWeaponPadsEntry, int, error) {
		if mapID == "concordante" {
			// IDENTIQUE au catalogue : la carte ne doit pas etre touchee.
			return entree("concordante", 400), 0, nil
		}
		// DERIVEE : moins d'objets, un socle deplace de 4 m, un socle en plus.
		np := append([]replay.MapSpawnPointSpot(nil), pointsNeufs...)
		return replay.MapWeaponPadsEntry{
			MapID: "derivee", MvarFile: base, ObjectsN: 410, LevelID: 7,
			Pads: []replay.MapWeaponPadSpot{
				rdSocle(1, "power"), rdSocle(9, "rack"), rdSocle(12, "rack")},
			SpawnPoints: &np,
		}, 0, nil
	})
	chemin := aoCatalogue(t, t.TempDir(), cat)
	refreshDrifted(context.Background(), objectifs,
		aoDepotAvec(t, objectifs, "concordante.mvar", "derivee.mvar"), chemin, false)
	relu := aoRelire(t, chemin)

	// Promesse 1 — la concordante est intacte.
	c := relu.Maps["concordante"]
	if c.ObjectsN != 400 || len(c.Pads) != 2 || c.SpawnPoints == nil ||
		len(*c.SpawnPoints) != 1 {
		t.Errorf("la carte CONCORDANTE a ete touchee : %+v", c)
	}
	// Promesse 2 — la derivee est regeneree, socles ET points.
	dv := relu.Maps["derivee"]
	if dv.ObjectsN != 410 {
		t.Errorf("objects_n = %d, attendu 410 (le .mvar frais fait foi)", dv.ObjectsN)
	}
	if len(dv.Pads) != 3 {
		t.Errorf("%d socles, attendu 3 — les SOCLES doivent etre regeneres aussi", len(dv.Pads))
	}
	if dv.SpawnPoints == nil || len(*dv.SpawnPoints) != 2 {
		t.Errorf("points d'apparition non regeneres : %v", dv.SpawnPoints)
	}
	// Promesse 3 — le rapport existe, nomme la carte, et signale le deplacement.
	note := relu.Notes["refresh_drifted"]
	if !strings.Contains(note, "derivee") {
		t.Errorf("la note ne nomme pas la carte regeneree : %q", note)
	}
	if !strings.Contains(note, "ATTENTION") {
		t.Errorf("un socle ajoute et un socle deplace de 4 m doivent etre SIGNALES en tete, "+
			"pas enterres : %q", note)
	}
	// LA FORME COMPTE : la note porte un COMPTEUR « 1 concordantes », ce qui est legitime.
	// Ce qui ne doit pas y etre, c'est une LIGNE DE DIFF pour cette carte — elle s'ecrit
	// `(map_id)`.
	if strings.Contains(note, "(concordante)") {
		t.Errorf("la carte concordante ne doit pas avoir de ligne de diff : %q", note)
	}
}

// TestComparerSoclesDitCeQuiABouge — le rapport lui-meme.
func TestComparerSoclesDitCeQuiABouge(t *testing.T) {
	avant := []replay.MapWeaponPadSpot{rdSocle(1, "power"), rdSocle(5, "rack")}
	cas := []struct {
		nom      string
		apres    []replay.MapWeaponPadSpot
		ajoutes  int
		retires  int
		deplaces int
		pire     float64
	}{
		{"identiques", avant, 0, 0, 0, 0},
		{"un socle en plus", append(append([]replay.MapWeaponPadSpot{}, avant...),
			rdSocle(9, "rack")), 1, 0, 0, 0},
		{"un socle en moins", avant[:1], 0, 1, 0, 0},
		{"un socle deplace de 4 m", []replay.MapWeaponPadSpot{
			rdSocle(1, "power"), rdSocle(9, "rack")}, 0, 0, 1, 4},
		// LA FAMILLE COMPTE : un `rack` ne s'apparie pas a un `power`, donc l'un est retire
		// et l'autre ajoute — jamais « deplace de 4 m ».
		{"la famille a change", []replay.MapWeaponPadSpot{
			rdSocle(1, "power"), rdSocle(5, "power")}, 1, 1, 0, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d := comparerSocles(avant, c.apres)
			if d.ajoutes != c.ajoutes || d.retires != c.retires || len(d.deplaces) != c.deplaces {
				t.Errorf("diff = %+v, attendu %d ajoutes, %d retires, %d deplaces",
					d, c.ajoutes, c.retires, c.deplaces)
			}
			if c.pire > 0 && d.pireDeplacement() < c.pire-0.01 {
				t.Errorf("deplacement max = %.2f, attendu ~%.2f", d.pireDeplacement(), c.pire)
			}
		})
	}
}

// TestRefreshDriftedRefuseUnDeplacementAnormal — LA GARDE, et elle passe AVANT l'ecriture.
//
// Un socle qui bouge de plus de dix metres n'est pas une mise a jour de carte : c'est la
// signature du MAUVAIS FICHIER (carte de base plaquee sur la variante). La premiere passe de ce
// chantier a rendu jusqu'a 79,87 m et allait les ECRIRE ; seule une verification humaine l'a
// arretee. Sans ce test, la garde peut sauter sans que rien ne rougisse.
func TestRefreshDriftedRefuseUnDeplacementAnormal(t *testing.T) {
	pts := []replay.MapSpawnPointSpot{
		{Pos: mapvar.Vec3{X: 1}, TypeID: "0xADEEE6D8", Kind: "grenade", Objects: 1},
	}
	entree := func() replay.MapWeaponPadsEntry {
		cp := append([]replay.MapSpawnPointSpot(nil), pts...)
		return replay.MapWeaponPadsEntry{
			MapID: "m", MvarFile: "m.mvar", ObjectsN: 462, LevelID: 7,
			Pads:        []replay.MapWeaponPadSpot{rdSocle(1, "power")},
			SpawnPoints: &cp,
		}
	}
	objectifs := &replay.MapObjectivesCatalog{
		SchemaVersion: replay.MapObjectivesSchemaVersion,
		Maps:          map[string]replay.MapObjectivesEntry{"m": {MapID: "m", MvarFile: "m.mvar"}},
	}
	cas := []struct {
		nom      string
		x        float64
		accepter bool
		ecrit    bool
	}{
		// Sous le seuil : derive normale, l'automatique fait son travail.
		{"socle deplace de 3 m — accepte", 4, false, true},
		// Au-dela : la signature du mauvais fichier. REFUSE.
		{"socle deplace de 80 m — REFUSE", 81, false, false},
		// Le geste humain leve la garde.
		{"socle deplace de 80 m avec --accept-large-moves", 81, true, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			ancien := accepterGrandsDeplacements
			accepterGrandsDeplacements = c.accepter
			t.Cleanup(func() { accepterGrandsDeplacements = ancien })
			np := append([]replay.MapSpawnPointSpot(nil), pts...)
			aoAvecIngestFactice(t, func(_ string, _ replay.MapObjectivesEntry, _, base string,
			) (replay.MapWeaponPadsEntry, int, error) {
				return replay.MapWeaponPadsEntry{
					MapID: "m", MvarFile: base, ObjectsN: 410, LevelID: 7,
					Pads:        []replay.MapWeaponPadSpot{rdSocle(c.x, "power")},
					SpawnPoints: &np,
				}, 0, nil
			})
			cat := &replay.MapWeaponPadsCatalog{
				SchemaVersion: replay.MapWeaponPadsSchemaVersion, TitleSlug: "halo_infinite",
				Maps: map[string]replay.MapWeaponPadsEntry{"m": entree()},
			}
			chemin := aoCatalogue(t, t.TempDir(), cat)
			refreshDrifted(context.Background(), objectifs,
				aoDepotAvec(t, objectifs, "m.mvar"), chemin, false)
			relu := aoRelire(t, chemin).Maps["m"]
			ecrit := relu.ObjectsN == 410
			if ecrit != c.ecrit {
				t.Errorf("ecrit = %v, attendu %v (objects_n = %d)", ecrit, c.ecrit, relu.ObjectsN)
			}
			if !c.ecrit && relu.Pads[0].Pos.X != 1 {
				t.Errorf("le socle a bouge alors que la carte devait etre REFUSEE : %v",
					relu.Pads[0].Pos)
			}
			// LE REFUS SE DIT dans la note : un refus silencieux serait un trou de plus.
			note := aoRelire(t, chemin).Notes["refresh_drifted"]
			if !c.ecrit && !strings.Contains(note, "REFUSEE") {
				t.Errorf("le refus doit figurer au rapport : %q", note)
			}
		})
	}
}
