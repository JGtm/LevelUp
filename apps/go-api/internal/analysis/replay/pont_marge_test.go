package replay

// pont_marge_test.go — LA MARGE DU CALAGE : ce qui EMPECHE le vote de se tromper en silence.
//
// Sorti de `pont_muet_test.go` apres la ronde PONT-R2 : ce fichier-la franchissait les 500
// lignes du depot. Le decoupage suit la frontiere du sujet — `pont_muet_test.go` porte le
// CALAGE et le ROSTER (ce que le lot repare), celui-ci porte la SURVEILLANCE du calage : la
// marge publiee, son alarme, la fusion des paniers voisins dont depend son exactitude, et la
// limite connue du filet. Les fixtures communes (`pontVie`, `pontFixture`) restent chez le
// premier ; meme paquet, aucun duplicata.

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// TestLaMargeDuCalageEstPubliee — la paire (retenu, suivant) doit sortir de `bestDeathOffset`,
// sinon rien ne permet de voir un vote trompé. Sur un film sain la marge est franche.
func TestLaMargeDuCalageEstPubliee(t *testing.T) {
	lives, deaths := pontFixtureAmas(5_000_000, 30_000, 40, 12, 4, 400_040)

	_, n, second := bestDeathOffset(lives, deaths)

	if n != 40 {
		t.Fatalf("appariements = %d, attendu 40", n)
	}
	if second == 0 {
		t.Fatalf("second candidat = 0 : l'amas de bruit devrait en former un, "+
			"la marge ne serait pas mesurée (retenu %d)", n)
	}
	if n < deathOffsetMargeMin*second {
		t.Fatalf("marge %d/%d sous le seuil de %d sur une fixture pourtant nette",
			n, second, deathOffsetMargeMin)
	}
}

// TestUnCalageSansConcurrentNAPasDeSecond — sans amas, il n'y a qu'un candidat : le second
// compte est nul, et la garde de marge ne doit pas se déclencher pour autant.
func TestUnCalageSansConcurrentNAPasDeSecond(t *testing.T) {
	lives, deaths := pontFixture(2_000_000, 20_000, 16)

	_, n, second := bestDeathOffset(lives, deaths)

	if n != 16 {
		t.Fatalf("appariements = %d, attendu 16", n)
	}
	if second*deathOffsetMargeMin > n {
		t.Fatalf("second = %d contre %d : la garde s'alarmerait sur un film sain", second, n)
	}
}

// TestLAlarmeDeMargeEtroiteSeDeclencheEtSeTait — LE TEST DE MUTATION du SEUIL (constat
// PONT-R2/D1). La moitié du remède de PONT-R1/C1 est de publier la marge ; l'autre est de
// l'ALARMER quand elle se resserre, et rien ne gardait ni le seuil ni l'appel : mettre
// `deathOffsetMargeMin` à 0 rendait la garde vacuement vraie sans faire rougir un seul test.
//
// Rouge si le seuil est neutralisé, si l'opérateur de comparaison est renversé, ou si l'appel
// disparaît de `buildCoverage`.
func TestLAlarmeDeMargeEtroiteSeDeclencheEtSeTait(t *testing.T) {
	prev := slog.Default()
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)
	const motif = "calage du fil des morts trop peu distinct"

	// Marge ÉTROITE : le calage retenu ne fait pas deux fois mieux que son suivant.
	BridgeHealth{DeathOffsetMatched: 9, DeathOffsetRunnerUp: 8, LivesTotal: 87, Slots: 84}.
		warnIfCalageEtroit()
	if !strings.Contains(buf.String(), motif) {
		t.Fatalf("marge 9:8 — aucune alarme, le seuil de %d n'est pas gardé. Journal : %q",
			deathOffsetMargeMin, buf.String())
	}

	// Marge FRANCHE : le silence est la réponse attendue. Les deux témoins du parc sont ici
	// (x7,1 sur `51ebbc0f`, x7,9 sur `d9781168`) : ils ne doivent produire aucun bruit.
	for _, cas := range []struct{ matched, runnerUp int }{{71, 10}, {143, 18}, {40, 15}} {
		buf.Reset()
		BridgeHealth{DeathOffsetMatched: cas.matched, DeathOffsetRunnerUp: cas.runnerUp}.
			warnIfCalageEtroit()
		if strings.Contains(buf.String(), motif) {
			t.Fatalf("marge %d:%d — alarme sur un calage pourtant franc : %q",
				cas.matched, cas.runnerUp, buf.String())
		}
	}

	// Pont non construit : aucun calage, donc aucune alarme à donner.
	buf.Reset()
	BridgeHealth{}.warnIfCalageEtroit()
	if strings.Contains(buf.String(), motif) {
		t.Fatalf("alarme sur un pont vide : %q", buf.String())
	}

	// L'APPEL fait partie du contrat : une garde jamais appelée ne garde rien.
	buf.Reset()
	own := OwnerReport{
		Owner: map[uint32]int{1: 0}, SlotXUID: map[uint32]uint64{1: 11},
		DeathOffsetMatches: 9, DeathOffsetRunnerUp: 8, DeathsNamed: 9, LivesTotal: 87,
	}
	buildCoverage(LayerCoverage{}, LayerCoverage{}, LayerCoverage{}, own, true, nil)
	if !strings.Contains(buf.String(), motif) {
		t.Fatalf("buildCoverage n'appelle plus la garde de marge : %q", buf.String())
	}
}

