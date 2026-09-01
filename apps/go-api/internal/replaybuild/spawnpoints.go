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
// 63 cartes sur 72 portent des points etablis ; neuf ne les ont PAS ETABLIS parce que leur
// `.mvar` servi par l'UGC ne redonne plus les memes socles qu'au catalogue.
//
// SANS POINTS, LES RAMASSAGES GARDENT LEUR ORIGINE `ground` — et il faut le dire, parce que
// l'inverse a ete ecrit ici et c'etait faux. Seul `spawner` devient impossible : `ground` ne
// depend que des poses du document, pas du catalogue. Ce que le client perd, c'est la
// possibilite de conclure quoi que ce soit d'une ABSENCE d'origine, et c'est exactement ce que
// `coverage.pickups.spawnPointsState` lui dit.
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

// spawnPoints rend les points d'apparition non-arme de la carte du match, et L'ETAT du
// catalogue pour cette carte (cf. replay.PickupCoverage.SpawnPointsState).
//
// L'ETAT N'EST PAS REDONDANT AVEC UNE LISTE VIDE, et deux valeurs n'y suffisaient pas : une
// carte peut etre absente du catalogue, y figurer SANS points etablis (sautee pour derive de
// source), ou y figurer avec des points etablis dont le nombre est zero. Les trois se
// distinguent, et le client ne peut lire une absence d'origine qu'en les distinguant.
func (b *Builder) spawnPoints(matchID, mapID string, mapNames []string,
) ([]replay.MapSpawnPoint, string) {
	cat := b.padsCatalog()
	if cat == nil {
		return nil, replay.SpawnPointsMapAbsent
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
			"— aucun ramassage ne pourra etre `spawner`",
			"map_id", mapID, "noms", mapNames, "match_id", matchID, "titleSlug", b.titleSlug)
		return nil, replay.SpawnPointsMapAbsent
	}
	// LA CLE ABSENTE EST UNE INFORMATION : la carte est au catalogue, mais ses points n'y sont
	// pas etablis (generateur en mode ajout-seul, carte sautee pour derive de source).
	if entry.SpawnPoints == nil {
		slog.Debug("replaybuild: points d'apparition NON ETABLIS pour cette carte",
			"map_id", mapID, "match_id", matchID, "titleSlug", b.titleSlug)
		return nil, replay.SpawnPointsNotEstablished
	}
	out := make([]replay.MapSpawnPoint, 0, len(*entry.SpawnPoints))
	for _, p := range *entry.SpawnPoints {
		out = append(out, replay.MapSpawnPoint{
			X: float32(p.Pos.X), Y: float32(p.Pos.Y), Z: float32(p.Pos.Z), Kind: p.Kind,
		})
	}
	return out, replay.SpawnPointsEstablished
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
