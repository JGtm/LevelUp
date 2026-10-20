//go:build integration

package killcollector

// placement_des_vies_integration_test.go — LE PLACEMENT DES VIES ECRIT PAR LE COLLECTEUR, sur une
// vraie base migree (plan `.ai/V7.5/PLAN_EMPRISE_VIES_2026-09-28.md`, lot V2).
//
//	TestProjeterPlacementDesVies_*   synthetique, tourne partout : la ligne ecrite, la revision,
//	                                 la portee, la passe qui remplace la precedente.
//	TestEmpriseV2Temoin              V2.7 : de VRAIS films du 22/09 passes par `CollectMatch`
//	                                 (cablage de production), les lignes ecrites egalent le calcul
//	                                 pur de V1.3 nourri INDEPENDAMMENT (faits de la cuisson par
//	                                 `ReplayFactsRepo`, catalogues relus). Saute sans ses donnees :
//
//	EMPRISE_V2_FILMS=<match_id,...> EMPRISE_V2_DB=<copie shared> EMPRISE_V2_CACHE=<data/cache> \
//	  go test -tags=integration ./internal/sync/killcollector/ -run '^TestEmpriseV2Temoin$' -v -count=1
//
// La copie de base n'est JAMAIS ecrite : elle est recopiee dans le dossier temporaire du test.

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	halomigrations "levelup/go-api/internal/games/halo_infinite/migrations"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/testutil"
)

// writerSur : un writer qui rend toujours la meme base (le test est seul a l'ecrire).
func writerSur(db *sql.DB) persist.SharedWriterFn {
	return func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil }
}

// lignePlacement : une ligne de `match_life_placement_latest`, telle que la base la rend.
type lignePlacement struct {
	xuid, rev                                                        string
	debut, fin, duree, mesure, porteur, aTerre, nonSitue, coequipier int64
	mediane, radar                                                   *float64
	horsRadar                                                        *int64
	frags                                                            int
}

func lirePlacement(t *testing.T, db *sql.DB, matchID string) []lignePlacement {
	t.Helper()
	rows, err := db.Query(`SELECT xuid, decoder_rev, start_ms, end_ms, duration_ms, measured_ms,
		carrier_ms, team_down_ms, unplaced_ms, teammate_unplaced_ms, median_m, radar_m, beyond_ms, kills
		FROM match_life_placement_latest WHERE match_id = ? ORDER BY xuid, start_ms`, matchID)
	if err != nil {
		t.Fatalf("lecture du placement %s : %v", matchID, err)
	}
	defer rows.Close() //nolint:errcheck
	var out []lignePlacement
	for rows.Next() {
		var l lignePlacement
		if err := rows.Scan(&l.xuid, &l.rev, &l.debut, &l.fin, &l.duree, &l.mesure, &l.porteur,
			&l.aTerre, &l.nonSitue, &l.coequipier, &l.mediane, &l.radar, &l.horsRadar, &l.frags); err != nil {
			t.Fatalf("scan du placement %s : %v", matchID, err)
		}
		out = append(out, l)
	}
	return out
}

