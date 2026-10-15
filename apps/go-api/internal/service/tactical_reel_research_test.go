//go:build research

package service

// tactical_reel_research_test.go — LA PREUVE SUR DONNEES REELLES de la lecture des coequipiers a la
// mort : le service tactique, branche sur de VRAIES bases (des COPIES, jamais `data/`) et sur les
// vrais artefacts de rejeu (lecture seule), appele comme le handler, sur la carte Streets du joueur.
//
//	REEL_PARC=<dossier des copies : warehouse/shared_matches_v2.duckdb, warehouse/metadata.duckdb,
//	           players/<gamertag>/stats.duckdb>
//	REEL_RUNTIME_ROOT=<racine contenant data/cache/replays (lecture seule)>
//	REEL_OUT=<dossier de sortie des reel-*.json>
//	REEL_GAMERTAG=JGtm REEL_MAP=Streets (defauts)
//	  go test -tags research -run TestReelCoequipiers -v -count=1 ./internal/service/
//
// Il ecrit les reponses JSON (raster, detail de cellule, sequence) que le script de capture sert, et
// CONTROLE leur coherence : un ecart de DONNEES est consigne avec l'identifiant du match.

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/platform/duckdb"
)

type reelEnv struct {
	parc, runtimeRoot, out, gamertag, carte string
}

func reelEnvironnement(t *testing.T) reelEnv {
	t.Helper()
	e := reelEnv{parc: os.Getenv("REEL_PARC"), runtimeRoot: os.Getenv("REEL_RUNTIME_ROOT"), out: os.Getenv("REEL_OUT"),
		gamertag: os.Getenv("REEL_GAMERTAG"), carte: os.Getenv("REEL_MAP")}
	if e.parc == "" || e.runtimeRoot == "" || e.out == "" {
		t.Skip("REEL_PARC / REEL_RUNTIME_ROOT / REEL_OUT non poses : preuve sur donnees reelles (copies)")
	}
	if e.gamertag == "" {
		e.gamertag = "JGtm"
	}
	if e.carte == "" {
		e.carte = "Streets"
	}
	return e
}

func reelEcrire(t *testing.T, e reelEnv, nom string, v any) {
	t.Helper()
	octets, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatalf("json %s: %v", nom, err)
	}
	if err := os.WriteFile(filepath.Join(e.out, nom), octets, 0o644); err != nil {
		t.Fatalf("ecriture %s: %v", nom, err)
	}
	t.Logf("%s : %d octets", nom, len(octets))
}

