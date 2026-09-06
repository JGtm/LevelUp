package replaydiff

// empreinte_durees.go — L'AXE « SOMME DES DUREES » : un calque peut garder son NOMBRE
// d'elements et pourtant perdre de la matiere si chaque intervalle est ROGNE.
//
// # POURQUOI CET AXE EXISTE
//
// La passe specialisee existante (empreinte_axes.go) compte des ELEMENTS : « 3 portages de
// drapeau » reste 3 meme si un rognage de bordure ampute chacun d'eux d'une frame. Sur un
// Oddball mesure le 2026-09-06, un rognage de fenetre de traction de grappin a coute 91,2 s
// de duree cumulee contre 32,6 s pour l'equivalent en REJETS PURS (intervalles supprimes en
// entier) — le comparateur existant voit le second (le compte de `grappleLines/n` baisse),
// jamais le premier : un rognage sous-estime la perte d'un facteur 3 s'il n'est mesure qu'en
// nombre d'elements.
//
// # CE QUE CET AXE MESURE, ET COMMENT IL TROUVE SES CALQUES
//
// Il ne connait AUCUN nom de calque a l'avance — meme doctrine que le reste du paquet
// (cf. l'en-tete de empreinte_axes.go, `axesParCle`/`axeDe`) : il descend recursivement dans
// TOUT le document et, pour CHAQUE OBJET qui porte a la fois un `t0` et un `t1` NUMERIQUES
// (`t1 >= t0`), calcule sa duree — T1 INCLUS, la convention ecrite noir sur blanc sur chacun
// des spans du document (`FlagSpan`, `GrappleLine`, `ZoneSpan`, `EquipmentEpisode`...) — et
// l'ajoute a la somme du calque. Un calque neuf a intervalles [t0,t1] entre donc dans ce
// rapport SANS qu'on ait besoin de l'y inscrire, exactement comme la passe generique.
//
// # LA VENTILATION « PAR JOUEUR » SUIT LA CLE QUE LA SOURCE UTILISE DEJA, JAMAIS UNE JOINTURE
//
// Le document identifie l'occupant d'un span de deux facons distinctes, et cet axe respecte
// laquelle sans en inventer une troisieme :
//
//   - certains spans portent un `xuid` DIRECT (FlagSpan, SkullCarry, BombCarry, VipPeriod,
//     VehicleRide) : la ventilation est PAR XUID, la cle stable du joueur ;
//   - d'autres ne portent qu'un `slot` (GrappleLine, EquipmentEpisode, EquipmentPlacement,
//     VehicleTrack...) : la ventilation est PAR SLOT. Un slot designe une VIE (le slot migre
//     aux reapparitions, cf. Track.Slot) pour les calques qui le documentent ainsi, ou
//     l'entite du calque elle-meme (VehicleTrack.Slot identifie un VEHICULE) — jamais un
//     joueur au sens strict. La mesure reste utile (une perte sur UNE vie ou UN vehicule se
//     voit) mais son nom le dit : `par-slot`, jamais `par-xuid`.
//
// Reconstruire un pont slot -> xuid par recoupement temporel avec le calque `tracks`
// ajouterait une dependance fragile (un slot peut changer de piste en cours de match) pour un
// outil de DIFF qui doit rester simple et correct sur quarante bumps de schema. Un span sans
// xuid NI slot (ZoneSpan : seul `owner`, une EQUIPE, y vit) n'est ventile qu'au niveau du
// calque entier — c'est deja ce que la source sait dire.
//
// # LE SENS DE L'ECART EST CELUI DE `comparaison.go`, SANS CHANGEMENT
//
// Cet axe ne fait qu'AJOUTER des mesures a l'empreinte (`<axe>/<chemin>/duree-totale[/...]`) ;
// `Comparer` les classe exactement comme les autres (une somme plus basse = perte, plus haute
// = gain, absente d'un cote = apparu/disparu). Aucune modification de comparaison.go.

import "strconv"

// dureesProfondeurMax borne la descente recursive. Les spans a intervalles vivent a 1 ou 2
// niveaux sous leur calque de premier niveau (`flagCarries[].spans[]`,
// `vehicles[].rides[]`) ; 6 laisse une marge large sans risquer une explosion sur un document
// profondement imbrique.
const dureesProfondeurMax = 6

// mesurerDurees parcourt tout le document et pose, pour chaque calque a intervalles [t0,t1]
// qu'il trouve, la somme de ses durees.
func mesurerDurees(e *Empreinte, doc map[string]any) {
	for k, v := range doc {
		mesurerDureesValeur(e, axeDe(k), k, v, 0)
	}
}

// mesurerDureesValeur descend dans UNE valeur du document. `chemin` ne grandit qu'en
// traversant un OBJET (`obj.champ`) — un element de tableau garde le chemin de son tableau
// parent, pour que tous ses freres accumulent sur LA MEME mesure.
func mesurerDureesValeur(e *Empreinte, axe, chemin string, v any, profondeur int) {
	if profondeur > dureesProfondeurMax {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		if d, ok := dureeSpan(t); ok {
			e.incr(axe, chemin+"/duree-totale", d)
			ventilerDureeParIdentite(e, axe, chemin, t, d)
		}
		for champ, sous := range t {
			mesurerDureesValeur(e, axe, chemin+"."+champ, sous, profondeur+1)
		}
	case []any:
		for _, it := range t {
			mesurerDureesValeur(e, axe, chemin, it, profondeur+1)
		}
	}
}

// dureeSpan lit un couple t0/t1 sur un objet et rend sa duree en frames, T1 INCLUS — la
// convention documentee sur chacun des spans du document de rejeu. Un objet sans les DEUX
// champs numeriques, ou avec t1 < t0, n'est pas un span : `ok` vaut faux, rien n'est mesure.
func dureeSpan(obj map[string]any) (float64, bool) {
	t0, ok0 := nombre(obj["t0"])
	t1, ok1 := nombre(obj["t1"])
	if !ok0 || !ok1 || t1 < t0 {
		return 0, false
	}
	return t1 - t0 + 1, true
}

// ventilerDureeParIdentite ajoute la ventilation individuelle QUAND le span porte une
// identite — xuid en priorite (la cle stable du joueur), slot a defaut (vie ou entite du
// calque). Un span sans l'un ni l'autre (ZoneSpan : seul `owner`, une equipe) ne ventile que
// le total du calque, deja pose par l'appelant.
func ventilerDureeParIdentite(e *Empreinte, axe, chemin string, obj map[string]any, duree float64) {
	if xuid := chaine(obj["xuid"]); xuid != "" {
		e.incr(axe, chemin+"/duree-totale/par-xuid/"+xuid, duree)
		return
	}
	if slot, ok := nombre(obj["slot"]); ok {
		e.incr(axe, chemin+"/duree-totale/par-slot/"+strconv.FormatInt(int64(slot), 10), duree)
	}
}
