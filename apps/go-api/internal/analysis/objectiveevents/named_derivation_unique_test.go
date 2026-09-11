package objectiveevents

// named_derivation_unique_test.go — LA SERIE PUBLIEE ET LA CLE D'APPARIEMENT NE DERIVENT
// QU'UNE FOIS (correctif 6.R, 2026-09-11).
//
// # LE DEFAUT, TROUVE PAR LE GATE LOCAL DE FIN DE VAGUE 6
//
// Le meme compteur avait DEUX lectures et une seule borne :
//
//	CLE D'APPARIEMENT   [SlotIdentityFrom] -> [countsOf] -> [incrementTimes] : la borne par pas
//	                    ([maxUnrollPerStep]) s'y appliquait depuis le lot 6.7-B1.
//	SERIE PUBLIEE       [SeriesTotal] / [SeriesByRound] -> `replay.scoreTicksOf` : AUCUNE borne.
//
// Sur la fixture `c0a82e88`, le slot 12 deroulait 60 assistances d'un coup alors que sa feuille
// de match en porte ZERO. La cle lisait 0 et nommait le joueur (progres de la vague 6 : 8 slots
// pontes sur 8) ; la courbe de score, elle, servait 60 a l'ecran. AVANT la vague le joueur
// n'etait pas ponte donc pas publie, et l'ecart restait invisible.
//
// # CE QUE CES TESTS TIENNENT
//
// La borne n'est plus lue qu'a UN endroit ([boundSteps]) et les deux formes en descendent :
// [incrementTimes] pour les EVENEMENTS, [boundedSeries] pour la SERIE. Les tests ci-dessous
// prouvent l'ACCORD des deux sur la meme table d'enregistrements — pas seulement que chacune
// prise a part donne un chiffre plausible.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// derivRecs fabrique la table de l'affaire : un slot SAIN qui progresse a 1 par emission, et un
// slot ABERRANT dont les assistances sautent d'un coup au-dela de la borne alors que sa feuille
// en porte zero — la forme exacte du slot 12 de `c0a82e88`.
//
// `assistsAberrant` est parametre pour que le test du cas SAIN reutilise la meme table.
func derivRecs(assistsAberrant int) []StatRecord {
	var out []StatRecord
	// Le slot 10 progresse normalement : 3 frags, 3 morts, 3 assistances.
	for n := 1; n <= 3; n++ {
		out = append(out, recKDA(1_000+n*1_000, 10, 0, n, n, n))
	}
	// Le slot 12 : 5 frags, 0 mort, et UNE emission d'assistances hors norme.
	for n := 1; n <= 5; n++ {
		out = append(out, recKDA(1_500+n*1_000, 12, 0, n, 0, 0))
	}
	out = append(out, recKDA(9_000, 12, 0, 5, 0, assistsAberrant))
	return out
}

// derivFeuille : la feuille de match des deux slots — le slot 12 y porte ZERO assistance.
var derivFeuille = []PlayerLine{
	{XUID: "aaa", Kills: 3, Deaths: 3, Assists: 3},
	{XUID: "bbb", Kills: 5, Deaths: 0, Assists: 0},
}

// finalDe rend la derniere valeur d'une serie, ou -1 quand elle est absente.
func finalDe(pts []ScorePoint) int64 {
	if len(pts) == 0 {
		return -1
	}
	return pts[len(pts)-1].Value
}

