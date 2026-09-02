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

// nameTracksByLives pose le xuid du porteur sur chaque trace PAR VIE : une trace est nommée
// par la vie nommée du MÊME slot qui la recouvre le mieux — jamais par le slot seul.
//
// POURQUOI PAR VIE (lot identité des vies, 2026-09-02). Le nommage par slot (`SlotXUID`)
// donnait tout l'intervalle d'un slot RECYCLÉ à son premier porteur nommé : le remplaçant
// (arrivant, bot) n'existait pas et l'ancien « vivait » à sa place. Les tracks étant
// désormais découpées à la MÊME règle que les vies (`lifeGapUS`, cf. decimateTracks), la
// correspondance track <-> vie est un recouvrement franc — le meilleur recouvrement suffit,
// et un recouvrement nul ne nomme rien.
//
// UNE TRACE SANS VIE NOMMÉE RESTE SANS XUID : le champ est vide, pas rempli d'un « inconnu »
// ni du porteur d'un slot voisin. C'est la même règle que celle qui a fait supprimer le vote
// — mieux vaut ne rien afficher que quelque chose de faux.
func nameTracksByLives(tracks []Track, lives []lifeSpan, origin, step uint64) {
	if len(lives) == 0 {
		return
	}
	for i := range tracks {
		from := int64(origin) + int64(tracks[i].StartFrame)*int64(step)
		to := int64(origin) + int64(tracks[i].EndFrame)*int64(step) + int64(step) - 1
		var bestXUID uint64
		var bestOverlap int64
		for _, l := range lives {
			if l.slot != tracks[i].Slot || l.xuid == 0 {
				continue
			}
			if ov := minI64(to, l.to) - maxI64(from, l.from); ov > bestOverlap {
				bestOverlap, bestXUID = ov, l.xuid
			}
		}
		if bestXUID != 0 {
			tracks[i].XUID = strconv.FormatUint(bestXUID, 10)
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

// BotIdentity est un bot tel que le film le DÉCLARE (BOT_METADATA), réduit à ce que
// l'assemblage consomme : son index de roster et son nom d'affichage.
type BotIdentity struct {
	FilmIndex int
	Name      string
}

// buildRoster publie les joueurs du film — humains du fil des morts, puis bots déclarés —
// triés par index pour que l'artefact soit reproductible : l'ordre d'itération d'une map Go
// est aléatoire, et un artefact qui change d'octets à chaque build sans changer de contenu
// est indiffable.
//
// UN BOT DONT L'INDEX EST TENU PAR UN HUMAIN N'ENTRE PAS : les deux déclarations se
// contredisent, et le fil des morts (une lecture par xuid) l'emporte sur un paquet de
// métadonnées. DEUX BOTS PEUVENT PARTAGER UN INDEX, et c'est mesuré (RE_LOG 7ter.62 :
// « 343 Aloysius » puis « 343 PardonMy », les deux déclarant slot=8) — des remplaçants
// SUCCESSIFS sur le même siège de réplication. Ils entrent tous les deux : le nom les
// différencie, l'index dit le siège.
func buildRoster(idx PlayerIndexTable, names map[uint64]string, bots []BotIdentity) []RosterEntry {
	if len(idx.ByXUID) == 0 && len(bots) == 0 {
		return nil
	}
	out := make([]RosterEntry, 0, len(idx.ByXUID)+len(bots))
	humanIdx := make(map[int]bool, len(idx.ByXUID))
	for x, pi := range idx.ByXUID {
		humanIdx[pi] = true
		out = append(out, RosterEntry{
			XUID: strconv.FormatUint(x, 10), FilmIndex: pi, Name: names[x],
		})
	}
	seen := map[string]bool{}
	for _, b := range bots {
		if humanIdx[b.FilmIndex] || b.Name == "" || seen[b.Name] {
			continue
		}
		seen[b.Name] = true
		out = append(out, RosterEntry{FilmIndex: b.FilmIndex, Name: b.Name, Bot: true})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FilmIndex != out[j].FilmIndex {
			return out[i].FilmIndex < out[j].FilmIndex
		}
		return out[i].XUID < out[j].XUID
	})
	return out
}

// nameBotTracks pose le NOM d'un bot sur les vies que le pont attribue à son index.
//
// LA PORTE EST LE PONT, PAS UNE DEVINETTE : `owner` (slot -> index) ne contient un slot que
// par une lecture ou une fermeture à candidat unique (cf. owners.go / closures.go). Un bot
// n'ayant pas de xuid, ses slots n'entrent jamais par le fil des morts — seules les
// fermetures (un bot TIRE, fermeture A) peuvent les apporter. Une vie déjà nommée par un
// xuid n'est jamais écrasée : sur un slot recyclé humain -> bot, l'humain garde SES vies.
func nameBotTracks(tracks []Track, owner map[uint32]int, bots []BotIdentity) {
	if len(owner) == 0 || len(bots) == 0 {
		return
	}
	nameByIndex := make(map[int]string, len(bots))
	for _, b := range bots {
		if b.Name != "" {
			nameByIndex[b.FilmIndex] = b.Name
		}
	}
	for i := range tracks {
		if tracks[i].XUID != "" {
			continue
		}
		pi, ok := owner[tracks[i].Slot]
		if !ok {
			continue
		}
		if name, isBot := nameByIndex[pi]; isBot {
			tracks[i].Bot = name
		}
	}
}
