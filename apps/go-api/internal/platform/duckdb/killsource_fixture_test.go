//go:build integration

// Package duckdb — killsource_fixture_test.go : LA fixture partagee des lecteurs adosses a
// la source de degat du film.
//
// Deux tests l'utilisent — le lecteur d'arme (killsource_weapon_kills_repo_test.go) et le
// lecteur de distance (kill_distance_repo_test.go). Elle vivait dans le fichier de test du
// lecteur hors arsenal ; celui-ci a disparu avec la fusion des chemins de chargement
// (decision D11 du plan du 2026-09-01), et la fixture a alors ete extraite ICI plutot que
// recopiee dans chacun des deux.
//
// Ce qu'elle monte : un shared `:memory:` avec les VRAIES migrations (dont la vue
// `match_kill_events_latest`) et une metadata portant le VRAI registre d'armes.
package duckdb

import (
	"context"
	"database/sql"
	"testing"

	titlepkg "levelup/go-api/internal/domain/title"

	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/migration"
)

const (
	kscMatchID  = "m_killsource_001"
	kscXUID     = "xuid(2533274000000042)"
	kscDecodeV1 = "pass_v1"
	kscDecodeV2 = "pass_v2"
)

// Tags de test. Les valeurs exactes n'ont pas d'importance : le classificateur est un
// double de test (le vrai vit dans games/halo_infinite et a ses propres tests).
const (
	kscTagRepulsor = uint32(0x07104b31) // hors arsenal : aucun identifiant numerique
	kscTagCoil     = uint32(0x0d203522) // hors arsenal
	kscTagFall     = uint32(0x00000024) // chute / environnement
	kscTagRifle    = uint32(0x0badc0de) // arme a feu ORDINAIRE
	kscTagSidearm  = uint32(0x0badc0df) // seconde arme a feu, autre classe
	kscTagInconnu  = uint32(0x00000099) // aucune cle : reste dans « Non attribue »
)

// fakeKillSourceClassifier : double de test de port.KillSourceClassifier, doublant AUSSI
// port.KillSourceDescriber (le nom de classe sert la journalisation des morts ecartees).
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
	case kscTagSidearm:
		return "hinf_sidekick", true
	}
	return "", false
}

func (fakeKillSourceClassifier) KillSourceClassName(tag uint32) (string, bool) {
	if tag == kscTagInconnu {
		return "VEHICULE", true
	}
	return "ARME", true
}

// newKillSourceTestPlayerDB : shared migre (vue `_latest` incluse) + metadata portant le
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

// insertKill pose une mort dans match_kill_events. killerXUID vide = tueur BOT (xuid NULL) ;
// tag nul = source non mesuree.
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
