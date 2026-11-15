package replayartifacts

// raster_reapparitions_test.go — LA PART « REAPPARITION » DU SIDECAR (schema 7) : l'equipe, la
// hauteur, la decision « reapparition » et sa source, la mort et l'instantane des autres joueurs.

import (
	"bytes"
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain"
)

// artefactReapparition : 100 ms par frame.
//
//	111 (equipe 0) : vie [0,100] puis vie [150,300] — 5 s d'ecart, donc une reapparition.
//	222 (equipe 1) : une vie [0,400], vivant a la frame 150, z = 3,456 (arrondi au centimetre).
//	333 (equipe 0) : une vie [0,400], coequipier de 111.
//	sans xuid (equipe 1) : une vie [0,400], vivante (un bot), qui compte dans l'instantane.
const artefactReapparition = `{
  "schemaVersion": 120,
  "matchId": "000d5950-1234-4abc-9def-0123456789ab",
  "frameCount": 401,
  "frameIntervalMs": 100,
  "tracks": [
    {"slot":1,"team":0,"xuid":"111","startFrame":0,"endFrame":100,
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

func TestSidecarV7_ReapparitionEtInstantane(t *testing.T) {
	s, err := ProjeterRasterTactique(ecrireFichier(t, "artefact.json", artefactReapparition))
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
	if !sp.Reapparition || sp.Source != tactical.SourceRepliEcart || sp.Z != 1.23 {
		t.Fatalf("reapparition de repli attendue, z arrondi a 1,23 : %+v", sp)
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
	if s.Reapparitions.RepliEcart != 1 || s.Reapparitions.Film != 0 || s.Reapparitions.DebutsDeManche != 0 {
		t.Errorf("sources = %+v", s.Reapparitions)
	}
}

// Le sidecar est derive d'un artefact immuable : deux projections donnent les memes octets (un
// sidecar qui change d'octets sans raison serait reecrit a chaque passe de rattrapage).
func TestSidecarV7_OctetsStables(t *testing.T) {
	chemin := ecrireFichier(t, "artefact.json", artefactReapparition)
	var avant []byte
	for i := 0; i < 5; i++ {
		s, err := ProjeterRasterTactique(chemin)
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

func TestSourcesDesReapparitions(t *testing.T) {
	c := sourcesDesReapparitions([]domain.TacticalRasterJoueur{{Spawns: []domain.TacticalRasterSpawn{
		{Reapparition: true, Source: tactical.SourceFilm},
		{Reapparition: true, Source: tactical.SourceRepliEcart},
		{DebutDeManche: true},
		{PremiereVie: true},
	}}})
	if c.Film != 1 || c.RepliEcart != 1 || c.DebutsDeManche != 1 {
		t.Errorf("sources = %+v", c)
	}
}
