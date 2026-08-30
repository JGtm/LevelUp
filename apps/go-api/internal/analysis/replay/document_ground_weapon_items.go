package replay

// document_ground_weapon_items.go — LES ARMES AU SOL, une par une, affichables du lâcher à la
// disparition OBSERVÉE.
//
// CE QUE C'EST, ET CE QUE ÇA REMPLACE. Le schéma 25 publiait le lâcher avec une borne
// d'affichage `until` tirée d'une TABLE DE DURÉES — une convention, pas une mesure, et
// l'utilisateur l'a refusée (« je veux juste voir quand elle est au sol et quand elle
// disparaît »). Ce calque publie à la place l'OBJET lui-même : sa position, son instant
// d'apparition, et sa fin telle que le film la MONTRE. La table de durées est supprimée.
//
// D'OÙ VIENNENT LES FINS, ET POURQUOI ELLES SONT SÛRES :
//
//   - `pickup`  une prise du flux delta (schéma 25, datée à la milliseconde) tombe dans la
//     fenêtre de vie de l'objet, à moins de 1,5 m de lui. MESURE FONDATRICE (2026-08-30,
//     ground_link_research_test.go) : à l'instant exact d'une prise, l'objet le plus proche
//     est à 0,61 m en médiane (Catalyst, 74,8 % sous 1 m ; Behemoth 0,75 m), contre 4 à 7 m
//     pour un autre bipède au même instant. C'est la condition de reprise écrite au
//     REGISTRE_REPORTS (« un oracle plus rapproché que 20 s ») — levée par le schéma 25.
//   - `seen`    l'objet cesse d'être recensé par les images-clés : sa fin d'affichage est la
//     DERNIÈRE image-clé qui le voit. Il a disparu entre elle et la suivante (~20 s plus
//     tard) — l'affichage s'arrête à la dernière preuve de présence, il ne prolonge rien.
//   - `open`    rien ne prouve sa disparition (encore recensé à la dernière image-clé, ou né
//     après elle) : il reste affiché jusqu'à la fin du document.
//
// CE QUE LE CALQUE NE PUBLIE PAS : les objets APPARUS AU REPOS sans jamais bouger — les armes
// de socle. Elles appartiennent au calque des socles (`weaponPads`), qui les publie par
// grappes récurrentes ; les publier ici en double ferait deux vérités pour un même objet.
// Ici ne sortent que les objets qui ont BOUGÉ : une arme lâchée tombe, une arme de socle non.

