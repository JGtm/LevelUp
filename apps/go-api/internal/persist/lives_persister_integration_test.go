//go:build integration

// Package persist — lives_persister_integration_test.go : LES DEUX TABLES DES FAITS
// D'ISOLEMENT, sur les VRAIES migrations.
//
// # POURQUOI SUR UNE VRAIE BASE, ET NON SUR UN DOUBLE
//
// Ce que ces tests prouvent tient au SQL et au schema, pas au Go : que la vue `_latest` retient
// une PASSE ENTIERE (et non la derniere ligne par cle), et qu'une passe plus courte que la
// precedente ne laisse survivre aucune ligne de l'ancienne. Un double de base ne pourrait rien
// en dire — c'est exactement la lecon du lot 7.10, ou un predicat SQL jamais execute a livre un
// NULL scanne dans un `bool`.
package persist

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

// baseDeTest ouvre un shared migre par les VRAIES migrations.
func baseDeTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", filepath.Join(t.TempDir(), "shared.duckdb"))
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}
	return db
}

// vie / contexte : deux fabriques breves, pour que les tests lisent comme leur intention.
func vie(xuid string, debut, fin int64, cause, nomPar string) LifeInsert {
	return LifeInsert{XUID: xuid, StartMS: debut, EndMS: fin, EndCause: cause, NamedBy: nomPar}
}

func contexte(xuid string, t int64, proche *float64, vis, att, hors, partis int) DeathContextInsert {
	return DeathContextInsert{
		VictimXUID: xuid, TimeMS: t, NearestTeammateM: proche,
		TeammatesVisible: vis, TeammatesWaiting: att, TeammatesOutOfSight: hors,
		TeammatesLeft: partis, TeammatesTotal: vis + att + hors + partis,
	}
}

func metres(v float64) *float64 { return &v }

