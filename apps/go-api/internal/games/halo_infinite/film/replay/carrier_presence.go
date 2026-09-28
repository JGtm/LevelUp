package replay

// carrier_presence.go — LA PORTE DE PRESENCE DES PORTEURS, partagee par le CRANE (`skull_carries.go`)
// et la BOMBE (`bomb_carries.go`) : une seule copie de la regle.
//
// Extraite de `skull_carries.go` au lot J8.7-bis (2026-09-28), sans changement de regle : le fichier
// touchait le plafond de 500 lignes, et la porte n est pas au crane — la bombe la traverse aussi.

// presenceSpan est une fenetre [f0,f1] fermee sur l'axe de frames publie.
type presenceSpan struct{ f0, f1 int }

// carrierPresence est l'index de PRESENCE des porteurs sur l'axe de frames publie — et, ce qui
// compte autant, ce qu'il NE SAIT PAS.
//
// POURQUOI DEUX CHAMPS ET PAS UNE SEULE MAP. Le gate de presence (2026-08-30) a d'abord indexe
// les seules vies NOMMEES, et lu « aucune vie nommee de X ne couvre l'intervalle » comme « X est
// ABSENT de la carte ». C'est un faux syllogisme : le pont d'identite laisse des vies ANONYMES
// (18 slots sur 160 sur `d9781168` — 142 portent au moins une vie nommee, ces 18-la aucune), et
// une vie anonyme est une PRESENCE SANS IDENTITE, pas une absence. Mesure du 2026-09-06, chaine
// independante : en Oddball le score EST le temps de portage, et la feuille de match donne
// 191 s / 196 s par equipe sur `d9781168` ; le gate publiait 60,1 s / 147,4 s. Il ecartait deux
// tiers du temps de portage d'une equipe, en croyant ecarter des fantomes. Le champ `unnamed`
// est ce qui rend l'ignorance VISIBLE au gate.
type carrierPresence struct {
	// named : les vies publiees et NOMMEES, groupees par xuid.
	named map[string][]presenceSpan
	// unnamed : les vies publiees que le pont n'a identifiees NI par xuid NI comme bot.
	// Quelqu'un est la, on ne sait pas qui — donc on ne peut RIEN affirmer sur l'absence d'un
	// joueur a cet instant.
	unnamed []presenceSpan
	// sansVieNommee compte les portages que la porte laisse passer faute de TOUTE vie nommee du
	// porteur — le repli du calque qui l a construite (`repli_crane_porteur_sans_vie_nommee`,
	// `repli_bombe_porteur_sans_vie_nommee`) : c est l APPELANT qui le nomme en le versant
	// ([carrierPresence.porteursSansVieNommee]), la porte n en a qu une copie. Partage par pointeur
	// entre les copies de l index ; nil (index fabrique a la main en test) ne compte rien.
	sansVieNommee *int
}

// carrierPresenceOf indexe les vies bipedes PUBLIEES (`doc.Tracks`) : les nommees par xuid, les
// non identifiees a part. Meme axe de frames que les portages (les deux passent par le meme
// `origin`/`step`), donc directement comparables. Sert le crane ET la bombe.
//
// UNE VIE DE BOT N'ENTRE NULLE PART, et c'est voulu. Elle n'a pas de xuid (seul cas ou une vie
// est nommee sans en avoir un, cf. [Track.Bot]), donc elle ne peut pas porter un portage ; mais
// elle est IDENTIFIEE, donc elle ne cree aucun doute sur ou se trouve un joueur. La ranger avec
// les anonymes ferait abstenir le gate sur les 20 films a bots du parc sans raison.
//
// UNE IDENTITE DEDUITE AJOUTE UNE PRESENCE, ELLE N'EN RETIRE JAMAIS UNE (2026-09-07). Depuis le
// nommage final (unnamed_lives.go), plus aucune vie n'est publiee sans identite : `unnamed`
// serait donc VIDE, et l'abstention n° 2 du gate — celle qui coute le plus, cf. son en-tete —
// mourrait avec elle. Or la voie qui a nomme ces vies-la est une DEDUCTION (l'occupation du slot
// dans le temps), pas une lecture : elle etablit qu'un joueur etait PROBABLEMENT la, jamais
// qu'un AUTRE n'y etait pas. Les faire compter comme une preuve d'absence transformerait une
// deduction en refutation — et la mesure le dit : sur `d9781168`, l'Oddball dont le score EST le
// temps de portage, cela coutait un portage et 101 frames, EN S'ELOIGNANT de la feuille de match
// (387 s reelles ; 331,3 s publiees contre 321,2 s).
//
// Une vie dont l'identite est deduite entre donc DANS LES DEUX : sous son xuid (c'est sa
// presence a lui) ET parmi les vies qui ne prouvent l'absence de personne.
func carrierPresenceOf(tracks []Track, deduced map[int]bool) carrierPresence {
	p := carrierPresence{named: map[string][]presenceSpan{}, sansVieNommee: new(int)}
	for i, t := range tracks {
		span := presenceSpan{t.StartFrame, t.EndFrame}
		switch {
		case t.XUID != "":
			p.named[t.XUID] = append(p.named[t.XUID], span)
			if deduced[i] {
				p.unnamed = append(p.unnamed, span)
			}
		case t.Bot == "":
			p.unnamed = append(p.unnamed, span)
		}
	}
	return p
}

