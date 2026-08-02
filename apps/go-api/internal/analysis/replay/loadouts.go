package replay

// Rattachement des ARMES PORTÉES (loadout de keyframe) aux trajectoires du rejeu.
//
// SOURCE : filmdec.ScanFilmKeyframeLoadouts — les familles d'arme trouvées dans le record
// biped de chaque slot, à chaque keyframe type-2 (~un toutes les 18-20 s). Le slot vient des
// bornes de record de WalkKeyframeWorld : la jointure loadout -> trajectoire est donc DIRECTE
// (même numérotation de slot que Track.Slot), sans passer par un champ « joueur » du record —
// c'est ce champ-là, cherché à un offset inconnu, qui bloquait les tentatives précédentes.
//
// TÉMOIN CROISÉ, sur une grandeur INDÉPENDANTE du loadout (l'arme des events de tir, record
// type 105, autre paquet, autre décodeur) — film 000d5950 Cliffhanger, Fiesta :
//
//	POSITIF   arme du tir présente dans le loadout du MÊME slot (dernier keyframe <= T) :
//	          233/237 = 98,3 % (par joueur : 100/100/97/100/98/93/100/100 %).
//	NÉGATIF A même tir, loadout d'un AUTRE slot vivant au même instant : 104/1353 = 7,7 %.
//	NÉGATIF B tirs du joueur p contre le loadout du joueur (p+k) mod 8 : 31/433 = 7,2 %.
//	NÉGATIF C mêmes loadouts, mêmes slots, même keyframe, mêmes tirs — on ne casse QUE
//	          l'appariement record->slot (rotation d'un cran) : 17/237 = 7,2 %. Ce témoin ne
//	          touche pas au décodage : c'est donc bien la JOINTURE qui porte le 98,3 %.
//
// DISCRIMINANCE (sans elle, 98 % ne prouverait rien) : 80 combinaisons distinctes sur 150
// loadouts, entropie 6,08 bits, 23 armes portées, paire la plus fréquente 7/150.
//
// CE QUE LE TÉMOIN NE VALIDE PAS. Sur le film 01e1f945 (Catalyst, Slayer standard) le
// positif monte à 99,0 % mais les négatifs aussi (76 à 81 %) : 120 loadouts sur 168 y sont
// IDENTIQUES (MA40 AR + Mk51 Sidekick). Quand tout le monde porte la même chose, un accord
// à 99 % ne distingue personne. Le chiffre de Catalyst confirme la cohérence du décodage
// d'arme, RIEN DE PLUS ; la validation de la jointure repose entièrement sur Cliffhanger.
//
// DÉSACCORDS EXAMINÉS (les 4 de Cliffhanger) : tous sont un tir survenu 3 à 17 s APRÈS la
// dernière image-clé de la vie concernée, laquelle n'a aucune image-clé suivante (mort avant).
// Explication structurelle cohérente mais invérifiable : comptés comme désaccords.

import (
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/weaponv3"
)

// loadoutFamilies est le catalogue de familles interrogé par le balayage : la table de
// production dérivée de l'enum d'armes (weaponv3, elle-même dérivée de analysis.WeaponIDToName).
// C'est la SEULE source de vérité sur ce qu'est une arme ici — pas de liste parallèle.
func loadoutFamilies() map[uint32]bool {
	m := make(map[uint32]bool, len(weaponv3.KnownWeaponHigh32))
	for f := range weaponv3.KnownWeaponHigh32 {
		m[f] = true
	}
	return m
}

// buildLoadouts projette les loadouts de keyframe sur la grille de frames du rejeu.
//
// REPLI DES ALIAS : un même canon apparaît deux fois dans le record, sous deux identifiants
// de famille distincts (le second à +97 bits) qui résolvent au MÊME nom canonique. On replie
// sur le nom et on publie UN identifiant par arme — sans quoi le client afficherait deux fois
// le même fusil.
func buildLoadouts(raw []filmdec.KeyframeLoadout, origin, step uint64) []Loadout {
	if len(raw) == 0 {
		return nil
	}
	out := make([]Loadout, 0, len(raw))
	for _, l := range raw {
		if l.TimestampUS < origin {
			continue
		}
		seen := map[string]bool{}
		var ids []string
		for _, fam := range l.Families {
			name := weaponv3.WeaponName(fam)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			ids = append(ids, formatWeaponFamily(fam))
		}
		if len(ids) == 0 {
			continue
		}
		out = append(out, Loadout{
			T:    int((l.TimestampUS - origin) / step),
			Slot: l.Slot,
			W:    ids,
		})
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

// keepLoadoutsOfPublishedTracks écarte les loadouts dont le slot n'a pas de trajectoire
// publiée : le client n'aurait personne à qui les attribuer. Même règle que pour les tirs.
func keepLoadoutsOfPublishedTracks(loadouts []Loadout, tracks []Track) []Loadout {
	return keepOfPublishedTracks(loadouts, tracks,
		func(l Loadout, published map[uint32]bool) bool { return published[l.Slot] })
}

// formatWeaponFamily rend un identifiant de FAMILLE (high-32 du weapon-id) en hexadécimal,
// même convention d'écriture que formatWeaponID — mais 8 chiffres, pas 16 : une famille
// n'est PAS un weapon-id complet et la longueur le dit.
func formatWeaponFamily(fam uint32) string {
	const hexDigits = "0123456789ABCDEF"
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = hexDigits[fam&0xF]
		fam >>= 4
	}
	return "0x" + string(buf)
}
