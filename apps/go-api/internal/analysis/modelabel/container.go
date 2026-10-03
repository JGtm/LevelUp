package modelabel

import "strings"

// containerTokens — LES JETONS DE CONTENEUR DE LA GRAMMAIRE DES pair_name HALO INFINITE, et
// il n'y en a qu'une liste dans le dépôt.
//
// Un pair_name existe sous DEUX grammaires : « Conteneur:Mode on Carte » (« Arena:Slayer on
// Bazaar », « Ranked:Doubles Slayer on Live Fire ») et la forme INVERSÉE « Mode:Conteneur
// [qualificatif] on Carte » (« Slayer:Arena on Live Fire », « CTF:Arena Neutral Flag on
// Cliffhanger », « CTF:BTB Fiesta on Highpower », « Slayer:Doubles on Empyrean »). Reconnaître
// un conteneur, à gauche comme en tête de la partie droite, est ce qui distingue le MODE du
// reste — sans cette liste, la forme inversée rend « Arena », « BTB Fiesta » ou « Doubles »
// comme libellé de mode, jamais traduits, et écrase des modes distincts dans un même item de
// filtre (531 matchs « Arena » dans la base partagée, mélange de CTF et de Slayer).
//
// POURQUOI LA LISTE VIT ICI ET PAS DANS `games/halo_infinite/mode_category.go` :
//   - `analysis` (NormalizeModeLabel) ne peut pas importer `games/halo_infinite` — cycle —, et
//     ce paquet est une FEUILLE (aucune dépendance hors `strings`/`regexp`) ;
//   - `modePrefixToCategory` porte une carte préfixe→CATÉGORIE dont la sémantique est
//     différente : Gruntpocalypse et Firefight y sont des catégories, pas des conteneurs de
//     grammaire (« Gruntpocalypse:Fiesta » a Gruntpocalypse pour MODE). Les deux listes se
//     recoupent mais ne sont pas la même chose ; `mode_category.go` renvoie ici.
//
// Ordre : du plus LONG au plus court, pour que « Super Fiesta » gagne sur « Fiesta » et
// « BTB Heavies » sur « BTB » sans dépendre de l'itération. Comparaison insensible à la casse.
// Jetons repris ailleurs (tests, appelants) : déclarés une fois.
const (
	ContainerArena       = "Arena"
	ContainerDoubles     = "Doubles"
	ContainerSuperFiesta = "Super Fiesta"
)

var containerTokens = []string{
	"Super Husky Raid",
	ContainerSuperFiesta,
	"BTB Heavies",
	"Castle Wars",
	"Husky Raid",
	"Community",
	"Tactical",
	"Assault",
	ContainerDoubles,
	"Ranked",
	"Fiesta",
	ContainerArena,
	"Event",
	"BTB",
}

// IsContainer dit si le libellé ENTIER (trimé, insensible à la casse) est un jeton de
// conteneur : « Arena », « btb heavies » → vrai ; « Arenax », « Slayer » → faux.
func IsContainer(label string) bool {
	label = strings.TrimSpace(label)
	for _, tok := range containerTokens {
		if strings.EqualFold(label, tok) {
			return true
		}
	}
	return false
}

// SplitContainer reconnaît un jeton de conteneur en TÊTE de `right` (mot entier, insensible à
// la casse, le jeton le plus LONG gagne) et rend le conteneur tel qu'écrit dans `right`, le
// reste trimé, et ok. « Arena Neutral Flag » → (« Arena », « Neutral Flag », true) ;
// « BTB Fiesta » → (« BTB », « Fiesta », true) — pas « BTB Heavies » ; « Arenax » →
// ("", "", false) : le mot entier est exigé.
func SplitContainer(right string) (container, remainder string, ok bool) {
	right = strings.TrimSpace(right)
	for _, tok := range containerTokens {
		n := len(tok)
		if len(right) < n || !strings.EqualFold(right[:n], tok) {
			continue
		}
		if len(right) > n && isWordChar(right[n]) {
			continue // « Arenax » : le jeton n'est pas un mot entier
		}
		return right[:n], strings.TrimSpace(right[n:]), true
	}
	return "", "", false
}
