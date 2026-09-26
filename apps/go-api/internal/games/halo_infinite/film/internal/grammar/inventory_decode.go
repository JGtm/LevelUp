package grammar

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// inventory_decode.go — L'INVENTAIRE COMPLET d'un biped à une image-clé : grenades portées
// avec leur type, capacité d'armure, munitions des deux emplacements, et emplacement dégainé.
//
// DESCENDU DE `film/replay` AU LOT J4.2 (2026-09-26, PLAN_SUITE_AUDIT_DECODEUR_FILM, DU-3 = S1),
// avec ses trois fichiers de regles (`inventory_ammo_rules.go`, `inventory_grenades_rules.go`,
// `inventory_grenade_selection.go`) : il LIT des bits, et l ADR 0034 D-1 dit que la couche de
// publication ne decode rien. Deplacement pur ; ses types de resultat vivent en `film/types`
// (`KeyframeInventory`, `SlotAmmo`, `KeyframeInventoryStats`). Le lecteur de bits prive
// `invBitAt` / `invBits` est descendu tel quel, puis a ete REMPLACE au lot J4.6 par les conventions
// nommees de `source` ([source.BitAt], [source.BitsTolerants] : zero hors bornes des deux cotes).
// Le REPLI du plafond de grenades se COMPTE chez l appelant (`replay`, `balayerInventaire`) :
// cette couche ne compte pas (D-4).
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
//     complet vient d'i48, dans les paquets delta (ScanFilmAbilityRanks) ;
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
// d'emplacements d'arme décrits par la carte mémoire (0x7F0 + s*0x90, quatre entrées). La
// dimension fait partie de la forme de [types.KeyframeInventory] : une seule source.
const invGrenadeSlots = types.InventorySlotCount

// DefaultGrenadeMax borne un compteur de grenade plausible. Un Spartan en porte deux par type ;
// la borne sert à écarter les motifs qui ressemblent à i22 par hasard, pas à contraindre une
// valeur réelle.
const DefaultGrenadeMax uint32 = 2

// LE BLOC MUNITIONS (i30..i42, SlotAmmo, invParseAmmoBlock, invSolveAmmoBlock, readAmmo,
// invAmmoSearchSpan) vit dans inventory_ammo_rules.go — seuil de taille du dépôt (CLAUDE.md
// n°5), même raison que le renvoi vers inventory_grenades_rules.go plus bas.

// ScanFilmKeyframeInventory décode l'inventaire de tous les keyframes du film de dir.
// `types.KeyframeInventoryStats` vit en `film/types`, a cote de `types.KeyframeInventory`.
// `known` est le prédicat d'appartenance au catalogue de familles d'arme : c'est lui qui borne
// le bloc de munitions (R4 s'appuie sur la position de la première arme). Sans lui, aucune
// munition n'est lue. HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de
// requête.
// ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle [ScanKeyframeInventory].
func ScanFilmKeyframeInventory(
	dir string, known map[uint32]bool, grenMax uint32,
) ([]types.KeyframeInventory, types.KeyframeInventoryStats, error) {
	if len(known) == 0 {
		return nil, types.KeyframeInventoryStats{}, nil // catalogue vide : rien a chercher
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, types.KeyframeInventoryStats{}, err
	}
	return ScanKeyframeInventory(NewFilmContext(film), known, grenMax)
}

// ScanKeyframeInventory décode l'inventaire des images-clés d'un film DEJA CHARGE.
//
// Les records viennent de la marche d'image-clé DU FILM ([FilmContext.MarcheDImageCle],
// lot D-fix) : celle des autres balayages de la cuisson.
//
// `grenMax` NUL : [DefaultGrenadeMax] s applique. C est le REPLI NOMME
// `repli_plafond_grenade_par_defaut` (le plafond est une donnee de MODE, pas une constante) ; il
// se COMPTE chez l appelant de production (`replay`, `balayerInventaire`) depuis le lot J4.2 —
// cette couche nomme ses replis, elle ne les compte pas (ADR 0034 D-4).
func ScanKeyframeInventory(
	fc *FilmContext, known map[uint32]bool, grenMax uint32,
) ([]types.KeyframeInventory, types.KeyframeInventoryStats, error) {
	var st types.KeyframeInventoryStats
	if len(known) == 0 {
		return nil, st, nil
	}
	if grenMax == 0 {
		grenMax = DefaultGrenadeMax
	}
	nums := fc.ChunkNumbers()
	st.Chunks = len(nums)
	marche := fc.MarcheDImageCle()
	var out []types.KeyframeInventory
	for _, c := range nums {
		chunk, pks, ok := fc.ChunkAt(c)
		if !ok {
			st.ChunksUnread++
			continue
		}
		for _, p := range pks {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			st.Keyframes++
			pay := p.Payload(chunk)
			invs := keyframeInventoriesDe(pay, invRecordSpansDe(pay, marche.Records(pay)), known, grenMax)
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
		return nil, st, ErrNoReadableFilmChunk
	}
	return out, st, nil
}

// keyframeInventories décode un payload de keyframe, un inventaire par record de biped.
// PUR (aucune I/O) — c'est le cœur testable. Sans preuve (marche des instruments) ; la cuisson passe
// par [keyframeInventoriesDe] sur les records de la marche de son film.
func keyframeInventories(pay []byte, known map[uint32]bool, grenMax uint32) []types.KeyframeInventory {
	return keyframeInventoriesDe(pay, invRecordSpans(pay), known, grenMax)
}

// keyframeInventoriesDe est [keyframeInventories] sur des records DEJA bornes.
func keyframeInventoriesDe(pay []byte, spans []invRecordSpan, known map[uint32]bool,
	grenMax uint32) []types.KeyframeInventory {
	out := make([]types.KeyframeInventory, 0, len(spans))
	for _, sp := range spans {
		if sp.ti != invBipedTI {
			continue
		}
		inv := types.KeyframeInventory{
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
	return invRecordSpansDe(pay, WalkKeyframeWorld(pay))
}

// invRecordSpansDe borne des records DEJA marches (cf. [invRecordSpans]).
func invRecordSpansDe(pay []byte, recs []KeyframeRec) []invRecordSpan {
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
		w = ((w << 1) | uint32(source.BitAt(pay, b))) & mask28
		if b-from < 27 || w != invAbilityAnchor {
			continue
		}
		for off := 0; off <= 60; off++ {
			p := b + 1 + off
			if p+20 > to {
				break
			}
			if uint32(source.BitsTolerants(pay, p, 20)) != invAbilityPattern {
				continue
			}
			out = append(out, invAbilityHit{anchorBit: b - 27, low: uint32(source.BitsTolerants(pay, p+20, 3))})
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
		w = w<<1 | uint32(source.BitAt(pay, b))
		if b-from < 31 {
			continue
		}
		if known[w] {
			return b - 31, true
		}
	}
	return 0, false
}