// reelBases ouvre les trois copies et rend le PlayerDB, le xuid, la carte et les matchs du joueur.
func reelBases(t *testing.T, e reelEnv) (*duckdb.PlayerDB, string, string, []string) {
	t.Helper()
	shared, err := duckdb.OpenReadOnly(filepath.Join(e.parc, "warehouse", "shared_matches_v2.duckdb"))
	if err != nil {
		t.Fatalf("shared: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	meta, err := duckdb.OpenReadWrite(filepath.Join(e.parc, "warehouse", "metadata.duckdb")) // COPIE : le wal se rejoue
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	player, err := duckdb.OpenReadWrite(filepath.Join(e.parc, "players", e.gamertag, "stats.duckdb"))
	if err != nil {
		t.Fatalf("player: %v", err)
	}
	t.Cleanup(func() { _ = player.Close() })

	db := shared.SQLDb()
	var xuid string
	if err := db.QueryRow(`SELECT xuid FROM match_participants WHERE gamertag = ? LIMIT 1`, e.gamertag).Scan(&xuid); err != nil {
		t.Fatalf("xuid de %s: %v", e.gamertag, err)
	}
	var carte string
	if err := db.QueryRow(`SELECT map_id FROM match_registry WHERE map_name = ? LIMIT 1`, e.carte).Scan(&carte); err != nil {
		t.Fatalf("carte %s: %v", e.carte, err)
	}
	rows, err := db.Query(`SELECT mr.match_id FROM match_registry mr JOIN match_participants mp ON mp.match_id = mr.match_id
		WHERE mp.xuid = ? AND mr.map_id = ? ORDER BY mr.match_id`, xuid, carte)
	if err != nil {
		t.Fatalf("matchs: %v", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	pdb := &duckdb.PlayerDB{Player: player, Shared: shared, SharedReader: duckdb.LegacySharedReader(shared),
		Metadata: meta, XUID: xuid, Gamertag: e.gamertag, TitleSlug: "halo_infinite"}
	return pdb, xuid, carte, ids
}

func reelService(t *testing.T, e reelEnv, pdb *duckdb.PlayerDB) *TacticalService {
	t.Helper()
	_, ici, _, _ := runtime.Caller(0)
	depot := filepath.Join(filepath.Dir(ici), "..", "..", "..", "..")
	reg, err := mappings.LoadRegulationForTitle(depot, "halo_infinite")
	if err != nil {
		t.Fatalf("regulation: %v", err)
	}
	caps := games.CapabilityMap{games.CapFilmKillPositions: games.CapSupported, games.CapFilmKillSource: games.CapSupported,
		games.CapFilmReplayArtifact: games.CapSupported}
	return NewTacticalService(duckdb.NewTacticalRepo(pdb), caps, pdb.XUID).
		WithRadarRange(reg.RadarRangeMap()).
		WithPlayerMatches(duckdb.NewCachedPlayerMatchesRepo(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithReplay(NewReplayService(pdb.TitleSlug, e.runtimeRoot, nil)).
		WithSequenceStore(NewTacticalSequenceStore(e.runtimeRoot, pdb.TitleSlug))
}

// reelCellulesParDensite rend les cellules du raster de la plus dense a la moins dense (la premiere est
// celle que le web preselectionne).
func reelCellulesParDensite(r domain.TacticalRaster) []domain.CelluleTactique {
	cells := append([]domain.CelluleTactique(nil), r.Cellules...)
	sort.SliceStable(cells, func(i, j int) bool {
		if cells[i].Valeur != cells[j].Valeur {
			return cells[i].Valeur > cells[j].Valeur
		}
		return cells[i].Matchs > cells[j].Matchs
	})
	return cells
}

// reelCellulesLues : combien de cellules (les plus denses) on lit pour chercher une sequence.
const reelCellulesLues = 40

func TestReelCoequipiers(t *testing.T) {
	e := reelEnvironnement(t)
	pdb, xuid, carte, ids := reelBases(t, e)
	svc := reelService(t, e, pdb)
	ctx := context.Background()
	scope := domain.TacticalScope{MatchIDs: ids}
	t.Logf("joueur %s (%s), carte %s (%s), %d matchs", e.gamertag, xuid, e.carte, carte, len(ids))

	page, err := svc.MapsPlayed(ctx, scope)
	if err != nil {
		t.Fatalf("MapsPlayed: %v", err)
	}
	reelEcrire(t, e, "reel-maps.json", page)
	reelEcrire(t, e, "reel-matchs.json", ids)

	var incoherences []string
	meta := map[string]any{"gamertag": e.gamertag, "xuid": xuid, "map_id": carte, "matchs": len(ids)}
	for _, question := range []string{domain.TacticalQuestionMorts, domain.TacticalQuestionKills} {
		raster, err := svc.Raster(ctx, domain.TacticalRasterRequest{MapID: carte, Question: question, Qui: domain.TacticalQuiMoi, Scope: scope})
		if err != nil {
			t.Fatalf("Raster %s: %v", question, err)
		}
		reelEcrire(t, e, "reel-raster-"+question+".json", raster)
		cells := reelCellulesParDensite(raster)
		if len(cells) == 0 {
			t.Fatalf("raster %s sans cellule", question)
		}
		lues := 1
		if question == domain.TacticalQuestionMorts {
			lues = min(reelCellulesLues, len(cells))
		}
		var sequenceTrouvee bool
		for i, cell := range cells[:lues] {
			detail, err := svc.Cellule(ctx, domain.TacticalCelluleRequest{MapID: carte, Question: question, Qui: domain.TacticalQuiMoi,
				Scope: scope, Col: cell.Col, Lig: cell.Lig, PasM: raster.PasM})
			if err != nil {
				t.Fatalf("Cellule %s (%d, %d): %v", question, cell.Col, cell.Lig, err)
			}
			reelEcrire(t, e, fmt.Sprintf("reel-cellule-%s-c%d-l%d.json", question, cell.Col, cell.Lig), detail)
			incoherences = append(incoherences, reelControles(question, detail)...)
			if i == 0 {
				t.Logf("%s : zone la plus dense col=%d lig=%d valeur=%.3f matchs=%d (pas %v m), %d contributions",
					question, cell.Col, cell.Lig, cell.Valeur, cell.Matchs, raster.PasM, len(detail.Contributions))
				meta[question+"_dense"] = map[string]any{"col": cell.Col, "lig": cell.Lig, "pas_m": raster.PasM}
			}
			if question != domain.TacticalQuestionMorts || sequenceTrouvee {
				continue
			}
			if req := reelSequenceCandidate(ctx, t, svc, carte, detail); req != nil {
				seq, err := svc.Sequence(ctx, *req)
				if err != nil {
					t.Fatalf("Sequence %+v: %v", *req, err)
				}
				reelEcrire(t, e, "reel-sequence.json", seq)
				reelEcrire(t, e, "reel-sequence-demande.json", req)
				incoherences = append(incoherences, reelControlesSequence(req.MatchID, seq)...)
				meta["sequence"] = map[string]any{"col": cell.Col, "lig": cell.Lig, "rang_de_densite": i + 1, "match_id": req.MatchID, "time_ms": req.TimeMs}
				sequenceTrouvee = true
			}
		}
		if question == domain.TacticalQuestionMorts && !sequenceTrouvee {
			t.Logf("AUCUNE mort des %d zones les plus denses dont le match a un artefact exploitable", lues)
		}
	}
	reelEcrire(t, e, "reel-meta.json", meta)
	sort.Strings(incoherences)
	reelEcrire(t, e, "reel-incoherences.json", incoherences)
	for _, s := range incoherences {
		t.Errorf("INCOHERENCE DE DONNEES : %s", s)
	}
}

// reelSequenceCandidate rend la premiere mort de la zone dont la sequence se sert.
func reelSequenceCandidate(ctx context.Context, t *testing.T, svc *TacticalService, carte string,
	detail domain.TacticalCelluleReponse) *domain.TacticalSequenceRequest {
	t.Helper()
	for _, c := range detail.Contributions {
		if c.Face != domain.TacticalFaceMort || !c.ReplayAvailable || c.Equipe == nil {
			continue
		}
		req := domain.TacticalSequenceRequest{MapID: carte, MatchID: c.MatchID, VictimXUID: c.XUID, TimeMs: c.InstantMs}
		if _, err := svc.Sequence(ctx, req); err == nil {
			t.Logf("sequence retenue : match %s, victime %s, instant %d ms", req.MatchID, req.VictimXUID, req.TimeMs)
			return &req
		} else {
			t.Logf("sequence indisponible pour %s (%d ms) : %v", c.MatchID, c.InstantMs, err)
		}
	}
	return nil
}

func reelControles(question string, d domain.TacticalCelluleReponse) []string {
	var out []string
	add := func(format string, a ...any) {
		out = append(out, fmt.Sprintf("[%s] ", question)+fmt.Sprintf(format, a...))
	}
	for _, c := range d.Contributions {
		if c.Equipe != nil {
			for _, m := range c.Equipe.Coequipiers {
				if m.Statut == domain.TacticalMateVisible && m.Point != nil && c.Point != nil && m.DistanceMortM != nil {
					if ecart := math.Abs(math.Hypot(m.Point.X-c.Point.X, m.Point.Y-c.Point.Y) - *m.DistanceMortM); ecart > 0.5 {
						add("match %s mort %d ms : distance du coequipier %s = %.2f m, mais ses points en donnent %.2f m",
							c.MatchID, c.InstantMs, m.XUID, *m.DistanceMortM, *m.DistanceMortM+ecart)
					}
				}
				if m.Statut != domain.TacticalMateVisible && (m.Point != nil || m.DistanceMortM != nil) {
					add("match %s mort %d ms : coequipier %s %s avec une position", c.MatchID, c.InstantMs, m.XUID, m.Statut)
				}
				if m.DansLeRayon && c.Equipe.RayonM != nil && m.DistanceMortM != nil && *m.DistanceMortM > *c.Equipe.RayonM {
					add("match %s mort %d ms : coequipier %s dit dans le rayon a %.2f m (rayon %.0f m)", c.MatchID, c.InstantMs, m.XUID, *m.DistanceMortM, *c.Equipe.RayonM)
				}
			}
		}
		if c.Face == domain.TacticalFaceMort && c.Point == nil {
			add("match %s mort %d ms sans position", c.MatchID, c.InstantMs)
		}
	}
	if b := d.Bilan; b != nil && b.Morts != nil {
		m := b.Morts
		if m.Accompagnees+m.VisiblesHorsRayon+m.AucunVisible+m.EquipeATerre != m.Examinees {
			add("bilan : somme des statuts %d != examinees %d", m.Accompagnees+m.VisiblesHorsRayon+m.AucunVisible+m.EquipeATerre, m.Examinees)
		}
		rose := 0
		for _, n := range m.Rose {
			rose += n
		}
		if rose > m.Total {
			add("bilan : rose %d > total %d", rose, m.Total)
		}
	}
	return out
}

func reelControlesSequence(matchID string, s domain.TacticalSequence) []string {
	var out []string
	add := func(format string, a ...any) {
		out = append(out, fmt.Sprintf("[sequence %s] ", matchID)+fmt.Sprintf(format, a...))
	}
	if len(s.Etapes) != 8 {
		add("%d etapes, attendu 8", len(s.Etapes))
	}
	for _, e := range s.Etapes {
		if e.DansLeRayon != len(e.DansLeRayonXUIDs) || e.DansLeRayon > e.Positionnes {
			add("etape %d ms : %d dans le rayon, %d xuids, %d positionnes", e.OffsetMs, e.DansLeRayon, len(e.DansLeRayonXUIDs), e.Positionnes)
		}
	}
	return out
}

// TestReelNomsDesCoequipiers : d'ou vient un nom de coequipier absent ? (sonde de la preuve reelle)
func TestReelNomsDesCoequipiers(t *testing.T) {
	e := reelEnvironnement(t)
	pdb, _, _, _ := reelBases(t, e)
	db := pdb.Shared.SQLDb()
	var total, sansNom int
	if err := db.QueryRow(`SELECT count(*), count(*) FILTER (WHERE gamertag IS NULL OR gamertag = '') FROM match_participants`).Scan(&total, &sansNom); err != nil {
		t.Fatal(err)
	}
	t.Logf("match_participants : %d lignes, %d sans gamertag", total, sansNom)
	rows, err := db.Query(`SELECT m.match_id, m.mate_xuid, COALESCE(p.gamertag, '<pas de ligne>'), COALESCE(a.gamertag, '<pas d alias>')
		FROM (SELECT DISTINCT match_id, mate_xuid FROM match_death_mates_latest LIMIT 400) m
		LEFT JOIN match_participants p ON p.match_id = m.match_id AND p.xuid = m.mate_xuid
		LEFT JOIN xuid_aliases a ON a.xuid = m.mate_xuid ORDER BY 1, 2 LIMIT 12`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a, b, c, d string
		if err := rows.Scan(&a, &b, &c, &d); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s %s participants=%q alias=%q", a[:8], b, c, d)
	}
}
