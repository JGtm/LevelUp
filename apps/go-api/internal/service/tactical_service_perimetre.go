// Package service — tactical_service_perimetre.go : LE PERIMETRE et LES AXES de
// l'onglet Tactique.
//
// Fichier separe de tactical_service.go (phase 4 bis, 2026-09-06) : celui-la
// orchestre (lire, rasteriser, mesurer, journaliser) et frolait le seuil de 500
// lignes ; ici vivent les regles qui disent CE QU'ON REGARDE — quels matchs, et
// quels joueurs dans ces matchs.
//
// Les deux notions se ressemblent et ne doivent surtout pas fusionner :
//
//	le PERIMETRE   les match_id resolus en amont (liste blanche) et la COMPOSITION
//	               qui les resserre ;
//	l'AXE « qui »  qui, DANS un match retenu, alimente la lecture — moi, la
//	               composition, ou l'autre equipe.
package service

import (
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
)

// validerLecture refuse une demande hors vocabulaire AVANT toute lecture de base.
func validerLecture(carte, question, qui string, coequipiers []string) error {
	// LA COMPOSITION D'ABORD : elle est la seule entree dont le COUT depend de la
	// taille (un `EXISTS` correle par coequipier), donc la seule qu'il faut refuser
	// avant meme de regarder si la carte existe.
	if err := domain.ValiderComposition(coequipiers); err != nil {
		return err
	}
	if carte == "" {
		// Sentinelle NUE, meme raison : ce message est publie tel quel, et « (carte vide) »
		// distinguerait ce refus-ci des deux autres 404 de la meme famille.
		return domain.ErrTacticalCarteInconnue
	}
	switch question {
	case domain.TacticalQuestionMorts, domain.TacticalQuestionKills, domain.TacticalQuestionGagne:
	default:
		return fmt.Errorf("%w (%q)", domain.ErrTacticalQuestionInconnue, question)
	}
	switch qui {
	case domain.TacticalQuiMoi, domain.TacticalQuiAdversaires:
		return nil
	case domain.TacticalQuiEscouade:
		// « Escouade » = LA COMPOSITION CHOISIE (arbitrage utilisateur du 2026-09-06).
		// Sans composition l'axe n'a aucun contenu : le refuser vaut mieux que retomber
		// sur les coequipiers du match, qui repondrait a une AUTRE question sous le meme
		// nom. Le client ne propose pas l'axe tant qu'aucun coequipier n'est choisi.
		if len(coequipiers) == 0 {
			return domain.ErrTacticalEscouadeSansComposition
		}
		return nil
	default:
		return fmt.Errorf("%w (%q)", domain.ErrTacticalQuiInconnu, qui)
	}
}

// requeteDuScope traduit le perimetre de la page en demande au lecteur. UNE SEULE
// traduction pour les trois lectures : la liste blanche est TOUJOURS posee (meme
// vide — ce qui vaut « aucun match »), et la composition est nettoyee.
func requeteDuScope(xuid, carte string, scope domain.TacticalScope) domain.TacticalQuery {
	return domain.TacticalQuery{
		PlayerXUID:  xuid,
		MapID:       carte,
		Matchs:      domain.RestreindreAux(scope.MatchIDs),
		Coequipiers: compositionNettoyee(scope.Coequipiers),
	}
}

// compositionNettoyee rend les xuids de la composition sans blanc, sans doublon et
// sans entree vide — un xuid vide ne designe personne et ne doit ni restreindre
// l'univers ni entrer dans l'axe « escouade ». Ordre d'entree conserve : la
// composition est ce que l'utilisateur a nomme, dans son ordre.
func compositionNettoyee(xuids []string) []string {
	if len(xuids) == 0 {
		return nil
	}
	vus := make(map[string]bool, len(xuids))
	out := make([]string, 0, len(xuids))
	for _, x := range xuids {
		v := strings.TrimSpace(x)
		if v == "" || vus[v] {
			continue
		}
		vus[v] = true
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// predicatQui dit si un joueur, DANS UN MATCH DONNE, appartient a l'axe demande.
type predicatQui func(matchID, xuid string) bool

// cible rend le predicat d'appartenance a l'axe demande.
//
// Une identite VIDE (bot, environnement) n'appartient a aucun axe : elle n'a pas
// d'equipe, et la ranger quelque part serait une invention.
//
// LES TROIS AXES N'ONT PAS LA MEME NATURE, et c'est voulu :
//
//	moi        une identite ;
//	escouade   LA COMPOSITION CHOISIE (2026-09-06) — la liste que l'utilisateur a
//	           nommee, independante du match. Le perimetre garantit deja que ces
//	           joueurs etaient dans MON equipe sur chaque match retenu
//	           (TacticalQuery.Coequipiers) : re-tester l'equipe ici ferait une seconde
//	           definition de la meme regle, a un endroit ou elle ne peut plus differer ;
//	adv        l'autre equipe DU MATCH — elle change a chaque partie, donc elle se lit
//	           par match, et un joueur dont l'equipe est inconnue n'en fait pas partie.
func cible(equipes domain.EquipesParMatch, qui, moi string, coequipiers []string) predicatQui {
	if qui == domain.TacticalQuiMoi {
		return func(_ string, xuid string) bool { return xuid != "" && xuid == moi }
	}
	if qui == domain.TacticalQuiEscouade {
		compo := make(map[string]bool, len(coequipiers))
		for _, x := range coequipiers {
			compo[x] = true
		}
		return func(_ string, xuid string) bool { return xuid != "" && compo[xuid] }
	}
	return adversairesDuMatch(equipes, moi)
}

// adversairesDuMatch : l'autre equipe, lue PAR MATCH.
func adversairesDuMatch(equipes domain.EquipesParMatch, moi string) predicatQui {
	return func(matchID, xuid string) bool {
		if xuid == "" {
			return false
		}
		duMatch := equipes[matchID]
		monEquipe, jeSuisLa := duMatch[moi]
		son, ilEstLa := duMatch[xuid]
		return jeSuisLa && ilEstLa && son != monEquipe
	}
}

// campDuMatch : MON CAMP au sens de la page Escouade — mes coequipiers DU MATCH,
// moi exclu. Distinct de l'axe « escouade » depuis le 2026-09-06 : le KPI d'echange
// porte sur le camp ENTIER (decision utilisateur), la ou les rasters portent sur la
// composition choisie. Deux perimetres, deux predicats — les confondre ferait varier
// le denominateur du taux avec le contenu du selecteur de coequipiers.
func campDuMatch(equipes domain.EquipesParMatch, moi string) predicatQui {
	return func(matchID, xuid string) bool {
		if xuid == "" || xuid == moi {
			return false
		}
		duMatch := equipes[matchID]
		monEquipe, jeSuisLa := duMatch[moi]
		son, ilEstLa := duMatch[xuid]
		return jeSuisLa && ilEstLa && son == monEquipe
	}
}
