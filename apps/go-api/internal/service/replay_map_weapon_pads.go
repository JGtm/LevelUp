package service

// replay_map_weapon_pads.go — LES EMPLACEMENTS DE SOCLE DE LA CARTE, croisés avec le match
// et servis AVEC le document de rejeu.
//
// LA CHAÎNE, et qui décide quoi :
//
//	match -> map_id (registre partagé)                   ReplayMapNameRepo (partagé, matchMapKeys)
//	map_id -> emplacements de la carte (catalogue figé)  replay.LoadMapWeaponPads
//	emplacements x socles du match -> ce qui est ALLUMÉ  replay.BuildMapWeaponPads
//
// PAS DE MODE DANS CETTE CHAÎNE, et c'est mesuré : le fichier de carte ne dit pas quel mode
// allume quel socle (ces objets ne portent aucun label sur les cartes DEV). Ce qui allume,
// c'est le MATCH lui-même — donc le film. La confirmation socle par socle remplace la table
// de rôles que les objectifs, eux, possèdent.
//
// TOUT ÉCHEC EST UNE ABSENCE (champ nil), TOUJOURS JOURNALISÉ, et l'absence est SÛRE : sans
// ce calque, le client dessine les socles du film comme il le faisait avant.

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// mapWeaponPadsForKeys résout le calque des emplacements de socle allumés du match.
//
// LE FILM EST LA CONDITION D'ENTRÉE : sans socle publié, rien ne peut être confirmé, et le
// calque est absent. C'est exactement le cas du Super Fiesta sur Cliffhanger — dix-sept
// emplacements au fichier, aucun servi en jeu, aucun artefact pour le dire : il ne doit RIEN
// s'afficher, et c'est ce court-circuit qui le garantit avant même d'ouvrir le catalogue.
func (s *replayService) mapWeaponPadsForKeys(ctx context.Context, matchID string,
	keys port.MatchMapKeys, pads []replay.WeaponPad) *replay.MapWeaponPads {
	if keys.MapID == "" {
		slog.DebugContext(ctx, "rejeu 2D : pas de map_id — pas d'emplacements de socle",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	if len(pads) == 0 {
		// Le film n'a vu aucun socle : aucun emplacement du catalogue n'est confirmé, donc
		// aucun ne part. Décision produit du 2026-08-19 : « on ne les affiche que si allumés ».
		return nil
	}
	res := title.NewPathResolver(s.repoRoot)
	cat, err := replay.LoadMapWeaponPads(res.MapWeaponPadsPath(s.titleSlug))
	if err != nil {
		// Le catalogue est VERSIONNÉ : son absence n'est pas le cas nominal d'une carte sans
		// socle — on le dit, puis on dégrade (même règle que les objectifs et les callouts).
		slog.WarnContext(ctx, "rejeu 2D : catalogue des socles illisible — pas d'emplacements servis",
			"err", err, "titleSlug", s.titleSlug)
		return nil
	}
	entry, err := cat.Lookup(keys.MapID)
	if err != nil {
		// Carte hors catalogue : absence propre, le client garde les socles du film.
		slog.DebugContext(ctx, "rejeu 2D : carte hors catalogue des socles",
			"map_id", keys.MapID, "match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	out := replay.BuildMapWeaponPads(entry, pads)
	if out == nil {
		// La carte est au catalogue mais AUCUN de ses emplacements n'est confirmé. C'est
		// anormal quand le film a vu des socles : ils sont ailleurs que là où le fichier les
		// pose — mauvaise carte jointe, ou repère décalé. Le dire, puis dégrader.
		slog.WarnContext(ctx, "rejeu 2D : aucun emplacement confirmé malgré des socles au film",
			"map_id", keys.MapID, "match_id", matchID, "socles_film", len(pads),
			"emplacements_catalogue", len(entry.Pads), "titleSlug", s.titleSlug)
		return nil
	}
	slog.DebugContext(ctx, "rejeu 2D : emplacements de socle servis",
		"map_id", keys.MapID, "match_id", matchID, "allumes", len(out.Pads),
		"catalogue", out.CatalogN, "socles_film", len(pads))
	return out
}
