package killsource

// golden_guard_test.go — LE GARDE-FOU DES GOLDEN EUX-MEMES.
//
// POURQUOI CE FICHIER EXISTE. Un golden peut se degrader sans jamais casser : il suffit qu un
// jour quelqu un simplifie le rendu et n y laisse que des nombres. Le fichier passerait encore, et
// il ne prouverait plus rien — un << 371 >> nu ne dit pas de quoi il est le numerateur.
//
// CE TEST TOURNE MEME SANS FIXTURE, et c est tout l interet : il lit les fichiers figes et
// verifie qu ils portent encore les PHRASES qui qualifient leurs chiffres, les 30 ancres avec
// leurs tags, la ligne discriminante et le controle negatif. C est le seul test de ce paquet qui
// protege la FORME de la preuve plutot que sa valeur.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// phrasesParFilm : ce que le golden d un film doit dire, et pas seulement chiffrer.
var phrasesParFilm = []string{
	"couples REELS (couples reconstruits moins les couples FABRIQUES)",
	"couples reconstruits par la reconstruction de kill-feed",
	"morts du KILL-FEED (il ne contient AUCUNE mort de bot)",
	"POPULATION NEUVE, jamais au denominateur ci-dessus",
	"SECONDE POPULATION NEUVE, jamais au denominateur",
	"le feed porte la MORT mais aucun kill en face",
	"ANCRES DE VERITE TERRAIN",
	"MORTS DONT LA SOURCE APPARTIENT A LA VICTIME",
	"le kill-feed credite un AUTRE joueur : les deux verites sont vraies et divergent",
	"MORTS INFLIGEES PAR UN BOT",
	"le feed porte la MORT, pas le kill : le tueur vient du roster de replication",
	"CONTROLE NEGATIF — le scan n invente pas de morts",
	"les morts sont toutes",
	"DESACCORD doit valoir 0",
	"COMPTEUR SEPARE, il ne se compte pas sur les candidats",
	"bruit MESURE nul sur 5 films, et AVEUGLE a un tag servi par le seul SCAN",
}

// phrasesCumul : les quatre denominateurs, chacun avec sa qualification.
var phrasesCumul = []string{
	"LES QUATRE DENOMINATEURS — ne jamais ecrire << X % des morts >> sans dire lequel",
	"couples REELS = 100.0 %",
	"C EST LE DENOMINATEUR DE REFERENCE",
	"couples reconstruits par la reconstruction de kill-feed = 99.7 %",
	"morts du KILL-FEED = 98.9 %",
	"le chunk HIGHLIGHT est HUMAIN SEUL",
	// LE SEUL TAUX PUBLIE QUI A CHANGE AVEC RE_LOG 7ter.79 — il valait 98.9 % (376/380). Les
	// morts infligees PAR un bot lui manquaient : elles n etaient decodees par personne. Le
	// DENOMINATEUR (380, constante declaree) n a pas bouge, et les trois autres non plus.
	"morts de l API = 100.0 %",
	"LA SEULE REFERENCE COMPLETE, bots compris",
	"LE NUMERATEUR A CHANGE, LE DENOMINATEUR NON",
	"CONSTANTE DECLAREE",
	"CONTROLE NEGATIF CUMULE — le scan n invente pas de morts",
	"POPULATION OU LA CIBLE EST ABSENTE PAR CONSTRUCTION",
	"30 ancres HORODATEES",
	"Dire << 30 ancres >> est juste",
	"<< les 34 ancres >> ne l est pas",
	"servies par le SCAN SEUL",
	"vaut en LIGNES PUBLIEES, pas en",
	"0 etiquette de source PUBLIEE et FAUSSE",
	"LE 4v4 SEULEMENT",
	"la bijection indice -> joueur y a une marge NULLE",
	"LES MORTS INFLIGEES PAR UN BOT N Y SONT PAS IDENTIFIEES NON PLUS",
	"ZERO appariee, ZERO publiee",
}

func lireGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(goldenPath(name)) //nolint:gosec // nom de film fige dans le code
	if err != nil {
		t.Fatalf("golden %s illisible : %v — regenerer avec "+
			"KILLSOURCE_FIXTURES=<dir> go test -run Golden -update", name, err)
	}
	return string(b)
}

