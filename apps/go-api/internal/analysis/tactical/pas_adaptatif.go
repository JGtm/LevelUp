package tactical

// pas_adaptatif.go — LE PAS DE LA GRILLE S'ADAPTE A LA DENSITE, LE PLANCHER NE BOUGE JAMAIS
// (decision D6 du plan master 2026-09-09, lot 3.2).
//
// # LE DEFAUT CORRIGE
//
// Sur Illusion, apres le backfill des positions de kill (couverture 26,6 % -> 67,9 %,
// 38 matchs retenus), le plan de la page Tactique restait VIDE : au pas de 0,5 m, AUCUNE
// cellule n'atteignait les trois matchs distincts du plancher. Le message affiche parlait
// de « pas assez de matchs mesures » — c'etait faux : les matchs etaient la, c'est la
// DENSITE PAR CELLULE qui manquait. Une cellule de 0,5 m fait 0,25 m² ; sur une carte
// ouverte, trois joueurs qui traversent la meme zone dans trois matchs differents n'y
// posent pas le pied au meme quart de metre carre.
//
// # LES DEUX LEVIERS, ET CELUI QU'ON REFUSE
//
// Pour qu'une cellule atteigne le plancher, on peut abaisser le plancher ou agrandir la
// cellule. LE PLANCHER NE BAISSE PAS : trois matchs distincts est ce qui rend une cellule
// FIABLE (deux observations independantes ne font pas une regularite, cf. la mesure de
// `cmd/mappos-build` rappelee dans doc.go) — l'abaisser ne rend pas la carte plus vraie, il
// rend son mensonge plus dense. C'est la cellule qui grossit, et le prix se voit : une
// grille de 2 m est plus grossiere, elle est PUBLIEE (`TacticalRaster.PasM`) et affichee a
// l'utilisateur, qui sait alors a quelle resolution il regarde.
//
// # LA SUITE DES PAS
//
// 0,5 -> 1 -> 2 m. Elle part du pas mesure (`PasParDefautM`) et DOUBLE : chaque cellule
// grosse est alors l'union exacte de quatre cellules fines, ce qui rend le regroupement
// exact (cf. `Readresser`) et permet de relire des sidecars cuits a 0,5 m sans les recuire.
// Elle s'arrete a 2 m parce qu'au-dela une cellule (4 m²) n'est plus un LIEU : c'est un
// quartier de carte, et « je meurs par ici » cesse d'etre une lecture de placement.
//
// # POURQUOI ON NE PREND PAS SIMPLEMENT LE PAS LE PLUS GROSSIER
//
// Le pas adaptatif ne doit RIEN changer aux cartes deja lisibles : sur celles-la, la
// premiere tentative suffit et le plan est exactement celui d'avant (test
// `TestChoisirPasNeChangeRienQuandLePasParDefautSuffit`). Grossir par defaut aurait paye
// pour toutes les cartes le probleme d'une seule.

import (
	"errors"
	"math"
)

// PasAdaptatifsM est la suite des pas essayes, en metres, DANS L'ORDRE : le plus fin
// d'abord (cf. l'en-tete). Variable et non constante parce qu'un slice ne peut pas etre
// constant ; elle n'est jamais modifiee — les tests la passent en argument.
var PasAdaptatifsM = []float64{PasParDefautM, 2 * PasParDefautM, 4 * PasParDefautM}

// CellulesLisiblesMin est N : le nombre de cellules passant le plancher a partir duquel un
// plan est considere comme LISIBLE, et donc le pas retenu.
//
// # LA VALEUR EST MESUREE SUR L'ECHELLE DE COULEUR, PAS CHOISIE A VUE
//
// La rampe du plan va du p50 au p95 des cellules lisibles (quantile.go, `Echelle`), et la
// doc de `OrdreP95` dit pourquoi le haut n'est pas le maximum : « une seule cellule extreme
// aplatit toutes les autres si elle borne l'echelle ». Avec l'interpolation lineaire de
// `quantile`, cette promesse n'est tenue qu'a partir d'un certain nombre de valeurs : en
// dessous de 22 cellules, le p95 est une interpolation entre les DEUX plus grandes, et la
// cellule extreme borne encore l'echelle a elle seule.
//
// 22 est le PLUS PETIT compte pour lequel au moins deux cellules depassent la borne — donc
// le plus petit pour lequel la carte peinte a une echelle qui decrit une distribution et non
// un extreme. En dessous, le plan n'est pas « peu fourni » : il est mal etalonne.
//
// Le seuil est verifie sur pieces par
// `TestCellulesLisiblesMinEstLeSeuilOuLEchelleCesseDEtreDicteeParUneSeuleCellule`, dans les
// deux sens (la propriete tient a N, elle ne tient pas a N-1) : N ne peut donc pas deriver
// sans qu'un test le dise.
const CellulesLisiblesMin = 22

// ErrAucunPas : choisir un pas dans une suite vide n'a pas de resultat par defaut — un
// appelant qui passe une suite vide s'est trompe, il ne demande pas « la grille habituelle ».
var ErrAucunPas = errors.New("tactical: aucun pas de grille a essayer")

