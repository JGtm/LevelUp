// Package duckdb — tactical_repo_test.go : tests TacticalRepo.
//
// Round-trip sur DB `:memory:` avec les VRAIES migrations shared — donc les vraies
// vues `_latest` (`kill_positions_latest`, `match_kill_events_latest`). Aucune DDL
// recopiee : une DDL de test recopiee derive de la vraie sans que rien ne le dise,
// et le test reste vert sur un schema qui n'existe plus.
//
// SANS TAG DE BUILD, contrairement a kill_distance_repo_test.go : le gate de la
// phase 2 joue `go test ./internal/platform/duckdb/...` sans `-tags=integration`,
// et un test derriere un tag que le gate ne pose pas ne garde rien. Le precedent
// suivi est achievements_repo_test.go / csr_thresholds_repo_test.go, qui montent
// eux aussi les vraies migrations sans tag.
//
// Ce que ces tests verrouillent, dans l'ordre d'importance :
//
//  1. L'UNIVERS contient les matchs retenus SANS point (le defaut P0 de la
//     phase 1 : un match muet doit compter au denominateur « par match ») ;
//  2. un match HORS filtre (autre carte, autre joueur, issue filtree) n'entre ni
//     dans l'univers ni dans les points ni dans les evenements ;
//  3. une position PARTIELLE (un seul cote connu) est ecartee, jamais approchee ;
//  4. un double kill au meme (tueur, instant) est ecarte EN ENTIER (la position de
//     victime ne peut pas etre attribuee) ;
//  5. une passe NON PUBLIABLE est ecartee (attribution PAR LIGNE) ;
//  6. les equipes sont lues PAR MATCH ;
//  7. tables absentes / xuid vide / carte vide degradent proprement.
package duckdb

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

const (
	tacXUIDMoi  = "xuid(2533274000000101)"
	tacXUIDAmi  = "xuid(2533274000000102)"
	tacXUIDAdv  = "xuid(2533274000000103)"
	tacXUIDTier = "xuid(2533274000000104)"

	tacCarteA = "map_streets"
	tacCarteB = "map_recharge"
)

