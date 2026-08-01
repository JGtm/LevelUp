//go:build integration

// Package persist — kill_events_persister_test.go : ce que le persister ECRIT, et surtout ce
// qu il REFUSE. Chaque refus protege une propriete que le schema affirme ; un test par refus.
//
// Le schema est celui des migrations REELLES (RunForDB sur TargetShared), pas un DDL recopie :
// un test qui inventerait sa propre table ne prouverait rien sur la prod.

package persist

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/migration"
)

func openKillEventsTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
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

// mortValide : une mort minimale acceptable, a modifier champ par champ dans les tests de refus.
func mortValide() KillEventInsert {
	return KillEventInsert{
		TimeMS:         1000,
		VictimGamertag: "Victime",
		FeedPresent:    true,
		AssistKnown:    true,
		ReadPath:       "marche",
		ReadOrigin:     "credit-concordant",
	}
}

func passeValide(deaths ...KillEventInsert) KillSourceBatch {
	return KillSourceBatch{
		MatchID:     "match-1",
		DecoderRev:  "killsource-v1",
		Publishable: true,
		Deaths:      deaths,
	}
}

func u8(v uint8) *uint8 { return &v }

// TestPersistPassEcritEtRelitParLaVue — le chemin nominal, lu par la VUE (jamais la table).
func TestPersistPassEcritEtRelitParLaVue(t *testing.T) {
	db := openKillEventsTestDB(t)
	ctx := context.Background()

	mort := mortValide()
	mort.FeedKillerGamertag = "Tueur"
	mort.FeedKillerXUID = "xuid(1)"
	mort.VictimXUID = "xuid(2)"
	mort.SourceTag = 0x6a707421
	mort.SourceCategory = "Headshot"
	mort.Diverges = true
	mort.KillerDamagePct = u8(70)

	if err := NewKillSourcePersister(db).PersistPass(ctx, passeValide(mort)); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var (
		victim, path string
		tag          sql.NullInt64
		cat          sql.NullString
		div          sql.NullBool
		killerPct    sql.NullInt64
		residu       sql.NullInt64
	)
	err := db.QueryRow(`SELECT victim_gamertag, read_path, source_tag, source_category,
		diverges, killer_damage_pct, damage_pct_residual
		FROM match_kill_events_latest WHERE match_id = 'match-1'`).
		Scan(&victim, &path, &tag, &cat, &div, &killerPct, &residu)
	if err != nil {
		t.Fatalf("select vue: %v", err)
	}
	if victim != "Victime" || path != "marche" {
		t.Errorf("victime=%q path=%q", victim, path)
	}
	if !tag.Valid || tag.Int64 != 0x6a707421 || !cat.Valid || cat.String != "Headshot" {
		t.Errorf("source relue tag=%v cat=%v", tag, cat)
	}
	if !div.Valid || !div.Bool {
		t.Errorf("diverges = %v, attendu TRUE", div)
	}
	if !residu.Valid || residu.Int64 != 30 {
		t.Errorf("residu = %v, attendu 30 (solo mesure, total 100)", residu)
	}
}

