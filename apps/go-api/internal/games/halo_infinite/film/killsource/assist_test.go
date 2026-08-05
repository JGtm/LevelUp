package killsource

// assist_test.go — CE QUE LA PASSE D ASSISTANTS FIGE.
//
// Cinq filets, et chacun attrape une chose que les autres ne voient pas :
//
//  1. L ORDRE DES CHAMPS. C est la seule erreur du chantier qui se serait payee en SILENCE :
//     inverser victime et tueur laisse le decodage << marcher >>, les comptes tenir, et rend des
//     assistants faux. Le test re-mesure les deux ordres et exige que le bon domine largement.
//  2. LES COMPTES PAR FILM. Une mutation qui deplacerait un champ d un bit ferait s effondrer le
//     taux d attachement bien avant de changer une etiquette.
//  3. LE MULTISET DES ASSISTANCES PAR JOUEUR. C EST LE MEILLEUR TEST DU DOSSIER — voir
//     [TestAssistMultisetParJoueur] pour l argument, qui est structurel et non statistique.
//  4. LES DEUX DESACCORDS ENTRE ENREGISTREMENTS ATTACHES A LA MEME MORT, et ils sont DISTINCTS :
//     sur l IDENTITE de l assistant ([TestAssistExtraNEstPasMuet], cas symetrique) et sur son
//     EXISTENCE ([TestAssistDesaccordAsymetrique], un porteur et un muet). Un garde-fou qui ne
//     peut pas se declencher est pire que pas de garde-fou ; les deux montrent qu il PEUT.
//  5. LES TROIS ETATS DES PARTS DE DEGATS. << non mesure >> et << zero >> ne doivent jamais se
//     confondre, et la part de l assistant ne doit pas se publier sans assistant.
//
// Les deux derniers ne demandent AUCUNE fixture : ils portent sur des regles, pas sur un corpus.
// Les trois premiers exigent `KILLSOURCE_FIXTURES` (les films ne sont pas versionnes, 107 Mo).

import (
	"context"
	"os"
	"sort"
	"testing"
)

// assistAttendu : ce que chaque film de reference rend. Ces nombres ne sont PAS des cibles, ce
// sont des MESURES : les regenerer sans comprendre pourquoi ils ont bouge est la faute que ce
// fichier existe pour empecher.
var assistAttendu = map[string]struct {
	killEvents, attached, named, noAssist int
	// multi : morts auxquelles PLUSIEURS kill-events non consommes correspondaient. Rien ne
	// l ancrait avant `am.fix`, alors que c est la seule population ou [pickAssistHit] et
	// [distinctAssists] ont quelque chose a trancher. Mesure : 0 / 1 / 0 / 1.
	multi int
	// disagree : morts ou les kill-events attaches ne s accordent pas sur la PRESENCE du champ
	// assistant. ZERO partout, et c est MESURE : les deux morts a multi-attachement du corpus
	// portent deux fois le MEME enregistrement (RE_LOG 7ter.77).
	disagree int
	// apiTotal : total d assists de l API pour ce match (`match_participants.assists`, humains
	// ET bots). IL N EST PAS CALCULABLE HORS LIGNE — il est declare, comme les morts de l API le
	// sont dans le golden cumule.
	apiTotal int
	// apiMultiset : les assistances de l API PAR JOUEUR, triees en ordre decroissant et privees
	// des noms. NIL quand le film n a pas ete confronte a ce niveau de detail. Voir
	// [TestAssistMultisetParJoueur].
	apiMultiset []int
}{
	// LES DEUX FILMS SANS BOT SONT INCHANGES PAR RE_LOG 7ter.79, ET C EST LE PREMIER CONTROLE DU
	// LOT : une population qui se serait remplie ici aurait ete un artefact de lecteur.
	"000d5950": {killEvents: 100, attached: 93, named: 17, noAssist: 76, multi: 0, apiTotal: 17,
		apiMultiset: []int{6, 3, 2, 2, 1, 1, 1, 1}},
	"78919882": {killEvents: 106, attached: 99, named: 29, noAssist: 70, multi: 0, apiTotal: 29,
		apiMultiset: []int{7, 5, 4, 3, 3, 3, 2, 2}},
	// fccc61cd : +1 attachee et +1 sans assistant avec 7ter.79 (la mort infligee par `bid(7.0)`
	// n en nomme aucun) — `named` NE BOUGE PAS. L ecart a l API y reste de 2, il n avait aucune
	// chance d etre comble par ce lot, et c est ce que la mesure dit.
	"fccc61cd": {killEvents: 105, attached: 95, named: 16, noAssist: 79, multi: 1, apiTotal: 18},
	// 9b191a7f porte la RESERVE ARITHMETIQUE consignee dans `assist.go`. RE_LOG 7ter.79 en
	// recupere DEUX des trois manquantes : les kills du bot sont desormais publies, donc leurs
	// assistants aussi (84 -> 87 attachees, 22 -> 24 nommees). Il en reste 6 a l API, et le
	// plafond honnete est 24 + 3 (les trois morts DU bot, sans kill-event trouve) = 27 < 30.
	// Son multiset n est donc toujours PAS declare : le figer reviendrait a figer l anomalie.
	"9b191a7f": {killEvents: 101, attached: 87, named: 24, noAssist: 63, multi: 1, apiTotal: 30},
}

