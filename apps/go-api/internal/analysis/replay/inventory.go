package replay

import (
	"log/slog"
	"sort"
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
	// LA CAPACITÉ D'ARMURE N'EST PLUS ICI (2026-08-14, SchemaVersion 6). Elle vivait sous `a`
	// et portait un INDEX TRONQUÉ — `rang − 16`, une grandeur différente du rang, sous un nom
	// qui ne le disait pas. Elle est publiée par `ReplayDocument.Abilities`, en RANG de
	// palette et sur son propre axe de temps : le canal i48 la transmet dans les paquets
	// delta, pas aux images-clés (cf. abilities.go).
	//
	// D est le sélecteur d'emplacement : 0 ou 1 = cet emplacement est dégainé, 2 = AUCUNE arme
	// dégainée. Pointeur pour la même raison, et le 2 compte : à la première image-clé le match
	// n'a pas commencé et les huit joueurs ont leurs armes rangées.
	D *int `json:"d,omitempty"`
	// Am est l'état de munitions des emplacements portant une arme, dans l'ordre de Loadout.W.
	Am []AmmoSlot `json:"am,omitempty"`
	// Cand est le nombre de lectures possibles du bloc de munitions. 1 = lecture unique ;
	// au-delà, la plus longue a été retenue et ce nombre rend le départage visible.
	Cand int `json:"cand,omitempty"`
	// Empty MARQUE une lecture qui ne rend RIEN — ni compteur de grenade, ni munition — et dit
	// POURQUOI. ABSENT = la lecture porte quelque chose ; les deux seules valeurs présentes sont
	// [InventoryEmptyDead] et [InventoryEmptyUnknown].
	//
	// POURQUOI CE CHAMP EXISTE (SchemaVersion 19, 2026-08-25). Une lecture vide était publiée
	// NUE : `{"t":N,"slot":S}`. Le client, qui retient la lecture la plus récente ≤ T, la
	// préférait à la lecture PLEINE qui la précédait, et faisait DISPARAÎTRE la fiche du joueur
	// jusqu'à l'image-clé suivante, soit ~20 s. 17,4 % des lectures publiées sont dans ce cas
	// (mesure du 2026-08-24, 6 721 records sur 24 films). Une lecture vide n'est pas une absence
	// de lecture : sans marqueur, elle EFFACE.
	//
	// LA VALEUR EST UNE MESURE, PAS UNE INTERPRÉTATION. `dead` n'est posé que lorsque le FIL DES
	// MORTS corrobore : l'instant de la lecture tombe dans les [invDeadWindowMS] qui suivent une
	// mort du porteur du slot. Recouvrement mesuré sur 8 films (1 419 records) : 88,3 % des
	// lectures vides, contre 1,1 % des lectures pleines soumises à la MÊME fenêtre — un rapport
	// de 82x qui ne s'obtient pas par construction (cf. inventory_mort_recouvrement_test.go).
	// Les 11,7 % restantes gardent `unknown` : le décodeur n'a rien lu, et personne ne sait
	// pourquoi. Les étiqueter « mort » affirmerait à l'écran ce qu'aucune pièce ne dit.
	Empty string `json:"empty,omitempty"`
}

// Les deux valeurs d'[Inventory.Empty]. `dead` est CORROBORÉ par le fil des morts ; `unknown`
// dit que la lecture est vide et que rien ne l'explique — jamais l'inverse.
const (
	InventoryEmptyDead    = "dead"
	InventoryEmptyUnknown = "unknown"
)

// invDeadWindowMS est la durée après une mort pendant laquelle un slot est tenu pour mort ou en
// réapparition.
//
// LA VALEUR N'EST PAS CHOISIE, ELLE EST MESURÉE DEUX FOIS. (1) `lives.go` relève une durée de
// réapparition de médiane 8,0 s. (2) Le balayage des fenêtres de 2 à 20 s
// (inventory_mort_recouvrement_test.go) place à 8 s le POINT DE SÉPARATION MAXIMALE entre le
// signal et son témoin : 88,3 % des lectures vides y tombent contre 1,1 % des lectures pleines
// (82x). Au-delà, le témoin s'envole — 7,3 % à 10 s, 13,1 % à 12 s : la fenêtre se met à
// attraper des joueurs réapparus, donc VIVANTS.
const invDeadWindowMS = 8_000

