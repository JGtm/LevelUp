package coordination

// riposte.go — APRES UNE MORT, LE TUEUR EST-IL ABATTU ? APRES UN FRAG, L'AUTEUR L'EST-IL ?
//
// DEUX QUESTIONS DU MEME JOURNAL, UNE SEULE FENETRE. Le journal des kills (`match_kill_events`) dit
// qui a tue qui et quand ; ce fichier ne fait que le parcourir, sans base ni fichier.
//
//	TueurAbattu     une mort : un COEQUIPIER de la victime tue son tueur dans la fenetre.
//	AbattuEnRetour  un frag  : l'auteur du frag est tue dans la fenetre, PAR N'IMPORTE QUI.
//
// LA FENETRE [0, FenetreRiposteMs] A SES BORNES INCLUSES : un coup au meme instant que l'evenement
// (delai 0, un echange simultane) est une riposte, un coup 3 000 ms apres aussi, 3 001 ms non.
//
// L'IDENTITE D'UN JOUEUR EST SON XUID, A DEFAUT SON GAMERTAG. Un bot n'a pas de xuid : il se designe
// par son nom, et un bot ne se confond jamais avec un humain (un xuid d'un cote, aucun de l'autre).
// Un vengeur doit avoir un xuid, parce que son appartenance a l'equipe de la victime vient de la
// base par xuid : un bot vengeur n'est donc pas attribue (aucune equipe pour lui), jamais devine.
//
// PLUSIEURS VENGEURS : le plus tot, puis le gamertag par ordre alphabetique — le meme resultat a
// chaque lecture.

import "levelup/go-api/internal/domain"

// FenetreRiposteMs est la fenetre d'une riposte, en millisecondes, bornes incluses. SOURCE UNIQUE :
// le web recoit le delai calcule ici, jamais la fenetre (garde-rail
// `archlint/no_second_riposte_window_test.go`).
const FenetreRiposteMs int64 = 3000

// memeJoueur dit si deux identites designent le meme joueur.
func memeJoueur(xuidA, nomA, xuidB, nomB string) bool {
	switch {
	case xuidA != "" && xuidB != "":
		return xuidA == xuidB
	case xuidA != "" || xuidB != "":
		return false
	default:
		return nomA != "" && nomA == nomB
	}
}

// TueurConnu dit que le journal identifie le tueur de la mort (xuid ou gamertag) : la mort entre
// alors au denominateur de « tueur abattu N sur M ».
func TueurConnu(mort domain.KillDuJournal) bool {
	return mort.KillerXUID != "" || mort.KillerGamertag != ""
}

// dansLaFenetre dit si `t` suit `depuis` de 0 a FenetreRiposteMs, bornes incluses.
func dansLaFenetre(t, depuis int64) bool {
	d := t - depuis
	return d >= 0 && d <= FenetreRiposteMs
}

// plusTot dit que `a` est a preferer a `b` : le plus tot, puis le gamertag.
func plusTot(a, b domain.KillDuJournal) bool {
	if a.TimeMs != b.TimeMs {
		return a.TimeMs < b.TimeMs
	}
	return a.KillerGamertag < b.KillerGamertag
}

// TueurAbattu cherche, dans le journal du match, le coequipier de la victime qui abat le tueur de
// `mort` dans la fenetre. `coequipier` dit si un xuid est un coequipier de la victime (equipe de la
// base). Faux quand le tueur est inconnu ou que personne ne l'abat.
func TueurAbattu(journal []domain.KillDuJournal, mort domain.KillDuJournal,
	coequipier func(xuid string) bool,
) (domain.TacticalRiposte, bool) {
	if !TueurConnu(mort) {
		return domain.TacticalRiposte{}, false
	}
	var meilleur *domain.KillDuJournal
	for i := range journal {
		k := &journal[i]
		if k.MatchID != mort.MatchID || !dansLaFenetre(k.TimeMs, mort.TimeMs) {
			continue
		}
		if !memeJoueur(k.VictimXUID, k.VictimGamertag, mort.KillerXUID, mort.KillerGamertag) {
			continue
		}
		if k.KillerXUID == "" || k.KillerXUID == mort.VictimXUID || !coequipier(k.KillerXUID) {
			continue
		}
		if meilleur == nil || plusTot(*k, *meilleur) {
			meilleur = k
		}
	}
	if meilleur == nil {
		return domain.TacticalRiposte{}, false
	}
	return riposteDe(domain.TacticalRiposteTueurAbattu, *meilleur, mort.TimeMs), true
}

// AbattuEnRetour cherche, dans le journal du match, le kill qui abat `joueurXUID` dans la fenetre
// qui suit son frag a `instantMs`, quel qu'en soit l'auteur (un bot, l'environnement identifie par
// son nom). Faux sans xuid ou quand personne ne l'abat.
func AbattuEnRetour(journal []domain.KillDuJournal, matchID, joueurXUID string, instantMs int64,
) (domain.TacticalRiposte, bool) {
	if joueurXUID == "" {
		return domain.TacticalRiposte{}, false
	}
	var meilleur *domain.KillDuJournal
	for i := range journal {
		k := &journal[i]
		if k.MatchID != matchID || k.VictimXUID != joueurXUID || !dansLaFenetre(k.TimeMs, instantMs) {
			continue
		}
		if meilleur == nil || plusTot(*k, *meilleur) {
			meilleur = k
		}
	}
	if meilleur == nil {
		return domain.TacticalRiposte{}, false
	}
	return riposteDe(domain.TacticalRiposteAbattuEnRetour, *meilleur, instantMs), true
}

// riposteDe traduit le kill retenu : l'auteur du coup, sa position quand elle est mesuree, le delai.
func riposteDe(typ string, k domain.KillDuJournal, depuis int64) domain.TacticalRiposte {
	r := domain.TacticalRiposte{
		Type: typ, DelaiMs: k.TimeMs - depuis, ParGamertag: k.KillerGamertag, ParXUID: k.KillerXUID,
	}
	if k.KillerX != nil && k.KillerY != nil {
		r.Point = &domain.TacticalPoint{X: *k.KillerX, Y: *k.KillerY}
	}
	return r
}
