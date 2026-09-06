// Package persist — builder.go : BatchBuilder fluent API pour construire un
// MatchBatch côté caller (sync engine, backfill CLI, scripts).
//
// Usage type :
//
//	builder := persist.NewBatchBuilder("halo_infinite", "Madina97294", "2533274858283686", "sync_delta")
//	builder.SetMatch(matchRegistryRow)
//	builder.AddParticipants(participants)
//	builder.AddMedals(medals)
//	builder.SetEnrichment(enrichment) // ALL enrichments en 1 row
//	builder.SetSkillRank(lusrRow)
//	batch := builder.Build()
//	queue.Submit(batch)

package persist

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"levelup/go-api/internal/domain"
)

// BatchBuilder accumule les données d'un batch puis produit un MatchBatch
// immutable via Build().
//
// Non thread-safe — chaque goroutine collectrice utilise son propre builder.
type BatchBuilder struct {
	batch MatchBatch
}

// NewBatchBuilder crée un builder pour un match donné. Initialise l'ID
// unique du batch (UUID v4) et le timestamp de création.
func NewBatchBuilder(titleSlug, gamertag, xuid, source string) *BatchBuilder {
	return &BatchBuilder{
		batch: MatchBatch{
			BatchID:   newBatchID(),
			TitleSlug: titleSlug,
			Player:    gamertag,
			XUID:      xuid,
			CreatedAt: time.Now().UTC(),
			Source:    source,
		},
	}
}

// SetMatch fixe la row match_registry du match (1 seul).
func (b *BatchBuilder) SetMatch(row *domain.MatchRegistryRow) *BatchBuilder {
	b.batch.Shared.Match = row
	return b
}

// AddParticipants ajoute les rows match_participants.
func (b *BatchBuilder) AddParticipants(rows []domain.MatchParticipantRow) *BatchBuilder {
	b.batch.Shared.Participants = append(b.batch.Shared.Participants, rows...)
	return b
}

// AddMedals ajoute les rows medals_earned.
func (b *BatchBuilder) AddMedals(rows []domain.MedalRow) *BatchBuilder {
	b.batch.Shared.Medals = append(b.batch.Shared.Medals, rows...)
	return b
}

// AddHighlightEvents ajoute les rows highlight_events.
func (b *BatchBuilder) AddHighlightEvents(rows []HighlightEventInsert) *BatchBuilder {
	b.batch.Shared.HighlightEvents = append(b.batch.Shared.HighlightEvents, rows...)
	return b
}

// AddWeaponKills ajoute les rows weapon_kills.
func (b *BatchBuilder) AddWeaponKills(rows []WeaponKillInsert) *BatchBuilder {
	b.batch.Shared.WeaponKills = append(b.batch.Shared.WeaponKills, rows...)
	return b
}

// AddWeaponAccuracy ajoute les rows weapon_accuracy (tirs par arme agrégés).
func (b *BatchBuilder) AddWeaponAccuracy(rows []WeaponAccuracyInsert) *BatchBuilder {
	b.batch.Shared.WeaponAccuracy = append(b.batch.Shared.WeaponAccuracy, rows...)
	return b
}

// AddKillerVictim ajoute les rows killer_victim_pairs.
func (b *BatchBuilder) AddKillerVictim(rows []KillerVictimInsert) *BatchBuilder {
	b.batch.Shared.KillerVictim = append(b.batch.Shared.KillerVictim, rows...)
	return b
}

// AddKillPositions ajoute les rows kill_positions (positions monde par kill).
func (b *BatchBuilder) AddKillPositions(rows []KillPositionInsert) *BatchBuilder {
	b.batch.Shared.KillPositions = append(b.batch.Shared.KillPositions, rows...)
	return b
}

// SetKillSource fixe le résultat d'une passe de décodage du film (1 ligne par
// mort, crédit + source du dégât) pour ce match.
//
// L'unité de production est le MATCH ENTIER : un décodage rend toute la liste
// des morts. On REMPLACE donc le sous-batch au lieu de l'accumuler — deux passes
// concaténées produiraient une liste de morts qui n'a jamais existé. C'est la
// raison du Set… plutôt que Add… ici.
func (b *BatchBuilder) SetKillSource(pass *KillSourceBatch) *BatchBuilder {
	b.batch.Shared.KillSource = pass
	return b
}

// SetWeaponShots fixe la ventilation des tirs par arme d'une passe de décodage
// du film (grain match x joueur x arme) pour ce match.
//
// Set… et non Add… pour la même raison que SetKillSource : l'unité de production
// est le FILM ENTIER. Concaténer deux passes produirait une ventilation qui n'a
// jamais existé — et, ici, un doublon (indice, arme) que le persister refuse.
func (b *BatchBuilder) SetWeaponShots(pass *WeaponShotsBatch) *BatchBuilder {
	b.batch.Shared.WeaponShots = pass
	return b
}