func fixturesRoot(t *testing.T) string {
	t.Helper()
	root := os.Getenv("KILLSOURCE_FIXTURES")
	if root == "" {
		t.Skip("KILLSOURCE_FIXTURES absent : les films de reference ne sont pas versionnes")
	}
	return root
}

// TestAssistOrdreDesChamps : le kill-event porte VICTIME puis TUEUR, et il faut le RE-MESURER,
// pas le croire. Les deux ordres sont confrontes au kill-feed sous la bijection deja resolue ;
// le bon doit gagner d un facteur au moins 4.
func TestAssistOrdreDesChamps(t *testing.T) {
	root := fixturesRoot(t)
	decodeMu.Lock()
	defer decodeMu.Unlock()
	for id := range assistAttendu {
		c, s := prepareForAssist(t, root, id)
		lu, inverse := 0, 0
		for _, r := range s.recs {
			k, v := c.roster.nameOf(r.fields.killer), c.roster.nameOf(r.fields.victim)
			for i := range c.feed.pairs {
				e := &c.feed.pairs[i]
				if dt := e.timeMS - r.ms; dt < -tolMS || dt > tolMS {
					continue
				}
				if e.killer == k && e.victim == v {
					lu++
					break
				}
			}
			for i := range c.feed.pairs {
				e := &c.feed.pairs[i]
				if dt := e.timeMS - r.ms; dt < -tolMS || dt > tolMS {
					continue
				}
				if e.killer == v && e.victim == k {
					inverse++
					break
				}
			}
		}
		if lu < 4*inverse+1 {
			t.Fatalf("%s: l ordre lu (victime, tueur) ne domine plus : %d contre %d a l envers",
				id, lu, inverse)
		}
	}
}

// TestAssistComptesParFilm : les denominateurs de la passe, film par film.
func TestAssistComptesParFilm(t *testing.T) {
	root := fixturesRoot(t)
	for id, want := range assistAttendu {
		res := decodeFixture(t, root, id)
		a := res.Stats.Assist
		if a.KillEvents != want.killEvents || a.Attached != want.attached ||
			a.Named != want.named || a.NoAssist != want.noAssist {
			t.Errorf("%s: kill-events %d/%d, attaches %d/%d, nommes %d/%d, sans assistant %d/%d",
				id, a.KillEvents, want.killEvents, a.Attached, want.attached,
				a.Named, want.named, a.NoAssist, want.noAssist)
		}
		// LA POPULATION A MULTI-ATTACHEMENT EST UNE QUANTITE VIVANTE : c est la SEULE ou
		// [pickAssistHit] et [distinctAssists] ont quelque chose a trancher. La laisser flotter,
		// c est laisser flotter le domaine d application des deux regles.
		if a.Multi != want.multi || a.AssistFieldDisagree != want.disagree {
			t.Errorf("%s: morts a multi-attachement %d/%d, desaccords sur la presence du champ "+
				"assistant %d/%d", id, a.Multi, want.multi, a.AssistFieldDisagree, want.disagree)
		}
		// L API est une BORNE SUPERIEURE : le film ne peut pas declarer plus d assistants
		// qu il n y en a eu. Un depassement serait une lecture qui invente.
		if a.Named > want.apiTotal {
			t.Errorf("%s: %d assistants decodes pour %d a l API — la lecture invente",
				id, a.Named, want.apiTotal)
		}
	}
}