// pontFixturePlateauADeuxPaniers construit un film SANS la moindre ambiguïté dont le vrai
// plateau est à cheval sur DEUX paniers de vote.
//
// Vingt-cinq paires ont un résidu nul et quinze un résidu de 220 ms : aucun offset n'apparie
// un groupe seul à 150 ms près, mais l'intervalle [70, 150] les apparie TOUS LES QUARANTE. Le
// vote, lui, dépose ses voix dans deux paniers distants de 220 ms — plus d'une largeur de
// panier (150 ms), donc deux candidats distincts, chacun à moins de deux largeurs (300 ms) du
// plateau commun, que l'affinage retrouve donc DEUX FOIS.
func pontFixturePlateauADeuxPaniers() ([]lifeSpan, []Death) {
	var lives []lifeSpan
	var deaths []Death
	const origine, residuTardif = 2_000_000, 220
	tMatch := int64(40_000)
	for i := 0; i < 40; i++ {
		var residu int64
		if i >= 25 {
			residu = residuTardif
		}
		lives = append(lives, pontVie(uint32(500+i), origine+tMatch+residu))
		deaths = append(deaths, Death{XUID: uint64(1000 + i), TimeMS: tMatch})
		tMatch += 3_000 + int64(i*2_777)%9_000
	}
	return lives, deaths
}

// TestDeuxPaniersDuMemePlateauNeComptentQuUneFois — LE TEST DE MUTATION de la fusion (constat
// PONT-R2/D2). Rouge si le bloc de fusion de `bestDeathOffset` est neutralisé : le même plateau,
// atteint deux fois, se compte alors comme deux candidats et la marge publiée devient 40:40 —
// une égalité 1:1 sur un film sans la moindre ambiguïté, qui déclencherait `slog.Warn` et
// ferait passer un calage parfait pour du bruit.
func TestDeuxPaniersDuMemePlateauNeComptentQuUneFois(t *testing.T) {
	lives, deaths := pontFixturePlateauADeuxPaniers()

	// Le vote rend bien DEUX candidats bruts distants de plus d'une largeur de panier : sans
	// cela la fixture ne prouverait rien.
	candidats := voteDeathOffsets(lifeEndsMS(lives), deaths, deathOffsetCandidats)
	if len(candidats) < 2 || absI64(candidats[0]-candidats[1]) <= deathMatchWindowMS {
		t.Fatalf("candidats %v : la fixture n'expose plus deux paniers distincts", candidats)
	}

	off, n, second := bestDeathOffset(lives, deaths)

	if n != 40 {
		t.Fatalf("appariements = %d, attendu 40 : le plateau commun n'est pas trouvé", n)
	}
	if second >= n {
		t.Fatalf("marge publiée %d:%d — les deux candidats du MÊME plateau (affinés à %d) "+
			"comptent deux fois : `deathOffsetRunnerUp` est faux et l'alarme se déclencherait "+
			"sur un film sain", n, second, off)
	}
}

