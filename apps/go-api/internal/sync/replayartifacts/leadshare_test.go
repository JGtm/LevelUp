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

	"github.com/JGtm/halo-infinite-film-decoder/replay"
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
	if !b.Measured || b.SilentCamps != 1 {
		t.Errorf("passe = %+v, attendu mesurée avec un camp muet", b)
	}
	// Le roster désigne le camp 1 : la passe le porte comme une ligne de camp à zéro MESURÉ, c'est ce
	// qui le distingue d'un camp absent du match.
	if len(b.Rows) != 2 || b.Rows[1].Camp != 1 || b.Rows[1].LeadMS != 0 {
		t.Errorf("lignes = %+v, attendu le camp 0 puis le camp muet 1 à zéro", b.Rows)
	}
	if err := persist.ValidateLeadShareBatch(b); err != nil {
		t.Errorf("la passe est refusée par le persister : %v", err)
	}
}

func TestProjeterMainmise_CampMuetNonDesigneParLeRoster(t *testing.T) {
	// Le roster ne désigne qu'un camp qui a une série et un camp muet est compté par le roster sans
	// qu'on puisse le nommer (désignateurs incohérents avec les séries) : la mesure se fait, le camp
	// muet n'a pas de ligne, donc sa part ne se lira pas.
	doc := lsDocument()
	doc.ScoreTimeline.Teams = doc.ScoreTimeline.Teams[:1]
	t1, t5 := 1, 5
	doc.Roster = []replay.RosterEntry{{XUID: "a", Team: &t1}, {XUID: "b", Team: &t5}}
	b := ProjeterMainmise("m1", doc, lsIdentite(), lsReglement(t))
	if !b.Measured {
		t.Fatalf("passe = %+v, attendu mesurée", b)
	}
	// Deux désignateurs (1 et 5), une série du camp 0 : le camp 0 n'est pas au roster, deux camps muets
	// pour un seul attendu (2 - 1) : aucun n'est nommé.
	if len(b.Rows) != 1 {
		t.Errorf("lignes = %+v, attendu la seule série publiée", b.Rows)
	}
}

// Au pas de 33 ms (30 images/s), la fenêtre jouée finit à la dernière image du film (9999 x 33 =
// 329 967 ms), pas à la durée du match : ignorer le pas donnerait 595 000 ms.
func TestProjeterMainmise_AuPasDe33ms(t *testing.T) {
	doc := lsDocument()
	doc.FrameIntervalMS = 33
	doc.FrameCount = lsFilmEnd + 1
	// Paliers (images) : camp 0 à 500 puis 1000, camp 1 à 800 puis 1200 ; soit, à 33 ms, 16 500,
	// 33 000, 26 400 et 39 600 ms.
	doc.ScoreTimeline = &replay.ScoreTimeline{Teams: []replay.TeamScore{
		{TeamID: lsInt(0), Total: []replay.ScoreTick{{T: 500, V: 1}, {T: 1000, V: 3}}},
		{TeamID: lsInt(1), Total: []replay.ScoreTick{{T: 800, V: 2}, {T: 1200, V: 3}}},
	}}
	b := ProjeterMainmise("m1", doc, lsIdentite(), lsReglement(t))
	if !b.Measured {
		t.Fatalf("passe = %+v", b)
	}
	// Fenêtre [15 000, 329 967] : égalité [15000,16500], camp 0 [16500,26400], camp 1 [26400,33000],
	// camp 0 [33000,39600], égalité jusqu'à 329 967.
	if b.MeasuredMS != 314967 || b.Rows[0].LeadMS != 16500 || b.Rows[1].LeadMS != 6600 || b.TieMS != 291867 {
		t.Errorf("mesure = %+v, attendu durée 314967, camp 0 : 16500, camp 1 : 6600, égalité 291867", b)
	}
	if b.LeadChanges != 2 {
		t.Errorf("retournements = %d, attendu 2 (camp 1 à 26 400 ms, camp 0 à 33 000 ms)", b.LeadChanges)
	}
	if err := persist.ValidateLeadShareBatch(b); err != nil {
		t.Errorf("la passe est refusée par le persister : %v", err)
	}
}

// Les retournements sont ceux de la fenêtre jouée : un retournement avant le coup d'envoi mesuré
// (le film déborde le match de son countdown) n'est pas joué. Fenêtre [15 000, 595 000].
func TestProjeterMainmise_LesRetournementsSontCeuxDeLaFenetre(t *testing.T) {
	doc := lsDocument()
	// Camp 0 : 1 à 1 000 ms, 3 à 30 000 ms. Camp 1 : 2 à 5 000 ms, 4 à 40 000 ms, 5 à 90 000 ms.
	// Le camp 0 prend la tête (1 000 ms), le camp 1 la lui prend à 5 000 ms (AVANT la fenêtre), puis
	// 30 000 ms (camp 0) et 40 000 ms (camp 1) : trois retournements au film, deux joués.
	doc.ScoreTimeline.Teams[0].Total = []replay.ScoreTick{{T: 10, V: 1}, {T: 300, V: 3}}
	doc.ScoreTimeline.Teams[1].Total = []replay.ScoreTick{{T: 50, V: 2}, {T: 400, V: 4}, {T: 900, V: 5}}
	b := ProjeterMainmise("m1", doc, lsIdentite(), lsReglement(t))
	if !b.Measured {
		t.Fatalf("passe = %+v", b)
	}
	if b.LeadChanges != 2 {
		t.Errorf("retournements = %d, attendu 2 (celui de 5 000 ms précède la fenêtre)", b.LeadChanges)
	}
	// Le meneur d'avant la fenêtre (camp 1) la tient jusqu'à 30 000 ms.
	if b.Rows[0].LeadMS != 10000 || b.Rows[1].LeadMS != 570000 || b.TieMS != 0 {
		t.Errorf("mesure = %+v, attendu camp 0 : 10000, camp 1 : 570000, aucune égalité", b)
	}
}
