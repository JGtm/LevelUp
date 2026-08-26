package replay

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmdec"
)

// inventory_decode.go — L'INVENTAIRE COMPLET d'un biped à une image-clé : grenades portées
// avec leur type, capacité d'armure, munitions des deux emplacements, et emplacement dégainé.
//
// POURQUOI CE CODE VIT DANS LA COUCHE REJEU ET NON DANS `filmdec`.
//
// C'est une question de COUCHES, pas de propriété. `filmdec` porte les primitives GÉNÉRIQUES du
// format — parcourir les paquets, borner les records d'une image-clé, déquantifier. L'inventaire,
// lui, est un décodage SPÉCIFIQUE au rejeu, bâti sur ces primitives : il vit donc avec ce qu'il
// sert, à côté de sa projection (inventory.go) et du fil des morts (deaths_source.go), qui
// suivent déjà cette règle dans ce paquet.
//
// La frontière est tenue par la discipline d'appel : ce fichier ne touche au parser que par sa
// SURFACE PUBLIQUE (`CountFilmChunks`, `ReadFilmChunk`, `WalkPackets`, `WalkKeyframeWorld`,
// `PacketTypeKeyframe`). Il n'emprunte aucune primitive interne — d'où `invBitAt` plutôt qu'un
// helper non exporté du paquet voisin. Si cette liste d'appels devait s'allonger, ce serait le
// signe qu'il faut un PORT explicite (une interface côté rejeu, implémentée par un adaptateur),
// et non une intrusion de plus.
//
// CE QUI, EN REVANCHE, N'APPARTIENT PAS À CE CHANTIER : le FIL DES ÉLIMINATIONS — qui a tué qui
// et comment, l'assistance et sa part de dégâts, le kill par véhicule. Il a sa source de vérité
// dans le chantier voisin (`filmdec-killweapon`, paquet `killsource`), où il continue d'évoluer.
// Le rejeu doit le CONSOMMER par une entrée de données, jamais le redécoder ici : deux décodeurs
// du même fait divergeraient, et c'est précisément ce que la séparation en couches évite.
//
// D'OÙ CE CODE VIENT. Il était une passe de recherche (`cmd/tmp_kfinv`). Les règles d'ancrage
// ci-dessous sont reprises telles quelles : elles ont toutes été énoncées AVANT toute
// confrontation à la vérité terrain, ce qui est la seule façon qu'un accord signifie quelque
// chose.
//
//	R1 capacité  : ancre 28 bits 0x8CAC57A, puis dans les 60 bits suivants le motif 20 bits
//	               0b00000000000000010010 ; les 3 bits qui suivent sont les BITS DE POIDS
//	               FAIBLE du rang de palette, dont le motif porte déjà les bits de poids fort
//	               (cf. invAbilityRankHigh). Retenue seulement si l'ancre est UNIQUE dans le
//	               record.
//	R2 grenades  : PREMIER motif i22 (R(3)=4 puis quatre R(8) bornés, somme > 0) situé APRÈS
//	               l'ancre capacité. La position de l'ancre vient de R1, donc sans aucune
//	               information de grenade.
//	R3 armes     : familles connues trouvées dans le record, dans l'ORDRE DES BITS.
//	R4 munitions : le bloc i30..i42 se termine EXACTEMENT sur le bit de porte d'i43, juste
//	               avant la première famille d'arme. On énumère les débuts possibles et on ne
//	               garde que ceux dont le parse ATTERRIT au bit près. Aucune VALEUR n'entre
//	               dans le critère : c'est un critère de LARGEUR, pas de contenu — sans quoi on
//	               choisirait la lecture qui donne le résultat attendu.
//
// CE QUE LE CONTRÔLE TERRAIN A DONNÉ, relevé à l'écran par l'utilisateur sur l'image-clé de la
// 16e seconde, huit joueurs : 8/8 sur la grenade portée, 8/8 sur le nom de la capacité, 8/8 sur
// la présence de l'arme relevée dans la paire, 7/7 sur chargeur et réserve. Le huitième est le
// marteau à gravité, qui n'émettait rien à cet instant — parce qu'il était PLEIN (cf. SlotAmmo :
// le plein est la valeur par défaut d'un flux différentiel, donc il n'est jamais transmis).
//
// CE QUI RESTE INCONNU, ET S'AFFICHE COMME TEL :
//   - le COMPTEUR D'UTILISATIONS de la capacité n'est pas localisé (36 006 positions testées
//     sur 6 ancres, aucune ne reproduit le relevé) ;
//   - la table des capacités est PARTIELLE, et propre à la PALETTE du match : un rang hors
//     table doit s'afficher « inconnu », jamais être deviné ;
//   - ce canal est BORGNE — il ne voit que les rangs 16 à 23 (invAbilityRankHigh). Le rang
//     complet vient d'i48, dans les paquets delta (filmdec.ScanFilmAbilityRanks) ;
//   - 51 records sur 150 admettent plusieurs parses du bloc de munitions. Le plus long est
//     retenu et le NOMBRE DE CANDIDATS est publié, pour que le départage reste visible.