// newTacticalTestPlayerDB : shared `:memory:` migre (vues `_latest` comprises) +
// une metadata portant `asset_translations`.
//
// LA METADATA N'EST PAS UN ORNEMENT (correction R3) : `match_registry.map_name_fr`
// est SYSTEMATIQUEMENT NULLE en prod, et le nom FR d'une carte se resout par
// `metadata.asset_translations`. Semer la colonne du registre — ce que faisait la
// fixture d'origine — fabriquait une donnee qui n'existe nulle part et rendait le
// test aveugle au defaut qu'il etait cense couvrir.
func newTacticalTestPlayerDB(t *testing.T) *PlayerDB {
	t.Helper()
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	_ = migration.All()
	if err := migration.RunForDB(sharedSQL, migration.TargetShared); err != nil {
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared := newTestDB(sharedSQL, ":memory:")
	return &PlayerDB{
		Shared:       shared,
		SharedReader: LegacySharedReader(shared),
		Metadata:     newTacticalTestMetadata(t),
		XUID:         tacXUIDMoi,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// newTacticalTestMetadata : metadata `:memory:` portant `asset_translations`, la
// SEULE source FR fiable des noms de carte. Meme forme de table et meme facon de
// semer que engagement_map_fr_test.go.
func newTacticalTestMetadata(t *testing.T) *DB {
	t.Helper()
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	ctx := context.Background()
	seed := []string{
		`CREATE TABLE asset_translations (
			asset_id    VARCHAR,
			asset_type  VARCHAR,
			lang        VARCHAR,
			name        VARCHAR,
			description VARCHAR,
			fetched_at  TIMESTAMP
		)`,
		`INSERT INTO asset_translations VALUES ('` + tacCarteA + `','map','fr-FR','Les Rues','',now())`,
		// Bruit : une playlist du meme asset_id ne doit PAS matcher asset_type='map'.
		`INSERT INTO asset_translations VALUES ('` + tacCarteA + `','playlist','fr-FR','NE PAS PRENDRE','',now())`,
		// tacCarteB n'a AUCUNE traduction : son nom FR doit rester vide, et c'est
		// l'etat NOMINAL d'une carte non traduite — pas une panne.
	}
	for _, q := range seed {
		if _, err := meta.Exec(ctx, q); err != nil {
			t.Fatalf("seed metadata %q: %v", q, err)
		}
	}
	return meta
}

// tacExec joue une commande sur le shared de test.
func tacExec(t *testing.T, pdb *PlayerDB, query string, args ...any) {
	t.Helper()
	if _, err := pdb.Shared.Exec(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %.60s...: %v", query, err)
	}
}

// tacMatch pose un match ORDINAIRE au registre (aucun game_variant_id).
func tacMatch(t *testing.T, pdb *PlayerDB, matchID, mapID string, start time.Time) {
	t.Helper()
	tacMatchVariant(t, pdb, matchID, mapID, start, nil)
}

// tacMatchVariant pose un match au registre avec son `game_variant_id` — nil pour
// un mode ordinaire, un GUID de Campagne pour exercer le masquage read-side.
func tacMatchVariant(t *testing.T, pdb *PlayerDB, matchID, mapID string, start time.Time, variantID any) {
	t.Helper()
	// map_name_fr N'EST PAS SEMEE : elle est systematiquement NULLE en prod, et le
	// nom FR se resout par metadata.asset_translations (cf. newTacticalTestMetadata).
	tacExec(t, pdb, `INSERT INTO match_registry
		(match_id, map_id, map_name, start_time, start_time_utc, playlist_name, pair_name, game_variant_id)
		VALUES (?, ?, ?, ?, ?, 'Ranked Arena', 'Arena:Slayer', ?)`,
		matchID, mapID, mapID+"_en", start, start, variantID)
}

// tacParticipant pose un participant (equipe + issue) sur un match.
func tacParticipant(t *testing.T, pdb *PlayerDB, matchID, xuid string, team, outcome int) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES (?, ?, ?, ?, ?)`, matchID, xuid, xuid, team, outcome)
}

// tacKill pose une mort dans match_kill_events. killerXUID / victimXUID vides =
// NULL (bot / environnement).
func tacKill(t *testing.T, pdb *PlayerDB, matchID, killerXUID, victimXUID string, timeMS int, publishable bool) {
	t.Helper()
	var killer, victim any
	if killerXUID != "" {
		killer = killerXUID
	}
	if victimXUID != "" {
		victim = victimXUID
	}
	tacExec(t, pdb, `INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms,
		 victim_gamertag, victim_xuid, feed_killer_gamertag, feed_killer_xuid,
		 feed_present, assist_known, read_path, read_origin)
		VALUES (?, 'pass_v1', 'rev_test', ?, ?, ?, ?, ?, ?, TRUE, FALSE, ?, 'credit-concordant')`,
		matchID, publishable, timeMS, victimXUID, victim, killerXUID, killer,
		killscope.ReadPathFilmWalk)
}

// tacPos pose une position monde. Les coordonnees acceptent nil (position
// partielle) : signature `any` pour cela.
func tacPos(t *testing.T, pdb *PlayerDB, matchID, killerXUID string, timeMS int, kx, ky, vx, vy any) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO kill_positions
		(match_id, killer_xuid, time_ms, killer_x, killer_y, killer_z, victim_x, victim_y, victim_z)
		VALUES (?, ?, ?, ?, ?, 0.0, ?, ?, 0.0)`,
		matchID, killerXUID, timeMS, kx, ky, vx, vy)
}

// seedTacticalCorpus monte LE corpus de reference des tests ci-dessous :
//
//	m1  carte A, VICTOIRE  — 2 morts mesurees (moi tue l'adversaire, l'adversaire me tue)
//	m2  carte A, DEFAITE   — AUCUNE position mesuree : le match MUET de l'univers
//	m3  carte B, VICTOIRE  — 1 mort mesuree : hors carte A
//	m4  carte A            — je n'y ai pas joue : hors univers, et son evenement
//	                         (deux tiers entre eux) ne doit apparaitre nulle part
func seedTacticalCorpus(t *testing.T, pdb *PlayerDB) {
	t.Helper()
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	tacMatch(t, pdb, "m1", tacCarteA, base)
	tacParticipant(t, pdb, "m1", tacXUIDMoi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "m1", tacXUIDAmi, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "m1", tacXUIDAdv, 1, domain.OutcomeLoss)
	tacKill(t, pdb, "m1", tacXUIDMoi, tacXUIDAdv, 1000, true)
	tacPos(t, pdb, "m1", tacXUIDMoi, 1000, 2.0, 2.0, 4.0, 4.0)
	tacKill(t, pdb, "m1", tacXUIDAdv, tacXUIDMoi, 3000, true)
	tacPos(t, pdb, "m1", tacXUIDAdv, 3000, 6.0, 6.0, 8.0, 8.0)

	tacMatch(t, pdb, "m2", tacCarteA, base.Add(time.Hour))
	tacParticipant(t, pdb, "m2", tacXUIDMoi, 0, domain.OutcomeLoss)
	tacParticipant(t, pdb, "m2", tacXUIDAdv, 1, domain.OutcomeWin)

	tacMatch(t, pdb, "m3", tacCarteB, base.Add(2*time.Hour))
	tacParticipant(t, pdb, "m3", tacXUIDMoi, 0, domain.OutcomeWin)
	tacKill(t, pdb, "m3", tacXUIDMoi, tacXUIDAdv, 1000, true)
	tacPos(t, pdb, "m3", tacXUIDMoi, 1000, 50.0, 50.0, 52.0, 52.0)

	tacMatch(t, pdb, "m4", tacCarteA, base.Add(3*time.Hour))
	tacParticipant(t, pdb, "m4", tacXUIDTier, 0, domain.OutcomeWin)
	tacParticipant(t, pdb, "m4", tacXUIDAdv, 1, domain.OutcomeLoss)
	tacKill(t, pdb, "m4", tacXUIDTier, tacXUIDAdv, 1000, true)
	tacPos(t, pdb, "m4", tacXUIDTier, 1000, 9.0, 9.0, 9.5, 9.5)
}

