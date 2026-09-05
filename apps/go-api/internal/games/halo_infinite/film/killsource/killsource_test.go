package killsource

// killsource_test.go — DEUX NIVEAUX DE TEST, ET ILS NE SE REMPLACENT PAS.
//
//  1. Les tests PURS tournent partout, sans fixture : ils verrouillent les regles que le chantier
//     a payees cher (le recollage de couples, la deduplication des vues, la multiplicite, la
//     regle d etiquetage, l epinglage des bots).
//  2. LA NON-REGRESSION SUR FILM REEL ne tourne que si `KILLSOURCE_FIXTURES` designe un
//     repertoire de films. Les chiffres de reference vivaient dans un journal ; ils vivent
//     desormais dans un test. Sans fixture, `t.Skip` — pas de faux vert, pas de gate casse.
//
//	KILLSOURCE_FIXTURES=<dir> go test ./internal/games/halo_infinite/film/killsource/ -run Reference -v

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/filmsource"
)

// ───────────────────────────────────────────────────────────────────────────────
// (1) TESTS PURS
// ───────────────────────────────────────────────────────────────────────────────

func TestReconstructPairsRecollageEtOrphelins(t *testing.T) {
	feed := []feedEvent{
		{timeMS: 1000, killer: "A", victim: "B"}, // couple au meme instant
		{timeMS: 2000, killer: "C"},              // kill orphelin...
		{timeMS: 2100, victim: "D"},              // ... recolle sur ce death
		{timeMS: 9000, killer: "E"},              // kill sans voisin : victime non humaine
	}
	got := reconstructPairs(feed)
	if len(got) != 2 {
		t.Fatalf("couples reconstruits = %d, attendu 2 (%v)", len(got), got)
	}
	if got[1].killer != "C" || got[1].victim != "D" || got[1].timeMS != 2000 {
		t.Errorf("recollage = %+v, attendu {2000 C D}", got[1])
	}
	kf := &killFeed{events: feed}
	kf.split()
	if len(kf.real) != 1 || len(kf.fab) != 1 || len(kf.orphK) != 1 {
		t.Errorf("split = reels %d / recolles %d / kills perdus %d, attendu 1/1/1",
			len(kf.real), len(kf.fab), len(kf.orphK))
	}
}

// TestSplitIsoleLesMortsSansTueur : LA POPULATION DES MORTS INFLIGEES PAR UN BOT, AU NIVEAU DE LA
// STRUCTURE (RE_LOG 7ter.79).
//
// Le kill-feed est humain-seul dans LES DEUX SENS, et c est la symetrie qui avait ete manquee :
// un KILL que rien ne consomme designe une mort DE bot (deja exploite), une MORT que rien ne
// consomme designe une mort PAR un bot (population neuve). Ce test fige la seconde moitie.
//
// IL VERIFIE AUSSI CE QUI NE DOIT PAS BOUGER : la marque de consommation est posee APRES coup et
// [reconstructPairs] rend exactement les memes couples qu avant.
func TestSplitIsoleLesMortsSansTueur(t *testing.T) {
	feed := []feedEvent{
		{timeMS: 1000, killer: "A", victim: "B"}, // couple au meme instant
		{timeMS: 2000, killer: "C"},              // kill orphelin...
		{timeMS: 2100, victim: "D"},              // ... recolle sur ce death
		{timeMS: 5000, victim: "E"},              // MORT que rien ne consomme : le tueur n est pas humain
		{timeMS: 9000, killer: "F"},              // kill sans voisin : la victime n est pas humaine
	}
	kf := &killFeed{events: feed}
	kf.split()
	if len(kf.orphD) != 1 || kf.orphD[0].victim != "E" {
		t.Fatalf("morts sans tueur = %v, attendu la seule mort de E", kf.orphD)
	}
	if len(kf.real) != 1 || len(kf.fab) != 1 || len(kf.orphK) != 1 {
		t.Errorf("split = reels %d / recolles %d / kills perdus %d, attendu 1/1/1",
			len(kf.real), len(kf.fab), len(kf.orphK))
	}
	// LA MARQUE DE CONSOMMATION NE DOIT RIEN CHANGER AUX COUPLES : c est l invariant qui protege
	// les trois denominateurs deja publies.
	if got := reconstructPairs(feed); len(got) != 2 {
		t.Errorf("couples reconstruits = %d, attendu 2 — le suivi de consommation a change le "+
			"resultat de reconstructPairs, donc les denominateurs publies", len(got))
	}
}

