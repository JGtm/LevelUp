package killsource

// golden_cumul_test.go — LE CUMUL DES QUATRE FILMS, ET LES QUATRE DENOMINATEURS.
//
// C EST LE FICHIER QUE L ON CITERA. Les chiffres du chantier circulent sous forme de
// pourcentages ; trois denominateurs coexistent hors ligne, un quatrieme n existe qu a l API, et
// ils ne donnent pas le meme resultat. Ce cumul les ecrit tous les quatre, chacun avec la phrase
// qui le qualifie, pour qu il n existe plus de raison d ecrire << X % des morts >> sans dire
// lequel.
//
// LE QUATRIEME N EST PAS CALCULE, IL EST DECLARE : les morts de l API sont la seule reference
// complete, et rien dans le film ne les rend. La constante est nommee, sourcee, et le fichier le
// dit — c est la seule facon honnete de publier 380/380. Ce NUMERATEUR valait 376 avant RE_LOG
// 7ter.79, qui a ajoute les morts INFLIGEES PAR un bot ; le DENOMINATEUR, lui, n a pas bouge, et
// [cumul.ecrireDenominateurs] ecrit ce changement dans la sortie figee plutot que de l effacer.

import (
	"fmt"
	"strings"
)

// cumul : l addition des quatre films. ON ADDITIONNE DES NUMERATEURS ET DES DENOMINATEURS, jamais
// des pourcentages — moyenner des taux de films de tailles differentes donne un nombre qui ne
// correspond a aucune population.
type cumul struct {
	films        []string
	couverts     int
	reels        int
	reconstruits int
	mortsFeed    int
	fabriques    int
	mortsBot     int
	mortsParBot  int
	divergences  int
	walkPop      int
	walkOK       int
	scanPop      int
	scanOK       int
	desaccords   int
	multi        int
	negPaquets   int
	negBits      int
	negCands     int
}

func newCumul() *cumul { return &cumul{} }

func (c *cumul) ajouter(res *Result, neg negControle) {
	cv := res.Coverage
	c.negPaquets += neg.paquets
	c.negBits += neg.bits
	c.negCands += neg.candidats
	c.films = append(c.films, res.Health.Film)
	c.couverts += cv.Covered
	c.reels += cv.RealPairs
	c.reconstruits += cv.ReconstructedPairs
	c.mortsFeed += cv.FeedDeaths
	c.fabriques += cv.GhostPairs
	c.mortsBot += cv.BotDeaths
	c.mortsParBot += cv.BotKillerDeaths
	c.walkPop += res.Stats.Walk.Population
	c.walkOK += res.Stats.Walk.Matched
	c.scanPop += res.Stats.Scan.Population
	c.scanOK += res.Stats.Scan.Matched
	c.desaccords += res.Stats.Disagree
	c.multi += res.Stats.MultiCandidate
	for _, k := range res.Kills {
		if k.Diverges {
			c.divergences++
		}
	}
}

func (c *cumul) rendre() string {
	var b strings.Builder
	fmt.Fprintf(&b, enteteGolden, "cumul des films "+strings.Join(c.films, ", "))
	c.ecrireDenominateurs(&b)
	c.ecrireVoies(&b)
	c.ecrireControleNegatif(&b)
	c.ecrireVeriteTerrain(&b)
	c.ecrirePerimetre(&b)
	return b.String()
}

// ecrireControleNegatif : LE SCAN N INVENTE PAS DE MORTS, cumule sur les quatre films.
//
// Les morts sont TOUTES dans les paquets a evenements. Les paquets sans evenement forment donc
// une population ou la cible est ABSENTE par construction : tout candidat qui y tombe est un faux
// positif, et leur taux mesure directement le pouvoir des quatre tests du scan.
func (c *cumul) ecrireControleNegatif(b *strings.Builder) {
	fmt.Fprint(b, "\n## CONTROLE NEGATIF CUMULE — le scan n invente pas de morts\n")
	fmt.Fprintf(b, "%d candidat(s) sur %d bits de paquets SANS evenement (%d paquets).\n",
		c.negCands, c.negBits, c.negPaquets)
	fmt.Fprint(b, "   POPULATION OU LA CIBLE EST ABSENTE PAR CONSTRUCTION : les morts sont toutes\n")
	fmt.Fprint(b, "   dans les paquets A EVENTS, donc chaque candidat compte ici est un faux positif.\n")
	fmt.Fprint(b, "   Sans ce controle, un decodeur devenu permissif passerait tous les autres tests.\n")
}

