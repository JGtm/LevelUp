package service

import "levelup/go-api/internal/service/squadagg"

// squadagg_reexport.go — ré-exports des helpers d'agrégation d'escouade extraits
// vers le package feuille internal/service/squadagg (K3b).
//
// Motivation : rompre le cycle service↔teammates. Le sous-package teammates (à
// extraire) et le service-root partagent ces helpers de calcul purs ; les loger
// dans une FEUILLE que les deux importent (squadagg ne dépend ni de service ni de
// teammates) casse le cycle. Les alias conservent les noms lowercase historiques
// pour que les appels service-root restent inchangés (zéro requalification).
type SquadV2Loader = squadagg.SquadV2Loader

// equipmentUsageQuery/buildEquipmentUsageBlock (E6.1bis, 2026-09-09) : déplacés
// vers squadagg pour que internal/service/teammates puisse aussi les appeler sans
// cycle (cf. commentaire de squadagg/equipment_usage.go). Alias conservés pour que
// synthesis_service_usage.go et equipment_usage_block_test.go restent inchangés.
type equipmentUsageQuery = squadagg.EquipmentUsageQuery

var (
	buildSquadHeader         = squadagg.BuildSquadHeader
	buildSquadOrder          = squadagg.BuildSquadOrder
	extractSquadXUIDs        = squadagg.ExtractSquadXUIDs
	filterRowsByCascade      = squadagg.FilterRowsByCascade
	intersectByMatchID       = squadagg.IntersectByMatchID
	projectSharedRows        = squadagg.ProjectSharedRows
	buildEquipmentUsageBlock = squadagg.BuildEquipmentUsageBlock
)
