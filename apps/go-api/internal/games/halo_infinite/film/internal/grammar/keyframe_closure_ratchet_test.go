package grammar

// keyframe_closure_ratchet_test.go — LA COUVERTURE D'IMAGE-CLE NE DESCEND JAMAIS (lot 0.A.3).
//
// # CE QUE CE RATCHET GARDE
//
// `KeyframeClosure` mesure, par archetype, combien de records d'image-cle FERMENT — c'est-a-dire
// combien atterrissent exactement sur le premier bit du record suivant. C'est le seul oracle qui
// prouve une largeur de composant sans capture live. Le golden fige ces comptes sur les bobines
// par build ; le test rougit des qu'un compte DESCEND.
//
// # POURQUOI IL ROUGIT SUR LA BAISSE ET PAS SUR LA HAUSSE
//
// Le chantier du lot 3.6 est un PORT de composants : chaque composant porte debloque des records
// qui butaient dessus, donc les comptes MONTENT, lot apres lot. Rougir sur une hausse ferait
// echouer le progres qu'on cherche (la lecon de `replaydiff/polarite.go` : un compteur qu'on lit
// a l'envers refuse exactement ce qu'il devait garder). Une hausse est donc acceptee et
// SIGNALEE — le golden se regenere alors explicitement, ce qui laisse la trace du gain dans le
// diff.
//
// # LA PREUVE QUE LE RATCHET MORD
//
// Fausser une largeur du registre — par exemple `R(6)` en `R(7)` sur ti=6 — decale tous les
// composants qui suivent, donc la marche n'atterrit plus sur la frontiere et la fermeture tombe.
// Verifie le 2026-09-13 (cf. le rapport du lot).
//
// REGENERATION (jamais d edition a la main) :
//
//	go test ./internal/games/halo_infinite/film/internal/grammar/ -run KeyframeClosureRatchet -update-keyframe-closure
//
// LE CHEMIN A ETE CORRIGE LE 2026-09-17 (lot 3.6.a) : il nommait encore `film/filmdec/`, le
// paquet d avant la descente du lot 2.5.e. Le golden, lui, portait le bon chemin — il avait ete
// corrige A LA MAIN, ce que l en-tete de cette fonction interdit, et la premiere regeneration
// venue le remettait a l ancien. Le meme geste a RENDU AU GENERATEUR le bloc d historique du lot
// 1.9.1 bis, qui ne vivait que dans le golden et que toute regeneration effacait : un historique
// que la porte d ecriture ne connait pas n est pas un historique, c est un sursis.

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// updateFermeture : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE.
//
// ELLE EST NOMMEE, ET C EST LE CORRECTIF DE LA REVUE R1 (P1-1). Ce golden a d abord ete
// accroche au `-update` du corpus de graines du fuzz, sous le pretexte qu un second drapeau
// ferait paniquer `flag`. C ETAIT FAUX : `flag` ne panique que sur un NOM deja pris, et le
// paquet declare deja `-update-golden-familles` (`golden_minibobine_test.go`). Le cout de
// l erreur etait exactement celui que le lot est cense empecher : avec une largeur faussee en
// place, `go test ./...filmdec/ -update` (sans `-run`) repondait `ok` et REECRIVAIT ce golden
// avec la grammaire cassee — le ratchet ne le disait que par un `t.Logf` invisible sans `-v`.
// C est la reouverture du defaut C5 ferme le 2026-09-06 : « une porte de regeneration doit
// nommer CE qu elle regenere » (`fuzz_records_test.go`).
var updateFermeture = flag.Bool("update-keyframe-closure", false,
	"reecrire testdata/keyframe_closure.golden (lot 0.A.3) — CE golden seulement")

// closureGoldenPath : le golden, a cote des autres references du paquet.
const closureGoldenPath = "testdata/keyframe_closure.golden"