// TestLivesPersister_DeuxTablesEcrites — la passe nominale ecrit les deux tables, et les vues
// `_latest` les rendent.
func TestLivesPersister_DeuxTablesEcrites(t *testing.T) {
	db := baseDeTest(t)
	err := NewLivesPersister(db).PersistPass(context.Background(), LivesBatch{
		MatchID: "m1", DecoderRev: "rev-1",
		Lives: []LifeInsert{
			vie("111", 0, 10_000, CauseFinMort, NommeParMort),
			vie("222", 0, 30_000, CauseFinFilm, NommeParFermeture),
		},
		Contexts: []DeathContextInsert{contexte("111", 10_000, metres(3), 1, 1, 1, 0)},
		Mates:    ctxMates(contexte("111", 10_000, metres(3), 1, 1, 1, 0)),
	})
	if err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var vies, ctxs int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_lives_latest WHERE match_id = 'm1'),
		(SELECT COUNT(*) FROM match_death_context_latest WHERE match_id = 'm1')`).
		Scan(&vies, &ctxs); err != nil {
		t.Fatalf("select: %v", err)
	}
	if vies != 2 || ctxs != 1 {
		t.Fatalf("vies = %d, contextes = %d, attendu 2 et 1", vies, ctxs)
	}

	// LE SURVIVANT EST NOMME SANS ETRE MORT : c'est tout l'objet des deux colonnes.
	var cause, nomPar string
	if err := db.QueryRow(`SELECT end_cause, named_by FROM match_lives_latest
		WHERE match_id = 'm1' AND xuid = '222'`).Scan(&cause, &nomPar); err != nil {
		t.Fatalf("select 222: %v", err)
	}
	if cause != CauseFinFilm || nomPar != NommeParFermeture {
		t.Fatalf("222 : end_cause=%q named_by=%q, attendu %q et %q — un joueur nomme par "+
			"fermeture n'est pas mort", cause, nomPar, CauseFinFilm, NommeParFermeture)
	}
}

// TestLivesPersister_LaVueRendLaDernierePasseEntiere — L'IDEMPOTENCE DE LA PASSE.
//
// La passe B rend MOINS de vies que la passe A. Si la vue retenait « la derniere ligne par
// cle », les vies que A seule portait survivraient et se melangeraient a B : le contexte d'une
// mort citerait des coequipiers issus de deux decodages differents. La vue retient donc la
// derniere PASSE, entiere.
func TestLivesPersister_LaVueRendLaDernierePasseEntiere(t *testing.T) {
	db := baseDeTest(t)
	p := NewLivesPersister(db)
	ctx := context.Background()

	if err := p.PersistPass(ctx, LivesBatch{
		MatchID: "m1", DecoderRev: "rev-1",
		Lives: []LifeInsert{
			vie("111", 0, 10_000, CauseFinMort, NommeParMort),
			vie("222", 0, 10_000, CauseFinMort, NommeParMort),
			vie("333", 0, 10_000, CauseFinMort, NommeParMort),
		},
		Contexts: []DeathContextInsert{
			contexte("111", 10_000, metres(3), 1, 0, 0, 0),
			contexte("222", 10_000, nil, 0, 1, 0, 0),
		},
		Mates: ctxMates(contexte("111", 10_000, metres(3), 1, 0, 0, 0), contexte("222", 10_000, nil, 0, 1, 0, 0)),
	}); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	// PASSE B : un decodeur ameliore ne retient plus que deux vies et une mort.
	if err := p.PersistPass(ctx, LivesBatch{
		MatchID: "m1", DecoderRev: "rev-2",
		Lives: []LifeInsert{
			vie("111", 0, 12_000, CauseFinMort, NommeParMort),
			vie("222", 0, 12_000, CauseFinFilm, NommeParFermeture),
		},
		Contexts: []DeathContextInsert{contexte("111", 12_000, metres(7), 1, 0, 0, 0)},
		Mates:    ctxMates(contexte("111", 12_000, metres(7), 1, 0, 0, 0)),
	}); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var vies, ctxs int
	var revVies, revCtx string
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_lives_latest WHERE match_id = 'm1'),
		(SELECT COUNT(*) FROM match_death_context_latest WHERE match_id = 'm1'),
		(SELECT MIN(decoder_rev) FROM match_lives_latest WHERE match_id = 'm1'),
		(SELECT MIN(decoder_rev) FROM match_death_context_latest WHERE match_id = 'm1')`).
		Scan(&vies, &ctxs, &revVies, &revCtx); err != nil {
		t.Fatalf("select: %v", err)
	}
	if vies != 2 {
		t.Errorf("vies = %d, attendu 2 : la vue melange deux passes — la troisieme vie de la "+
			"passe A a survecu", vies)
	}
	if ctxs != 1 {
		t.Errorf("contextes = %d, attendu 1 : meme melange sur la seconde table", ctxs)
	}
	if revVies != "rev-2" || revCtx != "rev-2" {
		t.Errorf("decoder_rev = %q / %q, attendu rev-2 des deux cotes : les deux tables doivent "+
			"servir LA MEME passe, sinon un contexte citerait des vies d'un autre decodage",
			revVies, revCtx)
	}

	// LA TABLE, ELLE, A TOUT GARDE : append-only, aucune ligne detruite.
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lives WHERE match_id = 'm1'`).Scan(&total); err != nil {
		t.Fatalf("select brut: %v", err)
	}
	if total != 5 {
		t.Errorf("lignes brutes = %d, attendu 5 (3 + 2) : la table est APPEND-ONLY, la vue seule filtre", total)
	}
}

// TestLivesPersister_LesDeuxTablesPartagentLaPasse — une seule transaction, un seul
// `decode_pass`.
//
// Deux identifiants differents rendraient les vues `_latest` incoherentes ENTRE ELLES : le
// contexte d'une mort pourrait citer des coequipiers dont les vies ne sont plus servies.
func TestLivesPersister_LesDeuxTablesPartagentLaPasse(t *testing.T) {
	db := baseDeTest(t)
	if err := NewLivesPersister(db).PersistPass(context.Background(), LivesBatch{
		MatchID: "m1", DecoderRev: "rev-1",
		Lives:    []LifeInsert{vie("111", 0, 10_000, CauseFinMort, NommeParMort)},
		Contexts: []DeathContextInsert{contexte("111", 10_000, metres(3), 1, 0, 0, 0)},
		Mates:    ctxMates(contexte("111", 10_000, metres(3), 1, 0, 0, 0)),
	}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var passVies, passCtx string
	if err := db.QueryRow(`SELECT
		(SELECT MIN(decode_pass) FROM match_lives WHERE match_id = 'm1'),
		(SELECT MIN(decode_pass) FROM match_death_context WHERE match_id = 'm1')`).
		Scan(&passVies, &passCtx); err != nil {
		t.Fatalf("select: %v", err)
	}
	if passVies != passCtx {
		t.Fatalf("decode_pass = %q (vies) vs %q (contextes) : les deux tables du MEME decodage "+
			"doivent partager la passe", passVies, passCtx)
	}
}

// TestLivesPersister_SansVie_RienNEstEcrit — sans vie, le pont slot->xuid n'a pas ete construit,
// et un contexte calcule sans lui ne reposerait sur rien.
func TestLivesPersister_SansVie_RienNEstEcrit(t *testing.T) {
	db := baseDeTest(t)
	if err := NewLivesPersister(db).PersistPass(context.Background(), LivesBatch{
		MatchID: "m1", DecoderRev: "rev-1",
		Contexts: []DeathContextInsert{contexte("111", 10_000, metres(3), 1, 0, 0, 0)},
		Mates:    ctxMates(contexte("111", 10_000, metres(3), 1, 0, 0, 0)),
	}); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var ctxs int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_death_context WHERE match_id = 'm1'`).Scan(&ctxs); err != nil {
		t.Fatalf("select: %v", err)
	}
	if ctxs != 0 {
		t.Fatalf("contextes = %d, attendu 0 : sans vie, le contexte ne repose sur rien", ctxs)
	}
}

