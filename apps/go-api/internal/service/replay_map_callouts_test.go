package service

import (
	"context"
	"errors"
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/testutil"
)

// calloutsFixture pose, sous une racine neuve, le catalogue de bornes et un catalogue de
// callouts minimal (une zone sur ridgeline). Rend la racine.
func calloutsFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, res.MapQuantBoundsPath(title.DefaultSlug),
		`{"schemaVersion":1,"source":"test","maps":{`+
			`"cliffhanger":{"module":"ridgeline","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]},`+
			`"dynasty":{"module":"fo08_wetland","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`)
	ecrire(t, res.MapCalloutsPath(title.DefaultSlug),
		`{"schema_version":1,"title_slug":"halo_infinite","source":"test","maps":{`+
			`"ridgeline":{"module":"ridgeline","provenance":"decoupe","zones":[`+
			`{"volume_index":10,"name":"ridgeline horses","en":"Horseshoe","fr":"Fer à cheval",`+
			`"x":19,"y":10,"z":1,"z_bottom":-0.2,"z_top":11,"big":false,`+
			`"polygon":[[14.8,7.5],[23.8,7.5],[23.8,15.3]]}]}}}`)
	return root
}

// TestMapCallouts_ChaineComplete — le contrat : match -> carte -> module -> zones.
func TestMapCallouts_ChaineComplete(t *testing.T) {
	root := calloutsFixture(t)
	repo := &mapNamesStub{names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapCallouts: %v", err)
	}
	if entry.Module != "ridgeline" || len(entry.Zones) != 1 {
		t.Fatalf("entrée inattendue: %+v", entry)
	}
	z := entry.Zones[0]
	if z.FR != "Fer à cheval" || z.EN != "Horseshoe" || len(z.Polygon) != 3 || z.ZTop != 11 {
		t.Errorf("zone incomplète: %+v", z)
	}
	if len(repo.vu) == 0 || repo.vu[0] != "m1" {
		t.Errorf("le match n'a pas été demandé au registre : %v", repo.vu)
	}
}

// TestMapCallouts_ForgeSansZones — LE CAS FORGE, PAR CONSTRUCTION.
//
// Dynasty résout vers son canevas fo08_wetland (catalogue de bornes) ; le canevas ne
// porte AUCUNE zone nommée et n'est pas au catalogue de callouts. La réponse est
// l'absence propre — jamais les zones d'une autre carte, jamais une erreur.
func TestMapCallouts_ForgeSansZones(t *testing.T) {
	root := calloutsFixture(t)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Dynasty"}})
	if _, err := svc.MapCallouts(context.Background(), "m1"); !errors.Is(err, port.ErrMapCalloutsNotAvailable) {
		t.Errorf("err = %v, attendu ErrMapCalloutsNotAvailable", err)
	}
}

// TestMapCallouts_Absences — chaque maillon manquant rend la MÊME sentinelle, jamais un
// 500 (même règle que le fond de carte).
func TestMapCallouts_Absences(t *testing.T) {
	cas := []struct {
		nom  string
		root func(t *testing.T) string
		repo port.ReplayMapNameRepo
	}{
		{
			nom:  "pas de résolution de carte câblée",
			root: calloutsFixture,
			repo: nil,
		},
		{
			nom:  "carte inconnue en base",
			root: calloutsFixture,
			repo: &mapNamesStub{err: errors.New("pas de ligne")},
		},
		{
			nom:  "nom hors du catalogue de modules",
			root: calloutsFixture,
			repo: &mapNamesStub{names: []string{"Nulle Part"}},
		},
		{
			nom:  "catalogue de callouts absent",
			root: func(t *testing.T) string { return fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", false) },
			repo: &mapNamesStub{names: []string{"Cliffhanger"}},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, c.root(t), c.repo)
			if _, err := svc.MapCallouts(context.Background(), "m1"); !errors.Is(err, port.ErrMapCalloutsNotAvailable) {
				t.Errorf("err = %v, attendu ErrMapCalloutsNotAvailable", err)
			}
		})
	}
}

// TestMapCallouts_IsoleParTitre — les zones d'un titre ne sont jamais servies à un autre
// (isolation par chemin FS, ADR 0008).
func TestMapCallouts_IsoleParTitre(t *testing.T) {
	root := calloutsFixture(t)
	svc := NewReplayService("halo_5", root, &mapNamesStub{names: []string{"Cliffhanger"}})
	if _, err := svc.MapCallouts(context.Background(), "m1"); !errors.Is(err, port.ErrMapCalloutsNotAvailable) {
		t.Errorf("err = %v, attendu ErrMapCalloutsNotAvailable", err)
	}
}

// TestMapCallouts_DonneesReelles — L'ORACLE SUR LE CATALOGUE VERSIONNÉ.
//
// Cliffhanger -> ridgeline : la carte de référence du gate visuel. 28 zones dont 11
// grandes (classement du POC), provenance découpée, libellés FR/EN pleins. Il ne se
// déclare pas SKIP quand le catalogue manque : il est versionné (git ls-files, 2026-08-19),
// son absence est un échec. La racine vient de testutil.RepoRoot() (déduite de l'arbre
// source) : jusqu'au 2026-08-19 la porte « racine introuvable » sautait ce test à chaque
// exécution hors LEVELUP_REPO_ROOT, CI comprise (revue ronde 2, R2-1).
func TestMapCallouts_DonneesReelles(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	if _, statErr := os.Stat(res.MapCalloutsPath(title.DefaultSlug)); statErr != nil {
		t.Fatalf("catalogue de callouts versionné absent : %v", statErr)
	}
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Cliffhanger"}})
	entry, err := svc.MapCallouts(context.Background(), "000d5950")
	if err != nil {
		t.Fatalf("zones de Cliffhanger : %v", err)
	}
	grandes := 0
	for _, z := range entry.Zones {
		if z.Big {
			grandes++
		}
		if z.FR == "" || z.EN == "" {
			t.Errorf("vi=%d : libellé vide", z.VolumeIndex)
		}
	}
	if entry.Module != "ridgeline" || len(entry.Zones) != 28 || grandes != 11 {
		t.Errorf("ridgeline : module=%q zones=%d grandes=%d, attendu ridgeline/28/11",
			entry.Module, len(entry.Zones), grandes)
	}
	if entry.Provenance != "decoupe" {
		t.Errorf("provenance = %q, attendu decoupe (dump versionné conservé)", entry.Provenance)
	}
}
