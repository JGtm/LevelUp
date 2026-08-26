package replay

// map_objectives.go — le CALQUE STATIQUE DES OBJECTIFS DU MODE JOUÉ, tel que l'API le
// sert AVEC le document de rejeu (ReplayDocument.MapObjectives).
//
// CE QUE C'EST : les objets d'objectif de la carte pour le mode du match — zones de
// Bastion/Extraction (volumes), apparitions et livraisons de drapeau, socles (points) —
// lus du catalogue versionné (objectives_catalog.go) et joints par map_id SEUL.
//
// CE QUE ÇA N'EST PAS :
//   - l'ÉTAT VIVANT des objectifs (qui tient la zone, qui porte le drapeau) : non
//     décodé, au registre des reports. Rien ici n'anime quoi que ce soit.
//   - un champ de l'ARTEFACT : l'artefact est décodé des seuls chunks du film, qui ne
//     nomment ni la carte ni le mode. Le champ se remplit À LA REQUÊTE, au service.
//
// AUCUN NOM DE ZONE : la lettre A/B/C affichée en jeu n'existe pas dans le fichier
// (cf. Zone.SpatialRank — un rang spatial n'est pas un nom, et le client ne doit PAS en
// inventer un). Le DTO ne publie donc AUCUN identifiant affichable.

import "levelup/go-api/internal/analysis/replay/mapvar"

// TeamNeutral est la valeur « aucun camp » du team_index, celle du fichier de carte.
const TeamNeutral = -1

// MapObjectives est le calque statique servi avec le rejeu.
type MapObjectives struct {
	// Zones : objectifs SURFACIQUES (boîtes orientées, cylindres), dans l'ordre spatial.
	Zones []ObjectiveZoneDTO `json:"zones,omitempty"`
	// Markers : objectifs PONCTUELS (apparitions, livraisons, socles), même ordre.
	Markers []ObjectiveMarkerDTO `json:"markers,omitempty"`
}

