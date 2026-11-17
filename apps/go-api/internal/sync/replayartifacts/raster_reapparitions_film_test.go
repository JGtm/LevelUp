package replayartifacts

// raster_reapparitions_film_test.go — LES LECTURES DE FIN DE VIE DU FILM dans l'entree de
// l'occupation : fin de piste (`endReason`, `releaseFrame`) et comptes a rebours rattaches a un
// xuid par l'identite des joueurs.

import (
	"encoding/json"
	"testing"

	"levelup/go-api/internal/analysis/tactical"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// documentLuAuFilm : 100 ms par frame.
//
//	111 : vie [0,100] lachee a la frame 101, puis vie [130,300] (3 s : le repli la refuserait).
//	222 : vie [0,100] supprimee, puis vie [160,300] (6 s : le repli la prendrait).
//	333 : vie [0,100] sans fin lue, compte a rebours qui commence a la frame 101, puis vie [200,300].
//	444 : vie [0,100] lachee a la frame 50, hors tolerance, puis vie [200,300].
//	Un compte a rebours du bot (index 7, sans xuid) et un d'un index inconnu (9).
const documentLuAuFilm = `{
  "schemaVersion": 124,
  "matchId": "000d5950-1234-4abc-9def-0123456789ab",
  "frameIntervalMs": 100,
  "tracks": [
    {"slot":1,"team":0,"xuid":"111","startFrame":0,"endFrame":100,"endReason":"released","releaseFrame":101,
     "points":[{"t":0,"x":1,"y":1},{"t":100,"x":2,"y":2}]},
    {"slot":1,"team":0,"xuid":"111","startFrame":130,"endFrame":300,
     "points":[{"t":130,"x":5,"y":5},{"t":300,"x":6,"y":6}]},
    {"slot":2,"team":1,"xuid":"222","startFrame":0,"endFrame":100,"endReason":"deleted",
     "points":[{"t":0,"x":10,"y":1},{"t":100,"x":11,"y":1}]},
    {"slot":2,"team":1,"xuid":"222","startFrame":160,"endFrame":300,
     "points":[{"t":160,"x":12,"y":1},{"t":300,"x":13,"y":1}]},
    {"slot":3,"team":0,"xuid":"333","startFrame":0,"endFrame":100,
     "points":[{"t":0,"x":20,"y":1},{"t":100,"x":21,"y":1}]},
    {"slot":3,"team":0,"xuid":"333","startFrame":200,"endFrame":300,
     "points":[{"t":200,"x":22,"y":1},{"t":300,"x":23,"y":1}]},
    {"slot":4,"team":1,"xuid":"444","startFrame":0,"endFrame":100,"endReason":"released","releaseFrame":50,
     "points":[{"t":0,"x":30,"y":1},{"t":100,"x":31,"y":1}]},
    {"slot":4,"team":1,"xuid":"444","startFrame":200,"endFrame":300,
     "points":[{"t":200,"x":32,"y":1},{"t":300,"x":33,"y":1}]}
  ],
  "respawnCountdowns": [
    {"filmIndex":2,"from":101,"to":151,"seconds":5,"startRead":true,"endRead":true},
    {"filmIndex":7,"from":40,"to":90,"seconds":5,"startRead":true,"endRead":true},
    {"filmIndex":9,"from":40,"to":90,"seconds":5,"startRead":true,"endRead":true}
  ],
  "identity": {"players": [
    {"filmIndex":0,"xuid":"111"},{"filmIndex":1,"xuid":"222"},{"filmIndex":2,"xuid":"333"},
    {"filmIndex":3,"xuid":"444"},{"filmIndex":7,"bid":"bid(1.0)"},{"filmIndex":-1,"xuid":"555"}
  ]}
}`

func lireDocumentDeTest(t *testing.T, brut string) *replay.ReplayDocument {
	t.Helper()
	var doc replay.ReplayDocument
	if err := json.Unmarshal([]byte(brut), &doc); err != nil {
		t.Fatalf("document de test illisible : %v", err)
	}
	return &doc
}

func TestEntreeOccupationDeLArtefact_LecturesDuFilm(t *testing.T) {
	e := EntreeOccupationDeLArtefact(lireDocumentDeTest(t, documentLuAuFilm))
	if f := e.Pistes[0].Fin; f == nil || f.Raison != tactical.RaisonLiberee || f.Frame != 101 {
		t.Errorf("fin de la premiere vie de 111 = %+v, attendu liberee a la frame 101", f)
	}
	if f := e.Pistes[2].Fin; f == nil || f.Raison != tactical.RaisonSupprimee || f.Frame != 100 {
		t.Errorf("fin de la premiere vie de 222 = %+v, attendu supprimee a la fin de la piste", f)
	}
	if e.Pistes[1].Fin != nil || e.Pistes[4].Fin != nil {
		t.Errorf("une piste sans fin lue ne porte aucune fin : %+v, %+v", e.Pistes[1].Fin, e.Pistes[4].Fin)
	}
	if len(e.DebutsDeCompte) != 1 || e.DebutsDeCompte[0] != (tactical.DebutCompte{XUID: "333", Frame: 101}) {
		t.Errorf("debuts de compte = %+v, attendu le seul compte de 333 a la frame 101", e.DebutsDeCompte)
	}
	if e.ComptesSansJoueur != 2 {
		t.Errorf("comptes sans joueur = %d, attendu 2 (le bot et l'index inconnu)", e.ComptesSansJoueur)
	}
}

func TestEntreeOccupationDeLArtefact_DecisionsAuFilm(t *testing.T) {
	e := EntreeOccupationDeLArtefact(lireDocumentDeTest(t, documentLuAuFilm))
	spawns := tactical.NouvelIndexVies(e).Spawns()
	cas := []struct {
		piste        int
		reapparition bool
		manche       bool
		quoi         string
	}{
		{1, true, false, "111 : lacher de commande a la fin de la piste"},
		{3, false, true, "222 : corps supprime, debut de manche"},
		{5, true, false, "333 : compte a rebours qui commence a la fin de la piste"},
		{7, false, false, "444 : lacher hors tolerance, aucune preuve"},
	}
	for _, c := range cas {
		s := spawns[c.piste]
		if s.Reapparition != c.reapparition || s.DebutDeManche != c.manche {
			t.Errorf("%s : reapparition=%v manche=%v", c.quoi, s.Reapparition, s.DebutDeManche)
		}
	}
}

// Un document sans aucune lecture du film n'a aucune reapparition : l'entree n'en porte aucune.
func TestEntreeOccupationDeLArtefact_SansLectureAucuneReapparition(t *testing.T) {
	e := EntreeOccupationDeLArtefact(lireDocumentDeTest(t, artefactReapparitionSansFilm))
	if len(e.DebutsDeCompte) != 0 || e.ComptesSansJoueur != 0 {
		t.Errorf("aucun compte attendu : %+v, %d", e.DebutsDeCompte, e.ComptesSansJoueur)
	}
	for i, p := range e.Pistes {
		if p.Fin != nil {
			t.Errorf("piste %d : fin %+v, attendu aucune", i, p.Fin)
		}
	}
	if tactical.LectureDuFilm(e) {
		t.Error("l'entree ne porte aucune lecture du film")
	}
	if s := tactical.NouvelIndexVies(e).Spawns()[1]; s.Reapparition {
		t.Errorf("aucune reapparition attendue sans lecture du film : %+v", s)
	}
}

func TestFinDeLaPiste_LacherSansFrameNeDitRien(t *testing.T) {
	if f := finDeLaPiste(replay.Track{EndReason: replay.EndReasonReleased}); f != nil {
		t.Errorf("lacher sans frame = %+v, attendu aucune fin", f)
	}
	if f := finDeLaPiste(replay.Track{EndReason: "autre"}); f != nil {
		t.Errorf("raison inconnue = %+v, attendu aucune fin", f)
	}
	sansFenetre := replay.Track{EndReason: replay.EndReasonDeleted, Points: []replay.Point{{T: 10}, {T: 42}}}
	if f := finDeLaPiste(sansFenetre); f == nil || f.Frame != 42 {
		t.Errorf("suppression sans fenetre declaree = %+v, attendu la frame du dernier point", f)
	}
}
