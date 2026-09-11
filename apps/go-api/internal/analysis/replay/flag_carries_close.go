package replay

// flag_carries_close.go — LE PLUS PETIT FERMOIR GAGNE, ET IL EFFACE CELUI QU'IL REMPLACE
// (revue 6.R, constat C1, 2026-09-11).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Cinq chaines ferment un portage, et elles s'appliquent EN SUITE : le passage de main en main,
// la rentree du drapeau chez lui, la chute du porteur creditee, le lacher volontaire. Chacune
// ecrivait ses propres champs dans `flagCarryRaw`, et AUCUNE ne defaisait ce que la precedente
// avait pose. `homed` en particulier n'avait qu'une seule ecriture — celle de
// `closeByHomecoming` —, si bien qu'un fermoir plus PRECOCE ramenait `t1` en arriere sans jamais
// dementir la rentree.
//
// LA SEQUENCE, ORDINAIRE : prise en t0, LACHER date en D par la vie libre de l'objet, drapeau
// laisse au sol, puis RENTREE seule en H > D. La rentree fermait d'abord (`t1 = H`, `homed`), le
// lacher fermait ensuite (`t1 = D`) et `homed` restait vrai. Quatre consequences, toutes fausses :
//
//	l'etat publie          `home` AU SOCLE des `frame(D)+1`, alors que l'objet gisait au sol
//	                       jusqu'a H (`applyFlagLifeEvent`, `endsHome`) ;
//	l'etat du sol          `flagGround.poser` retirait le drapeau du sol ET du jeu, faussant
//	                       l'attribution des prises suivantes ;
//	la position du lacher  `repositionFlagDrops` sautait le repositionnement ;
//	la couverture          le MEME portage se comptait dans `closedByHome` ET `closedByObject`.
//
// # LE PRINCIPE, UNE SEULE FOIS
//
// [flagCloseAt] est le SEUL endroit qui ferme un portage apres le bornage. Il n'accepte qu'un
// instant STRICTEMENT interieur a `]t0, t1[` — donc toujours plus petit que la borne en place,
// ce qui est le principe « le plus petit gagne » rendu structurel plutot que compare — et il
// REECRIT TOUT l'etat de fin : `captured` et `homed` valent ce que dit LE fermoir en vigueur, pas
// la memoire d'un fermoir perime.
//
// LES COMPTEURS SE DERIVENT, ILS NE S'INCREMENTENT PLUS. Chaque portage porte le fermoir qui le
// ferme ([flagCarryRaw.closedBy]) ; `tallyFlagCarries` compte les portages PAR fermoir en
// vigueur. Un portage ne peut donc plus peupler deux `closedBy*` : ce n'est pas une precaution de
// comptage, c'est une propriete de la representation. [FlagCarriesCoverage.Balanced] le verifie.

// flagCloser nomme le fait qui ferme un portage. L'ordre ne porte aucune priorite : la priorite
// est celle du TEMPS, et [flagCloseAt] la fait respecter a lui seul.
type flagCloser uint8

const (
	// flagCloserNone : rien n'a ferme le portage, sa borne est la fin du rejeu et `closed` est
	// faux. C'est le seul fermoir qui publie [FlagStateCarriedOpen].
	flagCloserNone flagCloser = iota
	// flagCloserBound : capture, mort du porteur, ou reprise du meme slot — les trois faits que
	// `boundFlagCarries` compare lui-meme, en une passe, avant toute chaine de correction.
	flagCloserBound
	// flagCloserHandoff : prise d'un AUTRE joueur du MEME drapeau (flag_carries_handoff.go).
	flagCloserHandoff
	// flagCloserReturn : retour CREDITE a un joueur (flag_carries_home.go).
	flagCloserReturn
	// flagCloserHome : rentree de l'OBJET a son socle (flag_carries_home.go).
	flagCloserHome
	// flagCloserCarrierKill : `flag_carriers_killed` sur un unique portage ouvert.
	flagCloserCarrierKill
	// flagCloserObject : vie libre de l'objet aux pieds du porteur — le LACHER VOLONTAIRE
	// (flag_objects.go).
	flagCloserObject
)

