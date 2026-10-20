package main

// verite_test.go — LE VERDICT DU GATE SUIT CELUI DU BANC DE VERITE (D-5, 2026-09-30).

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/replaybuild"
	"levelup/go-api/internal/replaydiff"
	"levelup/go-api/internal/replayverite"
)

// comparaisonDe fabrique un verdict du banc, avec les mesures presentes des deux cotes.
func comparaisonDe(statut replayverite.Statut, mesures ...string) *replayverite.Comparaison {
	b := replayverite.Bulletin{Scores: map[string]replayverite.Score{}, Preuves: map[string]replayverite.Preuve{}}
	for _, m := range mesures {
		if strings.HasPrefix(m, "P-") {
			b.Preuves[m] = replayverite.Preuve{Valeur: 1}
			continue
		}
		b.Scores[m] = replayverite.Score{VP: 1}
	}
	c := &replayverite.Comparaison{Statut: statut, Avant: b, Apres: b}
	if statut != replayverite.StatutOK {
		c.Constats = []replayverite.Constat{{Mesure: "V-4 deux corps", Sens: statut, Avant: "0", Apres: "1",
			Detail: []string{"+ 111 vies @0 et @5 (3 images)"}}}
	}
	return c
}

func perte(axe, metrique, ancien, nouveau string) replaydiff.Difference {
	return replaydiff.Difference{Axe: axe, Metrique: metrique, Ancien: ancien, Nouveau: nouveau, Sens: replaydiff.SensPerte}
}

// ligneJugee passe des ecarts par `remplirBilan` APRES avoir pose le verdict du banc — l'ordre de
// production (orchestrate.go).
func ligneJugee(c *replayverite.Comparaison, diffs ...replaydiff.Difference) ligneRapport {
	l := ligneRapport{Temoin: Temoin{ID: "temoin", Famille: "famille"}, Verite: c}
	l.remplirBilan(replaydiff.Rapport{SchemaAncien: 76, SchemaNouveau: 76, Differences: diffs})
	return l
}

// TestVerdictDuGateSuitLeBanc : FAUX et MANQUE du banc bloquent ; un banc `ok` avec des pertes
// COUVERTES et des changements rend `ok`, code 0.
func TestVerdictDuGateSuitLeBanc(t *testing.T) {
	couverte := perte("pistes", "tracks/vies-par-xuid/111", "3", "2")
	chang := replaydiff.Difference{Axe: "couverture", Metrique: "coverage.bridge.namedByNextLife",
		Ancien: "13", Nouveau: "8", Sens: replaydiff.SensChangement}
	cas := []struct {
		nom    string
		ligne  ligneRapport
		statut string
		code   int
	}{
		{"banc FAUX", ligneJugee(comparaisonDe(replayverite.StatutFaux)), statutFaux, codePerte},
		{"banc MANQUE", ligneJugee(comparaisonDe(replayverite.StatutManque)), statutManque, codePerte},
		{"banc ok, perte couverte et changement",
			ligneJugee(comparaisonDe(replayverite.StatutOK, replayverite.ScoreMortsVies), couverte, chang),
			statutOK, codeOK},
		{"banc FAUX prime sur un filet",
			ligneJugee(comparaisonDe(replayverite.StatutFaux), perte("armes", "shots/n", "10", "9")),
			statutFaux, codePerte},
	}
	for _, k := range cas {
		if got := k.ligne.statut(); got != k.statut {
			t.Errorf("%s : statut %q, veut %q", k.nom, got, k.statut)
		}
		if got := codeSortie([]ligneRapport{k.ligne}, true); got != k.code {
			t.Errorf("%s : code %d, veut %d", k.nom, got, k.code)
		}
	}
}

