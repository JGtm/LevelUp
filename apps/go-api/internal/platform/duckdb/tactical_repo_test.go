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
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	titlepkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
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

// newTacticalTestPlayerDB : shared `:memory:` migre (vues `_latest` comprises).
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
		XUID:         tacXUIDMoi,
		TitleSlug:    titlepkg.DefaultSlug,
	}
}

// tacExec joue une commande sur le shared de test.
func tacExec(t *testing.T, pdb *PlayerDB, query string, args ...any) {
	t.Helper()
	if _, err := pdb.Shared.Exec(context.Background(), query, args...); err != nil {
		t.Fatalf("exec %.60s...: %v", query, err)
	}
}

// tacMatch pose un match au registre et le participant `xuid` avec son issue.
func tacMatch(t *testing.T, pdb *PlayerDB, matchID, mapID string, start time.Time) {
	t.Helper()
	tacExec(t, pdb, `INSERT INTO match_registry
		(match_id, map_id, map_name, map_name_fr, start_time, start_time_utc, playlist_name, pair_name)
		VALUES (?, ?, ?, ?, ?, ?, 'Ranked Arena', 'Arena:Slayer')`,
		matchID, mapID, mapID+"_en", mapID+"_fr", start, start)
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
// approche — meme prudence que KillDistanceRepo.
func TestTacticalRepo_PositionPartielle_Ecartee(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)
	tacKill(t, pdb, "m2", tacXUIDMoi, tacXUIDAdv, 5000, true)
	tacPos(t, pdb, "m2", tacXUIDMoi, 5000, 1.0, 1.0, nil, nil)

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	for _, p := range got.Points {
		if p.MatchID == "m2" {
			t.Errorf("position partielle comptee a tort : %+v", p)
		}
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

// TestTacticalRepo_FiltreIssue : le filtre de l'Explorateur borne l'univers. Une
// issue « defaite » ne retient que m2 — et m1, muet du coup, disparait avec ses
// deux points.
func TestTacticalRepo_FiltreIssue(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	perdu := "loss"
	q := tacQuery(tacCarteA)
	q.Filtre = &domain.MatchFilterSpec{Outcome: &perdu}

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m2"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v", matchIDs(got.Univers.Matchs), want)
	}
	if len(got.Points) != 0 {
		t.Errorf("points = %+v, want 0 (m1 est hors filtre)", got.Points)
	}
}

// TestTacticalRepo_FiltreDate : les bornes de date passent par le fragment
// timezone canonique du builder partage — un match anterieur sort de l'univers.
func TestTacticalRepo_FiltreDate(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	depuis := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC) // apres m1, avant m2
	q := tacQuery(tacCarteA)
	q.Filtre = &domain.MatchFilterSpec{DateFrom: &depuis}

	got, err := NewTacticalRepo(pdb).KillPositions(context.Background(), q)
	if err != nil {
		t.Fatalf("KillPositions: %v", err)
	}
	if want := []string{"m2"}; !egales(matchIDs(got.Univers.Matchs), want) {
		t.Fatalf("univers = %v, want %v", matchIDs(got.Univers.Matchs), want)
	}
}

// TestTacticalRepo_MapsPlayed : cartes jouees, comptes et decomposition V/D, dans
// l'ordre matchs decroissants. m4 (joue par un tiers) n'y figure pas.
func TestTacticalRepo_MapsPlayed(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	seedTacticalCorpus(t, pdb)

	rows, err := NewTacticalRepo(pdb).MapsPlayed(context.Background(),
		domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("cartes = %d, want 2 : %+v", len(rows), rows)
	}
	if rows[0].MapID != tacCarteA || rows[0].Matchs != 2 {
		t.Fatalf("premiere carte = %+v, want %s a 2 matchs (tri matchs decroissants)", rows[0], tacCarteA)
	}
	if rows[0].Victoires != 1 || rows[0].Defaites != 1 {
		t.Errorf("carte A = %d V / %d D, want 1/1", rows[0].Victoires, rows[0].Defaites)
	}
	if rows[0].MapName != tacCarteA+"_en" || rows[0].MapNameFR != tacCarteA+"_fr" {
		t.Errorf("libelles de carte = %q / %q", rows[0].MapName, rows[0].MapNameFR)
	}
	if rows[1].MapID != tacCarteB || rows[1].Matchs != 1 || rows[1].Victoires != 1 {
		t.Errorf("seconde carte = %+v, want %s a 1 match / 1 V", rows[1], tacCarteB)
	}
}

