package service

// session_page_usage_labels.go — LE NOM DE L'ARME derrière la clé de famille du bloc
// « contrôle des armes spéciales » de la page Sessions.
//
// LE PROBLÈME. Le résumé d'usage ventile les prises de socle par famille d'ARME, et la
// clé de famille est celle du film : huit chiffres hexadécimaux
// (`replay.PadWeaponFamilyKey`). Le sidecar ne stocke QUE cette clé — il n'a jamais eu de
// nom à stocker. Faute de résolution, le client affichait « Famille d'arme a2b3c4d5 » :
// une ligne de graphe identifiée par un identifiant machine, illisible par construction
// (revue utilisateur 2026-09-09, D7).
//
// POURQUOI À LA REQUÊTE ET NON AU BUILD — même arbitrage, mot pour mot, que
// `replay_weapon_labels.go` : ce qui se résout d'un catalogue du titre se résout au
// service, pas dans l'artefact. Les sidecars déjà cuits (toute la production) resteraient
// muets jusqu'à une re-cuisson complète, et le dépôt ne stocke jamais une résolution qui
// peut s'améliorer.
//
// UNE FAMILLE INCONNUE GARDE SA CLÉ, et c'est la règle du chantier rejeu reprise telle
// quelle (`replay.NewLabelCatalog`) : un nom approchant se lit comme une certitude. Le
// client a son propre repli (`padFamilyFmt`), il n'a donc jamais de trou.
//
// TITLE-AGNOSTIC PAR CONSTRUCTION : le catalogue est chargé POUR LE TITRE du service
// (`replaylabels.Load` lit `config/titles/{slug}/mappings/`). Un titre sans ces tables ne
// reçoit aucun nom — absence propre, jamais le nom d'un autre titre.

import (
	"context"
	"log/slog"
	"strconv"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/replaylabels"
)

// resolvePadFamilyLabels nomme les familles d'arme d'un bloc usage, dans la langue de la
// requête.
//
// Best-effort ET DIT : un catalogue illisible est une erreur de CONFIGURATION, pas un
// bloc sans armes — le bloc se sert entier avec ses clés, mais le journal doit distinguer
// les deux (même règle que la table d'objectifs et que le catalogue du rejeu).
//
// Sans repoRoot (service monté sans wiring de catalogue : tests unitaires du service,
// montages partiels) la fonction ne fait RIEN et ne journalise rien — l'absence de
// catalogue n'est pas une anomalie, c'est un montage sans cette capacité.
func (s *SessionPageService) resolvePadFamilyLabels(
	ctx context.Context, block *domain.SessionUsageBlock, locale string,
) {
	if s.repoRoot == "" || block == nil || len(block.PadFamilies) == 0 {
		return
	}
	cat, err := replaylabels.Load(s.repoRoot, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "session page: catalogue d'armes illisible — familles de socle non nommées",
			"err", err, "titleSlug", s.titleSlug, "families", len(block.PadFamilies))
		return
	}
	if len(cat.Weapons) == 0 {
		return
	}
	frPreferred := locale != "en"
	named := 0
	for i := range block.PadFamilies {
		family, ok := parseWeaponFamilyKey(block.PadFamilies[i].FamilyKey)
		if !ok {
			continue
		}
		label, ok := cat.Weapons[family]
		if !ok {
			continue
		}
		name := label.En
		if frPreferred && label.Fr != "" {
			name = label.Fr
		}
		if name == "" {
			continue
		}
		block.PadFamilies[i].FamilyLabel = name
		named++
	}
	if named < len(block.PadFamilies) {
		// Une famille hors catalogue n'est pas une panne : elle se voit à l'écran sous sa
		// clé, et le compte permet de suivre si le catalogue du titre prend du retard.
		slog.DebugContext(ctx, "session page: familles de socle sans nom au catalogue",
			"titleSlug", s.titleSlug, "named", named, "families", len(block.PadFamilies))
	}
}

// parseWeaponFamilyKey rend l'identifiant 32 bits d'une clé de famille du contrat.
//
// La clé servie est CELLE DE `replay.PadWeaponFamilyKey` — huit hexadécimaux minuscules,
// sans « 0x » ; c'est cette forme, et elle seule, qui indexe `LabelCatalog.Weapons`
// (`uint32`). On ne re-teste pas ici ce qu'est une famille (la frontière socle d'ARME /
// socle de BONUS appartient à `replay`) : une clé qui ne se parse pas n'est simplement
// pas nommée.
func parseWeaponFamilyKey(key string) (uint32, bool) {
	v, err := strconv.ParseUint(key, 16, 32)
	if err != nil {
		return 0, false
	}
	return uint32(v), true
}
