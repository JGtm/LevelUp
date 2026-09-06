package replayview

// convert_map.go — calques de carte, structure, et les deux corps de route qui vivent a cote
// du document (fond de carte, zones nommees). Jumeau de `domain/replaydoc/map.go`.

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
)

func toMapObjectives(v replay.MapObjectives) replaydoc.MapObjectives {
	return replaydoc.MapObjectives{
		Zones:   sliceOf(v.Zones, toObjectiveZoneDTO),
		Markers: sliceOf(v.Markers, toObjectiveMarkerDTO),
	}
}

func toObjectiveZoneDTO(v replay.ObjectiveZoneDTO) replaydoc.ObjectiveZoneDTO {
	return replaydoc.ObjectiveZoneDTO{
		Role:   v.Role,
		Team:   v.Team,
		X:      v.X,
		Y:      v.Y,
		Z:      v.Z,
		Family: v.Family,
		HalfX:  v.HalfX,
		HalfY:  v.HalfY,
		Radius: v.Radius,
		FwdX:   v.FwdX,
		FwdY:   v.FwdY,
	}
}

func toObjectiveMarkerDTO(v replay.ObjectiveMarkerDTO) replaydoc.ObjectiveMarkerDTO {
	return replaydoc.ObjectiveMarkerDTO{
		Role: v.Role,
		Team: v.Team,
		X:    v.X,
		Y:    v.Y,
		Z:    v.Z,
	}
}

func toMapWeaponPads(v replay.MapWeaponPads) replaydoc.MapWeaponPads {
	return replaydoc.MapWeaponPads{
		Pads:     sliceOf(v.Pads, toMapWeaponPadDTO),
		CatalogN: v.CatalogN,
	}
}

func toMapWeaponPadDTO(v replay.MapWeaponPadDTO) replaydoc.MapWeaponPadDTO {
	return replaydoc.MapWeaponPadDTO{
		X:   v.X,
		Y:   v.Y,
		Z:   v.Z,
		Pad: v.Pad,
	}
}

func toMapObject(v replay.MapObject) replaydoc.MapObject {
	return replaydoc.MapObject{
		TypeID: v.TypeID,
		X:      v.X,
		Y:      v.Y,
		Z:      v.Z,
		DX:     v.DX,
		DY:     v.DY,
		Yaw:    v.Yaw,
	}
}

func toSurface(v replay.Surface) replaydoc.Surface {
	return replaydoc.Surface{
		X0:   v.X0,
		Y0:   v.Y0,
		X1:   v.X1,
		Y1:   v.Y1,
		Z:    v.Z,
		ZB:   v.ZB,
		Poly: v.Poly,
	}
}

func toMapBackground(v replay.MapBackground) replaydoc.MapBackground {
	return replaydoc.MapBackground{
		SchemaVersion: v.SchemaVersion,
		Module:        v.Module,
		MapNames:      v.MapNames,
		Image:         v.Image,
		Source:        v.Source,
		GeneratedAt:   v.GeneratedAt,
		Style:         v.Style,
		Calibration:   toMapBackgroundCalibration(v.Calibration),
		Stats:         toMapBackgroundStats(v.Stats),
		Degradations:  v.Degradations,
	}
}

func toMapBackgroundCalibration(v replay.MapBackgroundCalibration) replaydoc.MapBackgroundCalibration {
	return replaydoc.MapBackgroundCalibration{
		MetersPerPixel: v.MetersPerPixel,
		OriginX:        v.OriginX,
		OriginY:        v.OriginY,
		WidthPx:        v.WidthPx,
		HeightPx:       v.HeightPx,
		Convention:     v.Convention,
	}
}

func toMapBackgroundStats(v replay.MapBackgroundStats) replaydoc.MapBackgroundStats {
	return replaydoc.MapBackgroundStats{
		Anchors:                  v.Anchors,
		AnchorsInFrame:           v.AnchorsInFrame,
		AnchorsWithGround:        v.AnchorsWithGround,
		AnchorMedianGapM:         v.AnchorMedianGapM,
		InstancesDrawn:           v.InstancesDrawn,
		InstancesScenery:         v.InstancesScenery,
		PlayLevelZ:               v.PlayLevelZ,
		BoundaryApplied:          v.BoundaryApplied,
		BoundaryPlanes:           v.BoundaryPlanes,
		BoundaryCellsCleared:     v.BoundaryCellsCleared,
		WaterVolumes:             v.WaterVolumes,
		WaterCells:               v.WaterCells,
		CoveredShare:             v.CoveredShare,
		Covered:                  v.Covered,
		CellsSubstituted:         v.CellsSubstituted,
		CellsClipped:             v.CellsClipped,
		CellsAssumedFloor:        v.CellsAssumedFloor,
		ForgeObjects:             v.ForgeObjects,
		ForgeObjectsDrawn:        v.ForgeObjectsDrawn,
		ForgeObjectsWithoutModel: v.ForgeObjectsWithoutModel,
		ForgeDeathVolumes:        v.ForgeDeathVolumes,
	}
}

func toMapCalloutsEntry(v replay.MapCalloutsEntry) replaydoc.MapCalloutsEntry {
	return replaydoc.MapCalloutsEntry{
		Module:     v.Module,
		Provenance: v.Provenance,
		Zones:      sliceOf(v.Zones, toCalloutZone),
	}
}

func toCalloutZone(v replay.CalloutZone) replaydoc.CalloutZone {
	return replaydoc.CalloutZone{
		VolumeIndex: v.VolumeIndex,
		Name:        v.Name,
		EN:          v.EN,
		FR:          v.FR,
		X:           v.X,
		Y:           v.Y,
		Z:           v.Z,
		ZBottom:     v.ZBottom,
		ZTop:        v.ZTop,
		Big:         v.Big,
		Polygon:     v.Polygon,
		Parts:       v.Parts,
		Holes:       v.Holes,
	}
}
