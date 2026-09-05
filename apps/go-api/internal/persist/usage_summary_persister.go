// Package persist — usage_summary_persister.go : ecriture INSERT-ONLY d une PASSE DE RESUME
// D USAGE (equipement + socles) dans `shared.match_usage_players` + `shared.match_usage_films`,
// pour un match DEJA insere.
//
// POURQUOI UN PERSISTER DEDIE, ET PAS `SharedPersister` : meme raison que [KillPositionPersister]
// et [WeaponHitDistancePersister] — `SharedPersister.Persist` est un no-op des que le match existe
// deja dans `match_registry`, et le resume se derive de l ARTEFACT de rejeu, qui n existe qu APRES
// le sync primaire (etape post-sync replaybuild, ou backfill CLI sur les artefacts en cache).
//
// LA PASSE EST L ARTEFACT ENTIER : toutes les lignes (films + joueurs) portent le MEME
// `summary_pass` et le MEME `written_at` — c est ce qui rend la re-projection ATOMIQUE a la
// lecture. Les vues `_latest` retiennent LA DERNIERE PASSE PAR MATCH, jamais la derniere ligne
// par cle, et L AUTORITE DE PASSE EST LA LIGNE `match_usage_films` (toujours ecrite, meme sans
// ligne joueur — c est ce qui fait tenir la propriete suivante y compris quand la passe B est
// vide de joueurs, revue adversariale 2026-09-04) : un joueur present dans une passe A et
// absent de la passe B disparait de `_latest` avec la passe A (cf. steps_shared_usage_summary.go).
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs — aucun UPDATE, aucun DELETE, aucun ON CONFLICT ;
// rien a faire figurer dans l allowlist de `internal/sync/no_art_patterns_test.go`.
//
// LE PERSISTER SERIALISE LES JSON LUI-MEME (deployed_json, pad_pickups_json, weapon_pads_json,
// powerup_pickups_json) : la forme stockee est UNE decision d ecriture, pas une decision de
// projection — la garder ici evite que deux appelants (post-sync, backfill) fabriquent deux
// formes. `encoding/json` trie les cles de map : la sortie est deterministe.
//
// PRE-REQUIS : le caller tient le lease RW sur shared_matches_v2.duckdb (ADR 0013). `txBeginner`
// accepte aussi bien *sql.DB qu un LeasedWriter.

package persist

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/analysis/replay"
)

// UsageSummaryPersister ecrit une passe de resume d usage dans shared_matches_v2.duckdb.
type UsageSummaryPersister struct {
	db txBeginner
}

// NewUsageSummaryPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewUsageSummaryPersister(db txBeginner) *UsageSummaryPersister {
	return &UsageSummaryPersister{db: db}
}

// PersistPass ecrit le resume d UN match en 1 transaction, en INSERT purs.
//
// La ligne `match_usage_films` est TOUJOURS ecrite, meme sans ligne joueur : un match ou
// personne n a rien fait d attribuable (aucun grappin, aucun socle nomme) reste un match
// MESURE, et son echelle de temps sert l agregat de session. Zero ligne joueur n est donc
// pas une passe vide — mais elle se journalise, parce qu elle est rare et merite un oeil.
func (p *UsageSummaryPersister) PersistPass(
	ctx context.Context, matchID string, s *replay.UsageSummary,
) error {
	if err := validateUsageSummary(matchID, s); err != nil {
		return err
	}
	// Meme tirage que les autres passes de generation : un identifiant non unique ferait
	// rendre a la vue _latest un melange de passes (cf. newDecodePassID).
	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", matchID, err)
	}
	now := time.Now().UTC()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx resume usage %s: %w", matchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertUsageFilmRow(ctx, tx, matchID, pass, now, &s.Match); err != nil {
		return err
	}
	if err := insertUsagePlayerRows(ctx, tx, matchID, pass, now, s.Players); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit resume usage %s: %w", matchID, err)
	}
	if len(s.Players) == 0 {
		slog.InfoContext(ctx, "persist: resume usage sans ligne joueur (rien d attribuable)",
			"match_id", matchID, "pad_occupancies", s.Match.PadOccupancies)
	}
	slog.InfoContext(ctx, "persist: resume usage ecrit",
		"match_id", matchID, "joueurs", len(s.Players),
		"pad_named", s.Match.PadNamed, "pad_unnamed", s.Match.PadUnnamed,
		"summary_rev", replay.UsageSummaryRev, "artifact_schema", s.Match.SchemaVersion)
	return nil
}