// TestLivesPersister_RefusDeCeQuiNePeutPasEtreUnePasse — les validations REFUSENT, elles ne
// corrigent pas.
//
// Mettre un zero a la place d'un total incoherent, ou choisir une cause par defaut, publierait
// une donnee inventee sous l'autorite de la base.
func TestLivesPersister_RefusDeCeQuiNePeutPasEtreUnePasse(t *testing.T) {
	db := baseDeTest(t)
	p := NewLivesPersister(db)
	uneVie := []LifeInsert{vie("111", 0, 10_000, CauseFinMort, NommeParMort)}

	cas := []struct {
		nom   string
		batch LivesBatch
	}{
		{"decoder_rev vide", LivesBatch{MatchID: "m1", Lives: uneVie}},
		{"cause inconnue", LivesBatch{MatchID: "m1", DecoderRev: "r",
			Lives: []LifeInsert{vie("111", 0, 1, "explosion", NommeParMort)}}},
		{"nommage inconnu", LivesBatch{MatchID: "m1", DecoderRev: "r",
			Lives: []LifeInsert{vie("111", 0, 1, CauseFinMort, "devine")}}},
		{"vie anonyme", LivesBatch{MatchID: "m1", DecoderRev: "r",
			Lives: []LifeInsert{vie("", 0, 1, CauseFinMort, NommeParMort)}}},
		{"fin avant debut", LivesBatch{MatchID: "m1", DecoderRev: "r",
			Lives: []LifeInsert{vie("111", 10, 1, CauseFinMort, NommeParMort)}}},
		{"somme des etats fausse", LivesBatch{MatchID: "m1", DecoderRev: "r", Lives: uneVie,
			Contexts: []DeathContextInsert{{VictimXUID: "111", TimeMS: 1,
				TeammatesVisible: 1, TeammatesTotal: 3}}}},
		{"distance sans coequipier visible", LivesBatch{MatchID: "m1", DecoderRev: "r", Lives: uneVie,
			Contexts: []DeathContextInsert{contexte("111", 1, metres(3), 0, 1, 0, 0)}}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if err := p.PersistPass(context.Background(), c.batch); err == nil {
				t.Fatalf("aucune erreur : la passe a ete acceptee alors qu'elle est %s", c.nom)
			}
		})
	}
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_lives`).Scan(&total); err != nil {
		t.Fatalf("select: %v", err)
	}
	if total != 0 {
		t.Fatalf("%d ligne(s) ecrite(s) malgre les refus : une passe refusee n'ecrit RIEN", total)
	}
}

// ctxMates fabrique les lignes de coequipiers COHERENTES avec des contextes : un coequipier par
// unite de chaque compteur, les visibles a des distances croissantes depuis `NearestTeammateM`.
func ctxMates(cs ...DeathContextInsert) []DeathMateInsert {
	var out []DeathMateInsert
	for _, c := range cs {
		n := 0
		ajouter := func(etat string, x, y, d *float64) {
			n++
			out = append(out, DeathMateInsert{
				VictimXUID: c.VictimXUID, TimeMS: c.TimeMS, MateXUID: string(rune('a' + n)),
				State: etat, X: x, Y: y, DistanceM: d,
			})
		}
		for i := range c.TeammatesVisible {
			d := *c.NearestTeammateM + float64(i)
			x, y := d, 0.0
			ajouter(EtatCoequipierVisible, &x, &y, &d)
		}
		for range c.TeammatesWaiting {
			ajouter(EtatCoequipierEnAttente, nil, nil, nil)
		}
		for range c.TeammatesOutOfSight {
			ajouter(EtatCoequipierHorsDeVue, nil, nil, nil)
		}
	}
	return out
}

// TestLivesPersister_LesCoequipiersSuiventLaPasse — match_death_mates partage le decode_pass des
// deux autres tables, et sa vue `_latest` ne rend que la derniere passe entiere.
func TestLivesPersister_LesCoequipiersSuiventLaPasse(t *testing.T) {
	db := baseDeTest(t)
	p := NewLivesPersister(db)
	ctx := context.Background()
	uneVie := []LifeInsert{vie("111", 0, 10_000, CauseFinMort, NommeParMort)}
	for rev, c := range []DeathContextInsert{
		contexte("111", 10_000, metres(3), 2, 1, 1, 0), // passe A : 4 coequipiers
		contexte("111", 10_000, metres(5), 1, 0, 0, 0), // passe B : 1 coequipier
	} {
		if err := p.PersistPass(ctx, LivesBatch{
			MatchID: "m1", DecoderRev: string(rune('a' + rev)), Lives: uneVie,
			Contexts: []DeathContextInsert{c}, Mates: ctxMates(c),
		}); err != nil {
			t.Fatalf("passe %d: %v", rev, err)
		}
	}
	var brutes, retenues, passes int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_death_mates WHERE match_id = 'm1'),
		(SELECT COUNT(*) FROM match_death_mates_latest WHERE match_id = 'm1'),
		(SELECT COUNT(DISTINCT decode_pass) FROM (
			SELECT decode_pass FROM match_death_mates_latest UNION ALL
			SELECT decode_pass FROM match_lives_latest UNION ALL
			SELECT decode_pass FROM match_death_context_latest))`).
		Scan(&brutes, &retenues, &passes); err != nil {
		t.Fatalf("select: %v", err)
	}
	if brutes != 5 || retenues != 1 {
		t.Fatalf("brutes = %d, retenues = %d, attendu 5 et 1 : append-only, la vue seule filtre", brutes, retenues)
	}
	if passes != 1 {
		t.Fatalf("%d decode_pass distincts : les trois tables du meme decodage doivent partager la passe", passes)
	}
}

