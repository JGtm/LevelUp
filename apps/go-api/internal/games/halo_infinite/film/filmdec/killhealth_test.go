package filmdec

import (
	"strings"
	"testing"
)

// LES CINQ FILMS MESURES LE 2026-07-27 (mode `hyhealth`). Ce ne sont pas des cas fabriques : ce
// sont les chiffres du journal, figes ici pour qu'un ajustement de seuil se voie en test au lieu
// de se decouvrir trois sections plus tard.
var killHealthSeries = []struct {
	h            KillSourceHealth
	wantVerdict  string
	wantAlertsGE int
}{
	{KillSourceHealth{Film: "000d5950", Candidates: 100, Published: 93,
		UnexplainedPair: 4, UnexplainedSelf: 3, DeathsReal: 93, DeathsCovered: 93}, VerdictNominal, 0},
	{KillSourceHealth{Film: "9b191a7f", Candidates: 96, Published: 84,
		UnexplainedPair: 4, UnexplainedSelf: 2, UnexplainedBotIdx: 3, DeathsReal: 84, DeathsCovered: 84}, VerdictNominal, 0},
	{KillSourceHealth{Film: "78919882", Candidates: 112, Published: 99,
		UnexplainedPair: 4, UnexplainedSelf: 9, DeathsReal: 99, DeathsCovered: 99}, VerdictNominal, 0},
	{KillSourceHealth{Film: "fccc61cd", Candidates: 118, Published: 95,
		UnexplainedPair: 2, UnexplainedSelf: 17, UnexplainedBotIdx: 2, DeathsReal: 95, DeathsCovered: 95}, VerdictNominal, 0},
	// BTB : CONTROLE POSITIF DE DOMAINE. Il doit sortir, et par plusieurs criteres.
	// Son taux d'inexpliques vaut 79/304 = 26.0 % depuis le correctif de denominateur du
	// 2026-07-27 (le 27.0 % du journal comptait `OutOfRoster` au numerateur sans l'avoir au
	// denominateur — RE_LOG 7ter.73 (4e)).
	{KillSourceHealth{Film: "4f77afc1", Candidates: 304, Published: 224,
		UnexplainedPair: 26, UnexplainedSelf: 15, UnexplainedBotIdx: 38, OutOfRoster: 3,
		DeathsReal: 293, DeathsCovered: 224}, VerdictAlerte, 1},
}

func TestKillSourceHealthSerieDeReference(t *testing.T) {
	for _, c := range killHealthSeries {
		if got := c.h.Verdict(); got != c.wantVerdict {
			t.Errorf("%s : verdict = %q, attendu %q (inexpliques %.1f%%, couverture %.1f%%)",
				c.h.Film, got, c.wantVerdict, 100*c.h.UnexplainedRatio(), 100*c.h.CoverageRatio())
		}
		if n := len(c.h.Alerts()); n < c.wantAlertsGE {
			t.Errorf("%s : %d alerte(s), au moins %d attendue(s)", c.h.Film, n, c.wantAlertsGE)
		}
	}
}

// LES QUATRE FILMS A 8 JOUEURS DOIVENT RESTER SOUS LE SEUIL. Si un ajustement de seuil les fait
// sortir, la serie de reference ne serait plus la reference — et le seuil ne sortirait plus de la
// distribution observee, ce qui est precisement la derive que ce test interdit.
func TestKillSourceHealthSeuilsCouvrentLaSerieDeReference(t *testing.T) {
	for _, c := range killHealthSeries[:4] {
		if c.h.UnexplainedRatio() > UnexplainedWarnRatio {
			t.Errorf("%s : %.3f depasse UnexplainedWarnRatio %.3f — le seuil ne couvre plus sa propre serie",
				c.h.Film, c.h.UnexplainedRatio(), UnexplainedWarnRatio)
		}
		if c.h.CoverageRatio() < CoverageWarnRatio {
			t.Errorf("%s : couverture %.3f sous CoverageWarnRatio %.3f", c.h.Film, c.h.CoverageRatio(), CoverageWarnRatio)
		}
	}
	if UnexplainedAlertRatio <= UnexplainedWarnRatio {
		t.Fatal("le seuil d'ALERTE doit etre strictement au-dessus du seuil de sortie de domaine")
	}
}

// LE CONTROLE POSITIF DE L'ABLATION, EN TEST. Un tag hors catalogue vu par la MARCHE doit alerter
// MEME quand la couverture est intacte : c'est tout l'interet du compteur (il previent AVANT la
// perte). Mesure d'origine : 20 ablations sur 20 declenchent (section 7ter.72).
func TestKillSourceHealthTagHorsCatalogueAlerteSansPerteDeCouverture(t *testing.T) {
	h := KillSourceHealth{Film: "000d5950", Candidates: 100, Published: 93,
		TagOutOfCatalogueWalk: 6, DeathsReal: 93, DeathsCovered: 93}
	if h.Verdict() != VerdictAlerte {
		t.Fatalf("verdict = %q, attendu ALERTE alors que la couverture est intacte", h.Verdict())
	}
	if a := h.Alerts(); len(a) != 1 || !strings.Contains(a[0], "PERIMEE") {
		t.Fatalf("alertes = %v, attendu une alerte de table perimee", a)
	}
}