// invReadingIsEmpty dit si un record d'image-clé ne rendra RIEN À L'ÉCRAN : aucun compteur de
// grenade et aucune munition.
//
// POURQUOI CES DEUX DRAPEAUX ET PAS QUATRE. Les deux autres champs du record ne portent AUCUN
// contenu par eux-mêmes : le sélecteur d'emplacement (`DrawnSlot`) désigne une arme parmi celles
// que les munitions décrivent — sans elles il ne désigne rien —, et le rang de grenade
// sélectionné (`SelectedGrenadeRank`) désigne un type parmi les compteurs — sans eux, idem.
// Mesuré sur le film de vérité terrain : les 34 records sans grenade ni munition n'ont pas non
// plus de sélecteur (150 records portent munitions ET sélecteur, les mêmes).
//
// LA CONDITION EST CELLE DU DÉCODEUR, PAS CELLE DU DOCUMENT : elle se lit sur les drapeaux du
// record, avant toute projection. La tester après coup sur l'`Inventory` publié reviendrait à
// redécouvrir par ses champs vides ce que la lecture savait déjà.
func invReadingIsEmpty(r KeyframeInventory) bool {
	return !r.GrenadesRead && !r.AmmoRead
}

// buildInventory projette les inventaires décodés sur la grille de frames du rejeu. Rend
// aussi le nombre de lectures ÉCARTÉES parce qu'antérieures à l'origine — auparavant perdu
// sans compteur ni log (audit AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 2 ; seule la MESURE
// est ajoutée ici, le comportement du filtre lui-même reste hors périmètre de ce lot).
//
// Un inventaire ANTÉRIEUR à l'origine du rejeu est écarté : il n'a pas de place sur l'axe, et
// lui en inventer une le poserait sur la première image comme s'il y avait été mesuré.
func buildInventory(raw []KeyframeInventory, origin, step uint64) ([]Inventory, int) {
	if len(raw) == 0 {
		return nil, 0
	}
	out := make([]Inventory, 0, len(raw))
	droppedBeforeOrigin := 0
	for _, r := range raw {
		if r.TimestampUS < origin {
			droppedBeforeOrigin++
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
		if r.DrawnSlot >= 0 {
			d := r.DrawnSlot
			inv.D = &d
		}
		if r.AmmoRead {
			inv.Am = ammoSlotsOf(r)
		}
		if invReadingIsEmpty(r) {
			// LE DÉCODEUR NE SAIT PAS POURQUOI. Il sait seulement que la lecture est vide ;
			// l'étiquette `dead` se pose plus tard, à l'assemblage, où le fil des morts existe
			// (cf. markInventoryDeadReadings). Trancher ici obligerait à faire descendre les
			// morts dans le projecteur, qui n'a rien à en faire.
			inv.Empty = InventoryEmptyUnknown
		}
		out = append(out, inv)
	}
	if len(out) == 0 {
		return nil, droppedBeforeOrigin
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].T != out[j].T {
			return out[i].T < out[j].T
		}
		return out[i].Slot < out[j].Slot
	})
	return out, droppedBeforeOrigin
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

// KeyframeInventoryStats compte ce que ScanFilmKeyframeInventory (inventory_decode.go) a
// rencontré. Sans ces dénominateurs, une fiche clairsemée ne se diagnostique pas : rien ne
// distingue « peu de keyframes dans le film » de « chunks corrompus » (audit
// AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 3). Même vocabulaire que les scanners frères
// (filmdec.AbilityRankStats, CamoStateStats, GrappleStats). Vit ici, avec InventoryCoverage
// qu'elle alimente, et non dans inventory_decode.go (seuil de taille du dépôt, CLAUDE.md n°5).
type KeyframeInventoryStats struct {
	// Chunks est le nombre total de chunks du film (CountFilmChunks).
	Chunks int
	// ChunksUnread est le nombre de chunks dont la lecture disque a échoué — un `continue`
	// nu ne les révélait auparavant nulle part, ni compteur ni log.
	ChunksUnread int
	// Keyframes est le nombre de paquets d'image-clé parcourus, tous chunks confondus.
	Keyframes int
	// Records est le nombre de records de biped (ti=invBipedTI) rencontrés dans ces
	// images-clés. `keyframeInventories` n'en écarte AUCUN — chaque record produit une
	// lecture, lue ou non — donc ce compte est AUSSI le nombre de lectures rendues : la même
	// grandeur n'est pas dupliquée sous deux noms.
	Records int
}

