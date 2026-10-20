//go:build integration

package duckdb

import (
	"context"
	"testing"
	"time"
)

// TestHomeRepo_LoadSpartanIdentity_UneConnexion : la base joueur de production n'a qu'UNE
// connexion (poolSingleConn). LoadSpartanIdentity enchaîne plusieurs lectures sur cette base
// (identité, puis pics CSR et LUSR) : un curseur laissé ouvert garde la connexion, et la
// lecture suivante attend jusqu'à l'échéance du contexte. L'Explorer lit ainsi l'identité de
// chaque joueur suivi sous un budget de 8 s : la requête entière durait 8 s et les pics
// manquaient. Le test pose la même contrainte et exige les pics, bien avant l'échéance.
func TestHomeRepo_LoadSpartanIdentity_UneConnexion(t *testing.T) {
	pdb := newTestPlayerDB(t)
	seedMaturedSkillRankGroups(t, pdb)
	pdb.Player.SQLDb().SetMaxOpenConns(poolSingleConn)

	const echeance = 3 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), echeance)
	defer cancel()
	debut := time.Now()
	identity, err := NewHomeRepo(pdb).LoadSpartanIdentity(ctx)
	duree := time.Since(debut)
	if err != nil {
		t.Fatalf("LoadSpartanIdentity: %v", err)
	}
	if identity == nil || identity.HighestCSR == nil || identity.HighestLUSR == nil {
		t.Fatalf("identité = %+v, want les pics CSR et LUSR (lus sur la même connexion)", identity)
	}
	if duree >= echeance/2 {
		t.Fatalf("LoadSpartanIdentity a duré %s : une lecture a attendu la connexion jusqu'à l'échéance", duree)
	}
}
