package service

// replay_map_objectives.go — LES OBJECTIFS STATIQUES DU MODE JOUÉ, servis AVEC le
// document de rejeu (lot 4).
//
// LA CHAÎNE, et qui décide quoi :
//
//	match -> map_id + pair_name (registre partagé)     ReplayMapNameRepo
//	pair_name -> libellé de mode normalisé             analysis.NormalizeModeLabel
//	mode -> rôles à servir (DONNÉE du titre)           objective_roles.toml
//	map_id -> objets des rôles (catalogue versionné)   replay.LoadMapObjectives
//
// C'est le SERVEUR qui choisit les rôles servis ; le client n'affiche que ce qui
// arrive (title-agnostic, dégradation par absence). La jointure carte se fait par
// map_id SEUL — pas de cascade de noms ici : le catalogue d'objectifs est indexé par
// asset UGC, une carte Forge y a sa propre entrée quand elle a été extraite, et un
// map_id vide est une absence propre, jamais une erreur.
//
// TOUT ÉCHEC EST UNE ABSENCE (champ nil), TOUJOURS JOURNALISÉ : la page de rejeu reste
// entière sans son calque d'objectifs — mais une table illisible est une erreur de
// configuration, pas une carte sans objectifs, et le journal doit les distinguer.

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// objectiveRolesFilename est le nom du fichier de table sous mappings/ du titre.
const objectiveRolesFilename = "objective_roles.toml"

// mapObjectivesForMatch résout le calque statique des objectifs du match. Rend nil pour
// TOUTE absence (mode sans objectifs, carte hors catalogue, titre sans table...).
func (s *replayService) mapObjectivesForMatch(ctx context.Context, matchID string) *replay.MapObjectives {
	if s.maps == nil {
		return nil
	}
	keys, err := s.maps.MapKeysForMatch(ctx, matchID)
	if err != nil || keys.MapID == "" || keys.PairName == "" {
		// Sans map_id la jointure n'existe pas ; sans pair_name le mode non plus.
		slog.DebugContext(ctx, "rejeu 2D : clés de match incomplètes — pas d'objectifs statiques",
			"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	specs := s.objectiveRoleSpecs(ctx, keys.PairName)
	if len(specs) == 0 {
		// Mode sans objectifs statiques : le cas nominal. Slayer, Land Grab, Total Control —
		// PAS King of the Hill, qui sert le rôle `hill` depuis le lot C-ter volet 2 (la
		// table du titre le déclare ; l'exemple d'avant était devenu faux : revue ronde 1,
		// R1-5).
		return nil
	}
	res := title.NewPathResolver(s.repoRoot)
	cat, err := replay.LoadMapObjectives(res.MapObjectivesPath(s.titleSlug))
	if err != nil {
		// Le catalogue est VERSIONNÉ : son absence n'est pas le cas nominal d'une carte
		// sans objectifs — on le dit, puis on dégrade (même règle que les callouts).
		slog.WarnContext(ctx, "rejeu 2D : catalogue d'objectifs illisible — pas d'objectifs statiques",
			"err", err, "titleSlug", s.titleSlug)
		return nil
	}
	entry, err := cat.Lookup(keys.MapID)
	if err != nil {
		// Carte hors catalogue (72 couvertes sur la centaine jouée) : absence propre.
		slog.DebugContext(ctx, "rejeu 2D : carte hors catalogue d'objectifs",
			"map_id", keys.MapID, "match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	return replay.BuildMapObjectives(entry, specs)
}

// objectiveRoleSpecs projette la table du titre sur le pair_name du match : quels rôles
// servir, et lesquels s'affichent neutres. Vide = mode sans objectifs (ou titre sans
// table — un fichier absent est le cas nominal d'un titre sans rejeu à objectifs).
func (s *replayService) objectiveRoleSpecs(ctx context.Context, pairName string) []replay.ObjectiveRoleSpec {
	path := filepath.Join(title.NewPathResolver(s.repoRoot).TitleMappingsDir(s.titleSlug), objectiveRolesFilename)
	set, err := mappings.LoadObjectiveRolesFromFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			slog.DebugContext(ctx, "rejeu 2D : titre sans table d'objectifs", "titleSlug", s.titleSlug)
		} else {
			// Une table PRÉSENTE mais invalide est une erreur de configuration, pas une
			// donnée absente : elle doit se voir dans les journaux.
			slog.WarnContext(ctx, "rejeu 2D : table d'objectifs illisible",
				"err", err, "path", path, "titleSlug", s.titleSlug)
		}
		return nil
	}
	// La normalisation rend le SOUS-MODE ("Arena:CTF on Aquarius" -> "CTF") ; le
	// matching cherche chaque jeton de la table comme MOT ENTIER dans ce libellé
	// (analysis.ExtractKnownMode — le même matcher que les catégories de mode).
	label := analysis.NormalizeModeLabel(pairName)
	if label == "" {
		return nil
	}
	var specs []replay.ObjectiveRoleSpec
	for _, mode := range set.Modes() {
		if analysis.ExtractKnownMode(label, mode.Match) == "" {
			continue
		}
		for _, role := range mode.Roles {
			specs = append(specs, replay.ObjectiveRoleSpec{Role: role, Neutral: mode.Neutral})
		}
	}
	return specs
}
