package livesync

// csr_mapper.go — projection PURE du service record arena Halo 5
// (H5ArenaStats.ArenaPlaylistStats[]) vers les lignes CSR canoniques
// (sync.PlayerPlaylistCSR) persistées dans player_csr_snapshots.
//
// Halo 5 expose un CSR PAR PLAYLIST dans le service record agrégé (un seul appel
// GetServiceRecords(gamertag,"arena")), SANS notion de saison : c'est un agrégat
// lifetime. Phase 1 = une seule « saison » FIXE (h5LifetimeSeasonID) ; les saisons
// réelles sont Phase 2 (hors scope).
//
// Fonction PURE (zéro réseau / zéro DB) → testable directement. Le hook de
// persistance (runner) appelle GetServiceRecords puis ce mapper, puis écrit via
// le pattern append-only existant (sync.SaveCSRSnapshots).

import (
	"strings"

	halo5 "levelup/go-api/internal/games/halo_5"
	syncpkg "levelup/go-api/internal/sync"
)

// h5LifetimeSeasonID — saison CSR « lifetime » FIXE de Halo 5 (Phase 1). Le service
// record arena est un agrégat sans saison → un seul bucket. La LECTURE
// (CSRSeasonIDForTitle → GetCSRSnapshots) doit résoudre ce MÊME id pour h5 (cf.
// TitleDescriptor.CSRSeasonID alimenté par config/titles/halo_5/title.toml).
const h5LifetimeSeasonID = "h5-lifetime"

// h5DesignationTiersEN mappe DesignationId (palier CSR majeur Halo 5, 0..6) vers le
// libellé de tier EN CAPITALISÉ attendu par les consommateurs (badge builder
// canonicalHomeSkillTierName, front). Ordre officiel : Bronze < Silver < Gold <
// Platinum < Diamond < Onyx < Champion. DesignationId 6 (Champion) CONFIRMÉ live
// (sonde 2026-06-26 : joueur réel en Champion, Csr 1739, Rank #236) → mappé. Le rang
// mondial #N (H5Csr.Rank) du Champion n'est PAS encore porté par le snapshot canonique
// (CSRRankSnapshot sans champ rang) → follow-up si on veut afficher « #N ».
var h5DesignationTiersEN = []string{
	"Bronze",
	"Silver",
	"Gold",
	"Platinum",
	"Diamond",
	"Onyx",
	"Champion",
}

// h5DesignationTierEN retourne le libellé EN capitalisé d'un DesignationId, ""
// si hors borne (jamais classé / palier inconnu).
func h5DesignationTierEN(designationID int) string {
	if designationID < 0 || designationID >= len(h5DesignationTiersEN) {
		return ""
	}
	return h5DesignationTiersEN[designationID]
}

// mapH5ArenaToPlaylistCSRs projette le service record arena en lignes CSR
// canoniques (une par playlist arena). nil/vide → nil. Pour chaque playlist :
//   - Current  : depuis Csr (palier courant) ; nil pendant le placement → ligne
//     « en placement » (tier vide + MeasurementMatchesRemaining = MeasurementMatchesLeft).
//   - AllTime  : depuis HighestCsr (meilleur palier atteint).
//   - Season   : laissé vide (pas de notion de saison h5 — Phase 2).
//
// Mapping palier : DesignationId → tier EN (Bronze..Onyx) ; sous-palier = Tier
// (1..6), forcé à 0 pour Onyx (palier unique, sans sous-palier).
func mapH5ArenaToPlaylistCSRs(resp *halo5.H5ServiceRecordResponse) []syncpkg.PlayerPlaylistCSR {
	arena := firstArenaStats(resp)
	if arena == nil {
		return nil
	}
	out := make([]syncpkg.PlayerPlaylistCSR, 0, len(arena.ArenaPlaylistStats))
	for i := range arena.ArenaPlaylistStats {
		p := &arena.ArenaPlaylistStats[i]
		if strings.TrimSpace(p.PlaylistId) == "" {
			continue
		}
		// AllTime = pic de la playlist (HighestCsr) ; à défaut (vide pour comptes
		// inactifs en classé) → pic TOUTES playlists (ArenaStats.HighestCsrAttained,
		// le seul champ de pic qui survit hors saison courante). Garantit que le pic
		// CSR carrière (loadCSRAlltimePeak = MAX(alltime_value)) reste affiché.
		allTime := p.HighestCsr
		if allTime == nil {
			allTime = arena.HighestCsrAttained
		}
		out = append(out, syncpkg.PlayerPlaylistCSR{
			PlaylistID: p.PlaylistId,
			// PlaylistName/Queue/Input : non fournis par le service record (best-effort,
			// résolus plus tard via le seed metadata) → laissés vides.
			Current: h5CurrentSnapshot(p),
			AllTime: h5CsrSnapshot(allTime),
			// Season : pas de saison h5 en Phase 1 → snapshot zéro.
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// h5CurrentSnapshot construit le snapshot « courant » d'une playlist. Si Csr est
// nil ET qu'il reste des matchs de mesure (> 0), la playlist est EN PLACEMENT :
// tier vide + MeasurementMatchesRemaining renseigné (cohérent avec le pattern
// Infinite, cf. csr_writes.go : tier vide ou remaining>0 = placement).
func h5CurrentSnapshot(p *halo5.H5ArenaPlaylistStat) syncpkg.CSRRankSnapshot {
	if p.Csr == nil {
		// Placement (ou jamais classé) : exposer les matchs restants.
		return syncpkg.CSRRankSnapshot{
			MeasurementMatchesRemaining: p.MeasurementMatchesLeft,
		}
	}
	return h5CsrSnapshot(p.Csr)
}

// h5CsrSnapshot convertit un *H5Csr natif en CSRRankSnapshot canonique. nil → zéro.
// tier = libellé EN du DesignationId ; sous-palier = Tier (0 pour Onyx).
func h5CsrSnapshot(c *halo5.H5Csr) syncpkg.CSRRankSnapshot {
	if c == nil {
		return syncpkg.CSRRankSnapshot{}
	}
	tier := h5DesignationTierEN(c.DesignationId)
	subTier := c.Tier
	if strings.EqualFold(tier, "Onyx") || strings.EqualFold(tier, "Champion") {
		subTier = 0 // Onyx/Champion = paliers uniques, sans sous-palier.
	}
	return syncpkg.CSRRankSnapshot{
		Value:   float64(c.Csr),
		Tier:    tier,
		SubTier: subTier,
	}
}

// firstArenaStats retourne le corps arena du premier résultat OK (ResultCode 0)
// portant des ArenaStats, ou nil. Miroir de halo_5.firstArenaResult (non exporté).
func firstArenaStats(resp *halo5.H5ServiceRecordResponse) *halo5.H5ArenaStats {
	if resp == nil {
		return nil
	}
	for i := range resp.Results {
		if resp.Results[i].ResultCode == 0 && resp.Results[i].Result.ArenaStats != nil {
			return resp.Results[i].Result.ArenaStats
		}
	}
	return nil
}
