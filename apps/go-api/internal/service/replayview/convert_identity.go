package replayview

// convert_identity.go — LA PROJECTION DU REGISTRE D'IDENTITE (schema 50, lot P2).
//
// COPIE PURE, aucune decision : la section servie porte les memes liens que l'artefact, avec
// les memes provenances. Toute transformation ici ferait diverger ce que le gate corpus compare
// (l'artefact) de ce que l'ecran montre (le document servi).

import (
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/games/canonical"
)

func toIdentitySection(v replay.IdentitySection) replaydoc.IdentitySection {
	return replaydoc.IdentitySection{
		Players:       sliceOf(v.Players, toIdentityPlayer),
		BipedSlots:    sliceOf(v.BipedSlots, toIdentityBipedSlot),
		StatborgSlots: sliceOf(v.StatborgSlots, toIdentityStatborgSlot),
		Coverage:      toIdentityCoverage(v.Coverage),
	}
}

func toIdentityPlayer(v replay.IdentityPlayer) replaydoc.IdentityPlayer {
	return replaydoc.IdentityPlayer{
		FilmIndex: v.FilmIndex,
		XUID:      v.XUID,
		Bid:       v.Bid,
		Name:      v.Name,
		Link:      toLink(v.Link),
	}
}

func toIdentityBipedSlot(v replay.IdentityBipedSlot) replaydoc.IdentityBipedSlot {
	return replaydoc.IdentityBipedSlot{
		Slot: v.Slot, XUID: v.XUID, Bid: v.Bid, Link: toLink(v.Link),
	}
}

func toIdentityStatborgSlot(v replay.IdentityStatborgSlot) replaydoc.IdentityStatborgSlot {
	return replaydoc.IdentityStatborgSlot{
		Slot: v.Slot, Round: v.Round, XUID: v.XUID, Link: toLink(v.Link),
	}
}

func toLink(v canonical.Link) replaydoc.Link {
	return replaydoc.Link{
		Source:   string(v.Source),
		Method:   string(v.Method),
		Readings: v.Readings,
		Metric:   v.Metric,
		From:     v.From,
		To:       v.To,
	}
}

func toIdentityCoverage(v replay.IdentityCoverage) replaydoc.IdentityCoverage {
	return replaydoc.IdentityCoverage{
		FilmIndex:    toLinkCounts(v.FilmIndex),
		BipedSlot:    toBipedLinkCounts(v.BipedSlot),
		StatborgSlot: toLinkCounts(v.StatborgSlot),
	}
}

func toBipedLinkCounts(v canonical.BipedLinkCounts) replaydoc.BipedLinkCounts {
	return replaydoc.BipedLinkCounts{
		LinkCounts:       toLinkCounts(v.LinkCounts),
		DirectPropagated: v.DirectPropagated,
		UnresolvedByCause: replaydoc.UnresolvedCauses{
			IndexOutOfTable:   v.UnresolvedByCause.IndexOutOfTable,
			NoCreationRecord:  v.UnresolvedByCause.NoCreationRecord,
			DivergentReadings: v.UnresolvedByCause.DivergentReadings,
		},
	}
}

func toLinkCounts(v canonical.LinkCounts) replaydoc.LinkCounts {
	return replaydoc.LinkCounts{
		Direct:     v.Direct,
		Catalog:    v.Catalog,
		External:   v.External,
		Inferred:   v.Inferred,
		Unresolved: v.Unresolved,
	}
}