// invBipedTI est l'archétype (typeIndex) des records de biped joueur dans la table keyframe.
//
// Il est redéclaré ici plutôt qu'emprunté au parser, pour la raison donnée en tête de fichier.
// La valeur est un FAIT DU FORMAT, pas un réglage : les armes au sol (ti=42) portent elles
// aussi un identifiant de famille, et les confondre donnerait « l'arme posée par terre » comme
// arme d'un joueur.
const invBipedTI = 35

// invAbilityAnchor est l'ancre 28 bits du record d'archétype biped portant la capacité.
const invAbilityAnchor uint32 = 0x8CAC57A

// invAbilityPattern est le motif 20 bits cherché dans les 60 bits qui suivent l'ancre.
const invAbilityPattern uint32 = 0x00012

// invAbilityRankHigh est ce que le motif d'ancrage DIT DÉJÀ du rang de capacité, et c'est la
// découverte du 2026-08-14 (RECETTE_LOADOUT §14).
//
// Les trois derniers bits d'`invAbilityPattern` valent `010` : ce ne sont pas une signature
// de structure, ce sont les BITS DE POIDS FORT du rang de palette. Les 3 bits lus juste
// après en sont les bits de poids faible. Le champ « index » que ce décodeur publiait depuis
// le début était donc `rang − 16`, et l'ancre elle-même ne peut matcher QUE les rangs 16 à
// 23 — d'où les 21 films sur 40 qui ne rendaient aucune lecture, et les parties « où les
// huit joueurs portent le même équipement », qui n'en montraient que les porteurs du rang 23.
//
// LA VALEUR EST DÉRIVÉE DU MOTIF, pas écrite à côté de lui : si le motif changeait, la
// reconstruction suivrait au lieu de mentir. Contrôle sur pièces (film 000d5950, le film de
// vérité terrain) : le canal i48, totalement indépendant, rend le rang 20 sur les slots que
// le relevé Theater nomme grappin (index 4), 21 sur le propulseur (index 5) et 19 sur le mur
// (index 3) — soit `rang = index + 16` sur trois valeurs.
const invAbilityRankHigh = invAbilityPattern & 0x7

// invAbilityRankOf reconstruit le RANG complet à partir des 3 bits de poids faible lus.
func invAbilityRankOf(low uint32) int { return int(invAbilityRankHigh<<3 | (low & 0x7)) }

// invGrenadeSlots est le nombre de types de grenade décrits par i22, et aussi le nombre
// d'emplacements d'arme décrits par la carte mémoire (0x7F0 + s*0x90, quatre entrées).
const invGrenadeSlots = 4

// DefaultGrenadeMax borne un compteur de grenade plausible. Un Spartan en porte deux par type ;
// la borne sert à écarter les motifs qui ressemblent à i22 par hasard, pas à contraindre une
// valeur réelle.
const DefaultGrenadeMax uint32 = 2

// LE BLOC MUNITIONS (i30..i42, SlotAmmo, invParseAmmoBlock, invSolveAmmoBlock, readAmmo,
// invAmmoSearchSpan) vit dans inventory_ammo_rules.go — seuil de taille du dépôt (CLAUDE.md
// n°5), même raison que le renvoi vers inventory_grenades_rules.go plus bas.