func tacQuery(mapID string) domain.TacticalQuery {
	return domain.TacticalQuery{PlayerXUID: tacXUIDMoi, MapID: mapID}
}

func matchIDs(matchs []domain.TacticalMatch) []string {
	out := make([]string, 0, len(matchs))
	for _, m := range matchs {
		out = append(out, m.MatchID)
	}
	return out
}

func egales(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestTacticalRepo_UniversPorteLeMatchMuet : LE test du lot. m2 n'a AUCUNE
// position mesuree et doit pourtant figurer dans l'univers — c'est lui le
// denominateur qui empeche la lecture signee de peindre une zone gagnante la ou
// il n'y a que des matchs muets d'un cote (defaut P0 de la phase 1).
func TestTacticalRepo_UniversPorteLeMatchMuet(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m1", "m2"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v (m2 est MUET mais RETENU ; m3 est une autre carte ; m4 n'est pas a moi)",
			matchIDs(got.Univers.Matchs), want)
	}
	for _, m := range got.Univers.Matchs {
		if m.MatchID == "m2" && m.Outcome != domain.OutcomeLoss {
			t.Errorf("m2 Outcome = %d, want OutcomeLoss (%d)", m.Outcome, domain.OutcomeLoss)
		}
	}
	if len(got.Points) != 2 {
		t.Fatalf("points = %d, want 2 (les deux morts de m1) : %+v", len(got.Points), got.Points)
	}
	if got.Points[0].KillerXUID != tacXUIDMoi || got.Points[0].VictimXUID != tacXUIDAdv {
		t.Errorf("premier point = %+v, want tueur=moi victime=adversaire", got.Points[0])
	}
	if got.Points[0].KillerX != 2.0 || got.Points[0].VictimY != 4.0 {
		t.Errorf("coordonnees du premier point = %+v, want tueur (2,2) victime (4,4)", got.Points[0])
	}
}

// TestTacticalRepo_EquipesParMatch : la composition est lue PAR MATCH — c'est
// elle qui tranche l'axe « moi / escouade / adversaires » cote service.
func TestTacticalRepo_EquipesParMatch(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	m1 := got.Univers.Equipes["m1"]
	if len(m1) != 3 {
		t.Fatalf("equipes de m1 = %v, want 3 joueurs", m1)
	}
	if m1[tacXUIDMoi] != 0 || m1[tacXUIDAmi] != 0 || m1[tacXUIDAdv] != 1 {
		t.Errorf("equipes de m1 = %v, want moi/ami en 0 et adversaire en 1", m1)
	}
	if _, present := got.Univers.Equipes["m4"]; present {
		t.Errorf("m4 est hors univers : ses equipes ne doivent pas etre chargees (%v)", got.Univers.Equipes["m4"])
	}
}

// TestTacticalRepo_PositionPartielle_Ecartee : un seul cote connu n'est jamais
// approche — meme prudence que KillDistanceRepo. LES DEUX SENS sont exerces : la
// victime manquante (T7 : et le TUEUR manquant), parce qu'une garde qui ne
// couvrirait qu'un cote laisserait passer l'autre sans que rien ne le dise.
func TestTacticalRepo_PositionPartielle_Ecartee(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	// Position de VICTIME manquante.
	tacKill(t, pdb, "m2", tacXUIDMoi, tacXUIDAdv, 5000, true)
	tacPos(t, pdb, "m2", tacXUIDMoi, 5000, 1.0, 1.0, nil, nil)
	// Position de TUEUR manquante (T7).
	tacKill(t, pdb, "m2", tacXUIDMoi, tacXUIDAdv, 6000, true)
	tacPos(t, pdb, "m2", tacXUIDMoi, 6000, nil, nil, 3.0, 3.0)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	for _, p := range got.Points {
		if p.MatchID == "m2" {
			t.Errorf("position partielle comptee a tort : %+v", p)
		}
	}
	if len(got.Points) != 2 {
		t.Errorf("points = %d, want 2 (les deux morts COMPLETES de m1)", len(got.Points))
	}
}

