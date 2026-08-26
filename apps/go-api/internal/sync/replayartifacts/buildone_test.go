package replayartifacts

// buildone_test.go — LE CONTRAT DE LA CUISSON DELEGUEE (lot BUILDALL, 2026-08-26).
//
// CE QUE CES CAS PROTEGENT, ET C'EST LE COEUR DU LOT :
//   - un echec de l'enfant — y compris une MORT SUBITE — n'ecrit RIEN et ne fait pas tomber le
//     cycle ; le film suivant ne doit rien devoir a la sante du precedent ;
//   - sans strategie cablee, on NE DECODE PAS in-process : on saute, et on le dit ;
//   - la requete transmise a l'enfant porte tout ce qu'il faut pour qu'il n'ouvre AUCUNE base.
//
// AUCUN FILM N'EST DECODE ICI : la strategie est injectee. Un test qui decoderait vraiment
// serait le comportement que ce lot existe pour supprimer.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/port"
)

// deps monte des dependances minimales : ni base, ni cache, ni reseau.
func deps(build BuildOneFunc) Deps {
	return Deps{RepoRoot: t0RepoRoot, TitleSlug: "halo_infinite", BuildOne: build}
}

const t0RepoRoot = "/depot"

func work() buildWork {
	return buildWork{
		matchID:  "64e8adfa-0000-0000-0000-000000000000",
		mapNames: []string{"Catalyst"},
		facts: port.MatchFacts{
			GameVariantName: "CTF:Arena", MapID: "f7e8cde9",
			Players: []port.MatchPlayerFact{{XUID: "2533274819954312"}},
		},
	}
}

// SANS STRATEGIE CABLEE, ON NE DECODE PAS. C'est la regression que le lot supprime : avant lui,
// « pas de delegation » n'existait pas — le serveur decodait, point.
func TestBuildAndStoreOneSansStrategieNeDecodePas(t *testing.T) {
	_, err := buildAndStoreOne(context.Background(), deps(nil), work(), "/films/64e8adfa")
	if !errors.Is(err, ErrNoBuilder) {
		t.Fatalf("err = %v, attendu ErrNoBuilder — un serveur sans strategie doit SAUTER, "+
			"jamais retomber sur un decodage in-process", err)
	}
}

// LA REQUETE PORTE TOUT CE QU'IL FAUT A L'ENFANT, et surtout de quoi ne PAS ouvrir de base :
// le slug et la racine du depot voyagent avec elle (il monte son propre constructeur), et les
// faits sont deja lus par le parent.
func TestBuildAndStoreOneTransmetLaRequeteComplete(t *testing.T) {
	var vu BuildOneRequest
	_, _ = buildAndStoreOne(context.Background(), deps(func(_ context.Context, r BuildOneRequest) ([]byte, error) {
		vu = r
		return nil, errors.New("stop : la requete suffit a ce cas")
	}), work(), "/films/64e8adfa")

	if vu.TitleSlug != "halo_infinite" || vu.RepoRoot != t0RepoRoot {
		t.Errorf("slug=%q racine=%q — l'enfant ne pourrait pas monter son constructeur",
			vu.TitleSlug, vu.RepoRoot)
	}
	if vu.FilmDir != "/films/64e8adfa" || len(vu.MapNames) != 1 {
		t.Errorf("filmDir=%q mapNames=%v", vu.FilmDir, vu.MapNames)
	}
	// LES FAITS SONT DEJA LUS : c'est ce qui dispense l'enfant de toute base (mono-processus
	// DuckDB, ADR 0013/0016).
	if vu.Facts.GameVariantName != "CTF:Arena" || len(vu.Facts.Players) != 1 {
		t.Errorf("faits transmis incomplets : %+v", vu.Facts)
	}
}

// UN ECHEC DE L'ENFANT N'ECRIT RIEN. La racine du depot est bidon : si le code tentait
// d'ecrire malgre l'erreur, il echouerait sur le chemin — ce cas verifie qu'on n'y arrive
// meme pas.
func TestBuildAndStoreOneEchecNEcritRien(t *testing.T) {
	sentinelle := errors.New("cuisson en echec (issue mort subite, code 137)")
	_, err := buildAndStoreOne(context.Background(), deps(func(context.Context, BuildOneRequest) ([]byte, error) {
		return nil, sentinelle
	}), work(), "/films/64e8adfa")
	if !errors.Is(err, sentinelle) {
		t.Fatalf("err = %v, attendu l'echec de l'enfant tel quel", err)
	}
}

// LA MORT SUBITE REMONTE COMME UN ECHEC ORDINAIRE, et c'est voulu : l'appelant journalise et
// passe au film suivant. Un enfant tue par l'OS ne doit ni faire tomber le cycle, ni passer
// pour un succes.
func TestBuildAndStoreOneMortSubiteNeCasseriPasLeCycle(t *testing.T) {
	appels := 0
	d := deps(func(context.Context, BuildOneRequest) ([]byte, error) {
		appels++
		return nil, errors.New("cuisson en echec (issue mort subite, code -1)")
	})
	for i := 0; i < 3; i++ {
		if _, err := buildAndStoreOne(context.Background(), d, work(), "/films/x"); err == nil {
			t.Fatal("une mort subite doit rendre une erreur, jamais un succes")
		}
	}
	if appels != 3 {
		t.Errorf("%d appel(s), attendu 3 — chaque film est tente independamment du precedent", appels)
	}
}
