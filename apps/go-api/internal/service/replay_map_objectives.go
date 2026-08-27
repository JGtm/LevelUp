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
	"levelup/go-api/internal/port"
)

// objectiveRolesFilename est le nom du fichier de table sous mappings/ du titre.
const objectiveRolesFilename = "objective_roles.toml"

// matchMapKeys résout les identités de carte du match, UNE SEULE FOIS par requête.
//
// POURQUOI CE PALIER EXISTE : DEUX calques statiques en dépendent désormais — les
// objectifs du mode, et les emplacements de socle (replay_map_weapon_pads.go). Les laisser
// interroger chacun la base ferait deux allers-retours pour la même réponse, sur le chemin
// le plus chaud du rejeu. Rend la valeur zéro pour toute absence, toujours journalisée :
// une carte non résolue n'est pas une erreur, mais elle ne doit pas être un silence.
func (s *replayService) matchMapKeys(ctx context.Context, matchID string) port.MatchMapKeys {
	if s.maps == nil {
		return port.MatchMapKeys{}
	}
	keys, err := s.maps.MapKeysForMatch(ctx, matchID)
	if err != nil {
		slog.DebugContext(ctx, "rejeu 2D : carte du match non résolue — calques statiques absents",
			"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		return port.MatchMapKeys{}
	}
	return keys
}

// mapObjectivesForKeys résout le calque statique des objectifs du match. Rend nil pour
// TOUTE absence (mode sans objectifs, carte hors catalogue, titre sans table...).
func (s *replayService) mapObjectivesForKeys(ctx context.Context, matchID string,
	keys port.MatchMapKeys) *replay.MapObjectives {
	if keys.MapID == "" || keys.PairName == "" {
		// Sans map_id la jointure n'existe pas ; sans pair_name le mode non plus.
		slog.DebugContext(ctx, "rejeu 2D : clés de match incomplètes — pas d'objectifs statiques",
			"match_id", matchID, "titleSlug", s.titleSlug)
		return nil
	}
	specs := s.objectiveRoleSpecs(ctx, keys.PairName)
	if len(specs) == 0 {
		// Mode sans objectifs statiques : le cas nominal. Slayer, Land Grab — et désormais
		// TOTAL CONTROL, dont l'entrée a été RETIRÉE le 2026-08-27 (décision utilisateur) :
		// son vivier de 13 à 18 formes ne s'affiche plus, faute d'état vivant sur BTB.
		// PAS King of the Hill, en revanche (rôle `hill`, lot C-ter volet 2).
		//
		// CET EXEMPLE A MAINTENANT VIEILLI TROIS FOIS (KOTH ajouté en 2026-08-19, Total
		// Control ajouté en 2026-08-25 puis RETIRÉ en 2026-08-27) : la liste des modes
		// réellement sans objectif est objective_roles.toml en creux, pas ce commentaire.
		// Le vérifier avant d'y ajouter un nom — Land Grab, par exemple, a bien ses hashs
		// `landgrab_zone` dans le fichier de carte, il n'a simplement ni rôle ni entrée.
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
			specs = append(specs, replay.ObjectiveRoleSpec{
				Role: role, Neutral: mode.Neutral, PointsOnly: mode.PointsOnly,
			})
		}
	}
	return specs
}
