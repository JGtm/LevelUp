package revision

// messages.go — LES MESSAGES D ECHEC D UN GATE DE REVISION, PARAMETRES PAR COUCHE.
//
// # POURQUOI CES MESSAGES SONT DU CODE, ET PAS DE LA PROSE RECOPIEE
//
// Ce qu un gate d empreinte garde vraiment n est pas l egalite de deux chaines : c est la
// QUESTION que son message pose a celui qui le voit rouge. Les deux gates d aujourd hui posent
// deux questions differentes sur des paquets qui se recouvrent :
//
//	grammaire   « la sortie de killsource peut-elle changer (backlog de redecodage) ? le
//	            contenu cuit change-t-il (SchemaVersion, recuisson) ? »
//	faits       « ces lignes doivent-elles etre redecodees ? » — et la reponse part sur SIGNAL
//	            UTILISATEUR, jamais automatiquement (decouverte D6).
//
// Centraliser l empreinte sans centraliser les messages aurait produit quatre gates au message
// generique : « empreinte differente », ce qui ne dit a personne ce qu il doit decider. Chaque
// couche fournit donc sa [Messages], et le texte commun (les deux gestes, la commande de
// regeneration) est ecrit UNE fois.

import (
	"fmt"
	"strings"
)

// Messages : de quoi ecrire les messages d echec du gate d une couche.
type Messages struct {
	// Couche : `grammar`, `facts`, ... tel qu il apparait dans les messages.
	Couche string
	// Constante : le nom Go de la constante de revision, cite dans les deux gestes
	// (`grammar.Rev`, `facts.Rev`).
	Constante string
	// Forme : la forme d une revision, pour l exemple (`grammar-AAAA-MM-JJ[.N]`).
	Forme string
	// Porte : la porte de regeneration du golden de la couche.
	Porte Porte
	// Commande : la commande complete de regeneration, telle qu on la tape.
	Commande string
	// Question : CE QU IL FAUT SE DEMANDER quand ce gate rougit. C est le champ qui porte la
	// consequence propre a la couche — pour `facts`, le backlog killsource et le signal
	// utilisateur (D6) ; pour `grammar`, les deux etages du dessous.
	Question string
}

// SourcesOntChange : l empreinte a bouge, la revision non.
func (m Messages) SourcesOntChange(revisionFigee, empreinteFigee, empreinte string, fichiers int) string {
	return m.entete("LES SOURCES DE LA COUCHE "+strings.ToUpper(m.Couche)+" ONT CHANGE") +
		fmt.Sprintf("  revision  : %s (inchangee)\n  figee     : %s\n  mesuree   : %s (%d fichiers)\n\n",
			revisionFigee, empreinteFigee, empreinte, fichiers) +
		"DEUX GESTES, ET LES DEUX SONT OBLIGATOIRES :\n\n" +
		fmt.Sprintf("  1. DECIDER, puis faire monter %s (forme %s) si la sortie de la couche "+
			"peut changer.\n     %s\n", m.Constante, m.Forme, m.Question) +
		"     Si la sortie ne peut PAS changer (commentaire, renommage interne), laisser la\n" +
		"     revision et l ecrire dans le commit : le choix doit etre explicite.\n" +
		m.geste2()
}

// RevisionSeuleAChange : la revision a bouge, l empreinte non.
//
// Cette branche n est atteignable que parce que le fichier qui PORTE la revision est exclu du
// hachage (correctif R1 / P2-3) : sans cette exclusion, une montee de revision changerait aussi
// l empreinte, et ce message serait du code mort.
func (m Messages) RevisionSeuleAChange(revisionCode, revisionFigee, empreinte string) string {
	return m.entete("LA REVISION A CHANGE SANS QUE LA COUCHE "+strings.ToUpper(m.Couche)+" BOUGE") +
		fmt.Sprintf("  revision du code : %s\n  revision figee   : %s\n  empreinte        : %s (inchangee)\n\n",
			revisionCode, revisionFigee, empreinte) +
		"Deux lectures, et aucune ne se regle en laissant le gate vert :\n\n" +
		"  - la montee vise un changement qui vit AILLEURS que dans les racines hachees. C est\n" +
		"    legitime — le gate ne hache pas l amont : regenerer, et le dire dans le commit ;\n" +
		"  - la revision a ete modifiee par megarde, ou remise a une valeur anterieure : la\n" +
		"    remettre. Une revision qui monte pour rien rouvre un backlog pour rien.\n" +
		m.geste2()
}

// GoldenPerime : les deux ont bouge ensemble — il ne reste qu a regenerer.
func (m Messages) GoldenPerime(revision, empreinte string, fichiers int) string {
	return m.entete("GOLDEN PERIME POUR LA COUCHE "+strings.ToUpper(m.Couche)) +
		fmt.Sprintf("  revision et empreinte ont bouge ensemble : %s / %s (%d fichiers)\n\n",
			revision, empreinte, fichiers) +
		m.geste2()
}

// SansEntreeDeChronique : la revision courante n a pas son entree la ou elle devrait.
func (m Messages) SansEntreeDeChronique(faute string) string {
	return m.entete("CHRONIQUE INCOMPLETE POUR LA COUCHE "+strings.ToUpper(m.Couche)) +
		"  " + faute + "\n\n" +
		"Une montee sans entree ne dit pas ce qu elle change, et la chronique s arrete a un rang\n" +
		"que la constante a depasse (constat F5 de la revue de jalon M1).\n"
}

// entete : le titre d un message, en majuscules et suivi d une ligne vide.
func (m Messages) entete(titre string) string { return titre + ".\n\n" }

// geste2 : le second geste, commun aux trois branches du gate.
func (m Messages) geste2() string {
	return fmt.Sprintf("  2. REGENERER le golden, qui fige le couple (revision, empreinte) :\n\n"+
		"       %s\n\n"+
		"Sans le geste 2 le gate reste rouge ; sans le geste 1 la couche sert un decodage que\n"+
		"plus rien ne date.\n", m.Commande)
}
