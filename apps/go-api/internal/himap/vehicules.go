// Package himap — vehicules.go : la chaine `vehi -> hlmt -> mode` d'un vehicule pilotable.
//
// C'est le SEUL maillon neuf de la voie « sprite vue de dessus » (scout S2,
// `.ai/V7.5/film_re/SCOUT_SPRITES_VEHICULES_2026-08-31.md`). Il ne lit AUCUN offset propre
// au tag `vehi` : un vehicule reference son modele `hlmt` par un GlobalID inline, exactement
// comme un `bloc`/`scen`/`mach` Forge, et `modeleDuHlmt` (cuisson_forge.go) sait deja suivre
// ce saut jusqu'au render_model. Le `vehi` joue donc le role du tag d'objet, sans code
// specifique — le balayage d'octets contre l'index est title-agnostic et robuste au layout.
package himap

import (
	"context"
	"log/slog"
)

// GroupeVehi : le groupe de tag d'un vehicule pilotable (`vehicle`).
const GroupeVehi = "vehi"

// profondeurMaxHlmt borne la descente dans les hlmt imbriques. Un modele de vehicule
// s'organise en un hlmt PARENT (composite) qui reference des hlmt ENFANTS porteurs de la
// geometrie — mesure sur le Ghost (2026-08-31 : hlmt 0x3b3038e6 = 12 hlmt enfants, aucun
// `mode` direct). Six niveaux couvrent largement les cas observes sans risque de boucle.
const profondeurMaxHlmt = 6

// RefModeleVehicule resout le tag de geometrie de rendu d'un vehicule : `vehi` -> `hlmt`
// (eventuellement imbriques) -> `mode` (render_model) ou, a defaut, `rtgo`. Rend le GlobalID
// du modele et son groupe.
//
// On suit les hlmt DANS L'ORDRE (premier = variante principale, convention `RefsInline`) et on
// EPUISE le premier hlmt — enfants compris — avant de passer au suivant. C'est ce qui evite le
// piege du hlmt de repli partage (id bas 0x1f) qui porte un `mode` parasite : le vrai modele du
// vehicule, atteint par recursion sur le premier hlmt, prime.
func RefModeleVehicule(ctx context.Context, idx *ModuleIndex, tagVehi []byte) (id uint32, groupe string, ok bool) {
	// LA CHAINE hlmt D'ABORD, ref directe seulement en repli. Un `vehi` inline SOUVENT un `mode`
	// de repli d'ETAT (le modele « partial_emp », 0x000027d0, partage par une dizaine de vehis) :
	// le preferer ecraserait le vrai chassis. Le modele reel est TOUJOURS au bout du champ
	// `model` -> `hlmt` -> `mode`. On epuise donc la chaine hlmt (modeleParHlmtRecursif prend le
	// premier hlmt = le champ model), et on ne retombe sur une ref `rtgo`/`mode` directe (comme le
	// `food` Forge) que si aucun hlmt ne porte de geometrie.
	vus := map[uint32]bool{}
	if id, g, ok := modeleParHlmtRecursif(ctx, idx, tagVehi, vus, 0); ok {
		return id, g, true
	}
	if refs := refsVehicule(tagVehi, idx, GroupeRtgo); len(refs) > 0 {
		return refs[0], GroupeRtgo, true
	}
	if refs := refsVehicule(tagVehi, idx, GroupeMode); len(refs) > 0 {
		return refs[0], GroupeMode, true
	}
	return 0, "", false
}

// minTagIDVehicule : plancher modeste des GlobalID acceptes comme reference d'un tag de
// vehicule — il ecarte le bruit de champ-compteur (valeurs 0..255 lues comme un id par le
// balayage d'octets), pas les vrais modeles.
//
// HISTOIRE (2026-08-31 -> 2026-09-01). V4 avait pose ce plancher a 0x10000 pour ecarter DEUX
// tags PARTAGES a ID minuscule qui deraillaient la resolution : `hlmt 0x0000001f` (octets
// « 1f 00 00 00 » = l'entier 31, omnipresent) capte comme « premier hlmt », et son `mode
// 0x00003a73` de repli (modele plat de 452 sections). Mais 0x10000 etait TROP HAUT : il
// ecartait aussi des hlmt de tourelle legitimes a petit hash (mesure 2026-09-01 : la tourelle
// `warthog_g` a pour modele `hlmt 0x0000e0d4` = 57 556, ecrase par le plancher, d'ou une
// tourelle « SANS MODELE »). La bonne cible n'est pas un plancher haut mais l'exclusion NOMMEE
// des deux parasites (parasitesVehicule) : ainsi les hlmt de tourelle a petit hash passent.
const minTagIDVehicule = 0x00000100

// parasitesVehicule : les GlobalID PARTAGES de repli qui derailleraient le balayage d'octets —
// le `hlmt` omnipresent 0x1f (entier 31 present dans tout tag) et le `mode` plat 0x3a73 qu'il
// porte. Exclus NOMMEMENT (et non par un plancher aveugle) pour ne pas ecarter du meme coup les
// modeles de tourelle a petit hash.
var parasitesVehicule = map[uint32]bool{
	0x0000001f: true, // hlmt de repli partage
	0x00003a73: true, // mode plat de repli (452 sections)
	0x000027d0: true, // mode d'etat partage « vehicle_partial_emp » (~10 vehis y pointent)
}

// refsVehicule rend les refs inline d'un groupe dont le GlobalID passe le plancher anti-bruit
// et n'est pas un parasite partage.
func refsVehicule(tag []byte, idx *ModuleIndex, groupe string) []uint32 {
	return RefsInline(tag, func(h uint32) bool {
		if h < minTagIDVehicule || parasitesVehicule[h] {
			return false
		}
		g, _, ok := idx.Lookup(h)
		return ok && g == groupe
	})
}

// modeleParHlmtRecursif descend les hlmt d'un tag : pour chaque hlmt (dans l'ordre), une ref
// `rtgo`/`mode` directe gagne ; sinon on recurse dans ses hlmt enfants avant de passer au hlmt
// suivant. L'ensemble `vus` coupe les cycles et les modeles partages deja explores.
func modeleParHlmtRecursif(ctx context.Context, idx *ModuleIndex, tag []byte, vus map[uint32]bool, prof int) (uint32, string, bool) {
	if prof > profondeurMaxHlmt {
		return 0, "", false
	}
	for _, h := range refsVehicule(tag, idx, GroupeHlmt) {
		if vus[h] {
			continue
		}
		vus[h] = true
		hlmt, err := idx.Extract(h)
		if err != nil {
			slog.DebugContext(ctx, "tag hlmt illisible", "id", h, "err", err)
			continue
		}
		if refs := refsVehicule(hlmt, idx, GroupeRtgo); len(refs) > 0 {
			return refs[0], GroupeRtgo, true
		}
		if refs := refsVehicule(hlmt, idx, GroupeMode); len(refs) > 0 {
			return refs[0], GroupeMode, true
		}
		if id, g, ok := modeleParHlmtRecursif(ctx, idx, hlmt, vus, prof+1); ok {
			return id, g, ok
		}
	}
	return 0, "", false
}

// EntreesDuGroupe rend les GlobalID de tous les tags d'un groupe indexes, dans un ordre non
// garanti (l'appelant trie s'il lui faut un ordre stable). Sert a enumerer les `vehi`.
func (idx *ModuleIndex) EntreesDuGroupe(groupe string) []uint32 {
	var out []uint32
	for id, e := range idx.parID {
		if e.fichier.Group == groupe {
			out = append(out, id)
		}
	}
	return out
}