// KeyframeInventory est l'inventaire d'un biped à l'instant d'une image-clé.
type KeyframeInventory struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent l'image-clé dans le film.
	Chunk, PacketIndex int
	// Slot est le slot du biped porteur (celui des trajectoires).
	Slot uint32
	// Grenades porte le compteur de chaque type, par rang (0 Fragmentation, 1 Plasma,
	// 2 Dynamo, 3 Spike). Nulle et GrenadesRead faux = non lu, jamais « zéro grenade ».
	Grenades     [invGrenadeSlots]uint32
	GrenadesRead bool
	// GrenadesByPosition dit par QUELLE VOIE les compteurs ont été lus : faux = R2a, par
	// l'ancre de capacité ; vrai = R2b, par la position relative au bloc de munitions (cf.
	// inventory_grenades_rules.go). Sans conséquence sur la valeur publiée — les deux voies
	// lisent le même champ, et sur les 1 167 records où les deux s'appliquent elles rendent la
	// même position 1 167 fois. C'est une TÉLÉMÉTRIE : elle alimente KeyframeInventoryStats,
	// pour qu'une dérive du repli se voie au lieu de se fondre dans le total.
	GrenadesByPosition bool
	// SelectedGrenadeRank est le rang de grenade SÉLECTIONNÉ (i47), ou -1 non lu. C'est le
	// type qui partira au prochain lancer. Publié seulement si le masque lu recoupe
	// exactement les compteurs i22 et si la sélection est unanime dans la fenêtre (cf.
	// invGrenadeSelLo) : à défaut, -1 — une sélection ne se devine pas.
	SelectedGrenadeRank int
	// AbilityRank est le RANG de palette de la capacité portée, ou -1 non lu.
	//
	// C'EST UN RANG, PAS UN INDEX — il l'est depuis le 2026-08-14 (cf. invAbilityRankHigh), et
	// le champ a changé de nom parce qu'il a changé de grandeur. Ce canal ne voit QUE la
	// fenêtre 16..23 de la palette : hors d'elle, l'ancre ne matche pas et la lecture n'existe
	// pas. Le rang complet, sur toute la palette, vient d'i48 (filmdec.ScanFilmAbilityRanks).
	//
	// Le NOM ne se décide pas ici : la table est partielle ET propre à la palette du match,
	// et la nommer est le travail de la couche qui possède le catalogue.
	AbilityRank int
	// Ammo est l'état des quatre emplacements décrits par la carte mémoire. Seuls les deux
	// premiers portent une arme ; les deux autres sont vides, et cette vacuité fait partie du
	// critère de parse (44 bits nuls).
	Ammo     [invGrenadeSlots]SlotAmmo
	AmmoRead bool
	// DrawnSlot est le sélecteur i42 : 0 ou 1 = cet emplacement est dégainé, 2 = aucune arme
	// dégainée, -1 = non lu. LE 2 EST UNE VALEUR : au premier keyframe le match n'a pas
	// commencé et les huit joueurs ont leurs armes rangées.
	DrawnSlot int
	// AmmoCandidates est le nombre de débuts de bloc qui satisfaisaient le critère. 1 = lecture
	// unique ; au-delà, le plus long a été retenu et ce nombre dit que le départage a eu lieu.
	AmmoCandidates int
}

// ScanFilmKeyframeInventory décode l'inventaire de tous les keyframes du film de dir.
// `KeyframeInventoryStats` vit dans inventory.go avec `InventoryCoverage` (seuil de taille).
// `known` est le prédicat d'appartenance au catalogue de familles d'arme : c'est lui qui borne
// le bloc de munitions (R4 s'appuie sur la position de la première arme). Sans lui, aucune
// munition n'est lue. HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de
// requête.
func ScanFilmKeyframeInventory(
	dir string, known map[uint32]bool, grenMax uint32,
) ([]KeyframeInventory, KeyframeInventoryStats, error) {
	var st KeyframeInventoryStats
	if len(known) == 0 {
		return nil, st, nil
	}
	if grenMax == 0 {
		grenMax = DefaultGrenadeMax
	}
	n := filmdec.CountFilmChunks(dir)
	st.Chunks = n
	var out []KeyframeInventory
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			st.ChunksUnread++
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			st.Keyframes++
			invs := keyframeInventories(p.Payload(chunk), known, grenMax)
			st.Records += len(invs)
			for _, inv := range invs {
				inv.TimestampUS, inv.Chunk, inv.PacketIndex = p.TimestampUS, c, p.Index
				switch {
				case !inv.GrenadesRead:
				case inv.GrenadesByPosition:
					st.GrenadesByPosition++
				default:
					st.GrenadesByAnchor++
				}
				out = append(out, inv)
			}
		}
	}
	if st.ChunksUnread == st.Chunks {
		return nil, st, fmt.Errorf("aucun chunk film lisible dans %s", dir)
	}
	return out, st, nil
}

