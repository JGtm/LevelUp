//go:build integration

package replayartifacts

// vehicletakes_integration_test.go — LA RESSOURCE VEHICULES DE L'EMPRISE, DE L'ARTEFACT A LA VUE
// `_latest`, sur une VRAIE base migree et un VRAI artefact (plan
// `.ai/V7.5/PLAN_EMPRISE_VEHICULES_2026-09-28.md`, lot L7.2).
//
// Chaque test est rouge sous une mutation nommee au journal du lot ; aucun ne se contente d'un
// grep sur le source.

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/killscope"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/replaybuild"
)

const (
	vhOrigine = 5000 // ms : OriginMs du document de test
	vhPas     = 100  // ms : pas d'image
)

// vhTags rend un tag de source de classe ENGIN et un tag de source d'une AUTRE classe, tires de
// la table embarquee (le classificateur reel) : le test ne suppose aucun identifiant en dur.
func vhTags(t *testing.T) (engin, autre uint32) {
	t.Helper()
	classes := weapons.ClassesByKey()
	for _, tag := range killicon.ResolvedTags() {
		ic, _ := killicon.Lookup(tag)
		if ic.WeaponKey == "" {
			continue
		}
		switch {
		case domain.IsEngineFragClass(classes[ic.WeaponKey]) && engin == 0:
			engin = tag
		case !domain.IsEngineFragClass(classes[ic.WeaponKey]) && classes[ic.WeaponKey] != "" && autre == 0:
			autre = tag
		}
	}
	if engin == 0 || autre == 0 {
		t.Fatalf("table killicon sans source d'engin (%d) ou sans source d'une autre classe (%d)", engin, autre)
	}
	return engin, autre
}

// vhDocument : un artefact mesure (schema courant, calque balaye), camp 0 = a1, camp 1 = b1.
// Un warthog porte a1 (images 10 a 20), un ghost porte b1 (30 a 40), et un episode sans xuid (un
// bot, D10) sur un troisieme vehicule.
func vhDocument(matchID string) replay.ReplayDocument {
	o := int64(vhOrigine)
	t0, t1 := 0, 1
	ride := func(x string, a, b int) replay.VehicleRide {
		return replay.VehicleRide{XUID: x, T0: a, T1: b, Src: "film"}
	}
	return replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: matchID,
		FrameIntervalMS: vhPas, OriginMs: &o,
		Coverage: &replay.Coverage{Vehicles: &replay.VehicleCoverage{Scanned: true}},
		Roster: []replay.RosterEntry{
			{XUID: "a1", Name: "Alpha", Team: &t0}, {XUID: "b1", Name: "Bravo", Team: &t1},
		},
		Vehicles: []replay.VehicleTrack{
			{Slot: 1, Gen: 1, Family: "warthog", Rides: []replay.VehicleRide{ride("a1", 10, 20)}},
			{Slot: 2, Gen: 1, Family: "ghost", Rides: []replay.VehicleRide{ride("b1", 30, 40)}},
			{Slot: 3, Gen: 1, Family: "mongoose", Rides: []replay.VehicleRide{ride("", 50, 60)}},
		},
	}
}