import (
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

// Fins d'affichage publiées. Identifiants STABLES du document (même règle que Family/Origin).
const (
	// GroundWeaponEndPickup : un joueur l'a prise — fin datée à la milliseconde.
	GroundWeaponEndPickup = "pickup"
	// GroundWeaponEndSeen : disparue entre deux images-clés — fin à la dernière preuve.
	GroundWeaponEndSeen = "seen"
	// GroundWeaponEndOpen : disparition non observée — affichée jusqu'à la fin.
	GroundWeaponEndOpen = "open"
)

// gwItemLinkMaxDist est la distance maximale entre l'acteur d'un événement delta (lâcheur,
// ramasseur) et l'objet pour que le lien tienne. C'est `originDropMaxDist` : la constante de
// production de la règle du lâcher, dont ce lien est le miroir (même argument que
// gwPickupNearestPass — les deux bougent ensemble ou pas du tout).
const gwItemLinkMaxDist = originDropMaxDist

// GroundWeapon est UN objet arme au sol, borné par l'observation.
type GroundWeapon struct {
	// T0 / T1 / T1Max bornent l'affichage, sur le même axe que Point.T.
	//
	// POUR End == "seen", LA DISPARITION EST UN INTERVALLE, PAS UN INSTANT : T1 est la
	// DERNIÈRE PREUVE de présence (image-clé qui recense l'objet, à défaut sa naissance) et
	// T1Max la PREMIÈRE PREUVE d'absence (image-clé qui ne le recense plus). L'objet a
	// disparu quelque part entre les deux — le film ne dit pas où. Publier T1 seul
	// effaçait à tort les armes jamais recensées (vie plus courte qu'un intervalle
	// d'image-clé : T1 == T0, affichées zéro frame) ; publier T1Max seul les prolongerait
	// au-delà du prouvé. Le client choisit son rendu dans l'intervalle — plein jusqu'à T1,
	// dégradé jusqu'à T1Max, ou coupe franche — mais il choisit dans du MESURÉ.
	//
	// Pour End == "pickup" et "open", T1Max == T1 : la fin est exacte, ou l'objet reste.
	T0    int `json:"t0"`
	T1    int `json:"t1"`
	T1Max int `json:"t1max"`
	// X / Y / Z : la position de RÉFÉRENCE de l'objet — là où il s'est arrêté (dernier point
	// de sa piste), c'est-à-dire là où il gît. Mêmes axes que Point.X/Y.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z,omitempty"`
	// W est la famille d'arme en hexadécimal 8 chiffres — MÊME convention et MÊME espace
	// d'identifiants que Loadout.W et WeaponChange.W (le mot d'identité du record de création
	// se résout dans le même catalogue, c'est le filtre de la chaîne des socles).
	W string `json:"w"`
	// Origin est l'origine MESURÉE de l'apparition, même vocabulaire que les poses
	// d'équipement : `dropped` (une vie de bipède s'achève à moins de 2 frames et 1,5 m —
	// l'arme d'un mort) ou `spawned` (le reste : l'arme de départ abandonnée en ramassant
	// autre chose, l'arme éjectée d'un râtelier).
	Origin string `json:"origin"`
	// Dropper est le slot de la VIE qui l'a lâchée, quand un lâcher du flux delta coïncide
	// (même paquet à 500 ms près, moins de 1,5 m). -1 sinon.
	Dropper int `json:"dropper"`
	// End dit comment l'affichage se termine (GroundWeaponEnd*).
	End string `json:"end"`
	// Picker est le slot de la VIE qui l'a prise, pour End == "pickup". -1 sinon.
	Picker int `json:"picker"`
}

// GroundWeaponItemsCoverage dit ce que le calque a vu, lié, et refusé de dire.
type GroundWeaponItemsCoverage struct {
	// Objects est le nombre d'objets de la chaîne (identité résolue) ; Published ceux qui ont
	// bougé et sortent ici ; AtRest ceux laissés au calque des socles.
	Objects   int `json:"objects"`
	Published int `json:"published"`
	AtRest    int `json:"atRest"`
	// DropperNamed : objets publiés dont le LÂCHEUR est nommé — la vie de bipède qui s'achève
	// à leur naissance, celle-là même qui a classé l'apparition `dropped` (gwPadsClass).
	DropperNamed int `json:"dropperNamed"`
	// TakesTotal / PickupLinked : les prises du flux delta reçues, et celles qui ont trouvé
	// LEUR objet — MÊME FAMILLE, à moins de 1,5 m, dans la fenêtre de vie. La famille est un
	// CRITÈRE du lien, pas un contrôle après coup : sans elle, une prise de drapeau volait le
	// lien de l'arme voisine (mesuré : 27 désaccords sur 33 liens de la première version).
	// L'écart entre les deux compteurs n'est pas une anomalie : une prise de drapeau, une
	// prise au socle (objet jamais mobile, publié par `weaponPads`) ou une prise d'objet sans
	// piste ne se lient pas ici.
	TakesTotal   int `json:"takesTotal"`
	PickupLinked int `json:"pickupLinked"`
	// EndPickup / EndSeen / EndOpen ventilent les fins. Somme == Published.
	EndPickup int `json:"endPickup"`
	EndSeen   int `json:"endSeen"`
	EndOpen   int `json:"endOpen"`
}

// buildGroundWeaponItems projette les objets de la chaîne des socles sur l'axe du document et
// les LIE aux événements du flux delta. PUR.
//
// `objs` vient de `buildWeaponPads` (même chaîne que les socles) ; `changes` du balayage du
// schéma 25 ; `positions` est le nuage NON décimé trié par instant — la position d'un acteur à
// l'instant d'un événement se lit dedans.
func buildGroundWeaponItems(
	objs []gwPickupObject, changes []filmdec.HeldWeaponChange,
	positions []filmdec.BipedPosition, clock replayClock,
) ([]GroundWeapon, GroundWeaponItemsCoverage) {
	var cov GroundWeaponItemsCoverage
	cov.Objects = len(objs)
	if clock.step == 0 || len(objs) == 0 {
		return nil, cov
	}
	moving := make([]gwPickupObject, 0, len(objs))
	for _, o := range objs {
		if !o.Appar.HasDelta {
			cov.AtRest++
			continue
		}
		moving = append(moving, o)
	}
	bySlot := gwItemPositionsBySlot(positions)
	pickers := gwItemLinkPickups(moving, changes, bySlot, &cov)

	out := make([]GroundWeapon, 0, len(moving))
	for i, o := range moving {
		g := GroundWeapon{
			T0: clock.frame(o.Appar.TUS), X: o.Pos[0], Y: o.Pos[1], Z: o.Pos[2],
			W: fmt.Sprintf("%08x", o.FamilyID), Origin: o.Appar.Class,
			Dropper: o.DropperSlot, Picker: -1,
		}
		if g.Dropper >= 0 {
			cov.DropperNamed++
		}
		switch {
		case pickers[i].found:
			g.T1, g.End, g.Picker = clock.frame(pickers[i].tUS), GroundWeaponEndPickup,
				pickers[i].slot
			g.T1Max = g.T1
			cov.EndPickup++
		case o.Bounds.NeverPicked || o.Bounds.NoLaterKF:
			g.T1, g.End = clock.frames-1, GroundWeaponEndOpen
			g.T1Max = g.T1
			cov.EndOpen++
		default:
			g.T1, g.End = clock.frame(o.Bounds.LowUS), GroundWeaponEndSeen
			g.T1Max = clock.frame(o.Bounds.HighUS)
			cov.EndSeen++
		}
		if g.T1 < g.T0 {
			g.T1 = g.T0
		}
		if g.T1Max < g.T1 {
			g.T1Max = g.T1
		}
		out = append(out, g)
		cov.Published++
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].T0 != out[j].T0 {
			return out[i].T0 < out[j].T0
		}
		return out[i].W < out[j].W
	})
	if len(out) == 0 {
		return nil, cov
	}
	return out, cov
}