// ecrireDenominateurs : LES QUATRE, chacun avec sa phrase. C est le coeur du fichier.
func (c *cumul) ecrireDenominateurs(b *strings.Builder) {
	fmt.Fprint(b, "\n## LES QUATRE DENOMINATEURS — ne jamais ecrire << X % des morts >> sans dire lequel\n")
	fmt.Fprintf(b, "%d / %d couples REELS = %s\n",
		c.couverts, c.reels, taux(c.couverts, c.reels))
	fmt.Fprint(b, "   les couples que la reconstruction de kill-feed produit, MOINS les couples\n")
	fmt.Fprint(b, "   FABRIQUES. C EST LE DENOMINATEUR DE REFERENCE : un couple fabrique n est pas une\n")
	fmt.Fprint(b, "   mort manquee, c est une mort qui n existe pas (la vraie victime est un bot).\n")
	fmt.Fprintf(b, "%d / %d couples reconstruits par la reconstruction de kill-feed = %s\n",
		c.couverts, c.reconstruits, taux(c.couverts, c.reconstruits))
	fmt.Fprintf(b, "   dont %d couple(s) FABRIQUE(S).\n", c.fabriques)
	fmt.Fprintf(b, "%d / %d morts du KILL-FEED = %s\n",
		c.couverts, c.mortsFeed, taux(c.couverts, c.mortsFeed))
	fmt.Fprint(b, "   le chunk HIGHLIGHT est HUMAIN SEUL : il ne contient AUCUNE mort de bot.\n")
	api := c.couverts + c.mortsBot + c.mortsParBot
	fmt.Fprintf(b, "%d / %d morts de l API = %s\n", api, apiDeaths, taux(api, apiDeaths))
	fmt.Fprintf(b, "   LA SEULE REFERENCE COMPLETE, bots compris. Les %d morts de BOT et les %d morts\n",
		c.mortsBot, c.mortsParBot)
	fmt.Fprint(b, "   INFLIGEES PAR un bot s ajoutent ici et NULLE PART AILLEURS : aucune des deux n a\n")
	fmt.Fprint(b, "   jamais ete aux trois denominateurs precedents. Le 380 est une CONSTANTE DECLAREE\n")
	fmt.Fprint(b, "   (constante `apiDeaths`), pas une quantite decodee : rien dans le film ne la rend.\n")
	fmt.Fprintf(b, "   LE NUMERATEUR A CHANGE, LE DENOMINATEUR NON : il valait %d avant RE_LOG 7ter.79\n",
		c.couverts+c.mortsBot)
	fmt.Fprint(b, "   (98.9 %), les morts infligees PAR un bot n etant decodees par personne. Les trois\n")
	fmt.Fprint(b, "   autres denominateurs sont INCHANGES, numerateur compris.\n")
}

func (c *cumul) ecrireVoies(b *strings.Builder) {
	fmt.Fprint(b, "\n## LE COUT, VENTILE PAR VOIE — jamais deux precisions directement comparables\n")
	fmt.Fprintf(b, "marche : %d / %d apparie(s) au couple exact = %s\n",
		c.walkOK, c.walkPop, taux(c.walkOK, c.walkPop))
	fmt.Fprintf(b, "scan   : %d / %d apparie(s) au couple exact = %s\n",
		c.scanOK, c.scanPop, taux(c.scanOK, c.scanPop))
	fmt.Fprint(b, "   RESERVE NON LEVEE : la bijection indice -> joueur est ajustee sur l UNION des\n")
	fmt.Fprint(b, "   deux voies, dont la marche fournit ~91 % des candidats. Un objectif maximise\n")
	fmt.Fprint(b, "   conjointement n est pas neutre : le taux de la marche peut etre flatte et celui\n")
	fmt.Fprint(b, "   du scan penalise par cet effet seul. Lire ce couple comme une VENTILATION DU\n")
	fmt.Fprint(b, "   COUT (RE_LOG 7ter.73 (5)).\n")
	fmt.Fprintf(b, "DESACCORD entre les deux voies : %d — il DOIT valoir zero\n", c.desaccords)
	fmt.Fprintf(b, "morts a plusieurs candidats en concurrence : %d — il DOIT valoir zero\n", c.multi)
}

