// Package narrative resoud les badges narratifs (dominance, encounter, role
// d'impact, profil de participation) consommes par MatchView, Squad et Career.
//
// Pure analysis : 0 dependance DB, 0 HTTP. Toutes les fonctions prennent en
// entree des types canonical et retournent des structures expressives avec
// LabelKey (cle i18n manifest) et ColorToken (token semantique).
package narrative

import "levelup/go-api/internal/games/canonical"

// DominanceBadge represente le badge resolu d'une DominanceFlag.
//
//	LabelKey   : cle i18n manifest, ex. "narrative.dominance.domination".
//	ColorToken : token semantique, ex. "narrative.dominance.win.strong"
//	             (cf. PLAN_META_FOUNDATIONS_GO § 11.2).
type DominanceBadge struct {
	Flag       canonical.DominanceFlag
	LabelKey   string
	ColorToken string
}

// ResolveDominanceBadge retourne le badge si le flag est l'un des 5 narratifs.
// Pour DominanceNone (0) ou un flag inconnu, retourne nil (pas de badge).
func ResolveDominanceBadge(flag canonical.DominanceFlag) *DominanceBadge {
	switch flag {
	case canonical.DominanceDomination:
		return &DominanceBadge{
			Flag:       flag,
			LabelKey:   "narrative.dominance.domination",
			ColorToken: "narrative.dominance.win.strong",
		}
	case canonical.DominanceHumiliation:
		return &DominanceBadge{
			Flag:       flag,
			LabelKey:   "narrative.dominance.humiliation",
			ColorToken: "narrative.dominance.loss.strong",
		}
	case canonical.DominanceRemontada:
		return &DominanceBadge{
			Flag:       flag,
			LabelKey:   "narrative.dominance.remontada",
			ColorToken: "narrative.dominance.win.comeback",
		}
	case canonical.DominanceDebandade:
		return &DominanceBadge{
			Flag:       flag,
			LabelKey:   "narrative.dominance.debandade",
			ColorToken: "narrative.dominance.loss.collapse",
		}
	case canonical.DominanceContreRem:
		return &DominanceBadge{
			Flag:       flag,
			LabelKey:   "narrative.dominance.contre_remontada",
			ColorToken: "narrative.dominance.win.counter",
		}
	}
	return nil
}