// TestSeriePublieeEtCleAppariementSAccordent — LE TEST QUI FONDE LE CORRECTIF 6.R.
//
// MUTATION JOUEE : retirer `boundedSeries` de [SeriesTotal] (score.go) rend une serie qui finit
// a 60 alors que la cle en compte 0, et le test echoue en nommant les deux chiffres.
func TestSeriePublieeEtCleAppariementSAccordent(t *testing.T) {
	const aberrant = 60 // au-dela de maxUnrollPerStep = 16 : le pas 0 -> 60 est refuse
	if aberrant <= maxUnrollPerStep {
		t.Fatalf("le vecteur n'est plus aberrant : %d <= %d", aberrant, maxUnrollPerStep)
	}
	recs := derivRecs(aberrant)

	// LA CLE : ce que le pont d'identite compte comme increments.
	b := newEventBudget("test")
	cle := countsOf(recs, statSlotKey{coreAssistsComp, sideA}, b)
	if cle[12] != 0 {
		t.Fatalf("la cle compte %d assistance(s) pour le slot 12, attendu 0 — la borne par pas "+
			"ne joue plus, le vecteur du test ne prouve rien", cle[12])
	}
	if b.rejetes != 1 {
		t.Fatalf("%d deroulage(s) rejete(s), attendu 1", b.rejetes)
	}

	// LA SERIE : ce que le document publie.
	serie := SeriesTotal(recs, AssistsComponent, false)
	if got := finalDe(serie[12]); got != int64(cle[12]) {
		t.Errorf("SERIE PUBLIEE ≠ CLE D'APPARIEMENT pour le slot 12 : la serie finit a %d "+
			"assistances, la cle en compte %d — la borne par pas ne descend pas dans la serie",
			got, cle[12])
	}
	// Le slot SAIN n'est pas touche : la borne ne doit pas payer un film propre.
	if got, want := finalDe(serie[10]), int64(cle[10]); got != want || want != 3 {
		t.Errorf("slot sain : serie %d, cle %d, attendu 3 pour les deux", got, want)
	}

	// La forme par MANCHE doit dire la meme chose que le total : c'est elle que le chemin
	// multi-manche (`replay.buildPlayerScoresByRound`) recompose.
	parManche := SeriesByRound(recs, AssistsComponent, false)
	if got := finalDe(parManche[12][0]); got != 0 {
		t.Errorf("serie PAR MANCHE du slot 12 : finit a %d, attendu 0", got)
	}

	// ET LE PONT NOMME LE JOUEUR : c'est ce qui rend l'ecart VISIBLE a l'ecran depuis la
	// vague 6 — un slot non ponte n'etait pas publie du tout.
	if id := SlotIdentityFrom(recs, derivFeuille); id[12] != "bbb" {
		t.Errorf("le slot 12 est apparie a %q, attendu \"bbb\" : le vecteur ne reproduit plus "+
			"la situation du defaut", id[12])
	}
}

// TestSerieSaineTraverseLaDerivationIntacte — la contre-epreuve : un pas SOUS la borne passe
// entier, dans la serie comme dans la cle. Sans ce test, remplacer `boundedSeries` par « rendre
// zero partout » passerait le test precedent.
func TestSerieSaineTraverseLaDerivationIntacte(t *testing.T) {
	const sain = 3 // pire pas sain mesure sur le parc (lot 6.7-B1, item 6)
	recs := derivRecs(sain)

	b := newEventBudget("test")
	cle := countsOf(recs, statSlotKey{coreAssistsComp, sideA}, b)
	if b.rejetes != 0 {
		t.Fatalf("%d rejet(s) sur un vecteur SAIN", b.rejetes)
	}
	if cle[12] != sain {
		t.Fatalf("la cle compte %d, attendu %d", cle[12], sain)
	}
	if got := finalDe(SeriesTotal(recs, AssistsComponent, false)[12]); got != sain {
		t.Errorf("la serie finit a %d, attendu %d — la borne a coupe un compteur sain", got, sain)
	}
}

// TestBoundedSeriesGardeLaFormeDeLaSuite — [boundedSeries] recalcule les VALEURS et ne touche ni
// aux instants ni a la cardinalite.
//
// POURQUOI CETTE PROPRIETE COMPTE : un compteur reste a zero est represente par une serie de
// points a zero, et `replay.PlayerScore.empty()` lit cette presence. Une derivation qui jetterait
// les paliers ferait DISPARAITRE du document un joueur dont les quatre compteurs sont a zero.
func TestBoundedSeriesGardeLaFormeDeLaSuite(t *testing.T) {
	pts := []ScorePoint{pt(100, 0), pt(200, 1), pt(300, 1), pt(400, 1_000), pt(500, 1_002)}
	got := boundedSeries(pts)

	if len(got) != len(pts) {
		t.Fatalf("%d points rendus, attendu %d", len(got), len(pts))
	}
	for i := range got {
		if got[i].TimeMS != pts[i].TimeMS || got[i].Slot != pts[i].Slot {
			t.Fatalf("point %d deplace : {%d, slot %d} au lieu de {%d, slot %d}",
				i, got[i].TimeMS, got[i].Slot, pts[i].TimeMS, pts[i].Slot)
		}
	}
	// 0, +1, palier, PAS REJETE (999 > 16), puis +2 comptes depuis 1 000 : le cumul retenu
	// vaut 1 puis 3. `prev` avance malgre le rejet — sinon le dernier point rejouerait 1 002.
	for i, want := range []int64{0, 1, 1, 1, 3} {
		if got[i].Value != want {
			t.Errorf("point %d : valeur %d, attendu %d (suite rendue : %+v)", i, got[i].Value, want, got)
		}
	}
}