// InventoryCoverage est la couverture du calque INVENTAIRE (munitions, grenades, capacité,
// emplacement dégainé) : combien de lectures le décodeur a produites, combien ont été
// écartées parce qu'antérieures à l'origine du rejeu, et combien ont été retirées faute de
// trajectoire publiée — symétrique de `Shots`/`Grenades` (cf. coverage.go), jusqu'ici le
// seul calque des quatre à partager `keepOfPublishedTracks` sans publier ce compte (audit
// AUDIT_AVAL_INVENTAIRE_2026-08-24.md, point 5).
//
// TÉLÉMÉTRIE PURE : ce champ n'affecte aucune valeur consommée par le client (aucun rendu
// n'en dépend), et il n'incrémente donc pas SchemaVersion — même règle que
// Structure/StructureBounds (cf. TestStructureIsOptionalInDocument, structure_test.go).
type InventoryCoverage struct {
	// Decoded est le nombre de lectures que le décodeur a produites (ScanFilmKeyframeInventory),
	// avant tout filtrage — le dénominateur.
	Decoded int `json:"decoded"`
	// DroppedBeforeOrigin est le nombre de lectures écartées parce que leur horodatage précède
	// l'origine du rejeu (cf. buildInventory) — potentiellement la lecture LA PLUS RICHE du
	// match (grenades et munitions de spawn, avant tout dégainage).
	DroppedBeforeOrigin int `json:"droppedBeforeOrigin"`
	// Unpublished est le nombre de lectures retirées faute de trajectoire publiée pour leur
	// slot (keepInventoryOfPublishedTracks) — même filtre, comptée à part, que les tirs, les
	// lancers et les armes portées.
	Unpublished int `json:"unpublished"`
	// Published est le nombre de lectures effectivement publiées dans le document —
	// Decoded == DroppedBeforeOrigin + Unpublished + Published, exactement.
	Published int `json:"published"`
}

// buildInventoryCoverage assemble la couverture du calque depuis les trois étapes de son
// filtrage : décodé (avant buildInventory), construit (après le filtre d'origine, avant celui
// des trajectoires publiées), publié (après les deux). Pure télémétrie : n'affecte aucun champ
// consommé par le client — n'incrémente donc pas SchemaVersion (même règle que
// Structure/StructureBounds, cf. TestStructureIsOptionalInDocument).
func buildInventoryCoverage(decoded []KeyframeInventory, built, published []Inventory, droppedBeforeOrigin int) *InventoryCoverage {
	return &InventoryCoverage{
		Decoded:             len(decoded),
		DroppedBeforeOrigin: droppedBeforeOrigin,
		Published:           len(published),
		Unpublished:         countUnpublished(len(built), len(published)),
	}
}

// attachInventoryCoverage pose la couverture du calque sur le document, et journalise — un seul
// endroit le fait, comme `attachFlagLayer` pour le drapeau vivant.
//
// LA GARDE EST LE POINT. `decoded == nil` veut dire que l'appelant n'a RIEN fourni à lire (le
// balayage du film a échoué : `inventory = nil` dans BuildFromFilm) ; le calque n'a alors pas
// de couverture, et son ABSENCE est l'information. Une tranche VIDE mais NON NULLE est l'autre
// cas — la lecture a eu lieu et n'a rien rendu —, et celle-là publie bien {0,0,0,0}. Confondre
// les deux fait passer une panne de décodage pour un film sans inventaire.
func attachInventoryCoverage(doc *ReplayDocument, decoded []KeyframeInventory, built []Inventory, droppedBeforeOrigin int) {
	if decoded == nil || doc.Coverage == nil {
		return
	}
	cov := buildInventoryCoverage(decoded, built, doc.Inventory, droppedBeforeOrigin)
	doc.Coverage.Inventory = cov
	slog.Info("rejeu : couverture inventaire",
		"decodees", cov.Decoded,
		"ecarteesAvantOrigine", cov.DroppedBeforeOrigin,
		"ecarteesSansPiste", cov.Unpublished,
		"publiees", cov.Published)
}