// SetBombStats fixe les statistiques d'Assaut d'une passe de décodage du film
// (les cinq mesures par joueur + les faits datés) pour ce match.
//
// Set… et non Add… pour la même raison que SetKillSource et SetWeaponShots :
// l'unité de production est le FILM ENTIER. Concaténer deux passes produirait un
// jeu de statistiques qui n'a jamais existé — et, ici, un doublon de xuid que le
// persister refuse.
func (b *BatchBuilder) SetBombStats(pass *BombStatsBatch) *BatchBuilder {
	b.batch.Shared.BombStats = pass
	return b
}

// AddXUIDAliases ajoute les rows xuid_aliases.
func (b *BatchBuilder) AddXUIDAliases(rows []XUIDAliasInsert) *BatchBuilder {
	b.batch.Shared.XUIDAliases = append(b.batch.Shared.XUIDAliases, rows...)
	return b
}

// AddMatchCSRs ajoute les rows shared.match_csrs (CSR de tous les
// participants du match, pour le contexte lobby ranked).
func (b *BatchBuilder) AddMatchCSRs(rows []MatchCSRInsert) *BatchBuilder {
	b.batch.Shared.MatchCSRs = append(b.batch.Shared.MatchCSRs, rows...)
	return b
}

// AddCommendations ajoute les rows shared.match_commendations (commendations
// NATIVES progressées sur le match — Halo 5 natif, AXE B).
func (b *BatchBuilder) AddCommendations(rows []CommendationInsert) *BatchBuilder {
	b.batch.Shared.Commendations = append(b.batch.Shared.Commendations, rows...)
	return b
}

// AddObjectiveStats ajoute les rows shared.match_objective_stats (stats objectifs
// par joueur CTF/Zones/Oddball).
func (b *BatchBuilder) AddObjectiveStats(rows []ObjectiveStatsInsert) *BatchBuilder {
	b.batch.Shared.ObjectiveStats = append(b.batch.Shared.ObjectiveStats, rows...)
	return b
}

// SetEnrichment fixe la row player_match_enrichment (UNE row complète avec
// tous les enrichments locaux computed).
func (b *BatchBuilder) SetEnrichment(row *EnrichmentRow) *BatchBuilder {
	b.batch.PlayerData.Enrichment = row
	return b
}

// SetSkillRank fixe la row match_skill_rank pour ce match.
func (b *BatchBuilder) SetSkillRank(row *SkillRankInsert) *BatchBuilder {
	b.batch.PlayerData.SkillRank = row
	return b
}

// AddLUSRComponents ajoute le breakdown LUSR (8 composantes typiquement).
func (b *BatchBuilder) AddLUSRComponents(rows []LUSRComponentInsert) *BatchBuilder {
	b.batch.PlayerData.LUSRComponents = append(b.batch.PlayerData.LUSRComponents, rows...)
	return b
}

// AddCitations ajoute les citations gagnées pour ce match.
func (b *BatchBuilder) AddCitations(rows []CitationInsert) *BatchBuilder {
	b.batch.PlayerData.Citations = append(b.batch.PlayerData.Citations, rows...)
	return b
}

// AddPersonalScoreAwards ajoute les awards PersonalScore.
func (b *BatchBuilder) AddPersonalScoreAwards(rows []PersonalScoreAwardInsert) *BatchBuilder {
	b.batch.PlayerData.PersonalScoreAwards = append(b.batch.PlayerData.PersonalScoreAwards, rows...)
	return b
}

// SetCareerProgression fixe le snapshot career si le rang a changé.
func (b *BatchBuilder) SetCareerProgression(row *CareerProgressionInsert) *BatchBuilder {
	b.batch.PlayerData.CareerProgression = row
	return b
}

// SetSession fixe la session pour ce match.
func (b *BatchBuilder) SetSession(row *SessionInsert) *BatchBuilder {
	b.batch.PlayerData.Session = row
	return b
}

// AddPVEStats ajoute des rows pve_match_stats (1 par participant du match
// Firefight). Allocation lazy de PVEBatch si pas encore initialisé.
func (b *BatchBuilder) AddPVEStats(rows []PVEMatchStatsInsert) *BatchBuilder {
	if len(rows) == 0 {
		return b
	}
	if b.batch.PVE == nil {
		b.batch.PVE = &PVEBatch{}
	}
	b.batch.PVE.Stats = append(b.batch.PVE.Stats, rows...)
	return b
}

// AddModeNameTranslations ajoute des traductions au metadata batch (rare).
func (b *BatchBuilder) AddModeNameTranslations(rows []ModeNameTranslationInsert) *BatchBuilder {
	if b.batch.Metadata == nil {
		b.batch.Metadata = &MetadataBatch{}
	}
	b.batch.Metadata.ModeNameTranslations = append(
		b.batch.Metadata.ModeNameTranslations, rows...)
	return b
}

// Build retourne le MatchBatch immuable. Le builder ne doit plus être
// utilisé après Build (state non-clear par design — créer un nouveau
// builder pour un autre batch).
func (b *BatchBuilder) Build() *MatchBatch {
	out := b.batch
	return &out
}

// newBatchID génère un identifiant unique court (16 hex chars, 8 bytes).
// Pas un UUID v4 complet pour économiser l'espace dans le WAL filename.
// Collision probability négligeable au volume considéré (< 1M batches/jour).
func newBatchID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
