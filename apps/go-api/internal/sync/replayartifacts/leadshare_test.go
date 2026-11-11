package replayartifacts

// leadshare_test.go — la projection de la Mainmise, d'un document fabriqué à la passe à écrire :
// la mesure nominale, chaque refus nommé, le mode à peu de points.

import (
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/testutil"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

const (
	lsPas      = 100   // ms par image
	lsOrigine  = 5000  // ms : origine du film sur l'axe du match
	lsCoup     = 20000 // ms : coup d'envoi mesuré sur l'axe du match
	lsDuree    = 600   // s : durée du match au registre
	lsFilmEnd  = 9999  // dernière image du film
	lsPairSlay = "Team Slayer:Arena on Recharge"
)

func lsInt(v int) *int { return &v }

func lsInt64(v int64) *int64 { return &v }

// lsDocument : un film de 600 s dont l'équipe 0 mène dès l'image 200, l'équipe 1 passe devant à
// l'image 400, puis l'équipe 0 reprend la tête à l'image 600.
func lsDocument() *replay.ReplayDocument {
	t0, t1 := 0, 1
	return &replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: "m1",
		FrameIntervalMS: lsPas, FrameCount: lsFilmEnd + 1,
		OriginMs: lsInt64(lsOrigine), T0FilmMs: lsInt64(lsCoup),
		Roster: []replay.RosterEntry{{XUID: "a", Team: &t0}, {XUID: "b", Team: &t1}},
		Coverage: &replay.Coverage{Score: &replay.ScoreCoverage{
			TeamIdentity: "a", ModeSupported: true, Rounds: 1,
		}},
		ScoreTimeline: &replay.ScoreTimeline{Teams: []replay.TeamScore{
			{TeamID: lsInt(0), Total: []replay.ScoreTick{{T: 200, V: 1}, {T: 600, V: 3}}},
			{TeamID: lsInt(1), Total: []replay.ScoreTick{{T: 400, V: 2}, {T: 800, V: 3}}},
		}},
	}
}

func lsIdentite() IdentiteMainmise {
	d := int64(lsDuree)
	return IdentiteMainmise{PairName: lsPairSlay, DurationSeconds: &d}
}

func lsReglement(t *testing.T) *mappings.RegulationSet {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	reg, err := mappings.LoadRegulationFromFile(filepath.Join(racine, "config", "titles", "halo_infinite", "mappings", "regulation.toml"))
	if err != nil {
		t.Fatalf("regulation.toml : %v", err)
	}
	return reg
}

func TestProjeterMainmise_Nominal(t *testing.T) {
	b := ProjeterMainmise("m1", lsDocument(), lsIdentite(), lsReglement(t))
	if !b.Measured || b.Reason != "" {
		t.Fatalf("passe non mesurée : %+v", b)
	}
	// Fenêtre : début 20000-5000 = 15000 ms, fin 600000-5000 = 595000 ms.
	if b.MeasuredMS != 580000 {
		t.Errorf("durée mesurée = %d, attendu 580000", b.MeasuredMS)
	}
	if b.LeadChanges != 2 {
		t.Errorf("retournements = %d, attendu 2 (équipe 1 à l'image 400, équipe 0 à l'image 600)", b.LeadChanges)
	}
	if b.SilentCamps != 0 || len(b.Rows) != 2 {
		t.Errorf("camps muets %d, lignes %d, attendu 0 et 2", b.SilentCamps, len(b.Rows))
	}
	// Camp 0 mène [20000, 40000] et [60000, 80000] (3-2 puis 3-3 à l'image 800 = 80000 : égalité) ;
	// camp 1 mène [40000, 60000]. Avant l'image 200 (20000 ms) : égalité, dont la partie avant le coup d'envoi n'est pas comptée.
	var somme int64 = b.TieMS
	for _, r := range b.Rows {
		somme += r.LeadMS
	}
	if somme != b.MeasuredMS {
		t.Errorf("somme des temps %d != durée mesurée %d", somme, b.MeasuredMS)
	}
	if b.Rows[0].Camp != 0 || b.Rows[0].LeadMS != 40000 || b.Rows[1].LeadMS != 20000 {
		t.Errorf("temps en tête = %+v, attendu camp 0 : 40000 ms, camp 1 : 20000 ms", b.Rows)
	}
	if err := persist.ValidateLeadShareBatch(b); err != nil {
		t.Errorf("la passe projetée est refusée par le persister : %v", err)
	}
}

