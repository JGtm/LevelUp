package service

// replay_weapon_keys.go — LA CLÉ CANONIQUE DE CHAQUE ARME DU DOCUMENT, posée À LA REQUÊTE.
//
// CE QU'ELLE OUVRE. Un tir décodé du film porte un identifiant d'arme 64 bits ; les tables
// que le CLIENT tient — la banque de sons du rejeu — sont keyées par weapon_key, comme le
// sont déjà les effets de mort (`killEffects`). Sans jointure entre les deux, un tir ne
// peut pas sonner l'arme qui l'a produit, et lui prêter le son d'une voisine serait un
// mensonge sonore : la règle du chantier est le silence propre.
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

// resolveWeaponKeys pose `Key` sur chaque libellé d'arme du document.
//
// Best-effort ET DIT : un catalogue illisible est une erreur de configuration, pas un
// document sans armes — le rejeu se sert entier, mais le journal doit les distinguer
// (même règle que la table d'objectifs).
func (s *replayService) resolveWeaponKeys(ctx context.Context, doc *replay.ReplayDocument) {
	if len(doc.WeaponLabels) == 0 {
		return
	}
	cat, err := replaylabels.Load(s.repoRoot, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : catalogue d'armes illisible — aucun son de tir",
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
			// Arme hors registre : elle garde son libellé et reste MUETTE. Emprunter la
			// clé d'une famille voisine lui donnerait le son d'une autre arme.
			continue
		}
		lbl.Key = key
		doc.WeaponLabels[id] = lbl
	}
}