// TestLesDeuxDeclarationsDeMortsAPIConcordent : DEUX CONSTANTES DECLAREES DECRIVENT LA MEME CHOSE,
// et rien ne les reliait. `apiDeaths` (le cumul, cite dans le golden) et `reference.apiDeathsFilm`
// (le detail, qui fonde le controle de comptabilite) viennent toutes deux de
// `match_participants.deaths` — donc leur somme DOIT valoir le cumul. Sans ce test, corriger l une
// et oublier l autre laisserait un golden juste a cote d un controle faux, sans un mot.
func TestLesDeuxDeclarationsDeMortsAPIConcordent(t *testing.T) {
	somme := 0
	for _, ref := range references {
		somme += ref.apiDeathsFilm
	}
	if somme != apiDeaths {
		t.Errorf("somme des morts de l API par film = %d, constante cumulee `apiDeaths` = %d — "+
			"les deux declarations ont diverge", somme, apiDeaths)
	}
}

func TestDedupUneVueParMort(t *testing.T) {
	// Le MEME dead-state repete dans deux vues de replication du meme paquet est UNE mort.
	cs := []candidate{
		{chunk: 1, pidx: 7, bit: 100, tag: 0xacd1cff4, victim: 2, killer: 5, cat: 0},
		{chunk: 1, pidx: 7, bit: 900, tag: 0xacd1cff4, victim: 2, killer: 5, cat: 0},
		{chunk: 1, pidx: 8, bit: 100, tag: 0xacd1cff4, victim: 2, killer: 5, cat: 0},
	}
	if got := dedup(cs); len(got) != 2 {
		t.Fatalf("dedup = %d, attendu 2 (le doublon de vue tombe, l autre paquet reste)", len(got))
	}
	if got := dedup(cs); got[0].bit != 100 {
		t.Errorf("dedup garde le bit %d, attendu le PREMIER (100)", got[0].bit)
	}
}

func TestMultiplicityComptePaquetsPasVues(t *testing.T) {
	// Un faux positif se repete AU MEME BIT sur des PAQUETS differents : c est cela qui compte.
	cs := []candidate{
		{chunk: 1, pidx: 1, bit: 746, tag: 0x2808, victim: 3},
		{chunk: 1, pidx: 1, bit: 746, tag: 0x2808, victim: 3}, // meme paquet : ne compte pas deux fois
		{chunk: 1, pidx: 2, bit: 746, tag: 0x2808, victim: 3},
		{chunk: 1, pidx: 3, bit: 5000, tag: 0xacd1cff4, victim: 3},
	}
	m := multiplicity(cs)
	if got := multOf(m, cs[0]); got != 2 {
		t.Errorf("multiplicite du champ fixe = %d, attendu 2 paquets distincts", got)
	}
	if got := multOf(m, cs[3]); got != 1 {
		t.Errorf("multiplicite d un vrai dead-state = %d, attendu 1", got)
	}
}

func TestEtiquetageRegleAutres(t *testing.T) {
	cases := []struct {
		name    string
		tag     uint32
		display string
		named   bool
	}{
		// Nom propre publiable : il sort tel quel.
		{"arme nommee", 0xacd1cff4, "M41 SPNKr", true},
		// Classe sure, aucun nom propre : la regle dictee est << Autres >>.
		{"melee sans nom propre", 0xdaa03c35, LabelOther, false},
		// AMBIGU : l etiquette serait affirmative et fausse. NE PAS PUBLIER le nom.
		{"effet generique ambigu", 0x88f1034c, LabelOther, false},
		// Tag absent du catalogue : inconnu de bout en bout.
		{"hors catalogue", 0xffffffff, LabelOther, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := sourceTruthOf(c.tag, 0)
			if s.Display != c.display || s.Named != c.named {
				t.Errorf("Display=%q Named=%v, attendu %q/%v", s.Display, s.Named, c.display, c.named)
			}
			if s.Tag != c.tag {
				t.Errorf("le tag BRUT doit toujours ressortir : %08x", s.Tag)
			}
		})
	}
}

