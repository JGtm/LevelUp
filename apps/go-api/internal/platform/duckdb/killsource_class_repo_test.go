//go:build integration

// Package duckdb — killsource_class_repo_test.go : tests KillSourceClassRepo.
//
// Round-trip sur DB :memory: avec les VRAIES migrations shared (dont la vue
// match_kill_events_latest) et le VRAI registre d'armes, comme
// weapon_kills_v3_repo_test.go.
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. une passe NON PUBLIABLE est COMPTEE (c'est le point du lot : ces lignes sont
//     justes en agregat, et les exclure ferait perdre 40 % de la mesure) ;
//  2. une cle qui PORTE un id numerique n'est JAMAIS remontee (anti-double-comptage
//     avec weapon_kills) ;
//  3. la vue _latest ne melange pas deux passes de decodage ;
//  4. source non mesuree et tueur BOT sont ignores, sans erreur.
//
// Lancer avec : go test -tags=integration -run KillSourceClass ./internal/platform/duckdb/ -v
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/port"
)

const (
	kscMatchID  = "m_killsource_001"
	kscXUID     = "xuid(2533274000000042)"
	kscOtherGT  = "xuid(2533274000000043)"
	kscDecodeV1 = "pass_v1"
	kscDecodeV2 = "pass_v2"
)

// Tags de test. Les valeurs exactes n'ont pas d'importance : le classificateur est un
// double de test (le vrai vit dans games/halo_infinite et a ses propres tests).
const (
	kscTagRepulsor = uint32(0x07104b31)
	kscTagCoil     = uint32(0x0d203522)
	kscTagFall     = uint32(0x00000024)
	kscTagRifle    = uint32(0x0badc0de) // arme a feu ORDINAIRE : ne doit jamais remonter
)

// fakeKillSourceClassifier : double de test de port.KillSourceClassifier.
//
// Il rend volontairement une cle pour une ARME A FEU ordinaire (`hinf_br75`, qui PORTE un
// id numerique) : c'est ainsi qu'on prouve que le filtre anti-double-comptage tient au
// niveau du repo, et pas seulement par la bonne volonte du classificateur.
type fakeKillSourceClassifier struct{}

func (fakeKillSourceClassifier) KillSourceRegistryKey(tag uint32) (string, bool) {
	switch tag {
	case kscTagRepulsor:
		return "hinf_repulsor", true
	case kscTagCoil:
		return "hinf_coil_kinetic", true
	case kscTagFall:
		return "hinf_environment", true
	case kscTagRifle:
		return "hinf_br75", true
	}
	return "", false
}

// newKillSourceTestPlayerDB : shared migre (vue _latest incluse) + metadata portant le
// VRAI registre d'armes et ses libelles.
func newKillSourceTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	if err := migration.RunForDB(sharedSQL, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}

	metaSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open meta mem: %v", err)
	}
	t.Cleanup(func() { _ = metaSQL.Close() })
	if err := weapons.ApplyRegistry(metaSQL); err != nil {
		t.Fatalf("ApplyRegistry: %v", err)
	}

	shared := newTestDB(sharedSQL, ":memory:")
	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		Metadata:     newTestDB(metaSQL, ":memory:"),
		XUID:         pTestXUID,
		Gamertag:     pTestGamertag,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// insertKill pose une mort dans match_kill_events. killerXUID vide = tueur BOT (xuid NULL).
