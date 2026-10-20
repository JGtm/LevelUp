// Package persist — vehicle_takes_persister.go : ecriture INSERT-ONLY de LA RESSOURCE VEHICULES de
// l Emprise (prises, temps a bord, frags apparies), lue de l artefact de rejeu.
//
// Une destination, une transaction :
//
//	shared.match_vehicle_takes   une ligne par (camp, joueur, famille) + UNE ligne « match » par
//	                             passe (append-only, vue match_vehicle_takes_latest — ADR 0026)
//
// ─── LA LIGNE « MATCH », ET POURQUOI ELLE EXISTE ──────────────────────────────────────────
//
// Une passe s ecrit TOUJOURS, meme sans prise : c est la ligne `match` qui porte la COUVERTURE
// (mesure ou non et pourquoi — D8 ; frags de classe engin non apparies — D9 ; episodes sans xuid
// — D10) et qui fait que la vue `_latest` retient la passe entiere. Sans elle, un match dont la
// re-projection ne trouve plus aucune prise servirait a jamais les lignes de la passe precedente
// (l arbitrage par cle, piege ADR 0026), et un document « non mesure » serait indistinguable d un
// match sans film. Meme role que la ligne `aucune_prise` de `match_pad_pickups_by_tier`, en plus
// general : un « zero mesure » et un « non mesure » ne se confondent jamais.
//
// Les colonnes de couverture sont ecrites sur CHAQUE ligne de la passe (comme `pads_confirmed`
// de la table des niveaux) : toute ligne se lit seule.
//
// ─── ANTI-ART (ADR 0019/0026/0030) ────────────────────────────────────────────────────────
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT : rien a figurer dans l allowlist de
// `no_art_patterns_test.go`, et `match_vehicle_takes` y entre au contraire dans les tables
// PROTEGEES (ainsi que dans `sync/append_only_state_guard_test.go`, la regexp de
// `duckdb/no_raw_rating_reads_test.go` et le registre de compaction).
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb.

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Les deux natures de ligne de `match_vehicle_takes`. Valeurs de DONNEE, jamais des libelles.
const (
	VehicleRowKindTake  = "take"
	VehicleRowKindMatch = "match"
)

// VehicleTakeRow : les prises, le temps a bord et les frags apparies d UN joueur sur UNE famille,
// dans UN camp.
type VehicleTakeRow struct {
	// Camp : designateur d equipe du film (0 a 8).
	Camp int `json:"camp"`
	// XUID du joueur, en decimal. Obligatoire.
	XUID string `json:"xuid"`
	// Family : famille de chassis, ou « unknown » (chassis hors table). Obligatoire.
	Family string `json:"family"`
	// Takes : prises (un vehicule qui passe a ce camp, credite a ce joueur).
	Takes int `json:"takes"`
	// AboardMS : temps a bord, en ms.
	AboardMS int64 `json:"aboard_ms"`
	// Episodes et ProximityEpisodes : episodes comptes, dont ceux dates par proximite.
	Episodes          int `json:"episodes"`
	ProximityEpisodes int `json:"proximity_episodes"`
	// Frags : frags de classe engin tombes PENDANT un episode de ce joueur sur cette famille (D9).
	Frags int `json:"frags"`
}

// VehicleTakesBatch est LE RESULTAT D UNE PASSE DE LECTURE D UN ARTEFACT, cote vehicules. L unite
// de production est le MATCH ENTIER.
type VehicleTakesBatch struct {
	MatchID string `json:"match_id"`
	// Measured faux = « vehicules non mesures » (D8) : artefact sans occupation lue. Reason dit
	// pourquoi ; aucune ligne de prise.
	Measured bool   `json:"measured"`
	Reason   string `json:"reason,omitempty"`
	// DocSchema : la version de schema de l artefact projete (diagnostic de D8).
	DocSchema int `json:"doc_schema"`
	// EpisodesRead : episodes lus avant tout ecart ; EpisodesNoXUID (D10) : occupant non nomme ;
	// EpisodesNoCamp : occupant nomme mais sans camp. Ni l un ni l autre ne fait prise ni temps.
	EpisodesRead   int `json:"episodes_read"`
	EpisodesNoXUID int `json:"episodes_unnamed"`
	EpisodesNoCamp int `json:"episodes_no_camp"`
	// FragsRead faux = les frags de classe engin n ont pas pu etre apparies (FragsReason dit
	// pourquoi) : le rendement n est pas mesure. FragsTotal : frags de classe engin lus ;
	// FragsUnmatched : ceux qu aucun episode de leur tueur ne couvre.
	FragsRead      bool   `json:"frags_read"`
	FragsReason    string `json:"frags_reason,omitempty"`
	FragsTotal     int    `json:"frags_total"`
	FragsUnmatched int    `json:"frags_unmatched"`
	// Rows : les lignes de prise. Vide quand Measured est faux.
	Rows []VehicleTakeRow `json:"rows,omitempty"`
}

