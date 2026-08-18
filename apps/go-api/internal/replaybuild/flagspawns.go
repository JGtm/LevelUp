package replaybuild

// flagspawns.go — LES SOCLES DE DRAPEAU DE LA CARTE, pour le calque du drapeau vivant.
//
// # Ce que ce fichier apporte, et pourquoi le calque ne peut pas s'en passer
//
// Le rejeu sait QUI porte le drapeau et QUAND (evenements nommes du film + fil des morts) ; il
// ne sait pas DE QUEL drapeau il s'agit. L'equipe n'est pas dans le film : elle se deduit du
// SOCLE le plus proche du point de prise, et les socles vivent dans le catalogue versionne
// d'objectifs de carte (`data/titles/{slug}/reference/map_objectives.json`).
//
// # La jointure se fait par map_id, JAMAIS par le module ni par le nom public
//
// Mesure du 2026-08-18 (plan objectifs vivants, decouvertes de la phase 1) : `public_name` est
// VIDE sur la quasi-totalite des entrees du catalogue (il est produit depuis les variantes UGC,
// qui ne le portent pas), et le MODULE n'y porte pas le meme nom que dans le catalogue de bornes
// (`ridgeline` contre `cliffhanger_ridgeline`, `va_behemoth` contre `behemoth_va_behemoth`).
// Joindre sur l'un ou l'autre ne trouve rien, et ne dit rien. C'est deja par `map_id` que le
// service sert le calque STATIQUE des objectifs (`service/replay_map_objectives.go`).
//
// # Les socles NEUTRES sont ecartes, et c'est une regle de mode
//
// Chaque carte de CTF declare TROIS `flag_spawn` : un par equipe, plus un NEUTRE au centre, qui
// n'appartient a aucun camp et ne sert qu'aux variantes « drapeau neutre ». Le retenir sur une
// partie de CTF ordinaire ferait apparaitre un troisieme drapeau qui n'existe pas dans le match,
// immobile a la maison pour l'eternite.
//
// # Toute absence est une DEGRADATION JOURNALISEE, jamais une erreur
//
// Catalogue illisible, carte hors catalogue (72 couvertes sur la centaine jouee), match sans
// map_id : le calque du drapeau reste publie, mais tous les portages tombent dans UN drapeau
// d'equipe -1 et sans etat `home`. `coverage.flagCarries.spawns` publie le compte, donc le fait.

import (
	"log/slog"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
)

// flagSpawns rend les socles de drapeau D'EQUIPE de la carte du match, en coordonnees monde.
func (b *Builder) flagSpawns(matchID, mapID string) []replay.FlagSpawn {
	if mapID == "" {
		slog.Debug("replaybuild: match sans map_id — drapeaux sans equipe proprietaire",
			"match_id", matchID, "titleSlug", b.titleSlug)
		return nil
	}
	cat := b.objectivesCatalog()
	if cat == nil {
		return nil
	}
	entry, err := cat.Lookup(mapID)
	if err != nil {
		slog.Debug("replaybuild: carte hors catalogue d'objectifs — drapeaux sans equipe proprietaire",
			"map_id", mapID, "match_id", matchID, "titleSlug", b.titleSlug)
		return nil
	}
	out := make([]replay.FlagSpawn, 0, 2)
	for _, p := range entry.PointsOfRole(mapvar.RoleFlagSpawn) {
		if p.TeamIndex == replay.TeamNeutral {
			continue // socle du drapeau NEUTRE : pas un camp, pas un drapeau de cette partie
		}
		out = append(out, replay.FlagSpawn{
			Team: p.TeamIndex, X: float32(p.Center.X), Y: float32(p.Center.Y),
		})
	}
	return out
}

// objectivesCatalog charge (une fois par Builder) le catalogue versionne d'objectifs de carte.
//
// LE CHARGEMENT NE SE RETENTE PAS : une passe de masse construit des centaines d'artefacts, et
// un catalogue absent le resterait a chaque appel — autant d'ouvertures de fichier et de lignes
// de journal pour la meme absence. `objectivesTried` fige la tentative.
func (b *Builder) objectivesCatalog() *replay.MapObjectivesCatalog {
	if b.objectivesTried {
		return b.objectives
	}
	b.objectivesTried = true
	path := title.NewPathResolver(b.repoRoot).MapObjectivesPath(b.titleSlug)
	cat, err := replay.LoadMapObjectives(path)
	if err != nil {
		// Le catalogue est VERSIONNE : son absence n'est pas le cas nominal d'une carte sans
		// objectifs, c'est une installation incomplete. On le dit, puis on degrade.
		slog.Warn("replaybuild: catalogue d'objectifs illisible — drapeaux sans equipe proprietaire",
			"err", err, "path", path, "titleSlug", b.titleSlug)
		return nil
	}
	b.objectives = cat
	return cat
}
