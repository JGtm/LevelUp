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
// Le porteur est `ti=13`, deja porte. CHAQUE ZONE A DEUX CANAUX `tag 4` A VALEURS D EQUIPE dans
// son bloc de proprietes (zone_states_owner_nom.go) : le PROPRIETAIRE, et un canal qui vaut
// `0xFFFFFFFF` quand personne ne capture et le camp qui pousse pendant une rampe. C EST LE
// POUSSEUR. Mesure du lot 5.6 (proprietaire N, pousseur N+1, jauge N+2 sur ces deux films) :
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
// # LE NOM, ET L ELECTION EN REPLI
//
// Le pousseur se designe PAR LE NOM du pousseur du bloc dont la jauge de la zone est la jauge
// ([zoneCapturerOf]) : c est l identite de la propriete, pas un numero de slot. Faute de nom au
// vocabulaire, il est ELU PAR LE SIGNAL (repli nomme et compte, `repli_zone_pousseur_par_election`),
// et la ou les deux repondent l election est le controle du nom :
//
//	candidat   un canal `tag 4` a valeurs d equipe, AUTRE que le proprietaire de la zone ;
//	critere    sur chaque rampe ABOUTIE de la zone, sa valeur PENDANT la rampe doit valoir ce
//	           que le proprietaire prend JUSTE APRES le sommet — c est-a-dire exactement ce que
//	           la deduction du schema 64 publiait ;
//	seuil      au moins [zoneCapturerMinAgreements] accords et AUCUN desaccord.
//
// Le seuil est celui du vote du proprietaire (`zoneOwnerMinAgreements`) et pour la meme raison :
// sur des canaux qui ne prennent que trois valeurs, UN accord est ce que le hasard produit tout
// seul. C est ce seuil qu une zone peu disputee ne passe pas — et que le nom n exige pas.
//
// # LE TEMOIN DE CHAINAGE EST OBLIGATOIRE, ET C EST MESURE
//
// Les candidats se cherchent dans la serie CHAINEE (`ManagedPropertyRead.Chained`), le meme
// filtre que la serie `desig` du tag 5 depuis le lot C-ter. Sans lui, le canal pousseur de la
// troisieme zone de `7344d24f` porte SEPT valeurs distinctes, dont des `u32` hors plage
// d equipe : la contamination d ancrage le disqualifie avant meme l election. Avec lui, il rend
// 12 accords sur 12. Le filtre ne change rien aux trois zones de `396cfc92` (memes elus).
//
// # LA DEDUCTION SURVIT EN REPLI NOMME
//
// Elle reste, sous [fallback.NomZoneCampDeCaptureDeduitDeLIssue], pour une rampe ABOUTIE que le
// canal pousseur retenu ne dit pas : aucun canal (nom inconnu et election sans elu), ou un canal
// muet pendant la rampe. La retirer ferait PERDRE des camps aujourd hui publies ; elle se retire le
// jour ou le parc rend 0 declenchement.

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

// zoneCapturerChoix est le canal POUSSEUR retenu pour une zone, et ce que le nom et l election en
// ont dit.
type zoneCapturerChoix struct {
	// serie est la serie CHAINEE du canal retenu, nil quand aucun ne l est.
	serie []zoneSample
	// nomme : le canal est designe par le nom du pousseur du bloc de la jauge.
	nomme bool
	// parElection : le canal vient de l election, faute de nom (repli compte).
	parElection bool
	// slot est le canal retenu ; eluSlot est celui que l election elit (eluOK), pour le controle
	// du nom.
	slot, eluSlot uint32
	eluOK         bool
}

// zoneCapturerOf rend le canal POUSSEUR d une zone dont `jauge` est le slot de jauge : PAR LE NOM du
// pousseur du meme bloc (zone_states_owner_nom.go), l ELECTION par le signal en repli et en
// controle (une discordance se compte, le nom est retenu).
//
// LA SERIE EST LA SERIE CHAINEE, quel que soit le chemin : la contamination d ancrage fait porter
// des `u32` hors plage d equipe aux lectures non chainees (cf. zoneSeries.ownerChained).
func zoneCapturerOf(ser zoneSeries, ramps []zoneRamp, jauge uint32, c zoneCapturerCtx) zoneCapturerChoix {
	elu, eluOK := electZoneCapturer(ser, ramps, c)
	if s, ok := zonePousseurNomme(jauge, ser.noms); ok {
		return zoneCapturerChoix{serie: ser.ownerChained[s], nomme: true, slot: s, eluSlot: elu, eluOK: eluOK}
	}
	if !eluOK {
		return zoneCapturerChoix{}
	}
	return zoneCapturerChoix{serie: ser.ownerChained[elu], parElection: true, slot: elu}
}

// electZoneCapturer ELIT le canal POUSSEUR d une zone par le signal, et rend son slot ; faux quand
// aucun candidat ne passe le critere.
//
// LE PARCOURS EST DETERMINISTE (`sortedZoneSlots`) et l egalite se tranche par le slot le plus
// petit : deux cuissons du meme film doivent elire le meme canal.
func electZoneCapturer(ser zoneSeries, ramps []zoneRamp, c zoneCapturerCtx) (uint32, bool) {
	best, found := uint32(0), false
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
		best, bestN, found = slot, accord, true
	}
	return best, found
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
// dans `[t0, tPeak]`, hors d un NEUTRE emis a la frame du sommet. Sans emission dans la fenetre, le
// canal ne dit rien de cette rampe.
//
// LA DERNIERE ET NON LA PREMIERE : une rampe peut commencer avant que le camp pousseur soit
// pose, et c est la valeur au SOMMET qui designe celui qui a mene la poussee a son terme.
//
// LE NEUTRE A LA FRAME DU SOMMET N EST PAS LA POUSSEE, C EST SA FIN : quand une capture aboutit,
// le pousseur repasse au neutre a l instant meme ou la jauge retombe, et cette emission tombe
// parfois dans la frame du dernier echantillon de la rampe. La lire ferait dire « personne ne
// pousse » d une rampe que le film montre poussee jusqu au bout.
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
		if s.t == r.tPeak && s.v == zoneNeutralOwner && found {
			continue
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

// tallyZoneCapturer porte le choix du pousseur d une zone dans la couverture et le compteur de
// replis, et rend les discordances augmentees de celle du nom contre l election, s il y en a une.
func tallyZoneCapturer(choix zoneCapturerChoix, ref int, cov *ZonesCoverage, fb *fallback.Compteur,
	disc []zoneDiscordance,
) []zoneDiscordance {
	switch {
	case choix.nomme:
		cov.CapturerNamed++
		if choix.eluOK && choix.eluSlot != choix.slot {
			cov.CapturerElectionDisagreed++
			disc = append(disc, zoneDiscordance{ref: ref, canal: zoneCanalPousseur,
				nomme: choix.slot, autre: choix.eluSlot})
		}
	case choix.parElection:
		fb.Declenche(fallback.NomZonePousseurParElection)
	}
	return disc
}
