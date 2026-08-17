package replay

// identity.go — QUI EST QUI DANS LE FILM, et comment on le nomme.
//
// Ce fichier rassemble les quatre lectures qui donnent un NOM à ce que le décodage produit :
// l'identité du porteur d'une trace, le gamertag que le film écrit lui-même, le roster, et le
// libellé des armes. Elles sont sorties de build.go parce que l'assemblage y dépassait le
// seuil de 500 lignes, et parce qu'elles forment un sujet : traduire des identifiants en noms.
//
// UNE RÈGLE LES GOUVERNE TOUTES : ce qui n'est pas lu reste vide. Pas d'« inconnu » posé à la
// place d'une identité, pas de nom d'arme approchant, pas de propriétaire emprunté au slot
// voisin. C'est la règle qui a fait supprimer le vote de ce chantier — mieux vaut ne rien
// afficher que quelque chose de faux — et elle vaut aussi pour les libellés : un mot faux se
// lit comme une certitude, un identifiant brut non.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// nameTracks pose le xuid du porteur sur chaque trace dont le slot est ponté.
//
// UNE TRACE SANS XUID RESTE SANS XUID : le champ est vide, pas rempli d'un « inconnu » ni du
// porteur d'un slot voisin. C'est la même règle que celle qui a fait supprimer le vote — mieux
// vaut ne rien afficher que quelque chose de faux.
func nameTracks(tracks []Track, slotXUID map[uint32]uint64) {
	if len(slotXUID) == 0 {
		return
	}
	for i := range tracks {
		if x, ok := slotXUID[tracks[i].Slot]; ok {
			tracks[i].XUID = strconv.FormatUint(x, 10)
		}
	}
}

// gamertagsOf relève les noms que LE FILM porte, un par identité.
//
// EN CAS DE DIVERGENCE, LE PREMIER GAGNE — et il n'y a rien à arbitrer ici : les 60 octets
// d'un enregistrement d'événement portent le même champ de 32 octets pour un même xuid. Si
// deux lectures divergeaient, ce serait une lecture fausse, pas un désaccord ; le nom n'entre
// dans aucun rattachement, la conséquence resterait cosmétique.
func gamertagsOf(deaths []Death) map[uint64]string {
	out := map[uint64]string{}
	for _, d := range deaths {
		if d.Gamertag == "" {
			continue
		}
		if _, seen := out[d.XUID]; !seen {
			out[d.XUID] = d.Gamertag
		}
	}
	return out
}

// buildWeaponLabels nomme les identifiants d'arme que le document emploie réellement.
//
// UNE ARME NON CATALOGUÉE N'ENTRE PAS DANS LA TABLE : elle gardera son hexadécimal à l'écran.
// C'est la règle « on ne stocke jamais une résolution qui peut s'améliorer » — un nom approchant
// serait pire qu'un identifiant, parce qu'il se lit comme une certitude.
//
// LE CATALOGUE VIENT DU TITRE (cf. catalog.go), plus de l'enum d'armes du décodeur : le
// nom affiché a une source unique, `weapon_names.toml`, et il est bilingue.
//
// LES SOCLES Y ENTRENT AU MÊME TITRE QUE LES ARMES PORTÉES (schéma 11) : la famille d'un socle
// s'écrit comme celle d'un loadout, et sans cette table le calque afficherait un hexadécimal là
// où le reste du rejeu nomme l'arme. Une famille vue SEULEMENT au sol se nomme donc aussi.
func buildWeaponLabels(
	loadouts []Loadout, shots []Shot, pads []WeaponPad, cat LabelCatalog,
) map[string]WeaponLabel {
	out := map[string]WeaponLabel{}
	add := func(id string) {
		if id == "" {
			return
		}
		if _, seen := out[id]; seen {
			return
		}
		high, ok := FamilyOfWeaponID(id)
		if !ok {
			return
		}
		if lbl, ok := cat.Weapons[high]; ok {
			// L'icône suit le libellé quand la couche titre en pointe une pour la famille ;
			// une famille nommée sans visuel garde son texte (repli du client).
			if ic, okIc := cat.Icons[high]; okIc {
				lbl.Img, lbl.Tinted = ic.URL, ic.Tinted
			}
			out[id] = lbl
		}
	}
	for _, l := range loadouts {
		for _, w := range l.W {
			add(w)
		}
	}
	for _, s := range shots {
		add(s.Weapon)
	}
	for _, p := range pads {
		add(p.Weapon)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// FamilyOfWeaponID rend la FAMILLE (high-32) d'un identifiant d'arme tel que le document
// les écrit : soit une famille seule (8 chiffres hexadécimaux après « 0x »), soit un
// identifiant GLOBAL 64 bits (16 chiffres) dont la famille est la moitié haute.
//
// Exporté parce que DEUX lecteurs en ont besoin — la table de libellés construite ici, et
// la résolution du weapon_key à la requête (service/replay_weapon_keys.go). La règle des
// deux copies vaut aussi pour trois lignes de décodage : la troisième dérive.
//
// `ok` faux = l'identifiant n'est pas lisible ; l'appelant ne nomme alors rien.
func FamilyOfWeaponID(id string) (uint32, bool) {
	var v uint64
	if _, err := fmt.Sscanf(strings.TrimPrefix(id, "0x"), "%X", &v); err != nil {
		return 0, false
	}
	if len(id) > 10 {
		return uint32(v >> 32), true
	}
	return uint32(v), true
}

// buildRoster publie les joueurs du film, triés par index pour que l'artefact soit
// reproductible : l'ordre d'itération d'une map Go est aléatoire, et un artefact qui change
// d'octets à chaque build sans changer de contenu est indiffable.
func buildRoster(idx PlayerIndexTable, names map[uint64]string) []RosterEntry {
	if len(idx.ByXUID) == 0 {
		return nil
	}
	out := make([]RosterEntry, 0, len(idx.ByXUID))
	for x, pi := range idx.ByXUID {
		out = append(out, RosterEntry{
			XUID: strconv.FormatUint(x, 10), FilmIndex: pi, Name: names[x],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FilmIndex != out[j].FilmIndex {
			return out[i].FilmIndex < out[j].FilmIndex
		}
		return out[i].XUID < out[j].XUID
	})
	return out
}