// TestTacticalRepo_VictimeBot_ServieEnIdentiteVide : une position dont le kill-event
// n'a pas de victime resolue (`victim_xuid` NULL — victime BOT) REMONTE, avec une
// identite vide et sans erreur.
//
// Ce n'est pas un cas theorique : le producteur natif de Halo 5 ne pose que le
// tueur (games/halo_5/ingest/positions.go), sa ligne peut donc joindre un kill-event
// a victime bot. L'ecarter sous-compterait les kills du joueur ; la ranger dans un
// axe serait une invention — c'est l'appelant qui tranche, sur une identite vide.
func TestTacticalRepo_VictimeBot_ServieEnIdentiteVide(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	tacMatch(t, pdb, "m1", tacCarteA, base)
	tacParticipant(t, pdb, "m1", tacXUIDMoi, 0, domain.OutcomeWin)
	// Victime BOT : victim_xuid NULL (tacKill pose NULL sur une chaine vide).
	tacKill(t, pdb, "m1", tacXUIDMoi, "", 1000, true)
	tacPos(t, pdb, "m1", tacXUIDMoi, 1000, 2.0, 2.0, 4.0, 4.0)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(got.Points) != 1 {
		t.Fatalf("points = %d, want 1 (la victime bot ne doit pas faire disparaitre le kill) : %+v",
			len(got.Points), got.Points)
	}
	if got.Points[0].VictimXUID != "" {
		t.Errorf("VictimXUID = %q, want vide", got.Points[0].VictimXUID)
	}
	if got.Points[0].KillerXUID != tacXUIDMoi || got.Points[0].KillerX != 2.0 {
		t.Errorf("le tueur et sa position doivent etre intacts : %+v", got.Points[0])
	}
}

// TestTacticalRepo_DoubleKillMemeInstant_Ecarte : kill_positions n'a qu'UNE
// position de victime par (match, tueur, instant) ; deux morts au meme instant la
// rendraient ambigue. Le groupe entier sort — attribuer la position a la mauvaise
// victime rangerait le point du mauvais cote de l'axe « qui ».
func TestTacticalRepo_DoubleKillMemeInstant_Ecarte(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	// Seconde victime au MEME instant que la premiere mort de m1 (t=1000).
	tacKill(t, pdb, "m1", tacXUIDMoi, tacXUIDTier, 1000, true)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(got.Points) != 1 {
		t.Fatalf("points = %d, want 1 (le double kill de t=1000 sort en entier) : %+v", len(got.Points), got.Points)
	}
	if got.Points[0].KillerXUID != tacXUIDAdv {
		t.Errorf("le point restant devrait etre la mort de t=3000 : %+v", got.Points[0])
	}
}

// TestTacticalRepo_NonPublishable_Ecartee : la lecture est une attribution PAR
// LIGNE (ce point est le mien ou non) — une passe juste seulement en agregat ne
// peut pas la servir.
func TestTacticalRepo_NonPublishable_Ecartee(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	tacMatch(t, pdb, "m1", tacCarteA, time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	tacParticipant(t, pdb, "m1", tacXUIDMoi, 0, domain.OutcomeWin)
	tacKill(t, pdb, "m1", tacXUIDMoi, tacXUIDAdv, 1000, false)
	tacPos(t, pdb, "m1", tacXUIDMoi, 1000, 2.0, 2.0, 4.0, 4.0)

	repo := NewTacticalRepo(pdb)
	pos, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if len(pos.Points) != 0 {
		t.Errorf("passe non publiable comptee a tort : %+v", pos.Points)
	}
	ev, err := repo.KillEvents(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if len(ev.Events) != 0 {
		t.Errorf("evenement non publiable compte a tort : %+v", ev.Events)
	}
}

// TestTacticalRepo_KillEvents_TiersHorsFiltreExclu : l'evenement de m4 (deux tiers
// entre eux, sur la meme carte) ne doit apparaitre nulle part — je n'ai pas joue
// ce match, il n'est pas dans mon univers.
func TestTacticalRepo_KillEvents_TiersHorsFiltreExclu(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	got, err := NewTacticalRepo(pdb).KillEvents(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillEvents: %v", err)
	}
	if want := []string{"m1", "m2"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v", matchIDs(got.Univers.Matchs), want)
	}
	if len(got.Events) != 2 {
		t.Fatalf("evenements = %d, want 2 (les deux morts de m1) : %+v", len(got.Events), got.Events)
	}
	for _, e := range got.Events {
		if e.MatchID != "m1" {
			t.Errorf("evenement hors univers servi : %+v", e)
		}
		if e.KillerXUID == tacXUIDTier || e.VictimXUID == tacXUIDTier {
			t.Errorf("evenement d'un match tiers servi : %+v", e)
		}
	}
	if got.Events[0].TimeMs != 1000 || got.Events[1].TimeMs != 3000 {
		t.Errorf("evenements non ordonnes par instant : %+v", got.Events)
	}
}