func vhPoser(t *testing.T, repoRoot, matchID string, doc replay.ReplayDocument) string {
	t.Helper()
	path := titlePkg.NewPathResolver(repoRoot).ReplayArtifactPath(titlePkg.DefaultSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

// vhMort insere une mort a SOURCE MESUREE dans match_kill_events (une passe de decodage).
func vhMort(t *testing.T, db *sql.DB, matchID string, timeMS int, killer string, tag uint32) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO match_kill_events
		(match_id, decode_pass, decoder_rev, publishable, time_ms, victim_gamertag, feed_killer_xuid,
		 feed_present, assist_known, source_tag, read_path, read_origin)
		VALUES (?, 'p1', 'r1', TRUE, ?, 'victime', ?, TRUE, FALSE, ?, ?, ?)`,
		matchID, timeMS, nullIfEmpty(killer), tag, killscope.ReadPathFilmWalk, killscope.OriginCreditOnly); err != nil {
		t.Fatalf("insert mort: %v", err)
	}
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// vhLot : base migree + racine + artefact + Deps du fil de l'eau + artefact lu.
func vhLot(t *testing.T, matchID string, doc replay.ReplayDocument) (*sql.DB, Deps, string) {
	t.Helper()
	db := baseRegistre(t)
	repoRoot := racineAvecConfigDuTitre(t, titlePkg.DefaultSlug)
	inscrireAuRegistre(t, db, matchID, temoinT0(), 0)
	chemin := vhPoser(t, repoRoot, matchID, doc)
	d := Deps{
		RepoRoot: repoRoot, TitleSlug: titlePkg.DefaultSlug, Gamertag: "testeur",
		WithRead:      func(_ context.Context, _ string, fn func(*sql.DB)) { fn(db) },
		AcquireWriter: func(context.Context) (*sql.DB, func(), error) { return db, func() {}, nil },
	}
	return db, d, chemin
}

// vhFamille : la famille telle que Deriver l'enchaine (preparation puis ecriture).
func vhFamille(t *testing.T, d Deps, b *bilanDerivations, chemin, matchID string) {
	t.Helper()
	ctx := context.Background()
	lus := lireArtefacts(ctx, d, []ArtefactRange{{MatchID: matchID, Path: chemin}})
	if len(lus) != 1 {
		t.Fatalf("%d artefact(s) lu(s), attendu 1", len(lus))
	}
	ecrirePrisesDeVehicules(ctx, d, b, preparerPrisesDeVehicules(ctx, d, b, lus))
}

// vhLigne : une ligne de la vue `_latest`.
type vhLigne struct {
	kind, xuid, family                        string
	camp, takes, episodes, prox, frags        int
	aboard                                    int64
	measured, fragsRead                       bool
	reason, fragsReason                       string
	epNoXUID, fragsTotal, fragsUnmatched, sch int
}

func vhLire(t *testing.T, db *sql.DB, matchID string) []vhLigne {
	t.Helper()
	rows, err := db.Query(`SELECT row_kind, xuid, family, camp, takes, episodes, proximity_episodes, frags,
		aboard_ms, measured, frags_read, unmeasured_reason, frags_reason, episodes_unnamed, frags_total,
		frags_unmatched, doc_schema
		FROM match_vehicle_takes_latest WHERE match_id = ? ORDER BY row_kind, camp, xuid, family`, matchID)
	if err != nil {
		t.Fatalf("lecture de la vue: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var out []vhLigne
	for rows.Next() {
		var l vhLigne
		if err := rows.Scan(&l.kind, &l.xuid, &l.family, &l.camp, &l.takes, &l.episodes, &l.prox, &l.frags,
			&l.aboard, &l.measured, &l.fragsRead, &l.reason, &l.fragsReason, &l.epNoXUID, &l.fragsTotal,
			&l.fragsUnmatched, &l.sch); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, l)
	}
	return out
}

func vhPrise(lignes []vhLigne, xuid, famille string) (vhLigne, bool) {
	for _, l := range lignes {
		if l.kind == "take" && l.xuid == xuid && l.family == famille {
			return l, true
		}
	}
	return vhLigne{}, false
}

func vhMatch(t *testing.T, lignes []vhLigne) vhLigne {
	t.Helper()
	var out []vhLigne
	for _, l := range lignes {
		if l.kind == "match" {
			out = append(out, l)
		}
	}
	if len(out) != 1 {
		t.Fatalf("%d ligne(s) match dans la passe retenue, attendu 1 : %+v", len(out), lignes)
	}
	return out[0]
}

// TestVehicules_EcritLesPrisesLeTempsLesFragsEtLaCouverture — le cas NOMINAL. D9 : un frag
// d'engin DANS un episode de son tueur est apparie, un frag d'engin HORS episode est compte non
// apparie, un frag d'une autre classe n'est pas un frag d'engin. D10 : l'episode sans xuid est
// compte en couverture et ne fait ni prise ni temps.
func TestVehicules_EcritLesPrisesLeTempsLesFragsEtLaCouverture(t *testing.T) {
	engin, autre := vhTags(t)
	db, d, chemin := vhLot(t, "veh1", vhDocument("veh1"))
	vhMort(t, db, "veh1", vhOrigine+1500, "a1", engin) // dans l'episode de a1 (images 10 a 20)
	vhMort(t, db, "veh1", vhOrigine+9000, "a1", engin) // hors de tout episode
	vhMort(t, db, "veh1", vhOrigine+1600, "a1", autre) // dans l'episode, mais pas un engin
	vhMort(t, db, "veh1", vhOrigine+3500, "b1", engin) // dans l'episode de b1 (30 a 40)
	vhMort(t, db, "veh1", vhOrigine+1500, "", engin)   // un tueur bot : ni xuid ni episode nomme
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh1")

	lignes := vhLire(t, db, "veh1")
	m := vhMatch(t, lignes)
	if !m.measured || m.reason != "" || !m.fragsRead || m.fragsReason != "" {
		t.Fatalf("couverture du match : %+v", m)
	}
	if m.fragsTotal != 3 || m.fragsUnmatched != 1 {
		t.Errorf("frags_total=%d frags_unmatched=%d, attendu 3 et 1 (le bot n'est pas compte, l'autre classe non plus)",
			m.fragsTotal, m.fragsUnmatched)
	}
	if m.epNoXUID != 1 {
		t.Errorf("episodes_unnamed = %d, attendu 1 (D10)", m.epNoXUID)
	}
	a, ok := vhPrise(lignes, "a1", "warthog")
	if !ok || a.takes != 1 || a.aboard != 1000 || a.frags != 1 || a.camp != 0 {
		t.Errorf("ligne a1/warthog : %+v (ok=%v), attendu 1 prise, 1000 ms a bord ((20-10) images x 100 ms), 1 frag, camp 0", a, ok)
	}
	b, ok := vhPrise(lignes, "b1", "ghost")
	if !ok || b.frags != 1 || b.camp != 1 {
		t.Errorf("ligne b1/ghost : %+v (ok=%v)", b, ok)
	}
	for _, l := range lignes {
		if l.kind == "take" && l.xuid == "" {
			t.Errorf("ligne de prise sans xuid : %+v (D10 : un bot ne fait ni prise ni temps)", l)
		}
	}
}

// TestVehicules_LaVueNeRendQueLaDernierePasse — re-projeter un match ecrit une passe de plus ; la
// vue retient la derniere ENTIERE, et une passe sans prise RETRACTE les prises de la precedente
// (c'est la ligne match qui l'y autorise).
func TestVehicules_LaVueNeRendQueLaDernierePasse(t *testing.T) {
	db, d, chemin := vhLot(t, "veh2", vhDocument("veh2"))
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh2")
	if _, ok := vhPrise(vhLire(t, db, "veh2"), "a1", "warthog"); !ok {
		t.Fatal("premiere passe sans la prise de a1")
	}
	// Le calque est relu sans aucun episode : la passe suivante n'a plus de prise.
	doc := vhDocument("veh2")
	doc.Vehicles = nil
	vhPoser(t, d.RepoRoot, "veh2", doc)
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh2")

	lignes := vhLire(t, db, "veh2")
	if len(lignes) != 1 || lignes[0].kind != "match" || !lignes[0].measured {
		t.Fatalf("la vue doit rendre la seule ligne match de la derniere passe (zero mesure) : %+v", lignes)
	}
	var brut int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_vehicle_takes WHERE match_id = 'veh2'`).Scan(&brut); err != nil || brut < 4 {
		t.Errorf("la table brute doit garder les deux passes (append-only) : %d ligne(s), err=%v", brut, err)
	}
}

