package main

// verdict_metriques_test.go — LE GATE NE CONFOND PLUS LA TELEMETRIE, LE REJET ET LA PERTE.
//
// Les quatre cas ci-dessous sont ceux que le pilote a nommes en tranchant la regle (2026-09-17,
// D5 et D6 (3.3.1) du plan). Ils ne testent PAS une implementation : ils testent les quatre
// reponses que le verdict doit donner, et chacun rougit pour une raison differente.
//
// LE CINQUIEME CAS EST LE LOT 3.3.2 LUI-MEME, rejoue depuis une fixture extraite de son rapport
// JSON reel (`testdata/verdict_lot_3_3_2.json`) : c'est la seule facon de prouver que la regle
// repond juste sur la mesure qui l'a fait ecrire, et non sur des cas construits pour elle.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/replaydiff"
)

// ecartDe fabrique un ecart tel que `replaydiff` le rend : sens DEJA inverse pour les compteurs
// d'echec (`polarite.go`), donc un `noSlot` qui MONTE arrive en `perte`.
func ecartDe(metrique, ancien, nouveau, sens string) replaydiff.Difference {
	return replaydiff.Difference{Axe: "couverture", Metrique: metrique,
		Ancien: ancien, Nouveau: nouveau, Sens: sens}
}

// ligneDe passe des ecarts par `remplirBilan`, c'est-a-dire par le chemin de production.
func ligneDe(diffs ...replaydiff.Difference) ligneRapport {
	var l ligneRapport
	l.Temoin = Temoin{ID: "temoin", Famille: "famille"}
	l.remplirBilan(replaydiff.Rapport{SchemaAncien: 61, SchemaNouveau: 61, Differences: diffs})
	return l
}

// TestRevisionSeuleNeBloquePas — CAS 1 : une revision qui monte SEULE sort en code 0.
//
// C'est le defaut que ce lot ferme : avant lui, ces trois chaines faisaient echouer les dix-sept
// temoins d'un lot dont rien d'autre ne bougeait.
func TestRevisionSeuleNeBloquePas(t *testing.T) {
	l := ligneDe(
		ecartDe("coverage.decoder.profileRev", "profile-2026-09-17", "profile-2026-09-17.2", replaydiff.SensChangement),
		ecartDe("coverage.decoder.grammarRev", "grammar-2026-09-15.39", "grammar-2026-09-15.40", replaydiff.SensChangement),
		ecartDe("coverage.decoder.factsRev", "killsource-2026-09-16.6", "killsource-2026-09-16.7", replaydiff.SensChangement),
	)
	if l.Gains != 0 || l.Pertes != 0 || l.Changements != 0 {
		t.Fatalf("gains=%d pertes=%d changements=%d, attendu 0/0/0 — la telemetrie ne compte nulle part",
			l.Gains, l.Pertes, l.Changements)
	}
	if len(l.TelemetrieDetail) != 3 {
		t.Fatalf("%d ligne(s) de telemetrie, attendu 3 — elle est ecartee du verdict, pas du rapport",
			len(l.TelemetrieDetail))
	}
	if l.statut() != statutOK {
		t.Fatalf("statut %q, attendu %q", l.statut(), statutOK)
	}
	if got := codeSortie([]ligneRapport{l}, true); got != codeOK {
		t.Fatalf("code = %d, attendu %d — une revision qui monte est attendue a chaque lot", got, codeOK)
	}
}

// TestRejetQuiMonteDUnDenominateurNulEstUnChangement — CAS 2 : le cas du lot 3.3.2.
//
// `noSlot` 0 -> 54 avec `available` 0 -> 289 : le compteur de rejet ne monte que parce qu'il y a
// desormais de la matiere a rejeter. C'est un CHANGEMENT — affiche, instruit, TOUJOURS bloquant.
func TestRejetQuiMonteDUnDenominateurNulEstUnChangement(t *testing.T) {
	l := ligneDe(
		ecartDe("coverage.grenades.available", "0", "289", replaydiff.SensGain),
		ecartDe("coverage.grenades.noSlot", "0", "54", replaydiff.SensPerte),
	)
	if l.Pertes != 0 {
		t.Fatalf("pertes=%d, attendu 0 — un rejet qui monte d'un denominateur NUL n'est pas une perte", l.Pertes)
	}
	if l.Changements != 1 || len(l.ChangementsDetail) != 1 ||
		l.ChangementsDetail[0].Metrique != "coverage.grenades.noSlot" {
		t.Fatalf("changements=%d detail=%v, attendu 1 sur coverage.grenades.noSlot",
			l.Changements, l.ChangementsDetail)
	}
	if l.Gains != 1 {
		t.Fatalf("gains=%d, attendu 1 (available 0 -> 289)", l.Gains)
	}
	if l.statut() != statutChangement {
		t.Fatalf("statut %q, attendu %q", l.statut(), statutChangement)
	}
	if got := codeSortie([]ligneRapport{l}, true); got != codePerte {
		t.Fatalf("code = %d, attendu %d — un changement reste bloquant, le pilote l'instruit", got, codePerte)
	}
}

