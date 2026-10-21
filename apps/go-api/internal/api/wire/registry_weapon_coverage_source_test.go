//go:build cgo

package wire

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/persist"
)

// classifieurDeTest : deux sources résolues, les autres non.
type classifieurDeTest struct{}

func (classifieurDeTest) KillSourceRegistryKey(tag uint32) (string, bool) {
	switch tag {
	case 0x0a000001:
		return "arme_a", true
	case 0x0a000002:
		return "arme_b", true
	}
	return "", false
}

func mortAvecSource(ms int, tueur string, tag uint32) persist.KillEventInsert {
	d := persist.KillEventInsert{TimeMS: ms, VictimGamertag: "V", VictimXUID: "2", FeedKillerGamertag: "K",
		FeedKillerXUID: tueur, FeedPresent: true, ReadPath: killscope.ReadPathFilmWalk, ReadOrigin: "credit-concordant"}
	if tag != 0 {
		d.SourceTag, d.SourceCategory = tag, "None"
	}
	return d
}

// TestWeaponCoverageFromSource_LitLesResumes : la couverture de la flotte se lit sur les frags par
// paire (S1, ADR 0039) d'un journal écrit par `persist`, et compte exactement les morts à source
// mesurée de la dernière passe de chaque match — tueur bot, source non résolue et passe supplantée
// compris.
func TestWeaponCoverageFromSource_LitLesResumes(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrations : %v", err)
	}
	ctx := context.Background()
	p := persist.NewKillSourcePersister(db)
	passes := []persist.KillSourceBatch{
		// m1 : une passe supplantée (une mort A), puis la courante.
		{MatchID: "m1", DecoderRev: "t", Publishable: true, Deaths: []persist.KillEventInsert{mortAvecSource(1, "1", 0x0a000001)}},
		{MatchID: "m1", DecoderRev: "t", Publishable: true, Deaths: []persist.KillEventInsert{
			mortAvecSource(1, "1", 0x0a000001), mortAvecSource(2, "1", 0x0a000001), mortAvecSource(3, "", 0x0a000002),
			mortAvecSource(4, "1", 0x0a0000ff), mortAvecSource(5, "1", 0)}},
		// m2 : passe non publiable (les frags comptent quand même, comme le journal).
		{MatchID: "m2", DecoderRev: "t", Publishable: false, Deaths: []persist.KillEventInsert{
			mortAvecSource(1, "3", 0x0a000002), mortAvecSource(2, "3", 0x0a0000ff)}},
	}
	for _, b := range passes {
		if err := p.PersistPass(ctx, b); err != nil {
			t.Fatalf("PersistPass %s : %v", b.MatchID, err)
		}
	}

	resp, err := weaponCoverageFromSource(ctx, db, nil, classifieurDeTest{}, domain.AdminWeaponCoverage{})
	if err != nil {
		t.Fatalf("weaponCoverageFromSource : %v", err)
	}
	// Sources mesurées de la flotte : A (2 morts de m1), B (1 de m1 + 1 de m2), 0x0a0000ff (1 + 1).
	if resp.DistinctWeapons != 3 || resp.ResolvedRegistry != 2 || resp.Unresolved != 1 {
		t.Fatalf("couverture = %+v, attendu 3 sources dont 2 résolues et 1 non", resp)
	}
	if len(resp.TopUnresolved) != 1 || resp.TopUnresolved[0].Kills != 2 {
		t.Fatalf("non résolues = %+v, attendu la source inconnue à 2 morts (m1 et m2)", resp.TopUnresolved)
	}
}
