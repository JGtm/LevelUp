// Package duckdb — compare_repo_weapons.go : LE SCOPE DU PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 2, amendé au lot 3-bis).
//
// Fichier séparé de compare_repo.go, qui porte déjà les six lectures de la page : la frontière
// suit la donnée (le scope d'armes d'un joueur) et garde les deux fichiers loin du plafond de
// 500 lignes du dépôt.
//
// # UN SEUL SCOPE, ET C'EST UN RÉSULTAT, PAS UNE SIMPLIFICATION (lot 3-bis, 2026-09-17)
//
// Le plan prévoyait un SECOND scope, « croisé » : les matchs communs à A et B, pour un B non
// suivi localement. Il a été retiré parce qu'il ne pouvait JAMAIS être atteint. Sa requête
// lisait la même table `match_participants`, avec la même clause d'exclusion de campagne
// (portée sur `a.match_id`, égal à `b.match_id` par la jointure), et n'y ajoutait qu'un
// `EXISTS(a.xuid = A)`. Son résultat était donc un SOUS-ENSEMBLE strict du scope ci-dessous :
// « croisé non vide » impliquait « lifetime non vide », et la branche de repli qui l'appelait
// était morte — avec un test vert qui entretenait l'illusion (anti-pattern n°1 du dépôt).
//
// # UNE SEULE REQUÊTE, ET C'EST UNE CONTRAINTE DE JUSTESSE, PAS DE PERFORMANCE
//
// Elle rend À LA FOIS la liste des match_id et les totaux du joueur dessus. Les séparer en
// deux requêtes les ferait porter sur deux instantanés de la base partagée : entre les deux
// lectures, un sync peut écrire. La couverture publiée (« N frags mesurés sur M ») afficherait
// alors un dénominateur qui ne décrit plus le même corpus que son numérateur.
//
// # LECTURE SEULE, PAR LE SharedReader (ADR 0013/0016)
//
// `SharedReadDB().Get` et rien d'autre : le serveur peut tenir `shared_matches_v2.duckdb` en
// RW, et un `OpenReadOnly` forcé sur le même fichier dans le même process viole le modèle
// mono-writer. Aucune écriture ici, aucun agrégat append-only lu en direct.
package duckdb

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/domain"
)

// compareWeaponScopeTimeout : le budget de la lecture. Aligné sur GetLocalStats (15 s), qui
// balaie la même table pour le même joueur — un budget plus court ferait tomber le profil
// d'armes sur les carrières que les métriques de la page servent encore.
const compareWeaponScopeTimeout = 15 * time.Second

// GetWeaponScope rend le scope LIFETIME d'un joueur : tous ses matchs, campagne exclue, et ses
// totaux dessus.
//
// MÊMES CLAUSES QUE GetLocalStats, et c'est le point : deux définitions du « lifetime » sur la
// même page en feraient diverger les nombres à la première évolution de l'une. L'exclusion de
// campagne passe par `excludeCampaignByMatchID`, source unique du dépôt (no-op sur un titre
// sans mode masqué).
//
// ZÉRO MATCH REND (nil, nil) ET JAMAIS UN SCOPE VIDE : un scope vide passerait aux deux
// lecteurs d'armes, dont le `Validate()` refuse une liste de matchs vide — l'appelant
// recevrait une erreur de filtres là où la vérité est « ce joueur n'a aucun match ici ».
func (r *CompareRepo) GetWeaponScope(
	ctx context.Context, xuid, titleSlug string,
) (*domain.CompareWeaponScope, error) {
	ctx, cancel := context.WithTimeout(ctx, compareWeaponScopeTimeout)
	defer cancel()

	q := `
		SELECT
			mp.match_id,
			COALESCE(mp.kills, 0), COALESCE(mp.deaths, 0),
			COALESCE(mp.melee_kills, 0), COALESCE(mp.grenade_kills, 0)
		FROM match_participants mp
		WHERE mp.xuid = ?` + excludeCampaignByMatchID(titleSlug, "mp.match_id")

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetWeaponScope: shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, xuid)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.GetWeaponScope: %w", err)
	}
	defer func() { _ = rows.Close() }()

	scope := &domain.CompareWeaponScope{}
	for rows.Next() {
		var matchID string
		var kills, deaths, melee, grenade int
		if err := rows.Scan(&matchID, &kills, &deaths, &melee, &grenade); err != nil {
			return nil, fmt.Errorf("CompareRepo.GetWeaponScope: scan: %w", err)
		}
		scope.MatchIDs = append(scope.MatchIDs, matchID)
		scope.Kills += kills
		scope.Deaths += deaths
		scope.MeleeKills += melee
		scope.GrenadeKills += grenade
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CompareRepo.GetWeaponScope: rows: %w", err)
	}
	if len(scope.MatchIDs) == 0 {
		return nil, nil
	}
	scope.Matches = len(scope.MatchIDs)
	return scope, nil
}