// frame projette un instant du film sur l'axe de frames, borné au document.
func (c replayClock) frame(tUS uint64) int {
	if tUS < c.origin {
		return 0
	}
	return clampFrame(int((tUS-c.origin)/c.step), c.frames)
}

// gwItemPositionsBySlot indexe le nuage par slot ; chaque liste hérite du tri par instant.
func gwItemPositionsBySlot(positions []filmdec.BipedPosition) map[uint32][]filmdec.BipedPosition {
	out := map[uint32][]filmdec.BipedPosition{}
	for _, p := range positions {
		if !p.HasWorld {
			continue
		}
		out[p.Slot] = append(out[p.Slot], p)
	}
	return out
}

// gwItemActorAt rend la position du slot à l'instant demandé (échantillon le plus proche, à
// equipOwnerWindowUS près — la fenêtre du poseur d'équipement, même mesure).
func gwItemActorAt(
	bySlot map[uint32][]filmdec.BipedPosition, slot uint32, at uint64,
) ([3]float32, bool) {
	list := bySlot[slot]
	i := sort.Search(len(list), func(i int) bool { return list[i].TimestampUS >= at })
	best, bestGap, found := [3]float32{}, uint64(equipOwnerWindowUS)+1, false
	for _, j := range []int{i - 1, i} {
		if j < 0 || j >= len(list) {
			continue
		}
		p := list[j]
		gap := at - p.TimestampUS
		if p.TimestampUS > at {
			gap = p.TimestampUS - at
		}
		if gap < bestGap {
			best, bestGap, found = [3]float32{p.X, p.Y, p.Z}, gap, true
		}
	}
	return best, found && bestGap <= equipOwnerWindowUS
}

// gwItemPick est le résultat d'un appariement de prise.
type gwItemPick struct {
	found bool
	slot  int
	tUS   uint64
}

// gwItemLinkPickups apparie chaque prise delta à l'objet qu'elle consomme : MÊME FAMILLE, la
// prise dans la fenêtre de vie de l'objet, le ramasseur à moins de 1,5 m de sa position de
// REPOS. Les prises se traitent par instant croissant et un objet ne se prend qu'une fois.
//
// LA FAMILLE EST UN CRITÈRE, PAS UN CONTRÔLE : la première version liait « l'objet le plus
// proche » et une prise de DRAPEAU (qui occupe un emplacement d'arme mais n'a pas d'objet
// ti=42) volait le lien de l'arme voisine — 27 mauvaises familles sur 33 liens. On ne lie que
// l'arme que la prise NOMME.
func gwItemLinkPickups(
	objs []gwPickupObject, changes []filmdec.HeldWeaponChange,
	bySlot map[uint32][]filmdec.BipedPosition, cov *GroundWeaponItemsCoverage,
) []gwItemPick {
	out := make([]gwItemPick, len(objs))
	takes := make([]filmdec.HeldWeaponChange, 0, len(changes))
	for _, ch := range changes {
		if ch.Kind == filmdec.HeldWeaponTaken || ch.Kind == filmdec.HeldWeaponSwapped {
			takes = append(takes, ch)
		}
	}
	sort.SliceStable(takes, func(i, j int) bool { return takes[i].TimestampUS < takes[j].TimestampUS })
	cov.TakesTotal = len(takes)
	for _, ch := range takes {
		if ch.Family == filmdec.NoWeaponVariant {
			continue
		}
		actor, ok := gwItemActorAt(bySlot, ch.Slot, ch.TimestampUS)
		if !ok {
			continue
		}
		best, bestD := -1, float64(gwItemLinkMaxDist)
		for i, o := range objs {
			if out[i].found || o.FamilyID != ch.Family ||
				ch.TimestampUS < o.Appar.TUS || ch.TimestampUS > o.Bounds.HighUS {
				continue
			}
			d := dist3(actor, o.Pos)
			if d < bestD {
				best, bestD = i, d
			}
		}
		if best < 0 {
			continue
		}
		out[best] = gwItemPick{found: true, slot: int(ch.Slot), tUS: ch.TimestampUS}
		cov.PickupLinked++
	}
	return out
}

// logGroundWeaponItems journalise le calque avec ses dénominateurs.
func logGroundWeaponItems(cov GroundWeaponItemsCoverage) {
	slog.Info("rejeu : armes au sol individuelles",
		"objets", cov.Objects, "publiees", cov.Published, "auRepos", cov.AtRest,
		"lacheurNomme", cov.DropperNamed, "prisesRecues", cov.TakesTotal,
		"ramasseurNomme", cov.PickupLinked,
		"finPickup", cov.EndPickup, "finVue", cov.EndSeen, "finOuverte", cov.EndOpen)
}
