// Package persist — pad_tiers_persister.go : ecriture INSERT-ONLY des PRISES DE SOCLE
// VENTILEES PAR NIVEAU D ARME, lues de l artefact de rejeu.
//
// Une destination, une transaction :
//
//	shared.match_pad_pickups_by_tier   les prises par (joueur, niveau, famille d arme)
//	                                   (append-only, vue match_pad_pickups_by_tier_latest —
//	                                   ADR 0026)
//
// ─── POURQUOI UN PERSISTER DEDIE ──────────────────────────────────────────────────────────
//
// Meme raison que `FlagGrabsNetPersister` : `SharedPersister.Persist` est un no-op si
// `batch.Shared.Match == nil` et un skip si le match existe deja dans `match_registry` — or
// une passe de lecture d artefact arrive TOUJOURS sur un match deja insere (le film n est pas
// pret au sync primaire, il arrive un cycle plus tard).
//
// ─── ANTI-ART (ADR 0019/0026/0030) ────────────────────────────────────────────────────────
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT — rien a figurer dans
// l allowlist de `no_art_patterns_test.go`, et `match_pad_pickups_by_tier` y entre au
// contraire dans les tables PROTEGEES (ainsi que dans `sync/append_only_state_guard_test.go`
// et la regexp de `duckdb/no_raw_rating_reads_test.go`). « Remplacer » une passe consiste a en
// ecrire une nouvelle : la vue rend la DERNIERE PASSE ENTIERE par match.
//
// ─── L UNITE D ECRITURE EST LA PASSE, ET `decode_pass` EST CE QUI LA NOMME ────────────────
//
// Toutes les lignes d une passe portent le MEME `decode_pass` — meme doctrine que
// `flag_grabs_net_persister.go`. C est lui, et non `written_at`, qui fait gagner une
// generation ENTIERE : un niveau calcule sous une reference de carte PERIMEE ne survit pas a
// cote des niveaux recalcules, il est retracte d un bloc.
//
// ─── UNE PASSE VIDE N EST PAS UNE PASSE A ZERO ────────────────────────────────────────────
//
// Un match sans film, un titre sans la capability : la passe n arrive pas jusqu ici. Si elle
// arrive vide malgre tout, elle est IGNOREE et LOGGUEE — ecrire zero ligne serait
// indistinguable d un match non mesure, et la vue continue de servir la passe precedente.
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb.

package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Les NIVEAUX, tels qu ils sont ecrits en base. Ce sont des valeurs de DONNEE, jamais des
// libelles d ecran : la traduction vit dans l i18n du web.
//
//	PadTierBase         arme presente dans un equipement de DEPART d une vie du match
//	PadTierGround       arme apparue sur un RATELIER de la carte
//	PadTierPower        arme apparue sur un SOCLE DE PUISSANCE
//	PadTierUnclassified socle qu AUCUN emplacement de la reference ne confirme
//	PadTierPowerup      socle de BONUS. Aucun ramassage natif ne peut le nommer aujourd hui
//	                    (l identite d un tel socle est un NOM canonique, pas une famille
//	                    d arme) : la valeur existe pour que le jour ou le film en nommerait un,
//	                    la prise soit ECRITE plutot que silencieusement perdue.
//	PadTierNoPickup     LE ZERO MESURE : un joueur du roster d un match LU qui n a pris aucun
//	                    socle. Sans cette ligne il serait indistinguable d un joueur d un match
//	                    sans film — les deux se liraient « non mesure ». Meme role que
//	                    `completerRosterAZero` cote prises nettes de drapeau.
const (
	PadTierBase         = "base"
	PadTierGround       = "terrain"
	PadTierPower        = "puissance"
	PadTierUnclassified = "non_classe"
	PadTierPowerup      = "bonus"
	PadTierNoPickup     = "aucune_prise"
)

// padTiersValides : les seules valeurs que la table accepte. Une valeur inconnue est un defaut
// de projection, refuse AVANT la transaction.
var padTiersValides = map[string]bool{
	PadTierBase: true, PadTierGround: true, PadTierPower: true,
	PadTierUnclassified: true, PadTierPowerup: true, PadTierNoPickup: true,
}

