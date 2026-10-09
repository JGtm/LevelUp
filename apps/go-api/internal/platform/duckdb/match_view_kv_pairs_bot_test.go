//go:build integration

// Package duckdb — match_view_kv_pairs_bot_test.go : le scan Q20 survit aux lignes de BOT.
//
// RÉGRESSION COUVERTE (2026-08-03 → 2026-09-02) : la bascule des lecteurs sur la canonique
// `match_kill_events_latest` a introduit des xuid NULL (bots), que le scan recevait dans des
// `string` nus — la PREMIÈRE ligne de bot faisait échouer tout le chargement, et l'erreur
// était avalée en best-effort : section Duels vide sur 245 matchs Infinite qui avaient la
// donnée. Le scan passe par sql.NullString (NULL → "") sur les TROIS colonnes nullables de
// Q20 — `feed_killer_xuid`, `feed_killer_gamertag`, `victim_xuid` ; les agrégateurs écartent
// ensuite "" (cf. doctrine domain.KVPairRaw), le kill feed le NOMME.
package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/killscope"
)

// insertKVKill pose une mort dans match_kill_events avec les DEUX xuids contrôlables
// ("" = NULL = bot). insertKill (fixture killsource) fige victim_xuid à NULL — ici le
// scénario exige aussi des paires joueur↔joueur complètes.
//
// killerGT vide vaut NULL, pas la chaîne vide : `feed_killer_gamertag` est NULLABLE au DDL
// au même titre que les xuid, donc le scan doit lui survivre aussi. Seuls `victim_gamertag`
// et `time_ms` sont NOT NULL — le test ne peut pas les vider.
func insertKVKill(t *testing.T, pdb *PlayerDB, killerXUID, killerGT, victimXUID, victimGT string, timeMS int) {
	t.Helper()
	var killer any
	if killerXUID != "" {
		killer = killerXUID
	}
	var killerName any
	if killerGT != "" {
		killerName = killerGT
	}
	var victim any
	if victimXUID != "" {
		victim = victimXUID
	}
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO match_kill_events
			(match_id, decode_pass, decoder_rev, publishable, time_ms,
			 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
			 feed_present, assist_known, read_path, read_origin)
		VALUES (?, 'pass_kv', 'rev_test', TRUE, ?, ?, ?, ?, ?, TRUE, FALSE, ?, 'credit-concordant')`,
		kscMatchID, timeMS, victimGT, victim, killerName, killer, killscope.ReadPathFilmWalk)
	if err != nil {
		t.Fatalf("insert kv kill: %v", err)
	}
}

func TestGetMatchKVPairs_LignesDeBotNeCassentPasLeScan(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	// DEUX paires joueur↔joueur, une mort infligée PAR un bot (killer NULL), une mort DE bot
	// (victim NULL) : avant le correctif, la ligne 3 faisait échouer le scan ENTIER — les
	// quatre lignes disparaissaient, pas seulement celle du bot.
	insertKVKill(t, pdb, "2533274000000001", "Alice", "2533274000000002", "Bob", 1000)
	insertKVKill(t, pdb, "2533274000000002", "Bob", "2533274000000001", "Alice", 2000)
	insertKVKill(t, pdb, "", "343 Oscar [bot]", "2533274000000001", "Alice", 3000)
	insertKVKill(t, pdb, "2533274000000002", "Bob", "", "343 Guilty [bot]", 4000)

	pairs, err := repo.GetMatchKVPairs(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 4 {
		t.Fatalf("attendu 4 lignes (les lignes de bot sont SERVIES, pas jetées), obtenu %d", len(pairs))
	}
	// NULL → "" côté Go, gamertags intacts (le journal nomme les bots).
	if pairs[2].KillerXUID != "" || pairs[2].KillerGT != "343 Oscar [bot]" {
		t.Errorf("ligne bot-tueur inattendue : %+v", pairs[2])
	}
	if pairs[3].VictimXUID != "" || pairs[3].VictimGT != "343 Guilty [bot]" {
		t.Errorf("ligne bot-victime inattendue : %+v", pairs[3])
	}
	for i := 0; i < 2; i++ {
		if pairs[i].KillerXUID == "" || pairs[i].VictimXUID == "" {
			t.Errorf("paire joueur↔joueur %d amputée de son identité : %+v", i, pairs[i])
		}
	}
}

// TestGetMatchKVPairs_TueurSansNomNeCassePasLeScan : `feed_killer_gamertag` est NULLABLE au
// DDL exactement comme les xuid. Le scan le lisait encore dans un `string` nu après le
// premier correctif — une seule ligne de tueur non nommé aurait rejoué le même sinistre :
// TOUT le match perdu, l'erreur avalée en best-effort.
func TestGetMatchKVPairs_TueurSansNomNeCassePasLeScan(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	insertKVKill(t, pdb, "2533274000000001", "Alice", "2533274000000002", "Bob", 1000)
	insertKVKill(t, pdb, "", "", "2533274000000001", "Alice", 2000) // tueur ni nommé ni identifié

	pairs, err := repo.GetMatchKVPairs(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("attendu 2 lignes, obtenu %d", len(pairs))
	}
	if pairs[1].KillerGT != "" || pairs[1].KillerXUID != "" {
		t.Errorf("tueur anonyme : %+v, attendu xuid et gamertag vides", pairs[1])
	}
	if pairs[0].KillerGT != "Alice" {
		t.Errorf("la ligne nommée a été abîmée : %+v", pairs[0])
	}
}

// kvLigne : une mort posée avec sa portée (passe publiable ou non, origine de la ligne).
type kvLigne struct {
	publiable               bool
	origine                 string
	tueurXUID, tueurNom     string
	victimeXUID, victimeNom string
	instantMS               int
}

func insertKVLigne(t *testing.T, pdb *PlayerDB, l kvLigne) {
	t.Helper()
	nul := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	_, err := pdb.Shared.Exec(context.Background(), `
		INSERT INTO match_kill_events
			(match_id, decode_pass, decoder_rev, publishable, time_ms,
			 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
			 feed_present, assist_known, read_path, read_origin)
		VALUES (?, 'pass_kv', 'rev_test', ?, ?, ?, ?, ?, ?, TRUE, FALSE, ?, ?)`,
		kscMatchID, l.publiable, l.instantMS, l.victimeNom, nul(l.victimeXUID),
		nul(l.tueurNom), nul(l.tueurXUID), killscope.ReadPathFilmWalk, l.origine)
	if err != nil {
		t.Fatalf("insert kv ligne: %v", err)
	}
}

// TestGetMatchKVPairs_PasseNonPubliableGardeLeKillFeed : dans une passe NON publiable, la paire
// d'une ligne de kill-feed (crédit seul ou crédit enrichi par le film) reste lue — son identité
// vient du crédit — et seules les lignes dont un côté est nommé par la réplication (victime bot,
// tueur bot) tombent. Mutation : retirer le filtre de Q20 ramène 4 lignes ; filtrer sur
// `publishable` seul n'en laisse aucune.
func TestGetMatchKVPairs_PasseNonPubliableGardeLeKillFeed(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	alice, bob := "2533274000000001", "2533274000000002"
	insertKVLigne(t, pdb, kvLigne{false, "credit-concordant", alice, "Alice", bob, "Bob", 1000})
	insertKVLigne(t, pdb, kvLigne{false, killscope.OriginCreditOnly, bob, "Bob", alice, "Alice", 2000})
	insertKVLigne(t, pdb, kvLigne{false, killscope.OriginFilmBotVictim, alice, "Alice", "", "343 Guilty [bot]", 3000})
	insertKVLigne(t, pdb, kvLigne{false, killscope.OriginFilmBotKiller, "", "343 Oscar [bot]", bob, "Bob", 4000})

	pairs, err := repo.GetMatchKVPairs(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("attendu 2 lignes de kill-feed (les lignes de réplication d'une passe non "+
			"publiable tombent), obtenu %d : %+v", len(pairs), pairs)
	}
	if pairs[0].TimeMS != 1000 || pairs[1].TimeMS != 2000 {
		t.Errorf("lignes servies inattendues : %+v", pairs)
	}
}

// TestGetMatchKVPairs_PassePubliableServantLesBots : la même population dans une passe
// PUBLIABLE est servie entière — les bots comptent, leur xuid NULL reste vide côté Go.
func TestGetMatchKVPairs_PassePubliableServantLesBots(t *testing.T) {
	pdb := newKillSourceTestPlayerDB(t)
	repo := NewMatchViewRepo(pdb, pTestXUID)

	alice, bob := "2533274000000001", "2533274000000002"
	insertKVLigne(t, pdb, kvLigne{true, "credit-concordant", alice, "Alice", bob, "Bob", 1000})
	insertKVLigne(t, pdb, kvLigne{true, killscope.OriginFilmBotVictim, alice, "Alice", "", "343 Guilty [bot]", 3000})
	insertKVLigne(t, pdb, kvLigne{true, killscope.OriginFilmBotKiller, "", "343 Oscar [bot]", bob, "Bob", 4000})

	pairs, err := repo.GetMatchKVPairs(context.Background(), kscMatchID)
	if err != nil {
		t.Fatalf("GetMatchKVPairs: %v", err)
	}
	if len(pairs) != 3 {
		t.Fatalf("attendu 3 lignes (bots compris), obtenu %d : %+v", len(pairs), pairs)
	}
	if pairs[1].VictimXUID != "" || pairs[2].KillerXUID != "" {
		t.Errorf("xuid de bot normalisé : %+v", pairs)
	}
}
