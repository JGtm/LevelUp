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
	// Une ref `rtgo`/`mode` directe sur le vehi d'abord (comme le `food` Forge), sinon le saut
	// par les hlmt.
	if refs := refsVehicule(tagVehi, idx, GroupeRtgo); len(refs) > 0 {
		return refs[0], GroupeRtgo, true
	}
	if refs := refsVehicule(tagVehi, idx, GroupeMode); len(refs) > 0 {
		return refs[0], GroupeMode, true
	}
	vus := map[uint32]bool{}
	return modeleParHlmtRecursif(ctx, idx, tagVehi, vus, 0)
}

// minTagIDVehicule : plancher des GlobalID acceptes comme reference d'un tag de vehicule.
//
// POURQUOI, ET C'EST MESURE (2026-08-31). Le balayage d'octets (`RefsInline`) prend pour une
// reference toute valeur 4 octets qui resout dans l'index. Deux tags PARTAGES a ID minuscule
// derailent la resolution : `hlmt 0x0000001f` (octets « 1f 00 00 00 » = l'entier 31, omnipresent)
// capte comme « premier hlmt », et il porte un `mode 0x00003a73` de repli — d'ou 15 vehicules
// (ghost, banshee, wraith, chopper...) resolus vers CE meme modele parasite plat de 452 sections.
// Tous les vrais modeles de vehicule ont un GlobalID de HASH large (>= 0x06000000 sur les cas
// resolus) ; ecarter les ID sous ce plancher supprime les deux parasites sans toucher un seul
// modele reel. Le filtre est PROPRE au vehicule (le Forge, lui, n'a pas ce piege).
const minTagIDVehicule = 0x00010000

// refsVehicule rend les refs inline d'un groupe dont le GlobalID passe le plancher anti-parasite.
func refsVehicule(tag []byte, idx *ModuleIndex, groupe string) []uint32 {
	return RefsInline(tag, func(h uint32) bool {
		if h < minTagIDVehicule {
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
