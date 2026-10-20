package duckdb

// squad_vehicle_repo_test.go — LA RESSOURCE VEHICULES SE LIT BORNÉE, SUR LES VUES `_latest` (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.3 ; ADR 0036 I2, ADR 0026).
//
// Base `:memory:` migrée (les vraies migrations shared, donc les vraies vues) et le VRAI registre
// d'armes en metadata (la classe d'une source vient du même résolveur que la Répartition des
// frags). Ce que chaque test verrouille :
//   - la lecture ne rend que la DERNIÈRE PASSE ENTIÈRE de chaque match, et les deux fenêtres `_latest`
//     (prises, événements de mort) ne voient que les lignes des matchs demandés ;
//   - les événements de mort ne se lisent que pour les matchs dont la passe est mesurée, a apparié
//     ses frags et en compte ; seuls les frags de classe véhicule ou tourelle sortent, par camp du
//     tueur (un tueur bot, une source non mesurée, une arme à feu, un écrasement sans clé : écartés) ;
//   - un registre sans classe n'est PAS un zéro : les matchs restent « non lus » ;
//   - sans classificateur, les prises se lisent et aucun frag ne sort ; table absente :
//     games.ErrCapabilityNotSupported.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	duckdbdrv "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/weapons"
)

// Tags de test de ce fichier (le classificateur est un double : le vrai vit dans games/halo_infinite).
const (
	svrTagGhost  = uint32(0x5101) // classe vehicle
	svrTagRifle  = uint32(0x5102) // arme à feu, hors engin
	svrTagCrush  = uint32(0x5103) // aucune clé de registre (écrasement)
	svrTagTurret = uint32(0x5104) // classe turret
)

type svrClassifier struct{}

func (svrClassifier) KillSourceRegistryKey(tag uint32) (string, bool) {
	switch tag {
	case svrTagGhost:
		return "hinf_ghost", true
	case svrTagRifle:
		return "hinf_br75", true
	case svrTagTurret:
		return "hinf_turret_machinegun", true
	}
	return "", false
}

// ligneVehicule : une ligne de `match_vehicle_takes` à poser.
type ligneVehicule struct {
	match, pass, written string
	kind                 string // "take" ou "match"
	camp                 int
	xuid, family         string
	takes                int
	aboard               int64
	frags                int
	// Colonnes de la ligne `match`.
	measured, fragsRead       bool
	reason                    string
	fragsTotal, fragsUnmatch  int
	episodesRead, unnamed, nc int
}

func poserLigneVehicule(t *testing.T, pdb *PlayerDB, l ligneVehicule) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_vehicle_takes
		(match_id, decode_pass, written_at, row_kind, camp, xuid, family, takes, aboard_ms, episodes,
		 proximity_episodes, frags, measured, unmeasured_reason, doc_schema, episodes_read,
		 episodes_unnamed, episodes_no_camp, frags_read, frags_reason, frags_total, frags_unmatched)
		VALUES (?, ?, CAST(? AS TIMESTAMP), ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, 71, ?, ?, ?, ?, '', ?, ?)`,
		l.match, l.pass, l.written, l.kind, l.camp, l.xuid, l.family, l.takes, l.aboard, l.takes,
		l.frags, l.measured, l.reason, l.episodesRead, l.unnamed, l.nc, l.fragsRead, l.fragsTotal, l.fragsUnmatch)
}

// passeMatch / prise : les deux natures de ligne d'une passe.
func passeMatch(match, pass, written string, measured, fragsRead bool, reason string, total, unmatched int) ligneVehicule {
	return ligneVehicule{match: match, pass: pass, written: written, kind: "match", measured: measured,
		fragsRead: fragsRead, reason: reason, fragsTotal: total, fragsUnmatch: unmatched,
		episodesRead: 5, unnamed: 1, nc: 0}
}

func prise(match, pass, written string, camp int, xuid, family string, takes int, aboard int64, frags int) ligneVehicule {
	return ligneVehicule{match: match, pass: pass, written: written, kind: "take", camp: camp, xuid: xuid,
		family: family, takes: takes, aboard: aboard, frags: frags, measured: true, fragsRead: true}
}

func poserMort(t *testing.T, pdb *PlayerDB, match, killer string, tag any, n int) {
	t.Helper()
	var k any
	if killer != "" {
		k = killer
	}
	for i := 0; i < n; i++ {
		tacExec(t, pdb, `INSERT INTO match_kill_events
			(match_id, decode_pass, decoder_rev, publishable, time_ms, victim_gamertag, victim_xuid,
			 feed_killer_gamertag, feed_killer_xuid, feed_present, assist_known, source_tag, read_path,
			 read_origin)
			VALUES (?, 'pass_v1', 'rev_test', TRUE, ?, 'Victime', NULL, 'Tueur', ?, TRUE, FALSE, ?, 'film-walk',
			        'credit-concordant')`, match, 1000+i, k, tag)
	}
}

func poserParticipant(t *testing.T, pdb *PlayerDB, match, xuid string, team int) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES (?, ?, ?, ?, 2)`, match, xuid, "gt"+xuid, team)
}