// TestVehicules_ArtefactSansOccupationEstNonMesure — D8 : un artefact de schema < 67 n'est ni
// zero ni absent : une passe « non mesure », avec sa raison.
func TestVehicules_ArtefactSansOccupationEstNonMesure(t *testing.T) {
	doc := vhDocument("veh3")
	doc.SchemaVersion = 61
	db, d, chemin := vhLot(t, "veh3", doc)
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh3")

	lignes := vhLire(t, db, "veh3")
	m := vhMatch(t, lignes)
	if len(lignes) != 1 || m.measured || m.reason != "schema_before_67" || m.sch != 61 {
		t.Errorf("passe non mesuree attendue (raison %q, schema 61) : %+v", "schema_before_67", lignes)
	}
	if m.fragsRead || m.fragsReason != VehicleFragsNotMeasured {
		t.Errorf("frags d'une passe non mesuree : %+v", m)
	}
}

// TestVehicules_SansEvenementDeMortLesFragsSontNonLusEtLesPrisesEcrites — D9 : pas de frag lu ne
// veut pas dire zero frag.
func TestVehicules_SansEvenementDeMortLesFragsSontNonLusEtLesPrisesEcrites(t *testing.T) {
	db, d, chemin := vhLot(t, "veh4", vhDocument("veh4"))
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh4")

	lignes := vhLire(t, db, "veh4")
	m := vhMatch(t, lignes)
	if m.fragsRead || m.fragsReason != "no_kill_source" || m.fragsTotal != 0 {
		t.Errorf("frags non lus attendus (raison %q) : %+v", "no_kill_source", m)
	}
	if a, ok := vhPrise(lignes, "a1", "warthog"); !ok || a.takes != 1 || a.frags != 0 {
		t.Errorf("les prises s'ecrivent quand meme : %+v (ok=%v)", a, ok)
	}
}

