//go:build cgo

// Package duckdb — flag_grabs_net_repo_test.go : la lecture des PRISES NETTES de drapeau.
//
// Deux propriétés à verrouiller, et la seconde est la raison d'être de `decode_pass` :
//
//   - LA LECTURE PASSE PAR LA VUE `_latest`, et cette vue rend la DERNIÈRE PASSE ENTIÈRE par
//     match. Un joueur que la nouvelle passe ne nomme plus doit DISPARAÎTRE, et non survivre
//     avec l'ANCIENNE fenêtre de jonglage à côté des nouvelles — ce qui donnerait un scope à
//     deux fenêtres, donc un scope qui n'annonce plus aucune règle à l'écran.
//   - LA DÉGRADATION : vue absente (DB non migrée, titre qui ne produit pas la grandeur) ⇒
//     nil + nil, JAMAIS d'échec dur d'une page.
package duckdb

import (
	"context"
	"database/sql"

	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
)

// passePrisesNettes décrit UNE passe de lecture d'artefact, dans la forme que le persister
// écrit.
//
// ⚠ L'INSERT EST RECOPIÉ ICI, ET C'EST UNE CONTRAINTE, PAS UN CHOIX : `internal/persist`
// importe ce paquet, donc un test d'ici ne peut pas appeler `persist.FlagGrabsNetPersister`
// sans cycle d'import. Ce que ce fichier juge est la LECTURE (la vue `_latest` et sa
// rétractation par passe) ; la forme des ÉCRITURES est verrouillée, elle, par
// `persist/flag_grabs_net_persister_test.go`, qui passe par le vrai persister. Le seul risque
// résiduel est qu'une colonne soit ajoutée sans venir ici — et la vue étant un `SELECT *`,
// elle la servirait quand même.
type passePrisesNettes struct {
	matchID  string
	pass     string
	windowMS int
	openings int
	joueurs  map[string][2]int // xuid -> {brut, net}
}

func ecrirePassePrisesNettes(t *testing.T, db *sql.DB, p passePrisesNettes) {
	t.Helper()
	for xuid, v := range p.joueurs {
		_, err := db.Exec(`INSERT INTO match_flag_grabs_net
			(match_id, decode_pass, xuid, flag_grabs_raw, flag_grabs_net, openings, juggle_window_ms)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			p.matchID, p.pass, xuid, v[0], v[1], p.openings, p.windowMS)
		if err != nil {
			t.Fatalf("écriture de la passe %s/%s: %v", p.matchID, xuid, err)
		}
	}

}

// TestLoadFlagGrabsNet_LitLaDernierePasseEntiere : le chemin nominal ET la rétractation.
func TestLoadFlagGrabsNet_LitLaDernierePasseEntiere(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	ctx := context.Background()

	ecrirePassePrisesNettes(t, db, passePrisesNettes{
		matchID: "m1", pass: "p1", windowMS: 1000, openings: 30,
		joueurs: map[string][2]int{"P": {9, 6}, "A": {4, 4}},
	})
	ecrirePassePrisesNettes(t, db, passePrisesNettes{
		matchID: "m2", pass: "p1", windowMS: 1500, openings: 12,
		joueurs: map[string][2]int{"P": {5, 5}},
	})

	rows, err := repo.LoadFlagGrabsNet(ctx, []string{"m1", "m2"})
	if err != nil {
		t.Fatalf("LoadFlagGrabsNet: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("%d lignes, attendu 3", len(rows))
	}
	vu := map[string]sessionRow{}
	for _, r := range rows {
		vu[r.MatchID+"/"+r.XUID] = sessionRow{r.Raw, r.Net, r.Openings, r.WindowMS}
	}
	if got := vu["m1/P"]; got != (sessionRow{9, 6, 30, 1000}) {
		t.Errorf("m1/P = %+v, attendu {9 6 30 1000}", got)
	}
	if got := vu["m2/P"]; got != (sessionRow{5, 5, 12, 1500}) {
		t.Errorf("m2/P = %+v, attendu {5 5 12 1500}", got)
	}

	// UNE SECONDE PASSE SUR m1, SOUS UNE AUTRE FENÊTRE, QUI NE NOMME PLUS `A`.
	ecrirePassePrisesNettes(t, db, passePrisesNettes{
		matchID: "m1", pass: "p2", windowMS: 1500, openings: 30,
		joueurs: map[string][2]int{"P": {9, 4}},
	})
	rows, err = repo.LoadFlagGrabsNet(ctx, []string{"m1"})
	if err != nil {
		t.Fatalf("LoadFlagGrabsNet (2e passe): %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("%d lignes après la 2e passe, attendu 1 — la passe précédente fuit dans la vue", len(rows))
	}
	if rows[0].XUID != "P" || rows[0].Net != 4 || rows[0].WindowMS != 1500 {
		t.Errorf("ligne servie = %+v, attendu P net=4 fenêtre=1500", rows[0])
	}
}

// sessionRow : un tuple comparable, pour lire les assertions d'un coup d'œil.
type sessionRow struct{ raw, net, openings, window int }

// TestLoadFlagGrabsNet_ScopeFerme : le filtre match_id est respecté — une page ne lit que son
// scope.
func TestLoadFlagGrabsNet_ScopeFerme(t *testing.T) {
	repo, db := newObjectiveStatsRepoOnMem(t, true)
	ecrirePassePrisesNettes(t, db, passePrisesNettes{
		matchID: "dedans", pass: "p1", windowMS: 1500, openings: 4,
		joueurs: map[string][2]int{"P": {2, 2}},
	})
	ecrirePassePrisesNettes(t, db, passePrisesNettes{
		matchID: "dehors", pass: "p1", windowMS: 1500, openings: 4,
		joueurs: map[string][2]int{"P": {7, 7}},
	})
	rows, err := repo.LoadFlagGrabsNet(context.Background(), []string{"dedans"})
	if err != nil {
		t.Fatalf("LoadFlagGrabsNet: %v", err)
	}
	if len(rows) != 1 || rows[0].MatchID != "dedans" {
		t.Errorf("lignes = %+v, attendu le seul match du scope", rows)
	}
}

// TestLoadFlagGrabsNet_ScopeVideNeLitRien : aucune requête sur un scope vide.
func TestLoadFlagGrabsNet_ScopeVideNeLitRien(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, true)
	rows, err := repo.LoadFlagGrabsNet(context.Background(), nil)
	if err != nil || rows != nil {
		t.Errorf("scope vide = (%v, %v), attendu (nil, nil)", rows, err)
	}
}

// TestLoadFlagGrabsNet_VueAbsenteDegrade : une DB non migrée (ou un titre qui ne produit pas
// la grandeur) rend nil + nil — la page perd cette grandeur, jamais tout le reste.
func TestLoadFlagGrabsNet_VueAbsenteDegrade(t *testing.T) {
	repo, _ := newObjectiveStatsRepoOnMem(t, false)
	rows, err := repo.LoadFlagGrabsNet(context.Background(), []string{"m1"})
	if err != nil {
		t.Errorf("erreur dure sur une vue absente : %v — la dégradation doit être silencieuse", err)
	}
	if rows != nil {
		t.Errorf("lignes = %+v, attendu nil", rows)
	}
}