// TestGoldenPhrasesJustes : les fichiers figes disent-ils encore ce que leurs chiffres signifient ?
func TestGoldenPhrasesJustes(t *testing.T) {
	for _, ref := range references {
		t.Run(ref.film, func(t *testing.T) {
			g := lireGolden(t, ref.film)
			for _, p := range phrasesParFilm {
				if !strings.Contains(g, p) {
					t.Errorf("phrase absente du golden : %q\n"+
						"UN CHIFFRE SANS SA PHRASE NE PROUVE RIEN — le golden a perdu sa qualification", p)
				}
			}
		})
	}
	t.Run("cumul", func(t *testing.T) {
		g := lireGolden(t, "cumul")
		for _, p := range phrasesCumul {
			if !strings.Contains(g, p) {
				t.Errorf("phrase absente du cumul : %q", p)
			}
		}
	})
}

// TestGoldenAncresToutesPresentesAvecLeurTag : LE FILET LE PLUS IMPORTANT.
//
// Un compte de couverture peut rester juste pendant qu une etiquette devient fausse. Les 30 ancres
// sont la seule chose qui l attrape, et elles ne valent que si leur TAG est ecrit : une ancre
// reduite a un instant ne verifie plus rien.
func TestGoldenAncresToutesPresentesAvecLeurTag(t *testing.T) {
	total := 0
	for _, ref := range references {
		g := lireGolden(t, ref.film)
		as := anchors[ref.film]
		total += len(as)
		for _, a := range as {
			ligne := fmt.Sprintf("%s  %08x  divergence=%s", a.at, a.tag, ouiNonGolden(a.diverges))
			if !strings.Contains(g, ligne) {
				t.Errorf("%s : ancre absente ou modifiee dans le golden : %q", ref.film, ligne)
			}
		}
		if strings.Contains(g, "ANCRE NON RETROUVEE") {
			t.Errorf("%s : le golden fige une ancre NON RETROUVEE — un golden ne fige pas un echec", ref.film)
		}
		if strings.Contains(g, "AUCUNE LIGNE PUBLIEE A CETTE SECONDE") {
			t.Errorf("%s : le golden fige une seconde sans ligne publiee", ref.film)
		}
	}
	if total != 30 {
		t.Errorf("%d ancres horodatees declarees, attendu 30 — c est le chiffre juste ; les 34 du "+
			"bilan comptent aussi des confrontations qui ne sont pas des lignes de kill", total)
	}
}

// TestGoldenLigneDiscriminante : LA LIGNE QUI SEPARE UNE CONFIGURATION JUSTE D UNE FAUSSE.
//
// `fccc61cd 01:25` est LA SEULE des 371 dont l etiquette change selon la configuration, et le
// changement est INVISIBLE AUX COMPTES : la couverture reste a 100 % dans les deux cas (RE_LOG
// 7ter.71). Un golden qui ne porterait que des nombres validerait la configuration fausse sans
// broncher ; celui-ci porte le TAG, et il echoue.
//
// PORTEE, MESUREE ET A NE PAS ELARGIR : le tag concurrent `00000028` — etiquete << degat
// global >>, statut VALIDE, affirmativement faux — l emporte sous une architecture SCAN-D-ABORD.
// Sous l HYBRIDE que ce paquet implemente, la MARCHE sert la ligne et le tag juste sort quelle que
// soit la bascule. Le regime est verifie par [TestLigneDiscriminanteEstServieParLaMarche] ; ce
// test-ci se contente de figer le RESULTAT, qui doit tenir dans les deux regimes.
func TestGoldenLigneDiscriminante(t *testing.T) {
	const film = "fccc61cd"
	g := lireGolden(t, film)
	// L ancre, avec son tag, sa divergence et la voie qui la sert.
	if !strings.Contains(g, "01:25  acd1cff4  divergence=oui") {
		t.Fatalf("%s 01:25 : la ligne discriminante a disparu ou a change de tag.\n"+
			"Attendu `acd1cff4` (M41 SPNKr, Theater). Le tag `00000028` a cet instant signifierait "+
			"que la configuration a bascule du cote faux — la couverture, elle, resterait a 100 %%", film)
	}
	if !strings.Contains(g, `01:25  acd1cff4  divergence=oui  "M41 SPNKr"  ARME  marche`) {
		t.Errorf("%s 01:25 : l etiquette ou la VOIE ont change. La voie compte : c est parce que "+
			"la MARCHE sert cette ligne que le test de magnitude est inerte aujourd hui", film)
	}
	// La meme mort, cote table des divergences : le CREDIT doit y figurer.
	if !strings.Contains(g, "01:25  JGtm  acd1cff4") {
		t.Errorf("%s 01:25 : la ligne n apparait plus dans la table des morts a source-victime "+
			"avec sa victime et son tag", film)
	}
	if strings.Contains(g, "00000028") {
		t.Errorf("%s : le tag `00000028` (<< degat global >>) apparait dans le golden — c est la "+
			"signature de la configuration FAUSSE que la ligne 01:25 discrimine", film)
	}
	// La configuration gelee. Elle reste le filet du regime scan-servi, meme inerte aujourd hui.
	if !DefaultOptions().StrongTagRequired {
		t.Error("StrongTagRequired est passe a false : c est le filet du regime ou le SCAN sert " +
			"cette ligne, et il ne se retire pas sans re-mesurer ce regime")
	}
}