// LE POINT AVEUGLE DU COMPTEUR PRINCIPAL, EN TEST — c'est lui qui justifie `CoverageWarnRatio`.
//
// Un tag perime dont la seule occurrence publiee etait servie par le SCAN ne fait PAS sonner
// `TagOutOfCatalogueWalk` : le mode existe, il est mesure (21e ablation, `0c64b43f` sur
// `78919882`, RE_LOG 7ter.73 (4a)). Ce test fige les deux moities du constat : aucune alerte, ET
// une sortie de domaine par la couverture. Si quelqu'un assouplit `CoverageWarnRatio`, ce test
// tombe — et c'est exactement ce qu'on veut, parce que ce seuil est le SEUL filet de ce mode.
func TestKillSourceHealthPointAveugleTagServiParLeScanSeul(t *testing.T) {
	h := KillSourceHealth{Film: "78919882", Candidates: 112, Published: 98,
		UnexplainedPair: 4, UnexplainedSelf: 9,
		TagOutOfCatalogueWalk: 0, TagOutOfCatalogueScan: 3,
		DeathsReal: 99, DeathsCovered: 98}
	if len(h.Alerts()) != 0 {
		t.Fatalf("alertes = %v : le point aveugle veut dire AUCUNE alerte — s'il en sort une, "+
			"le compteur principal a change de portee et la doc doit suivre", h.Alerts())
	}
	if h.Verdict() != VerdictHorsDomaine {
		t.Fatalf("verdict = %q, attendu HORS DOMAINE MESURE : sans le plancher de couverture a "+
			"1.00, une table perimee passerait entierement inapercue dans ce mode", h.Verdict())
	}
}

// UN SEUL DENOMINATEUR — CORRECTIF 7ter.73 (4e).
//
// `OutOfRoster` se compte sur les dead-states de la marche, pas sur `Candidates` (le filtre de
// credibilite exclut deja tout indice hors roster). L'avoir au numerateur d'un ratio dont le
// denominateur ne le contient pas est le defaut de denominateur que 7ter.66 interdit. Ce test le
// verrouille par la mesure du BTB : 82/304 = 27.0 % (faux) contre 79/304 = 26.0 % (juste).
func TestKillSourceHealthRatioNInclutPasLeHorsRoster(t *testing.T) {
	btb := killHealthSeries[4].h
	if btb.OutOfRoster == 0 {
		t.Fatal("le cas de reference doit porter un hors-roster non nul, sinon il ne teste rien")
	}
	if got := btb.UnexplainedTotal(); got != btb.UnexplainedPair+btb.UnexplainedSelf+btb.UnexplainedBotIdx {
		t.Fatalf("UnexplainedTotal = %d : le hors-roster est revenu dans le numerateur", got)
	}
	sansRoster := btb.UnexplainedRatio()
	avecRoster := btb
	avecRoster.UnexplainedPair += avecRoster.OutOfRoster
	if avecRoster.UnexplainedRatio() <= sansRoster {
		t.Fatal("le cas de reference ne distingue pas les deux formules : test sans valeur")
	}
	if d := 100*sansRoster - 26.0; d > 0.1 || d < -0.1 {
		t.Errorf("taux corrige = %.1f %%, attendu 26.0 %% (79/304) — la mesure du journal a bouge",
			100*sansRoster)
	}
	// Le BTB reste le CONTROLE POSITIF DE DOMAINE : la correction ne doit pas le faire rentrer.
	if btb.Verdict() != VerdictAlerte {
		t.Errorf("verdict du BTB = %q apres correction, attendu ALERTE (hors roster = %d)",
			btb.Verdict(), btb.OutOfRoster)
	}
}

// LA SONDE A T4 RELACHE N'ALERTE PAS. Son rapport signal/hasard mesure est de 1.10 a 1.72 : elle
// informe. Si un jour elle se met a alerter, ce test le signale.
func TestKillSourceHealthSondeRelacheeNAlertePas(t *testing.T) {
	h := KillSourceHealth{Film: "x", Candidates: 100, Published: 100,
		TagOutOfCatalogueScan: 1465, DeathsReal: 100, DeathsCovered: 100}
	if len(h.Alerts()) != 0 {
		t.Fatalf("alertes = %v, la sonde relachee ne doit pas alerter seule", h.Alerts())
	}
}

// LES DENOMINATEURS NULS NE DOIVENT PAS PANIQUER NI RENDRE NaN (un film illisible passe ici).
func TestKillSourceHealthVide(t *testing.T) {
	var h KillSourceHealth
	if h.UnexplainedRatio() != 0 || h.CoverageRatio() != 0 {
		t.Fatalf("ratios sur structure vide = %v / %v, attendu 0 / 0", h.UnexplainedRatio(), h.CoverageRatio())
	}
	if h.Verdict() != VerdictHorsDomaine {
		t.Fatalf("verdict sur structure vide = %q (couverture 0 < plancher)", h.Verdict())
	}
}

// EXPVAR : compteurs ENTIERS, snake_case, prefixe de categorie (ADR 0009). Aucun ratio publie —
// une agregation multi-films doit sommer des numerateurs, jamais moyenner des pourcentages.
func TestKillSourceHealthExpvarPairs(t *testing.T) {
	h := killHealthSeries[0].h
	ps := h.ExpvarPairs()
	if len(ps) != 10 {
		t.Fatalf("%d compteurs publies, attendu 10", len(ps))
	}
	seen := map[string]bool{}
	for _, p := range ps {
		if !strings.HasPrefix(p.Name, "killsource_") {
			t.Errorf("compteur %q sans prefixe de categorie", p.Name)
		}
		if strings.ToLower(p.Name) != p.Name || strings.Contains(p.Name, " ") {
			t.Errorf("compteur %q n'est pas en snake_case", p.Name)
		}
		if seen[p.Name] {
			t.Errorf("compteur %q publie deux fois", p.Name)
		}
		seen[p.Name] = true
	}
	if !seen["killsource_candidates_total"] || !seen["killsource_tag_out_of_catalogue_walk"] {
		t.Error("compteurs porteurs absents de la publication")
	}
}