// TestDeuxPassesLaSecondeRemplaceEntierement — le comportement que `decode_pass` existe pour
// garantir : la vue ne melange jamais deux decodages.
func TestDeuxPassesLaSecondeRemplaceEntierement(t *testing.T) {
	db := openKillEventsTestDB(t)
	ctx := context.Background()
	p := NewKillSourcePersister(db)

	m1, m2, m3 := mortValide(), mortValide(), mortValide()
	m2.TimeMS, m3.TimeMS = 2000, 3000
	m2.VictimGamertag, m3.VictimGamertag = "V2", "V3"
	if err := p.PersistPass(ctx, passeValide(m1, m2, m3)); err != nil {
		t.Fatalf("passe A: %v", err)
	}
	if err := p.PersistPass(ctx, passeValide(m1, m2)); err != nil {
		t.Fatalf("passe B: %v", err)
	}

	var vue, table int
	if err := db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM match_kill_events_latest),
		(SELECT COUNT(*) FROM match_kill_events)`).Scan(&vue, &table); err != nil {
		t.Fatalf("select: %v", err)
	}
	if vue != 2 {
		t.Errorf("vue = %d, attendu 2 (la passe B entiere)", vue)
	}
	if table != 5 {
		t.Errorf("table = %d, attendu 5 (append-only : rien n est efface)", table)
	}
}

// TestPassePartageUnSeulDecodePass — toutes les lignes d une passe portent la MEME generation.
// Si ce n etait pas le cas, la vue ne rendrait qu une ligne au lieu de la passe entiere.
func TestPassePartageUnSeulDecodePass(t *testing.T) {
	db := openKillEventsTestDB(t)
	m2 := mortValide()
	m2.TimeMS = 2000
	if err := NewKillSourcePersister(db).PersistPass(context.Background(),
		passeValide(mortValide(), m2)); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}
	var distincts int
	if err := db.QueryRow(`SELECT COUNT(DISTINCT decode_pass) FROM match_kill_events`).Scan(&distincts); err != nil {
		t.Fatalf("select: %v", err)
	}
	if distincts != 1 {
		t.Errorf("%d decode_pass distincts sur une seule passe, attendu 1", distincts)
	}
}

// TestPasseVideNEcritRien — ecrire zero ligne serait indistinguable d un match sans mort, et
// effacerait la passe precedente de la lecture.
func TestPasseVideNEcritRien(t *testing.T) {
	db := openKillEventsTestDB(t)
	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeValide()); err != nil {
		t.Fatalf("passe vide doit etre ignoree, pas une erreur: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d lignes ecrites pour une passe vide", n)
	}
}

// TestRefusDuPersister — un test par propriete que le schema affirme. Le message d erreur
// compte autant que le refus : c est lui qui dira au brancheur ce qu il a confondu.
func TestRefusDuPersister(t *testing.T) {
	cas := []struct {
		nom      string
		mutation func(*KillEventInsert)
		batch    func(*KillSourceBatch)
		extrait  string
	}{
		{"victime vide", func(d *KillEventInsert) { d.VictimGamertag = "" }, nil, "VictimGamertag"},
		{"instant negatif", func(d *KillEventInsert) { d.TimeMS = -1 }, nil, "TimeMS"},
		{"portee absente", func(d *KillEventInsert) { d.ReadPath = "" }, nil, "portee"},
		{"demi-source (tag sans categorie)", func(d *KillEventInsert) { d.SourceTag = 42 }, nil, "moitie"},
		{"demi-source (categorie sans tag)", func(d *KillEventInsert) { d.SourceCategory = "None" }, nil, "moitie"},
		{"divergence sans source", func(d *KillEventInsert) { d.Diverges = true }, nil, "divergence"},
		{"assistant nomme mais AssistKnown faux", func(d *KillEventInsert) {
			d.AssistKnown = false
			d.AssistGamertag = "A"
		}, nil, "on ne sait pas"},
		{"part d assistant sans assistant", func(d *KillEventInsert) {
			d.AssistDamagePct = u8(30)
		}, nil, "sans assistant nomme"},
		{"AssistExtra negatif", func(d *KillEventInsert) { d.AssistExtra = -1 }, nil, "AssistExtra"},
		{"DecoderRev vide", nil, func(b *KillSourceBatch) { b.DecoderRev = "" }, "DecoderRev"},
		{"MatchID vide", nil, func(b *KillSourceBatch) { b.MatchID = "" }, "MatchID"},
	}

	db := openKillEventsTestDB(t)
	ctx := context.Background()
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			d := mortValide()
			if c.mutation != nil {
				c.mutation(&d)
			}
			b := passeValide(d)
			if c.batch != nil {
				c.batch(&b)
			}
			err := NewKillSourcePersister(db).PersistPass(ctx, b)
			if err == nil {
				t.Fatalf("refus attendu, la passe a ete acceptee")
			}
			if !strings.Contains(err.Error(), c.extrait) {
				t.Errorf("message = %q, attendu contenant %q", err.Error(), c.extrait)
			}
		})
	}

	// La validation passe AVANT la transaction : aucun refus ne laisse de ligne derriere lui.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 0 {
		t.Errorf("%d ligne(s) laissee(s) par des passes refusees — la validation doit passer "+
			"AVANT la transaction", n)
	}
}

// TestCasNormalMortDeBotAccepte — `FeedPresent = false` AVEC un tueur nomme est le cas NORMAL
// d une victime bot (le KILL est au feed, la MORT n y est pas), et `victim_xuid` NULL en est la
// signature. `killer_victim_pairs` ne sait pas representer ce cas : 0 ligne de bot en prod.
func TestCasNormalMortDeBotAccepte(t *testing.T) {
	db := openKillEventsTestDB(t)
	d := mortValide()
	d.FeedPresent = false
	d.FeedKillerGamertag = "Tueur"
	d.FeedKillerXUID = "xuid(1)"
	d.VictimGamertag = "bid(0000000000000)"
	d.ReadOrigin = "bot"

	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeValide(d)); err != nil {
		t.Fatalf("mort de bot refusee: %v", err)
	}
	var xuid sql.NullString
	if err := db.QueryRow(`SELECT victim_xuid FROM match_kill_events_latest`).Scan(&xuid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if xuid.Valid {
		t.Errorf("victim_xuid = %v, attendu NULL (un bot n a pas de XUID)", xuid)
	}
}

// TestCasSymetriqueTueurBotAccepte — RE_LOG 7ter.79 : le TUEUR est un bot, donc un
// `feed_killer_gamertag` RENSEIGNE sans `feed_killer_xuid`. Le guide posait explicitement la
// question « le persister accepte-t-il un tueur nomme sans XUID ? » — la reponse est OUI, et
// c est ce test qui la fixe.
func TestCasSymetriqueTueurBotAccepte(t *testing.T) {
	db := openKillEventsTestDB(t)
	d := mortValide()
	d.FeedKillerGamertag = "bid(0000000000000)"
	d.FeedKillerXUID = ""
	d.VictimXUID = "xuid(2)"
	d.ReadOrigin = "tueur-bot"

	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeValide(d)); err != nil {
		t.Fatalf("tueur bot refuse: %v", err)
	}
	var gt string
	var xuid sql.NullString
	if err := db.QueryRow(`SELECT feed_killer_gamertag, feed_killer_xuid
		FROM match_kill_events_latest`).Scan(&gt, &xuid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if gt == "" || xuid.Valid {
		t.Errorf("tueur bot : gamertag=%q xuid=%v, attendu gamertag renseigne + xuid NULL", gt, xuid)
	}
}

// TestPartDeDegatsSuperieureA100Acceptee — 1,7 % des kill-events attaches a de vraies morts
// nommees vont jusqu a 228. Le plafond a existe : il faisait echouer la passe ENTIERE d un
// match sur UNE ligne, et jetait de la donnee reelle pour proteger une interpretation.
func TestPartDeDegatsSuperieureA100Acceptee(t *testing.T) {
	db := openKillEventsTestDB(t)
	d := mortValide()
	d.KillerDamagePct = u8(228)

	if err := NewKillSourcePersister(db).PersistPass(context.Background(), passeValide(d)); err != nil {
		t.Fatalf("part > 100 refusee: %v — aucun plafond ne doit exister", err)
	}
	var pct int
	if err := db.QueryRow(`SELECT killer_damage_pct FROM match_kill_events_latest`).Scan(&pct); err != nil {
		t.Fatalf("select: %v", err)
	}
	if pct != 228 {
		t.Errorf("part relue = %d, attendu 228 (stockee telle qu elle a ete lue)", pct)
	}
}

// TestTroisEtatsDeLAssistantSurviventEnBase — LE test de la doctrine : « on ne sait pas »,
// « pas d assistant, MESURE » et « assistant nomme » sont trois etats DISTINCTS en base.
// Les confondre publierait des faits « 0 assist » jamais observes.
func TestTroisEtatsDeLAssistantSurviventEnBase(t *testing.T) {
	db := openKillEventsTestDB(t)

	inconnu := mortValide()
	inconnu.AssistKnown = false
	inconnu.VictimGamertag = "V-inconnu"

	aucun := mortValide()
	aucun.TimeMS = 2000
	aucun.VictimGamertag = "V-aucun"

	nomme := mortValide()
	nomme.TimeMS = 3000
	nomme.VictimGamertag = "V-nomme"
	nomme.AssistGamertag = "Assistant"
	nomme.AssistXUID = "xuid(9)"
	idx := 7
	nomme.AssistIndex = &idx

	if err := NewKillSourcePersister(db).PersistPass(context.Background(),
		passeValide(inconnu, aucun, nomme)); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	type etat struct {
		known bool
		name  sql.NullString
		index sql.NullInt64
	}
	lu := map[string]etat{}
	rows, err := db.Query(`SELECT victim_gamertag, assist_known, assist_gamertag, assist_index
		FROM match_kill_events_latest`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var v string
		var e etat
		if err := rows.Scan(&v, &e.known, &e.name, &e.index); err != nil {
			t.Fatalf("scan: %v", err)
		}
		lu[v] = e
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}

	if e := lu["V-inconnu"]; e.known || e.name.Valid {
		t.Errorf("etat 1 (on ne sait pas) : known=%v name=%v", e.known, e.name)
	}
	if e := lu["V-aucun"]; !e.known || e.name.Valid {
		t.Errorf("etat 2 (pas d assistant, MESURE) : known=%v name=%v", e.known, e.name)
	}
	if e := lu["V-nomme"]; !e.known || !e.name.Valid || !e.index.Valid || e.index.Int64 != 7 {
		t.Errorf("etat 3 (assistant nomme) : known=%v name=%v index=%v", e.known, e.name, e.index)
	}
	// L indice ABSENT ne s ecrit jamais -1 : il s ecrit NULL.
	if e := lu["V-aucun"]; e.index.Valid {
		t.Errorf("assist_index = %v sans assistant, attendu NULL (le -1 du decodeur ne doit "+
			"jamais entrer en base)", e.index)
	}
}

// TestAssistExtraInterrogeableEnSQL — le garde-fou de l hypothese « un seul assistant » doit
// etre lisible par `SELECT SUM(assist_extra_count) FROM match_kill_events_latest`. ⚠ Un
// garde-fou muet est pire que pas de garde-fou : on FABRIQUE ici le cas que le corpus n a
// jamais produit, pour prouver que le compteur PEUT bouger.
func TestAssistExtraInterrogeableEnSQL(t *testing.T) {
	db := openKillEventsTestDB(t)

	normal := mortValide()
	suspect := mortValide()
	suspect.TimeMS = 2000
	suspect.VictimGamertag = "V-suspect"
	suspect.AssistGamertag = "Assistant"
	suspect.AssistExtra = 2

	if err := NewKillSourcePersister(db).PersistPass(context.Background(),
		passeValide(normal, suspect)); err != nil {
		t.Fatalf("PersistPass: %v", err)
	}

	var somme int
	if err := db.QueryRow(`SELECT SUM(assist_extra_count) FROM match_kill_events_latest`).Scan(&somme); err != nil {
		t.Fatalf("le compteur de sante doit etre interrogeable en SQL: %v", err)
	}
	if somme != 2 {
		t.Errorf("SUM(assist_extra_count) = %d, attendu 2 — le compteur doit REELLEMENT "+
			"remonter la valeur portee par les lignes, pas valoir zero par construction", somme)
	}
}

// TestPersistViaBatchBuilder — le chemin BatchBuilder (SetKillSource + Persist) ecrit la meme
// chose que le chemin direct, et un batch sans sous-batch est un NO-OP silencieux.
func TestPersistViaBatchBuilder(t *testing.T) {
	db := openKillEventsTestDB(t)
	ctx := context.Background()

	vide := NewBatchBuilder("halo_infinite", "Joueur", "xuid(1)", "test").Build()
	if err := NewKillSourcePersister(db).Persist(ctx, vide); err != nil {
		t.Fatalf("batch sans KillSource doit etre un no-op: %v", err)
	}

	pass := passeValide(mortValide())
	batch := NewBatchBuilder("halo_infinite", "Joueur", "xuid(1)", "test").SetKillSource(&pass).Build()
	if err := NewKillSourcePersister(db).Persist(ctx, batch); err != nil {
		t.Fatalf("Persist via builder: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_kill_events_latest`).Scan(&n); err != nil {
		t.Fatalf("select: %v", err)
	}
	if n != 1 {
		t.Errorf("%d ligne(s), attendu 1", n)
	}
}
