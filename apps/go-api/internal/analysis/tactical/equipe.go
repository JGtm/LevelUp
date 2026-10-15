package tactical

// equipe.go — L'EQUIPE DE LA VICTIME A L'INSTANT DE SA MORT, pour la fiche d'une mort du detail d'une
// zone : qui etait la, dans quel etat, a quelle distance du mort et du tueur, et dans le rayon ou non.
//
// Pur : le contexte de la mort, les lignes de `match_death_mates` et le rayon du match sont donnes.
// Les REGLES DE PORTEE sont ici (`coordination.APortee`, borne inclusive) : le web lit `statut` et
// `dans_le_rayon`, il ne compare jamais une distance a un rayon.

import (
	"sort"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// EquipeDeLaMort assemble l'equipe d'une mort. `lignes` sont les coequipiers de la cle exacte de
// `c` ; l'appelant a verifie qu'ils sont `c.Total`. `tueur` est la position du tueur (nil = non
// mesuree : aucune distance au tueur). Sans rayon connu, ni statut ni appartenance au rayon : la
// liste et les distances restent.
func EquipeDeLaMort(c domain.ContexteDeMort, lignes []domain.MateDeMort, rayon float64, aUnRayon bool,
	tueur *domain.TacticalPoint) domain.TacticalEquipeALaMort {
	eq := domain.TacticalEquipeALaMort{Coequipiers: make([]domain.TacticalCoequipier, 0, len(lignes))}
	if aUnRayon {
		r := rayon
		eq.RayonM = &r
		eq.Statut = coordination.StatutDeLaMort(c.Visibles, c.HorsDeVue, c.PlusProcheM, rayon)
	}
	for _, l := range lignes {
		eq.Coequipiers = append(eq.Coequipiers, coequipierDe(l, rayon, aUnRayon, tueur))
	}
	sort.SliceStable(eq.Coequipiers, func(i, j int) bool {
		return xuidAvant(eq.Coequipiers[i].XUID, eq.Coequipiers[j].XUID)
	})
	return eq
}

// coequipierDe traduit une ligne ; position et distances pour le seul etat visible.
func coequipierDe(l domain.MateDeMort, rayon float64, aUnRayon bool, tueur *domain.TacticalPoint) domain.TacticalCoequipier {
	c := domain.TacticalCoequipier{XUID: l.MateXUID, Gamertag: l.Gamertag, Statut: l.Etat}
	if l.Etat != domain.TacticalMateVisible || l.X == nil || l.Y == nil {
		return c
	}
	p := domain.TacticalPoint{X: *l.X, Y: *l.Y}
	c.Point = &p
	if l.DistanceM != nil {
		d := *l.DistanceM
		c.DistanceMortM = &d
		c.DansLeRayon = aUnRayon && coordination.APortee(&d, rayon)
	}
	if tueur != nil {
		d := DistanceM(p, *tueur)
		c.DistanceTueurM = &d
	}
	return c
}

// xuidAvant ordonne deux xuids decimaux numeriquement (longueur, puis texte) : l'ordre dans lequel
// le collecteur les a ecrits.
func xuidAvant(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// KillDeLaMort retrouve, dans le journal d'un match, le kill qui est la mort de `victime` a
// l'instant `t` : la meme victime, a l'instant exact, a defaut a ± TolerancePlacementMs (le plus
// proche ; a egalite d'ecart, le plus ancien). nil si le journal ne la porte pas.
func KillDeLaMort(journal []domain.KillDuJournal, victime string, t int64) *domain.KillDuJournal {
	var meilleur *domain.KillDuJournal
	var ecartMin int64
	for i := range journal {
		k := &journal[i]
		if k.VictimXUID != victime {
			continue
		}
		ecart := k.TimeMs - t
		if ecart < 0 {
			ecart = -ecart
		}
		if ecart > TolerancePlacementMs {
			continue
		}
		if meilleur == nil || ecart < ecartMin || (ecart == ecartMin && k.TimeMs < meilleur.TimeMs) {
			meilleur, ecartMin = k, ecart
		}
	}
	return meilleur
}