func (c *cumul) ecrireVeriteTerrain(b *strings.Builder) {
	n := 0
	for _, as := range anchors {
		n += len(as)
	}
	fmt.Fprint(b, "\n## VERITE TERRAIN\n")
	fmt.Fprintf(b, "%d ancres HORODATEES confirmees en mode Theater, tags compris.\n", n)
	fmt.Fprint(b, "   PORTEE : ce sont 30 LIGNES DE KILL horodatees. Le bilan du chantier parle de 34\n")
	fmt.Fprint(b, "   CONFRONTATIONS — les 4 autres ne sont pas des lignes de kill (questions de\n")
	fmt.Fprint(b, "   classe, corrections d interpretation). Dire << 30 ancres >> est juste ; dire\n")
	fmt.Fprint(b, "   << les 34 ancres >> ne l est pas (RE_LOG 7ter.73 (4f)).\n")
	fmt.Fprintf(b, "%d morts dont la SOURCE APPARTIENT A LA VICTIME, confirmees 8/8 en Theater.\n", c.divergences)
	fmt.Fprint(b, "   Le kill-feed y credite un AUTRE joueur. LES DEUX VERITES SONT VRAIES : elles\n")
	fmt.Fprint(b, "   repondent a des questions differentes.\n")
	fmt.Fprint(b, "5 des 30 ancres sont servies par le SCAN SEUL et ne sont donc PAS protegees par la\n")
	fmt.Fprint(b, "   robustesse de l hybride : 4 sur 9b191a7f (01:00, 03:41, 04:50, 05:12) et 1 sur\n")
	fmt.Fprint(b, "   78919882 (09:18). Le << 11.2x plus robuste >> vaut en LIGNES PUBLIEES, pas en\n")
	fmt.Fprint(b, "   ANCRES (RE_LOG 7ter.73 (4b)(4c)).\n")
	fmt.Fprint(b, "0 etiquette de source PUBLIEE et FAUSSE sur l ensemble du chantier.\n")
}

func (c *cumul) ecrirePerimetre(b *strings.Builder) {
	fmt.Fprint(b, "\n## PERIMETRE — ce que ces chiffres NE couvrent PAS\n")
	fmt.Fprint(b, "LE 4v4 SEULEMENT. En BTB (24 slots, 36 participants) le decodage tient en AGREGAT\n")
	fmt.Fprint(b, "   (224/293 = 76.5 %) mais la bijection indice -> joueur y a une marge NULLE : aucune\n")
	fmt.Fprint(b, "   attribution ligne par ligne n y est publiable. Decision utilisateur : perimetre 4v4.\n")
	fmt.Fprint(b, "   LES MORTS INFLIGEES PAR UN BOT N Y SONT PAS IDENTIFIEES NON PLUS, et c est mesure :\n")
	fmt.Fprint(b, "   7 morts orphelines pour 5 kills de bot plus 1 suicide atteste, 6 des 8 bots hors\n")
	fmt.Fprint(b, "   de la regle d epinglage, ZERO appariee, ZERO publiee. Le code ne publie rien de\n")
	fmt.Fprint(b, "   faux ; la population, elle, n est pas identifiee (RE_LOG 7ter.79 (6)).\n")
	fmt.Fprint(b, "LES SOURCES SANS NOM s affichent << Autres >>. 206 des 468 identifiants du catalogue\n")
	fmt.Fprint(b, "   ne remontent a aucune source nommee. La NATURE, elle, reste juste.\n")
	fmt.Fprint(b, "LA PEREMPTION DU CATALOGUE est un risque MINEUR (les ajouts d armes sont rares). La\n")
	fmt.Fprint(b, "   detection reste posee et testee ; ce n est pas un entretien regulier.\n")
}

// taux : un pourcentage a une decimale. Il accompagne toujours son couple numerateur/denominateur,
// jamais seul.
func taux(num, den int) string {
	if den <= 0 {
		return "denominateur vide"
	}
	return fmt.Sprintf("%.1f %%", 100*float64(num)/float64(den))
}
