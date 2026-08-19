package service

// Tests — LE CALQUE DES SOCLES SERVI AVEC LE REJEU, de bout en bout.
//
// CE QUE CE FICHIER VERROUILLE :
//   - un artefact qui porte des socles ressort avec les emplacements CONFIRMÉS, aux
//     positions du catalogue, et le compte du catalogue à côté ;
//   - LE TÉMOIN DE LA DÉCISION : un artefact SANS socle (le Super Fiesta) ne reçoit RIEN,
//     même quand la carte en porte au fichier ;
//   - une carte hors catalogue, un map_id vide, un catalogue illisible : absence propre,
//     jamais d'erreur — le rejeu se sert entier sans ce calque.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
)

// artefactAvecSocles écrit un artefact minimal portant deux socles.
const artefactAvecSocles = `{"schemaVersion":17,"matchId":"m1","titleSlug":"halo_infinite",` +
	`"frameCount":10,"bounds":{"minX":0,"minY":0,"maxX":10,"maxY":10},"tracks":[],` +
	`"weaponPads":[` +
	`{"x":-9.74,"y":0,"z":22.4,"weapon":"0x0A1992BC","spawns":[0],"presence":[{"t0":0,"tLow":5,"tHigh":9}]},` +
	`{"x":5.16,"y":0,"z":26.51,"weapon":"0x2B1824D5","spawns":[0],"presence":[{"t0":0,"tLow":5,"tHigh":9}]}]}`

// artefactSansSocle est le Super Fiesta : le film n'a publié aucun socle.
const artefactSansSocle = `{"schemaVersion":17,"matchId":"m2","titleSlug":"halo_infinite",` +
	`"frameCount":10,"bounds":{"minX":0,"minY":0,"maxX":10,"maxY":10},"tracks":[]}`

// catalogueSocles — trois emplacements au fichier pour la carte `map-pads`.
const catalogueSocles = `{"schema_version":1,"title_slug":"halo_infinite","maps":{` +
	`"map-pads":{"map_id":"map-pads","mvar_file":"t.mvar","level_id":7,"objects_n":30,"pads":[` +
	`{"pos":{"x":-9.738,"y":-0.003,"z":22.403},"type_id":"0x5F379533","family":"power","objects":1},` +
	`{"pos":{"x":0.257,"y":-0.003,"z":21.36},"type_id":"0x5E86D110","family":"powerup","objects":1},` +
	`{"pos":{"x":5.16,"y":-0.003,"z":26.501},"type_id":"0x6253CFC0","family":"rack","objects":1}]}}}`

// soclesFixture pose une racine neuve : les deux artefacts et le catalogue des socles.
func soclesFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, res.ReplayArtifactPath(title.DefaultSlug, "m1"), artefactAvecSocles)
	ecrire(t, res.ReplayArtifactPath(title.DefaultSlug, "m2"), artefactSansSocle)
	ecrire(t, res.MapWeaponPadsPath(title.DefaultSlug), catalogueSocles)
	return root
}

func TestMapWeaponPads_AllumesSeulement(t *testing.T) {
	root := soclesFixture(t)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: "map-pads"})

	doc, err := svc.GetReplay(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	mp := doc.MapWeaponPads
	if mp == nil {
		t.Fatal("MapWeaponPads absent, attendu servi")
	}
	if len(mp.Pads) != 2 {
		t.Fatalf("%d emplacements servis, attendu 2 : %+v", len(mp.Pads), mp.Pads)
	}
	if mp.CatalogN != 3 {
		t.Errorf("catalogN = %d, attendu 3 (le power-up éteint est compté, pas servi)", mp.CatalogN)
	}
	// LA POSITION EST CELLE DU CATALOGUE (-9.738), pas celle du film (-9.74).
	if mp.Pads[0].X != -9.738 || mp.Pads[0].Pad != 0 {
		t.Errorf("emplacement 0 = %+v, attendu la position du catalogue et le socle 0", mp.Pads[0])
	}
	// ET LES SOCLES DU MATCH SONT INTACTS : la présence reste celle du film.
	if len(doc.WeaponPads) != 2 || len(doc.WeaponPads[0].Presence) != 1 {
		t.Errorf("les socles du match ont été touchés : %+v", doc.WeaponPads)
	}
}

// TestMapWeaponPads_SansSocleAuFilm — LE TEST DE LA DÉCISION. La carte porte trois
// emplacements ; le film n'en a servi aucun ; il ne doit RIEN s'afficher.
func TestMapWeaponPads_SansSocleAuFilm(t *testing.T) {
	root := soclesFixture(t)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: "map-pads"})

	doc, err := svc.GetReplay(context.Background(), "m2")
	if err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	if doc.MapWeaponPads != nil {
		t.Fatalf("des emplacements sont servis sur un match SANS socle : %+v", doc.MapWeaponPads)
	}
}

func TestMapWeaponPads_AbsencesPropres(t *testing.T) {
	root := soclesFixture(t)
	cas := []struct {
		nom  string
		repo *mapNamesStub
	}{
		{"carte hors catalogue", &mapNamesStub{mapID: "carte-inconnue"}},
		{"map_id vide", &mapNamesStub{}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, root, c.repo)
			doc, err := svc.GetReplay(context.Background(), "m1")
			if err != nil {
				t.Fatalf("GetReplay: %v", err)
			}
			if doc.MapWeaponPads != nil {
				t.Errorf("calque servi alors qu'il ne peut pas l'être : %+v", doc.MapWeaponPads)
			}
			// Le rejeu reste ENTIER : les socles du film sont toujours là.
			if len(doc.WeaponPads) != 2 {
				t.Errorf("le rejeu a perdu ses socles : %+v", doc.WeaponPads)
			}
		})
	}
}

// TestMapWeaponPads_SansCatalogue — sans fichier de référence, le rejeu se sert entier et
// le client retombe sur son comportement d'avant.
func TestMapWeaponPads_SansCatalogue(t *testing.T) {
	root := t.TempDir()
	ecrire(t, title.NewPathResolver(root).ReplayArtifactPath(title.DefaultSlug, "m1"), artefactAvecSocles)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: "map-pads"})

	doc, err := svc.GetReplay(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	if doc.MapWeaponPads != nil {
		t.Errorf("calque servi sans catalogue : %+v", doc.MapWeaponPads)
	}
	if len(doc.WeaponPads) != 2 {
		t.Errorf("le rejeu a perdu ses socles : %+v", doc.WeaponPads)
	}
}

// TestMapWeaponPads_UneSeuleLectureDesCles — les deux calques statiques partagent la même
// résolution de carte : la base est interrogée UNE fois par requête, pas deux.
func TestMapWeaponPads_UneSeuleLectureDesCles(t *testing.T) {
	root := soclesFixture(t)
	repo := &mapNamesStub{mapID: "map-pads", pairName: "Arena:Slayer"}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	if _, err := svc.GetReplay(context.Background(), "m1"); err != nil {
		t.Fatalf("GetReplay: %v", err)
	}
	if len(repo.vu) != 1 {
		t.Errorf("%d interrogations de la base, attendu 1 : %v", len(repo.vu), repo.vu)
	}
}