func TestEtiquetageGardeLaClasseMemeSansNom(t *testing.T) {
	// LA REGLE << Autres >> NE DOIT PAS FAIRE PERDRE LA CLASSE : c est ce qui permet a un
	// consommateur de raffiner ou de traduire sans que ce paquet porte un libelle en dur.
	s := sourceTruthOf(0xdaa03c35, 3)
	if s.Class == "" || s.Status == "" {
		t.Fatalf("classe/statut vides pour un tag connu : %+v", s)
	}
	if s.Detail == "" {
		t.Error("Detail vide : la qualification mesuree doit accompagner l etiquette")
	}
	if s.Category != CategorySilentMelee || s.Category.Name() != "SilentMelee" {
		t.Errorf("categorie = %d/%s, attendu SilentMelee", s.Category, s.Category.Name())
	}
}

func TestRosterEpingleLesBotsAuDelaDesHumains(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B", "C", "D", "E", "F", "G", "H"}}
	bm := botMeta{NBots: 2, Bots: []bot{
		{Slot: 8, BotID: 39, Name: "343 Aloysius"}, // au-dela des humains : EPINGLE
		{Slot: 3, BotID: 7, Name: "343 Contredit"}, // dans l espace des humains : NON epingle
	}}
	r := buildRoster(kf, bm, true)
	if r.nPlay != 9 {
		t.Errorf("nPlay = %d, attendu 9 (la borne se DERIVE du slot declare)", r.nPlay)
	}
	if !r.isBotIndex(8) || r.isBotIndex(3) {
		t.Errorf("epinglage errone : 8 doit etre epingle, 3 non (pin=%v)", r.pin)
	}
	if len(r.unpinned) != 1 {
		t.Errorf("le bot contredisant le modele doit etre SIGNALE, pas dissimule : %v", r.unpinned)
	}
	free, freeNames := r.freeSlots()
	if len(free) != len(freeNames) {
		t.Errorf("probleme d affectation non carre : %d indices pour %d noms", len(free), len(freeNames))
	}
}

func TestRosterSansBotsResteAlEtatDAvant(t *testing.T) {
	kf := &killFeed{names: []string{"A", "B"}}
	bm := botMeta{NBots: 1, Bots: []bot{{Slot: 8, BotID: 39, Name: "343 Aloysius"}}}
	if r := buildRoster(kf, bm, false); r.nPlay != 2 || len(r.pin) != 0 {
		t.Errorf("bascule Bots=false : nPlay=%d pin=%v, attendu 2 et vide", r.nPlay, r.pin)
	}
}

func TestOptionsDefautEstLaConfigurationGelee(t *testing.T) {
	o := DefaultOptions()
	if !o.Bots || !o.SelfSource || o.MultiplicityMax != 2 || !o.StrongTagRequired {
		t.Fatalf("la configuration gelee a bouge : %+v", o)
	}
	// Un appelant qui ne renseigne qu un champ ne doit pas se retrouver avec zero redemarrage.
	partial := Options{Bots: true}
	partial.normalize()
	if partial.BijectionRestarts != o.BijectionRestarts || partial.Views != o.Views {
		t.Errorf("normalize incomplet : %+v", partial)
	}
}

func TestCatalogueEmbarque(t *testing.T) {
	if CatalogueSize() < 400 {
		t.Fatalf("catalogue embarque = %d ids, attendu ~468", CatalogueSize())
	}
	if p := CatalogueProvenance(); p.IDsDate == "" || p.LabelsDate == "" {
		t.Errorf("provenance sans date : %+v — une table sans date ne se verifie pas", p)
	}
	if !isCatalogued(0xacd1cff4) || isCatalogued(0xffffffff) {
		t.Error("le test d acceptation du catalogue ne discrimine pas")
	}
}

func TestPublicationLigneParLigneRefuseeSansMargeDeBijection(t *testing.T) {
	// MARGE NULLE = deux joueurs interchangeables : seul l AGREGAT est publiable.
	r := &Result{BijectionMargin: 0}
	if r.LineByLinePublishable() {
		t.Error("marge nulle : les lignes ne doivent PAS etre declarees publiables")
	}
	r.BijectionMargin = 12
	if !r.LineByLinePublishable() {
		t.Error("marge large et sante nominale : les lignes doivent etre publiables")
	}
	r.Health.TagOutOfCatalogueWalk = 1
	if r.LineByLinePublishable() {
		t.Error("une ALERTE de sante doit suspendre la publication ligne par ligne")
	}
}