// baseVehicules : shared migré + registre d'armes en metadata.
func baseVehicules(t *testing.T) baseNotee {
	t.Helper()
	b := newBaseNotee(t)
	metaSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open meta mem: %v", err)
	}
	metaSQL.SetMaxOpenConns(1)
	metaSQL.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = metaSQL.Close() })
	if err := weapons.ApplyRegistry(metaSQL); err != nil {
		t.Fatalf("ApplyRegistry: %v", err)
	}
	b.pdb.Metadata = newTestDB(metaSQL, ":memory:")
	return b
}

const (
	svrAncien  = "2026-09-28 10:00:00"
	svrCourant = "2026-09-29 10:00:00"
)

// corpusVehicules : m1 (deux passes ; la courante compte des frags), m2 (mesuré, zéro frag d'engin),
// m3 (non mesuré), m4 (mesuré, frags non appariés), m5 (aucune passe) et m9, HORS de la liste
// demandée, avec autant de lignes que possible.
func corpusVehicules(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	poserLigneVehicule(t, pdb, passeMatch("m1", "p1", svrAncien, true, true, "", 9, 0))
	for _, f := range []string{"warthog", "ghost", "banshee", "wraith"} {
		poserLigneVehicule(t, pdb, prise("m1", "p1", svrAncien, 0, "P", f, 9, 9, 9))
	}
	poserLigneVehicule(t, pdb, passeMatch("m1", "p2", svrCourant, true, true, "", 3, 1))
	poserLigneVehicule(t, pdb, prise("m1", "p2", svrCourant, 0, "P", "ghost", 2, 5000, 2))
	poserLigneVehicule(t, pdb, prise("m1", "p2", svrCourant, 1, "E", "ghost", 1, 3000, 0))
	poserLigneVehicule(t, pdb, passeMatch("m2", "p2", svrCourant, true, true, "", 0, 0))
	poserLigneVehicule(t, pdb, passeMatch("m3", "p2", svrCourant, false, false, "schema_before_67", 0, 0))
	poserLigneVehicule(t, pdb, passeMatch("m4", "p2", svrCourant, true, false, "", 0, 0))
	for i := 0; i < 12; i++ {
		poserLigneVehicule(t, pdb, prise("m9", "p2", svrCourant, 0, "P", fmt.Sprintf("f%d", i), 1, 1, 0))
		poserMort(t, pdb, "m9", "P", int64(svrTagGhost), 1)
	}
	poserLigneVehicule(t, pdb, passeMatch("m9", "p2", svrCourant, true, true, "", 12, 0))

	// Participants : P au camp 0 dans m1 à m4 (pas dans m5) ; E camp 1 ; X absent (camp inconnu).
	for _, m := range []string{"m1", "m2", "m3", "m4"} {
		poserParticipant(t, pdb, m, "P", 0)
	}
	poserParticipant(t, pdb, "m1", "E", 1)
	poserParticipant(t, pdb, "m9", "P", 0)
	// Événements de m1 : 2 frags de P au Ghost, 1 de E, 1 de X (sans camp) ; bruit à écarter : tueur
	// bot, source non mesurée, arme à feu, écrasement sans clé.
	poserMort(t, pdb, "m1", "P", int64(svrTagGhost), 2)
	poserMort(t, pdb, "m1", "E", int64(svrTagGhost), 1)
	poserMort(t, pdb, "m1", "X", int64(svrTagGhost), 1)
	poserMort(t, pdb, "m1", "", int64(svrTagGhost), 1)
	poserMort(t, pdb, "m1", "P", nil, 1)
	poserMort(t, pdb, "m1", "P", int64(svrTagRifle), 2)
	poserMort(t, pdb, "m1", "P", int64(svrTagCrush), 1)
	// m2 et m4 ont des événements, mais leur passe dit qu'il n'y a rien à répartir.
	poserMort(t, pdb, "m2", "P", int64(svrTagGhost), 1)
	poserMort(t, pdb, "m4", "P", int64(svrTagGhost), 1)
}

