package killsource

// paquet_identite.go — L IDENTITE DE PAQUET DECIDE L APPARIEMENT, LA FENETRE DE 2,5 s DEVIENT UN
// REPLI (lot 1.9.7 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, D13 et D14).
//
// # CE QUE LE FILM ECRIT, ET QUI N ETAIT PAS LU
//
// Apparier une mort lue au DEAD-STATE avec une ligne du KILL-FEED se decidait par une FENETRE
// TEMPORELLE de 2,5 s ([tolMS]), a cinq endroits. Cette valeur n a jamais ete derivee d une
// mesure : son commentaire disait « valeur historique du chantier, employee par TOUTES les
// mesures publiees », c est-a-dire une justification de COMPARABILITE, pas de justesse.
//
// Or les deux cotes portent une IDENTITE DE PAQUET, et le film les ecrit DANS LE MEME PAQUET :
//
//	le dead-state    [candidate] porte `(chunk, pidx)`, aussi bien depuis le scan (`scan.go`) que
//	                 depuis la marche (`walk.go`, [walkResult.candidates]) ;
//	le kill-feed     un instant du feed n a que son horodatage, mais le lot 1.9.3 lui associe un
//	                 KILL-EVENT 85, qui porte `(chunk, pidx)` ([killEventRec], `assist.go`).
//	                 L association existait ; elle etait JETEE (`pris[j] = true`, et rien
//	                 d autre). [feedEvent.paquet] la transporte desormais.
//
// # LA MESURE QUI A DECIDE (21 films entiers, 8 builds, 14 temoins du corpus gate)
//
// Sur 2 899 appariements decides par la fenetre :
//
//	2 205  identite EGALE des deux cotes        76,1 %
//	    2  identite DIFFERENTE                   0,07 % (`4f77afc1` temps 1, `fb1a1a72` temps 3)
//	  692  identite ABSENTE du cote feed        23,9 % (aucun kill-event 85 associe)
//
// Et ce que rend l appariement PAR IDENTITE SEULE : accord 2 204, DESACCORD **0**, ambigu 1,
// muet 2. La lecture ne contredit JAMAIS la fenetre — la conversion change QUI DECIDE, pas la
// valeur. Par temps : le critere fort (2 834 appariements) rend 2 176 accords, 0 desaccord,
// 0 ambigu, 1 muet ; les temps de BOT et la mort non revendiquee n ont, eux, presque aucune
// identite en face (26 sans identite sur 27) — le kill-feed est humain-seul, donc un instant qui
// ne porte pas de kill humain ne porte pas de kill-event 85 a associer. C est un NEGATIF MESURE,
// pas une lecture qui manque, et c est ce qui garde un repli a ces endroits.
//
// # L ORDRE EST LE RESULTAT (D14 b)
//
// LIRE D ABORD, SE REPLIER ENSUITE. Le repli ne se declenche que sur le silence de la lecture —
// identite absente d un cote, ou paquet designe qui ne porte aucun enregistrement satisfaisant la
// contrainte de couple — JAMAIS sur un desaccord avec elle. La contrainte de couple, elle, ne
// bouge pas d un iota : l identite remplace la FENETRE, pas le critere.

// paquetID : l identite d un paquet de replication — le couple `(chunk, paquet)` ou un
// enregistrement a ete ECRIT.
//
// `ok` FAUX N EST PAS UNE ERREUR : c est le diagnostic « le film ne rattache rien ici », le seul
// qui ouvre un repli (D14 b). Le distinguer d un `(0, 0)` legitime est tout l interet du champ.
type paquetID struct {
	chunk, pidx int
	ok          bool
}

// memeQue : cette identite est-elle celle du paquet `(chunk, pidx)` ? Faux quand elle est absente.
func (p paquetID) memeQue(chunk, pidx int) bool {
	return p.ok && p.chunk == chunk && p.pidx == pidx
}

// choisirParIdentitePuisFenetre : LES DEUX PASSES, DANS L ORDRE DE D14 (b).
//
//	LECTURE   le premier element dont l IDENTITE DE PAQUET est celle de l autre cote et qui
//	          satisfait la contrainte de couple. Le film a ecrit les deux enregistrements dans
//	          le MEME paquet : le lien est LU.
//	REPLI     le premier element dont l instant tombe dans la demi-fenetre de 2,5 s et qui
//	          satisfait la MEME contrainte de couple.
//
// Rend l indice retenu (-1 si aucune passe ne trouve) et si le REPLI a servi. Les trois
// predicats sont donnes par l appelant, qui est le seul a savoir ce qu il apparie ; cette
// fonction ne connait que l ORDRE, et c est precisement ce qu il ne faut pas recopier a cinq
// endroits.
func choisirParIdentitePuisFenetre(n int, identite, fenetre, couple func(int) bool) (int, bool) {
	for i := 0; i < n; i++ {
		if identite(i) && couple(i) {
			return i, false
		}
	}
	for i := 0; i < n; i++ {
		if fenetre(i) && couple(i) {
			return i, true
		}
	}
	return -1, false
}
