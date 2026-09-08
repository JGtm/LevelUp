package replay

// identity_registry_health.go — LA SANTE DU PONT, TELLE QUE LA COUVERTURE LA PUBLIE.
//
// Ce fichier n'accede JAMAIS aux tables brutes : il ne passe que par les accesseurs du registre
// (garde-rail `archlint`, une seule entree d'allowlist). C'est ce qui permet de le lire sans
// avoir a verifier qu'il ne contourne pas une garde — il n'en a pas les moyens.

import "log/slog"

// SanteDuPont rend la sante du pont slot -> joueur : sur quoi il repose, et ce qu'il refuse.
func (r IdentityRegistry) SanteDuPont() BridgeHealth {
	return BridgeHealth{
		Slots:               len(r.IndexParSlot()),
		FromReading:         r.SlotsParLaLecture(),
		LivesNamed:          r.ViesNommeesParLaLecture(),
		LivesTotal:          r.ViesTotal(),
		IndexReadings:       r.LecturesIndex(),
		IndexDisagreements:  r.DesaccordsIndex(),
		SlotCollisions:      r.CollisionsDeSlot(),
		DeathOffsetMatched:  r.DeathOffsetMatches(),
		DeathOffsetRunnerUp: r.CalageSecond(),
		DeathOffsetMs:       r.CalageSiConnu(),
		ClosedByShot:        r.FermeturesParTir(),
		ClosedByRespawn:     r.FermeturesParReapparition(),
		ClosedContested:     r.FermeturesContestees(),
		ClosedRefused:       r.FermeturesRefusees(),
	}
}

// buildCoverage assemble la couverture publiee et ses verdicts. `originResolved` dit si l'origine
// de la frame 0 a ete etablie — elle conditionne la justesse de l'axe de temps de TOUS les
// calques dates depuis l'horloge du film (cf. Coverage.OriginResolved).
func buildCoverage(shots, grenades, objectives LayerCoverage, reg IdentityRegistry,
	originResolved bool, score *ScoreCoverage) *Coverage {
	b := reg.SanteDuPont()
	b.warnIfCalageEtroit()
	return &Coverage{
		Shots: shots, Grenades: grenades, Objectives: objectives, Bridge: b,
		OriginResolved: originResolved, Score: score,
		Verdict: map[string]string{
			"shots":      verdictOf(shots),
			"grenades":   verdictOf(grenades),
			"objectives": verdictOf(objectives),
			"bridge":     verdictOfBridge(b),
		},
	}
}

// logRegistry ALARME sur ce que le registre n'a PAS su nommer. Un lien non resolu se publie et se
// compte (doctrine §0.2 du plan v2) ; il ne se tait jamais.
func (r IdentityRegistry) logRegistry(matchID string) {
	total := r.Section.Coverage.Total()
	slog.Info("rejeu : registre d'identite",
		"match_id", matchID,
		"slots", len(r.IndexParSlot()), "viesNommees", r.ViesNommeesParLaLecture(),
		"viesTotal", r.ViesTotal(), "lecturesIndex", r.LecturesIndex(),
		"desaccordsIndex", r.DesaccordsIndex(), "collisionsSlot", r.CollisionsDeSlot(),
		"parElimination", r.eliminated,
		"liensDirects", total.Direct, "liensDeduits", total.Inferred,
		"liensNonResolus", total.Unresolved)
	if r.DesaccordsIndex() > 0 {
		slog.Warn("rejeu : desaccord de lecture de l'index de joueur — liens directs NON publies",
			"match_id", matchID, "desaccords", r.DesaccordsIndex())
	}
	if total.Unresolved > 0 {
		slog.Warn("rejeu : liens d'identite NON RESOLUS — publies et comptes, jamais inventes",
			"match_id", matchID, "liens", total.Unresolved)
	}
}