// TestProjeterPlacementDesVies_EcritUneLigneParVie — la vie du materiau synthetique, sa revision,
// son frag rattache, sa portee ; puis une seconde passe, sans portee, qui REMPLACE la premiere
// dans la vue et se compte « sans portee ».
func TestProjeterPlacementDesVies_EcritUneLigneParVie(t *testing.T) {
	db := baseBacklog(t)
	c := (&KillSourceCollector{acquireShared: writerSur(db)}).AvecPorteeDuRadar(
		func(v string) (float64, bool) { return 18, v == "Team Slayer:Arena" })
	mat := materiauAvecUneVie()
	vie := mat.registre.ViesNommees()[0]
	ids := MatchIdentities{Equipes: map[string]int{"111": 0, "222": 1}, Variante: "Team Slayer:Arena"}
	journal := persist.KillSourceBatch{Publishable: true, Deaths: []persist.KillEventInsert{
		{TimeMS: int(vie.DebutMS) + 100, FeedKillerXUID: "111", VictimXUID: "222"},
	}}

	c.projeterPlacementDesVies(context.Background(), "m1", mat, ids, journal)
	lignes := lirePlacement(t, db, "m1")
	if len(lignes) != 1 {
		t.Fatalf("%d lignes, attendu 1 (une par vie nommee)", len(lignes))
	}
	l := lignes[0]
	if l.xuid != "111" || l.rev != PlacementRev || l.debut != vie.DebutMS || l.fin != vie.FinMS ||
		l.duree != vie.FinMS-vie.DebutMS || l.frags != 1 {
		t.Fatalf("ligne = %+v, vie = %+v", l, vie)
	}
	// SEUL DANS SON CAMP : tous les instants sont « equipe a terre », aucune mediane, 0 ms hors
	// radar sur une portee CONNUE (et non NULL).
	if l.mesure != 0 || l.aTerre == 0 || l.mediane != nil || l.radar == nil || *l.radar != 18 ||
		l.horsRadar == nil || *l.horsRadar != 0 {
		t.Fatalf("ligne = %+v : attendu equipe a terre, portee 18, hors radar 0", l)
	}

	avant := observability.LoadCounter(metricPlacementSansPortee)
	ids.Variante = "Super Fiesta:Fiesta"
	c.projeterPlacementDesVies(context.Background(), "m1", mat, ids, journal)
	lignes = lirePlacement(t, db, "m1")
	if len(lignes) != 1 || lignes[0].radar != nil || lignes[0].horsRadar != nil {
		t.Fatalf("seconde passe : %+v — attendu UNE ligne (la passe remplace), sans portee", lignes)
	}
	if got := observability.LoadCounter(metricPlacementSansPortee) - avant; got != 1 {
		t.Fatalf("%s a bouge de %d, attendu 1", metricPlacementSansPortee, got)
	}
}

// TestProjeterPlacementDesVies_PontNonPubliable_EcritLesVies — lot V2b.2 (decision V1 amendee le
// 2026-09-29) : un pont slot->xuid non publiable ECRIT une ligne par vie, entiere « non situee »
// (la grille fermee de la vie), rien de mesure, la portee recopiee, le frag rattache — la ligne
// que le persister accepte et que `matchsAJour` reconnait (convergence du rattrapage).
func TestProjeterPlacementDesVies_PontNonPubliable_EcritLesVies(t *testing.T) {
	db := baseBacklog(t)
	c := (&KillSourceCollector{acquireShared: writerSur(db)}).AvecPorteeDuRadar(
		func(string) (float64, bool) { return 18, true })
	pos := positionsDUneVie()
	mat := materiauDIsolement{registre: registreDeTest(pos, 1), positions: pos}
	if mat.registre.PontPubliable() {
		t.Fatal("fixture : le pont devait etre refuse")
	}
	vie := mat.registre.ViesNommees()[0]
	ids := MatchIdentities{Equipes: map[string]int{"111": 0, "222": 1}, Variante: "CTF:Arena"}
	journal := persist.KillSourceBatch{Publishable: true, Deaths: []persist.KillEventInsert{
		{TimeMS: int(vie.DebutMS) + 100, FeedKillerXUID: "111", VictimXUID: "222"},
	}}
	avant := observability.LoadCounter(metricPlacementPontRefuse)

	c.projeterPlacementDesVies(context.Background(), "m1", mat, ids, journal)
	lignes := lirePlacement(t, db, "m1")
	if len(lignes) != 1 {
		t.Fatalf("%d lignes, attendu 1 (une par vie nommee, pont refuse compris)", len(lignes))
	}
	l := lignes[0]
	grille := ((vie.FinMS-vie.DebutMS)/100 + 1) * 100
	if l.rev != PlacementRev || l.mesure != 0 || l.nonSitue != grille || l.porteur != 0 ||
		l.aTerre != 0 || l.coequipier != 0 || l.mediane != nil || l.frags != 1 {
		t.Fatalf("ligne = %+v : attendu %d ms non situees, rien de mesure, 1 frag", l, grille)
	}
	if l.radar == nil || *l.radar != 18 || l.horsRadar == nil || *l.horsRadar != 0 {
		t.Fatalf("ligne = %+v : attendu portee 18 et 0 ms hors radar", l)
	}
	if got := observability.LoadCounter(metricPlacementPontRefuse) - avant; got != 1 {
		t.Fatalf("%s a bouge de %d, attendu 1", metricPlacementPontRefuse, got)
	}
}

