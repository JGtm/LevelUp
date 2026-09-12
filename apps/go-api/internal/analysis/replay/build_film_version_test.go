package replay

// build_film_version_test.go — LA VERSION DU FILM ARRIVE DANS LA COUVERTURE.
//
// Revue adversariale du 2026-09-12, constat P2-5 : `Coverage.FilmMajorVersion` etait publie
// (contrat OpenAPI, miroir `replaydoc`, projection `replayview`) sans qu AUCUN test n en asserte
// la valeur. Un champ de telemetrie qui ne dit pas la verite est pire que pas de champ : la mesure
// par version du lot H s appuierait dessus.
//
// DEUX CAS, ET LE SECOND COMPTE AUTANT QUE LE PREMIER. Le champ est un POINTEUR : son absence dit
// « film sans registre, ou artefact anterieur au lot G » et ne doit jamais se lire « version 0 ».

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// positionsPourVersion : le plus petit jeu de positions qui produise un document assemble (deux
// echantillons d un meme slot, l origine etant le premier paquet).
func positionsPourVersion() []filmdec.BipedPosition {
	const slot uint32 = 512
	return []filmdec.BipedPosition{
		pos(slot, 10_000, 1, 1, 0.5),
		pos(slot, 10_100, 2, 2, 0.5),
	}
}

// TestCoverageFilmMajorVersionPubliee : la version passee en option se retrouve dans la couverture.
func TestCoverageFilmMajorVersionPubliee(t *testing.T) {
	version := 40
	doc := BuildFromPositions("m", "halo_infinite", positionsPourVersion(), nil, Options{
		FrameIntervalMS:  100,
		FilmMajorVersion: &version,
	})
	if doc.Coverage.FilmMajorVersion == nil {
		t.Fatal("Coverage.FilmMajorVersion est nil alors que l option porte la version 40 : " +
			"l artefact ne dit plus sous quelle grammaire il a ete cuit (build.go)")
	}
	if got := *doc.Coverage.FilmMajorVersion; got != version {
		t.Fatalf("Coverage.FilmMajorVersion = %d, %d attendu", got, version)
	}
}

// TestCoverageFilmMajorVersionAbsente : sans option, le champ reste nil — jamais 0.
//
// 0 EST UNE VALEUR SIGNIFIANTE AILLEURS (`filmdec.FilmMajorVersionUnknown`, le decoupage
// historique du gamertag) : la publier ici ferait passer « on ne sait pas » pour « version 0 ».
func TestCoverageFilmMajorVersionAbsente(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", positionsPourVersion(), nil, Options{
		FrameIntervalMS: 100,
	})
	if doc.Coverage.FilmMajorVersion != nil {
		t.Fatalf("Coverage.FilmMajorVersion = %d alors qu aucune version n a ete lue : nil est le "+
			"seul rendu correct de l inconnu (cf. filmdec.FilmMajorVersionUnknown = %d)",
			*doc.Coverage.FilmMajorVersion, filmdec.FilmMajorVersionUnknown)
	}
}
