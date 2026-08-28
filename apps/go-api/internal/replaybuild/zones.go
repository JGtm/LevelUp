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
// # Seules les zones TENUES entrent au catalogue du match
//
// L'etat des zones (`ti=13`) n'a de sens que pour une zone que l'on TIENT — Bastion, colline de
// KOTH — et le balayage de `ti=13` est une marche bit a bit de tous les paquets delta du film
// (11-12 s par film). Or la table du titre sert aussi des VOLUMES a des modes sans etat de zone :
// 18 cartes portent des cylindres `flag_delivery` avec forme (Catalyst, Aquarius, Fortress, ...),
// et l'Extraction a ses `extraction_zone`. Avant le lot C-ter, un CTF ou une Extraction sur ces
// cartes rendait 2 a 5 zones, payait le balayage et publiait une couverture VIDE (revue de la
// phase 2b, P2 hors perimetre). Le catalogue du match ne retient donc que les roles de
// `heldZoneRoles` : c'est le ROLE qui decide, jamais la presence d'une forme.
//
// # La colline de KOTH vient du role `hill`, plus d'un repli
//
// Jusqu'au lot C-ter, la table ne servait rien en KOTH et l'artefact se repliait sur les formes
// de Bastion/Extraction (`strongholds_zone,extraction_zone`), qui n'etaient les collines que par
// coincidence (3 sur 6 sur Catalyst). Le catalogue porte desormais le role `hill` (les volumes
// que la variante de carte declare sous la paire de hashs de mapvar), la table le sert en KOTH,
// et l'artefact le lit par le MEME chemin que Bastion : le repli a disparu avec sa raison d'etre.
// `ZoneInput.Hill` reste, lui, la porte de la METHODE (periodes de garde par la grappe des
// positions au lieu des captures nommees) — c'est le mode qui la decide, pas le catalogue.

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

// heldZoneRoles sont les roles dont la zone se TIENT — ceux pour lesquels un etat de zone
// (`zoneStates`) peut etre publie, et donc les seuls qui valent le balayage de `ti=13`. Un
// role servi par la table mais absent d'ici (livraison de drapeau, zone d'Extraction) reste
// dessine par le service et n'entre pas au catalogue du match.
var heldZoneRoles = map[mapvar.Role]bool{
	mapvar.RoleStrongholdZone: true,
	mapvar.RoleHill:           true,
}

// matchZones rend le catalogue de zones du match et les roles qui le composent, DANS L'ORDRE.
func (b *Builder) matchZones(matchID, mapID, variant string) ([]replay.Zone, string) {
	if mapID == "" {
		slog.Debug("replaybuild: match sans map_id — rejeu sans etat de zone",
			"match_id", matchID, "titleSlug", b.titleSlug)
		return nil, ""
	}
	roles := b.zoneRoles(variant)
	if len(roles) == 0 {
		// Mode sans zone TENUE : le cas nominal (Assassin, CTF, Oddball, Extraction) — meme
		// quand la carte declare des volumes sous d'autres roles.
		return nil, ""
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

// zoneRoles rend les roles de zone TENUE du mode, dans l'ordre de la table du titre.
func (b *Builder) zoneRoles(variant string) []mapvar.Role {
	var out []mapvar.Role
	for _, r := range b.tableRoles(variant) {
		if heldZoneRoles[r] {
			out = append(out, r)
		}
	}
	return out
}

// isHillVariant dit si la variante du match est un mode a COLLINE — la porte de la METHODE par
// les positions (`replay.ZoneInput.Hill`). Un seul predicat, pour qu'il ne diverge pas.
func isHillVariant(variant string) bool {
	return objectiveevents.ObjectiveTypeOf(variant) == objectiveevents.ObjectiveTypeHill
}

// isVipVariant dit si la variante du match est un mode VIP — la GARDE DE MODE de la couronne
// (`replay.VipInput.Scanned`). Elle est ICI, chez l'appelant, parce que `comp 22 A` vaut
// `flag_grabs` en CTF : lu sur un film CTF, il rendrait de fausses couronnes ; le paquet `replay`
// ne devine aucun mode.
//
// CRITERE : le jeton `vip` dans le nom de variante, MEME approche par mot-clef que les autres
// modes (`ctf`, `koth`, `oddball`...). Le marqueur canonique du mode est `GameVariantCategory=23`
// (verifie sur les payloads bruts, `VIP_COURONNE_PROTOCOLE.md`), mais la categorie n'est pas
// portee par `MatchFacts` — le nom l'est. LA GARDE ECHOUE FERMEE : un film VIP dont le nom ne
// porterait pas `vip` ne montre simplement pas de couronne (degradation gracieuse) ; la seule
// erreur dangereuse — une couronne sur un film non-VIP — exigerait un nom non-VIP contenant
// `vip`, ce qu'aucune variante Halo ne fait.
func isVipVariant(variant string) bool {
	return strings.Contains(strings.ToLower(variant), "vip")
}

// isSkullVariant dit si la variante du match est un mode ODDBALL — la GARDE DE MODE du porteur du
// crane (`replay.SkullInput.Scanned`). Elle est ICI, chez l'appelant, parce que `comp 0 A` est le
// score de mode de tout mode : lu sur un film d'un autre mode, il rendrait de faux porteurs ; le
// paquet `replay` ne devine aucun mode. MEME predicat canonique que la colline
// (`ObjectiveTypeOf`), pour qu'il ne diverge pas du reste de la reconnaissance de mode.
func isSkullVariant(variant string) bool {
	return objectiveevents.ObjectiveTypeOf(variant) == objectiveevents.ObjectiveTypeSkull
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