// pontFixtureAmasPlusGrosQueLeVrai est la fixture ADVERSARIALE de PONT-R2/D3 : un vrai calage
// propre de quinze paires 1:1, et un amas de VINGT morts distinctes — plus que le vrai calage
// n'en apparie — groupées dans un seul panier mais ne disposant que de cinq fins de vie.
func pontFixtureAmasPlusGrosQueLeVrai() ([]lifeSpan, []Death) {
	var lives []lifeSpan
	var deaths []Death
	const origine, amas, decalageAmas = 200_000, 400_000, 50_000
	tMatch := int64(10_000)
	for i := 0; i < 15; i++ {
		lives = append(lives, pontVie(uint32(500+i), origine+tMatch))
		deaths = append(deaths, Death{XUID: uint64(1000 + i), TimeMS: tMatch})
		tMatch += 11_000 + int64(i*i*7_919)%37_000
	}
	for j := 0; j < 20; j++ {
		deaths = append(deaths, Death{XUID: uint64(9000 + j), TimeMS: amas + int64(j)*3})
	}
	for j := 0; j < 5; j++ {
		lives = append(lives, pontVie(uint32(800+j), origine+decalageAmas+amas+int64(j)*3))
	}
	return lives, deaths
}

// TestUnAmasPlusGrosQueLeVraiCalageALARMEAuLieuDeSeTaire — TEST DE DOCUMENTATION du report
// PONT-R2/D3. Il ne décrit pas un comportement souhaitable : il FIGE le fait que la limite
// connue du filet ne se franchit pas en silence.
//
// LA LIMITE. Le dédoublonnage par mort (correctif de PONT-R1/C1) a un effet de bord : un amas
// de M morts distinctes groupées dépose M voix dans un panier DIFFÉRENT pour CHACUNE des fins
// séparées du vrai calage. Dès que M dépasse le nombre de morts du vrai calage et que celui-ci
// a au moins `deathOffsetCandidats` fins séparées, ces « paniers fantômes » remplissent à eux
// seuls le budget de candidats : ni le vrai calage ni celui de l'amas n'est jamais examiné.
//
// POURQUOI CE N'EST PAS CORRIGÉ ICI. La précondition est extrême — il faut un amas de morts
// DISTINCTES plus nombreux que la totalité des morts appariables du film, quand un multi-kill
// réel plafonne à trois ou quatre. Non mesurée sur les 106 films du parc. Le report et sa
// condition de reprise sont au registre (`.ai/V7.5/REGISTRE_REPORTS.md`).
//
// CE QUE CE TEST GARANTIT : dans ce cas, le calage retenu n'atteint pas le vrai (15) ET
// l'alarme de marge SE DÉCLENCHE. Si un correctif futur fait mieux, ce test rougit — et c'est
// exactement ce qu'on attend de lui.
func TestUnAmasPlusGrosQueLeVraiCalageALARMEAuLieuDeSeTaire(t *testing.T) {
	lives, deaths := pontFixtureAmasPlusGrosQueLeVrai()
	const vraiCalage = 200_000

	// Le vrai calage EXISTE : un balayage exhaustif le trouverait. C'est la LOCALISATION en
	// trois candidats qui échoue, pas l'affinage.
	if reel := countDeathMatches(lifeEndsMS(lives), deaths, vraiCalage); reel != 15 {
		t.Fatalf("le vrai calage apparie %d morts, attendu 15 : la fixture a dérivé", reel)
	}

	_, n, second := bestDeathOffset(lives, deaths)

	if n >= 15 {
		t.Fatalf("le calage retenu apparie %d : la limite de PONT-R2/D3 est franchie — "+
			"retirer ce test de documentation et fermer le report au registre", n)
	}
	prev := slog.Default()
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(prev)

	BridgeHealth{DeathOffsetMatched: n, DeathOffsetRunnerUp: second}.warnIfCalageEtroit()

	if !strings.Contains(buf.String(), "calage du fil des morts trop peu distinct") {
		t.Fatalf("calage %d contre %d : la limite est franchie EN SILENCE — c'est le seul "+
			"point qui rendrait ce report bloquant. Journal : %q", n, second, buf.String())
	}
}
