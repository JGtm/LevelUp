package service

// replay_destruction_causes.go — LA CLÉ D'ARME DE LA CAUSE D'UNE DESTRUCTION DE VÉHICULE, posée À LA
// REQUÊTE.
//
// CE QUE L'ARTEFACT PORTE ET CE QU'IL NE PORTE PAS. Une destruction écrit la source de dégât de sa
// cause (`VehicleDestruction.Cause`, un tag `jpt!`, le même espace que la source d'un frag). Il n'écrit
// jamais `CauseKey`, le weapon_key du registre d'armes du titre : c'est le patron de
// `WeaponLabel.Key` (cf. replay_weapon_labels.go), et la raison est la même — une résolution qui peut
// s'améliorer ne se stocke pas, et la figer au build laisserait muets les artefacts déjà cuits.
//
// LA MÊME RÉSOLUTION QUE LES LIGNES D'ÉLIMINATION. La clé est celle que `KillSourceIcon` de l'adapter
// d'assets du titre rend pour la source du dégât : c'est elle que la Match View publie comme
// `weaponKey` d'une élimination, et la page joint la destruction au pictogramme de l'arme sur cette
// clé. Une cause que le titre n'identifie pas sans ambiguïté, ou dont l'arme n'a ni clé de registre
// ni image (une grenade, la mêlée, une chute), ne reçoit AUCUNE clé : silence propre, jamais la clé
// d'une arme voisine.
//
// TITLE-AGNOSTIC PAR CONSTRUCTION : le résolveur est celui du titre du service, injecté au câblage ;
// aucune comparaison de slug. Un titre qui n'en fournit pas ne reçoit aucune clé.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// ReplayOption est une capacité facultative du titre, donnée à la construction du service de rejeu.
type ReplayOption func(*replayService)

// WithReplayKillSourceIcons donne au service le résolveur de source de dégât du titre : il pose la
// clé d'arme de la cause de chaque destruction de véhicule. Nil = aucune clé.
func WithReplayKillSourceIcons(r port.KillSourceIconResolver) ReplayOption {
	return func(s *replayService) { s.killIcons = r }
}

func applyReplayOptions(s *replayService, opts []ReplayOption) *replayService {
	for _, o := range opts {
		o(s)
	}
	return s
}

// resolveDestructionCauses pose `CauseKey` sur chaque destruction qui porte une cause, et le vide
// sur toutes les autres : le champ appartient à la requête, jamais à ce que l'artefact a pu contenir.
func (s *replayService) resolveDestructionCauses(ctx context.Context, doc *replay.ReplayDocument) {
	if doc == nil {
		return
	}
	resolved, unresolved := 0, 0
	for i := range doc.Vehicles {
		d := doc.Vehicles[i].Destruction
		if d == nil {
			continue
		}
		d.CauseKey = ""
		if d.Cause == nil {
			continue
		}
		if key := s.weaponKeyOfCause(*d.Cause); key != "" {
			d.CauseKey = key
			resolved++
			continue
		}
		unresolved++
	}
	if resolved+unresolved > 0 {
		slog.DebugContext(ctx, "rejeu 2D : causes de destruction de véhicule résolues en clé d'arme",
			"resolues", resolved, "sansCle", unresolved, "titleSlug", s.titleSlug)
	}
}

// weaponKeyOfCause rend la weapon_key de la source de dégât, "" quand le titre n'en a pas : pas de
// résolveur, source non identifiée, ou arme sans clé de registre ni image (le pictogramme que la page
// retrouve par cette clé n'existerait pas).
func (s *replayService) weaponKeyOfCause(sourceTag uint32) string {
	if s.killIcons == nil {
		return ""
	}
	icon, ok := s.killIcons.KillSourceIcon(sourceTag)
	if !ok || icon.ImageURL == "" {
		return ""
	}
	return icon.WeaponKey
}
