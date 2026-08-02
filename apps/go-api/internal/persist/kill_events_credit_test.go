//go:build integration

// Package persist — kill_events_credit_test.go : LE PRODUCTEUR LIVE, et les deux proprietes
// qu il existe pour tenir.
//
// La premiere est une COUVERTURE : avant lui, `match_kill_events` n etait alimentee que par une
// sous-commande hors ligne, donc tout match synchronise apres le dernier backfill n avait aucune
// mort. La seconde est une ERADICATION : les 46,5 % de doublons de `killer_victim_pairs`
// venaient de deux ecrivains qui s additionnaient ; ici ils produisent deux PASSES, et la vue
// n en rend qu une. Le second test est celui qui compte — il echouerait si l append-only
// redevenait un cumul.
//
// Le schema est celui des migrations REELLES (cf. openKillEventsTestDB) : un DDL recopie ne
// prouverait rien sur la prod.

package persist

import (
	"context"
	"database/sql"
	"testing"
)

// coupleLive : un couple tueur -> victime tel que le pipeline de sync le rend (les deux titres
// produisent cette forme — carnage natif pour Halo 5, completion combat pour Halo Infinite).
func coupleLive(matchID string, timeMS int64) KillerVictimInsert {
	return KillerVictimInsert{
		MatchID:        matchID,
		KillerXUID:     "xuid(1)",
		KillerGamertag: "Tueur",
		VictimXUID:     "xuid(2)",
		VictimGamertag: "Victime",
		Count:          1,
		TimeMS:         timeMS,
	}
}