// TestLigneDiscriminanteEstServieParLaMarche : POURQUOI LA LIGNE 01:25 EST DISCRIMINANTE, ET DANS
// QUEL REGIME ELLE L EST.
//
// CE TEST EXISTE PARCE QUE L A/B A REFUTE LA FORMULATION HERITEE. Le paquet portait, en
// commentaire, l enonce de 7ter.70/7ter.71 : *<< sans le test de magnitude, `fccc61cd 01:25`
// publie `00000028` >>*. C EST VRAI SOUS L ARCHITECTURE SCAN-D-ABORD, sur laquelle cet enonce a
// ete mesure. C EST FAUX ICI : le paquet implemente l HYBRIDE, la MARCHE sert cette ligne, et elle
// publie `acd1cff4` que le filtre soit actif ou non. A/B mesure : tag inchange, couverture
// inchangee. Cela concorde avec 7ter.72 (0)(C), section verifiee : *sous l hybride le filtre ne
// change plus AUCUNE etiquette sur les quatre films*.
//
// CE QUE LE TEST FIGE DONC, ET C EST L INVARIANT UTILE :
//
//	(1) 01:25 publie `acd1cff4`, divergence comprise — l ancre Theater ;
//	(2) elle est servie par la MARCHE — c est CE FAIT qui rend le filtre inerte aujourd hui ;
//	(3) desactiver le filtre ne change RIEN a l etiquette — mesure, pas suppose.
//
// SI (2) TOMBAIT, le scan reprendrait la ligne et le filtre redeviendrait porteur : c est
// exactement le regime ou l enonce herite est vrai. Le test attrape ce basculement de regime, qui
// est la seule chose interessante a surveiller ici.
//
// LE FILTRE RESTE ACTIF PAR DEFAUT. Il n achete rien sur ce corpus ; il est le filet du regime
// scan-servi, et son cout en vrais positifs est nul (mesure : multiplicite 1 partout).
func TestLigneDiscriminanteEstServieParLaMarche(t *testing.T) {
	dir := os.Getenv("KILLSOURCE_FIXTURES")
	if dir == "" {
		t.Skip("KILLSOURCE_FIXTURES non defini : l A/B sur film reel est ignore")
	}
	src, err := filmsource.LoadDir(filepath.Join(dir, "fccc61cd"), nil)
	if err != nil {
		t.Skipf("film absent : %v", err)
	}
	gelee, err := Decode(t.Context(), "fccc61cd", src, nil)
	if err != nil {
		t.Fatalf("Decode (configuration gelee) : %v", err)
	}
	ligne := ligneA(gelee, "01:25", 0xacd1cff4)
	if ligne == nil {
		t.Fatalf("01:25 : la ligne `acd1cff4` a disparu — c est l ancre Theater qui tombe (vu : %v)",
			tagsA(gelee, "01:25"))
	}
	if !ligne.Diverges {
		t.Error("01:25 : la divergence credit/source a disparu")
	}
	if ligne.Read.Path != PathWalk {
		t.Errorf("01:25 est desormais servie par %q et non par la MARCHE. LE REGIME A CHANGE : le "+
			"test de magnitude redevient PORTEUR sur cette ligne, et l enonce << sans lui elle "+
			"publie 00000028 >> redevient vrai. A re-mesurer avant de toucher a la configuration.",
			ligne.Read.Path)
	}

	faible := DefaultOptions()
	faible.StrongTagRequired = false
	sans, err := Decode(t.Context(), "fccc61cd", src, &faible)
	if err != nil {
		t.Fatalf("Decode (sans le test de magnitude) : %v", err)
	}
	if sans.Coverage.Covered != gelee.Coverage.Covered {
		t.Errorf("la couverture bouge sans le test de magnitude (%d contre %d) : ce filtre a "+
			"toujours achete de la JUSTESSE, jamais de la couverture",
			sans.Coverage.Covered, gelee.Coverage.Covered)
	}
	if ligneA(sans, "01:25", 0xacd1cff4) == nil {
		t.Errorf("01:25 change d etiquette sans le test de magnitude (vu : %v). C est le regime "+
			"scan-servi : mettre a jour le commentaire de StrongTagRequired, qui affirme "+
			"aujourd hui l inverse sur la foi d une mesure faite sous l hybride.", tagsA(sans, "01:25"))
	}
	t.Logf("regime confirme : 01:25 est servie par la MARCHE, et le test de magnitude ne change "+
		"ni l etiquette ni la couverture (%d/%d)", sans.Coverage.Covered, sans.Coverage.RealPairs)
}

