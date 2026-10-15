package wire

// registry_replay_build_test.go — L'ACTION ADMIN « CONSTRUIRE LE REJEU » NE DECODE PLUS DANS LE
// SERVEUR (lot J2.13, constat OPS-2, 2026-09-26).
//
// Elle appelait `replaybuild.NewBuilder` puis `BuildMatch` DANS le processus serveur : hors du
// verrou solo, sans sentinelle memoire — le septieme point d'entree du decodage, celui que l'ADR
// 0034 ne comptait pas. Elle passe desormais par le chemin de l'etape 1.58 : une STRATEGIE de
// construction hors processus (en production `replayartifacts.SpawnBuildOne`, l'enfant borne),
// puis `replaybuild.StoreArtifact` chez le parent.

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/sync/replayartifacts"
)

func TestRunReplayBuild_PasseParLaStrategieEnfant(t *testing.T) {
	demande := replayartifacts.BuildOneRequest{
		MatchID: "0123456789abcdef", TitleSlug: title.DefaultSlug, RepoRoot: t.TempDir(),
		MapNames: []string{"Live Fire"}, FilmDir: "chunks/01234567",
		Facts: port.MatchFacts{GameVariantName: "Slayer"},
	}
	var recues []replayartifacts.BuildOneRequest
	strategie := func(_ context.Context, req replayartifacts.BuildOneRequest) (replayartifacts.BuildOneResult, error) {
		recues = append(recues, req)
		// DES OCTETS QUE `StoreArtifact` REFUSE : s'ils lui parviennent, l'erreur le dit.
		return replayartifacts.BuildOneResult{Blob: []byte("{}")}, nil
	}
	_, err := construireRejeuAdmin(context.Background(), strategie, demande)
	if len(recues) != 1 {
		t.Fatalf("la strategie de construction a ete appelee %d fois, attendu 1 — l'action a "+
			"decode dans le processus serveur (erreur : %v)", len(recues), err)
	}
	if recues[0].MatchID != demande.MatchID || recues[0].FilmDir != demande.FilmDir ||
		recues[0].RepoRoot != demande.RepoRoot || recues[0].Facts.GameVariantName != "Slayer" {
		t.Errorf("requete transmise %+v, attendu %+v", recues[0], demande)
	}
	if !errors.Is(err, domain.ErrBuildArtifactInvalid) {
		t.Errorf("erreur %v : les octets de l'enfant doivent etre RANGES par StoreArtifact, qui "+
			"refuse ce blob (ErrBuildArtifactInvalid)", err)
	}
}

// TestReplayMatchIdentity_RefuseLeModeNonGere — ni l'action admin ni la mise en file ne
// fabriquent un rejeu de Firefight (decision du 2026-10-15) : l'identite du match, point commun
// des deux, refuse avec la sentinelle ; un match gere se resout.
func TestReplayMatchIdentity_RefuseLeModeNonGere(t *testing.T) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, q := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, map_name VARCHAR, map_id VARCHAR, is_firefight BOOLEAN)`,
		`INSERT INTO match_registry VALUES
			('edc5daf6-0df8-4eb0-b578-b8eb892ce184', 'Cole Protocol', 'b', TRUE),
			('39910eb1-8c01-4e26-bc60-965cb2dcc789', 'Recharge', 'a', FALSE)`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if _, _, err := replayMatchIdentity(ctx, db, nil, "edc5daf6"); !errors.Is(err, replaybuild.ErrModeNonGere) {
		t.Fatalf("Firefight (forme courte) : %v, attendu ErrModeNonGere", err)
	}
	id, names, err := replayMatchIdentity(ctx, db, nil, "39910eb1-8c01-4e26-bc60-965cb2dcc789")
	if err != nil || id == "" || len(names) != 1 || names[0] != "Recharge" {
		t.Fatalf("match gere : %q %v %v", id, names, err)
	}
}
