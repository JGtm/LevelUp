package testutil

// replay_non_servis_v160.go — CE QUE LE DECODEUR v1.6.0 (schema 127) ECRIT ET QUE L'API NE PUBLIE PAS.
//
// Meme regle que `replay_servis_seuls.go` : un champ sans lecteur dans LevelUp reste dans le decodeur, qui est public ;
// le miroir se fait le jour ou un lecteur le lit. Les entrees se rangent par raison commune, et chacune porte la date
// de la decision. Les lecteurs existants des champs voisins ne changent pas de forme : les calques neufs sont des
// tableaux ou des champs ajoutes, jamais des champs renommes.

const (
	motifV160Calque = "2026-11-24 (decodeur v1.6.0) : calque neuf du schema 127 (repliques, annonces, gestes, sante et bouclier par vie, " +
		"armes en main, munitions, projectiles colles, candidats de socle, apparence) ; aucun lecteur ici, il reste dans le decodeur."
	motifV160ChampPose = "2026-11-24 (decodeur v1.6.0) : champ ajoute par le schema 127 a un calque que l'API sert deja " +
		"(pleine cadence, decalage d'un tir, hauteur d'un projectile, instants d'un rechargement, pistes d'une ligne de degat, " +
		"orientation d'un socle) ; aucun lecteur ici."
	motifV160CibleBot = "2026-11-25 (decodeur v1.6.0) : la cible d'une action de score quand c'est un bot, designee par sa place ; " +
		"aucun lecteur ici ne lit `scoreActions[].target`, la place n'est donc pas servie."
	motifV160Compteur = "2026-11-24 (decodeur v1.6.0) : compteur de couverture du schema 127 ; aucun lecteur ici, la fiche " +
		"ne lit que les compteurs qui disent qu'une valeur est approximative et ils sont deja servis."
)

func init() {
	for _, cle := range []string{
		"Coverage.AmmoReads", "Coverage.Announcements", "Coverage.Appearance", "Coverage.Chatter",
		"Coverage.ControlPlaces", "Coverage.DrawnWeapons", "Coverage.Vitals",
		"ReplayDocument.AmmoReads", "ReplayDocument.Announcements", "ReplayDocument.Attachments", "ReplayDocument.Chatter",
		"ReplayDocument.DrawnWeapons", "ReplayDocument.PadCandidates", "ReplayDocument.Vitals",
		"RosterEntry.Appearance", "ProjectileCoverage.Attach",
	} {
		ReplayChampsNonServis[cle] = motifV160Calque
	}
	for _, cle := range []string{
		"Track.FullRate", "Shot.O", "Stance.Ms0", "Stance.Ms1", "Grenade.Deto", "Projectile.Z", "Projectile.FullRate",
		"Projectile.Attach", "Reload.Ms", "Reload.Copies", "Gesture.Ms", "Damage.Slot", "Damage.By",
		"MapWeaponPadDTO.Forward", "MapWeaponPadDTO.Up", "MapWeaponPadDTO.Scale",
	} {
		ReplayChampsNonServis[cle] = motifV160ChampPose
	}
	ReplayChampsNonServis["ScoreAction.TargetPlace"] = motifV160CibleBot
	for _, cle := range []string{
		"EliminationCoverage.BotEliminations", "EliminationCoverage.VictimAmbiguous", "EliminationCoverage.VictimPlaces",
		"FlagCarriesCoverage.GaugeReadsAmbiguous", "FlagCarriesCoverage.GaugeReadsOffDrops",
		"FlagCarriesCoverage.GaugeReadsUnpublished", "FlagCarriesCoverage.GaugeSlotsFewReads",
		"FlagCarriesCoverage.GaugeSlotsOffDrops", "FlagCarriesCoverage.GaugeSlotsTeamTaken",
		"GestureCoverage.AfterEnd", "GestureCoverage.BeforeOrigin", "GestureCoverage.ClosedFrames", "GestureCoverage.Copies",
		"GestureCoverage.Frames", "GestureCoverage.Published", "GestureCoverage.Read",
		"GroundWeaponCoverage.Candidates", "GroundWeaponCoverage.PowerupCandidates",
		"HitCoverage.DamagesZero", "HitCoverage.DamagesZeroOffUnit", "HitCoverage.DamagesZeroOutOfAxis",
		"ProjectileCoverage.FullRateSamples", "ScoreActionCoverage.BotRecipient", "ScoreActionCoverage.BotTarget",
		"ScoreCoverage.RoundsAdmittedByReset", "ScoreCoverage.RoundsAdmittedByShare", "ShotAimCoverage.SansPosition",
		"StanceControlCoverage.CrouchCommand", "StanceControlCoverage.FullCadence", "StanceControlCoverage.JumpCommand",
		"StanceControlCoverage.StepUS", "TrackCoverage.ViseesSeulesCoverage", "TrackCoverage.AimOnlyBeforeOrigin", "TrackCoverage.AimOnlyNoLife",
		"TrackCoverage.FullRateAimOnlySamples", "TrackCoverage.FullRateAimSamples", "TrackCoverage.FullRateSamples",
		"TrackCoverage.NoWorld",
	} {
		ReplayChampsNonServis[cle] = motifV160Compteur
	}
	for _, nom := range []string{
		"AmmoRead", "AmmoReadCoverage", "Announcement", "AnnouncementCoverage", "Appearance", "AppearanceCoverage",
		"Attachment", "Chatter", "ChatterCoverage", "ControlKindCoverage", "ControlPlaceCoverage", "DrawnWeapon",
		"DrawnWeaponCoverage", "FullRateFlight", "FullRateTrack", "GrenadeDetonation",
		"ProjectileAttach", "ProjectileAttachCoverage", "ViseesSeulesCoverage", "Vital", "VitalCoverage",
	} {
		ReplayTypesNonServis[nom] = motifV160Calque
	}
}