// ecrireCouples joue le chemin du flux primaire (persistKillerVictim) dans sa propre
// transaction, comme le fait SharedPersister.
func ecrireCouples(t *testing.T, db *sql.DB, rows []KillerVictimInsert) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := persistKillerVictim(ctx, tx, rows); err != nil {
		t.Fatalf("persistKillerVictim: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// TestProducteurLiveEcritLesMortsCreditSeul — le chemin nominal : les couples du pipeline
// arrivent dans la table canonique, avec la portee et les TROIS ETATS de l assistant corrects.
//
// Ce qui est verifie ligne a ligne, et pourquoi : `assist_known = FALSE` (« on ne sait pas »,
// jamais « pas d assistant » — le kill-feed ne porte aucun assistant) ; source, categorie,
// divergence et parts ABSENTES et non nulles-comme-zero (la source du degat se lit dans le
// film, et il n y en a pas) ; `publishable = TRUE` (ces lignes valent ce que valait
// `killer_victim_pairs`, deja lue ligne par ligne).
func TestProducteurLiveEcritLesMortsCreditSeul(t *testing.T) {
	db := openKillEventsTestDB(t)
	ecrireCouples(t, db, []KillerVictimInsert{coupleLive("m-live", 4200)})

	var (
		victime, tueur             string
		xuidVictime, xuidTueur     string
		feedPresent, assistKnown   bool
		publishable                bool
		readPath, readOrigin       string
		sourceTag, sourceCategorie sql.NullString
		diverges                   sql.NullBool
		pctTueur, pctAssist        sql.NullInt64
		timeMS                     int
	)
	err := db.QueryRow(`
		SELECT victim_gamertag, feed_killer_gamertag, victim_xuid, feed_killer_xuid,
		       feed_present, assist_known, publishable, read_path, read_origin,
		       CAST(source_tag AS VARCHAR), source_category, diverges,
		       CAST(killer_damage_pct AS BIGINT), CAST(assist_damage_pct AS BIGINT), time_ms
		FROM match_kill_events_latest WHERE match_id = 'm-live'`).
		Scan(&victime, &tueur, &xuidVictime, &xuidTueur,
			&feedPresent, &assistKnown, &publishable, &readPath, &readOrigin,
			&sourceTag, &sourceCategorie, &diverges, &pctTueur, &pctAssist, &timeMS)
	if err != nil {
		t.Fatalf("relecture par la vue: %v", err)
	}

	if victime != "Victime" || tueur != "Tueur" || xuidVictime != "xuid(2)" || xuidTueur != "xuid(1)" {
		t.Errorf("identites: victime=%q tueur=%q xuidV=%q xuidT=%q", victime, tueur, xuidVictime, xuidTueur)
	}
	if timeMS != 4200 {
		t.Errorf("time_ms = %d, attendu 4200 — l instant de la mort ne doit pas se perdre", timeMS)
	}
	if !feedPresent || !publishable {
		t.Errorf("feed_present=%v publishable=%v, attendu true/true", feedPresent, publishable)
	}
	if assistKnown {
		t.Error("assist_known = TRUE : le kill-feed ne porte AUCUN assistant. Ecrire « connu » " +
			"ici fabriquerait un « pas d assistant » jamais observe")
	}
	if readPath != ReadPathLiveFeed || readOrigin != ReadOriginCreditOnly {
		t.Errorf("portee = %q/%q, attendu %q/%q", readPath, readOrigin, ReadPathLiveFeed, ReadOriginCreditOnly)
	}
	if sourceTag.Valid || sourceCategorie.Valid || diverges.Valid || pctTueur.Valid || pctAssist.Valid {
		t.Errorf("source/divergence/parts renseignees (tag=%v cat=%v div=%v pctT=%v pctA=%v) — "+
			"sans film elles sont NON MESUREES, donc NULL et jamais zero",
			sourceTag.Valid, sourceCategorie.Valid, diverges.Valid, pctTueur.Valid, pctAssist.Valid)
	}
}

// TestDeuxPassesNeSAdditionnentPas — L ERADICATION DU DOUBLON, PROUVEE.
//
// C est exactement le scenario qui a produit 46,5 % de doublons dans `killer_victim_pairs` : le
// flux primaire ecrit, puis la completion combat repasse sur le meme match. Dans l ancienne
// table les deux ecritures s ADDITIONNAIENT (INSERT sans suppression d un cote,
// DELETE-then-INSERT de l autre) ; ici elles produisent deux PASSES et la vue n en sert qu une.
//
// Le test verifie les DEUX faces : la table conserve tout (append-only, ADR 0026) et la vue ne
// rend qu une generation. Verifier seulement la vue laisserait passer une regression ou la
// table cesserait d etre append-only.
func TestDeuxPassesNeSAdditionnentPas(t *testing.T) {
	db := openKillEventsTestDB(t)
	couples := []KillerVictimInsert{
		coupleLive("m-double", 1000),
		coupleLive("m-double", 2000),
		coupleLive("m-double", 3000),
	}

	ecrireCouples(t, db, couples) // flux primaire
	ecrireCouples(t, db, couples) // completion combat, meme match

	var vue, table, passes int
	if err := db.QueryRow(`
		SELECT (SELECT COUNT(*) FROM match_kill_events_latest WHERE match_id = 'm-double'),
		       (SELECT COUNT(*) FROM match_kill_events        WHERE match_id = 'm-double'),
		       (SELECT COUNT(DISTINCT decode_pass) FROM match_kill_events WHERE match_id = 'm-double')`).
		Scan(&vue, &table, &passes); err != nil {
		t.Fatalf("comptes: %v", err)
	}

	if vue != len(couples) {
		t.Errorf("la vue rend %d morts pour %d couples — deux passes se sont ADDITIONNEES. "+
			"C est le bug de killer_victim_pairs (46,5 %% de doublons) qui revient", vue, len(couples))
	}
	if table != 2*len(couples) {
		t.Errorf("la table porte %d lignes, attendu %d — elle doit rester append-only "+
			"(une passe remplacee n est pas une passe effacee)", table, 2*len(couples))
	}
	if passes != 2 {
		t.Errorf("%d passes distinctes, attendu 2 — deux ecritures partageant un decode_pass "+
			"feraient rendre a la vue un MELANGE de passes", passes)
	}
}

// TestPreseanceFilmSurProducteurLive — LE FILM GAGNE, ET LE PRODUCTEUR LIVE S EFFACE.
//
// Sans cette regle, une re-synchronisation d un match deja decode ferait de la passe credit-seul
// la generation servie : elle n ajouterait pas de lignes, elle RETIRERAIT de la lecture la
// source du degat fatal. La panne serait invisible — meme nombre de morts, memes tueurs.
func TestPreseanceFilmSurProducteurLive(t *testing.T) {
	db := openKillEventsTestDB(t)
	ctx := context.Background()

	mort := mortValide()
	mort.FeedKillerGamertag = "Tueur"
	mort.SourceTag = 0x6a707421
	mort.SourceCategory = "Headshot"
	passe := passeValide(mort)
	passe.MatchID = "m-film"
	if err := NewKillSourcePersister(db).PersistPass(ctx, passe); err != nil {
		t.Fatalf("passe film: %v", err)
	}

	ecrireCouples(t, db, []KillerVictimInsert{coupleLive("m-film", 1000)})

	var readPath string
	var sourceTag sql.NullString
	if err := db.QueryRow(`
		SELECT read_path, CAST(source_tag AS VARCHAR)
		FROM match_kill_events_latest WHERE match_id = 'm-film'`).Scan(&readPath, &sourceTag); err != nil {
		t.Fatalf("relecture: %v", err)
	}
	if readPath != "marche" {
		t.Errorf("read_path = %q apres passage du producteur live, attendu \"marche\" — "+
			"la passe credit-seul a supplante le film", readPath)
	}
	if !sourceTag.Valid {
		t.Error("source_tag devenu NULL : la source du degat mesuree par le film a ete " +
			"effacee de la lecture par une passe qui ne la mesure pas")
	}
}

// TestPreseanceTesteeEnPositifSurLesVoiesDeFilm — le garde-fou du garde-fou.
//
// La preseance se teste sur la LISTE des voies de film, jamais par la negation des voies de
// credit : un test « read_path <> 'kill-feed' » ferait passer toute voie future pour un film et
// bloquerait silencieusement le producteur live. Ce test fige le sens de la liste.
func TestPreseanceTesteeEnPositifSurLesVoiesDeFilm(t *testing.T) {
	db := openKillEventsTestDB(t)
	ctx := context.Background()

	for _, voie := range FilmReadPaths {
		mort := mortValide()
		mort.ReadPath = voie
		passe := passeValide(mort)
		passe.MatchID = "m-" + voie
		if err := NewKillSourcePersister(db).PersistPass(ctx, passe); err != nil {
			t.Fatalf("passe %s: %v", voie, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		couvert, err := matchCoveredByFilmPass(ctx, tx, "m-"+voie)
		_ = tx.Rollback()
		if err != nil {
			t.Fatalf("preseance %s: %v", voie, err)
		}
		if !couvert {
			t.Errorf("voie %q non reconnue comme une voie de film — le producteur live "+
				"ecraserait la source du degat de ces matchs", voie)
		}
	}
}