// TestAssistMultisetParJoueur : LE MEILLEUR TEST DU DOSSIER, ET IL EST STRUCTUREL.
//
// POURQUOI IL N EST PAS AUTO-CONFIRMANT — c est tout l argument, et il tient en deux moities :
//
//	(1) LA BIJECTION NE FAIT QUE RENOMMER. Elle attribue des NOMS a des indices ; le MULTISET
//	    (les comptes, tries, prives des noms) ignore precisement ce que la bijection choisit.
//	(2) L ASSISTANT N ENTRE PAS DANS L OBJECTIF OPTIMISE. `quadScore` ne regarde que le COUPLE
//	    (tueur, victime). Le solveur ne peut donc pas << arranger >> les assistances : il ne les
//	    voit jamais.
//
// Les deux degres de liberte par lesquels un accord pourrait etre fabrique sont donc hors jeu, ce
// qui distingue radicalement ce test de l ancienne validation par joueur — laquelle comparait des
// comptes NOMMES, donc exactement ce que la bijection produit. Cette validation-la est morte ;
// celle-ci la remplace.
//
// PORTEE : deux films, ceux ou le decodage reproduit l API a l unite. Elle ne s etend pas au BTB
// (marge de bijection nulle) ni a `9b191a7f` (reserve arithmetique consignee dans `assist.go`).
func TestAssistMultisetParJoueur(t *testing.T) {
	root := fixturesRoot(t)
	for id, want := range assistAttendu {
		if want.apiMultiset == nil {
			continue
		}
		res := decodeFixture(t, root, id)
		got := multisetAssistances(res)
		if !memeMultiset(got, want.apiMultiset) {
			t.Errorf("%s: multiset des assistances hors ligne %v, API %v — l egalite qui tenait "+
				"lieu de preuve de justesse des assistants est rompue", id, got, want.apiMultiset)
		}
	}
}

