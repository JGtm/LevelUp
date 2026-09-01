package replay

// held_object_carry.go — LA CHRONOLOGIE D'UN OBJET DE MODE TENU EN MAIN (bombe d'Assaut,
// crâne d'Oddball), reconstruite du canal des armes tenues.
//
// # D'OÙ VIENT LA DONNÉE
//
// La bombe et le crâne sont des OBJETS TENUS : le moteur les réplique dans le composant
// weapon-state-type-info du bipède, exactement comme une arme (`filmdec.ScanFilmHeldWeaponChanges`).
// Leur famille (32 bits hauts de l'identifiant filmshell) est HORS du catalogue d'armes
// (`weaponv3.KnownWeaponHigh32`) mais DANS l'atlas HUD du jeu :
//
//	bombe  0x3fee4fcf  sprite contour-34 « ball | bomb » — B1 2026-09-01 : unique candidate
//	                   des 9 films d'Assaut (prise et lâchée sur chacun, médiane 13 slots-vies)
//	crâne  0x0017592c  contour-25 — déjà au manifeste replay_labels.toml ([[objective_objects]])
//
// # LA RECONSTRUCTION, ET SES DEUX RÈGLES
//
// PRISE = transition VERS la famille ; LÂCHER = transition DEPUIS. Une période de portage
// s'ouvre à la prise du slot s et se ferme au premier des trois : lâcher de s, MORT du
// porteur, fin du film. La mort ferme SANS émission : le canal ne lâche rien quand la vie du
// bipède s'arrête — c'est le fil des morts (via le pont slot->xuid) qui date la fermeture.
// Une prise par un AUTRE slot pendant une période encore ouverte la borne aussi (lecture
// prudente : l'objet n'a qu'un exemplaire, un échange main à main n'existe pas).
//
// # CE QUE LES MESURES DU 2026-09-01 ÉTABLISSENT (bombe_b2_chronologie_test.go)
//
//	V3 témoin Oddball `43716616` : 46/46 heartbeats de possession th=10 tombent dans une
//	    période pontée du MÊME joueur (100 %) ; les 10 événements th=10 restants créditent le
//	    TUEUR du porteur (skull_carriers_killed), vérifié contre le fil des kills.
//	V1 Assaut : le porteur à la pose est confronté au détonateur du statborg — chiffres
//	    dans l'en-tête du test, publiés avec leurs dénominateurs.
//
// HORS LIGNE par construction (les entrées viennent des scans disque) — jamais depuis un
// chemin de requête.

import "sort"

// HeldObjectEvent est une prise ou un lâcher de l'objet, daté sur l'horloge du match.
type HeldObjectEvent struct {
	// TimeMS est l'instant sur l'horloge du match (celle des StatRecord et du fil des morts).
	TimeMS int
	// Slot est le slot du bipède (une VIE, pas un joueur).
	Slot uint32
	// XUID est l'identité pontée du slot, 0 si le pont ne l'a pas nommée.
	XUID uint64
	// Pickup vaut vrai pour une prise, faux pour un lâcher.
	Pickup bool
}

// HeldObjectPeriod est une période de portage : un slot tient l'objet de DebutMS à FinMS.
type HeldObjectPeriod struct {
	Slot           uint32
	XUID           uint64
	DebutMS, FinMS int
	// FinParMort : la période a été fermée par la mort du porteur, pas par un lâcher.
	FinParMort bool
	// Ouverte : ni lâcher ni mort avant la fin du film (FinMS vaut alors HeldObjectOpenEndMS).
	Ouverte bool
}

// HeldObjectOpenEndMS est la borne de fin d'une période restée ouverte à la fin du film.
const HeldObjectOpenEndMS = 1 << 30

// HeldObjectCarry est la chronologie complète de l'objet sur un film.
type HeldObjectCarry struct {
	// Events : les prises et lâchers, dans l'ordre du film.
	Events []HeldObjectEvent
	// Periods : les périodes de portage reconstruites, dans l'ordre du film.
	Periods []HeldObjectPeriod
	// CarryMSByXUID : temps de portage cumulé par joueur ponté (les périodes ouvertes ne
	// comptent pas — leur fin n'est pas connue).
	CarryMSByXUID map[uint64]int
}

