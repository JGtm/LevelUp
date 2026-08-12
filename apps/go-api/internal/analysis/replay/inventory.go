package replay

import (
	"sort"
	"strconv"
)

// inventory.go — L'INVENTAIRE porté à la grille du rejeu.
//
// SOURCE : ScanFilmKeyframeInventory (inventory_decode.go) — les records de biped des paquets
// d'image-clé, les mêmes que les armes portées. Les règles d'ancrage et les contrôles vivent
// là-bas ; ici on ne fait que projeter sur l'axe de temps et nommer.
//
// CE QUE LE CALQUE GARANTIT : à l'instant T, ce slot AVAIT cet inventaire. Contrôle terrain
// relevé à l'écran sur une image-clé, huit joueurs : 8/8 sur la grenade portée, 8/8 sur le nom
// de la capacité, 7/7 sur chargeur et réserve.
//
// CE QU'IL NE GARANTIT PAS, et que le client doit dire :
//   - la CONTINUITÉ. Une image-clé toutes les ~20 s ; entre deux, ce qui s'affiche est la
//     dernière lecture connue. Son ÂGE est donc une information à part entière — âge médian
//     mesuré 8,4 s, et 7,1 % seulement des affichages ont moins d'une seconde.
//   - le COMPTEUR D'UTILISATIONS de la capacité : non localisé, donc jamais publié.

// LES DEUX TABLES DE LIBELLÉS QUI VIVAIENT ICI SONT PARTIES DANS LE TITRE (lot 3.1/3.2,
// 2026-08-02) : `config/titles/{slug}/mappings/replay_labels.toml`.
//
// Les rangs de grenade parce qu'ils étaient nommés DEUX FOIS et différemment (« Dynamo »
// ici, « Shock » dans le décodeur, pour le même rang et sur la même fiche) ; les
// capacités parce que leurs noms étaient EN FRANÇAIS DANS DU GO — ce qui interdisait
// l'anglais autant que l'ajout d'un titre. L'ordre des rangs, lui, reste une mesure du
// décodeur (filmdec.GrenadeTypeIDsByRank) : c'est une donnée, pas un libellé.

// AmmoSlot est l'état de munitions d'un emplacement d'arme, dans l'ordre de Loadout.W.
//
// LES TROIS CAS SONT DISTINCTS : chargeur (Mag/Res), jauge de consommation (Gauge), ou AUCUN
// des deux. Ce dernier cas veut dire que le film n'écrit RIEN pour cet emplacement : pour une
// arme à charge, c'est le PLEIN — le flux est différentiel et le plein est la valeur par
// défaut. Ce n'est PAS « zéro » : publier 0 affirmerait un chargeur vide.
type AmmoSlot struct {
	Mag *uint32 `json:"mag,omitempty"`
	Res *uint32 `json:"res,omitempty"`
	// Gauge est une fraction dans [0,1] sur 4096 niveaux, et elle compte CE QUI A ETE CONSOMME,
	// pas ce qui reste.
	//
	// DEUX TEMOINS CONCORDANTS (2026-07-31). (1) A la premiere image-cle du match — huit joueurs,
	// rien de degaine, chargeurs au plein conformes a la table — les armes a charge n emettent
	// AUCUN champ : si la jauge disait le restant, le plein serait une valeur maximale, pas une
	// absence. (2) Dans une meme vie, sur la meme arme, la valeur ne redescend JAMAIS : 6
	// hausses, 0 baisse, 5 stables, et les hausses sont des multiples du quantum de l arme.
	//
	// CONSEQUENCE POUR UN CLIENT : une barre de charge restante doit afficher le COMPLEMENT.
	// L afficher tel quel donne une barre inversee.
	Gauge *float32 `json:"gauge,omitempty"`
}