// validateUsageSummary : ce que le persister REFUSE, AVANT la transaction — un refus ne
// laisse aucune ligne derriere lui.
func validateUsageSummary(matchID string, s *replay.UsageSummary) error {
	if matchID == "" {
		return errors.New("persist: UsageSummaryPersister.PersistPass: matchID vide")
	}
	if s == nil {
		return fmt.Errorf("persist: UsageSummaryPersister.PersistPass %s: resume nil", matchID)
	}
	seen := make(map[string]bool, len(s.Players))
	for i := range s.Players {
		xuid := s.Players[i].XUID
		if xuid == "" {
			return fmt.Errorf("persist: resume usage %s: ligne #%d sans xuid "+
				"(la projection ne doit jamais produire de ligne anonyme)", matchID, i)
		}
		if seen[xuid] {
			return fmt.Errorf("persist: resume usage %s: xuid %s en double dans la meme passe",
				matchID, xuid)
		}
		seen[xuid] = true
	}
	return nil
}

// insertUsageFilmRow ecrit la ligne de grain MATCH.
func insertUsageFilmRow(
	ctx context.Context, tx *sql.Tx, matchID, pass string, now time.Time, m *replay.UsageMatchSummary,
) error {
	weaponPads, err := usageWeaponPadsJSON(m.WeaponPads)
	if err != nil {
		return fmt.Errorf("persist: resume usage %s: weapon_pads_json: %w", matchID, err)
	}
	powerups, err := usageCountMapJSON(m.PowerupPadPickups)
	if err != nil {
		return fmt.Errorf("persist: resume usage %s: powerup_pickups_json: %w", matchID, err)
	}
	if _, err := tx.ExecContext(ctx, insertMatchUsageFilmSQL,
		matchID, pass, replay.UsageSummaryRev, m.SchemaVersion, now,
		m.FrameIntervalMS, m.FrameCount, m.DurationMS,
		m.PadOccupancies, m.PadNamed, m.PadUnnamed,
		weaponPads, powerups); err != nil {
		return fmt.Errorf("persist: INSERT match_usage_films %s: %w", matchID, err)
	}
	return nil
}

// insertUsagePlayerRows ecrit les lignes de grain (match, joueur), toutes sous la meme passe.
func insertUsagePlayerRows(
	ctx context.Context, tx *sql.Tx, matchID, pass string, now time.Time, rows []replay.UsagePlayerSummary,
) error {
	if len(rows) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, insertMatchUsagePlayerSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_usage_players %s: %w", matchID, err)
	}
	defer func() { _ = stmt.Close() }()
	for i := range rows {
		r := &rows[i]
		deployed, err := usageCountMapJSON(r.DeployedByFamily)
		if err != nil {
			return fmt.Errorf("persist: resume usage %s/%s: deployed_json: %w", matchID, r.XUID, err)
		}
		padPickups, err := usageCountMapJSON(r.PadPickupsByWeapon)
		if err != nil {
			return fmt.Errorf("persist: resume usage %s/%s: pad_pickups_json: %w", matchID, r.XUID, err)
		}
		if _, err := stmt.ExecContext(ctx,
			matchID, pass, replay.UsageSummaryRev, now, r.XUID,
			r.GrapplePulls,
			r.CamoEpisodes, r.CamoMS, r.CamoKills,
			r.OvershieldEpisodes, r.OvershieldMS, r.OvershieldKills,
			deployed, r.DroppedObjects, r.GrenadesThrown,
			r.PadPickups, padPickups); err != nil {
			return fmt.Errorf("persist: INSERT match_usage_players %s/%s: %w", matchID, r.XUID, err)
		}
	}
	return nil
}

// usageCountMapJSON serialise une map de comptes. Une map vide ou nil devient `{}` — jamais
// `null` : la colonne est NOT NULL et « aucun » est une valeur, pas une absence.
func usageCountMapJSON(m map[string]int) (string, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// usageWeaponPadsJSON serialise la liste des socles d arme, dans l ordre du document.
// Vide devient `[]`, meme raison que usageCountMapJSON.
func usageWeaponPadsJSON(pads []replay.UsageWeaponPad) (string, error) {
	if len(pads) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(pads)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

const insertMatchUsageFilmSQL = `
	INSERT INTO match_usage_films
		(match_id, summary_pass, summary_rev, artifact_schema, written_at,
		 frame_interval_ms, frame_count, duration_ms,
		 pad_occupancies, pad_named, pad_unnamed,
		 weapon_pads_json, powerup_pickups_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

const insertMatchUsagePlayerSQL = `
	INSERT INTO match_usage_players
		(match_id, summary_pass, summary_rev, written_at, xuid,
		 grapple_pulls,
		 camo_episodes, camo_ms, camo_kills,
		 overshield_episodes, overshield_ms, overshield_kills,
		 deployed_json, dropped_objects, grenades_thrown,
		 pad_pickups, pad_pickups_json)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
