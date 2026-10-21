//go:build integration

package persist

// kill_summaries_integration_test.go — LES RÉSUMÉS PAR MATCH ÉCRITS AVEC LA PASSE (ADR 0039), sur
// les migrations réelles : chaque écrivain du paquet (sync primaire des deux titres, complétion,
// BatchBuilder, persister de passe) pose une tête qui résume la passe courante, deux passes
// successives ne se mélangent pas, une relecture à révision nouvelle ne compte pas deux fois, les
// deux adaptateurs de la fonction pure rendent le même résumé, et une erreur base pendant
// l'écriture des résumés emporte la passe.

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/killsummary"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/sync/testutil"
)

// batchSyncPrimaire : un match neuf et trois couples nommés, tel que le flux primaire le rend.
func batchSyncPrimaire(titre, matchID string) *MatchBatch {
	b := NewBatchBuilder(titre, "Alice", "1111", "test")
	b.SetMatch(&domain.MatchRegistryRow{MatchID: matchID, StartTime: time.Now().UTC(),
		ModeCategory: "PVP", FirstSyncBy: "Alice"})
	couple := func(k, kgt, v, vgt string, ms int64) KillerVictimInsert {
		return KillerVictimInsert{MatchID: matchID, KillerXUID: k, KillerGamertag: kgt,
			VictimXUID: v, VictimGamertag: vgt, Count: 1, TimeMS: ms}
	}
	b.AddKillerVictim([]KillerVictimInsert{
		couple("1111", "Alice", "2222", "Bob", 5000),
		couple("2222", "Bob", "1111", "Alice", 9000),
		couple("1111", "Alice", "2222", "Bob", 12000),
	})
	return b.Build()
}

// paireS1 : une ligne de match_kill_pairs_latest, NULL lus vides.
type paireS1 struct {
	tueur, victime string
	source         int64
	frags          int
	publiable      bool
}