// PadTierRow porte LES PRISES D UN JOUEUR, A UN NIVEAU, POUR UNE FAMILLE D ARME.
//
// LA FAMILLE EST DANS LA CLE, ET CE N EST PAS UN DETAIL : l ecran montre le detail par arme au
// survol d un niveau. Sans elle il faudrait une seconde table, ou un survol qui ment.
type PadTierRow struct {
	// XUID du joueur, en decimal (la clef de match_participants). Obligatoire.
	XUID string `json:"xuid"`
	// Tier : l une des constantes PadTier* ci-dessus.
	Tier string `json:"tier"`
	// WeaponFamily : la cle NORMALISEE de la famille d arme du socle. VIDE, et seulement vide,
	// sur une ligne PadTierNoPickup : un zero mesure ne porte aucune arme.
	WeaponFamily string `json:"weapon_family"`
	// Pickups : les prises NOMMEES de ce joueur a ce niveau pour cette arme. Zero uniquement
	// sur une ligne PadTierNoPickup.
	Pickups int `json:"pickups"`
}

// PadTiersBatch est LE RESULTAT D UNE PASSE DE LECTURE D UN ARTEFACT, cote niveaux d armes.
//
// L unite de production est le MATCH ENTIER : c est ce qui rend la vue `_latest` capable de
// retenir une generation entiere plutot qu un melange.
type PadTiersBatch struct {
	MatchID string `json:"match_id"`
	// PadsConfirmed / PadsTotal : combien de socles du film la reference de la carte a
	// CONFIRMES, sur combien le film en a publies. Valeurs de MATCH, ecrites sur chaque ligne.
	//
	// ELLES SEPARENT TROIS ETATS QUE RIEN D AUTRE NE SEPARE :
	//
	//	PadsTotal == 0      le film n a vu AUCUN socle — le mode n en allume aucun (cas mesure
	//	                    de toutes les Super Fiesta du parc). « Aucun socle », et non
	//	                    « aucune prise » ;
	//	PadsConfirmed == 0  des socles, mais la carte n est pas dans la reference : les niveaux
	//	                    ne sont PAS ETABLIS, et tout tombe en `non_classe` ;
	//	PadsConfirmed > 0   les niveaux sont mesures.
	PadsConfirmed int `json:"pads_confirmed"`
	PadsTotal     int `json:"pads_total"`
	// RandomStarts : le mode distribue des equipements de depart ALEATOIRES (Fiesta et
	// consorts). Le niveau `base` n y est jamais produit, et l ecran doit pouvoir dire
	// POURQUOI il ne le montre pas.
	RandomStarts bool `json:"random_starts"`
	// Rows : les lignes de la passe. Une passe sans ligne ne s ecrit pas.
	Rows []PadTierRow `json:"rows,omitempty"`
}

// PadTiersPersister ecrit une passe de niveaux d armes dans shared_matches_v2.duckdb.
type PadTiersPersister struct {
	db txBeginner
}

// NewPadTiersPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewPadTiersPersister(db txBeginner) *PadTiersPersister {
	return &PadTiersPersister{db: db}
}

// Persist ecrit le sous-batch `batch.Shared.PadTiers` s il existe. No-op sinon.
//
// C est le chemin BatchBuilder ; le chemin direct (lecture tardive d un artefact, backfill)
// passe par [PadTiersPersister.PersistPass].
func (p *PadTiersPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: PadTiersPersister.Persist: batch nil")
	}
	if batch.Shared.PadTiers == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.PadTiers)
}

