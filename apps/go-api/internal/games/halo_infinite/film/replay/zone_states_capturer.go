package replay

// zone_states_capturer.go — LE CAMP QUI POUSSE UNE RAMPE, LU DANS LE FILM (lot 5.6).
//
// # CE QUE CE FICHIER REMPLACE
//
// Le schema 64 (lot 5.2a.3) publiait `gaugeRamps[].capturingTeam` par DEDUCTION : le
// proprietaire a l issue d une rampe qui ABOUTIT. Une rampe qui AVORTE n avait donc aucun camp
// — le canal de propriete y nomme encore le defenseur — et son remplissage restait neutre a
// l ecran. Ce fichier lit le camp au lieu de le deduire.
//
// # CE QUE LE FILM PORTE, ET OU (mesure du lot 5.6, deux films, 0 desaccord)
//
// L archetype `zones` `ti=23` N EST PAS le porteur, et deux lectures le disent : son ecrivain
// (`FUN_142ed6cec` -> `FUN_141454340`) ecrit un nom, une position et un handle de participant —
// la selection de zone de REAPPARITION —, et le film ne l instancie meme pas (0 slot recense aux
// images-cles de deux films a zones, quand la meme marche rend 26 slots pour `ti=13`).
//
// Le porteur est `ti=13`, deja porte. CHAQUE ZONE A DEUX CANAUX `tag 4` A VALEURS D EQUIPE, et
// le depot en nommait deja un « canal neutre » (`zone_states_owner.go`, blocs de pas 5 :
// « proprietaire, canal neutre, jauge ») parce qu il vaut `0xFFFFFFFF` quand personne ne
// capture. C EST LE POUSSEUR. Triplet mesure : proprietaire N, POUSSEUR N+1, jauge N+2.
//
//	| film     | jauge | pousseur | proprietaire | abouties | accord | desaccord | avortees nommees |
//	|---|---:|---:|---:|---:|---:|---:|---:|
//	| 396cfc92 | 1603  | 1602     | 1601         |  9 |  9 | 0 | 3 / 3 |
//	| 396cfc92 | 1608  | 1607     | 1606         | 12 | 12 | 0 | 6 / 6 |
//	| 396cfc92 | 1613  | 1612     | 1611         |  9 |  9 | 0 | 0 / 0 |
//	| 7344d24f | 1532  | 1531     | 1530         | 16 | 16 | 0 | 2 / 2 |
//	| 7344d24f | 1537  | 1536     | 1535         | 12 | 12 | 0 | 5 / 5 |
//	| 7344d24f | 1542  | 1541     | 1540         | 11 | 11 | 0 | 3 / 3 |
//
// # L ELECTION, ET POURQUOI PAS L ARITHMETIQUE DE SLOT
//
// Le triplet est REGULIER sur les deux films mesures, mais un numero de slot est un ORDRE
// D ALLOCATION du moteur au chargement (c est deja ce que `zoneLetterRanks` en dit) : figer
// « pousseur = proprietaire + 1 » figerait une coincidence. Le pousseur est donc ELU PAR LE
// SIGNAL, comme le proprietaire l est deja :
//
//	candidat   un canal `tag 4` a valeurs d equipe, AUTRE que le proprietaire de la zone ;
//	critere    sur chaque rampe ABOUTIE de la zone, sa valeur PENDANT la rampe doit valoir ce
//	           que le proprietaire prend JUSTE APRES le sommet — c est-a-dire exactement ce que
//	           la deduction du schema 64 publiait ;
//	seuil      au moins [zoneCapturerMinAgreements] accords et AUCUN desaccord.
//
// Le seuil est celui du proprietaire (`zoneOwnerMinAgreements`) et pour la meme raison : sur des
// canaux qui ne prennent que trois valeurs, UN accord est ce que le hasard produit tout seul.
// Le zero desaccord, lui, est plus dur que pour le proprietaire — et il le peut : la mesure rend
// 0 desaccord sur 69 rampes abouties de deux films.
//
// # LE TEMOIN DE CHAINAGE EST OBLIGATOIRE, ET C EST MESURE
//
// Les candidats se cherchent dans la serie CHAINEE (`ManagedPropertyRead.Chained`), le meme
// filtre que la serie `desig` du tag 5 depuis le lot C-ter. Sans lui, le canal pousseur de la
// troisieme zone de `7344d24f` porte SEPT valeurs distinctes, dont des `u32` hors plage
// d equipe : la contamination d ancrage le disqualifie avant meme l election. Avec lui, il rend
// 12 accords sur 12. Le filtre ne change rien aux trois zones de `396cfc92` (memes elus).
//
// # LA DEDUCTION SURVIT EN REPLI NOMME, ET C EST UNE MESURE QUI L IMPOSE
//
// Le brief du lot demandait de SUPPRIMER la deduction. Elle reste, sous
// [fallback.NomZoneCampDeCaptureDeduitDeLIssue], parce qu une zone peut n avoir aucun canal
// elu — la quatrieme jauge de `396cfc92` (slot 1622) n a qu une rampe aboutie, sous le seuil —
// et que retirer la deduction ferait alors PERDRE des camps aujourd hui publies. Un repli qui
// fait perdre est un repli qu on garde, nomme et compte (D14). Il se retire le jour ou le parc
// rend 0 declenchement.

import "levelup/go-api/internal/games/halo_infinite/film/internal/facts/fallback"