// ───────────────────────────────────────────────────────────────────────────────
// (2) NON-REGRESSION SUR FILM REEL — les chiffres de reference, figes
// ───────────────────────────────────────────────────────────────────────────────

// reference : ce qu un film doit rendre. Source : RE_LOG 7ter.70 (couverture, origines),
// verifie 7ter.71 ; hybride et gate (b) par voie 7ter.72, verifie 7ter.73.
type reference struct {
	film       string
	realPairs  int
	selfSource int
	botDeaths  int
	ghost      int
	// botKillerDeaths : morts INFLIGEES PAR un bot (RE_LOG 7ter.79). CE CHIFFRE EST CONTRAINT DE
	// L EXTERIEUR, et c est ce qui en fait autre chose qu une empreinte : c est EXACTEMENT le
	// nombre de kills que l API attribue au `bid(...)` du film — 3 pour `bid(39.0)`, 1 pour
	// `bid(7.0)` — et il vaut ZERO sur les deux films SANS bot. Une population qui se remplit sur
	// les films a bot et reste vide sur les autres n est pas un artefact de lecteur.
	botKillerDeaths int
	// apiDeathsFilm : morts declarees par l API pour ce match, bots compris. DECLAREE, pas
	// decodee (`match_participants.deaths` somme sur les participants). Elle sert au seul
	// controle qui decide de ce lot : `couverts + morts de bot + morts par un bot == API`.
	apiDeathsFilm int
}

var references = []reference{
	{"000d5950", 93, 1, 0, 0, 0, 93},
	{"9b191a7f", 84, 2, 3, 1, 3, 90},
	{"78919882", 99, 2, 0, 0, 0, 99},
	{"fccc61cd", 95, 3, 2, 0, 1, 98},
}

// anchors : LES ANCRES DE VERITE TERRAIN, confirmees en mode Theater. C est la ressource la plus
// rare du chantier : 30 confrontations, zero etiquette publiee fausse. Elles vivaient dans un
// journal ; si une bascule change un seul tag, cela doit se voir ICI et tout de suite.
//
// PIEGE, ET IL A MORDU : deux morts peuvent tomber dans la MEME seconde. On verifie donc qu AU
// MOINS UNE ligne de la seconde porte le tag attendu, jamais qu une map indexee par "MM:SS" le
// porte.
type anchor2 struct {
	at       string
	tag      uint32
	diverges bool
}

var anchors = map[string][]anchor2{
	"000d5950": {
		{"00:35", 0xeea85c26, false}, {"00:44", 0x130c4b61, false},
		{"02:08", 0x4a555df8, false}, {"02:09", 0x4a555df8, false},
		{"03:08", 0xeea85c26, false}, {"04:26", 0x130c4b61, false},
		{"06:19", 0x4a555df8, false}, {"07:57", 0xdaa03c35, false},
		{"07:59", 0x4a555df8, false}, {"08:03", 0x4a555df8, false},
		{"01:12", 0xacd1cff4, true},
	},
	"9b191a7f": {
		{"01:00", 0x5eaa815c, false}, {"03:41", 0x0000b29c, false},
		{"04:50", 0x5eaa815c, false}, {"05:12", 0x5eaa815c, false},
		{"08:31", 0x5eaa815c, true}, {"09:13", 0xfca02b3c, true},
	},
	"78919882": {
		{"09:35", 0xd468178e, false}, {"09:44", 0xd468178e, false},
		{"09:54", 0xd468178e, false}, {"02:35", 0x119861b4, false},
		{"06:19", 0x0d203522, false},
		{"01:44", 0xd468178e, true}, {"09:18", 0x0d203522, true},
	},
	"fccc61cd": {
		{"03:16", 0x0d203522, false}, {"02:09", 0x00426796, false},
		{"04:53", 0xffebe84e, false},
		{"01:25", 0xacd1cff4, true}, {"03:37", 0xba5ac78e, true},
		{"08:38", 0x5e389b5d, true},
	},
}

func clock(ms int) string { return fmt.Sprintf("%02d:%02d", ms/60000, (ms/1000)%60) }