// ObjectiveZoneDTO est une zone d'objectif prête à dessiner : centre monde, forme en
// demi-extents (la conversion tailles-pleines -> demi est déjà faite, shape.go) et
// orientation projetée au plan.
type ObjectiveZoneDTO struct {
	// Role est le rôle mapvar (`strongholds_zone`, `extraction_zone`, ...).
	Role string `json:"role"`
	// Team est l'index d'équipe À AFFICHER : -1 = neutre. Les modes dont la possession
	// est dynamique (Bastion, Extraction) sont FORCÉS neutres par la table de rôles du
	// titre — le team_index du fichier est une affinité d'emplacement, pas une allégeance.
	Team int     `json:"team"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Z    float32 `json:"z"`
	// Family : "box" ou "cylinder" (les deux seules familles mesurées, shape.go).
	Family string `json:"family"`
	// HalfX/HalfY (boîte) : demi-cotés le long de Forward et de sa perpendiculaire.
	HalfX float32 `json:"halfX,omitempty"`
	HalfY float32 `json:"halfY,omitempty"`
	// Radius (cylindre) : rayon.
	Radius float32 `json:"radius,omitempty"`
	// FwdX/FwdY : le vecteur Forward de la boîte projeté au plan, NON normalisé — le
	// client le normalise en 2D. PAS d'omitempty : une composante exactement nulle est
	// une valeur (boîte alignée sur un axe), pas une absence.
	FwdX float32 `json:"fwdX"`
	FwdY float32 `json:"fwdY"`
}

// ObjectiveMarkerDTO est un objectif ponctuel prêt à dessiner.
type ObjectiveMarkerDTO struct {
	Role string  `json:"role"`
	Team int     `json:"team"` // -1 = neutre, même règle que la zone
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Z    float32 `json:"z"`
}

// ObjectiveRoleSpec est un rôle à servir, avec sa règle d'affichage — la projection de
// la table du titre (objective_roles.toml) une fois le mode reconnu.
type ObjectiveRoleSpec struct {
	Role mapvar.Role
	// Neutral force Team=-1 sur tous les objets du rôle (possession dynamique non
	// décodée : afficher la couleur du fichier affirmerait une allégeance inventée).
	Neutral bool
	// PointsOnly force le rôle en PONCTUEL : un objet à forme sort en MARQUEUR posé à son
	// centre, jamais en zone dessinée. Cf. `mappings.ObjectiveModeEntry.PointsOnly` pour
	// le pourquoi (correctif « des bases s'affichent sur un CTF », 2026-08-26).
	PointsOnly bool
}

// BuildMapObjectives pose les objets des rôles demandés en DTO. Rend nil quand rien
// n'est à servir (le champ du document reste absent — dégradation par absence).
//
// Les rôles sont traités DANS L'ORDRE DONNÉ et dédupliqués (première spécification
// gagne) : deux entrées de la table qui matchent le même mode ne servent pas deux fois
// les mêmes objets.
//
// UN RÔLE `PointsOnly` NE PRODUIT AUCUNE ZONE, quoi que porte le catalogue : ses objets à
// forme sortent en marqueurs, AVANT ceux qui n'en ont jamais eu (ordre déterministe — les
// deux sous-listes sont chacune triées spatialement, et le rendu ne lit pas cet ordre : seul
// `zones` porte un index de jointure, `zoneStates[].zoneRef`).
func BuildMapObjectives(e MapObjectivesEntry, specs []ObjectiveRoleSpec) *MapObjectives {
	out := &MapObjectives{}
	seen := map[mapvar.Role]bool{}
	for _, spec := range specs {
		if seen[spec.Role] {
			continue
		}
		seen[spec.Role] = true
		for _, z := range e.ZonesOfRole(spec.Role).Zones {
			// PONCTUEL PAR DÉCISION DU TITRE : l'objet a bien une forme dans le fichier de
			// carte — on ne la nie pas, on refuse de la PRÉSENTER comme une zone à tenir.
			// Il sort au même endroit (son centre) et par la même porte que les objets qui
			// n'ont jamais eu de forme, donc le client n'a aucune règle à apprendre.
			if spec.PointsOnly {
				out.Markers = append(out.Markers, ObjectiveMarkerDTO{
					Role: string(z.Role),
					Team: displayTeam(z.TeamIndex, spec.Neutral),
					X:    float32(z.Center.X),
					Y:    float32(z.Center.Y),
					Z:    float32(z.Center.Z),
				})
				continue
			}
			dto := ObjectiveZoneDTO{
				Role: string(z.Role),
				Team: displayTeam(z.TeamIndex, spec.Neutral),
				X:    float32(z.Center.X),
				Y:    float32(z.Center.Y),
				Z:    float32(z.Center.Z),
			}
			// Shape est présent PAR CONSTRUCTION (ZonesOfRole écarte les sans-forme sous
			// Pointless) ; le garde évite un déréférencement aveugle si l'invariant bouge.
			if z.Shape == nil {
				continue
			}
			dto.Family = string(z.Shape.Family)
			switch z.Shape.Family {
			case mapvar.ShapeBox:
				if z.Shape.HalfX != nil {
					dto.HalfX = float32(*z.Shape.HalfX)
				}
				if z.Shape.HalfY != nil {
					dto.HalfY = float32(*z.Shape.HalfY)
				}
			case mapvar.ShapeCylinder:
				if z.Shape.Radius != nil {
					dto.Radius = float32(*z.Shape.Radius)
				}
			}
			dto.FwdX = float32(z.Shape.Forward.X)
			dto.FwdY = float32(z.Shape.Forward.Y)
			out.Zones = append(out.Zones, dto)
		}
		for _, p := range e.PointsOfRole(spec.Role) {
			out.Markers = append(out.Markers, ObjectiveMarkerDTO{
				Role: string(p.Role),
				Team: displayTeam(p.TeamIndex, spec.Neutral),
				X:    float32(p.Center.X),
				Y:    float32(p.Center.Y),
				Z:    float32(p.Center.Z),
			})
		}
	}
	if len(out.Zones) == 0 && len(out.Markers) == 0 {
		return nil
	}
	return out
}

// displayTeam applique la règle d'affichage : neutre forcé, ou le team_index du fichier.
func displayTeam(teamIndex int, neutral bool) int {
	if neutral {
		return TeamNeutral
	}
	return teamIndex
}