// TestRejetQuiMonteADenominateurConstantEstUnePerte — CAS 3 : la lecture d'avant ce lot, et elle
// reste juste. `noSlot` 5 -> 40 avec `available` 100 des deux cotes : le rapport se degrade.
//
// Le denominateur est ABSENT des ecarts parce qu'il est IDENTIQUE — c'est precisement la branche
// que ce cas garde : un denominateur inchange ne doit jamais valoir « pas de denominateur ».
func TestRejetQuiMonteADenominateurConstantEstUnePerte(t *testing.T) {
	l := ligneDe(ecartDe("coverage.grenades.noSlot", "5", "40", replaydiff.SensPerte))
	if l.Pertes != 1 || len(l.PertesDetail) != 1 {
		t.Fatalf("pertes=%d, attendu 1 — a denominateur constant, un rejet qui monte EST une perte", l.Pertes)
	}
	if l.statut() != statutPerte {
		t.Fatalf("statut %q, attendu %q", l.statut(), statutPerte)
	}
}

// TestRejetQuiBaisseResteUnGain — CAS 4 : `polarite.go` lit deja les compteurs d'echec a
// l'envers, et ce lot ne touche pas a cette lecture.
func TestRejetQuiBaisseResteUnGain(t *testing.T) {
	l := ligneDe(ecartDe("coverage.grenades.noSlot", "40", "5", replaydiff.SensGain))
	if l.Gains != 1 || l.Pertes != 0 || l.Changements != 0 {
		t.Fatalf("gains=%d pertes=%d changements=%d, attendu 1/0/0", l.Gains, l.Pertes, l.Changements)
	}
	if l.statut() != statutOK {
		t.Fatalf("statut %q, attendu %q", l.statut(), statutOK)
	}
}

// TestRejetSansDenominateurGardeLaMesure — un rejet DECLARE mais sans denominateur publie garde
// la lecture de `replaydiff`. La table le declare avec sa case vide, et c'est ce qui distingue
// « pas de denominateur » de « pas encore inventorie ».
func TestRejetSansDenominateurGardeLaMesure(t *testing.T) {
	l := ligneDe(ecartDe("coverage.bridge.closedRefused", "0", "17", replaydiff.SensPerte))
	if l.Pertes != 1 {
		t.Fatalf("pertes=%d, attendu 1 — sans denominateur, la mesure de replaydiff fait foi", l.Pertes)
	}
	if denom, estRejet := denominateurDe("coverage.bridge.closedRefused"); !estRejet || denom != "" {
		t.Fatalf("denominateurDe = (%q, %v), attendu (\"\", true)", denom, estRejet)
	}
}

// TestTableDesRejetsEstCoherente — LA TABLE EST UNE DONNEE, ET ELLE SE RELIT.
//
// Deux gardes : un denominateur declare n'est jamais lui-meme un rejet du meme bloc (il se
// classerait contre lui-meme), et chaque famille porte sa preuve — une entree recopiee de
// memoire n'a rien a faire dans une table dont tout l'interet est d'etre verifiable.
func TestTableDesRejetsEstCoherente(t *testing.T) {
	for _, f := range rejetsDeCouverture() {
		if len(f.Blocs) == 0 || len(f.Rejets) == 0 {
			t.Errorf("famille vide : %+v", f)
		}
		if strings.TrimSpace(f.Preuve) == "" {
			t.Errorf("famille %v sans preuve : une entree sans source se recopie de memoire", f.Blocs)
		}
		for _, bloc := range f.Blocs {
			if !strings.HasSuffix(bloc, ".") {
				t.Errorf("bloc %q : le prefixe doit finir par un point", bloc)
			}
			for _, r := range f.Rejets {
				if r == f.Denominateur {
					t.Errorf("%s%s est declare a la fois rejet et denominateur", bloc, r)
				}
				if denom, ok := denominateurDe(bloc + r); !ok {
					t.Errorf("%s%s n'est pas retrouve par denominateurDe", bloc, r)
				} else if f.Denominateur != "" && denom != bloc+f.Denominateur {
					t.Errorf("%s%s -> %q, attendu %q", bloc, r, denom, bloc+f.Denominateur)
				}
			}
		}
	}
	if _, ok := denominateurDe("coverage.grenades.attached"); ok {
		t.Error("`attached` est declare comme un rejet : c'est la RICHESSE du calque")
	}
}

// temoinDeFixture : une ligne de `testdata/verdict_lot_3_3_2.json`.
type temoinDeFixture struct {
	Temoin        string                  `json:"temoin"`
	Famille       string                  `json:"famille"`
	SchemaAncien  int                     `json:"schemaAncien"`
	SchemaNouveau int                     `json:"schemaNouveau"`
	Differences   []replaydiff.Difference `json:"differences"`
}

