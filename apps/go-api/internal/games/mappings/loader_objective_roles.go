package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// loader_objective_roles.go — table MODE DE JEU -> ROLES D'OBJECTIF statiques du rejeu 2D
// (config/titles/{slug}/mappings/objective_roles.toml).
//
// POURQUOI EN DONNEE : quels modes ont des objectifs statiques, et lesquels, est un savoir
// DU TITRE (meme frontiere que replay_labels.toml, ADR 0011). Le SERVICE de rejeu lit cette
// table pour choisir les roles servis avec le document ; le client n'affiche que ce qui
// arrive. Un titre sans fichier n'a simplement pas de calque d'objectifs.
//
// CE QUE CE LOADER NE FAIT PAS : le MATCHING des libelles de mode. Il vit au service
// (analysis.ExtractKnownMode — mot entier, insensible a la casse) parce que ce package ne
// peut pas importer analysis (analysis importe deja games/mappings : cycle).

// ObjectiveModeEntry est une entree de la table : les jetons qui reconnaissent le mode, et
// les roles d'objectif a servir quand il est reconnu.
type ObjectiveModeEntry struct {
	// Match : jetons de mode, a chercher comme MOT ENTIER dans le libelle normalise.
	Match []string
	// Roles : roles d'objectif du decodeur (mapvar) a servir.
	Roles []mapvar.Role
	// Neutral force l'affichage NEUTRE des objets du mode, meme si le fichier de carte
	// leur donne un team_index : la possession d'une zone de Bastion/Extraction est
	// DYNAMIQUE et n'est pas decodee — la colorer affirmerait une allegeance inventee.
	Neutral bool
	// PointsOnly dit que les objectifs de ce mode sont des POINTS, quelle que soit la forme
	// que le fichier de carte leur donne : un objet a forme sort en MARQUEUR (son centre),
	// jamais en zone dessinee.
	//
	// POURQUOI CE DRAPEAU EXISTE (correctif du 2026-08-26, signalement utilisateur « des
	// bases s'affichent sur un CTF »). Le catalogue porte des formes sur des roles qui ne
	// sont PAS des zones a tenir — recensement du jour sur les 73 entrees : `flag_delivery`
	// 28 objets a forme sur 14 cartes (l'aire de capture declaree par le fichier),
	// `stockpile_navpoint` 32 sur 16, `assault_bomb` 2 sur 2. `BuildMapObjectives`
	// transformait tout objet a forme en ZONE, si bien qu'un CTF dessinait des bases de
	// Bastion a l'emplacement de ses livraisons.
	//
	// LE CATALOGUE N'EST PAS CORRIGE, ET C'EST DELIBERE : cette forme est une donnee REELLE
	// du fichier de carte (l'aire ou la capture est validee). Ce qui etait faux, c'est de la
	// PRESENTER comme une base. La distinction « objectif qu'on TIENT » contre « objectif
	// qu'on TOUCHE » est un savoir du titre, donc elle vit ici, en donnee.
	PointsOnly bool
}

// ObjectiveRoleSet porte la table d'un titre, dans l'ordre du fichier.
type ObjectiveRoleSet struct {
	titleSlug     string
	schemaVersion int
	modes         []ObjectiveModeEntry
}

// objectiveRolesTOML — projection brute du fichier.
type objectiveRolesTOML struct {
	Meta  metaSection              `toml:"meta"`
	Modes []objectiveModeEntryTOML `toml:"modes"`
}

type objectiveModeEntryTOML struct {
	Match      []string `toml:"match"`
	Roles      []string `toml:"roles"`
	Neutral    bool     `toml:"neutral"`
	PointsOnly bool     `toml:"points_only"`
}

// objectiveRolesAdmis — la liste FERMEE des roles du decodeur. Un role libre tomberait en
// silence sur « aucun objet de ce role au catalogue », indistinguable d'une carte qui n'en
// a pas : on refuse au chargement.
var objectiveRolesAdmis = map[mapvar.Role]bool{
	mapvar.RoleFlagSpawn:          true,
	mapvar.RoleFlagDelivery:       true,
	mapvar.RoleStockpileSocket:    true,
	mapvar.RoleStockpileNavpoint:  true,
	mapvar.RoleStrongholdZone:     true,
	mapvar.RoleExtractionZone:     true,
	mapvar.RoleOddballSpawn:       true,
	mapvar.RoleAssaultBomb:        true,
	mapvar.RoleHill:               true,
	mapvar.RoleTotalControlZone:   true,
	mapvar.RoleFirefightObjective: true,
	mapvar.RoleLandGrabZone:       true,
}

