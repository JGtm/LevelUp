package mappings

import (
	"strings"

	"levelup/go-api/internal/analysis/modelabel"
)

// TeamSize rend la TAILLE D'EQUIPE officielle de la playlist d'une variante, et true quand la table
// `[team_size]` du titre la declare.
//
// LE JETON DE PLAYLIST SE LIT A UNE PLACE, JAMAIS AU MILIEU DU NOM : a gauche du « : »
// (`BTB:CTF`, `Squad:King of the Hill`), sinon en tete de la partie droite (`CTF:Arena`,
// `Slayer:Arena Super Fiesta`), sinon en tete du nom entier (`BTB (2025-03)`). Le suffixe de carte
// (« on … ») est retire d'abord. Chercher le jeton n'importe ou ferait d'un `BTB:Fiesta Slayer`
// une Fiesta.
//
// nil-safe, table absente, nom vide ou playlist non declaree -> (0, false) : l'appelant garde sa
// capacite estimee (aucune taille devinee).
func (s *RegulationSet) TeamSize(gameVariantName string) (int, bool) {
	if s == nil || len(s.teamSize) == 0 {
		return 0, false
	}
	nom := strings.TrimSpace(modelabel.StripMapSuffix(gameVariantName))
	if nom == "" {
		return 0, false
	}
	gauche, droite, deuxParts := strings.Cut(nom, ":")
	candidats := []string{nom}
	if deuxParts {
		candidats = []string{gauche, droite}
	}
	for _, c := range candidats {
		if jeton := s.jetonDEquipeEnTete(c); jeton != "" {
			return s.teamSize[jeton], true
		}
	}
	return 0, false
}

// jetonDEquipeEnTete rend la cle de `[team_size]` qui ouvre `texte` comme mot entier (insensible a la
// casse, la plus longue gagnant), ou "".
func (s *RegulationSet) jetonDEquipeEnTete(texte string) string {
	texte = strings.TrimSpace(texte)
	low := strings.ToLower(texte)
	meilleur := ""
	for jeton := range s.teamSize {
		j := strings.ToLower(jeton)
		if len(j) <= len(meilleur) || !strings.HasPrefix(low, j) {
			continue
		}
		if len(low) > len(j) && estAlphanumerique(low[len(j)]) {
			continue // « Arenax » : le jeton n'est pas un mot entier
		}
		meilleur = jeton
	}
	return meilleur
}
