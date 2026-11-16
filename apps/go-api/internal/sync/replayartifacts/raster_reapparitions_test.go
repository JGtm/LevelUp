package replayartifacts

// raster_reapparitions_test.go — LA PART « REAPPARITION » DU SIDECAR : l'equipe, la hauteur, la
// decision « reapparition » et sa source, la mort et l'instantane des autres joueurs.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/observability"
)

// artefactReapparition : 100 ms par frame.
//
//	111 (equipe 0) : vie [0,100] lachee a la frame 100, puis vie [150,300] : une reapparition lue au film.
//	222 (equipe 1) : une vie [0,400], vivant a la frame 150, z = 3,456 (arrondi au centimetre).
//	333 (equipe 0) : une vie [0,400], coequipier de 111.
//	sans xuid (equipe 1) : une vie [0,400], vivante (un bot), qui compte dans l'instantane.
const artefactReapparition = `{
  "schemaVersion": 120,
  "matchId": "000d5950-1234-4abc-9def-0123456789ab",
  "frameCount": 401,
  "frameIntervalMs": 100,
  "tracks": [
    {"slot":1,"team":0,"xuid":"111","startFrame":0,"endFrame":100,"endReason":"released","releaseFrame":100,
     "points":[{"t":0,"x":1,"y":1,"z":0.5},{"t":100,"x":2,"y":2,"z":0.5}]},
    {"slot":2,"team":0,"xuid":"111","startFrame":150,"endFrame":300,
     "points":[{"t":150,"x":30.123,"y":4,"z":1.234},{"t":300,"x":31,"y":4,"z":1.234}]},
    {"slot":3,"team":1,"xuid":"222","startFrame":0,"endFrame":400,
     "points":[{"t":0,"x":50,"y":0,"z":3.456},{"t":400,"x":50,"y":0,"z":3.456}]},
    {"slot":4,"team":0,"xuid":"333","startFrame":0,"endFrame":400,
     "points":[{"t":0,"x":10,"y":10},{"t":400,"x":10,"y":10}]},
    {"slot":5,"team":1,"startFrame":0,"endFrame":400,
     "points":[{"t":0,"x":70,"y":0},{"t":400,"x":70,"y":0}]}
  ]
}`

// artefactReapparitionSansFilm : le meme match sans aucune lecture de fin de vie du film.
var artefactReapparitionSansFilm = strings.Replace(artefactReapparition, `,"endReason":"released","releaseFrame":100`, "", 1)

func TestSidecarV7_ReapparitionEtInstantane(t *testing.T) {
	s, err := ProjeterRasterTactique(context.Background(), ecrireFichier(t, "artefact.json", artefactReapparition), true)
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	if domain.TacticalRasterSchemaVersion != 7 || s.SchemaVersion != 7 {
		t.Fatalf("schema attendu 7, obtenu %d", s.SchemaVersion)
	}
	var j111 domain.TacticalRasterJoueur
	for _, j := range s.Joueurs {
		if j.XUID == "111" {
			j111 = j
		}
	}
	if j111.Equipe != 0 || len(j111.Spawns) != 2 {
		t.Fatalf("111 = equipe %d, %d spawns", j111.Equipe, len(j111.Spawns))
	}
	if sp := j111.Spawns[0]; !sp.PremiereVie || sp.Reapparition || sp.Instantane != nil {
		t.Errorf("la premiere vie n'est ni une reapparition ni porteuse d'instantane : %+v", sp)
	}
	sp := j111.Spawns[1]
	if !sp.Reapparition || sp.Source != tactical.SourceFilm || sp.Z != 1.23 {
		t.Fatalf("reapparition lue au film attendue, z arrondi a 1,23 : %+v", sp)
	}
	if sp.Mort == nil || sp.Mort.Frame != 100 || sp.Mort.X != 2 {
		t.Errorf("mort = %+v", sp.Mort)
	}
	// Instantane a la frame 150 : 222, 333 et la vie anonyme ; 111 lui-meme est exclu.
	if len(sp.Instantane) != 3 {
		t.Fatalf("instantane = %+v, attendu 3 joueurs vivants", sp.Instantane)
	}
	if v := sp.Instantane[1]; v.XUID != "222" || v.Equipe != 1 || v.Z != 3.46 {
		t.Errorf("222 dans l'instantane = %+v, attendu equipe 1 et z arrondi a 3,46", v)
	}
	if v := sp.Instantane[0]; v.XUID != "" || v.Equipe != 1 {
		t.Errorf("la vie anonyme compte, sans identite : %+v", v)
	}
}