func TestSquadVehicleRepo_BorneDernierePasseEtFragsParCamp(t *testing.T) {
	b := baseVehicules(t)
	corpusVehicules(t, b.pdb)
	b.carnet.vider()

	got, err := NewSquadVehicleRepo(b.pdb, svrClassifier{}).LoadVehicleUsage(context.Background(),
		[]string{"m1", "m2", "m3", "m4", "m5"}, "P")
	if err != nil {
		t.Fatalf("LoadVehicleUsage : %v", err)
	}
	// Fenêtres : les lignes des cinq matchs demandés seulement (m1 : 5 lignes de la passe ancienne et
	// 3 de la courante ; m2, m3, m4 : une chacun = 11 ; m1 compte 9 événements) ; m9 a 13 lignes et 12
	// événements de plus, que rien ne doit faire voir.
	exigerFenetresBornees(t, b, "LoadVehicleUsage", 11, 2)

	var passes []string
	for _, p := range got.Passes {
		passes = append(passes, fmt.Sprintf("%s:%v:%v:%d", p.MatchID, p.Measured, p.FragsRead, p.FragsTotal))
	}
	if fmt.Sprint(passes) != "[m1:true:true:3 m2:true:true:0 m3:false:false:0 m4:true:false:0]" {
		t.Fatalf("passes = %v : la passe ancienne de m1 a survécu, ou m9 est lu", passes)
	}
	if p := got.Passes[0]; p.DocSchema != 71 {
		t.Errorf("passe m1 mal relue : %+v", p)
	}
	if len(got.Rows) != 2 {
		t.Fatalf("prises = %+v, attendu les deux lignes de la passe courante de m1", got.Rows)
	}
	r := got.Rows[0]
	if r.MatchID != "m1" || r.Camp != 0 || r.XUID != "P" || r.Family != "ghost" || r.Takes != 2 ||
		r.AboardMS != 5000 || r.Frags != 2 || r.Episodes != 2 {
		t.Errorf("prise mal relue : %+v", r)
	}
	// Frags d'engin de m1 par camp du tueur : 2 (camp 0), 1 (camp 1), 1 (tueur sans camp) ; rien
	// du bot, de la source non mesurée, de l'arme à feu ni de l'écrasement ; rien de m2 ni de m4.
	want := map[string]int{"0": 2, "1": 1, "nil": 1}
	vu := map[string]int{}
	for _, f := range got.Frags {
		if f.MatchID != "m1" {
			t.Errorf("frags lus pour %s, attendu m1 seul", f.MatchID)
		}
		k := "nil"
		if f.TeamID != nil {
			k = fmt.Sprint(*f.TeamID)
		}
		vu[k] += f.Frags
	}
	if fmt.Sprint(vu) != fmt.Sprint(want) {
		t.Errorf("frags d'engin par camp = %v, attendu %v", vu, want)
	}
	if len(got.EventsRead) != 1 || !got.EventsRead["m1"] {
		t.Errorf("matchs aux événements lus = %v, attendu m1 seul", got.EventsRead)
	}
	if len(got.PlayerTeam) != 4 || got.PlayerTeam["m1"] != 0 || got.PlayerTeam["m4"] != 0 {
		t.Errorf("camp du joueur = %v, attendu le camp 0 sur m1 à m4 (pas m5, pas m9)", got.PlayerTeam)
	}
}