// multisetAssistances : les assistances par joueur, triees en ordre DECROISSANT et PRIVEES DES
// NOMS. C est cette derniere operation qui rend la mesure independante de la bijection.
func multisetAssistances(res *Result) []int {
	par := map[string]int{}
	for _, k := range res.Kills {
		if k.Assist.Name != "" {
			par[k.Assist.Name]++
		}
	}
	out := make([]int, 0, len(par))
	for _, n := range par {
		out = append(out, n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func memeMultiset(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestAssistAucunSurplusSurLesKillEventsAttaches : le garde-fou, AVEC SON VRAI DENOMINATEUR.
//
// CE QUE CE TEST ETABLIT, ET RIEN DE PLUS : aucune mort n a recu deux KILL-EVENTS ATTACHES nommant
// des assistants differents. Il NE dit PAS << aucune mort ne porte deux assistants >> — la
// grammaire ne lit qu UN champ d assistant par kill-event, donc un second assistant qui ne se
// manifesterait pas par un second enregistrement attache est INVISIBLE ici. La reserve
// arithmetique sur `9b191a7f` (voir `assist.go`) donne une raison de ne pas confondre les deux.
//
// Il verifie AUSSI que le compteur agrege et les champs de ligne racontent la meme chose : un
// garde-fou qui vit a deux endroits doit y valoir la meme valeur, sinon l un des deux ment.
func TestAssistAucunSurplusSurLesKillEventsAttaches(t *testing.T) {
	root := fixturesRoot(t)
	for id := range assistAttendu {
		res := decodeFixture(t, root, id)
		somme := 0
		for _, k := range res.Kills {
			somme += k.Assist.Extra
		}
		if somme != res.Stats.Assist.AssistExtraTotal {
			t.Errorf("%s: surplus par ligne %d, agregat %d — les deux garde-fous divergent",
				id, somme, res.Stats.Assist.AssistExtraTotal)
		}
		if n := res.Stats.Assist.AssistMulti; n != 0 {
			t.Fatalf("%s: %d mort(s) ont recu DEUX kill-events attaches nommant des assistants "+
				"distincts — le champ simple ne suffit plus, il faut une table fille", id, n)
		}
	}
}

// TestAssistRejetsRestentComptes : les deux rejets connus valent ZERO sur ce corpus une fois
// l ordre des champs corrige. Ce n est PAS une raison de les retirer : ce test existe pour que
// leur retour soit visible, pas pour justifier leur suppression.
func TestAssistRejetsRestentComptes(t *testing.T) {
	root := fixturesRoot(t)
	for id := range assistAttendu {
		res := decodeFixture(t, root, id)
		a := res.Stats.Assist
		if a.RejectedSelf != 0 || a.RejectedVictim != 0 || a.RejectedRoster != 0 {
			t.Logf("%s: rejets non nuls — ==tueur %d, ==victime %d, hors-roster %d (a instruire, pas a ignorer)",
				id, a.RejectedSelf, a.RejectedVictim, a.RejectedRoster)
		}
	}
}

// rosterSynthetique : un roster minimal a bijection IDENTITE, pour les tests de REGLE. Il ne
// remplace aucune mesure : il sert a verifier ce que le code FAIT d une entree donnee, pas ce que
// les films contiennent.
func rosterSynthetique() *roster {
	noms := []string{"TUEUR", "VICTIME", "AIDE1", "AIDE2"}
	return &roster{names: noms, pin: map[int]int{}, nPlay: len(noms), nHumans: len(noms),
		perm: []int{0, 1, 2, 3}}
}

// mortSynthetique : une mort publiee, creditee au kill-feed.
func mortSynthetique() []Kill {
	return []Kill{{TimeMS: 10_000, Victim: "VICTIME",
		Feed: FeedTruth{Killer: "TUEUR", Present: true}}}
}

// TestAssistExtraNEstPasMuet : LE GARDE-FOU PEUT-IL SEULEMENT SE DECLENCHER ?
//
// C est la question que le corpus ne peut PAS trancher : `Extra` y vaut 0 partout, et un champ
// jamais ecrit vaut 0 lui aussi. Sans ce test, `SELECT SUM(assist_extra_count)` vaudrait zero PAR
// CONSTRUCTION — et un garde-fou muet est pire que pas de garde-fou, parce qu il rassure.
//
// On fabrique donc le cas que le corpus n a jamais produit : DEUX kill-events attaches a la MEME
// mort, nommant DEUX assistants differents.
func TestAssistExtraNEstPasMuet(t *testing.T) {
	c := &decodeCtx{roster: rosterSynthetique(), opts: DefaultOptions()}
	kills := mortSynthetique()
	s := &assistScan{recs: []killEventRec{
		{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: 2, killerPct: 60, assistPct: 39}},
		{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: 3, killerPct: 60, assistPct: 39}},
	}}
	st := c.attachAssists(kills, s)

	if kills[0].Assist.Extra != 1 {
		t.Errorf("surplus de la ligne = %d, attendu 1 — le garde-fou ne remonte pas jusqu a la mort",
			kills[0].Assist.Extra)
	}
	if st.AssistExtraTotal != 1 || st.AssistMulti != 1 || st.Multi != 1 {
		t.Errorf("agregats : surplus %d, morts a assistants distincts %d, morts multi-events %d "+
			"— attendu 1, 1, 1", st.AssistExtraTotal, st.AssistMulti, st.Multi)
	}
	// Cas SYMETRIQUE : les deux enregistrements portent un champ assistant, donc AUCUN desaccord
	// sur sa PRESENCE. C est ce qui distingue ce test du suivant.
	if st.AssistFieldDisagree != 0 {
		t.Errorf("desaccords sur la presence du champ = %d, attendu 0 — les deux enregistrements "+
			"portent un assistant, le desaccord porte sur son IDENTITE", st.AssistFieldDisagree)
	}
}

// TestAssistDesaccordAsymetrique : LE CAS QUE [TestAssistExtraNEstPasMuet] NE PROUVE PAS.
//
// Celui-la ne fabrique que le desaccord SYMETRIQUE — deux enregistrements nommant deux assistants
// DIFFERENTS. Le desaccord ASYMETRIQUE est autrement dangereux : un enregistrement SANS champ
// assistant, un autre en NOMMANT un. `attachAssists` prenait `hits[0]` sans regarder, et
// publiait alors << PAS D ASSISTANT, MESURE >> alors qu un enregistrement attache en nommait un —
// une MESURE D ABSENCE FABRIQUEE A PARTIR D UN DESACCORD, qu aucun compteur ne voyait.
//
// SA PRECONDITION EST REELLE, PAS THEORIQUE : `Stats.Assist.Multi` vaut 1 sur `9b191a7f` et 1 sur
// `fccc61cd` (fige dans `assistAttendu`). Ce que le corpus ne produit pas, c est le DESACCORD
// lui-meme — les deux enregistrements y sont identiques (RE_LOG 7ter.77). D ou ce test : la regle
// de preference est une DECISION, et une decision se teste sur le cas qu elle tranche.
//
// DEUX EXIGENCES, ET LA SECONDE COMPTE AUTANT QUE LA PREMIERE : la ligne ne doit PAS publier
// << pas d assistant mesure >>, ET le desaccord doit etre COMPTE. Resolu ne suffit pas : invisible,
// le choix redeviendrait une hypothese dormante.
func TestAssistDesaccordAsymetrique(t *testing.T) {
	c := &decodeCtx{roster: rosterSynthetique(), opts: DefaultOptions()}
	kills := mortSynthetique()
	// Le PREMIER enregistrement est muet, le SECOND nomme — c est l ordre qui piegeait `hits[0]`.
	s := &assistScan{recs: []killEventRec{
		{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: -1, killerPct: 100, assistPct: 149}},
		{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: 2, killerPct: 61, assistPct: 38}},
	}}
	st := c.attachAssists(kills, s)

	k := kills[0]
	if !k.Assist.Known || k.Assist.Name != "AIDE1" || k.Assist.Index != 2 {
		t.Errorf("ligne publiee : known=%v nom=%q indice=%d — attendu true, \"AIDE1\", 2 ; un "+
			"enregistrement attache NOMMAIT un assistant", k.Assist.Known, k.Assist.Name, k.Assist.Index)
	}
	if st.NoAssist != 0 {
		t.Errorf("<< pas d assistant >> compte %d fois, attendu 0 — c est une mesure d absence "+
			"fabriquee a partir d un desaccord", st.NoAssist)
	}
	if st.Named != 1 || st.Multi != 1 {
		t.Errorf("agregats : nommes %d, morts multi-events %d — attendu 1, 1", st.Named, st.Multi)
	}
	if st.AssistFieldDisagree != 1 {
		t.Errorf("desaccords sur la presence du champ = %d, attendu 1 — le desaccord doit etre "+
			"VISIBLE, pas seulement resolu", st.AssistFieldDisagree)
	}
	// La part de l ASSISTANT suit le champ retenu, pas la constante par film du muet.
	verifierPart(t, "assistant", k.AssistDamage, true, 38)
	verifierPart(t, "tueur", k.KillerDamage, true, 61)
	// Un seul assistant NOMME : pas de surplus. Les deux garde-fous ne mesurent pas la meme chose.
	if k.Assist.Extra != 0 || st.AssistMulti != 0 || st.AssistExtraTotal != 0 {
		t.Errorf("surplus ligne %d, morts a assistants distincts %d, surplus agrege %d — attendu "+
			"0, 0, 0 : un seul assistant est NOMME ici", k.Assist.Extra, st.AssistMulti, st.AssistExtraTotal)
	}
}