// Un document sans aucune lecture de fin de vie du film n'a aucune reapparition : la degradation se
// compte, le reste du sidecar (positions, premieres vies) est celui du match lu au film.
func TestSidecar_SansLectureDuFilmAucuneReapparition(t *testing.T) {
	avant := observability.LoadCounterT("", CompteurRastersSansLectureFilm)
	s, err := ProjeterRasterTactique(context.Background(), ecrireFichier(t, "artefact.json", artefactReapparitionSansFilm), true)
	if err != nil {
		t.Fatalf("projection: %v", err)
	}
	for _, j := range s.Joueurs {
		for k, sp := range j.Spawns {
			if sp.Reapparition || sp.DebutDeManche || sp.Mort != nil || sp.Instantane != nil {
				t.Errorf("joueur %s, spawn %d : aucune reapparition sans lecture du film : %+v", j.XUID, k, sp)
			}
		}
	}
	if apres := observability.LoadCounterT("", CompteurRastersSansLectureFilm); apres != avant+1 {
		t.Errorf("compteur de la degradation = %d, attendu %d", apres, avant+1)
	}
	// Un titre qui ne declare pas la prevision ne projette aucune reapparition : pas de degradation a dire.
	avant = observability.LoadCounterT("", CompteurRastersSansLectureFilm)
	if _, err := ProjeterRasterTactique(context.Background(), ecrireFichier(t, "artefact2.json", artefactReapparitionSansFilm), false); err != nil {
		t.Fatal(err)
	}
	if apres := observability.LoadCounterT("", CompteurRastersSansLectureFilm); apres != avant {
		t.Errorf("sans prevision, le compteur ne bouge pas : %d -> %d", avant, apres)
	}
}

// Le sidecar est derive d'un artefact immuable : deux projections donnent les memes octets (un
// sidecar qui change d'octets sans raison serait reecrit a chaque passe de rattrapage).
func TestSidecarV7_OctetsStables(t *testing.T) {
	chemin := ecrireFichier(t, "artefact.json", artefactReapparition)
	var avant []byte
	for i := 0; i < 5; i++ {
		s, err := ProjeterRasterTactique(context.Background(), chemin, true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if avant != nil && !bytes.Equal(avant, raw) {
			t.Fatalf("octets differents a la projection %d", i)
		}
		avant = raw
	}
}

// Sans `film.spawn_forecast`, le sidecar ne porte aucune matiere de prevision : ni reapparition,
// ni source, ni debut de manche, ni mort, ni instantane, ni couverture. Le reste (cellules,
// positions du spawn, premiere vie, equipe) est celui du sidecar d'un titre qui la declare.
func TestSidecar_SansPrevisionNePorteAucuneMatiereDeReapparition(t *testing.T) {
	chemin := ecrireFichier(t, "artefact.json", artefactReapparition)
	avec, err := ProjeterRasterTactique(context.Background(), chemin, true)
	if err != nil {
		t.Fatal(err)
	}
	sans, err := ProjeterRasterTactique(context.Background(), chemin, false)
	if err != nil {
		t.Fatal(err)
	}
	porte := false
	for _, j := range avec.Joueurs {
		for _, sp := range j.Spawns {
			porte = porte || sp.Reapparition
		}
	}
	if !porte {
		t.Fatal("le jeu d'essai doit porter une reapparition avec la prevision")
	}
	for i, j := range sans.Joueurs {
		if j.Equipe != avec.Joueurs[i].Equipe || len(j.Cellules) != len(avec.Joueurs[i].Cellules) ||
			len(j.Spawns) != len(avec.Joueurs[i].Spawns) {
			t.Fatalf("joueur %s : le reste du sidecar doit etre inchange", j.XUID)
		}
		for k, sp := range j.Spawns {
			ref := avec.Joueurs[i].Spawns[k]
			if sp.Reapparition || sp.Source != "" || sp.DebutDeManche || sp.Mort != nil || sp.Instantane != nil {
				t.Errorf("joueur %s, spawn %d : matiere de prevision sans capability : %+v", j.XUID, k, sp)
			}
			if sp.Frame != ref.Frame || sp.X != ref.X || sp.Y != ref.Y || sp.Z != ref.Z || sp.PremiereVie != ref.PremiereVie {
				t.Errorf("joueur %s, spawn %d : position ou premiere vie changees : %+v contre %+v", j.XUID, k, sp, ref)
			}
		}
	}
}

// La porte du fil de l'eau : le titre qui declare la capability recoit la matiere de prevision, le
// titre qui declare le rejeu sans la prevision ne la recoit pas.
func TestProjeterRastersTactiques_PorteDeLaPrevision(t *testing.T) {
	for _, c := range []struct {
		nom      string
		sansCle  bool
		attendue bool
	}{{"avec la capability", false, true}, {"sans la capability", true, false}} {
		t.Run(c.nom, func(t *testing.T) {
			racine := racineAvecTitres(t, titlePkg.DefaultSlug)
			if c.sansCle {
				retirerCapability(t, racine, string(games.CapFilmSpawnForecast))
			}
			chemin := ecrireFichier(t, "artefact.json", artefactReapparition)
			doc, octets, err := lireDocumentRange(chemin)
			if err != nil {
				t.Fatal(err)
			}
			d := Deps{RepoRoot: racine, TitleSlug: titlePkg.DefaultSlug, Gamertag: "TestGT"}
			projeterRastersTactiques(context.Background(), d, []artefactLu{
				{matchID: doc.MatchID, path: chemin, doc: doc, octets: octets},
			})
			raw, err := os.ReadFile(titlePkg.NewPathResolver(racine).TacticalRasterPath(titlePkg.DefaultSlug, titlePkg.FilmShortMatchID(doc.MatchID)))
			if err != nil {
				t.Fatalf("sidecar absent : %v", err)
			}
			var s domain.TacticalRasterSidecar
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatal(err)
			}
			porte := false
			for _, j := range s.Joueurs {
				for _, sp := range j.Spawns {
					porte = porte || sp.Reapparition
				}
			}
			if porte != c.attendue {
				t.Errorf("matiere de prevision presente = %v, attendue %v", porte, c.attendue)
			}
		})
	}
}

