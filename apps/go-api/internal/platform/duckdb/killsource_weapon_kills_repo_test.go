//go:build integration

// Package duckdb — killsource_weapon_kills_repo_test.go : tests du lecteur UNIQUE de
// l'arme d'un kill (KillSourceWeaponKillsRepo).
//
// Round-trip sur DB `:memory:` avec les VRAIES migrations shared (dont la vue
// `match_kill_events_latest`) et le VRAI registre d'armes. La fixture (base migree,
// registre, insertion d'une mort) est partagee avec le lecteur de distance :
// killsource_fixture_test.go.
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. une arme a feu ORDINAIRE remonte desormais — c'est tout le point du lot, et c'est
//     exactement ce que l'ancien lecteur hors arsenal refusait de faire ;
//  2. les dimensions du registre (classe, role, famille, cle, identifiant numerique) sont
//     resolues en UNE passe, sans fan-out sur les armes a plusieurs identifiants ;
//  3. la provenance `FromDamageSource` est posee — sans elle les classes hors arsenal ne
//     seraient pas servies, et le sunburst de Halo 5 bougerait ;
//  4. la vue `_latest` ne melange pas deux passes de decodage ;
//  5. source non mesuree, tueur BOT et source hors registre sont ignores, sans erreur ;
//  6. table absente -> games.ErrCapabilityNotSupported, jamais de panique.
//
// Lancer avec :
//
//	go test -tags=integration -run KillSourceWeaponKills ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

func loadKSW(t *testing.T, pdb *PlayerDB) []port.WeaponKillRow {
	t.Helper()
	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceClassifier{})
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{
			MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}, ResolveRoles: true,
		})
	if err != nil {
		t.Fatalf("LoadWeaponKillsAggregated: %v", err)
	}
	return rows
}

func kswByKey(rows []port.WeaponKillRow) map[string]port.WeaponKillRow {
	out := map[string]port.WeaponKillRow{}
	for _, r := range rows {
		out[r.WeaponKey] = r
	}
	return out
}

// TestKillSourceWeaponKills_ArmeAFeuRemonte : LE test du lot. L'ancien lecteur hors
// arsenal ECARTAIT toute cle portant un identifiant numerique ; celui-ci les sert, avec
// leurs dimensions de registre.
func TestKillSourceWeaponKills_ArmeAFeuRemonte(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 2000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagSidearm, 3000)

	got := kswByKey(loadKSW(t, pdb))
	br, ok := got["hinf_br75"]
	if !ok {
		t.Fatalf("hinf_br75 absent : l'arme a feu ne remonte pas, got %v", got)
	}
	if br.Kills != 2 {
		t.Errorf("Kills = %d, want 2", br.Kills)
	}
	if br.Class != "shoulder" || br.Role != "precision" || br.Family != "battle_rifle" {
		t.Errorf("dimensions = %q/%q/%q, want shoulder/precision/battle_rifle",
			br.Class, br.Role, br.Family)
	}
	if br.WeaponID == 0 {
		t.Error("WeaponID = 0 : une arme a feu porte un identifiant numerique au registre")
	}
	if !br.FromDamageSource {
		t.Error("FromDamageSource = false : la provenance doit etre posee, sinon les classes " +
			"hors arsenal ne seront pas servies")
	}
	if sk, ok := got["hinf_sidekick"]; !ok || sk.Class != "sidearm" {
		t.Errorf("hinf_sidekick = %+v, want classe sidearm", sk)
	}
}

// TestKillSourceWeaponKills_HorsArsenalSansIdentifiant : une bobine remonte AUSSI, avec sa
// classe, et sans identifiant numerique — elle n'en a pas au registre, et c'est exact.
func TestKillSourceWeaponKills_HorsArsenalSansIdentifiant(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 1000)

	got := kswByKey(loadKSW(t, pdb))
	coil, ok := got["hinf_coil_kinetic"]
	if !ok {
		t.Fatalf("hinf_coil_kinetic absent, got %v", got)
	}
	if coil.Class != "environmental" {
		t.Errorf("Class = %q, want environmental", coil.Class)
	}
	if coil.WeaponID != 0 {
		t.Errorf("WeaponID = %d, want 0 : une cle hors arsenal n'a aucun identifiant numerique",
			coil.WeaponID)
	}
}

