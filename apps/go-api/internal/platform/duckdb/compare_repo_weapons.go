// Package duckdb — compare_repo_weapons.go : LES DEUX SCOPES DU PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md, lot 2).
//
// Fichier séparé de compare_repo.go, qui porte déjà les six lectures de la page : la frontière
// suit la donnée (le scope d'armes d'un côté) et garde les deux fichiers loin du plafond de
// 500 lignes du dépôt.
//
// # UNE SEULE REQUÊTE PAR SCOPE, ET C'EST UNE CONTRAINTE DE JUSTESSE, PAS DE PERFORMANCE
//
// Chaque méthode rend À LA FOIS la liste des match_id et les totaux du joueur dessus. Les
// séparer en deux requêtes les ferait porter sur deux instantanés de la base partagée : entre
// les deux lectures, un sync peut écrire. La couverture publiée (« N frags mesurés sur M »)
// afficherait alors un dénominateur qui ne décrit plus le même corpus que son numérateur.
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

// compareWeaponScopeTimeout : le budget des deux lectures. Aligné sur GetLocalStats (15 s),
// qui balaie la même table pour le même joueur — un budget plus court ferait tomber le profil
// d'armes sur les carrières que les métriques de la page servent encore.
const compareWeaponScopeTimeout = 15 * time.Second

// GetWeaponScope rend le scope LIFETIME d'un joueur : tous ses matchs, campagne exclue.
//
// MÊMES CLAUSES QUE GetLocalStats, et c'est le point : deux définitions du « lifetime » sur la
// même page en feraient diverger les nombres à la première évolution de l'une. L'exclusion de
// campagne passe par `excludeCampaignByMatchID`, source unique du dépôt (no-op sur un titre
// sans mode masqué).
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

	return r.scanWeaponScope(ctx, "GetWeaponScope", q, xuid)
}

// GetCrossWeaponScope rend le scope CROISÉ : les matchs communs à xuidA et xuidB, et les
// totaux de xuidB sur ces matchs seuls.
//
// # L'AUTO-JOINTURE EST CELLE DE GetCrossMatchSample, L'EXCLUSION DE CAMPAGNE NE L'EST PAS
//
// `GetCrossMatchSample` ne filtre pas la campagne (constat du 2026-09-17, consigné au journal
// du plan). Ici elle est appliquée, délibérément : le côté A du même profil d'armes est
// mesuré campagne exclue, et deux scopes aux règles différentes rendraient les deux colonnes
// non comparables — ce qui est exactement ce que la page prétend faire. La clause porte sur
// `a.match_id` : les deux joueurs partagent le match, filtrer un côté les filtre tous deux.
func (r *CompareRepo) GetCrossWeaponScope(
	ctx context.Context, xuidA, xuidB, titleSlug string,
) (*domain.CompareWeaponScope, error) {
	ctx, cancel := context.WithTimeout(ctx, compareWeaponScopeTimeout)
	defer cancel()

	q := `
		SELECT
			b.match_id,
			COALESCE(b.kills, 0), COALESCE(b.deaths, 0),
			COALESCE(b.melee_kills, 0), COALESCE(b.grenade_kills, 0)
		FROM match_participants a
		JOIN match_participants b ON b.match_id = a.match_id AND b.xuid = ?
		WHERE a.xuid = ?` + excludeCampaignByMatchID(titleSlug, "a.match_id")

	return r.scanWeaponScope(ctx, "GetCrossWeaponScope", q, xuidB, xuidA)
}

// scanWeaponScope exécute une requête « une ligne par match du scope » et la replie en
// CompareWeaponScope.
//
// EXTRAIT PARCE QUE LES DEUX SCOPES NE DIFFÈRENT QUE PAR LEUR CLAUSE FROM/WHERE : le repli, le
// régime d'erreur et la convention « zéro match -> (nil, nil) » sont les mêmes, et les
// recopier les ferait diverger (règle des copies du dépôt).
//
// ZÉRO MATCH REND (nil, nil) ET JAMAIS UN SCOPE VIDE : un scope vide passerait les deux
// lecteurs d'armes, dont le `Validate()` refuse une liste de matchs vide — l'appelant
// recevrait une erreur de filtres là où la vérité est « ce joueur n'a aucun match ici ».
func (r *CompareRepo) scanWeaponScope(
	ctx context.Context, methode, query string, args ...any,
) (*domain.CompareWeaponScope, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.%s: shared reader: %w", methode, err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("CompareRepo.%s: %w", methode, err)
	}
	defer func() { _ = rows.Close() }()

	scope := &domain.CompareWeaponScope{}
	for rows.Next() {
		var matchID string
		var kills, deaths, melee, grenade int
		if err := rows.Scan(&matchID, &kills, &deaths, &melee, &grenade); err != nil {
			return nil, fmt.Errorf("CompareRepo.%s: scan: %w", methode, err)
		}
		scope.MatchIDs = append(scope.MatchIDs, matchID)
		scope.Kills += kills
		scope.Deaths += deaths
		scope.MeleeKills += melee
		scope.GrenadeKills += grenade
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("CompareRepo.%s: rows: %w", methode, err)
	}
	if len(scope.MatchIDs) == 0 {
		return nil, nil
	}
	scope.Matches = len(scope.MatchIDs)
	return scope, nil
}