const (
	// zoneCapturerMinAgreements est le nombre MINIMAL de rampes abouties concordantes qu un
	// canal doit porter pour etre elu POUSSEUR d une zone. Meme valeur et meme raison que
	// `zoneOwnerMinAgreements` : un accord unique est ce que le hasard produit sur un canal a
	// trois valeurs.
	zoneCapturerMinAgreements = 2
	// zoneCampMaxTeamID borne la valeur qu un canal peut porter pour etre reconnu comme un
	// canal d EQUIPE. Halo Infinite compte au plus huit camps ; au-dela, la valeur est un
	// identifiant de chaine ou un handle, pas une equipe. C est la garde qui empeche de prendre
	// un `tag 4` quelconque pour un camp (mesure : sur `396cfc92`, 4 des 11 canaux `tag 4`
	// portent des `u32` de l ordre de 5 x 10^8).
	zoneCampMaxTeamID = 7
)

// zoneCampLike dit si une serie ne porte que des valeurs de CAMP : un identifiant d equipe court
// ou le neutre. Une serie vide n est pas un canal de camp.
func zoneCampLike(ss []zoneSample) bool {
	if len(ss) == 0 {
		return false
	}
	for _, s := range ss {
		if s.v > zoneCampMaxTeamID && s.v != zoneNeutralOwner {
			return false
		}
	}
	return true
}

// zoneCapturerCtx porte ce qu une election a besoin de savoir (regle des 5 parametres).
type zoneCapturerCtx struct {
	// owner est la serie du canal de PROPRIETE de la zone : la REFERENCE de l election.
	owner []zoneSample
	// ownerSlot est le slot de cette reference — un candidat ne peut pas etre sa propre
	// reference.
	ownerSlot uint32
	// win est la fenetre d appariement en frames, celle du reste du volet.
	win int
}

// electZoneCapturer rend la serie du canal POUSSEUR d une zone, ou nil quand aucun candidat ne
// passe le critere.
//
// LE PARCOURS EST DETERMINISTE (`sortedZoneSlots`) et l egalite se tranche par le slot le plus
// petit : deux cuissons du meme film doivent elire le meme canal.
func electZoneCapturer(ser zoneSeries, ramps []zoneRamp, c zoneCapturerCtx) []zoneSample {
	var best []zoneSample
	bestN := 0
	for _, slot := range sortedZoneSlots(ser.ownerChained) {
		if slot == c.ownerSlot {
			continue
		}
		ss := ser.ownerChained[slot]
		if !zoneCampLike(ss) {
			continue
		}
		accord, desaccord := zoneCapturerScore(ss, ramps, c)
		if desaccord > 0 || accord < zoneCapturerMinAgreements || accord <= bestN {
			continue
		}
		best, bestN = ss, accord
	}
	return best
}

// zoneCapturerScore compte les accords et les desaccords d un candidat sur les rampes ABOUTIES.
func zoneCapturerScore(ss []zoneSample, ramps []zoneRamp, c zoneCapturerCtx) (int, int) {
	accord, desaccord := 0, 0
	for _, r := range ramps {
		if gaugeProgressOf(r.top) < zoneGaugeRampComplete {
			continue
		}
		v, ok := zoneValueDuringRamp(ss, r)
		if !ok {
			continue
		}
		attendu, connu := zoneValueAfter(c.owner, r.tPeak, c.win)
		if !connu {
			continue
		}
		if v == attendu {
			accord++
			continue
		}
		desaccord++
	}
	return accord, desaccord
}

// zoneValueDuringRamp rend la valeur qu un canal porte PENDANT la rampe : la DERNIERE emission
// dans `[t0, tPeak]`. Sans emission dans la fenetre, le canal ne dit rien de cette rampe.
//
// LA DERNIERE ET NON LA PREMIERE : une rampe peut commencer avant que le camp pousseur soit
// pose, et c est la valeur au SOMMET qui designe celui qui a mene la poussee a son terme.
func zoneValueDuringRamp(ss []zoneSample, r zoneRamp) (uint64, bool) {
	var v uint64
	found := false
	for _, s := range ss {
		if s.t < r.t0 {
			continue
		}
		if s.t > r.tPeak {
			break
		}
		v, found = s.v, true
	}
	return v, found
}

// zoneRampCapturerRead rend le camp LU sur le canal pousseur pour cette rampe, ou nil quand le
// canal se tait ou nomme le neutre (« personne ne capture »).
func zoneRampCapturerRead(capt []zoneSample, r zoneRamp, teams map[uint64]bool) (*int, bool) {
	if len(capt) == 0 {
		return nil, false
	}
	v, ok := zoneValueDuringRamp(capt, r)
	if !ok {
		return nil, false
	}
	team, known := zoneOwnerTeam(v, teams)
	if !known {
		return nil, false
	}
	// UN CAMP LU, MEME NUL, EST UNE REPONSE : `zoneOwnerTeam` rend (nil, true) sur le neutre,
	// et cela veut dire « le film dit que personne ne pousse ». Le repli n a alors rien a
	// rattraper — le deduire ferait publier un camp que le film contredit.
	return team, true
}

// zoneRampCapturerDeduit est LE REPLI : le camp de l ISSUE, comme au schema 64. Il ne repond que
// sur une rampe ABOUTIE — sur une rampe avortee le canal de propriete nomme le defenseur.
func zoneRampCapturerDeduit(r zoneRamp, owner []zoneSample, teams map[uint64]bool, win int,
	fb *fallback.Compteur,
) *int {
	if gaugeProgressOf(r.top) < zoneGaugeRampComplete {
		return nil
	}
	v, ok := zoneValueAfter(owner, r.tPeak, win)
	if !ok {
		return nil
	}
	team, known := zoneOwnerTeam(v, teams)
	if !known || team == nil {
		return nil
	}
	fb.Declenche(fallback.NomZoneCampDeCaptureDeduitDeLIssue)
	return team
}