// keyframeInventories décode un payload de keyframe, un inventaire par record de biped.
// PUR (aucune I/O) — c'est le cœur testable.
func keyframeInventories(pay []byte, known map[uint32]bool, grenMax uint32) []KeyframeInventory {
	spans := invRecordSpans(pay)
	out := make([]KeyframeInventory, 0, len(spans))
	for _, sp := range spans {
		if sp.ti != invBipedTI {
			continue
		}
		inv := KeyframeInventory{
			Slot: uint32(sp.slot), AbilityRank: -1, DrawnSlot: -1, SelectedGrenadeRank: -1,
		}
		// R1 : l'ancre doit être UNIQUE dans le record. Deux ancres, c'est une lecture qu'on
		// ne sait pas départager — et on ne départage pas au hasard.
		if hits := invAbilityIn(pay, sp.from, sp.to); len(hits) == 1 {
			inv.AbilityRank = invAbilityRankOf(hits[0].low)
			// R2 : les grenades se cherchent APRÈS l'ancre de capacité, dont la position a été
			// établie sans aucune information de grenade.
			if g, ok := invGrenadesAfter(pay, hits[0].anchorBit, sp.to, grenMax); ok {
				inv.Grenades, inv.GrenadesRead = g, true
			}
		}
		// R3 puis R4 : la première famille d'arme borne le bloc de munitions par la DROITE.
		if first, ok := invFirstFamily(pay, sp.from, sp.to, known); ok {
			if start, got := readAmmo(pay, &inv, sp.from, first); got && !inv.GrenadesRead {
				// R2b : LE REPLI POSITIONNEL. Il n'entre en jeu que si R2a n'a rien rendu —
				// la voie par l'ancre reste prioritaire, donc aucune lecture existante ne
				// change. Son repère est le début du bloc de munitions, que R4 vient
				// d'établir au bit près et qui ne doit RIEN à l'ancre de capacité.
				if g, ok2 := invGrenadesNearAmmo(pay, start, sp.from, grenMax); ok2 {
					inv.Grenades, inv.GrenadesRead = g, true
					inv.GrenadesByPosition = true
				}
			}
		}
		// R5 : la sélection de grenade (i47) vit après la DERNIÈRE famille d'arme ; sa
		// lecture est bornée par les compteurs i22 eux-mêmes (masque == bitmap).
		if inv.GrenadesRead {
			if last, ok := invLastFamily(pay, sp.from, sp.to, known); ok {
				inv.SelectedGrenadeRank = invGrenadeSelection(pay, last+32, sp.to, inv.Grenades)
			}
		}
		out = append(out, inv)
	}
	return out
}

// invRecordSpan borne un record dans le payload : de son premier bit à celui du suivant.
type invRecordSpan struct {
	slot, ti int
	from, to int
}

// invRecordSpans découpe le payload en records, bornes données par WalkKeyframeWorld — le même
// walker que keyframe_loadout.go, déjà validé 249/250 entités et 8/8 bipeds.
func invRecordSpans(pay []byte) []invRecordSpan {
	recs := filmdec.WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	out := make([]invRecordSpan, 0, len(recs))
	total := len(pay) * 8
	for i, r := range recs {
		to := total
		if i+1 < len(recs) {
			to = recs[i+1].Bit
		}
		out = append(out, invRecordSpan{slot: r.Slot, ti: r.TI, from: r.Bit, to: to})
	}
	return out
}

// invAbilityHit est une occurrence de l'ancre de capacité.
type invAbilityHit struct {
	anchorBit int
	// low est le champ de 3 bits qui suit le motif : les bits de POIDS FAIBLE du rang.
	low uint32
}

// invAbilityIn cherche l ancre puis le motif, et lit les 3 bits de poids faible du rang.
func invAbilityIn(pay []byte, from, to int) []invAbilityHit {
	var out []invAbilityHit
	var w uint32
	const mask28 = (uint32(1) << 28) - 1
	for b := from; b < to; b++ {
		w = ((w << 1) | invBitAt(pay, b)) & mask28
		if b-from < 27 || w != invAbilityAnchor {
			continue
		}
		for off := 0; off <= 60; off++ {
			p := b + 1 + off
			if p+20 > to {
				break
			}
			if invBits(pay, p, 20) != invAbilityPattern {
				continue
			}
			out = append(out, invAbilityHit{anchorBit: b - 27, low: invBits(pay, p+20, 3)})
			break
		}
	}
	return out
}

// LES DEUX VOIES DE LECTURE DES GRENADES (R2a par l'ancre, R2b par la position) vivent dans
// inventory_grenades_rules.go — seuil de taille du dépôt (CLAUDE.md n°5).

// invFirstFamily rend la position bit de la PREMIÈRE famille d'arme connue du record.
func invFirstFamily(pay []byte, from, to int, known map[uint32]bool) (int, bool) {
	var w uint32
	for b := from; b < to; b++ {
		w = w<<1 | invBitAt(pay, b)
		if b-from < 31 {
			continue
		}
		if known[w] {
			return b - 31, true
		}
	}
	return 0, false
}

// invBitAt lit UN bit, et rend 0 hors bornes.
//
// LA TOLÉRANCE HORS BORNES EST LE POINT : ce décodeur lit délibérément jusqu'aux limites d'un
// record — c'est même son critère d'arrêt. Un lecteur qui paniquerait au-delà de la fin du
// payload ferait tomber le décodage sur des films parfaitement valides.
//
// Ce helper vit ICI plutôt que d'emprunter celui du parser : c'est le prix, assumé, de la
// frontière décrite en tête de fichier. Emprunter une primitive non exportée reviendrait à
// souder les deux paquets.
func invBitAt(buf []byte, p int) uint32 {
	if idx := p >> 3; idx >= 0 && idx < len(buf) {
		return uint32(buf[idx]>>(7-uint(p&7))) & 1
	}
	return 0
}

// invBits lit n bits à partir de p, avec la même tolérance hors bornes.
func invBits(pay []byte, p, n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | invBitAt(pay, p+i)
	}
	return v
}
