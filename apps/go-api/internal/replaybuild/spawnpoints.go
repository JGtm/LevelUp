package replaybuild

// spawnpoints.go — LES POINTS D'APPARITION D'OBJET RAMASSABLE de la carte, pour l'origine des
// ramassages.
//
// # Ce que ce fichier apporte
//
// Le film dit qu'un joueur a ramasse une grenade ; il ne dit pas si elle l'attendait a un point
// d'apparition ou si elle gisait la, lachee par un mort. La carte, elle, DECLARE ses points
// d'apparition — au centimetre et des la premiere image. Ce fichier va les chercher au
// catalogue versionne pour que `replay.buildPickups` puisse trancher.
//
// # La jointure se fait par map_id, pour la MEME raison que les socles de drapeau
//
// `public_name` est vide sur la quasi-totalite des entrees (produites depuis les variantes UGC)
// et le module n'y porte pas le meme nom que dans le catalogue de bornes. Joindre autrement ne
// trouverait rien — et ne le dirait pas.
//
// # Le trou se COMPTE, il ne se comble pas
//
// 63 cartes sur 72 portent des points ; neuf n'en ont pas parce que leur `.mvar` servi par
// l'UGC ne redonne plus les memes socles qu'au catalogue. Une carte hors catalogue, un catalogue
// illisible, un match sans map_id : dans les trois cas les ramassages restent publies, sans
// origine, et `coverage.pickups.mapCatalogMissing` porte le fait.
//
// LA CUISSON NE TELECHARGE RIEN. C'est la doctrine du depot et elle vaut ici sans exception :
// une carte manquante se comble par la CLI (`mapopads-build`) ou par le sync, jamais pendant la
// generation d'un artefact.

import (
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

// spawnPoints rend les points d'apparition non-arme de la carte du match, et VRAI si la carte
// a ete trouvee au catalogue. Le booleen n'est pas redondant avec une liste vide : une carte
// connue peut n'avoir aucun point, et ce n'est pas la meme chose qu'une carte inconnue.
func (b *Builder) spawnPoints(matchID, mapID string, mapNames []string,
) ([]replay.MapSpawnPoint, bool) {
	cat := b.padsCatalog()
	if cat == nil {
		return nil, false
	}
	entry, ok := cat.Maps[mapID]
	if !ok && mapID != "" {
		slog.Debug("replaybuild: carte hors catalogue des socles — ramassages sans origine",
			"map_id", mapID, "match_id", matchID, "titleSlug", b.titleSlug)
	}
	// REPLI PAR NOM PUBLIC, et il a un usage precis : la CLI `replay-build` cuit un film a
	// partir d'un NOM de carte (`--map Catalyst`) et n'a pas de `map_id` sans fichier de faits.
	// Sans ce repli, une cuisson unitaire ne pourrait jamais rendre d'origine `spawner`, et on
	// ne pourrait pas verifier la chaine autrement qu'en production.
	//
	// LE `map_id` RESTE PRIORITAIRE : c'est lui que le service utilise, et lui seul est fiable.
	// `public_name` est VIDE sur la quasi-totalite des entrees (elles viennent des variantes
	// UGC) — le repli ne sert donc que les cartes nommees, ce qui est exactement le cas d'usage
	// de la CLI.
	if !ok {
		for _, n := range mapNames {
			if n == "" {
				continue
			}
			for id, e := range cat.Maps {
				if strings.EqualFold(e.PublicName, n) {
					entry, ok, mapID = e, true, id
					break
				}
			}
			if ok {
				break
			}
		}
	}
	if !ok {
		slog.Debug("replaybuild: carte introuvable au catalogue des socles (ni map_id ni nom) "+
			"— ramassages sans origine",
			"map_id", mapID, "noms", mapNames, "match_id", matchID, "titleSlug", b.titleSlug)
		return nil, false
	}
	out := make([]replay.MapSpawnPoint, 0, len(entry.SpawnPoints))
	for _, p := range entry.SpawnPoints {
		out = append(out, replay.MapSpawnPoint{
			X: float32(p.Pos.X), Y: float32(p.Pos.Y), Z: float32(p.Pos.Z), Kind: p.Kind,
		})
	}
	return out, true
}

// padsCatalog charge le catalogue des socles au plus une fois par Builder — meme motif et meme
// raison que `objectivesCatalog` : une passe de masse rejoue la meme carte des dizaines de fois.
func (b *Builder) padsCatalog() *replay.MapWeaponPadsCatalog {
	if b.padsTried {
		return b.pads
	}
	b.padsTried = true
	path := title.NewPathResolver(b.repoRoot).MapWeaponPadsPath(b.titleSlug)
	cat, err := replay.LoadMapWeaponPads(path)
	if err != nil {
		// Le catalogue est VERSIONNE : son absence est une installation incomplete, pas le cas
		// nominal. On le dit, puis on degrade.
		slog.Warn("replaybuild: catalogue des socles illisible — ramassages sans origine",
			"err", err, "path", path, "titleSlug", b.titleSlug)
		return nil
	}
	b.pads = cat
	return cat
}
