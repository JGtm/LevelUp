package mappings

// loader_replay_labels_vehicles.go — LES FAMILLES DE CHASSIS DE VEHICULE QUI ONT BESOIN D ETRE
// DITES, ajoutees le 2026-09-16 (lot 1.9.9 du plan decodeur).
//
// POURQUOI CETTE TABLE EST PARTIELLE, ET DOIT LE RESTER. Le nom d une famille de vehicule est un
// NOM PROPRE du jeu (Warthog, Banshee, Mongoose) : il ne se traduit pas, et la CLE de la table
// des sprites EST deja ce nom (cf. `replay.VehicleLabel`, dont l absence de `en`/`fr` est une
// decision et pas un oubli). Dix-huit familles sur dix-neuf n ont donc AUCUNE entree ici, et
// c est le regime normal.
//
// CE QUI FAIT EXCEPTION : une famille qui n est PAS un vehicule. La TOURELLE AUTOMATIQUE BANNIE
// (`tourelle_auto_bannie`, chassis `0x038df01a`) est un ELEMENT DE CARTE — decision utilisateur
// du 2026-09-14, « ce sont des elements de la map » : un objet qui interdit la sortie de la zone
// de jeu, immobile, que personne ne conduit. Son nom n est pas un nom propre repris du jeu mais
// une DESCRIPTION, donc il se traduit ; et sa NATURE doit traverser jusqu au client, qui la
// dessine par un pictogramme dedie au lieu du marqueur neutre des chassis non resolus.
//
// LES DEUX CHAMPS SONT DONC INDISSOCIABLES : le libelle bilingue (jamais en dur cote Go, regle 1
// du depot) et le `kind`, qui est ce que le document publie pour que le web sache de quoi il
// s agit sans connaitre la famille.
//
// `sprite` DIT SI UN ASSET EST SERVI POUR CETTE FAMILLE, et c est le POINT D EXTENSION du jour ou
// l utilisateur en fournira un : passer `sprite` a `true` et deposer le PNG dans
// `static/vehicles-assets/{slug}/replay/` avec sa ligne d `index.json` suffit — aucune ligne de
// Go ne change. A `false`, le service ne compose AUCUNE URL (cf. `replay_vehicle_labels.go`) :
// une URL morte ferait un 404 par match, et le client ne saurait pas distinguer « pas encore
// charge » de « aucun asset ».

import (
	"fmt"
	"sort"
	"strings"
)

// vehicleFamilyEntry — une ligne de [[vehicle_families]].
type vehicleFamilyEntry struct {
	Family string `toml:"family"`
	En     string `toml:"en"`
	Fr     string `toml:"fr"`
	Kind   string `toml:"kind"`
	Sprite bool   `toml:"sprite"`
}

// VehicleFamilyKindMapElement — la seule NATURE declarable a ce jour : un ELEMENT DE CARTE.
//
// C EST UNE VALEUR PUBLIEE (elle traverse le document jusqu au client), donc ANGLAISE comme
// toutes les enumerations du contrat de rejeu (`unknown`, `deployed`, `event`/`mixed`/`gap`) —
// meme raison qu au 2026-09-02 : une valeur francaise couterait un changement de contrat apres
// backfill.
const VehicleFamilyKindMapElement = "map_element"

// vehicleFamilyKinds — liste FERMEE, meme doctrine que `equipmentKinds` et `shotTintKinds` : une
// valeur libre ferait tomber le client sur son rendu neutre EN SILENCE, ce qui est
// indistinguable d une famille volontairement non qualifiee.
var vehicleFamilyKinds = map[string]bool{
	VehicleFamilyKindMapElement: true,
}

// VehicleFamily — ce que le titre DIT d une famille de chassis : son nom dans les deux langues,
// sa nature, et si un asset est servi pour elle.
type VehicleFamily struct {
	// Family est la cle stable publiee par le document (`VehicleTrack.family`).
	Family string
	// En / Fr : le libelle affichable. Les deux sont obligatoires (meme regle que
	// `BilingualLabel` : quand aucun FR officiel n existe, le fichier met le EN dans les deux —
	// jamais de FR invente).
	En, Fr string
	// Kind est la NATURE publiee, dans `vehicleFamilyKinds`.
	Kind string
	// Sprite dit qu un asset est servi pour cette famille sous
	// `static/vehicles-assets/{slug}/replay/{family}.png`.
	Sprite bool
}

// VehicleFamilies retourne les familles declarees par le titre, keyees par famille (copie).
// nil-safe. Vide = le titre n en qualifie aucune, ce qui est le regime normal.
func (s *ReplayLabelSet) VehicleFamilies() map[string]VehicleFamily {
	if s == nil || len(s.vehicleFamilies) == 0 {
		return nil
	}
	out := make(map[string]VehicleFamily, len(s.vehicleFamilies))
	for k, v := range s.vehicleFamilies {
		out[k] = v
	}
	return out
}

// parseVehicleFamilies valide la table [[vehicle_families]].
//
// STRICTE ET TOUT-OU-RIEN, comme le reste du fichier : une famille a moitie declaree (libelle
// vide, nature inventee) produirait un rendu neutre qu on ne saurait pas distinguer d une
// famille non declaree.
func parseVehicleFamilies(path string, entries []vehicleFamilyEntry) (map[string]VehicleFamily, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make(map[string]VehicleFamily, len(entries))
	for i, e := range entries {
		fam := strings.TrimSpace(e.Family)
		if fam == "" {
			return nil, fmt.Errorf("%s: [[vehicle_families]] #%d sans `family`", path, i)
		}
		if _, dup := out[fam]; dup {
			return nil, fmt.Errorf("%s: [[vehicle_families]] famille %q declaree deux fois", path, fam)
		}
		en, fr := strings.TrimSpace(e.En), strings.TrimSpace(e.Fr)
		if en == "" || fr == "" {
			return nil, fmt.Errorf(
				"%s: [[vehicle_families]] %q — `en` et `fr` sont obligatoires (recu en=%q fr=%q)",
				path, fam, e.En, e.Fr)
		}
		kind := strings.TrimSpace(e.Kind)
		if !vehicleFamilyKinds[kind] {
			return nil, fmt.Errorf("%s: [[vehicle_families]] %q — `kind` %q hors de la liste fermee %v",
				path, fam, e.Kind, vehicleFamilyKindsTriees())
		}
		out[fam] = VehicleFamily{Family: fam, En: en, Fr: fr, Kind: kind, Sprite: e.Sprite}
	}
	return out, nil
}

// vehicleFamilyKindsTriees rend les natures admises, triees — un message d erreur dont l ordre
// change d une execution a l autre ne se compare pas en test.
func vehicleFamilyKindsTriees() []string {
	out := make([]string, 0, len(vehicleFamilyKinds))
	for k := range vehicleFamilyKinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