// TestKillSourceWeaponKills_SourceSansCleEcartee : une source qui ne resout vers aucune
// cle reste dans « Non attribue » — on ne devine pas (D7).
func TestKillSourceWeaponKills_SourceSansCleEcartee(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagInconnu, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 2000)

	got := kswByKey(loadKSW(t, pdb))
	if len(got) != 1 {
		t.Fatalf("%d cles remontees, want 1 (la source inconnue ne doit rien produire) : %v",
			len(got), got)
	}
	if _, ok := got["hinf_br75"]; !ok {
		t.Errorf("hinf_br75 absent, got %v", got)
	}
}

// TestKillSourceWeaponKills_VueLatestNeMelangePas : deux passes de decodage sur le meme
// match — la vue `_latest` ne sert que la derniere, entierement (regle ART n2).
func TestKillSourceWeaponKills_VueLatestNeMelangePas(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 2000)
	insertKill(t, pdb, kscDecodeV2, true, kscXUID, kscTagRifle, 1000)

	got := kswByKey(loadKSW(t, pdb))
	br, ok := got["hinf_br75"]
	if !ok {
		t.Fatalf("hinf_br75 absent, got %v", got)
	}
	if br.Kills != 1 {
		t.Errorf("Kills = %d, want 1 : la vue _latest doit servir la SEULE derniere passe", br.Kills)
	}
}

// TestKillSourceWeaponKills_PasseNonPubliableCompte : une passe marquee non publiable porte
// des lignes justes EN AGREGAT — les exclure ferait perdre 40 % de la mesure. Le lecteur
// les compte, comme le faisait son predecesseur.
func TestKillSourceWeaponKills_PasseNonPubliableCompte(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, false, kscXUID, kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, false, kscXUID, kscTagRifle, 2000)

	if br := kswByKey(loadKSW(t, pdb))["hinf_br75"]; br.Kills != 2 {
		t.Errorf("Kills = %d, want 2 (la passe non publiable COMPTE)", br.Kills)
	}
}

// TestKillSourceWeaponKills_BotEtSourceNonMesuree : un tueur sans xuid (BOT) et une mort
// sans source mesuree sont ignores, sans erreur.
func TestKillSourceWeaponKills_BotEtSourceNonMesuree(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, "", kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, 0, 2000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 3000)

	got := kswByKey(loadKSW(t, pdb))
	br, ok := got["hinf_br75"]
	if !ok {
		t.Fatalf("hinf_br75 absent, got %v", got)
	}
	if br.Kills != 1 {
		t.Errorf("Kills = %d, want 1", br.Kills)
	}
}

// TestKillSourceWeaponKills_MinKills : le filtre MinKills du port est respecte.
func TestKillSourceWeaponKills_MinKills(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 2000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagSidearm, 3000)

	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceClassifier{})
	rows, err := repo.LoadWeaponKillsAggregated(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{
			MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}, MinKills: 2,
		})
	if err != nil {
		t.Fatalf("LoadWeaponKillsAggregated: %v", err)
	}
	if len(rows) != 1 || rows[0].WeaponKey != "hinf_br75" {
		t.Errorf("MinKills=2 rend %v, want la seule hinf_br75", rows)
	}
}

// TestKillSourceWeaponKills_TableAbsente : degradation gracieuse. Une base sans la vue
// `match_kill_events_latest` (titre sans decodeur, migration non appliquee) rend
// games.ErrCapabilityNotSupported — jamais une panique, jamais une erreur opaque.
func TestKillSourceWeaponKills_TableAbsente(t *testing.T) {
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	shared := newTestDB(sharedSQL, ":memory:")
	pdb := &PlayerDB{
		Shared: shared, SharedReader: LegacySharedReader(shared), TitleSlug: titlepkg.DefaultSlug,
	}
	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceClassifier{})
	_, err = repo.LoadWeaponKillsAggregated(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}})
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("err = %v, want games.ErrCapabilityNotSupported", err)
	}
}

// TestKillSourceWeaponKills_FiltresInvalides : sans matchs ni joueurs, la requete
// degenererait en scan complet de la table partagee. Le repo refuse.
func TestKillSourceWeaponKills_FiltresInvalides(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillSourceWeaponKillsRepo(pdb, fakeKillSourceClassifier{})
	if _, err := repo.LoadWeaponKillsAggregated(context.Background(), titlepkg.DefaultSlug,
		port.WeaponKillFilters{}); err == nil {
		t.Error("filtres vides acceptes : le garde-fou anti-scan-complet ne tient pas")
	}
}
