package replay

// identity_registry_occupations.go — LES LIENS D UN SIEGE STATBORG RECYCLE, BORNES (lot R1 du plan de
// suite d audit, constat C1 du rapport G-corpus J11).
//
// Un slot statborg dont le siege change d'occupant en cours de manche porte UN LIEN PAR OCCUPATION
// (`objectives.RoundIdentity.Occupations`, cf. slotidentity_occupations.go). Publier une seule
// ligne « de 0 a la derniere frame » pour ce slot, c etait publier le lien d'un joueur sur un
// intervalle ou il n'etait pas assis — le defaut du constat. Chaque occupation se publie ici avec
// SES bornes, et une occupation que le pont n'a pas nommee se publie `non_resolu` : l'abstention se
// voit et se compte (`coverage.statborgSlot.unresolved`).
//
// LES BORNES SONT SUR L HORLOGE DU FIL (celle des enregistrements d entite et des morts) : elles se
// posent sur l'axe de frames par le calage du fil des morts (`horlogeFilm = horlogeFil +
// DeathOffsetMS`). Sans calage apparie, aucune borne n'est exprimable : le slot se publie alors en
// UNE ligne `non_resolu` — un lien borne dont les bornes ne se disent pas ne se publie pas nomme.

import (
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite/film/internal/facts/objectives"
)

// lignesDesOccupations publie les occupations d'un siege recycle, une ligne chacune.
func lignesDesOccupations(slot, round int, occ []objectives.Occupation, c IdentityClock,
	calage *int64) []IdentityStatborgSlot {
	if calage == nil {
		return []IdentityStatborgSlot{{Slot: slot, Round: round, Link: canonical.Link{
			Source: canonical.LinkUnresolved, Method: canonical.MethodNone, From: 0, To: c.lastFrame()}}}
	}
	out := make([]IdentityStatborgSlot, 0, len(occ))
	for _, o := range occ {
		lien := canonical.Link{From: 0, To: c.lastFrame()}
		if !o.OpenFrom {
			lien.From = c.frameOf((int64(o.FromMS) + *calage) * 1000)
		}
		if !o.OpenTo {
			lien.To = c.frameOf((int64(o.ToMS)+*calage)*1000 - 1)
		}
		if o.XUID == "" {
			lien.Source, lien.Method = canonical.LinkUnresolved, canonical.MethodNone
		} else {
			lien.Source, lien.Method = canonical.LinkInferred, methodeStatborg(o.Origin)
		}
		out = append(out, IdentityStatborgSlot{Slot: slot, Round: round, XUID: o.XUID, Link: lien})
	}
	return out
}