// TestPartsDeDegatsTroisEtats : les trois etats des deux parts, et LA REGLE QUI LES SEPARE.
//
// Elle ne se deduit d aucune grammaire, elle vient d une MESURE : sans assistant, le second bloc
// porte une CONSTANTE PAR FILM (149, 70, 20, 197 selon le film) et 74 % des lignes attachees sont
// dans ce cas. Publier la valeur brute remplirait les trois quarts de la colonne de bruit, avec
// des nombres parfaitement credibles.
//
// Le test verifie aussi qu AUCUN PLAFOND A 100 n est applique : plafonner jetterait 1.7 % des
// kill-events attaches a de vraies morts nommees, avec des valeurs mesurees jusqu a 228.
func TestPartsDeDegatsTroisEtats(t *testing.T) {
	cas := []struct {
		nom                   string
		rec                   *killEventRec
		tueurConnu, aideConnu bool
		tueurPct, aidePct     int
	}{
		{nom: "aucun kill-event attache : RIEN n est mesure", rec: nil},
		{nom: "assistant present : les deux parts sont mesurees",
			rec:        &killEventRec{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: 2, killerPct: 60, assistPct: 39}},
			tueurConnu: true, aideConnu: true, tueurPct: 60, aidePct: 39},
		{nom: "champ assistant ABSENT : la part de l assistant N EXISTE PAS (constante par film)",
			rec:        &killEventRec{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: -1, killerPct: 100, assistPct: 149}},
			tueurConnu: true, tueurPct: 100},
		{nom: "part du tueur > 100 : DONNEE REELLE, publiee telle quelle",
			rec:        &killEventRec{ms: 10_000, fields: killEventFields{victim: 1, killer: 0, assist: -1, killerPct: 228, assistPct: 70}},
			tueurConnu: true, tueurPct: 228},
	}
	for _, tc := range cas {
		t.Run(tc.nom, func(t *testing.T) {
			c := &decodeCtx{roster: rosterSynthetique(), opts: DefaultOptions()}
			kills := mortSynthetique()
			s := &assistScan{}
			if tc.rec != nil {
				s.recs = []killEventRec{*tc.rec}
			}
			st := c.attachAssists(kills, s)
			verifierPart(t, "tueur", kills[0].KillerDamage, tc.tueurConnu, tc.tueurPct)
			verifierPart(t, "assistant", kills[0].AssistDamage, tc.aideConnu, tc.aidePct)
			if tc.tueurPct > 100 && st.KillerPctOver100 != 1 {
				t.Errorf("part du tueur a %d : le compteur de depassement vaut %d, attendu 1 — "+
					"la population > 100 doit rester VISIBLE, pas devenir du folklore",
					tc.tueurPct, st.KillerPctOver100)
			}
		})
	}
}

func verifierPart(t *testing.T, quoi string, p DamageShare, connu bool, pct int) {
	t.Helper()
	if p.Known != connu {
		t.Fatalf("part %s : mesuree=%v, attendu %v (« non mesure » et « zero » ne se confondent pas)",
			quoi, p.Known, connu)
	}
	if connu && p.Pct != pct {
		t.Errorf("part %s : %d %%, attendu %d %%", quoi, p.Pct, pct)
	}
}

func decodeFixture(t *testing.T, root, id string) *Result {
	t.Helper()
	src, err := DirChunks(root + "/" + id)
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	res, err := Decode(context.Background(), id, src, nil)
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return res
}

// prepareForAssist : la preparation seule, pour les tests qui ont besoin des internes. Le verrou
// de paquet est tenu par l APPELANT (les globaux de replication sont partages).
func prepareForAssist(t *testing.T, root, id string) (*decodeCtx, *assistScan) {
	t.Helper()
	src, err := DirChunks(root + "/" + id)
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	resetGlobals()
	c := &decodeCtx{name: id, opts: DefaultOptions()}
	if err := c.prepare(context.Background(), src); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	return c, scanKillEvents(c.film)
}
