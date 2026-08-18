package replaybuild

// zones.go — LE CATALOGUE DE ZONES DU MATCH, pour le calque de l'ETAT DES ZONES.
//
// # Pourquoi ce fichier existe, et ce qu'il garantit
//
// `ZoneState.ZoneRef` est un INDEX dans `mapObjectives.zones` — le calque statique que le SERVICE
// sert a la requete. L'artefact et le service doivent donc construire la MEME liste, sinon la
// teinte se poserait sur la mauvaise zone : une erreur invisible et credible. Ce fichier rejoue
// exactement la resolution du service (`service/replay_map_objectives.go`) :
//
//	mode -> roles a servir     la table DU TITRE (`objective_roles.toml`), meme matcher ;
//	map_id -> zones du role    le catalogue versionne, role par role puis rang spatial.
//
// L'ordre est celui de `replay.BuildMapObjectives` : les roles DANS L'ORDRE DE LA TABLE, et les
// zones de chaque role dans leur ordre spatial. Un test de garde compare les deux listes.
//
// # Une difference assumee avec le service : le mode se lit sur la VARIANTE
//
// Le service normalise le `pair_name` du match ; ce paquet n'a que `game_variant_name`, et c'est
// ce qui lui est fourni pour nommer les actions d'objectif. Les deux portent le meme jeton de
// mode, mais PAS DANS LE MEME ORDRE — releve du 2026-08-18 : `game_variant_name` vaut
// `Strongholds:Arena` et `KOTH:Arena`, la ou le registre ecrit ses `pair_name` `Arena:Slayer`.
// Le jeton se cherche donc dans la chaine BRUTE, comme MOT ENTIER (cf. `tableRoles`) : les deux
// ordres rendent le meme jeu de roles. Une variante sans jeton connu ne rend AUCUN role, ce qui
// est le cas nominal des modes sans zone.
//
// # Le repli des modes a COLLINE, et la limite qu'il porte
//
// La table du titre ne sert aucun role en KOTH, et c'est justifie cote service : le catalogue de
// formes ne contient AUCUN role de colline (mesure du 2026-08-18 sur 6 cartes). L'artefact, lui,
// SAIT lire les periodes de garde — il les apparie a la grappe des positions, sur les zones que
// la carte declare sous d'autres roles. Il les publie donc, en le disant
// (`coverage.zones.method` et `coverage.zones.roles`), et le client ne les dessinera que le jour
// ou le catalogue portera un role de colline. Publier une mesure que le rendu n'exploite pas
// encore vaut mieux que de la perdre a chaque cuisson.

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// objectiveRolesFilename est le nom du fichier de table sous mappings/ du titre. MEME NOM QUE
// COTE SERVICE : les deux lisent le meme fichier, c'est ce qui fait tenir la jointure.
const objectiveRolesFilename = "objective_roles.toml"

// hillFallbackRoles sont les roles SURFACIQUES essayes pour les modes a colline, dans un ordre
// FIXE. Il est fixe pour que l'index publie reste valable le jour ou la table du titre servirait
// ces memes roles a KOTH : elle devra les declarer dans cet ordre.
var hillFallbackRoles = []mapvar.Role{mapvar.RoleStrongholdZone, mapvar.RoleExtractionZone}

// matchZones rend le catalogue de zones du match et les roles qui le composent, DANS L'ORDRE.
func (b *Builder) matchZones(matchID, mapID, variant string) ([]replay.Zone, string) {
	if mapID == "" {
		slog.Debug("replaybuild: match sans map_id — rejeu sans etat de zone",
			"match_id", matchID, "titleSlug", b.titleSlug)
		return nil, ""
	}
	roles := b.zoneRoles(variant)
	if len(roles) == 0 {
		return nil, "" // mode sans zone : le cas nominal (Assassin, CTF, Oddball)
	}
	cat := b.objectivesCatalog()
	if cat == nil {
		return nil, ""
	}
	entry, err := cat.Lookup(mapID)
	if err != nil {
		slog.Debug("replaybuild: carte hors catalogue d'objectifs — rejeu sans etat de zone",
			"map_id", mapID, "match_id", matchID, "titleSlug", b.titleSlug)
		return nil, ""
	}
	var zones []replay.Zone
	names := make([]string, 0, len(roles))
	for _, r := range roles {
		zones = append(zones, entry.ZonesOfRole(r).Zones...)
		names = append(names, string(r))
	}
	if len(zones) == 0 {
		return nil, ""
	}
	return zones, strings.Join(names, ",")
}