// gate applique la regle de PRESENCE a un portage [f0,f1] attribue a `xuid`. Il rend les bornes
// a publier et `false` quand le portage est un FANTOME (a ecarter, `CarrierAbsent`).
//
// TROIS CAS D'ABSTENTION, tous ramenes au meme principe : ON NE REJETTE PAS L'INCONNU.
//  1. `xuid` n'a AUCUNE vie nommee (jamais ponte, ou `named` vide en test) : rien a opposer.
//  2. Une vie ANONYME recouvre l'intervalle : la presence y est INCONNUE, pas nulle. Ni rejet ni
//     rognage — rogner reviendrait a affirmer que le porteur n'etait pas la ou une vie sans nom
//     dit que quelqu'un l'etait.
//  3. Une vie nommee de `xuid` recouvre l'intervalle : le portage est publie, ROGNE a la vie qui
//     le recouvre le plus (le crane n'est porte que tant que son porteur est present).
//
// Le rejet ne subsiste donc que quand les pistes publiees rendent COMPTE de tout l'intervalle et
// que le porteur n'y est pas — le seul cas ou « absent » est une mesure et non une ignorance.
func (p carrierPresence) gate(xuid string, f0, f1 int) (int, int, bool) {
	spans := p.named[xuid]
	if len(spans) == 0 {
		if p.sansVieNommee != nil {
			*p.sansVieNommee++
		}
		return f0, f1, true
	}
	// L'IGNORANCE PASSE AVANT LE ROGNAGE, et c'est la moitie la plus couteuse du correctif : sur
	// `d9781168`, le rejet coutait 32,6 s de portage et le rognage 91,2 s. Rogner un portage a une
	// vie nommee alors qu'une vie SANS NOM couvre le reste, c'est affirmer une absence que rien
	// n'etablit.
	if _, unknown := unionOverlap(p.unnamed, f0, f1); unknown {
		return f0, f1, true
	}
	if span, ok := unionOverlap(spans, f0, f1); ok {
		if f0 < span.f0 {
			f0 = span.f0
		}
		if f1 > span.f1 {
			f1 = span.f1
		}
		return f0, f1, true
	}
	return f0, f1, false
}

// unionOverlap rend l'UNION des fenetres de presence que [f0,f1] recouvre, et si au moins une
// le recouvre.
//
// POURQUOI L'UNION, ET PAS LA FENETRE QUI RECOUVRE LE PLUS (residu instruit le 2026-09-06). Le
// rognage a `bestOverlap` etait le MEME defaut que `windowFor` avant le schema 45 : un portage
// qu'un trou de replication de plus de `lifeGapUS` coupe en deux vies NOMMEES du meme porteur
// etait tronque a la moitie la plus longue, et la part couverte par l'autre vie — dont, selon
// le cote, l'instant de PRISE — partait a la trappe. `spanFor` (equipment_episodes.go) a tranche
// la question pour les episodes d'equipement au schema 45 ; c'est la meme mesure, la meme cause
// et la meme reponse. Le correctif du schema 43 n'avait traite que le REJET (« l'ignorance passe
// avant le rognage »), jamais le rognage lui-meme.
//
// L'UNION NE DEBORDE JAMAIS L'INTERVALLE MESURE : les bornes rendues sont ensuite CLAMPEES sur
// [f0,f1] par l'appelant, et une fenetre que l'intervalle ne recouvre pas n'entre pas dans
// l'union. Un portage qu'AUCUNE vie nommee ne recouvre reste ecarte : la regle de rejet ne
// bouge pas.
func unionOverlap(spans []presenceSpan, f0, f1 int) (presenceSpan, bool) {
	out, found := presenceSpan{}, false
	for _, s := range spans {
		lo, hi := f0, f1
		if s.f0 > lo {
			lo = s.f0
		}
		if s.f1 < hi {
			hi = s.f1
		}
		if hi < lo {
			continue // cette vie ne recouvre pas l'intervalle
		}
		switch {
		case !found:
			out, found = s, true
		default:
			if s.f0 < out.f0 {
				out.f0 = s.f0
			}
			if s.f1 > out.f1 {
				out.f1 = s.f1
			}
		}
	}
	return out, found
}

// porteursSansVieNommee rend le nombre de portages que [carrierPresence.gate] a laisses passer faute
// de toute vie nommee du porteur (abstention n° 1) — le compte que l appelant verse sous le nom du
// repli de SON calque (lot J8.7-bis, 2026-09-28).
func (p carrierPresence) porteursSansVieNommee() int {
	if p.sansVieNommee == nil {
		return 0
	}
	return *p.sansVieNommee
}