// ligneA : la mort publiee a cette seconde portant ce tag. DEUX MORTS PEUVENT TOMBER DANS LA MEME
// SECONDE — on cherche donc le tag, on ne prend pas << la >> ligne de la seconde.
func ligneA(res *Result, at string, tag uint32) *Kill {
	for i := range res.Kills {
		if clock(res.Kills[i].TimeMS) == at && res.Kills[i].Source.Tag == tag {
			return &res.Kills[i]
		}
	}
	return nil
}

func tagsA(res *Result, at string) []string {
	var out []string
	for _, k := range res.Kills {
		if clock(k.TimeMS) == at {
			out = append(out, fmt.Sprintf("%08x", k.Source.Tag))
		}
	}
	return out
}

// TestGoldenMortsAutoInfligeesAvecTagEtCredit : les huit morts ou les deux verites divergent.
//
// CHACUNE PORTE SON TAG *ET* SON CREDIT, parce que c est leur COUPLE qui est la mesure : la
// doctrine dit que le jeu credite un autre joueur pendant que la source appartient a la victime.
// Un golden qui ne garderait que le tag ne verifierait plus la moitie de l affirmation.
func TestGoldenMortsAutoInfligeesAvecTagEtCredit(t *testing.T) {
	// Les tags mesures, film par film (RE_LOG 7ter.73 (1)). Ils sont ecrits ici et non deduits :
	// une valeur attendue qui se recalcule depuis la sortie ne teste rien.
	attendus := map[string][]string{
		"000d5950": {"acd1cff4"},
		"9b191a7f": {"5eaa815c", "fca02b3c"},
		"78919882": {"d468178e", "0d203522"},
		"fccc61cd": {"acd1cff4", "ba5ac78e", "5e389b5d"},
	}
	total := 0
	for _, ref := range references {
		g := lireGolden(t, ref.film)
		bloc := sectionDe(g, "## MORTS DONT LA SOURCE APPARTIENT A LA VICTIME")
		if bloc == "" {
			t.Fatalf("%s : section des morts a source-victime absente du golden", ref.film)
		}
		for _, tag := range attendus[ref.film] {
			if !strings.Contains(bloc, tag) {
				t.Errorf("%s : le tag %s n est plus dans les morts a source-victime", ref.film, tag)
			}
		}
		n := strings.Count(bloc, "credit=")
		if n != len(attendus[ref.film]) {
			t.Errorf("%s : %d ligne(s) a source-victime, attendu %d", ref.film, n, len(attendus[ref.film]))
		}
		total += n
	}
	if total != 8 {
		t.Errorf("%d morts a source-victime au total, attendu 8 (confirmees 8/8 en Theater)", total)
	}
}