// heldObjectTransitions extrait les transitions de la famille depuis les événements datés.
type heldObjectTransition struct {
	tMS    int
	slot   uint32
	pickup bool
}

// BuildHeldObjectCarry reconstruit la chronologie d'un objet tenu.
//
//   - events : transitions datées ms match (l'appelant convertit TimestampUS via l'origine
//     d'horloge du film, cf. ScanFilmClockOrigin) ;
//   - slotXUID : le pont slot->xuid (ResolveSlotXUID) — un slot absent reste XUID 0 ;
//   - deaths : le fil des morts (ScanFilmDeaths), qui ferme les périodes des porteurs morts.
func BuildHeldObjectCarry(events []HeldObjectEvent, slotXUID map[uint32]uint64, deaths []Death) HeldObjectCarry {
	trans := make([]heldObjectTransition, 0, len(events))
	for _, e := range events {
		trans = append(trans, heldObjectTransition{tMS: e.TimeMS, slot: e.Slot, pickup: e.Pickup})
	}
	sort.SliceStable(trans, func(i, j int) bool { return trans[i].tMS < trans[j].tMS })

	mortsDe := map[uint64][]int{}
	for _, d := range deaths {
		mortsDe[d.XUID] = append(mortsDe[d.XUID], int(d.TimeMS))
	}
	out := HeldObjectCarry{CarryMSByXUID: map[uint64]int{}}
	for _, tr := range trans {
		out.Events = append(out.Events, HeldObjectEvent{
			TimeMS: tr.tMS, Slot: tr.slot, XUID: slotXUID[tr.slot], Pickup: tr.pickup,
		})
	}
	out.Periods = heldObjectPeriods(trans, slotXUID, mortsDe)
	for _, p := range out.Periods {
		if p.XUID != 0 && !p.Ouverte {
			out.CarryMSByXUID[p.XUID] += p.FinMS - p.DebutMS
		}
	}
	return out
}

// heldObjectPeriods déroule les transitions en périodes (prise -> lâcher | mort | fin).
func heldObjectPeriods(
	trans []heldObjectTransition, slotXUID map[uint32]uint64, mortsDe map[uint64][]int,
) []HeldObjectPeriod {
	var out []HeldObjectPeriod
	ouverte := -1
	fermer := func(fin int, parMort bool) {
		out[ouverte].FinMS = fin
		out[ouverte].FinParMort = parMort
		out[ouverte].Ouverte = false
		ouverte = -1
	}
	for _, tr := range trans {
		if !tr.pickup {
			if ouverte >= 0 && out[ouverte].Slot == tr.slot {
				fermer(tr.tMS, false)
			}
			continue
		}
		if ouverte >= 0 {
			fin, parMort := premiereMortDans(mortsDe, out[ouverte], tr.tMS)
			fermer(fin, parMort)
		}
		out = append(out, HeldObjectPeriod{
			Slot: tr.slot, XUID: slotXUID[tr.slot], DebutMS: tr.tMS, Ouverte: true,
		})
		ouverte = len(out) - 1
	}
	if ouverte >= 0 {
		p := out[ouverte]
		if fin, parMort := premiereMortDans(mortsDe, p, HeldObjectOpenEndMS); parMort {
			fermer(fin, true)
		} else {
			out[ouverte].FinMS = HeldObjectOpenEndMS
		}
	}
	return out
}

// premiereMortDans rend la première mort du porteur de p dans [p.DebutMS, avant], ou
// (avant, false) si aucune — la borne par défaut est l'instant de la prise suivante.
func premiereMortDans(mortsDe map[uint64][]int, p HeldObjectPeriod, avant int) (int, bool) {
	if p.XUID == 0 {
		return avant, false
	}
	for _, m := range mortsDe[p.XUID] {
		if m >= p.DebutMS && m <= avant {
			return m, true
		}
	}
	return avant, false
}