func insertKill(t *testing.T, pdb *PlayerDB, pass string, publishable bool,
	killerXUID string, tag uint32, timeMS int,
) {
	t.Helper()
	var killer any
	if killerXUID != "" {
		killer = killerXUID
	}
	var srcTag any
	var srcCat any
	if tag != 0 {
		srcTag = int64(tag)
		srcCat = "None"
	}
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO match_kill_events
			(match_id, decode_pass, decoder_rev, publishable, time_ms,
			 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
			 feed_present, assist_known, source_tag, source_category, read_path, read_origin)
		VALUES (?, ?, 'rev_test', ?, ?, 'Victime', NULL, 'Tueur', ?, TRUE, FALSE, ?, ?, ?, 'credit-concordant')`,
		kscMatchID, pass, publishable, timeMS, killer, srcTag, srcCat, killscope.ReadPathFilmWalk)
	if err != nil {
		t.Fatalf("insert kill: %v", err)
	}
}

func loadKSC(t *testing.T, pdb *PlayerDB) []port.KillSourceClassRow {
	t.Helper()
	repo := NewKillSourceClassRepo(pdb, fakeKillSourceClassifier{})
	rows, err := repo.LoadKillSourceClassesAggregated(context.Background(), titlepkg.DefaultSlug,
		port.KillSourceClassFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}})
	if err != nil {
		t.Fatalf("LoadKillSourceClassesAggregated: %v", err)
	}
	return rows
}

func byKey(rows []port.KillSourceClassRow) map[string]port.KillSourceClassRow {
	out := map[string]port.KillSourceClassRow{}
	for _, r := range rows {
		out[r.WeaponKey] = r
	}
	return out
}

// TestKillSourceClass_ComptePassesNonPubliables : LE test du lot. Deux kills a la bobine,
// un dans une passe publiable et un dans une passe qui ne l'est pas — les DEUX comptent,
// et le compteur non-publiable le dit sans rien retrancher.
func TestKillSourceClass_ComptePassesNonPubliables(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 1000)
	insertKill(t, pdb, kscDecodeV1, false, kscXUID, kscTagCoil, 2000)

	got := byKey(loadKSC(t, pdb))
	coil, ok := got["hinf_coil_kinetic"]
	if !ok {
		t.Fatalf("hinf_coil_kinetic absent, got %v", got)
	}
	if coil.Kills != 2 {
		t.Errorf("Kills = %d, want 2 (la passe non publiable COMPTE)", coil.Kills)
	}
	if coil.NonPublishableKills != 1 {
		t.Errorf("NonPublishableKills = %d, want 1", coil.NonPublishableKills)
	}
	if coil.Class != "environmental" {
		t.Errorf("Class = %q, want environmental", coil.Class)
	}
}

// TestKillSourceClass_JamaisDArmeAFeu : une source qui resout vers une cle PORTANT un id
// numerique est ecartee — sinon elle serait comptee une seconde fois par-dessus
// weapon_kills.
func TestKillSourceClass_JamaisDArmeAFeu(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRifle, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRepulsor, 2000)

	got := byKey(loadKSC(t, pdb))
	if _, ok := got["hinf_br75"]; ok {
		t.Error("hinf_br75 remonte : DOUBLE-COMPTAGE avec weapon_kills")
	}
	rep, ok := got["hinf_repulsor"]
	if !ok {
		t.Fatalf("hinf_repulsor absent, got %v", got)
	}
	if rep.Kills != 1 || rep.Class != "equipment" {
		t.Errorf("repulseur = %d kills classe %q, want 1 / equipment", rep.Kills, rep.Class)
	}
}

// TestKillSourceClass_LabelENViaWeaponNameLabels (V2.1, D2+D3, 2026-08-29) : quand
// weapon_name_labels est seedee (comme en prod, via ReconcileNameLabels), le repo rend
// LabelEN EN-first au meme titre que Label FR-first — meme requete, meme passe
// (resolveOffArsenalKeys). Les tests precedents de ce fichier n'ont PAS cette table
// (metadata non seedee, cas nominal des fixtures) : Label/LabelEN y restent vides, ce que
// ce test NE change pas (DB propre a ce test, cf. newKillSourceTestPlayerDB).
func TestKillSourceClass_LabelENViaWeaponNameLabels(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	ctx := context.Background()
	if _, err := pdb.Metadata.Exec(ctx, `CREATE TABLE weapon_name_labels (
		title_slug VARCHAR, weapon_key VARCHAR, name_en VARCHAR, name_fr VARCHAR,
		PRIMARY KEY (title_slug, weapon_key))`); err != nil {
		t.Fatalf("create weapon_name_labels: %v", err)
	}
	if _, err := pdb.Metadata.Exec(ctx,
		"INSERT INTO weapon_name_labels VALUES ('halo_infinite', 'hinf_repulsor', 'Repulsor', 'Répulseur')"); err != nil {
		t.Fatalf("seed weapon_name_labels: %v", err)
	}
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagRepulsor, 1000)

	rep, ok := byKey(loadKSC(t, pdb))["hinf_repulsor"]
	if !ok {
		t.Fatal("hinf_repulsor absent")
	}
	if rep.Label != "Répulseur" {
		t.Errorf("Label = %q, want \"Répulseur\" (FR-first, inchangé)", rep.Label)
	}
	if rep.LabelEN != "Repulsor" {
		t.Errorf("LabelEN = %q, want \"Repulsor\" (EN-first)", rep.LabelEN)
	}
}

// TestKillSourceClass_UneSeulePasse : la vue _latest ne retient qu'UNE passe de decodage.
// Deux passes sur le meme match, avec des comptes differents : c'est la plus recente qui
// sert, jamais la somme des deux.
func TestKillSourceClass_UneSeulePasse(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 1000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 2000)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 3000)
	// Passe plus recente : un seul kill.
	insertKill(t, pdb, kscDecodeV2, true, kscXUID, kscTagCoil, 1000)

	got := byKey(loadKSC(t, pdb))
	if got["hinf_coil_kinetic"].Kills != 1 {
		t.Errorf("Kills = %d, want 1 (la vue _latest retient la passe la plus recente, "+
			"jamais un melange)", got["hinf_coil_kinetic"].Kills)
	}
}

// TestKillSourceClass_IgnoreSourceNulleEtBots : source non mesuree et tueur BOT sont des
// cas NOMINAUX — ils ne remontent pas, et ils ne font pas d'erreur.
func TestKillSourceClass_IgnoreSourceNulleEtBots(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, 0, 1000)             // source non mesuree
	insertKill(t, pdb, kscDecodeV1, true, "", kscTagFall, 2000)         // tueur bot
	insertKill(t, pdb, kscDecodeV1, true, kscOtherGT, kscTagFall, 3000) // autre joueur
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagFall, 4000)

	got := byKey(loadKSC(t, pdb))
	if len(got) != 1 {
		t.Fatalf("%d cles remontees, want 1 : %v", len(got), got)
	}
	env, ok := got["hinf_environment"]
	if !ok || env.Kills != 1 {
		t.Errorf("hinf_environment = %+v, want 1 kill (le joueur demande, et lui seul)", env)
	}
}

// TestKillSourceClass_FiltresRefusentLeScanComplet : le garde-fou de port, verifie a
// travers le repo (l'appelant valide, le repo re-valide en defense).
func TestKillSourceClass_FiltresRefusentLeScanComplet(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewKillSourceClassRepo(pdb, fakeKillSourceClassifier{})
	for _, f := range []port.KillSourceClassFilters{
		{},
		{MatchIDs: []string{kscMatchID}},
		{XUIDs: []string{kscXUID}},
	} {
		if _, err := repo.LoadKillSourceClassesAggregated(
			context.Background(), titlepkg.DefaultSlug, f); err == nil {
			t.Errorf("filtres %+v acceptes : scan complet non refuse", f)
		}
	}
}

// TestKillSourceClass_SansClassificateur : un titre qui n'en fournit pas rend zero ligne
// et AUCUNE erreur — c'est l'etat nominal d'un titre sans decodeur de film (Halo 5).
func TestKillSourceClass_SansClassificateur(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	insertKill(t, pdb, kscDecodeV1, true, kscXUID, kscTagCoil, 1000)

	repo := NewKillSourceClassRepo(pdb, nil)
	rows, err := repo.LoadKillSourceClassesAggregated(context.Background(), titlepkg.DefaultSlug,
		port.KillSourceClassFilters{MatchIDs: []string{kscMatchID}, XUIDs: []string{kscXUID}})
	if err != nil {
		t.Fatalf("erreur inattendue : %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("%d lignes, want 0", len(rows))
	}
}
