package service

// replay_weapon_tier_check.go — LE GARDE-RAIL DE LA JOINTURE EMPLACEMENTS × SOCLES.
//
// CE QU'IL SURVEILLE, ET RIEN D'AUTRE. Le niveau d'une arme (base / terrain / puissance) vient
// de la CARTE : l'emplacement Forge qui confirme le socle du match à moins d'un mètre. Si ce
// croisement se décalait — mauvaise carte jointe, repère de coordonnées changé, référence
// re-générée de travers — les niveaux deviendraient faux SANS que rien n'échoue : chaque socle
// recevrait la nature de son voisin. Ce contrôle est le seul endroit où ça se verrait.
//
// IL NE REGARDE QUE DANS UN SENS, ET C'EST MESURÉ (étape 0, 76 artefacts, 669 socles) :
// « arme de rôle lourd sur un râtelier » est NOMINAL à 10,5 % (Hydra, Needler, Sentinel Beam,
// Shock Rifle sont posés sur râtelier par le jeu lui-même) ; alerter dessus ne produirait que
// du bruit. « Arme de rôle léger sur un socle de PUISSANCE » est à 0,45 % et ne peut guère
// s'expliquer autrement que par une jointure qui glisse.
//
// IL NE CORRIGE RIEN. Un socle reste au niveau que la carte lui donne, même compté ici.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/weapontier"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/weapons"
)

// checkWeaponTierJoin journalise quand la jointure des niveaux d'armes paraît décalée.
//
// Silencieux dans le cas nominal : un journal qui parle à chaque match ne se lit plus.
func (s *replayService) checkWeaponTierJoin(ctx context.Context, matchID string,
	doc *replay.ReplayDocument) {
	if doc.MapWeaponPads == nil || len(doc.WeaponPads) == 0 {
		return
	}
	// Le rôle se lit sur la clé canonique que `resolveWeaponLabels` vient de poser — la
	// donnée est déjà en mémoire, aucune seconde jointure, aucune ouverture de base.
	roles := weapons.RolesByKey()
	roleOf := func(weapon string) string {
		lbl, ok := doc.WeaponLabels[weapon]
		if !ok {
			return ""
		}
		return roles[lbl.Key]
	}
	m := weapontier.NewMatch(doc.WeaponPads, doc.MapWeaponPads, nil, false)
	c := m.RunCrossCheck(doc.WeaponPads, roleOf)
	if !c.Alert() {
		return
	}
	slog.WarnContext(ctx, "rejeu 2D : niveaux d'armes — des socles de puissance portent des armes legeres, "+
		"la jointure emplacements x socles est peut-etre decalee",
		"match_id", matchID, "titleSlug", s.titleSlug,
		"socles", c.Pads, "inversions", c.LightOnPower,
		"part", float64(c.LightOnPower)/float64(c.Pads), "seuil", weapontier.CrossCheckAlertShare,
		"armes", c.Weapons)
}