// TestVerdictDuLot332RejoueSurSaFixture — CAS 5 : LA MESURE QUI A FAIT ECRIRE LA REGLE.
//
// La fixture est extraite du rapport JSON REEL du gate du lot 3.3.2 (17 temoins, corpus complet,
// `--base=d620a1a56`) : ses `pertesDetail` et ses `changementsDetail` au complet, plus les deux
// ecarts de `coverage.grenades.{available, attached}` releves sur les artefacts cuits des deux
// cotes. Elle NE PORTE PAS les autres gains du run (le rapport JSON du gate ne detaille que les
// pertes et les changements) : ce test n'affirme donc rien sur le compte de gains, et tout sur
// la CLASSIFICATION — qui est ce que ce lot change.
//
// CE QU'IL PROUVE : avec l'outil corrige, les 17 temoins du lot 3.3.2 ne portent PLUS AUCUNE
// PERTE (les 7 `noSlot` deviennent des changements), la telemetrie disparait du verdict
// (51 lignes), et le code de sortie reste 1 — a cause des 8 changements de verdict grenades et
// des 7 `noSlot`, qui sont exactement ce que le pilote doit instruire.
func TestVerdictDuLot332RejoueSurSaFixture(t *testing.T) {
	brut, err := os.ReadFile(filepath.Join("testdata", "verdict_lot_3_3_2.json"))
	if err != nil {
		t.Fatalf("fixture du lot 3.3.2 : %v", err)
	}
	var fixture []temoinDeFixture
	if err := json.Unmarshal(brut, &fixture); err != nil {
		t.Fatalf("fixture illisible : %v", err)
	}
	if len(fixture) != 17 {
		t.Fatalf("%d temoin(s) dans la fixture, 17 attendus (le corpus complet)", len(fixture))
	}
	var lignes []ligneRapport
	totalPertes, totalChangements, totalTelemetrie := 0, 0, 0
	for _, f := range fixture {
		var l ligneRapport
		l.Temoin = Temoin{ID: f.Temoin, Famille: f.Famille}
		l.remplirBilan(replaydiff.Rapport{SchemaAncien: f.SchemaAncien,
			SchemaNouveau: f.SchemaNouveau, Differences: f.Differences})
		totalPertes += l.Pertes
		totalChangements += l.Changements
		totalTelemetrie += len(l.TelemetrieDetail)
		lignes = append(lignes, l)
	}
	if totalPertes != 0 {
		t.Errorf("%d perte(s) sur les 17 temoins, attendu 0 — la mesure n'en montrait aucune", totalPertes)
	}
	if totalTelemetrie != 51 {
		t.Errorf("%d ligne(s) de telemetrie, attendu 51 (trois revisions x dix-sept temoins)", totalTelemetrie)
	}
	if totalChangements != 15 {
		t.Errorf("%d changement(s), attendu 15 : 8 verdicts de grenades + 7 noSlot", totalChangements)
	}
	if got := codeSortie(lignes, true); got != codePerte {
		t.Errorf("code = %d, attendu %d — les changements restent bloquants, le pilote les instruit",
			got, codePerte)
	}
	for _, l := range lignes {
		if l.statut() == statutPerte {
			t.Errorf("%s sort en PERTE : aucun temoin du lot 3.3.2 n'en portait", l.Temoin.ID)
		}
	}
}

// TestImprimerTelemetrieGroupeParValeur — la section existe, et elle ne repete pas dix-sept fois
// la meme ligne. Sans ce test, le seul appelant du rendu serait `main.go` et personne ne verrait
// une section vide ou dupliquee.
func TestImprimerTelemetrieGroupeParValeur(t *testing.T) {
	rev := ecartDe("coverage.decoder.grammarRev", "grammar-2026-09-15.39",
		"grammar-2026-09-15.40", replaydiff.SensChangement)
	var lignes []ligneRapport
	for _, id := range []string{"aaaa1111", "bbbb2222", "cccc3333"} {
		l := ligneDe(rev)
		l.Temoin = Temoin{ID: id, Famille: "famille"}
		lignes = append(lignes, l)
	}
	var b strings.Builder
	imprimerTelemetrie(&b, lignes)
	sortie := b.String()
	if !strings.Contains(sortie, "TELEMETRIE (1 valeur(s)") {
		t.Fatalf("en-tete absente ou non groupee :\n%s", sortie)
	}
	if !strings.Contains(sortie, "3 temoin(s)") {
		t.Errorf("le compte de temoins manque :\n%s", sortie)
	}
	if n := strings.Count(sortie, "coverage.decoder.grammarRev"); n != 1 {
		t.Errorf("%d ligne(s) pour la meme valeur, attendu 1 — une revision monte pour tous a la fois", n)
	}
	var vide strings.Builder
	imprimerTelemetrie(&vide, []ligneRapport{ligneDe()})
	if vide.String() != "" {
		t.Errorf("section imprimee alors qu aucune telemetrie n a bouge :\n%s", vide.String())
	}
}