func TestProjeterMainmise_Refus(t *testing.T) {
	cas := []struct {
		nom    string
		mutate func(*replay.ReplayDocument, *IdentiteMainmise)
		want   leadshare.Reason
	}{
		{"mode à peu de points", func(_ *replay.ReplayDocument, i *IdentiteMainmise) { i.PairName = "CTF:Arena on Recharge" }, leadshare.ReasonModeEvents},
		{"pas de calque de score", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.ScoreTimeline = nil }, leadshare.ReasonScoreMissing},
		{"score tronqué", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.Coverage.Score.Truncated = true }, leadshare.ReasonScoreTruncated},
		{"score non répliqué", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.Coverage.Score.ModeSupported = false }, leadshare.ReasonScoreNotCovered},
		{"aucune équipe publiée", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.ScoreTimeline.Teams = nil }, leadshare.ReasonScoreNotCovered},
		{"série sans camp", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.ScoreTimeline.Teams[0].TeamID = nil }, leadshare.ReasonTeamUnresolved},
		{"origine absente", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.OriginMs = nil }, leadshare.ReasonClockOriginMissing},
		{"pas de grille absent", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.FrameIntervalMS = 0 }, leadshare.ReasonClockStepMissing},
		{"coup d'envoi non mesuré", func(d *replay.ReplayDocument, _ *IdentiteMainmise) { d.T0FilmMs = nil }, leadshare.ReasonKickoffUnmeasured},
		{"durée inconnue", func(_ *replay.ReplayDocument, i *IdentiteMainmise) { i.DurationSeconds = nil }, leadshare.ReasonDurationUnknown},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			doc, ident := lsDocument(), lsIdentite()
			c.mutate(doc, &ident)
			b := ProjeterMainmise("m1", doc, ident, lsReglement(t))
			if b.Measured || b.Reason != string(c.want) {
				t.Errorf("passe = %+v, attendu non mesurée pour %q", b, c.want)
			}
			if len(b.Rows) != 0 {
				t.Errorf("une passe refusée porte %d ligne(s) de camp", len(b.Rows))
			}
			if err := persist.ValidateLeadShareBatch(b); err != nil {
				t.Errorf("la passe refusée est refusée par le persister : %v", err)
			}
		})
	}
}

func TestProjeterMainmise_SlayerEtAssassinSontMesures(t *testing.T) {
	// Les modes `hidden` (le mode marque au frag) lisent le score du rejeu comme les autres.
	for _, pair := range []string{"Team Slayer:Arena on Recharge", "Slayer on Recharge", "Super Fiesta:Slayer on Recharge"} {
		ident := lsIdentite()
		ident.PairName = pair
		if b := ProjeterMainmise("m1", lsDocument(), ident, lsReglement(t)); !b.Measured {
			t.Errorf("%q refusé (%s)", pair, b.Reason)
		}
	}
}

func TestProjeterMainmise_UnCampMuetNeBloquePas(t *testing.T) {
	doc := lsDocument()
	doc.ScoreTimeline.Teams = doc.ScoreTimeline.Teams[:1] // le camp 1 n'a jamais marqué
	b := ProjeterMainmise("m1", doc, lsIdentite(), lsReglement(t))
	if !b.Measured || b.SilentCamps != 1 || len(b.Rows) != 1 {
		t.Errorf("passe = %+v, attendu mesurée avec un camp muet et une ligne de camp", b)
	}
}