// ── V2.7 : le temoin sur de vrais films ────────────────────────────────────────────────────

// v2Env : les donnees du temoin, ou un saut.
func v2Env(t *testing.T) (films []string, base, cache string) {
	t.Helper()
	for _, f := range strings.Split(os.Getenv("EMPRISE_V2_FILMS"), ",") {
		if f = strings.TrimSpace(f); f != "" {
			films = append(films, f)
		}
	}
	base, cache = os.Getenv("EMPRISE_V2_DB"), os.Getenv("EMPRISE_V2_CACHE")
	if len(films) == 0 || base == "" || cache == "" {
		t.Skip("EMPRISE_V2_FILMS, EMPRISE_V2_DB et EMPRISE_V2_CACHE requis : temoin saute")
	}
	return films, base, cache
}

// v2CopieMigree recopie la base dans le dossier du test et la migre (la table neuve y nait).
func v2CopieMigree(t *testing.T, source string) *sql.DB {
	t.Helper()
	cible := filepath.Join(t.TempDir(), "shared.duckdb")
	src, err := os.Open(source)
	if err != nil {
		t.Fatalf("copie de base : %v", err)
	}
	defer src.Close() //nolint:errcheck
	dst, err := os.Create(cible)
	if err != nil {
		t.Fatalf("copie de base : %v", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("copie de base : %v", err)
	}
	if err := dst.Close(); err != nil {
		t.Fatalf("copie de base : %v", err)
	}
	db, err := sql.Open("duckdb", cible)
	if err != nil {
		t.Fatalf("ouverture de la copie : %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	migration.SetTitleStepsProvider(halomigrations.StepsFor)
	if err := migration.RunForDB(db, migration.TargetShared); err != nil {
		t.Fatalf("migration de la copie : %v", err)
	}
	return db
}

// v2Portee : la portee du radar du titre, lue INDEPENDAMMENT de la capture (le fichier relu par
// `LoadRegulationFromFile`), resolue par le helper unique `mappings.PorteeDuRadar` — la MEME
// resolution que la lecture et l'ecriture (lot V2b).
func v2Portee(t *testing.T, repoRoot string) PorteeDuRadar {
	t.Helper()
	dir := title.NewPathResolver(repoRoot).TitleMappingsDir(title.DefaultSlug)
	reg, err := mappings.LoadRegulationFromFile(filepath.Join(dir, "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation.toml : %v", err)
	}
	table := reg.RadarRangeMap()
	return func(v string) (float64, bool) { return mappings.PorteeDuRadar(table, v) }
}

func TestEmpriseV2Temoin(t *testing.T) {
	films, base, cache := v2Env(t)
	repoRoot, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	db := v2CopieMigree(t, base)
	portee := v2Portee(t, repoRoot)
	// LA PORTEE N'EST PAS INJECTEE PAR LE TEST (lot V2b) : elle vient de la capture, comme en
	// production. `v2CritereDeLaPortee` la confronte a la lecture independante du test.
	col := v0Collecteur(t, v0Env{cac: cache}, db, db)
	ctx := context.Background()
	for _, id := range films {
		outcome, _, err := col.CollectMatch(ctx, id)
		if err != nil || outcome != OutcomeWritten {
			t.Fatalf("%s : CollectMatch = %s, %v", id, outcome, err)
		}
		ecrites := lirePlacement(t, db, id)
		v2CritereDesVies(t, db, id, ecrites)
		v2CritereDeLaPortee(t, db, id, ecrites, portee)
		pures := v2CalculPur(t, ctx, col, db, id, repoRoot, portee)
		v2Comparer(t, id, ecrites, pures)
	}
}

// v2CritereDesVies : une ligne de placement par vie de `match_lives_latest`, meme revision.
func v2CritereDesVies(t *testing.T, db *sql.DB, id string, ecrites []lignePlacement) {
	t.Helper()
	var vies int
	if err := db.QueryRow(`SELECT count(*) FROM match_lives_latest WHERE match_id = ?`, id).Scan(&vies); err != nil {
		t.Fatalf("%s : match_lives_latest : %v", id, err)
	}
	if len(ecrites) == 0 || len(ecrites) != vies {
		t.Fatalf("%s : %d lignes de placement pour %d vies", id, len(ecrites), vies)
	}
	var porteur, mesure int64
	nonMesurees := 0
	for _, l := range ecrites {
		if l.rev != PlacementRev {
			t.Fatalf("%s : revision %q, attendu %q", id, l.rev, PlacementRev)
		}
		porteur, mesure = porteur+l.porteur, mesure+l.mesure
		if l.mediane == nil {
			nonMesurees++
		}
	}
	t.Logf("%s  %d vies ecrites = %d vies en base ; %d non mesurees ; mesure %d ms, porteur %d ms ; "+
		"portee %v", id[:8], len(ecrites), vies, nonMesurees, mesure, porteur, ecrites[0].radar != nil)
}

// v2CritereDeLaPortee (lot V2b.3) : chaque ligne porte la portee de la variante du match, telle
// que la lecture independante du test la resout, et un temps hors radar des qu'elle est connue.
// Journalise la part hors radar du match (ms hors radar / ms mesurees).
func v2CritereDeLaPortee(t *testing.T, db *sql.DB, id string, ecrites []lignePlacement, portee PorteeDuRadar) {
	t.Helper()
	var variante string
	if err := db.QueryRow(`SELECT coalesce(game_variant_name, '') FROM match_registry WHERE match_id = ?`,
		id).Scan(&variante); err != nil {
		t.Fatalf("%s : variante : %v", id, err)
	}
	attendu, connue := portee(variante)
	var hors, mesure int64
	for i, l := range ecrites {
		if !connue {
			if l.radar != nil || l.horsRadar != nil {
				t.Fatalf("%s ligne %d : variante %q sans portee, ligne %+v", id, i, variante, l)
			}
			continue
		}
		if l.radar == nil || *l.radar != attendu || l.horsRadar == nil {
			t.Fatalf("%s ligne %d : variante %q, portee attendue %v m, ligne radar %s / hors radar %s",
				id, i, variante, attendu, v2Clair(l.radar), v2Clair(l.horsRadar))
		}
		hors, mesure = hors+*l.horsRadar, mesure+l.mesure
	}
	part := 0.0
	if mesure > 0 {
		part = 100 * float64(hors) / float64(mesure)
	}
	t.Logf("%s  variante %q : portee %v m (connue %v) ; hors radar %d ms sur %d ms mesurees (%.2f %%)",
		id[:8], variante, attendu, connue, hors, mesure, part)
}

// v2Clair rend la valeur pointee, ou « nil ».
func v2Clair[T any](p *T) string {
	if p == nil {
		return "nil"
	}
	return fmt.Sprint(*p)
}

// v2CalculPur : le calcul de V1.3 (`replay.PlacementDesVies`) sur la passe rejouee, nourri par des
// sources INDEPENDANTES du cablage teste : les faits de match de la CUISSON (`ReplayFactsRepo`),
// les catalogues relus, la portee du test.
func v2CalculPur(t *testing.T, ctx context.Context, col *KillSourceCollector, db *sql.DB,
	id, repoRoot string, portee PorteeDuRadar) []persist.LifePlacementInsert {
	t.Helper()
	p := v1RejouerLaPasse(t, ctx, col, db, id)
	faits, err := duckdb.NewReplayFactsRepo(db).FactsForMatch(ctx, id)
	if err != nil {
		t.Fatalf("%s : faits de match : %v", id, err)
	}
	lignes := make([]decfilm.PlayerLine, 0, len(faits.Players))
	for _, pl := range faits.Players {
		lignes = append(lignes, decfilm.PlayerLine{XUID: pl.XUID, Kills: pl.Kills, Deaths: pl.Deaths, Assists: pl.Assists})
	}
	libelles, err := replaylabels.Load(repoRoot, title.DefaultSlug)
	if err != nil {
		t.Fatalf("libelles : %v", err)
	}
	var socles []replay.FlagSpawn
	cat, err := replay.LoadMapObjectives(title.NewPathResolver(repoRoot).MapObjectivesPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("objectifs : %v", err)
	}
	if e, err := cat.Lookup(faits.MapID); err == nil {
		socles = e.SoclesDeDrapeau()
	}
	portages, bilan := replay.PortagesAuSync(ctx, replay.EntreePorteursAuSync{
		MatchID: id, Film: p.mat.film, Contexte: p.mat.contexte, Carte: p.mat.carte,
		Variante: faits.GameVariantName, ProfilDeBalayage: p.mat.profil, Identite: p.mat.identite,
		Lignes: lignes, Socles: socles, Libelles: libelles,
	})
	var radar *float64
	if m, ok := portee(faits.GameVariantName); ok {
		radar = &m
	}
	vies, _ := replay.PlacementDesVies(replay.EntreePlacement{
		Positions: p.mat.positions, Registre: p.mat.registre, Equipes: equipesNumeriques(p.ids.Equipes),
		Journal: journalDuPlacement(p.fusionne), Portages: portages, RadarM: radar,
	})
	t.Logf("%s  calcul pur : variante %q, gardes %+v, %d portages", id[:8], faits.GameVariantName,
		bilan.Gardes, bilan.Intervalles)
	rows, _ := toPlacementRows(vies)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].XUID != rows[j].XUID {
			return rows[i].XUID < rows[j].XUID
		}
		return rows[i].StartMS < rows[j].StartMS
	})
	return rows
}