// TestFilets_PerteNonCouverteEtCalqueDisparu : une perte hors de tout bloc couvert bloque (PERTE),
// un calque qui disparait bloque meme dans un bloc couvert, une perte couverte par une mesure
// ABSENTE d'un cote ne l'est pas.
func TestFilets_PerteNonCouverteEtCalqueDisparu(t *testing.T) {
	ok := func(mesures ...string) *replayverite.Comparaison {
		return comparaisonDe(replayverite.StatutOK, mesures...)
	}
	cas := []struct {
		nom    string
		c      *replayverite.Comparaison
		d      replaydiff.Difference
		filets int
	}{
		{"arme non couverte", ok(replayverite.ScoreKills), perte("armes", "shots/n", "10", "9"), 1},
		{"kills couverts", ok(replayverite.ScoreKills), perte("score", "joueur/111/kills", "5", "4"), 0},
		{"kills sans la mesure", ok(), perte("score", "joueur/111/kills", "5", "4"), 1},
		{"score personnel sans oracle", ok(replayverite.ScoreKills), perte("score", "joueur/111/score", "500", "400"), 1},
		{"score personnel avec oracle", ok(replayverite.ScorePersonnel), perte("score", "joueur/111/score", "500", "400"), 0},
		{"O-S3 ne couvre pas les kills", ok(replayverite.ScorePersonnel), perte("score", "joueur/111/kills", "5", "4"), 1},
		{"fermeture couverte par P-1", ok(replayverite.PreuveFermeture), perte("couverture", "coverage.continuousFire.closed", "9", "8"), 0},
		{"verdict couvert par P-4", ok(replayverite.PreuveVerdicts + " shots"), perte("couverture", "coverage.verdict.shots", "nominal", ""), 0},
		{"calque de pistes disparu", ok(replayverite.ScoreFinsDeVie), perte("pistes", "tracks/n", "80", "0"), 1},
		{"calque sans cle", ok(), replaydiff.Difference{Axe: "objectifs", Metrique: "objectives/n", Ancien: "27", Sens: replaydiff.SensDisparu}, 1},
		{"revision de calque retiree", ok(replayverite.ScoreKills, replayverite.ScoreFinsDeVie, replayverite.PreuveFermeture),
			perte("autres:layers", "layers/n", "40", "39"), 1},
		{"sans banc, toute perte bloque", nil, perte("score", "joueur/111/kills", "5", "4"), 1},
	}
	for _, k := range cas {
		l := ligneJugee(k.c, k.d)
		if len(l.Filets) != k.filets {
			t.Errorf("%s : %d filet(s) %v, veut %d", k.nom, len(l.Filets), l.Filets, k.filets)
		}
		if k.filets > 0 && l.statut() != statutPerte {
			t.Errorf("%s : statut %q, veut %q", k.nom, l.statut(), statutPerte)
		}
	}
}

// TestO_S1CirculaireNeCouvreRien : un score non note (circulaire) n'est pas une mesure presente.
func TestO_S1CirculaireNeCouvreRien(t *testing.T) {
	c := comparaisonDe(replayverite.StatutOK, replayverite.ScoreKills)
	c.Avant.Scores[replayverite.ScoreKills] = replayverite.Score{NonNote: true}
	if l := ligneJugee(c, perte("score", "joueur/111/kills", "5", "4")); len(l.Filets) != 1 {
		t.Errorf("une mesure non notee couvre la perte : filets %v", l.Filets)
	}
}

