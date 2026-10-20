package service

// replay_weapon_tiers.go — LES RÉGLAGES DE NIVEAU D'ARME DU MATCH, servis AVEC le document.
//
// # CE QU'IL SERT, ET POURQUOI LE SERVEUR PLUTÔT QUE LE CLIENT
//
// Un seul fait : le mode de CE match distribue-t-il les équipements de début de vie AU HASARD ?
// De lui dépend la publication du niveau « arme de base » et la note qui explique son absence.
//
// La réponse ne se lit pas dans l'artefact — le film ne nomme pas le mode. Elle vient du
// registre (`pair_name`) croisé à la règle du titre (`regulation.toml`, `[weapon_tiers]`). Le
// client pourrait la déduire de la catégorie de mode de l'en-tête ; il l'a fait une semaine, et
// la copie a divergé (revue du 2026-09-14). Servi, il n'y a qu'une vérité — et c'est la même
// que celle qui gouverne l'écriture en base.
//
// # MÊME RÉGIME QUE LES DEUX CALQUES DE CARTE
//
// Résolu À LA REQUÊTE, jamais cuit : `SchemaVersion` ne bouge pas, et le champ figure dans
// l'allowlist des calques de requête du ratchet de forme (`document_shape_test.go`).
//
// Tout échec est une ABSENCE (champ nil), toujours journalisée, et l'absence est SÛRE : sans
// ce champ, le client ne publie pas la note « départs aléatoires » — il ne l'invente pas.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/replayartifacts"
)

// weaponTiersForKeys résout les réglages de niveau d'arme du match.
//
// Rend nil quand le titre ne déclare aucune règle OU quand le match n'a pas de mode connu : un
// `randomStarts: false` servi sur un mode inconnu affirmerait « ce mode a des armes de départ »
// là où la vérité est « on ne sait pas ».
func (s *replayService) weaponTiersForKeys(
	ctx context.Context, matchID string, keys port.MatchMapKeys,
) *replay.WeaponTiersInfo {
	if keys.PairName == "" {
		slog.DebugContext(ctx, "rejeu 2D : pas de pair_name — niveaux d'armes non renseignes",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	reg, err := replayartifacts.ReglesDepartsAleatoires(s.repoRoot, s.titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "rejeu 2D : regulation.toml illisible — niveaux d'armes non renseignes",
			"err", err, "titleSlug", s.titleSlug)
		return nil
	}
	if len(reg.RandomStartModeTokens()) == 0 {
		// Le titre ne declare aucun mode aleatoire : ce n'est pas une panne, mais rien n'est a
		// dire — le niveau « base » se mesure partout et se verifie de lui-meme.
		return nil
	}
	return &replay.WeaponTiersInfo{RandomStarts: replayartifacts.DepartsAleatoires(reg, keys.PairName)}
}
