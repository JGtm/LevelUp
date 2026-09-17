// Package service — weapon_range_by_role.go : LA PORTÉE D'UN JOUEUR, AGRÉGÉE PAR RÔLE
// D'ARME, sur un ensemble de matchs donné.
//
// # POURQUOI CE FICHIER EXISTE
//
// Deux surfaces publient cette lecture — le profil d'armes du Face-à-face et le bloc
// « Portée des frags » de l'encart cible de l'Explorer — et la chaîne est la même de bout en
// bout : frags mesurés bornés par (matchs, xuid), clés d'arme traduites en rôles, mesures
// regroupées sous ces rôles, agrégat P10/médiane/P90 mis à la forme du contrat. La deuxième
// copie est le moment où l'on factorise (règle des ≤ 2 copies) : une chaîne recopiée diverge
// au premier réglage de seuil, et les deux pages afficheraient alors deux portées.
//
// # LE GRAIN EST LE RÔLE, PAS L'ARME
//
// C'est la différence avec `weapon_range_section.go`, qui publie une ligne par ARME pour
// l'onglet Résumé. Par famille il y aurait une trentaine de lignes presque toutes sous le
// seuil ; par classe, « lourde » mélangerait un sniper et une épée. Le rôle est le seul grain
// où le graphe se lit du contact à la longue portée.
//
// # BEST-EFFORT STRICT, ET LA DISTINCTION QUI COMPTE
//
// Aucune erreur ne remonte : un bloc absent vaut mieux qu'une page tombée. Mais une
// capability manquante (titre sans positions par kill) se journalise en Debug et une panne de
// lecture en Warn — sans quoi, sur tout titre sans décodeur de film, le Warn permanent
// noierait le jour où un vrai bug SQL arrive.
package service

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// buildWeaponRangeByRole rend la portée par rôle d'UN joueur sur UN scope, ou nil.
//
// `surface` ne sert qu'aux journaux : il dit quelle page a demandé la lecture, pour qu'un
// Warn soit rattachable sans remonter la pile.
func buildWeaponRangeByRole(
	ctx context.Context,
	repo port.WeaponRangeRepository,
	surface, slug, xuid string,
	scope weaponRangeScopeInfo,
) *domain.SynthesisWeaponRange {
	if repo == nil || xuid == "" || len(scope.matchIDs) == 0 {
		return nil
	}
	filtres := port.WeaponRangeFilters{MatchIDs: scope.matchIDs, XUIDs: []string{xuid}}
	mesures, err := repo.LoadWeaponRange(ctx, slug, filtres)
	if err != nil {
		logWeaponRangeByRoleFailure(ctx, surface, slug, xuid, len(scope.matchIDs), err)
		return nil
	}
	if len(mesures) == 0 {
		return nil
	}
	roles := resolveWeaponRolesFor(ctx, repo, surface, slug, mesures)
	regroupes, ecartes := analysis.RegroupMeasuredKills(mesures, func(cle string) string {
		return roles[cle]
	})
	if ecartes > 0 {
		// Une clé sans rôle est écartée. Debug et non Warn : un registre incomplet est un
		// état connu, pas une panne — mais un silence total en ferait un zéro.
		slog.DebugContext(ctx, "portee par role: frags mesures ecartes faute de role",
			"surface", surface, "title", slug, "xuid", xuid, "ecartes", ecartes, "gardes", len(regroupes))
	}
	if len(regroupes) == 0 {
		return nil
	}
	return buildWeaponRangeBlock(regroupes, nil, scope)
}

// resolveWeaponRolesFor traduit les clés d'arme des frags mesurés en clés de RÔLE.
//
// UNE SEULE RÉSOLUTION POUR TOUT LE LOT : les clés sont dédupliquées avant l'appel. Résoudre
// clé par clé ferait une requête registre par arme.
//
// LA CLÉ EST SON PROPRE RÔLE QUAND LE REGISTRE N'EN DONNE PAS D'AUTRE (`grenade`, `melee`,
// `sidearm`, `equipment`, `vehicle`, `turret`, `environmental` : le registre y pose
// class == role). Rien de particulier n'est fait pour ces cas — le registre rend déjà le bon
// rôle ; une liste en dur ici serait une seconde source qui divergerait du TOML du titre.
func resolveWeaponRolesFor(
	ctx context.Context, repo port.WeaponRangeRepository, surface, slug string, mesures []analysis.MeasuredKill,
) map[string]string {
	cles := make([]string, 0, len(mesures))
	vues := make(map[string]bool, len(mesures))
	for _, m := range mesures {
		if m.WeaponKey != "" && !vues[m.WeaponKey] {
			vues[m.WeaponKey] = true
			cles = append(cles, m.WeaponKey)
		}
	}
	dims, err := repo.ResolveWeaponDimensions(ctx, slug, cles)
	if err != nil {
		logBestEffortErr(ctx, "portee par role: dimensions d armes non resolues", err,
			"surface", surface, "title", slug, "cles", len(cles))
		return nil
	}
	roles := make(map[string]string, len(dims))
	for cle, d := range dims {
		if d.Role != "" {
			roles[cle] = d.Role
		}
	}
	return roles
}

// logWeaponRangeByRoleFailure distingue l'ABSENCE LÉGITIME de l'ANOMALIE — parité
// `logWeaponRangeFailure`.
func logWeaponRangeByRoleFailure(
	ctx context.Context, surface, slug, xuid string, matchCount int, err error,
) {
	if errors.Is(err, games.ErrCapabilityNotSupported) {
		slog.DebugContext(ctx, "portee par role: capability absente",
			"surface", surface, "title", slug, "xuid", xuid)
		return
	}
	slog.WarnContext(ctx, "portee par role: lecture en echec (best-effort, bloc omis)",
		"surface", surface, "title", slug, "xuid", xuid, "match_count", matchCount, "err", err)
}