// objectiveRolesSurfaciques — les roles dont l'objectif EST une aire qu'on tient, et qui ne
// peuvent donc jamais etre servis par une entree `points_only`.
//
// LA LISTE EST LA POUR REFUSER UNE ERREUR PRECISE, pas pour classer joliment : poser
// `points_only = true` sur l'entree Bastion ferait disparaitre les trois bases de la carte —
// une regression majeure, en une ligne de configuration, et parfaitement silencieuse (le
// calque publierait des marqueurs au centre des zones, ce qui reste credible a l'ecran). Un
// role absent d'ici est ponctuel par nature : le drapeau se TOUCHE, la base se TIENT.
var objectiveRolesSurfaciques = map[mapvar.Role]bool{
	mapvar.RoleStrongholdZone:     true,
	mapvar.RoleExtractionZone:     true,
	mapvar.RoleHill:               true,
	mapvar.RoleTotalControlZone:   true,
	mapvar.RoleFirefightObjective: true,
	mapvar.RoleLandGrabZone:       true,
}

// Modes retourne les entrees DANS L'ORDRE DU FICHIER (copie).
func (s *ObjectiveRoleSet) Modes() []ObjectiveModeEntry {
	if s == nil {
		return nil
	}
	out := make([]ObjectiveModeEntry, len(s.modes))
	copy(out, s.modes)
	return out
}

// TitleSlug retourne le slug declare.
func (s *ObjectiveRoleSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schema declaree.
func (s *ObjectiveRoleSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadObjectiveRolesFromFile lit et valide un objective_roles.toml a un chemin donne.
// L'ABSENCE du fichier est un cas de l'appelant (titre sans objectifs statiques) : elle
// remonte telle quelle (os.IsNotExist), jamais maquillee en table vide.
func LoadObjectiveRolesFromFile(path string) (*ObjectiveRoleSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadObjectiveRolesFromBytes(path, raw)
}

// LoadObjectiveRolesFromBytes parse et valide un payload TOML deja en memoire.
//
// Validation STRICTE (tout-ou-rien, meme regle que replay_labels) : une entree sans jeton
// ne matcherait jamais rien, un role hors liste ne servirait jamais rien — les deux sont
// des erreurs de configuration, pas des donnees absentes.
func LoadObjectiveRolesFromBytes(path string, raw []byte) (*ObjectiveRoleSet, error) {
	var doc objectiveRolesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	if len(doc.Modes) == 0 {
		return nil, fmt.Errorf("%s: aucune entrée [[modes]] — un titre sans mode à objectifs n'a pas ce fichier", path)
	}
	modes := make([]ObjectiveModeEntry, 0, len(doc.Modes))
	for i, e := range doc.Modes {
		entry, err := parseObjectiveMode(path, i, e)
		if err != nil {
			return nil, err
		}
		modes = append(modes, entry)
	}
	return &ObjectiveRoleSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		modes:         modes,
	}, nil
}

func parseObjectiveMode(path string, idx int, e objectiveModeEntryTOML) (ObjectiveModeEntry, error) {
	out := ObjectiveModeEntry{Neutral: e.Neutral, PointsOnly: e.PointsOnly}
	for _, tok := range e.Match {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			return ObjectiveModeEntry{}, fmt.Errorf("%s: [[modes]] #%d : jeton match vide", path, idx)
		}
		out.Match = append(out.Match, tok)
	}
	if len(out.Match) == 0 {
		return ObjectiveModeEntry{}, fmt.Errorf("%s: [[modes]] #%d : match vide (l'entrée ne reconnaîtrait aucun mode)", path, idx)
	}
	for _, r := range e.Roles {
		role := mapvar.Role(strings.TrimSpace(r))
		if !objectiveRolesAdmis[role] {
			return ObjectiveModeEntry{}, fmt.Errorf("%s: [[modes]] #%d : rôle %q inconnu du décodeur (mapvar)", path, idx, r)
		}
		if out.PointsOnly && objectiveRolesSurfaciques[role] {
			return ObjectiveModeEntry{}, fmt.Errorf(
				"%s: [[modes]] #%d : points_only interdit sur le rôle surfacique %q — "+
					"l'objectif de ce rôle est une AIRE qu'on tient, le réduire à un marqueur "+
					"effacerait les zones de la carte sans que rien ne le signale", path, idx, r)
		}
		out.Roles = append(out.Roles, role)
	}
	if len(out.Roles) == 0 {
		return ObjectiveModeEntry{}, fmt.Errorf("%s: [[modes]] #%d : roles vide (l'entrée ne servirait rien)", path, idx)
	}
	return out, nil
}