// TestGoldenMortsParUnBotAvecTagEtCategorie : LA POPULATION QUI N A PAS D ANCRE THEATER.
//
// Les 371 autres lignes sont protegees par 30 confrontations en mode Theater. LES QUATRE
// CI-DESSOUS N EN ONT AUCUNE, ET ELLES N EN AURONT JAMAIS : le kill-feed du jeu n attribue ces
// morts a aucun gamertag, donc on ne peut pas designer la ligne a verifier. C est une LIMITE, elle
// est ecrite, et ce test est ce qui la compense.
//
// CE QU IL FIGE, ET POURQUOI CHAQUE COLONNE COMPTE : le TAG (la quantite brute), la CATEGORIE (le
// modificateur de degat, lu a `comp+0x0c`, INDEPENDANT du tag) et la VOIE. Le couple tag/categorie
// est la seule verification interne disponible ici, et il tient :
//
//	9b191a7f 08:24  `0000d57f` Needler + AttachedDamage  = supercombinaison Needler. La categorie
//	                `AttachedDamage` designe un projectile FIXE sur la cible (RE_LOG 7ter.37), et
//	                le tag designe un Needler — ni l un ni l autre n a servi a choisir l autre.
//	9b191a7f 08:03  `ec919da5` Needler + None            = aiguilles ordinaires, MEME arme, AUTRE
//	                effet. Deux tags pour un Needler, c est exactement la relation many-to-one PAR
//	                EFFET etablie en 7ter.44 — et ici les deux effets tombent sur les deux
//	                categories qui leur correspondent.
//
// RESERVE A PORTER AVEC `0000d57f` : il vit SOUS 0x10000, ou le taux de faux positif du test
// `jpt!` monte a 1.15 % (contre ~1/9 000 000 au-dessus). Ce qui le tient n est pas sa magnitude :
// c est la MARCHE (chaine complete de largeurs connues) et la coherence de categorie ci-dessus.
func TestGoldenMortsParUnBotAvecTagEtCategorie(t *testing.T) {
	attendues := map[string][]string{
		"000d5950": nil,
		"78919882": nil,
		"9b191a7f": {
			`05:46  Madina97294  343 Aloysius [bot]  1f11df6b  "VK78 Commando"  Headshot  marche  assistant=xSONNYX117`,
			`08:03  JGtm  343 Aloysius [bot]  ec919da5  "Needler"  None  marche  assistant=RaiiZeNBack`,
			`08:24  Chocoboflor  343 Aloysius [bot]  0000d57f  "Needler"  AttachedDamage  marche  assistant=AUCUN (mesure)`,
		},
		"fccc61cd": {
			`08:49  HizaroMne4262  343 PardonMy [bot]  0ff7b934  "Shock Rifle"  Headshot  marche  assistant=AUCUN (mesure)`,
		},
	}
	total := 0
	for _, ref := range references {
		g := lireGolden(t, ref.film)
		bloc := sectionDe(g, "## MORTS INFLIGEES PAR UN BOT")
		if bloc == "" {
			t.Fatalf("%s : section des morts infligees par un bot absente du golden", ref.film)
		}
		for _, l := range attendues[ref.film] {
			if !strings.Contains(bloc, l) {
				t.Errorf("%s : ligne absente ou modifiee :\n  %s", ref.film, l)
			}
		}
		n := strings.Count(bloc, "assistant=")
		if n != len(attendues[ref.film]) {
			t.Errorf("%s : %d mort(s) infligee(s) par un bot, attendu %d",
				ref.film, n, len(attendues[ref.film]))
		}
		total += n
	}
	if total != 4 {
		t.Errorf("%d morts infligees par un bot au total, attendu 4 — c est EXACTEMENT la somme "+
			"des kills que l API attribue aux bid(...) des quatre films (3 + 1)", total)
	}
}

// TestGoldenControleNegatif : le scan invente-t-il des morts ?
//
// Les morts sont TOUTES dans les paquets a evenements. Les paquets sans evenement forment donc une
// population ou la cible est ABSENTE par construction, et le nombre de candidats qui y tombent
// mesure directement le pouvoir des quatre tests. La borne retenue est LARGE et volontairement
// insensible au bruit : ce qu on veut attraper, c est un decodeur devenu permissif, pas une
// variation d une unite.
func TestGoldenControleNegatif(t *testing.T) {
	const maxCandidatsParFilm = 20
	for _, ref := range references {
		g := lireGolden(t, ref.film)
		bloc := sectionDe(g, "## CONTROLE NEGATIF")
		if bloc == "" {
			t.Fatalf("%s : section de controle negatif absente", ref.film)
		}
		var paquets, bits, cands int
		if _, err := fmt.Sscanf(ligneChiffres(bloc),
			"%d paquet(s) SANS evenement · %d bits balayes · %d candidat(s) accepte(s)",
			&paquets, &bits, &cands); err != nil {
			t.Fatalf("%s : ligne de controle negatif illisible (%v) : %q", ref.film, err, ligneChiffres(bloc))
		}
		if paquets == 0 || bits == 0 {
			t.Errorf("%s : controle negatif sur une population VIDE (%d paquets, %d bits) — "+
				"un controle negatif sans population ne prouve rien", ref.film, paquets, bits)
		}
		if cands > maxCandidatsParFilm {
			t.Errorf("%s : %d candidat(s) dans %d bits de paquets SANS evenement (borne %d). "+
				"Le scan est devenu permissif : ses quatre tests ne discriminent plus.",
				ref.film, cands, bits, maxCandidatsParFilm)
		}
	}
}

// sectionDe : le corps d une section du golden, jusqu a la suivante.
func sectionDe(g, titre string) string {
	i := strings.Index(g, titre)
	if i < 0 {
		return ""
	}
	reste := g[i+len(titre):]
	if j := strings.Index(reste, "\n## "); j >= 0 {
		return reste[:j]
	}
	return reste
}

// ligneChiffres : la premiere ligne d une section qui commence par un chiffre.
func ligneChiffres(bloc string) string {
	for _, l := range strings.Split(bloc, "\n") {
		if l != "" && l[0] >= '0' && l[0] <= '9' {
			return l
		}
	}
	return ""
}