// TestSquadVehicleRepo_TourelleEstUnEngin — une source de classe `turret` compte comme un frag
// d'engin (IsEngineFragClass, la définition de la Répartition des frags).
func TestSquadVehicleRepo_TourelleEstUnEngin(t *testing.T) {
	b := baseVehicules(t)
	poserLigneVehicule(t, b.pdb, passeMatch("m1", "p2", svrCourant, true, true, "", 3, 0))
	poserParticipant(t, b.pdb, "m1", "P", 0)
	poserMort(t, b.pdb, "m1", "P", int64(svrTagTurret), 3)
	got, err := NewSquadVehicleRepo(b.pdb, svrClassifier{}).LoadVehicleUsage(context.Background(), []string{"m1"}, "P")
	if err != nil || len(got.Frags) != 1 || got.Frags[0].Frags != 3 {
		t.Fatalf("frags de tourelle = %+v, err %v ; attendu 3", got.Frags, err)
	}
}

// TestSquadVehicleRepo_RegistreSansClasse_PasUnZero — un registre d'armes illisible ne reconnaît
// aucun frag d'engin : les matchs restent « non lus », jamais un zéro qui passerait pour mesuré.
func TestSquadVehicleRepo_RegistreSansClasse_PasUnZero(t *testing.T) {
	b := newBaseNotee(t) // metadata sans registre d'armes
	poserLigneVehicule(t, b.pdb, passeMatch("m1", "p2", svrCourant, true, true, "", 3, 0))
	poserMort(t, b.pdb, "m1", "P", int64(svrTagGhost), 3)
	got, err := NewSquadVehicleRepo(b.pdb, svrClassifier{}).LoadVehicleUsage(context.Background(), []string{"m1"}, "P")
	if err != nil {
		t.Fatalf("LoadVehicleUsage : %v", err)
	}
	if len(got.Frags) != 0 || len(got.EventsRead) != 0 {
		t.Errorf("frags %+v, événements lus %v ; attendu « non lu » sans registre de classes", got.Frags, got.EventsRead)
	}
	if len(got.Passes) != 1 {
		t.Errorf("la passe doit se lire quand même : %+v", got.Passes)
	}
}

// TestSquadVehicleRepo_SansClassificateur — les prises se lisent, aucun frag par camp ne sort.
func TestSquadVehicleRepo_SansClassificateur(t *testing.T) {
	b := baseVehicules(t)
	corpusVehicules(t, b.pdb)
	got, err := NewSquadVehicleRepo(b.pdb, nil).LoadVehicleUsage(context.Background(), []string{"m1"}, "P")
	if err != nil || len(got.Rows) != 2 || len(got.Frags) != 0 || len(got.EventsRead) != 0 {
		t.Errorf("sans classificateur : %d prises, %d frags, lus %v, err %v", len(got.Rows), len(got.Frags), got.EventsRead, err)
	}
}

// TestSquadVehicleRepo_ListeVide : aucune requête sans match.
func TestSquadVehicleRepo_ListeVide(t *testing.T) {
	b := baseVehicules(t)
	corpusVehicules(t, b.pdb)
	b.carnet.vider()
	got, err := NewSquadVehicleRepo(b.pdb, svrClassifier{}).LoadVehicleUsage(context.Background(), nil, "P")
	if err != nil || len(got.Passes) != 0 || len(got.Rows) != 0 {
		t.Errorf("liste vide : %+v, err %v", got, err)
	}
	if n := len(b.carnet.vider()); n != 0 {
		t.Errorf("%d requête(s) envoyée(s) pour une liste vide", n)
	}
}

// TestSquadVehicleRepo_TableAbsente : une base sans la table rend games.ErrCapabilityNotSupported.
func TestSquadVehicleRepo_TableAbsente(t *testing.T) {
	connector, err := duckdbdrv.NewConnector(":memory:", nil)
	if err != nil {
		t.Fatalf("NewConnector: %v", err)
	}
	raw := sql.OpenDB(connector)
	t.Cleanup(func() { _ = raw.Close(); _ = connector.Close() })
	shared := newTestDB(raw, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared)}
	_, err = NewSquadVehicleRepo(pdb, svrClassifier{}).LoadVehicleUsage(context.Background(), []string{"m1"}, "P")
	if !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("err = %v, attendu games.ErrCapabilityNotSupported", err)
	}
}
