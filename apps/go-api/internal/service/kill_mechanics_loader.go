// Package service — kill_mechanics_loader.go : LE CHARGEUR PARTAGÉ DES MÉCANIQUES DE KILL
// NATIVES (assassinats + capacités spartanes), par joueur et par scope de matchs.
//
// # POURQUOI CE FICHIER EXISTE (règle des copies, 2026-09-17)
//
// Ces mécaniques ne sont pas un détail d'affichage : sur un titre qui les fournit
// nativement (Halo 5), `fragdist.Build` RETRANCHE les frags de mécanique des classes d'arme
// — une mêlée y est attribuée à l'arme TENUE dans `weapon_kills` — en supposant que
// `domain.FragKillTypeCounts` les rapporte par ailleurs. Un appelant qui laisse ces trois
// compteurs à zéro ne perd donc pas seulement la classe « Capacités spartanes » : il
// sous-évalue « Mêlée » ET gonfle « Non attribué » du même volume. L'erreur est silencieuse et
// arithmétiquement cohérente — le total boucle toujours.
//
// Le chargement vivait chez l'Explorer (`loadTargetKillMechanics`). Le profil d'armes du
// Face-à-face en aurait été une SECONDE copie, et l'Escouade en porte déjà une variante : à la
// troisième, la règle n°6 du dépôt impose de centraliser. C'est fait ici, en FONCTION LIBRE —
// deux services différents l'appellent, aucun n'en est propriétaire.
//
// # CE QUI N'EST PAS CENTRALISÉ, ET POURQUOI
//
// `teammates.loadSquadMechanicsByGT` reste distinct, et ce n'est pas un oubli : il vit dans le
// paquet `service/teammates` (qui ne peut pas importer son parent `service`), passe par un
// AUTRE port (`squadLoader.LoadKillMechanics(ctx, slug, filters)`, trois arguments), charge N
// joueurs en une requête et rend une map keyée par GAMERTAG. Le plier à cette signature
// changerait son comportement pour un gain nul.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/port"
)

// killMechanicsLoader est la capability OPTIONNELLE (type-assertion, même pattern que
// lobbySizeProvider/objectiveScoreProvider côté session page) permettant de charger les
// mécaniques de kill NATIVES (assassinats + capacités spartanes) agrégées par xuid. Le repo
// weapon_kills DuckDB concret (duckdb.WeaponKillsRepo) l'implémente ; un loader qui ne
// l'implémente pas dégrade proprement (Mêlée non splittée, pas de classe « Capacités
// spartanes »).
type killMechanicsLoader interface {
	LoadKillMechanicsAggregated(ctx context.Context, filters port.WeaponKillFilters) ([]port.KillMechanicsRow, error)
}

// loadKillMechanicsForXUID rend les mécaniques natives d'UN joueur sur un scope de matchs.
//
// NIL BEST-EFFORT dans tous les cas dégradés : repo absent, repo qui ne porte pas la
// capability, xuid ou scope vide, erreur de lecture, aucune ligne. L'appelant passe alors des
// compteurs à zéro à `fragdist.Build` — ce qui est la bonne valeur pour un titre SANS
// mécaniques natives, et c'est pourquoi l'appel doit être gardé par
// `titleHasNativeKillMechanics(slug)` : sur un titre qui EN A, un zéro serait un mensonge.
//
// L'erreur est journalisée en Debug et jamais avalée : sur un titre sans la capability, elle
// est l'état nominal, et la remonter en Warn noierait les vraies anomalies.
func loadKillMechanicsForXUID(
	ctx context.Context, repo port.WeaponKillsRepository, xuid string, matchIDs []string,
) *port.KillMechanicsRow {
	loader, ok := repo.(killMechanicsLoader)
	if !ok || xuid == "" || len(matchIDs) == 0 {
		return nil
	}
	rows, err := loader.LoadKillMechanicsAggregated(ctx, port.WeaponKillFilters{
		MatchIDs: matchIDs, XUIDs: []string{xuid},
	})
	if err != nil {
		slog.DebugContext(ctx, "kill mechanics skipped (best-effort)", "xuid", xuid, "err", err)
		return nil
	}
	if len(rows) == 0 {
		return nil
	}
	// Au plus une ligne par xuid ; on somme par sûreté (parité loadSquadMechanicsByGT).
	agg := port.KillMechanicsRow{XUID: xuid}
	for _, r := range rows {
		agg.Assassinations += r.Assassinations
		agg.GroundPound += r.GroundPound
		agg.ShoulderBash += r.ShoulderBash
	}
	return &agg
}