// TestBoundedSeriesEtIncrementTimesNeDiventJamaisDeuxChoses — la propriete GENERALE, sur un
// echantillon de formes : le dernier point de [boundedSeries] vaut toujours le nombre d'instants
// rendus par [incrementTimes]. C'est l'invariant que le defaut 6.R violait.
func TestBoundedSeriesEtIncrementTimesNeDiventJamaisDeuxChoses(t *testing.T) {
	cas := map[string][]ScorePoint{
		"serie vide":           {},
		"palier seul":          {pt(100, 0), pt(200, 0)},
		"progression douce":    {pt(100, 1), pt(200, 2), pt(300, 5)},
		"premier terme enorme": {pt(100, 1_000_000), pt(200, 1_000_001)},
		"saut au milieu":       {pt(100, 1), pt(200, 65), pt(300, 66)},
		"deux sauts":           {pt(100, 40), pt(200, 41), pt(300, 200), pt(400, 201)},
		"pile a la borne":      {pt(100, maxUnrollPerStep), pt(200, maxUnrollPerStep+1)},
		"un cran au-dela":      {pt(100, maxUnrollPerStep+1)},
	}
	for nom, pts := range cas {
		serie := boundedSeries(pts)
		evts := incrementTimes(pts, statSlotKey{coreAssistsComp, sideA}, newEventBudget("test"))
		var total int64
		if len(serie) > 0 {
			total = serie[len(serie)-1].Value
		}
		if total != int64(len(evts)) {
			t.Errorf("%s : la serie finit a %d, la cle compte %d increments", nom, total, len(evts))
		}
	}
}

// TestUneSeuleLectureDeLaBorneParPas — LE GARDE-RAIL DE LA FACTORISATION (regle n° 6 du depot :
// une centralisation sans garde-rail re-diverge).
//
// [maxUnrollPerStep] ne doit etre lu QUE par [boundSteps]. Toute autre lecture dans le code de
// production est, par construction, une seconde derivation — c'est-a-dire le defaut 6.R qui
// revient. Les tests, eux, ont le droit de s'y adosser pour calibrer leurs vecteurs.
//
// L'ALLOWLIST TIENT EN DEUX ENTREES, ET LA SECONDE NE DECIDE RIEN (2026-09-11) :
// [eventBudget.rejeter] IMPRIME la borne dans son avertissement pour que la ligne de journal se
// lise seule (« deroulage 60, borne 16 »). Elle ne la compare a rien. Un troisieme nom dans
// cette liste doit etre justifie et date, comme toute allowlist du depot.
func TestUneSeuleLectureDeLaBorneParPas(t *testing.T) {
	// autorisees : `boundSteps` DECIDE, `rejeter` JOURNALISE (cf. l'en-tete du test).
	autorisees := map[string]bool{"boundSteps": true, "rejeter": true}

	racine, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("racine du module introuvable : %v", err)
	}
	var fautives []string
	err = filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
			return err
		}
		brut, err := os.ReadFile(chemin)
		if err != nil {
			return err
		}
		fonction := ""
		for i, ligne := range strings.Split(string(brut), "\n") {
			if strings.HasPrefix(ligne, "func ") {
				fonction = nomDeFonction(ligne)
			}
			nu := strings.TrimSpace(ligne)
			switch {
			case !strings.Contains(nu, "maxUnrollPerStep"):
			case strings.HasPrefix(nu, "//"): // prose : ce n'est pas une lecture
			case strings.HasPrefix(nu, "maxUnrollPerStep ="): // la declaration de la constante
			case autorisees[fonction]:
			default:
				rel, _ := filepath.Rel(racine, chemin)
				fautives = append(fautives, fmt.Sprintf("%s:%d (fonction %q) : %s",
					filepath.ToSlash(rel), i+1, fonction, nu))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage du module : %v", err)
	}
	if len(fautives) > 0 {
		t.Errorf("maxUnrollPerStep est lu hors de `boundSteps` — c'est une SECONDE derivation des "+
			"increments filtres, exactement le defaut 6.R :\n  %s", strings.Join(fautives, "\n  "))
	}
}

// nomDeFonction extrait le nom d'une declaration `func ...`, methodes comprises.
func nomDeFonction(ligne string) string {
	reste := strings.TrimPrefix(ligne, "func ")
	if strings.HasPrefix(reste, "(") { // methode : sauter le recepteur
		if i := strings.Index(reste, ")"); i >= 0 {
			reste = strings.TrimSpace(reste[i+1:])
		}
	}
	if i := strings.IndexAny(reste, "([ "); i >= 0 {
		reste = reste[:i]
	}
	return reste
}
