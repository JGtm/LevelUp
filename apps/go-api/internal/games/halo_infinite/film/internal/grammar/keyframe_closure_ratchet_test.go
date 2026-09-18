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