// retirerCapability supprime une cle de capability du capabilities.toml d'une racine de test.
func retirerCapability(t *testing.T, racine, cle string) {
	t.Helper()
	chemin := filepath.Join(racine, "config", "titles", titlePkg.DefaultSlug, "mappings", "capabilities.toml")
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	var gardees []string
	retirees := 0
	for _, ligne := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(ligne), `"`+cle+`"`) {
			retirees++
			continue
		}
		gardees = append(gardees, ligne)
	}
	if retirees != 1 {
		t.Fatalf("la cle %s doit apparaitre une fois, trouvee %d fois", cle, retirees)
	}
	if err := os.WriteFile(chemin, []byte(strings.Join(gardees, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Des capabilities illisibles ne produisent AUCUN sidecar : un sidecar ecrit sans la matiere de
// prevision porterait le schema courant et ne serait jamais reecrit par le rattrapage.
func TestProjeterRastersTactiques_CapabilitiesIllisiblesNEcriventRien(t *testing.T) {
	racine := racineAvecTitres(t, titlePkg.DefaultSlug)
	chemin := filepath.Join(racine, "config", "titles", titlePkg.DefaultSlug, "mappings", "capabilities.toml")
	if err := os.WriteFile(chemin, []byte("[[ce n'est pas du toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	artefact := ecrireFichier(t, "artefact.json", artefactReapparition)
	doc, octets, err := lireDocumentRange(artefact)
	if err != nil {
		t.Fatal(err)
	}
	d := Deps{RepoRoot: racine, TitleSlug: titlePkg.DefaultSlug, Gamertag: "TestGT"}
	projeterRastersTactiques(context.Background(), d, []artefactLu{
		{matchID: doc.MatchID, path: artefact, doc: doc, octets: octets},
	})
	sidecar := titlePkg.NewPathResolver(racine).TacticalRasterPath(titlePkg.DefaultSlug, titlePkg.FilmShortMatchID(doc.MatchID))
	if _, err := os.Stat(sidecar); !os.IsNotExist(err) {
		t.Fatalf("aucun sidecar attendu quand les capabilities sont illisibles, stat = %v", err)
	}
}
