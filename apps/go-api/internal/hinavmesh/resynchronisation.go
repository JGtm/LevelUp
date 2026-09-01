package hinavmesh

// resynchronisation.go — FRANCHIR UNE ENTREE TBDY QU'ON NE SAIT PAS LIRE.
//
// LE PROBLEME, mesure le 2026-08-31 sur Absolution. Sa section TBDY se lit parfaitement jusqu'a
// l'entree 67 incluse, puis bute sur l'entree du type `hkPropertyId` : elle y lit « 196 609
// membres ». Ce type n'a evidemment pas cinquante membres — l'entree est encodee d'une facon que
// ce decodeur ne connait pas, et le flux glisse a partir de la.
//
// CE QUE LA MESURE ETABLIT, et qui rend la reprise sure plutot que devinee : en sautant EXACTEMENT
// 13 octets — la taille de cette entree — le reste de la section se lit ENTIEREMENT, jusqu'au
// dernier octet. Ce n'est pas une tolerance vague : c'est une longueur unique et verifiable, et
// l'entree suivante (`45 00 2b` : type 69, parent 0, drapeaux 0x2b) tombe pile.
//
// LA REGLE ADOPTEE. Quand une entree est illisible, on cherche le PLUS COURT saut qui permet de
// lire tout le reste de la section. S'il existe, le type concerne est declare OPAQUE — aucun
// membre, taille inconnue — et la lecture reprend. S'il n'en existe pas, l'erreur remonte comme
// avant : on ne devine jamais.
//
// POURQUOI UN TYPE OPAQUE EST ACCEPTABLE ICI, et ou est la limite. La table des types ne sert qu'a
// interpreter les objets de la section DATA. Un type dont on ignore les membres devient
// inutilisable — mais SEULEMENT lui. Le maillage de navigation se lit par `hkaiNavMesh` et ses
// tableaux de faces, d'aretes et de sommets, tous decodes normalement. Si un jour un type opaque
// se trouve sur le chemin de la geometrie, la lecture echouera plus loin, franchement, avec le nom
// du membre manquant — jamais en silence.
//
// GARDE-FOU : `Recuperations` compte les entrees franchies. Un fichier sain en a ZERO, et le
// temoin l'exige d'Isolation ; Absolution en a exactement une. Un decodeur qui se mettrait a en
// recuperer beaucoup ne serait plus un decodeur, il serait devenu un devineur — le compteur est la
// pour que ça se voie.

import "fmt"

// sautMaxResynchronisation : longueur maximale, en octets, d'une entree qu'on accepte de franchir.
// Les entrees TBDY observees pesent de 3 a une trentaine d'octets ; au-dela, ce n'est plus une
// entree qu'on saute mais un morceau de section, et l'echec est preferable.
const sautMaxResynchronisation = 64

// resynchronise cherche le plus court saut, depuis `debut`, apres lequel tout le reste de la
// section se lit. Rend la position de reprise et vrai si elle existe.
func resynchronise(corps []byte, debut int, types tableTypes, nomsChamps []string) (int, bool) {
	for saut := 1; saut <= sautMaxResynchronisation && debut+saut < len(corps); saut++ {
		if lisibleJusquAuBout(corps, debut+saut, types, nomsChamps) {
			return debut + saut, true
		}
	}
	return 0, false
}

// lisibleJusquAuBout dit si TBDY se lit sans erreur depuis cet offset jusqu'a la fin de section.
// Il ne RETIENT rien : c'est une lecture d'essai, sur une table de types jetable.
func lisibleJusquAuBout(corps []byte, depuis int, types tableTypes, nomsChamps []string) bool {
	essai := make(tableTypes, len(types))
	l := &lecteurEmpaquete{buf: corps, pos: depuis}
	for l.err == nil && !l.fini() {
		idx := l.entier()
		if l.err != nil || idx <= 0 || idx >= len(essai) {
			return false
		}
		if err := lireUnCorps(l, &essai[idx], nomsChamps); err != nil {
			return false
		}
	}
	return l.err == nil
}

// erreurEntreeIllisible enveloppe l'echec d'une entree pour que le message dise ce qui a ete tente.
func erreurEntreeIllisible(idx int, nom string, offset int, cause error) error {
	return fmt.Errorf("hinavmesh: TBDY, type %d (%s) a l'offset %d, et aucun saut de 1 a %d octets "+
		"ne resynchronise la section: %w", idx, nom, offset, sautMaxResynchronisation, cause)
}