func lirePairesS1(t *testing.T, db *sql.DB, matchID string) []paireS1 {
	t.Helper()
	rows, err := db.Query(`SELECT COALESCE(feed_killer_xuid, ''), COALESCE(victim_xuid, ''),
		COALESCE(CAST(source_tag AS BIGINT), 0), kills, publishable
		FROM match_kill_pairs_latest WHERE match_id = ?
		ORDER BY 1, 2, 3`, matchID)
	if err != nil {
		t.Fatalf("lecture S1 : %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []paireS1
	for rows.Next() {
		var p paireS1
		if err := rows.Scan(&p.tueur, &p.victime, &p.source, &p.frags, &p.publiable); err != nil {
			t.Fatalf("scan S1 : %v", err)
		}
		out = append(out, p)
	}
	return out
}

// lireAppuisS2 relit match_assist_pairs_latest sous la forme du résumé (clé + comptes).
func lireAppuisS2(t *testing.T, db *sql.DB, matchID string) []killsummary.AssistPair {
	t.Helper()
	rows, err := db.Query(`SELECT COALESCE(assist_xuid, ''), COALESCE(assist_gamertag, ''),
		COALESCE(feed_killer_xuid, ''), COALESCE(feed_killer_gamertag, ''), COALESCE(victim_xuid, ''),
		n, n_stolen, sum_assist_pct, n_assist_pct, n_low, n_mid, n_high
		FROM match_assist_pairs_latest WHERE match_id = ? ORDER BY 1, 2, 3, 4, 5`, matchID)
	if err != nil {
		t.Fatalf("lecture S2 : %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []killsummary.AssistPair
	for rows.Next() {
		var a killsummary.AssistPair
		if err := rows.Scan(&a.AssistXUID, &a.AssistGamertag, &a.KillerXUID, &a.KillerGamertag,
			&a.VictimXUID, &a.N, &a.Stolen, &a.PctSum, &a.PctCount, &a.Low, &a.Mid, &a.High); err != nil {
			t.Fatalf("scan S2 : %v", err)
		}
		out = append(out, a)
	}
	return out
}

// passeVariee : une passe de film qui traverse le grain — tueur et victime nommés, victime bot,
// tueur bot (source mesurée), assistant nommé avec parts, assistant bot d'un tueur bot (cas bot-bot), absence
// d'assistant mesurée, assistant inconnu, source mesurée et non mesurée.
func passeVariee(matchID string, publiable bool) KillSourceBatch {
	m := func(ms int, victime, victimeX, tueur, tueurX string) KillEventInsert {
		d := mortValide()
		d.TimeMS, d.VictimGamertag, d.VictimXUID = ms, victime, victimeX
		d.FeedKillerGamertag, d.FeedKillerXUID = tueur, tueurX
		return d
	}
	nommee := m(1000, "Bob", "2222", "Alice", "1111")
	nommee.SourceTag, nommee.SourceCategory = 0x6a707421, "None"
	nommee.AssistGamertag, nommee.AssistXUID = "Carol", "3333"
	nommee.KillerDamagePct, nommee.AssistDamagePct = u8(40), u8(59)
	victimeBot := m(2000, "Bot1", "", "Alice", "1111")
	victimeBot.SourceTag, victimeBot.SourceCategory = 0x6a707421, "Headshot"
	tueurBot := m(3000, "Bob", "2222", "BotK", "")
	// source mesurée MAIS tueur sans xuid : compte dans S1, pas dans n_src_killer.
	tueurBot.SourceTag, tueurBot.SourceCategory = 0x6a707421, "None"
	tueurBot.ReadOrigin = "tueur-bot"
	botBot := m(4000, "Dan", "4444", "BotK", "")
	botBot.AssistGamertag, botBot.AssistDamagePct = "BotA", u8(20)
	sansAssistant := m(5000, "Alice", "1111", "Bob", "2222")
	sansAssistant.KillerDamagePct = u8(100)
	inconnu := m(6000, "Bob", "2222", "Alice", "1111")
	inconnu.AssistKnown = false
	return KillSourceBatch{MatchID: matchID, DecoderRev: "killsource-test", Publishable: publiable,
		Deaths: []KillEventInsert{nommee, victimeBot, tueurBot, botBot, sansAssistant, inconnu}}
}

// TestResumes_SyncPrimaire_DeuxTitres : la passe du flux primaire (SharedPersister) pose sa tête,
// sous Halo Infinite comme sous Halo 5, et S1 porte les frags par paire, source NULL.
func TestResumes_SyncPrimaire_DeuxTitres(t *testing.T) {
	for _, tc := range []struct {
		titre string
		db    func(*testing.T) *sql.DB
	}{
		{"halo_infinite", openKillEventsTestDB},
		{titreConservantWeaponKills, openSharedTestDB},
	} {
		t.Run(tc.titre, func(t *testing.T) {
			db := tc.db(t)
			if err := NewSharedPersister(db).Persist(context.Background(),
				batchSyncPrimaire(tc.titre, "m-sync")); err != nil {
				t.Fatalf("Persist : %v", err)
			}
			testutil.ExigerResumeDeLaPasseCourante(t, db, "m-sync")
			want := []paireS1{{"1111", "2222", 0, 2, true}, {"2222", "1111", 0, 1, true}}
			if got := lirePairesS1(t, db, "m-sync"); !reflect.DeepEqual(got, want) {
				t.Fatalf("S1 = %+v, attendu %+v", got, want)
			}
			if got := lireAppuisS2(t, db, "m-sync"); len(got) != 0 {
				t.Fatalf("S2 = %+v : une passe crédit n'a aucun assistant connu", got)
			}
		})
	}
}

// TestResumes_SyncPrimaire_ErreurDesResumesAnnuleLeMatch : une erreur base à l'écriture de la
// tête fait tomber le match entier du cycle, comme une erreur du journal.
func TestResumes_SyncPrimaire_ErreurDesResumesAnnuleLeMatch(t *testing.T) {
	db := openKillEventsTestDB(t)
	testutil.CasserLaTableDesTetes(t, db)
	if err := NewSharedPersister(db).Persist(context.Background(),
		batchSyncPrimaire("halo_infinite", "m-sync-ko")); err == nil {
		t.Fatal("Persist sans erreur alors que la tête n'a pas pu s'écrire")
	}
	if n := countRows(t, db, "match_registry", "match_id = ?", "m-sync-ko"); n != 0 {
		t.Fatalf("match_registry = %d ligne(s) : le match devait tomber avec ses résumés", n)
	}
	testutil.ExigerAucunePasse(t, db, "m-sync-ko")
}

// TestResumes_Completion : la complétion des événements écrit sa passe crédit avec ses résumés,
// et tombe entière sur une erreur des résumés.
func TestResumes_Completion(t *testing.T) {
	ctx := context.Background()
	db := openCompletionTestDB(t)
	seedCompletionRegistry(t, db, "m-compl")
	if _, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput("m-compl")); err != nil {
		t.Fatalf("Persist : %v", err)
	}
	testutil.ExigerResumeDeLaPasseCourante(t, db, "m-compl")

	seedCompletionRegistry(t, db, "m-compl-ko")
	testutil.CasserLaTableDesTetes(t, db)
	if _, err := NewEventsCompletionPersister(db).Persist(ctx, sampleCompletionInput("m-compl-ko")); err == nil {
		t.Fatal("Persist sans erreur alors que la tête n'a pas pu s'écrire")
	}
	testutil.ExigerAucunePasse(t, db, "m-compl-ko")
	if _, loaded := completionRegistryState(t, db, "m-compl-ko"); loaded {
		t.Fatal("events_loaded posé alors que la transaction de complétion a échoué")
	}
}

// TestResumes_BatchBuilder : le chemin BatchBuilder (SetKillSource + Persist) pose la tête de sa
// passe, et une erreur des résumés n'en laisse rien.
func TestResumes_BatchBuilder(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	pass := passeVariee("m-builder", true)
	batch := NewBatchBuilder("halo_infinite", "Alice", "1111", "test").SetKillSource(&pass).Build()
	if err := NewKillSourcePersister(db).Persist(ctx, batch); err != nil {
		t.Fatalf("Persist : %v", err)
	}
	testutil.ExigerResumeDeLaPasseCourante(t, db, "m-builder")

	ko := passeVariee("m-builder-ko", true)
	testutil.CasserLaTableDesTetes(t, db)
	batchKO := NewBatchBuilder("halo_infinite", "Alice", "1111", "test").SetKillSource(&ko).Build()
	if err := NewKillSourcePersister(db).Persist(ctx, batchKO); err == nil {
		t.Fatal("Persist sans erreur alors que la tête n'a pas pu s'écrire")
	}
	testutil.ExigerAucunePasse(t, db, "m-builder-ko")
}

// TestResumes_DeuxPassesSuccessives : la seconde passe remplace entièrement la première dans
// les vues de résumé ; les tables gardent les deux générations (append-only).
func TestResumes_DeuxPassesSuccessives(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	p := NewKillSourcePersister(db)
	if err := p.PersistPass(ctx, passeVariee("m-2p", true)); err != nil {
		t.Fatalf("passe A : %v", err)
	}
	b := passeVariee("m-2p", false)
	b.Deaths = b.Deaths[:2]
	if err := p.PersistPass(ctx, b); err != nil {
		t.Fatalf("passe B : %v", err)
	}
	e := testutil.ExigerResumeDeLaPasseCourante(t, db, "m-2p")
	if e.FragsS1 != 2 {
		t.Fatalf("S1 compte %d frags, attendu 2 (la passe B entière)", e.FragsS1)
	}
	if e.Publiables != 0 {
		t.Fatalf("%d ligne(s) publiable(s) au journal : la passe B n'est pas publiable", e.Publiables)
	}
	if got := lireAppuisS2(t, db, "m-2p"); len(got) != 0 {
		t.Fatalf("S2 sert %+v : la passe B n'est pas publiable, les appuis de A ne survivent pas", got)
	}
	for _, p := range lirePairesS1(t, db, "m-2p") {
		if p.publiable {
			t.Fatalf("S1 expose publishable=TRUE alors que la tête courante est non publiable : %+v", p)
		}
	}
	if n := countRows(t, db, "match_kill_heads", "match_id = ?", "m-2p"); n != 2 {
		t.Fatalf("%d têtes en table, attendu 2 (rien n'est effacé)", n)
	}
}

// TestResumes_RelectureARevisionNouvelleSansDoubleCompte : la même passe résumée une seconde fois
// (révision nouvelle, puis rattrapage depuis la base) écrit des générations entières de plus ; les
// vues ne servent que la dernière.
func TestResumes_RelectureARevisionNouvelleSansDoubleCompte(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	if err := NewKillSourcePersister(db).PersistPass(ctx, passeVariee("m-relu", true)); err != nil {
		t.Fatalf("PersistPass : %v", err)
	}
	avant := lireAppuisS2(t, db, "m-relu")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	pass, ok, err := readLatestKillPass(ctx, tx, "m-relu")
	if err != nil || !ok {
		t.Fatalf("relecture : ok=%v err=%v", ok, err)
	}
	s := killsummary.Summarize(pass.publishable, pass.rows)
	s.Rev = "ks-relecture-test"
	if err := insertKillSummaryRows(ctx, tx, killSummaryWrite{matchID: "m-relu",
		srcPass: pass.decodePass, writtenAt: pass.writtenAt, summary: s}); err != nil {
		t.Fatalf("insertKillSummaryRows : %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	e := testutil.LireEtatResume(t, db, "m-relu")
	if e.RevTete != "ks-relecture-test" || e.PasseTete != e.PasseJournal || e.FragsS1 != e.LignesJournal {
		t.Fatalf("après relecture : %+v — la nouvelle génération doit être servie seule", e)
	}
	if got := lireAppuisS2(t, db, "m-relu"); !reflect.DeepEqual(got, avant) {
		t.Fatalf("S2 après relecture = %+v, attendu %+v (aucun double compte)", got, avant)
	}
	if ecrit, err := PersistLatestKillPassSummaries(ctx, db, "m-relu"); err != nil || !ecrit {
		t.Fatalf("PersistLatestKillPassSummaries : écrit=%v err=%v", ecrit, err)
	}
	testutil.ExigerResumeDeLaPasseCourante(t, db, "m-relu")
	if n := countRows(t, db, "match_kill_heads", "match_id = ?", "m-relu"); n != 3 {
		t.Fatalf("%d têtes en table, attendu 3 générations", n)
	}
	if ecrit, err := PersistLatestKillPassSummaries(ctx, db, "m-absent"); err != nil || ecrit {
		t.Fatalf("match sans journal : écrit=%v err=%v, attendu rien d'écrit", ecrit, err)
	}
}

// TestResumes_PariteDesAdaptateurs : l'adaptateur en mémoire et l'adaptateur de la base rendent
// le même résumé de la même passe, et ce qui est relu des vues est ce qui a été calculé.
func TestResumes_PariteDesAdaptateurs(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	in := passeVariee("m-parite", true)
	if err := NewKillSourcePersister(db).PersistPass(ctx, in); err != nil {
		t.Fatalf("PersistPass : %v", err)
	}
	memoire := killsummary.Summarize(in.Publishable, killSummaryRowsOf(in.Deaths))
	pass, ok, err := readLatestKillPass(ctx, db, "m-parite")
	if err != nil || !ok {
		t.Fatalf("relecture : ok=%v err=%v", ok, err)
	}
	base := killsummary.Summarize(pass.publishable, pass.rows)
	if !reflect.DeepEqual(memoire, base) {
		t.Fatalf("adaptateurs divergents :\nmémoire %+v\nbase    %+v", memoire, base)
	}
	if got := lireAppuisS2(t, db, "m-parite"); !reflect.DeepEqual(got, memoire.Assists) {
		t.Fatalf("S2 relu = %+v, calculé %+v", got, memoire.Assists)
	}
	var paires []paireS1
	for _, p := range memoire.Pairs {
		paires = append(paires, paireS1{p.KillerXUID, p.VictimXUID, int64(p.SourceTag), p.Kills, true})
	}
	if got := lirePairesS1(t, db, "m-parite"); !reflect.DeepEqual(got, paires) {
		t.Fatalf("S1 relu = %+v, calculé %+v", got, paires)
	}
	if len(memoire.Assists) != 5 {
		t.Fatalf("%d groupes d'appui, attendu 5 (nommé, deux sans assistant, tueur bot, bot-bot)",
			len(memoire.Assists))
	}
}

// TestResumes_CoherencePermanente : sur un corpus écrit par les écrivains réels (passe de film,
// passe de film non publiable, deux passes de film successives, relecture, passe crédit SEULE,
// passe crédit puis passe de film), chaque match a une tête qui résume sa passe courante et S1
// en compte exactement les morts.
func TestResumes_CoherencePermanente(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	p := NewKillSourcePersister(db)
	for _, in := range []KillSourceBatch{
		passeVariee("c-film", true), passeVariee("c-npub", false),
		passeVariee("c-2p", true), passeVariee("c-relu", true),
	} {
		if err := p.PersistPass(ctx, in); err != nil {
			t.Fatalf("PersistPass %s : %v", in.MatchID, err)
		}
	}
	second := passeVariee("c-2p", true)
	second.Deaths = second.Deaths[1:]
	if err := p.PersistPass(ctx, second); err != nil {
		t.Fatalf("seconde passe : %v", err)
	}
	if _, err := PersistLatestKillPassSummaries(ctx, db, "c-relu"); err != nil {
		t.Fatalf("relecture : %v", err)
	}
	ecrireCouples(t, db, []KillerVictimInsert{coupleLive("c-credit-seul", 1000), coupleLive("c-credit-seul", 2000)})
	ecrireCouples(t, db, []KillerVictimInsert{coupleLive("c-credit", 1000), coupleLive("c-credit", 4000)})
	creditPuisFilm := passeVariee("c-credit", true)
	if err := p.PersistPass(ctx, creditPuisFilm); err != nil {
		t.Fatalf("passe de film après crédit : %v", err)
	}
	if n := testutil.ExigerCoherenceDesResumes(t, db); n != 6 {
		t.Fatalf("%d matchs vérifiés, attendu 6", n)
	}
}

// compterResume compte les lignes d'une vue de résumé d'un match qui vérifient `cond`.
func compterResume(t *testing.T, db *sql.DB, vue, matchID, cond string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM `+vue+` WHERE match_id = ? AND (`+cond+`)`, matchID).Scan(&n); err != nil {
		t.Fatalf("%s [%s] : %v", vue, cond, err)
	}
	return n
}

// TestResumes_NullsEcritsEnNull : « absent » est NULL en base, jamais une sentinelle (texte vide,
// source 0) ; les relectures à COALESCE des autres tests ne distinguent pas les deux.
func TestResumes_NullsEcritsEnNull(t *testing.T) {
	db := openKillEventsTestDB(t)
	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeVariee("m-null", true)); err != nil {
		t.Fatalf("PersistPass : %v", err)
	}
	for _, c := range []struct {
		vue, cond string
		attendu   int
	}{
		// S1 : tueurBot et botBot (tueur NULL), victimeBot (victime NULL), trois groupes sans source.
		{"match_kill_pairs_latest", "feed_killer_xuid IS NULL", 2},
		{"match_kill_pairs_latest", "victim_xuid IS NULL", 1},
		{"match_kill_pairs_latest", "source_tag IS NULL", 3},
		{"match_kill_pairs_latest", "feed_killer_xuid = '' OR victim_xuid = '' OR source_tag = 0", 0},
		// S2 : assistant sans xuid (victimeBot, tueurBot, botBot, sansAssistant), sans nom (les trois
		// hors botBot), tueur sans xuid (tueurBot, botBot), victime hors grain (les trois à tueur nommé).
		{"match_assist_pairs_latest", "assist_xuid IS NULL", 4},
		{"match_assist_pairs_latest", "assist_gamertag IS NULL", 3},
		{"match_assist_pairs_latest", "feed_killer_xuid IS NULL", 2},
		{"match_assist_pairs_latest", "feed_killer_gamertag IS NULL", 0},
		{"match_assist_pairs_latest", "victim_xuid IS NULL", 3},
		{"match_assist_pairs_latest", "assist_xuid = '' OR assist_gamertag = '' OR feed_killer_xuid = '' " +
			"OR feed_killer_gamertag = '' OR victim_xuid = ''", 0},
	} {
		if n := compterResume(t, db, c.vue, "m-null", c.cond); n != c.attendu {
			t.Errorf("%s [%s] = %d ligne(s), attendu %d", c.vue, c.cond, n, c.attendu)
		}
	}
}

// TestResumes_ComptesDeLaTeteRelusEnBase : les quatre comptes de la tête, relus de la vue. Le
// corpus a une mort à source mesurée et tueur sans xuid : n_src_killer (2) se distingue de
// n_killer_xuid (4) comme du nombre de sources mesurées (3).
func TestResumes_ComptesDeLaTeteRelusEnBase(t *testing.T) {
	db := openKillEventsTestDB(t)
	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeVariee("m-tete", true)); err != nil {
		t.Fatalf("PersistPass : %v", err)
	}
	var rows, assistKnown, killer, srcKiller int
	var publiable bool
	if err := db.QueryRow(`SELECT n_rows, n_assist_known, n_killer_xuid, n_src_killer, publishable
		FROM match_kill_heads_latest WHERE match_id = 'm-tete'`).Scan(&rows, &assistKnown, &killer, &srcKiller, &publiable); err != nil {
		t.Fatalf("tête : %v", err)
	}
	if rows != 6 || assistKnown != 5 || killer != 4 || srcKiller != 2 || !publiable {
		t.Fatalf("tête = (n_rows %d, n_assist_known %d, n_killer_xuid %d, n_src_killer %d, publiable %v), "+
			"attendu (6, 5, 4, 2, true)", rows, assistKnown, killer, srcKiller, publiable)
	}
	if n := compterResume(t, db, "match_kill_events_latest", "m-tete", "source_tag IS NOT NULL"); n != 3 {
		t.Fatalf("%d sources mesurées au journal, attendu 3 (le corpus doit distinguer n_src_killer)", n)
	}
}

// TestResumes_RelectureDuMatchADeuxPasses : un match écrit en deux passes puis relu par le
// rattrapage est résumé sur sa DERNIÈRE passe seule — pas de mélange des deux, pas de doublon.
func TestResumes_RelectureDuMatchADeuxPasses(t *testing.T) {
	ctx := context.Background()
	db := openKillEventsTestDB(t)
	p := NewKillSourcePersister(db)
	if err := p.PersistPass(ctx, passeVariee("m-relu2", true)); err != nil {
		t.Fatalf("passe A : %v", err)
	}
	b := passeVariee("m-relu2", true)
	b.Deaths = b.Deaths[:3]
	if err := p.PersistPass(ctx, b); err != nil {
		t.Fatalf("passe B : %v", err)
	}
	if ecrit, err := PersistLatestKillPassSummaries(ctx, db, "m-relu2"); err != nil || !ecrit {
		t.Fatalf("PersistLatestKillPassSummaries : écrit=%v err=%v", ecrit, err)
	}
	e := testutil.ExigerResumeDeLaPasseCourante(t, db, "m-relu2")
	if e.LignesJournal != 3 || e.FragsS1 != 3 || e.LignesTete != 3 {
		t.Fatalf("journal %d, tête %d, frags S1 %d : le résumé doit être celui de la passe B (3 morts)",
			e.LignesJournal, e.LignesTete, e.FragsS1)
	}
	voulu := killsummary.Summarize(true, killSummaryRowsOf(b.Deaths))
	if got := lireAppuisS2(t, db, "m-relu2"); !reflect.DeepEqual(got, voulu.Assists) {
		t.Fatalf("S2 = %+v, attendu celui de la passe B seule %+v", got, voulu.Assists)
	}
	if n := countRows(t, db, "match_kill_heads", "match_id = ?", "m-relu2"); n != 3 {
		t.Fatalf("%d têtes en table, attendu 3 générations (A, B, relecture de B)", n)
	}
}

// TestResumes_ErreurDesFillesAnnuleLaPasse : une erreur base à l'insertion de S1 ou de S2 (la tête
// a déjà été écrite dans la transaction) annule la passe entière, journal compris.
func TestResumes_ErreurDesFillesAnnuleLaPasse(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		nom, table string
		ecrire     func(*sql.DB, string) error
	}{
		{"S1 par le persister de passe", "match_kill_pairs", func(db *sql.DB, id string) error {
			return NewKillSourcePersister(db).PersistPass(ctx, passeVariee(id, true))
		}},
		{"S2 par le persister de passe", "match_assist_pairs", func(db *sql.DB, id string) error {
			return NewKillSourcePersister(db).PersistPass(ctx, passeVariee(id, true))
		}},
		{"S1 par le sync primaire", "match_kill_pairs", func(db *sql.DB, id string) error {
			return NewSharedPersister(db).Persist(ctx, batchSyncPrimaire("halo_infinite", id))
		}},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			db := openKillEventsTestDB(t)
			testutil.CasserLaTable(t, db, tc.table)
			if err := tc.ecrire(db, "m-ko"); err == nil {
				t.Fatalf("écriture sans erreur alors que %s n'a pas pu s'écrire", tc.table)
			}
			testutil.ExigerAucunePasse(t, db, "m-ko")
		})
	}
}