// zoneRoles rend les roles SURFACIQUES du mode, dans l'ordre de la table du titre — ou le repli
// des modes a colline.
func (b *Builder) zoneRoles(variant string) []mapvar.Role {
	roles := b.tableRoles(variant)
	if len(roles) > 0 {
		return roles
	}
	if isHillVariant(variant) {
		return hillFallbackRoles
	}
	return nil
}

// isHillVariant dit si la variante du match est un mode a COLLINE.
//
// DEUX APPELANTS, UN SEUL PREDICAT : les roles de repli (ici) et l'autorisation du repli par les
// positions dans le calque (`replay.ZoneInput.Hill`) doivent parler du MEME ensemble de modes.
// Les ecrire deux fois les laisserait diverger en silence — et la divergence ne se verrait que
// par des collines publiees sur un mode qui n'en a pas.
func isHillVariant(variant string) bool {
	return objectiveevents.ObjectiveTypeOf(variant) == objectiveevents.ObjectiveTypeHill
}

// tableRoles projette la table du titre sur la variante du match : les memes entrees, le meme
// matcher et le meme ordre que le service.
//
// LE JETON SE CHERCHE DANS LA VARIANTE BRUTE, ET C'EST UNE CORRECTION MESUREE. Le service
// normalise d'abord son `pair_name` (`NormalizeModeLabel`), qui pour un libelle technique
// `A:B` garde ce qui suit les deux-points — la convention du registre etant `Arena:Slayer`. Or
// `game_variant_name` porte l'ordre INVERSE : releve du 2026-08-18 sur les temoins du lot,
// `Strongholds:Arena` et `KOTH:Arena`. Normaliser y garderait « Arena » et perdrait le mode.
// `ExtractKnownMode` cherche le jeton comme MOT ENTIER : applique a la chaine brute, il trouve
// « Strongholds » dans les deux ordres, et rend donc le meme jeu de roles que le service.
func (b *Builder) tableRoles(variant string) []mapvar.Role {
	set := b.objectiveRoles()
	if set == nil || variant == "" {
		return nil
	}
	var out []mapvar.Role
	seen := map[mapvar.Role]bool{}
	for _, mode := range set.Modes() {
		if analysis.ExtractKnownMode(variant, mode.Match) == "" {
			continue
		}
		for _, role := range mode.Roles {
			if seen[role] {
				continue // premiere specification gagne, comme BuildMapObjectives
			}
			seen[role] = true
			out = append(out, role)
		}
	}
	return out
}

// objectiveRoles charge (une fois par Builder) la table de roles du titre.
//
// LE CHARGEMENT NE SE RETENTE PAS, meme regle que le catalogue d'objectifs : une passe de masse
// construit des centaines d'artefacts, et une table absente le resterait a chaque appel.
func (b *Builder) objectiveRoles() *mappings.ObjectiveRoleSet {
	if b.rolesTried {
		return b.roles
	}
	b.rolesTried = true
	path := filepath.Join(title.NewPathResolver(b.repoRoot).TitleMappingsDir(b.titleSlug),
		objectiveRolesFilename)
	set, err := mappings.LoadObjectiveRolesFromFile(path)
	switch {
	case os.IsNotExist(err):
		slog.Debug("replaybuild: titre sans table d'objectifs — rejeu sans etat de zone",
			"titleSlug", b.titleSlug)
		return nil
	case err != nil:
		// Une table PRESENTE mais invalide est une erreur de configuration, pas une donnee
		// absente : elle doit se voir dans les journaux (meme regle que cote service).
		slog.Warn("replaybuild: table d'objectifs illisible — rejeu sans etat de zone",
			"err", err, "path", path, "titleSlug", b.titleSlug)
		return nil
	}
	b.roles = set
	return set
}