// closureMiniFilms : les bobines par build, celles du lot 0.A.2. Elles portent toutes leur
// `chunk_00`, sans lequel il n'y aurait ni registre ni archetypes — donc aucune fermeture a
// mesurer. La bobine historique `minifilm_000d5950` n'y est PAS : elle n'a pas de `chunk_00`.
func closureMiniFilms() []string {
	return []string{
		"a521164d", // HI_1_4_1
		"60ae07c4", // HI_1_8_0
		"11de8353", // HI_1_9_0
		"111fa685", // HI_1_10_0
		"e5adf7b2", // HI_1_11_0
		"bcb6d393", // HI_1_12_0
		"fb1a1a72", // HI_1_13_0
	}
}

// TestKeyframeClosureRatchet : la couverture par archetype ne descend jamais.
func TestKeyframeClosureRatchet(t *testing.T) {
	got := mesurerFermetureBobines(t)
	if *updateFermeture {
		if err := os.MkdirAll(filepath.Dir(closureGoldenPath), 0o750); err != nil {
			t.Fatalf("creation de testdata : %v", err)
		}
		if err := os.WriteFile(closureGoldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture du golden : %v", err)
		}
		// UNE PORTE DE REGENERATION NE REND JAMAIS `ok` (revue R2, C1). `go test` JETTE la sortie
		// d un paquet qui PASSE : un `t.Logf`, et meme une ecriture directe sur stderr, sont
		// INVISIBLES avec la commande documentee (sans `-v`). Mesure : le golden etait reecrit,
		// stdout rendait `ok`, stderr restait vide. Terminer en ECHEC est la seule forme qui rende
		// la reecriture visible ET qui empeche de confondre une regeneration avec un run vert.
		t.Fatalf("1 reference(s) reecrite(s) : %s (%d octets) ; "+
			"relancer sans -update-keyframe-closure pour verifier", closureGoldenPath, len(got))
	}
	brut, err := os.ReadFile(closureGoldenPath) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent (%s) : %v — regenerer avec -update-keyframe-closure",
			closureGoldenPath, err)
	}
	comparerFermeture(t, string(brut), got)
}

// comparerFermeture confronte le golden a la mesure, ligne a ligne.
//
// Une BAISSE est une erreur ; une HAUSSE est un gain a figer ; une ligne qui DISPARAIT est une
// erreur (un archetype cesse d'etre mesure), une ligne NEUVE est un gain.
func comparerFermeture(t *testing.T, want, got string) {
	t.Helper()
	fige, obtenu := lignesFermeture(want), lignesFermeture(got)
	for cle, ref := range fige {
		cur, ok := obtenu[cle]
		if !ok {
			t.Errorf("%s : l'archetype a DISPARU de la mesure (fige : %d/%d fermes)",
				cle, ref.closed, ref.total)
			continue
		}
		if cur.closed < ref.closed {
			t.Errorf("%s : fermeture en BAISSE, %d/%d fige contre %d/%d obtenu — bloquant %q.\n"+
				"Une largeur a bouge : la corriger, ou regenerer le golden si la baisse est voulue "+
				"et justifiee dans le commit.",
				cle, ref.closed, ref.total, cur.closed, cur.total, cur.blocking)
		}
		if cur.closed > ref.closed {
			t.Logf("%s : fermeture en HAUSSE, %d/%d -> %d/%d — figer par -update-keyframe-closure",
				cle, ref.closed, ref.total, cur.closed, cur.total)
		}
	}
	for cle, cur := range obtenu {
		if _, ok := fige[cle]; !ok {
			t.Logf("%s : archetype NEUF dans la mesure (%d/%d) — figer par -update-keyframe-closure",
				cle, cur.closed, cur.total)
		}
	}
}

// compteFermeture est une ligne de golden relue.
type compteFermeture struct {
	closed, total int
	blocking      string
}