// v2Comparer : ligne a ligne, champ a champ.
func v2Comparer(t *testing.T, id string, ecrites []lignePlacement, pures []persist.LifePlacementInsert) {
	t.Helper()
	if len(ecrites) != len(pures) {
		t.Fatalf("%s : %d lignes ecrites, %d calculees", id, len(ecrites), len(pures))
	}
	egal := func(a, b *float64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	for i, p := range pures {
		e := ecrites[i]
		memeHors := (e.horsRadar == nil) == (p.BeyondMS == nil) && (e.horsRadar == nil || *e.horsRadar == *p.BeyondMS)
		if e.xuid != p.XUID || e.debut != p.StartMS || e.fin != p.EndMS || e.duree != p.DurationMS ||
			e.mesure != p.MeasuredMS || e.porteur != p.CarrierMS || e.aTerre != p.TeamDownMS ||
			e.nonSitue != p.UnplacedMS || e.coequipier != p.TeammateUnplacedMS || e.frags != p.Kills ||
			!egal(e.mediane, p.MedianM) || !egal(e.radar, p.RadarM) || !memeHors {
			t.Fatalf("%s ligne %d : ecrite %+v, calculee %+v", id, i, e, p)
		}
	}
	t.Logf("%s  %d lignes ecrites = calcul pur, champ a champ", id[:8], len(pures))
}

// TestSharedRoster_LitVarianteCarteEtFeuille — les trois faits de base que la lecture des
// porteurs demande viennent de la MEME lecture que les equipes : la variante telle que la base
// la porte, le `map_id` nettoye (meme regle que `ReplayFactsRepo`), la feuille (frags, morts,
// assistances ; NULL = 0) de chaque participant.
func TestSharedRoster_LitVarianteCarteEtFeuille(t *testing.T) {
	db := baseBacklog(t)
	if _, err := db.Exec(`INSERT INTO match_registry (match_id, game_variant_name, map_id)
		VALUES ('m1', 'CTF:Arena', ' carte-1 ')`); err != nil {
		t.Fatalf("registre : %v", err)
	}
	if _, err := db.Exec(`INSERT INTO match_participants (match_id, xuid, team_id, kills, deaths, assists)
		VALUES ('m1', '111', 0, 3, 1, 2), ('m1', '222', 1, NULL, NULL, NULL)`); err != nil {
		t.Fatalf("participants : %v", err)
	}
	ids, err := NewSharedRoster(db).IdentitiesForMatch(context.Background(), "m1")
	if err != nil {
		t.Fatalf("IdentitiesForMatch : %v", err)
	}
	if ids.Variante != "CTF:Arena" || ids.CarteID != "carte-1" {
		t.Fatalf("variante %q, carte %q ; attendu CTF:Arena et carte-1", ids.Variante, ids.CarteID)
	}
	attendu := []decfilm.PlayerLine{{XUID: "111", Kills: 3, Deaths: 1, Assists: 2}, {XUID: "222"}}
	if len(ids.Feuille) != 2 || ids.Feuille[0] != attendu[0] || ids.Feuille[1] != attendu[1] {
		t.Fatalf("feuille = %+v, attendu %+v", ids.Feuille, attendu)
	}
}
