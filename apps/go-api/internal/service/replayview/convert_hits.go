package replayview

// convert_hits.go — les touches, les degats et le bilan de tirs. Jumeau de `domain/replaydoc/hits.go` et
// `domain/replaydoc/damage.go`.

import (
	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func toHit(v replay.Hit) replaydoc.Hit {
	return replaydoc.Hit{T: v.T, Shooter: v.Shooter, Victim: v.Victim, Effect: v.Effect, Weapon: v.Weapon}
}

func toDamage(v replay.Damage) replaydoc.Damage {
	return replaydoc.Damage{
		T:        v.T,
		Attacker: v.Attacker,
		Victim:   v.Victim,
		Effect:   v.Effect,
		Weapon:   v.Weapon,
		Shield:   v.Shield,
		Body:     v.Body,
		Measured: v.Measured,
	}
}

func toShotStat(v replay.ShotStat) replaydoc.ShotStat {
	return replaydoc.ShotStat{
		XUID:        v.XUID,
		Shots:       v.Shots,
		ShotEvents:  v.ShotEvents,
		Messages:    v.Messages,
		Hits:        v.Hits,
		DamageDealt: v.DamageDealt,
		DamageTaken: v.DamageTaken,
		Coverage:    v.Coverage,
		Duplicates:  v.Duplicates,
		Sufficient:  v.Sufficient,
	}
}

func toHitCoverage(v replay.HitCoverage) replaydoc.HitCoverage {
	return replaydoc.HitCoverage{
		Estimated:       v.Estimated,
		MinCoverage:     v.MinCoverage,
		MaxDuplicates:   v.MaxDuplicates,
		Frames:          v.Frames,
		StoppedFrames:   v.StoppedFrames,
		VitalsUnread:    v.VitalsUnread,
		MaxStoppedShare: v.MaxStoppedShare,
		Messages:        v.Messages,
		WeaponMessages:  v.WeaponMessages,
		Hits:            v.Hits,
		Published:       v.Published,
		NoShooter:       v.NoShooter,
		NoVictim:        v.NoVictim,
		BeforeOrigin:    v.BeforeOrigin,
		AfterEnd:        v.AfterEnd,
		Sufficient:      v.Sufficient,
		Damages:         v.Damages,
		DamagesMeasured: v.DamagesMeasured,
		DamagesOffUnit:  v.DamagesOffUnit,
		SelfHits:        v.SelfHits,
		SelfDamages:     v.SelfDamages,
		Players:         v.Players,
	}
}