// TentativePas est le resultat d'UN pas essaye. Publiee (dans le journal du service) parce
// qu'un plan grossier doit pouvoir s'expliquer : « a 0,5 m, 0 cellule ; a 1 m, 4 ; a 2 m, 31 ».
type TentativePas struct {
	PasM             float64
	CellulesLisibles int
}

// LectureAdaptative est le resultat du choix : le raster retenu, sa suffisance, et la trace
// des tentatives.
type LectureAdaptative struct {
	// Raster est la lecture RETENUE. Jamais nil quand l'erreur est nil.
	Raster *Raster

	// Suffisante dit qu'un pas a atteint le seuil N. FAUX ne veut pas dire vide : il veut
	// dire « meme a 2 m, la densite ne suffit pas » — ce que la page doit ANNONCER comme
	// tel plutot que de parler d'un manque de matchs.
	Suffisante bool

	// Tentatives porte les pas essayes, dans l'ordre. Une seule entree quand le premier
	// pas a suffi : les suivants ne sont pas calcules.
	Tentatives []TentativePas
}

// PasM rend le pas du raster retenu, en metres.
func (l LectureAdaptative) PasM() float64 {
	if l.Raster == nil {
		return 0
	}
	return l.Raster.PasM()
}

// ChoisirPas essaie les pas dans l'ordre et retient LE PREMIER dont la lecture atteint
// `minCellules` cellules lisibles.
//
// `essayer` construit le raster sur la grille proposee ET rend le nombre de cellules qui
// seront PEINTES a ce pas. Les deux voyagent ensemble parce que ce compte depend de la
// lecture : une lecture signee (« ou je gagne ») applique son plancher PAR COTE, et un
// comptage generique ici aurait mesure autre chose que ce que la page affiche.
//
// FAUTE DE PAS SUFFISANT, LA TENTATIVE LA PLUS FOURNIE EST RETENUE — a egalite, la plus
// FINE (donc la moins agregee). Rendre un raster vide dans ce cas aurait jete des cellules
// mesurees, vraies et affichables ; rendre systematiquement la plus grossiere aurait
// agrege pour rien quand l'agregation n'apporte aucune cellule.
func ChoisirPas(pasM []float64, minCellules int,
	essayer func(Grille) (*Raster, int, error)) (LectureAdaptative, error) {
	if len(pasM) == 0 {
		return LectureAdaptative{}, ErrAucunPas
	}
	out := LectureAdaptative{Tentatives: make([]TentativePas, 0, len(pasM))}
	meilleur := -1
	for _, pas := range pasM {
		g, err := NouvelleGrille(pas)
		if err != nil {
			return LectureAdaptative{}, err
		}
		raster, lisibles, err := essayer(g)
		if err != nil {
			return LectureAdaptative{}, err
		}
		out.Tentatives = append(out.Tentatives, TentativePas{PasM: pas, CellulesLisibles: lisibles})
		// STRICTEMENT SUPERIEUR : a egalite, la tentative deja retenue est la plus fine,
		// et c'est elle qu'on garde (la moins agregee des lectures egales).
		if lisibles > meilleur {
			meilleur = lisibles
			out.Raster = raster
		}
		if lisibles >= minCellules {
			out.Suffisante = true
			return out, nil
		}
	}
	return out, nil
}

// Readresser rend l'adresse, sur la grille `cible`, de la cellule `c` adressee sur la
// grille `source`.
//
// L'ANCRAGE EST L'ORIGINE DU MONDE DES DEUX COTES (cf. doc.go) : la conversion passe donc
// par le CENTRE de la cellule source, jamais par un decalage relatif aux bornes d'une
// lecture. Quand le pas cible est un multiple du pas source — le seul cas produit par
// `PasAdaptatifsM`, qui double —, la cellule source est entierement CONTENUE dans la
// cellule cible, et le regroupement est exact.
func Readresser(c Cellule, source, cible Grille) Cellule {
	x, y := source.Centre(c)
	nouvelle, ok := cible.Cellule(x, y)
	if !ok {
		// Le centre d'une cellule d'adresse entiere est toujours fini ; ce cas ne se
		// produit qu'a des adresses assez grandes pour deborder le float. On rend alors
		// l'adresse d'origine plutot qu'une cellule inventee.
		return c
	}
	return nouvelle
}

// ReadresserComptes reexprime des comptes deja agreges sur la grille `cible`.
//
// C'est ce qui permet d'essayer un pas plus grossier SANS RECUIRE les sidecars : ceux-ci
// portent des cellules de 0,5 m (`domain.TacticalRasterPasM`), et un pas de 1 ou 2 m s'en
// deduit exactement (cf. `Readresser`). Les doublons ne sont pas fusionnes ici : c'est
// `RasteriseComptes` qui accumule, par cellule ET par match.
func ReadresserComptes(comptes []CompteCellule, source, cible Grille) []CompteCellule {
	if math.Abs(source.PasM()-cible.PasM()) < 1e-9 {
		return comptes
	}
	out := make([]CompteCellule, 0, len(comptes))
	for _, c := range comptes {
		out = append(out, CompteCellule{
			Cellule:     Readresser(c.Cellule, source, cible),
			MatchID:     c.MatchID,
			Occurrences: c.Occurrences,
		})
	}
	return out
}