// TestLivesPersister_RefusDesCoequipiersIncoherents — les lignes doivent retomber sur les comptes
// de leur mort ; une passe refusee n'ecrit rien.
func TestLivesPersister_RefusDesCoequipiersIncoherents(t *testing.T) {
	db := baseDeTest(t)
	p := NewLivesPersister(db)
	uneVie := []LifeInsert{vie("111", 0, 10_000, CauseFinMort, NommeParMort)}
	c := contexte("111", 10_000, metres(3), 1, 1, 0, 0)
	bon := ctxMates(c)

	avec := func(mod func([]DeathMateInsert) []DeathMateInsert) LivesBatch {
		m := mod(append([]DeathMateInsert(nil), bon...))
		return LivesBatch{MatchID: "m1", DecoderRev: "r", Lives: uneVie, Contexts: []DeathContextInsert{c}, Mates: m}
	}
	cas := map[string]LivesBatch{
		"coequipier manquant": avec(func(m []DeathMateInsert) []DeathMateInsert { return m[:1] }),
		"coequipier en trop":  avec(func(m []DeathMateInsert) []DeathMateInsert { return append(m, m[1]) }),
		"aucune ligne":        avec(func([]DeathMateInsert) []DeathMateInsert { return nil }),
		"etat inconnu":        avec(func(m []DeathMateInsert) []DeathMateInsert { m[1].State = "mort"; return m }),
		"visible sans lieu":   avec(func(m []DeathMateInsert) []DeathMateInsert { m[0].X = nil; return m }),
		"attente avec lieu":   avec(func(m []DeathMateInsert) []DeathMateInsert { d := 1.0; m[1].DistanceM = &d; return m }),
		"plus proche faux":    avec(func(m []DeathMateInsert) []DeathMateInsert { d := 9.0; m[0].DistanceM = &d; return m }),
		"mort sans contexte":  avec(func(m []DeathMateInsert) []DeathMateInsert { m[0].TimeMS = 99; m[1].TimeMS = 99; return m }),
		"mate_xuid vide":      avec(func(m []DeathMateInsert) []DeathMateInsert { m[0].MateXUID = ""; return m }),
	}
	for nom, b := range cas {
		t.Run(nom, func(t *testing.T) {
			if err := p.PersistPass(context.Background(), b); err == nil {
				t.Fatalf("aucune erreur : la passe a ete acceptee (%s)", nom)
			}
		})
	}
	var total int
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM match_death_mates) + (SELECT COUNT(*) FROM match_lives)`).Scan(&total); err != nil {
		t.Fatalf("select: %v", err)
	}
	if total != 0 {
		t.Fatalf("%d ligne(s) ecrite(s) malgre les refus", total)
	}
	// Le cas nominal passe, lui.
	if err := p.PersistPass(context.Background(), avec(func(m []DeathMateInsert) []DeathMateInsert { return m })); err != nil {
		t.Fatalf("passe nominale refusee: %v", err)
	}
}