// decompresserArtefactReel ecrit, dans un dossier temporaire, l'artefact reel elague du banc.
func decompresserArtefactReel(t *testing.T) (string, replaybuild.FactsFile) {
	t.Helper()
	dir := filepath.Join("..", "..", "internal", "replayverite", "testdata")
	f, err := os.Open(filepath.Join(dir, "bcb6d393.json.gz")) //nolint:gosec // chemin fige du test
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	blob, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "bcb6d393.json")
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatal(err)
	}
	faits, err := replaybuild.ReadFactsFile(filepath.Join(dir, "bcb6d393.facts.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path, faits
}

// TestRapportDuGate_BancAvantLeDetailSurUnArtefactReel : sur l'artefact reel, un HEAD abime (une
// vie dupliquee) sort FAUX ; la section du banc precede le detail `replaydiff` ; le JSON la porte.
func TestRapportDuGate_BancAvantLeDetailSurUnArtefactReel(t *testing.T) {
	ref, faits := decompresserArtefactReel(t)
	var doc map[string]any
	blob, _ := os.ReadFile(ref) //nolint:gosec // chemin du test
	if err := json.Unmarshal(blob, &doc); err != nil {
		t.Fatal(err)
	}
	pistes := doc["tracks"].([]any)
	doc["tracks"] = append(pistes, pistes[0])
	abime, _ := json.Marshal(doc)
	head := filepath.Join(t.TempDir(), "head.json")
	if err := os.WriteFile(head, abime, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := juger(jugement{Reference: ref, HEAD: head, Faits: faits.MatchFacts})
	if err != nil {
		t.Fatal(err)
	}
	l := ligneRapport{Temoin: Temoin{ID: "bcb6d393", Famille: "ctf_mono_manche"}, Verite: c}
	rap, err := compareTemoin(ref, head)
	if err != nil {
		t.Fatal(err)
	}
	l.remplirBilan(rap)
	if l.statut() != statutFaux {
		t.Fatalf("statut %q, veut %q (%+v)", l.statut(), statutFaux, c.Bloquants())
	}
	var b bytes.Buffer
	imprimerTableau(&b, []ligneRapport{l}, "base")
	imprimerVerite(context.Background(), &b, []ligneRapport{l}, true)
	imprimerDetailPertes(&b, []ligneRapport{l})
	imprimerDetailChangements(&b, []ligneRapport{l})
	out := b.String()
	iBanc, iDetail := strings.Index(out, "BANC DE VERITE"), strings.Index(out, "DETAIL DES")
	if iBanc < 0 || (iDetail >= 0 && iDetail < iBanc) || !strings.Contains(out, "[FAUX] V-4 deux corps") {
		t.Fatalf("section du banc absente ou apres le detail :\n%s", out)
	}
	lj := ligneVersJSON(l)
	if lj.Verite == nil || lj.Verite.Statut != string(replayverite.StatutFaux) || len(lj.Verite.Constats) == 0 {
		t.Fatalf("section verite du JSON = %+v", lj.Verite)
	}
}

// TestRegistreDeLaBase_SansOutilEstInconnu : une base anterieure au banc n'a pas l'outil ; le
// registre est inconnu (nil), sans compilation tentee.
func TestRegistreDeLaBase_SansOutilEstInconnu(t *testing.T) {
	if reg := registreDeLaBase(t.Context(), t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "x")); reg != nil {
		t.Fatalf("registre = %v, veut nil", reg)
	}
}

// TestImprimerVerite_RegistreInconnuLeDit : le rapport dit quand le registre d'avant est inconnu.
func TestImprimerVerite_RegistreInconnuLeDit(t *testing.T) {
	var b bytes.Buffer
	imprimerVerite(context.Background(), &b, []ligneRapport{ligneJugee(comparaisonDe(replayverite.StatutFaux))}, false)
	if !strings.Contains(b.String(), "registre des replis d'avant INCONNU") {
		t.Errorf("rendu :\n%s", b.String())
	}
	b.Reset()
	imprimerVerite(context.Background(), &b, []ligneRapport{{Temoin: Temoin{ID: "x"}, Absent: true}}, false)
	if b.String() != "" {
		t.Errorf("aucun temoin juge : section attendue vide, obtenu %q", b.String())
	}
}

// TestRapportDuGate_ReattributionVisibleNonBloquante — revue finale P1-e (2026-10-02) : un kill qui
// passe d'un joueur juste (111) a un joueur qui en manquait un (222) laisse les totaux FP/FN
// egaux. Le temoin reste `ok` (on ne sait pas lequel est juste), mais la section du banc du
// rapport texte et le JSON montrent la reattribution, joueur par joueur.
func TestRapportDuGate_ReattributionVisibleNonBloquante(t *testing.T) {
	bulletin := func(ecarts map[string]replayverite.Ecart) replayverite.Bulletin {
		return replayverite.Bulletin{Scores: map[string]replayverite.Score{
			replayverite.ScoreKills: {VP: 3, FN: 1, Ecarts: ecarts},
		}}
	}
	c := replayverite.Comparer(bulletin(map[string]replayverite.Ecart{"222": {Pub: 0, Off: 1}}),
		bulletin(map[string]replayverite.Ecart{"111": {Pub: 2, Off: 3}}), nil)
	l := ligneJugee(&c)
	if l.statut() != statutOK {
		t.Fatalf("statut %q, veut %q : une reattribution n'est pas bloquante", l.statut(), statutOK)
	}
	var b bytes.Buffer
	imprimerVerite(context.Background(), &b, []ligneRapport{l}, true)
	if out := b.String(); !strings.Contains(out, "[reattribution] "+replayverite.ScoreKills) ||
		!strings.Contains(out, "111 : exact -> publie 2 / officiel 3") ||
		!strings.Contains(out, "222 : publie 0 / officiel 1 -> exact") {
		t.Fatalf("la section du banc ne montre pas la reattribution joueur par joueur :\n%s", out)
	}
	v := veriteVersJSON(l)
	if v == nil || len(v.Constats) != 1 || v.Constats[0].Sens != "reattribution" || len(v.Constats[0].Detail) != 2 {
		t.Fatalf("JSON du banc = %+v, veut un constat de reattribution a deux joueurs", v)
	}
}