// lignesFermeture relit le golden en table `film ti=N` -> comptes. Les lignes de commentaire et
// les lignes vides sont ignorees.
func lignesFermeture(s string) map[string]compteFermeture {
	out := map[string]compteFermeture{}
	for _, l := range strings.Split(s, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		cols := strings.Split(l, "\t")
		if len(cols) < 4 {
			continue
		}
		closed, err1 := strconv.Atoi(cols[2])
		total, err2 := strconv.Atoi(cols[3])
		if err1 != nil || err2 != nil {
			continue
		}
		bloquant := ""
		if len(cols) > 4 {
			bloquant = cols[4]
		}
		out[cols[0]+" "+cols[1]] = compteFermeture{closed: closed, total: total, blocking: bloquant}
	}
	return out
}

// mesurerFermetureBobines rend le rendu textuel de la mesure sur toutes les bobines par build.
func mesurerFermetureBobines(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("# FERMETURE DES RECORDS D'IMAGE-CLE PAR ARCHETYPE — golden du lot 0.A.3.\n")
	b.WriteString("# Une colonne par mesure : film, archetype, fermes, total, bloquant le plus frequent.\n")
	b.WriteString("# Le ratchet rougit sur une BAISSE de `fermes`. Regeneration :\n")
	b.WriteString("#   go test ./internal/games/halo_infinite/film/internal/grammar/ -run KeyframeClosureRatchet -update-keyframe-closure\n")
	b.WriteString("#\n")
	b.WriteString("# HISTORIQUE DES REGENERATIONS — une ligne par lot, avec CE QUI MONTE ET POURQUOI.\n")
	b.WriteString("# Un golden de couverture qu'on refige sans dire ce qu'il gagne ne garde plus rien.\n")
	b.WriteString("#\n")
	b.WriteString("#   2026-09-13 lot 0.A.3 : creation, 7 bobines par build, 217 lignes de mesure.\n")
	b.WriteString("#   2026-09-14 lot 1.3   : les CINQ etats par defaut manquants relus chez l'ecrivain\n")
	b.WriteString("#     (ti14 `V ; R(5)` FUN_140FED6F4 · ti17 `V ; R(7)` FUN_14101A0A4 · ti21 `R(18)`\n")
	b.WriteString("#     FUN_141133C24 · ti29 `V` seul FUN_14116F514 · ti47 `V ; R(5)` FUN_1410F44F8).\n")
	b.WriteString("#     21 lignes MONTENT, 0 descend, aucune ne disparait, aucun total ne bouge :\n")
	b.WriteString("#       ti=14    0/3520 -> 3520/3520 (100 %, les 7 bobines)\n")
	b.WriteString("#       ti=17    0/3729 -> 3729/3729 (100 %, les 7 bobines)\n")
	b.WriteString("#       ti=29    0/110  ->  102/110  (100 % sur 6 bobines, 22/30 sur 60ae07c4)\n")
	b.WriteString("#     ti=21 (0/357) et ti=47 (0/1716) ne bougent PAS, et c'etait prevu : leur largeur\n")
	b.WriteString("#     est prouvee par deux chaines, mais un composant reste faux — ti=47 bute sur\n")
	b.WriteString("#     `i2 personal-ai-data-component`, ti=21 sous-lit. Une largeur juste ne remplace\n")
	b.WriteString("#     pas un deserialiseur manquant (lot 3.6).\n")
	b.WriteString("#     Meme geste sur les 6 films de recherche : 8 796/62 686 (14,0 %) -> 19 337/62 686\n")
	b.WriteString("#     (30,8 %), delta = 5 024 + 5 379 + 138, et le plancher de hasard DESCEND de\n")
	b.WriteString("#     529 a 391. Mesure rejouable : `TestImageCleFermetureParArchetype` sous CHUNK00_FILMS.\n")
	b.WriteString("#\n")
	b.WriteString("#   2026-09-15  lot 1.9.1 bis, pas 2 bis + 2 quater : LES DEUX MOTS DE TAILLE SONT DES GARDES,\n")
	b.WriteString("#     ET LA TABLE DE PLAGES DE FUN_1406d3140 A NEUF CATEGORIES.\n")
	b.WriteString("#     NET : 26 lignes MONTENT (+89 records), 2 DESCENDENT (-21), soit +68 records.\n")
	b.WriteString("#     Les cinq archetypes objet passent de 184/21 698 (0,85 %) a 246/21 698 (1,13 %) ;\n")
	b.WriteString("#     ti=41 passe de 0/110 a 34/110 (30,9 %), ti=42 de 1/2 087 a 10, ti=37 de 3/3 331 a 15.\n")
	b.WriteString("#     Des archetypes HORS prefixe objet montent aussi : ti=35 (bipede) 0 -> 6, ti=10 0 -> 3,\n")
	b.WriteString("#     ti=12 0 -> 1.\n")
	b.WriteString("#     CE QUI A CHANGE, RELU CHEZ L ECRIVAIN :\n")
	b.WriteString("#       (a) FUN_142e2bfd0 porte DEUX fois `if (0 < (int)uVar7)` : le premier garde\n")
	b.WriteString("#           `vtable[0x60]` (l etat par defaut), le second `vtable[0x88]` PUIS la boucle de\n")
	b.WriteString("#           composants. `n1 == 0` -> aucun etat par defaut ; `n2 == 0` -> aucun composant.\n")
	b.WriteString("#           Comparaison SIGNEE. Le depot lisait les deux inconditionnellement.\n")
	b.WriteString("#       (b) FUN_140d10bb0 remplit neuf entrees de plages (constantes du binaire) et\n")
	b.WriteString("#           FUN_1406d3140 y indexe par `param_3` : W vaut 13 pour 0/1/7/8, 8 pour 2/3/5,\n")
	b.WriteString("#           9 pour 4/6, et la categorie 1 bascule sur l entree 4 quand sa sonde rend 1.\n")
	b.WriteString("#           Quatre sites de ti=37 corriges (i10, i21, i22, i28).\n")
	b.WriteString("#     LES DEUX DESCENTES, DOCUMENTEES RECORD PAR RECORD (condition posee par le pilote) :\n")
	b.WriteString("#       `a521164d` ti=13 8 -> 0. MESURE : `n1` vaut 136 sur les 454 records mesures — une\n")
	b.WriteString("#         taille de tampon CONSTANTE, donc l en-tete de 108 bits est juste — tandis que `n2`,\n")
	b.WriteString("#         lu juste apres l etat par defaut, prend -1 (48 fois), 2147483392 (24), 32768 (20),\n")
	b.WriteString("#         98304 (20), 0 (20), 68, 422710486, 2073479726, 1239369857, 4, 3, 2... C est du BRUIT :\n")
	b.WriteString("#         la marche est deja desalignee AVANT le mot de taille, donc dans l etat par defaut de\n")
	b.WriteString("#         ti=13. Les huit fermetures perdues etaient des fermetures obtenues APRES desalignement ;\n")
	b.WriteString("#         la garde ne les casse pas, elle les revele.\n")
	b.WriteString("#       `60ae07c4` ti=38 43 -> 30. Meme nature, et la meme mesure le montre : `n1` vaut 100 sur\n")
	b.WriteString("#         les 2 098 records, `n2` rend 0, 110, 14112, 14113, 14114, 3528, 3612672, 451709...\n")
	b.WriteString("#     L ORACLE QUE CETTE MESURE OUVRE, ET QUI VAUT POUR LA SUITE : sur les archetypes qui\n")
	b.WriteString("#     FERMENT, `n1` ET `n2` sont constants et plausibles (ti=14 4/28, ti=17 4/432, ti=22 12/12,\n")
	b.WriteString("#     ti=29 1/256, ti=6 4/7896) ; sur ceux qui echouent, `n1` est constant et `n2` est du bruit.\n")
	b.WriteString("#     `n2` est donc un oracle GRATUIT de justesse de l ETAT PAR DEFAUT, et il designe la cause :\n")
	b.WriteString("#     le premier bit faux de ti=13, 37 et 38 est DANS LEUR ETAT PAR DEFAUT, avant tout composant.\n")
	b.WriteString("#   2026-09-17 lot 3.6.a : le JOUEUR GERE (ti=9) n a plus aucun composant sans lecteur.\n")
	b.WriteString("#     `i4 managed-player-forge-weather-effect-overrides-component` (FUN_142ed5bc8,\n")
	b.WriteString("#     R(32) + R(32), 64 bits inconditionnels) etait le BLOQUANT NOMME des sept bobines ;\n")
	b.WriteString("#     `i9 managed-player-custom-input-prompt-widget` (FUN_141fcf160) passe de `partiel`\n")
	b.WriteString("#     a `porte` (boucle a etiquette du sac texte FUN_14080b034). Grammaires relevees chez\n")
	b.WriteString("#     l ecrivain, aucune entree de profil.\n")
	b.WriteString("#     7 lignes MONTENT, 0 descend, aucune ne disparait, aucun total ne bouge :\n")
	b.WriteString("#       ti=9     0/1717 -> 1716/1717 (99,94 %), et plus AUCUN bloquant nomme\n")
	b.WriteString("#         a521164d 0/262 -> 262/262 · 60ae07c4 0/240 -> 240/240 · 11de8353 0/393 -> 393/393\n")
	b.WriteString("#         111fa685 0/337 -> 336/337 · e5adf7b2 0/261 -> 261/261 · bcb6d393 0/144 -> 144/144\n")
	b.WriteString("#         fb1a1a72  0/80 ->   80/80\n")
	b.WriteString("#     LE 1 717e N EST PAS UN RECORD : sur `111fa685`, chunk 1, bit 9145, la marche va au\n")
	b.WriteString("#     bout de ses composants (DesyncAt = -1) et finit 3 415 bits AVANT la frontiere ; son\n")
	b.WriteString("#     `n1` vaut 2 154 823 696 quand les 1 716 autres valent 12. C est une ANCRE FORTUITE,\n")
	b.WriteString("#     la population que `default_state_n2_constant_test.go` ecarte deja par ce critere —\n")
	b.WriteString("#     aucune largeur de ti=9 ne peut la fermer. La cible du lot est donc 1 716, pas 1 717.\n")
	b.WriteString("#     `i9` ne fait monter AUCUN compte ici : il n existe au registre que sur `bcb6d393` et\n")
	b.WriteString("#     `fb1a1a72` (9 composants sur les cinq autres bobines), et aucun record n y atteignait\n")
	b.WriteString("#     la branche refusee. Sa grammaire est tenue par un test de largeur sur tampon\n")
	b.WriteString("#     synthetique (`TestTI9InputPromptLargeursDuSacTexte`, 11 chemins).\n")
	b.WriteString("#     Les 6 films de recherche NE SONT PAS mesures a ce lot : la voie libre du pilote\n")
	b.WriteString("#     (un seul decodage a la fois sur la machine) n avait pas ete donnee. Ligne `[!]` du\n")
	b.WriteString("#     plan, a jouer avec les deux gates de decodage.\n")
	b.WriteString("#   2026-09-18 lot 5.1.1 : LE POINT DE NAVIGATION (ti=12) EST PORTE DE `i1` AU MINUTEUR.\n")
	b.WriteString("#     Douze composants portes d un bloc (`i1` a `i12`), grammaires relevees chez le\n")
	b.WriteString("#     deserialiseur `+0x40` de chaque descripteur et recoupees par son serialiseur `+0x28`\n")
	b.WriteString("#     (NOTE_3_6_TI12_GRAMMAIRES_A_2026-09-17, § 5 a § 16). Cible : `i11`/`i12`, la duree\n")
	b.WriteString("#     INITIALE et la duree COURANTE du minuteur manuel — le compte a rebours de retour\n")
	b.WriteString("#     d un objectif, `R(17)` au pas de 50 ms, sans indirection.\n")
	b.WriteString("#     CE QUE LA MESURE DIT, ET C EST EXACTEMENT CE QUE LE LOT VISAIT : le bloquant de\n")
	b.WriteString("#     ti=12 avance de `i1 managed-navpoint-flags-component` a\n")
	b.WriteString("#     `i13 managed-navpoint-top-progress` sur les SEPT bobines. 0 ligne monte, 0 descend,\n")
	b.WriteString("#     aucun total ne bouge : la FERMETURE de ti=12 demande encore `i13` a `i27` (quinze\n")
	b.WriteString("#     composants, dont les huit `visual-state-groups`), donc elle n etait pas atteignable\n")
	b.WriteString("#     a ce lot et ne l est pas. Le bloquant qui avance de douze rangs EST la mesure.\n")
	b.WriteString("#     Les 6 films de recherche ne sont pas mesures ici non plus (meme raison qu au 3.6.a :\n")
	b.WriteString("#     un backfill tenait la machine). Les largeurs sont tenues par un test sur tampon\n")
	b.WriteString("#     synthetique (`components_navpoint_test.go`, 6 largeurs plates + 9 formes du bloc de\n")
	b.WriteString("#     filtres + le tag invalide + la dequantification).\n")
	b.WriteString("#   2026-09-18 lot 5.1.7-a : `param_4` EST LE `level` DU REGISTRE DU FILM, ET IL SE LIT.\n")
	b.WriteString("#     `param_4` — la propriete que le descripteur rend au deserialiseur, et dont huit\n")
	b.WriteString("#     lecteurs font une LARGEUR — venait d une table par nom plus, hors table, du BALAYAGE\n")
	b.WriteString("#     de `killsource.calibrateRSP` (0 a 5, la valeur qui maximisait la croissance des slots\n")
	b.WriteString("#     sur les records de BIPEDE). Il EST le `level` que le registre du film porte par\n")
	b.WriteString("#     composant (`entree + 0x100`, ce que `FUN_142e2c690` passe au deserialiseur), que le\n")
	b.WriteString("#     traverseur descendait deja jusqu a `consumeByName` sans que personne s en serve.\n")
	b.WriteString("#     TROIS lecteurs n avaient AUCUNE entree et prenaient donc la valeur balayee :\n")
	b.WriteString("#     `i10 object-parent-state` (vrai niveau 3), `i19 unit-actor-control` (2) et\n")
	b.WriteString("#     `i20 unit-actor-state` (4). `i10` est sur le chemin de TOUS les archetypes objet.\n")
	b.WriteString("#     NET SUR LES SEPT BOBINES : 14 lignes MONTENT (+627 records), 2 DESCENDENT (-2).\n")
	b.WriteString("#       ti=38 (arme au sol) : +575 — `fb1a1a72` 10/2147 -> 261/2147, `bcb6d393` 24 -> 129,\n")
	b.WriteString("#         `11de8353` 48 -> 99, `a521164d` 33 -> 122, `e5adf7b2` 12 -> 49, `111fa685` 30 -> 72\n")
	b.WriteString("#       ti=42 (objet pose)  : +51 sur les sept bobines\n")
	b.WriteString("#       ti=37 (equipement)  : `e5adf7b2` 0/489 -> 1/489\n")
	b.WriteString("#     LES DEUX BAISSES SONT D UN RECORD CHACUNE, ET ELLES SONT NOMMEES : `fb1a1a72` ti=37\n")
	b.WriteString("#     1/220 -> 0/220 et `a521164d` ti=37 4/762 -> 3/762. Meme archetype, meme lot, et\n")
	b.WriteString("#     `e5adf7b2` ti=37 MONTE de 1 dans le meme geste : a l echelle du record unique, une\n")
	b.WriteString("#     fermeture est une coincidence de frontiere, pas une largeur. Le bloquant de ti=37 ne\n")
	b.WriteString("#     bouge pas, et aucune ligne ne perd plus d un record contre +627 gagnes.\n")
	b.WriteString("#     Les 6 films de recherche ne sont pas mesures ici (un seul decodage a la fois).\n")
	b.WriteString("#   2026-09-18 lot 5.1.7-b : L ETAT PAR DEFAUT DE ti=40 EST LU, SA BOUCLE TOURNE.\n")
	b.WriteString("#     ti=40 n etait pas dans `defaultStateDeserByTI` : le jeu ecrivait 79 bits au\n")
	b.WriteString("#     minimum (FUN_1410A5A74), le lecteur en consommait ZERO, et le R(32) n2 se lisait\n")
	b.WriteString("#     79 bits trop tot -> consumeFullStateDefaultBlock faux -> LA BOUCLE DE COMPOSANTS\n")
	b.WriteString("#     N ETAIT JAMAIS LANCEE. C est ce que ce golden disait sans qu on le lise : 0 ferme\n")
	b.WriteString("#     sur 777 avec une colonne `bloquant` VIDE sur 48 composants dont 16 non portes.\n")
	b.WriteString("#     La feuille 4 (quaternion FUN_14076e494 + FUN_140c1e79c) portait la mention\n")
	b.WriteString("#     « largeur config-dependante » : PERIMEE. Ses deux globaux entrent par le catalogue\n")
	b.WriteString("#     de la carte depuis 3.4.1, ses deux fonctions sont portees depuis R7-b.\n")
	b.WriteString("#     MESURE ET TEMOIN NEGATIF (4f77afc1, 1 140 records ti=40 d image-cle) : la porte\n")
	b.WriteString("#     bVar14 vaut 1 sur 470 records (41,2 %). Feuille LUE : bVar14=0 661/661 a i30,\n")
	b.WriteString("#     bVar14=1 470/470 a i30 — meme rang, sans exception. Feuille modelisee ABSENTE :\n")
	b.WriteString("#     les 470 rendent DesyncAt == -1, la boucle ne tourne pas. Oracle binaire.\n")
	b.WriteString("#     CE QUI CHANGE ICI : le bloquant de ti=40 passe de VIDE a\n")
	b.WriteString("#     `i30 vehicle-auto-turret-triggers-component`. La fermeture NE MONTE PAS et ne le\n")
	b.WriteString("#     peut pas tant que les seize vehicle-* ne sont pas portes ; 0 ligne descend.\n")
	b.WriteString("#     Le document publie ne bouge d AUCUN octet (4f77afc1 : 256 recensees, 97 publiees,\n")
	b.WriteString("#     3 fins datees, avant comme apres) : le calque des vehicules passe par des\n")
	b.WriteString("#     balayages ANCRES, pas par la marche d etat complet.\n")
	b.WriteString("#   2026-09-23 lot M3.1 (campagne retours rejeu) : LA MARCHE D ANCRES NE COUPE PLUS LA TABLE.\n")
	b.WriteString("#     Une fenetre de 120 000 bits SANS candidat arretait le balayeur ; elle glisse\n")
	b.WriteString("#     desormais, et un en-tete EXACT de bipede recale l election. Les totaux MONTENT\n")
	b.WriteString("#     (records atteints : bcb6d393 ti=9 144 -> 151, ti=38 1841 -> 1878, ti=43 594 -> 624 ;\n")
	b.WriteString("#     fb1a1a72 ti=9 80 -> 94, ti=10 433 -> 461 ; a521164d ti=35 205 -> 209) et 0 ligne\n")
	b.WriteString("#     `fermes` ne descend. UNE ligne DISPARAIT (a521164d ti=0, 0/1 : une fausse ancre de\n")
	b.WriteString("#     slot bas que l election retenait) et UNE NAIT (bcb6d393 ti=1, 0/1 : la fausse\n")
	b.WriteString("#     ancre 192/ti 1 que l election retient dans l image-cle d avant-match, devant le\n")
	b.WriteString("#     record ti=9 slot 1297 — le repli nomme `repli_ancre_d_image_cle_par_election`).\n")
	b.WriteString("#     fb1a1a72 ti=1 0/1 -> 0/3 : DEUX records de plus, et c est la MEME fausse ancre\n")
	b.WriteString("#     (revue adverse du 2026-09-24, instrument `TestM3RevueMarche` sur la bobine) : le\n")
	b.WriteString("#     glissement franchit une fenetre vide du morceau 1 (deux images-cles), et l election\n")
	b.WriteString("#     y retient `0x400000C0 00000001` (slot 192, ti 1) a 186 bits DEVANT le vrai record\n")
	b.WriteString("#     ti=9 slot 1299 — slot plus bas, donc elu. Comptee dans `coverage.fallbacks`, elle\n")
	b.WriteString("#     n a pas de voisin a reparer sans une regle de plus : la marche deterministe\n")
	b.WriteString("#     (critere de retrait du repli) la fermera.\n")
	b.WriteString("#   2026-09-24 reprise du lot M3.1 : une fin de table a cheval sur deux fenetres arrete\n")
	b.WriteString("#     le glissement (`kfScanGlissant` reprend au debut de la trainee de sentinelles).\n")
	b.WriteString("#     AUCUNE ligne ne bouge sur les sept bobines.\n")
	b.WriteString("#   2026-09-24 lot D-fix (retours rejeu) : L ELECTION NE CONTREDIT PLUS UN RECORD PROUVE.\n")
	b.WriteString("#     La mesure porte sur la marche DU FILM (`FilmContext.MarcheDImageCle`), celle des\n")
	b.WriteString("#     balayages : l elu qu un record prouve par la grammaire contredit est refuse. Les deux\n")
	b.WriteString("#     lignes bcb6d393 `ti=1` et `ti=0` (0/1 chacune : les fausses ancres 192 et 1536)\n")
	b.WriteString("#     DISPARAISSENT, fb1a1a72 `ti=1` 0/3 -> 0/1 (la valeur d avant le lot M3.1 : les deux\n")
	b.WriteString("#     lectures de la fausse ancre 192 s en vont), et les records qu elles effacaient\n")
	b.WriteString("#     reviennent (bcb6d393 ti=9 151 -> 152, ti=38 1878 -> 1894 ; fb1a1a72 ti=9 94 -> 96,\n")
	b.WriteString("#     ti=38 2550 -> 2584 ; ti=37, 42, 43, 47, 10 aussi). 0 ligne `fermes` ne descend.\n")
	for _, court := range closureMiniFilms() {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		stats := fermetureDUneBobine(t, dir)
		tis := make([]int, 0, len(stats))
		for ti := range stats {
			tis = append(tis, int(ti))
		}
		sort.Ints(tis)
		for _, ti := range tis {
			s := stats[uint32(ti)] //nolint:gosec // ti vient d'une cle uint32
			fmt.Fprintf(&b, "%s\tti=%d\t%d\t%d\t%s\n", court, ti, s.Closed, s.Total, s.Blocking)
		}
	}
	return b.String()
}

// fermetureDUneBobine charge une bobine et rend sa fermeture par archetype.
func fermetureDUneBobine(t *testing.T, dir string) map[uint32]KeyframeClosureStat {
	t.Helper()
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bobine absente (%s) : %v — regenerer les bobines du lot 0.A.2", dir, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	stats, err := KeyframeClosure(NewFilmContext(film))
	if err != nil {
		t.Fatalf("KeyframeClosure %s : %v", dir, err)
	}
	return stats
}