func TestReferenceFilms(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_FIXTURES")
	if dir == "" {
		t.Skip("KILLSOURCE_FIXTURES non defini : non-regression sur film reel ignoree")
	}
	for _, ref := range references {
		t.Run(ref.film, func(t *testing.T) {
			checkFilm(t, filepath.Join(dir, ref.film), ref)
		})
	}
}

func checkFilm(t *testing.T, dir string, ref reference) {
	t.Helper()
	src, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		t.Skipf("film absent : %v", err)
	}
	res, err := Decode(context.Background(), ref.film, src, nil)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	t.Logf("calibration : %s", res.Calibration)
	t.Logf("couverture %d/%d couples REELS | %d couples reconstruits | %d morts du feed | %d morts de BOT",
		res.Coverage.Covered, res.Coverage.RealPairs, res.Coverage.ReconstructedPairs,
		res.Coverage.FeedDeaths, res.Coverage.BotDeaths)
	t.Logf("gate (b) MARCHE %d/%d = %.1f%% | SCAN %d/%d = %.1f%% | verdict de sante %s | marge %d",
		res.Stats.Walk.Matched, res.Stats.Walk.Population, 100*res.Stats.Walk.Ratio(),
		res.Stats.Scan.Matched, res.Stats.Scan.Population, 100*res.Stats.Scan.Ratio(),
		res.Health.Verdict(), res.BijectionMargin)
	t.Logf("concordance : redondants %d | ACCORD %d | DESACCORD %d | morts a plusieurs candidats %d | paquets a events localises %d/%d",
		res.Stats.Redundant, res.Stats.Agree, res.Stats.Disagree, res.Stats.MultiCandidate,
		res.Stats.PacketsLocated, res.Stats.PacketsWithEvents)

	// LA PROPRIETE QUI REND L HYBRIDE LEGITIME : il PREFERE, il n arbitre pas.
	if res.Stats.Disagree != 0 {
		t.Errorf("DESACCORD entre les deux voies = %d, attendu 0 : l hybride serait devenu un ARBITRAGE",
			res.Stats.Disagree)
	}
	if res.Stats.MultiCandidate != 0 {
		t.Errorf("morts a plusieurs candidats en concurrence = %d, attendu 0", res.Stats.MultiCandidate)
	}

	if res.Coverage.RealPairs != ref.realPairs {
		t.Errorf("couples REELS = %d, attendu %d", res.Coverage.RealPairs, ref.realPairs)
	}
	if res.Coverage.Covered != ref.realPairs {
		t.Errorf("couverture = %d/%d, attendu 100 %%", res.Coverage.Covered, ref.realPairs)
	}
	if res.Coverage.GhostPairs != ref.ghost {
		t.Errorf("couples FANTOMES = %d, attendu %d", res.Coverage.GhostPairs, ref.ghost)
	}
	if res.Coverage.BotDeaths != ref.botDeaths {
		t.Errorf("morts de BOT = %d, attendu %d", res.Coverage.BotDeaths, ref.botDeaths)
	}
	checkBotKillerDeaths(t, res, ref)
	checkOrigins(t, res, ref)
	checkAnchors(t, res, ref.film)
}

// checkBotKillerDeaths : LE CONTROLE QUI DECIDE DU LOT (RE_LOG 7ter.79).
//
// Il ne verifie PAS que le decodeur retrouve son propre chiffre — ce serait une empreinte. Il
// verifie DEUX egalites dont aucune des deux moities ne vient du decodeur :
//
//	(1) le cardinal de la population vaut le nombre de kills que l API attribue au bot du film ;
//	(2) `couverts + morts de bot + morts par un bot` vaut EXACTEMENT les morts de l API.
//
// La seconde ferme la comptabilite du film entier avec trois populations disjointes, chacune
// nommee. Elle vaut aussi sur les deux films SANS bot, ou les deux termes de bot sont nuls : c est
// la moitie NEGATIVE du controle, et elle est indispensable — une population qui se remplirait
// partout ne mesurerait que la permissivite du lecteur.
func checkBotKillerDeaths(t *testing.T, res *Result, ref reference) {
	t.Helper()
	if res.Coverage.BotKillerDeaths != ref.botKillerDeaths {
		t.Errorf("morts INFLIGEES PAR un bot = %d, attendu %d (= les kills de l API du bid(...))",
			res.Coverage.BotKillerDeaths, ref.botKillerDeaths)
	}
	// Toutes les morts orphelines proposees doivent avoir ete decodees ET publiees : une seule
	// perdue signifierait qu une mort de l API reste sans ligne.
	b := res.Stats.BotKiller
	if b.Population != b.Matched || b.Matched != b.Published {
		t.Errorf("morts infligees PAR un bot : %d proposees, %d appariees, %d publiees — les trois "+
			"doivent etre egales, sinon une mort de l API reste sans ligne", b.Population, b.Matched, b.Published)
	}
	total := res.Coverage.Covered + res.Coverage.BotDeaths + res.Coverage.BotKillerDeaths
	if total != ref.apiDeathsFilm {
		t.Errorf("comptabilite du film : %d couverts + %d morts de bot + %d morts par un bot = %d, "+
			"attendu %d morts a l API", res.Coverage.Covered, res.Coverage.BotDeaths,
			res.Coverage.BotKillerDeaths, total, ref.apiDeathsFilm)
	}
}