const insertPadTiersSQL = `
	INSERT INTO match_pad_pickups_by_tier (
		match_id, decode_pass, xuid, written_at,
		tier, weapon_family, pickups, pads_confirmed, pads_total, random_starts
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
func (p *PadTiersPersister) PersistPass(ctx context.Context, in PadTiersBatch) error {
	if err := ValidatePadTiersBatch(in); err != nil {
		return err
	}
	if len(in.Rows) == 0 {
		slog.WarnContext(ctx, "persist: passe pad_tiers vide, aucune ligne ecrite",
			"match_id", in.MatchID)
		return nil
	}

	// Le tirage AVANT la transaction : un decode_pass non distinguable ferait rendre a la vue
	// _latest un MELANGE de passes, sans aucun symptome visible (cf. newDecodePassID).
	pass, err := newDecodePassID()
	if err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_pad_pickups_by_tier %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertPadTierRows(ctx, tx, in, pass, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_pad_pickups_by_tier %s: %w", in.MatchID, err)
	}
	return nil
}

// insertPadTierRows ecrit les lignes. `pass` et `now` sont partages par toutes les lignes de
// la passe : c est `pass` qui arbitre la vue `_latest`, `now` ne fait que l ordonner.
func insertPadTierRows(ctx context.Context, tx *sql.Tx, in PadTiersBatch, pass string, now time.Time) error {
	stmt, err := tx.PrepareContext(ctx, insertPadTiersSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_pad_pickups_by_tier %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	for _, r := range in.Rows {
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, r.XUID, now,
			r.Tier, r.WeaponFamily, r.Pickups, in.PadsConfirmed, in.PadsTotal, in.RandomStarts,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_pad_pickups_by_tier %s/%s/%s: %w",
				in.MatchID, r.XUID, r.Tier, err)
		}
	}
	return nil
}

// ValidatePadTiersBatch : ce que le persister REFUSE.
//
// EXPORTEE POUR QUE LE PROJECTEUR PUISSE PROUVER SA SORTIE. `sync/replayartifacts` construit
// ces passes ; sans ce point d entree, ses tests ne pourraient verifier qu une passe projetee
// sera ACCEPTEE qu en ouvrant une base — et la validation serait alors re-ecrite ailleurs,
// donc dupliquee. Elle n a aucun effet de bord et ne touche a rien. La validation passe AVANT la
// transaction, donc un refus ne laisse aucune ligne derriere lui.
//
//	(1) un match sans identifiant ;
//	(2) des comptes de socles incoherents (negatifs, ou plus de confirmes que de publies) :
//	    ce n est pas une mesure, c est un defaut de projection amont ;
//	(3) un niveau hors du vocabulaire ecrit ;
//	(4) un XUID vide : la ligne n aurait pas de proprietaire ;
//	(5) un DOUBLON de (xuid, niveau, famille) : la lecture sommerait deux fois la meme prise ;
//	(6) une ligne `aucune_prise` qui porte une arme ou un compte — c est un zero, pas une
//	    mesure d arme ; et symetriquement une ligne de niveau sans arme ou a zero prise.
func ValidatePadTiersBatch(in PadTiersBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: PadTiersBatch.MatchID vide")
	}
	if in.PadsTotal < 0 || in.PadsConfirmed < 0 {
		return fmt.Errorf("persist: %s: socles negatifs (confirmes=%d publies=%d)",
			in.MatchID, in.PadsConfirmed, in.PadsTotal)
	}
	if in.PadsConfirmed > in.PadsTotal {
		return fmt.Errorf("persist: %s: %d socles confirmes pour %d publies — la jointure ne "+
			"cree jamais de socle", in.MatchID, in.PadsConfirmed, in.PadsTotal)
	}
	vus := make(map[string]bool, len(in.Rows))
	for i := range in.Rows {
		if err := validatePadTierRow(&in.Rows[i], vus); err != nil {
			return fmt.Errorf("persist: %s ligne #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validatePadTierRow applique les controles (3) a (6) a UNE ligne.
func validatePadTierRow(r *PadTierRow, vus map[string]bool) error {
	if r.XUID == "" {
		return errors.New("XUID vide — une ligne de prises sans proprietaire ne se lit pas")
	}
	if !padTiersValides[r.Tier] {
		return fmt.Errorf("niveau %q hors vocabulaire", r.Tier)
	}
	cle := r.XUID + "\x00" + r.Tier + "\x00" + r.WeaponFamily
	if vus[cle] {
		return fmt.Errorf("doublon (%s, %s, %s) dans la meme passe — la lecture sommerait deux "+
			"fois la meme prise", r.XUID, r.Tier, r.WeaponFamily)
	}
	vus[cle] = true
	if r.Pickups < 0 {
		return fmt.Errorf("compte negatif pour %q (%d) — ce n est pas une mesure", r.XUID, r.Pickups)
	}
	if r.Tier == PadTierNoPickup {
		if r.WeaponFamily != "" || r.Pickups != 0 {
			return fmt.Errorf("ligne %s de %q avec arme %q et %d prises — un zero mesure ne "+
				"porte aucune arme", PadTierNoPickup, r.XUID, r.WeaponFamily, r.Pickups)
		}
		return nil
	}
	if r.WeaponFamily == "" {
		return fmt.Errorf("ligne de niveau %s sans famille d arme pour %q — le detail par arme "+
			"la lirait comme une arme sans nom", r.Tier, r.XUID)
	}
	if r.Pickups == 0 {
		return fmt.Errorf("ligne de niveau %s a zero prise pour %q — un zero se dit par %s",
			r.Tier, r.XUID, PadTierNoPickup)
	}
	return nil
}
