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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
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
	_, _ = buildAndStoreOne(context.Background(), deps(func(_ context.Context, r BuildOneRequest) (BuildOneResult, error) {
		vu = r
		return BuildOneResult{}, errors.New("stop : la requete suffit a ce cas")
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
	_, err := buildAndStoreOne(context.Background(), deps(func(context.Context, BuildOneRequest) (BuildOneResult, error) {
		return BuildOneResult{}, sentinelle
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
	d := deps(func(context.Context, BuildOneRequest) (BuildOneResult, error) {
		appels++
		return BuildOneResult{}, errors.New("cuisson en echec (issue mort subite, code -1)")
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

// LA MESURE DE LA CUISSON TRAVERSE LA FRONTIERE (PLAN_CUISSON_PERF §3 D5). Duree et pic
// memoire sont mesures par le lanceur de l'enfant ; s'ils s'arretaient a `buildAndStoreOne`, le
// log de succes du cycle ne pourrait toujours rien dire de ce qu'un film a coute. Ce cas verifie
// que ce sont EXACTEMENT les valeurs rendues par la strategie qui remontent — pas une mesure
// refaite ici, qui mesurerait le test.
func TestBuildAndStoreOneRemonteLaMesure(t *testing.T) {
	repoRoot := t.TempDir()
	w := work()
	// UNE TRAJECTOIRE AU MOINS : `StoreArtifact` refuse un document vide (il se servirait comme
	// le rejeu « propre » d'un match sans joueur).
	blob, err := json.Marshal(replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: w.matchID,
		Tracks: []replay.Track{{Slot: 1, Team: -1}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	const dureeAttendue = 1234 * time.Millisecond
	const picAttendu = uint64(987_654_321)

	d := Deps{RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug,
		BuildOne: func(context.Context, BuildOneRequest) (BuildOneResult, error) {
			return BuildOneResult{Blob: blob, Dur: dureeAttendue, Peak: picAttendu}, nil
		}}

	out, err := buildAndStoreOne(context.Background(), d, w, "/films/64e8adfa")
	if err != nil {
		t.Fatalf("buildAndStoreOne: %v", err)
	}
	if out.dur != dureeAttendue {
		t.Errorf("duree = %v, attendu %v", out.dur, dureeAttendue)
	}
	if out.peak != picAttendu {
		t.Errorf("pic = %d, attendu %d", out.peak, picAttendu)
	}
	// L'ARTEFACT EST BIEN RANGE : la mesure accompagne un rangement reel, elle ne le remplace pas.
	if out.stored.Path == "" || out.stored.Bytes != len(blob) {
		t.Errorf("artefact range = %+v, attendu %d octets a une place canonique", out.stored, len(blob))
	}
}