func checkOrigins(t *testing.T, res *Result, ref reference) {
	t.Helper()
	n := map[Origin]int{}
	for _, k := range res.Kills {
		n[k.Read.Origin]++
		if (k.Read.Origin == OriginSelfSource) != k.Diverges {
			t.Errorf("%s : divergence et origine incoherentes (%v / %s)",
				clock(k.TimeMS), k.Diverges, k.Read.Origin)
		}
		if k.Feed.Present == (k.Read.Origin == OriginBot) {
			t.Errorf("%s : une mort de BOT ne doit PAS etre marquee presente au feed", clock(k.TimeMS))
		}
		// SYMETRIE STRICTE ET NON NEGOCIABLE : une mort infligee PAR un bot EST au feed (la
		// victime est humaine, elle a un XUID) ; c est le KILL qui n y est pas. La marquer
		// absente reviendrait a jeter la moitie que le feed porte vraiment.
		if k.Read.Origin == OriginBotKiller && !k.Feed.Present {
			t.Errorf("%s : une mort INFLIGEE PAR un bot doit etre marquee PRESENTE au feed — "+
				"le feed porte la mort, c est le kill qui manque", clock(k.TimeMS))
		}
		if k.Source.Display == "" {
			t.Errorf("%s : etiquette vide — le repli %q doit toujours s appliquer", clock(k.TimeMS), LabelOther)
		}
	}
	if n[OriginSelfSource] != ref.selfSource {
		t.Errorf("morts a SOURCE APPARTENANT A LA VICTIME = %d, attendu %d",
			n[OriginSelfSource], ref.selfSource)
	}
	if n[OriginBotKiller] != ref.botKillerDeaths {
		t.Errorf("lignes d origine %q = %d, attendu %d", OriginBotKiller,
			n[OriginBotKiller], ref.botKillerDeaths)
	}
	if res.Health.Verdict() == filmdec.VerdictAlerte {
		t.Errorf("verdict de sante ALERTE sur un film de reference : %v", res.Health.Alerts())
	}
	if res.Health.TagOutOfCatalogueWalk != 0 {
		t.Errorf("tags hors catalogue vus par la marche = %d, attendu 0 (table a jour)",
			res.Health.TagOutOfCatalogueWalk)
	}
}

func checkAnchors(t *testing.T, res *Result, film string) {
	t.Helper()
	bySecond := map[string][]Kill{}
	for _, k := range res.Kills {
		bySecond[clock(k.TimeMS)] = append(bySecond[clock(k.TimeMS)], k)
	}
	for _, a := range anchors[film] {
		lines := bySecond[a.at]
		if len(lines) == 0 {
			t.Errorf("ancre %s %s : AUCUNE ligne publiee a cette seconde", film, a.at)
			continue
		}
		ok := false
		for _, k := range lines {
			if k.Source.Tag == a.tag && k.Diverges == a.diverges {
				ok = true
			}
		}
		if !ok {
			got := make([]string, 0, len(lines))
			for _, k := range lines {
				got = append(got, fmt.Sprintf("%08x(div=%v,%s)", k.Source.Tag, k.Diverges, k.Read))
			}
			t.Errorf("ancre %s %s : attendu %08x div=%v, obtenu %v", film, a.at, a.tag, a.diverges, got)
		}
	}
}
