package service

// replay_weapon_labels.go — CE QUE LE TITRE SAIT DE CHAQUE ARME DU DOCUMENT, posé À LA
// REQUÊTE : sa CLÉ canonique et la NATURE de sa décharge.
//
// CE QUE LA CLÉ OUVRE. Un tir décodé du film porte un identifiant d'arme 64 bits ; les
// tables que le CLIENT tient — la banque de sons du rejeu — sont keyées par weapon_key,
// comme le sont déjà les effets de mort (`killEffects`). Sans jointure entre les deux, un
// tir ne peut pas sonner l'arme qui l'a produit, et lui prêter le son d'une voisine serait
// un mensonge sonore : la règle du chantier est le silence propre.
//
// CE QUE LA TEINTE OUVRE. L'éclair de bouche dit la NATURE de ce qui sort du canon (poudre,
// plasma froid ou chaud, énergie Forerunner, arc, cristal, déflagration) — décision
// utilisateur du 2026-08-15, qui ROUVRE l'arbitrage du lot 3.2. La table vit dans le TITRE
// (`replay_labels.toml`, `[shot_tints]`), la COULEUR dans le thème du client : le serveur
// ne publie jamais une couleur, seulement ce que l'arme EST.
//
// POURQUOI À LA REQUÊTE ET NON AU BUILD. C'est le patron déjà posé par `mapObjectives` :
// ce qui se résout d'un catalogue du titre se résout au service, pas dans l'artefact.
// La mesure tranche pour de bon — 23 artefacts locaux et toute la production sont déjà
// cuits ; une clé figée au build les laisserait muets jusqu'à une re-cuisson complète.
// Et la règle du dépôt le dit : on ne stocke jamais une résolution qui peut s'améliorer.
//
// TITLE-AGNOSTIC PAR CONSTRUCTION : le catalogue est chargé POUR LE TITRE du service
// (`replaylabels.Load(repoRoot, slug)` lit `config/titles/{slug}/mappings/`), exactement
// comme le fait l'assemblage hors ligne. Un titre sans ces tables ne reçoit aucune clé —
// absence propre, jamais une erreur, jamais la clé d'un autre titre.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
)

// resolveWeaponLabels pose `Key` et `Tint` sur chaque libellé d'arme du document.
//
// Best-effort ET DIT : un catalogue illisible est une erreur de configuration, pas un
// document sans armes — le rejeu se sert entier, mais le journal doit les distinguer
// (même règle que la table d'objectifs).
func (s *replayService) resolveWeaponLabels(ctx context.Context, doc *replay.ReplayDocument) {
	if len(doc.WeaponLabels) == 0 {
		return
	}
	cat, err := replaylabels.Load(s.repoRoot, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue d'armes illisible — ni son ni teinte de tir",
			"err", err, "titleSlug", s.titleSlug)
		return
	}
	if len(cat.Keys) == 0 {
		return
	}
	for id, lbl := range doc.WeaponLabels {
		family, ok := replay.FamilyOfWeaponID(id)
		if !ok {
			continue
		}
		key, ok := cat.Keys[family]
		if !ok {
			// Arme hors registre : elle garde son libellé, reste MUETTE et sans teinte.
			// Emprunter la clé d'une famille voisine lui donnerait le son d'une autre arme.
			continue
		}
		lbl.Key = key
		// Une arme sans teinte déclarée (mêlée, arme non classée) garde la teinte neutre
		// du thème : la table est partielle par nature, comme celle des effets.
		lbl.Tint = cat.Tints[key]
		doc.WeaponLabels[id] = lbl
	}
}