// VehicleTakesPersister ecrit une passe de la ressource vehicules dans shared_matches_v2.duckdb.
type VehicleTakesPersister struct {
	db txBeginner
}

// NewVehicleTakesPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewVehicleTakesPersister(db txBeginner) *VehicleTakesPersister {
	return &VehicleTakesPersister{db: db}
}

// Persist ecrit le sous-batch `batch.Shared.VehicleTakes` s il existe. No-op sinon. Chemin
// BatchBuilder ; le chemin direct (lecture tardive d un artefact, backfill) passe par
// [VehicleTakesPersister.PersistPass].
func (p *VehicleTakesPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: VehicleTakesPersister.Persist: batch nil")
	}
	if batch.Shared.VehicleTakes == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.VehicleTakes)
}

const insertVehicleTakesSQL = `
	INSERT INTO match_vehicle_takes (
		match_id, decode_pass, written_at, row_kind,
		camp, xuid, family, takes, aboard_ms, episodes, proximity_episodes, frags,
		measured, unmeasured_reason, doc_schema, episodes_read, episodes_unnamed, episodes_no_camp,
		frags_read, frags_reason, frags_total, frags_unmatched
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs : la ligne match, puis les prises.
func (p *VehicleTakesPersister) PersistPass(ctx context.Context, in VehicleTakesBatch) error {
	if err := ValidateVehicleTakesBatch(in); err != nil {
		return err
	}
	// Le tirage AVANT la transaction : un decode_pass non distinguable ferait rendre a la vue
	// _latest un MELANGE de passes (cf. newDecodePassID).
	pass, err := newDecodePassID()
	if err != nil {
		return err
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_vehicle_takes %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertVehicleTakeRows(ctx, tx, in, pass, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_vehicle_takes %s: %w", in.MatchID, err)
	}
	return nil
}

// insertVehicleTakeRows ecrit la ligne match puis les prises. `pass` et `now` sont partages.
func insertVehicleTakeRows(ctx context.Context, tx *sql.Tx, in VehicleTakesBatch, pass string, now time.Time) error {
	stmt, err := tx.PrepareContext(ctx, insertVehicleTakesSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_vehicle_takes %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	all := make([]VehicleTakeRow, 0, len(in.Rows)+1)
	all = append(all, VehicleTakeRow{Camp: -1}) // la ligne match : camp -1, xuid et famille vides
	all = append(all, in.Rows...)
	for i, r := range all {
		kind := VehicleRowKindTake
		if i == 0 {
			kind = VehicleRowKindMatch
		}
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, now, kind,
			r.Camp, r.XUID, r.Family, r.Takes, r.AboardMS, r.Episodes, r.ProximityEpisodes, r.Frags,
			in.Measured, in.Reason, in.DocSchema, in.EpisodesRead, in.EpisodesNoXUID, in.EpisodesNoCamp,
			in.FragsRead, in.FragsReason, in.FragsTotal, in.FragsUnmatched,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_vehicle_takes %s/%s/%s: %w",
				in.MatchID, r.XUID, kind, err)
		}
	}
	return nil
}

// ValidateVehicleTakesBatch : ce que le persister REFUSE. EXPOSEE pour que le projecteur prouve sa
// sortie sans ouvrir de base. La validation passe AVANT la transaction.
//
//	(1) un match sans identifiant ;
//	(2) une passe « non mesuree » sans raison, ou avec des lignes ; une passe mesuree avec une raison ;
//	(3) une ligne sans xuid, sans famille ou sans camp (0 a 8), ou avec un compte negatif ;
//	(4) des episodes de proximite en trop, ou une ligne sans episode ;
//	(5) un DOUBLON de (camp, xuid, famille) : la lecture sommerait deux fois la meme prise ;
//	(6) des frags incoherents : non lus mais non nuls, lus avec une raison, plus de non apparies que
//	    de lus, ou une somme de frags par ligne qui n est pas le total moins les non apparies.
func ValidateVehicleTakesBatch(in VehicleTakesBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: VehicleTakesBatch.MatchID vide")
	}
	if err := validateVehicleTakesMeasure(in); err != nil {
		return fmt.Errorf("persist: %s: %w", in.MatchID, err)
	}
	vus := make(map[string]bool, len(in.Rows))
	fragsRows := 0
	for i := range in.Rows {
		if err := validateVehicleTakeRow(&in.Rows[i], vus); err != nil {
			return fmt.Errorf("persist: %s ligne #%d: %w", in.MatchID, i, err)
		}
		fragsRows += in.Rows[i].Frags
	}
	if in.FragsRead && fragsRows != in.FragsTotal-in.FragsUnmatched {
		return fmt.Errorf("persist: %s: %d frags sur les lignes pour %d lus et %d non apparies",
			in.MatchID, fragsRows, in.FragsTotal, in.FragsUnmatched)
	}
	if !in.FragsRead && fragsRows != 0 {
		return fmt.Errorf("persist: %s: %d frags sur les lignes alors qu ils n ont pas ete lus", in.MatchID, fragsRows)
	}
	return nil
}

// validateVehicleTakesMeasure : les controles (2) et la partie match de (6).
func validateVehicleTakesMeasure(in VehicleTakesBatch) error {
	switch {
	case !in.Measured && in.Reason == "":
		return errors.New("passe non mesuree sans raison — D8 exige de dire pourquoi")
	case !in.Measured && len(in.Rows) > 0:
		return errors.New("passe non mesuree avec des lignes de prise")
	case in.Measured && in.Reason != "":
		return fmt.Errorf("passe mesuree avec une raison de non mesure %q", in.Reason)
	case in.EpisodesRead < 0 || in.EpisodesNoXUID < 0 || in.EpisodesNoCamp < 0:
		return errors.New("compte d episodes negatif")
	}
	return validateVehicleFragsMeasure(in)
}

// validateVehicleFragsMeasure : la partie frags de (6), sur la ligne match.
func validateVehicleFragsMeasure(in VehicleTakesBatch) error {
	switch {
	case in.FragsRead && in.FragsReason != "":
		return fmt.Errorf("frags lus avec une raison de non lecture %q", in.FragsReason)
	case !in.FragsRead && in.FragsReason == "":
		return errors.New("frags non lus sans raison")
	case !in.FragsRead && (in.FragsTotal != 0 || in.FragsUnmatched != 0):
		return errors.New("frags non lus mais comptes non nuls")
	case in.FragsTotal < 0 || in.FragsUnmatched < 0 || in.FragsUnmatched > in.FragsTotal:
		return fmt.Errorf("frags incoherents (total=%d non apparies=%d)", in.FragsTotal, in.FragsUnmatched)
	}
	return nil
}

// validateVehicleTakeRow applique les controles (3) a (5) a UNE ligne.
func validateVehicleTakeRow(r *VehicleTakeRow, vus map[string]bool) error {
	if r.XUID == "" {
		return errors.New("XUID vide — une ligne de prises sans proprietaire ne se lit pas")
	}
	if r.Family == "" {
		return fmt.Errorf("famille vide pour %q", r.XUID)
	}
	if r.Camp < 0 || r.Camp > 8 {
		return fmt.Errorf("camp %d hors 0..8 pour %q", r.Camp, r.XUID)
	}
	if r.Takes < 0 || r.AboardMS < 0 || r.Episodes < 0 || r.ProximityEpisodes < 0 || r.Frags < 0 {
		return fmt.Errorf("compte negatif pour %q — ce n est pas une mesure", r.XUID)
	}
	if r.Episodes == 0 {
		return fmt.Errorf("ligne de %q sans episode — une ligne de prise en porte au moins un", r.XUID)
	}
	if r.ProximityEpisodes > r.Episodes {
		return fmt.Errorf("%d episodes de proximite pour %d episodes (%q)", r.ProximityEpisodes, r.Episodes, r.XUID)
	}
	cle := fmt.Sprintf("%d\x00%s\x00%s", r.Camp, r.XUID, r.Family)
	if vus[cle] {
		return fmt.Errorf("doublon (%d, %s, %s) dans la meme passe — la lecture sommerait deux fois "+
			"la meme prise", r.Camp, r.XUID, r.Family)
	}
	vus[cle] = true
	return nil
}