// flagCloseAt ferme `r` a l'instant `at`, et SEULEMENT si ce fait est STRICTEMENT interieur au
// portage — un instant sur l'une des deux bornes, ou au-dela, ne raccourcit rien et ne doit
// jamais allonger. Il EFFACE l'etat de fin pose par le fermoir precedent : un portage qu'on
// croyait rentre chez lui redevient un portage lache des qu'un lacher plus precoce est date.
//
// IL NE REND RIEN, ET C'EST VOULU : aucun appelant n'a a savoir s'il a mordu. Le compte des
// fermetures ne se tient plus au passage (chaque chaine incrementait alors son propre compteur, y
// compris quand une autre lui reprenait le portage ensuite) : il se DERIVE du fermoir en vigueur,
// une fois toutes les chaines passees (`tallyFlagCarries`).
//
// UNE FIN CHEZ LUI EST CELLE DE SON FERMOIR, ET DE LUI SEUL : seuls le retour credite et la
// rentree de l'objet renvoient le drapeau a sa base sans capture. La capture, elle, est posee par
// `boundFlagCarries` et tout fermoir posterieur la dement — c'est la meme decision, prise sur la
// meme preuve (cf. `closeByFreeLives`).
func flagCloseAt(r *flagCarryRaw, at int64, by flagCloser) {
	if at <= r.t0 || at >= r.t1 {
		return
	}
	r.t1, r.closedBy = at, by
	r.closed = r.closed || by.ferme()
	r.captured = false
	r.homed = by == flagCloserReturn || by == flagCloserHome
}

// ferme dit si ce fait CLOT le portage, ou s'il en ramene seulement la BORNE.
//
// UN SEUL FAIT NE CLOT PAS : `flag_carriers_killed`. Il est credite au TUEUR, pas a la victime
// (cf. l'en-tete de flag_carries.go), et il n'a jamais pose `closed` — un portage que rien ne
// fermait par ailleurs restait publie [FlagStateCarriedOpen] jusqu'a la fin de l'axe malgre sa
// borne ramenee. La revue 6.R a MESURE ce que changerait de le faire clore, et la mesure dit
// non : sur `b8a44fe8`, le seul span `carried_open` du parc tombe de 25,0 s a 7,3 s pour un
// oracle de 17,2 s — l'ecart passe de +7,8 s a -9,9 s. Le fait ne date donc PAS la chute de ce
// porteur-la, et le faire clore serait une regression.
//
// LE DESACCORD ENTRE `t1` ET L'ETAT PUBLIE EST CONSIGNE, PAS TRAITE (decouverte de la revue 6.R,
// rapport `.ai/V7.5/REVUE_VAGUE6_2026-09-11.md`) : la borne bouge sans que la publication la
// suive. Le trancher demande de qualifier ce que `flag_carriers_killed` date vraiment quand
// plusieurs portages se succedent — un sujet de mesure, pas de refactorisation.
func (c flagCloser) ferme() bool { return c != flagCloserCarrierKill }

// flagClosedByCounter rend le pointeur du compteur de couverture d'un fermoir, ou nil quand ce
// fermoir n'en publie aucun. Une seule table : ajouter un fermoir publie, c'est ajouter une ligne
// ici, et le comptage par portage en vigueur suit sans autre changement.
func flagClosedByCounter(by flagCloser, cov *FlagCarriesCoverage) *int {
	switch by {
	case flagCloserHandoff:
		return &cov.ClosedByHandoff
	case flagCloserReturn:
		return &cov.ClosedByReturn
	case flagCloserHome:
		return &cov.ClosedByHome
	case flagCloserObject:
		return &cov.ClosedByObject
	}
	return nil
}