// TestVehicules_FragsIllisiblesNEcriventPasLeMatchMesureEtNeLeMarquentPas — une lecture de base
// en echec est un DEFAUT : rien n'est ecrit pour le match mesure, et le bilan l'inscrit (sa
// marque ne se poserait pas, le rattrapage le reprendra).
func TestVehicules_FragsIllisiblesNEcriventPasLeMatchMesureEtNeLeMarquentPas(t *testing.T) {
	db, d, chemin := vhLot(t, "veh5", vhDocument("veh5"))
	d.WithRead = nil
	b := &bilanDerivations{}
	vhFamille(t, d, b, chemin, "veh5")

	if n := len(vhLire(t, db, "veh5")); n != 0 {
		t.Errorf("%d ligne(s) ecrite(s) sans lecture des frags", n)
	}
	if !b.aEchoue("veh5") {
		t.Error("frags illisibles sans echec au bilan : la marque se poserait sur un match non ecrit")
	}
}

// TestVehicules_NEcritRienSansLaCapabilite — la porte gouverne, et c'est la SEULE chose qui
// change entre ce test et le nominal.
func TestVehicules_NEcritRienSansLaCapabilite(t *testing.T) {
	db, d, chemin := vhLot(t, "veh6", vhDocument("veh6"))
	p := filepath.Join(d.RepoRoot, "config", "titles", titlePkg.DefaultSlug, "mappings", "capabilities.toml")
	blob, err := os.ReadFile(p) //nolint:gosec // racine de test
	if err != nil {
		t.Fatalf("lecture capabilities: %v", err)
	}
	var out []string
	for _, l := range strings.Split(string(blob), "\n") {
		if !strings.Contains(l, string(games.CapFilmVehicleUsage)) {
			out = append(out, l)
		}
	}
	if err := os.WriteFile(p, []byte(strings.Join(out, "\n")), 0o600); err != nil {
		t.Fatalf("ecriture capabilities: %v", err)
	}
	vhFamille(t, d, &bilanDerivations{}, chemin, "veh6")

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM match_vehicle_takes`).Scan(&n); err != nil || n != 0 {
		t.Errorf("%d ligne(s) ecrite(s) sur un titre qui ne declare pas %s (err=%v)", n, games.CapFilmVehicleUsage, err)
	}
}

// TestDeriver_VehiculesAuFilDeLEauEtMarque — le point d'entree REEL : la famille ecrit dans le
// segment unique de la passe, et le match est marque derive (aucune famille n'a echoue).
func TestDeriver_VehiculesAuFilDeLEauEtMarque(t *testing.T) {
	engin, _ := vhTags(t)
	db, d, chemin := vhLot(t, "veh7", vhDocument("veh7"))
	vhMort(t, db, "veh7", vhOrigine+1500, "a1", engin)

	Deriver(context.Background(), DerivationsDeps{
		RepoRoot: d.RepoRoot, TitleSlug: d.TitleSlug, Gamertag: d.Gamertag,
		WithRead: d.WithRead, AcquireWriter: d.AcquireWriter,
	}, []ArtefactRange{{MatchID: "veh7", Path: chemin}})

	lignes := vhLire(t, db, "veh7")
	if a, ok := vhPrise(lignes, "a1", "warthog"); !ok || a.frags != 1 {
		t.Errorf("Deriver n'a pas ecrit la famille vehicules : %+v (ok=%v)", lignes, ok)
	}
	if !replaybuild.DerivationsUpToDate(chemin) {
		t.Error("match non marque alors que toutes les familles ont pu ecrire")
	}
}