// Inventory est l'inventaire complet d'un slot à un instant d'image-clé.
type Inventory struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Slot désigne la Track concernée.
	Slot uint32 `json:"slot"`
	// G porte le compteur de chaque type de grenade, par rang (cf. GrenadeLabels). ABSENT =
	// non lu ; un tableau présent dont une case vaut 0 dit « ce type, aucune en réserve »,
	// ce qui est une mesure.
	G []uint32 `json:"g,omitempty"`
	// Gs est le rang de grenade SÉLECTIONNÉ (i47) — le type qui partira au prochain lancer.
	// POINTEUR : le rang 0 est une valeur. Nil = non lu ; publié seulement quand le masque
	// lu recoupe exactement les compteurs G et que la lecture est unanime (cf. décodeur) —
	// une sélection ne se devine pas, un client peut toutefois la DÉDUIRE quand un seul
	// type est porté.
	Gs *int `json:"gs,omitempty"`
	// A est l'index de capacité lu. POINTEUR : l'index 0 est une valeur, et omitempty
	// l'effacerait. Nil = non lu.
	A *int `json:"a,omitempty"`
	// D est le sélecteur d'emplacement : 0 ou 1 = cet emplacement est dégainé, 2 = AUCUNE arme
	// dégainée. Pointeur pour la même raison, et le 2 compte : à la première image-clé le match
	// n'a pas commencé et les huit joueurs ont leurs armes rangées.
	D *int `json:"d,omitempty"`
	// Am est l'état de munitions des emplacements portant une arme, dans l'ordre de Loadout.W.
	Am []AmmoSlot `json:"am,omitempty"`
	// Cand est le nombre de lectures possibles du bloc de munitions. 1 = lecture unique ;
	// au-delà, la plus longue a été retenue et ce nombre rend le départage visible.
	Cand int `json:"cand,omitempty"`
}

// buildInventory projette les inventaires décodés sur la grille de frames du rejeu.
//
// Un inventaire ANTÉRIEUR à l'origine du rejeu est écarté : il n'a pas de place sur l'axe, et
// lui en inventer une le poserait sur la première image comme s'il y avait été mesuré.
func buildInventory(raw []KeyframeInventory, origin, step uint64) []Inventory {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Inventory, 0, len(raw))
	for _, r := range raw {
		if r.TimestampUS < origin {
			continue
		}
		inv := Inventory{
			T:    int((r.TimestampUS - origin) / step),
			Slot: r.Slot,
			Cand: r.AmmoCandidates,
		}
		if r.GrenadesRead {
			inv.G = append(inv.G, r.Grenades[:]...)
		}
		if r.SelectedGrenadeRank >= 0 {
			gs := r.SelectedGrenadeRank
			inv.Gs = &gs
		}
		if r.AbilityIndex >= 0 {
			a := r.AbilityIndex
			inv.A = &a
		}
		if r.DrawnSlot >= 0 {
			d := r.DrawnSlot
			inv.D = &d
		}
		if r.AmmoRead {
			inv.Am = ammoSlotsOf(r)
		}
		out = append(out, inv)
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		return out[i].Slot < out[j].Slot
	})
	return out
}

// ammoSlotsOf ne publie que les DEUX emplacements qui portent une arme.
//
// Les deux autres décrits par la carte mémoire sont structurellement vides — leur vacuité sert
// de critère au décodage, elle n'a rien à dire à l'écran.
func ammoSlotsOf(r KeyframeInventory) []AmmoSlot {
	out := make([]AmmoSlot, 0, 2)
	for k := 0; k < 2 && k < len(r.Ammo); k++ {
		a := r.Ammo[k]
		slot := AmmoSlot{Mag: a.Mag, Res: a.Res}
		if a.Gauge != nil {
			g := float32(*a.Gauge)
			slot.Gauge = &g
		}
		out = append(out, slot)
	}
	return out
}

// keepInventoryOfPublishedTracks écarte les inventaires dont le slot n'a pas de trajectoire
// publiée : le client n'aurait aucune fiche où les poser.
func keepInventoryOfPublishedTracks(inv []Inventory, tracks []Track) []Inventory {
	return keepOfPublishedTracks(inv, tracks,
		func(i Inventory, published map[uint32]bool) bool { return published[i.Slot] })
}

// abilityLabelsUsed nomme les index de capacité que le document emploie RÉELLEMENT.
//
// Un index hors table n'entre pas : il gardera son numéro à l'écran, marqué comme non
// interprétable. La table est partielle et le dire vaut mieux que combler.
func abilityLabelsUsed(inv []Inventory, catalog map[int]Label) map[string]Label {
	out := map[string]Label{}
	for _, i := range inv {
		if i.A == nil {
			continue
		}
		if name, ok := catalog[*i.A]; ok {
			out[strconv.Itoa(*i.A)] = name
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