// TestTacticalRepo_AucuneDonnee_ZeroLigneZeroErreur : un joueur sans aucun match
// est l'etat NOMINAL d'un compte neuf, pas une panne.
func TestTacticalRepo_AucuneDonnee_ZeroLigneZeroErreur(t *testing.T) {
	pdb := newTacticalTestPlayerDB(t)
	repo := NewTacticalRepo(pdb)

	rows, err := repo.MapsPlayed(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi})
	if err != nil || len(rows) != 0 {
		t.Errorf("MapsPlayed = %+v / %v, want 0 ligne, nil", rows, err)
	}
	pos, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA))
	if err != nil || len(pos.Univers.Matchs) != 0 || len(pos.Points) != 0 {
		t.Errorf("KillPositions = %+v / %v, want vide, nil", pos, err)
	}
}

// TestTacticalRepo_EntreesVides_Refus : jamais de scan complet — un xuid ou une
// carte vides sont un refus, pas un balayage de shared.kill_positions.
func TestTacticalRepo_EntreesVides_Refus(t *testing.T) {
	repo := NewTacticalRepo(newTacticalTestPlayerDB(t))
	if _, err := repo.MapsPlayed(context.Background(), domain.TacticalQuery{}); err == nil {
		t.Error("MapsPlayed sans xuid : attendu un refus")
	}
	if _, err := repo.KillPositions(context.Background(), domain.TacticalQuery{MapID: tacCarteA}); err == nil {
		t.Error("KillPositions sans xuid : attendu un refus")
	}
	if _, err := repo.KillEvents(context.Background(), domain.TacticalQuery{PlayerXUID: tacXUIDMoi}); err == nil {
		t.Error("KillEvents sans carte : attendu un refus")
	}
}

// TestTacticalRepo_TablesAbsentes_Capability : un shared sans les tables du film
// (titre sans decodeur) rend ErrCapabilityNotSupported — 503 propre en bout de
// chaine, jamais un 500.
func TestTacticalRepo_TablesAbsentes_Capability(t *testing.T) {
	sharedSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared mem: %v", err)
	}
	t.Cleanup(func() { _ = sharedSQL.Close() })
	// Le strict minimum pour que l'univers se lise : pas de kill_positions, pas de
	// match_kill_events — exactement la situation d'un titre sans decodeur de film.
	for _, ddl := range []string{
		`CREATE TABLE match_registry (match_id VARCHAR, map_id VARCHAR, map_name VARCHAR,
			map_name_fr VARCHAR, start_time TIMESTAMP, start_time_utc TIMESTAMPTZ,
			playlist_name VARCHAR, pair_name VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
			team_id INTEGER, outcome INTEGER)`,
		`INSERT INTO match_registry VALUES ('m1', 'map_streets', 'a', 'a', NULL, NULL, NULL, NULL)`,
		`INSERT INTO match_participants VALUES ('m1', '` + tacXUIDMoi + `', 'moi', 0, 2)`,
	} {
		if _, err := sharedSQL.Exec(ddl); err != nil {
			t.Fatalf("ddl minimale: %v", err)
		}
	}
	shared := newTestDB(sharedSQL, ":memory:")
	pdb := &PlayerDB{Shared: shared, SharedReader: LegacySharedReader(shared),
		XUID: tacXUIDMoi, TitleSlug: titlepkg.DefaultSlug}

	repo := NewTacticalRepo(pdb)
	if _, err := repo.KillPositions(context.Background(), tacQuery(tacCarteA)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("KillPositions sur schema sans film: err = %v, want ErrCapabilityNotSupported", err)
	}
	if _, err := repo.KillEvents(context.Background(), tacQuery(tacCarteA)); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Errorf("KillEvents sur schema sans film: err = %v, want ErrCapabilityNotSupported", err)
	}
}
