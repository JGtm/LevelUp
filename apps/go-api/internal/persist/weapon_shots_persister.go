// Package persist — weapon_shots_persister.go : ecriture INSERT-ONLY d une PASSE DE DECODAGE de
// film dans `shared.match_weapon_shots` (grain match x joueur x arme, une seule mesure : le
// nombre de tirs decodes).
//
// POURQUOI UN PERSISTER DEDIE : meme raison que `KillSourcePersister`. `SharedPersister.Persist`
// est un no-op si `batch.Shared.Match == nil` et un skip si le match existe deja dans
// `match_registry` — or une passe de decodage arrive TOUJOURS sur un match deja insere (le film
// n est pas pret au sync primaire, il arrive un cycle plus tard).
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun DELETE, aucun UPDATE, aucun ON CONFLICT.
// « Remplacer » une passe consiste a en ecrire une nouvelle ; la vue `match_weapon_shots_latest`
// ne rend que la derniere. Rien a faire figurer dans l allowlist de `no_art_patterns_test.go`.
//
// ─── LA PORTE VIT ICI, ET NULLE PART AILLEURS ──────────────────────────────────────────────
//
// La regle des +-10 % est ecrite UNE FOIS, dans [EvaluateShotsGate], et le persister l applique
// lui-meme : l appelant fournit la REFERENCE (`shots_fired`), pas le VERDICT. C est ce qui rend
// impossible le scenario « un collecteur publie `publishable = true` sans avoir eu de
// reference ». Le lecteur, lui, ne recalcule jamais rien : il lit la colonne.
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb.

package persist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

// shotsGateTolerance — la porte de publication : le total decode doit tomber a +-10 % du
// `shots_fired` de l API.
//
// CE SEUIL N EST PAS UN REGLAGE LIBRE. Il vient de la mesure qui etablit la loi k = 1 : 84.0 %
// des observations de mode standard tiennent dans cette tolerance, mediane 0.994 sur 3 129
// observations. Le deplacer changerait la POPULATION publiee sans changer le decodage — et ce
// serait un choix de couverture deguise en choix de qualite.
const shotsGateTolerance = 0.10

// Verdicts de la porte. Ils sont ecrits tels quels dans `gate_reason` : une portee doit se lire
// sans table de correspondance.
const (
	// GateReasonWithinTolerance : le total decode tombe dans la tolerance. SEUL cas publiable.
	GateReasonWithinTolerance = "total-dans-tolerance-10pct"
	// GateReasonOutOfTolerance : le total decode s ecarte de plus de 10 % de la reference.
	GateReasonOutOfTolerance = "total-hors-tolerance-10pct"
	// GateReasonNoReference : aucun `shots_fired` connu pour ce joueur. La porte EXIGE la
	// reference API — aucun detecteur offline ne la remplace (taux de pas du compteur :
	// r = 0.15 ; porte sur le nombre d armes : +3.8 points en jetant 46.6 % des observations).
	GateReasonNoReference = "reference-shots-fired-absente"
	// GateReasonReferenceZero : l API annonce EXACTEMENT 0 tir et le decodeur en attribue.
	// Ce n est pas une perte, c est une ERREUR D ATTRIBUTION — 168 observations connues, pour
	// 32 090 tirs attribues, toutes hors du mode standard.
	GateReasonReferenceZero = "reference-nulle-decodage-non-nul"
)

// weaponSentinelMax — les identifiants 0, 1 et 2 sont reserves par `analysis` (grenade, melee,
// vehicule). Recopie ici pour ne pas faire dependre `persist` du package `analysis`, avec un
// test qui interdit la derive (weapon_shots_sentinels_test.go).
const weaponSentinelMax = 2

// WeaponShotCount : UNE ARME, et le nombre de tirs decodes avec elle.
type WeaponShotCount struct {
	// WeaponID : identifiant filmshell 64 bits (celui de `metadata.weapon_labels.weapon_id`).
	// ⚠ Ce n est PAS le tag jpt! 32 bits du dead-state (`match_kill_events.source_tag`).
	WeaponID uint64
	// Shots : nombre de fire-events decodes. Un fire-event est un tir, sans coefficient.
	Shots int
}

// WeaponShotsPlayer : LA VENTILATION D UN JOUEUR, et la reference qui la juge.
//
// La porte est PAR JOUEUR parce que la reference l est : `shots_fired` est un total de joueur.
// Toutes les armes d un joueur partagent donc le meme verdict — ce n est pas une approximation,
// c est la granularite de la seule reference existante.
type WeaponShotsPlayer struct {
	// PlayerIndex : indice de replication BRUT (5 bits, eventStart+31). Requis : c est la
	// quantite qui ne depend d aucune resolution.
	PlayerIndex int
	// XUID : vide = bot, ou indice non rattache au roster. Colonne NULL.
	XUID string
	// ShotsFired : la reference API. nil = absente — la porte refuse alors la publication,
	// elle ne la suppose pas.
	ShotsFired *int
	// Weapons : la ventilation. Une arme absente = aucun tir decode avec elle, et cela n est
	// une mesure de zero QUE si la porte passe.
	//
	// ⚠ UNE LISTE VIDE N ECRIT RIEN, DONC N ECRIT PAS NON PLUS LE VERDICT DE LA PORTE : le
	// joueur disparait de la table. Un joueur dont l API annonce des tirs et dont le decodeur
	// n en trouve aucun est donc SILENCIEUX — le detecter exige de croiser avec
	// `match_participants`. Non reparable a ce grain : une ligne « zero tir » serait une
	// mesure fabriquee.
	Weapons []WeaponShotCount
}

// WeaponShotsBatch : LE RESULTAT D UNE PASSE DE DECODAGE D UN FILM, cote tirs.
//
// L unite de production est le MATCH ENTIER, comme pour `KillSourceBatch` : c est ce qui rend
// la vue `_latest` capable de retenir une generation entiere plutot qu un melange.
type WeaponShotsBatch struct {
	MatchID string `json:"match_id"`
	// DecoderRev : version du decodeur. Requis.
	DecoderRev string `json:"decoder_rev"`
	// Players : les joueurs, dans l ordre rendu par le decodeur.
	Players []WeaponShotsPlayer `json:"players,omitempty"`
}

// WeaponShotsPersister ecrit une passe de decodage dans shared_matches_v2.duckdb.
type WeaponShotsPersister struct {
	db txBeginner
}

// NewWeaponShotsPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewWeaponShotsPersister(db txBeginner) *WeaponShotsPersister {
	return &WeaponShotsPersister{db: db}
}

// EvaluateShotsGate applique LA porte de publication, et c est son unique copie.
//
// `decodedTotal` est la somme des tirs decodes du joueur ; `shotsFired` la reference API (nil =
// absente). Retourne (publiable, motif). Le motif est TOUJOURS renseigne, y compris en cas de
// succes : une portee qui ne s ecrit que sur les echecs laisse croire que le succes n en a pas.
//
// LES DEUX CAS LIMITES, ECRITS PLUTOT QUE SUBIS :
//   - reference nulle et decodage nul : les deux disent zero, la porte PASSE ;
//   - reference nulle et decodage non nul : erreur d attribution connue et mesuree, la porte
//     REFUSE. Le ratio n est pas calculable, et l ecrire comme « hors tolerance » masquerait
//     que c est un phenomene different d un simple ecart.
func EvaluateShotsGate(decodedTotal int, shotsFired *int) (bool, string) {
	if shotsFired == nil {
		return false, GateReasonNoReference
	}
	if *shotsFired == 0 {
		if decodedTotal == 0 {
			return true, GateReasonWithinTolerance
		}
		return false, GateReasonReferenceZero
	}
	ratio := float64(decodedTotal) / float64(*shotsFired)
	if ratio >= 1-shotsGateTolerance && ratio <= 1+shotsGateTolerance {
		return true, GateReasonWithinTolerance
	}
	return false, GateReasonOutOfTolerance
}

// Persist ecrit le sous-batch `batch.Shared.WeaponShots` s il existe. No-op sinon.
//
// C est le chemin BatchBuilder ; le chemin direct (completion tardive d un film, backfill) passe
// par [WeaponShotsPersister.PersistPass].
func (p *WeaponShotsPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: WeaponShotsPersister.Persist: batch nil")
	}
	if batch.Shared.WeaponShots == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.WeaponShots)
}

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
//
// Toutes les lignes portent le MEME `decode_pass` et le MEME `written_at` : c est ce qui rend la
// vue `_latest` capable de retenir une generation entiere.
//
// Une passe VIDE est ignoree et LOGGUEE : ecrire zero ligne serait indistinguable d un match
// sans tir, et la vue continue de servir la passe precedente.
func (p *WeaponShotsPersister) PersistPass(ctx context.Context, in WeaponShotsBatch) error {
	if err := validateWeaponShotsBatch(in); err != nil {
		return err
	}
	if countWeaponShotRows(in) == 0 {
		slog.WarnContext(ctx, "persist: passe weapon_shots vide, aucune ligne ecrite",
			"match_id", in.MatchID, "decoder_rev", in.DecoderRev)
		return nil
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", in.MatchID, err)
	}

	refused, err := p.insertShotsPass(ctx, in, pass, time.Now().UTC())
	if err != nil {
		return err
	}

	// La portee remonte aussi en log : une passe majoritairement refusee est un signal de
	// decodage, pas une fatalite a decouvrir plus tard en SQL.
	if refused > 0 {
		slog.InfoContext(ctx, "persist: joueurs refuses par la porte de publication des tirs",
			"match_id", in.MatchID, "joueurs_refuses", refused,
			"joueurs_total", len(in.Players), "decoder_rev", in.DecoderRev)
	}
	return nil
}

// insertShotsPass ouvre la transaction et ecrit toutes les lignes. Retourne le nombre de joueurs
// refuses par la porte.
func (p *WeaponShotsPersister) insertShotsPass(
	ctx context.Context, in WeaponShotsBatch, pass string, now time.Time,
) (int, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("persist: BeginTx match_weapon_shots %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	stmt, err := tx.PrepareContext(ctx, insertWeaponShotSQL)
	if err != nil {
		return 0, fmt.Errorf("persist: prepare match_weapon_shots %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	refused := 0
	for i := range in.Players {
		pl := &in.Players[i]
		publishable, reason := EvaluateShotsGate(totalShots(pl), pl.ShotsFired)
		if !publishable {
			refused++
		}
		for _, w := range pl.Weapons {
			// ⚠ L IDENTIFIANT D ARME PASSE EN CHAINE DECIMALE, PAS EN int64. Plus de la
			// moitie du catalogue filmshell a son bit de poids fort a 1 (Fuel Rod SPNKr
			// 0x9d6aaed242c9679f, Needler, Sidekick, Commando, les deux grenades...) : un
			// `int64(w.WeaponID)` les rend NEGATIFS et le CAST AS UBIGINT echoue
			// (« value is out of range »). C est le pattern deja etabli du depot
			// (`ubigintArg`, shared_persister.go).
			args := []any{
				in.MatchID, pass, in.DecoderRev, now,
				int64(pl.PlayerIndex), nullIfEmpty(pl.XUID),
				strconv.FormatUint(w.WeaponID, 10), int64(w.Shots),
				publishable, reason,
			}
			if _, err := stmt.ExecContext(ctx, args...); err != nil {
				return 0, fmt.Errorf("persist: INSERT match_weapon_shots %s pi=%d arme=%d: %w",
					in.MatchID, pl.PlayerIndex, w.WeaponID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("persist: Commit match_weapon_shots %s: %w", in.MatchID, err)
	}
	return refused, nil
}

const insertWeaponShotSQL = `
	INSERT INTO match_weapon_shots (
		match_id, decode_pass, decoder_rev, written_at,
		player_index, player_xuid,
		weapon_id, shots_decoded,
		publishable, gate_reason
	) VALUES (
		?, ?, ?, ?,
		CAST(? AS SMALLINT), ?,
		CAST(? AS UBIGINT), CAST(? AS INTEGER),
		?, ?
	)`

func totalShots(pl *WeaponShotsPlayer) int {
	n := 0
	for _, w := range pl.Weapons {
		n += w.Shots
	}
	return n
}

func countWeaponShotRows(in WeaponShotsBatch) int {
	n := 0
	for i := range in.Players {
		n += len(in.Players[i].Weapons)
	}
	return n
}

// validateWeaponShotsBatch : ce que le persister REFUSE au niveau de la passe. La validation
// passe AVANT la transaction : un refus ne laisse aucune ligne derriere lui.
func validateWeaponShotsBatch(in WeaponShotsBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: WeaponShotsBatch.MatchID vide")
	}
	if in.DecoderRev == "" {
		return fmt.Errorf("persist: WeaponShotsBatch.DecoderRev vide (%s) — "+
			"une passe doit dire quel decodeur l a produite", in.MatchID)
	}
	seen := make(map[[2]uint64]bool)
	for i := range in.Players {
		if err := validateWeaponShotsPlayer(&in.Players[i], seen); err != nil {
			return fmt.Errorf("persist: %s joueur #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validateWeaponShotsPlayer : ce que le persister REFUSE au niveau du joueur et de la ligne.
//
// CE QUI EST REFUSE, ET POURQUOI :
//
//	(1) un indice de replication negatif ou hors des 5 bits du champ (0..31) : ce ne serait pas
//	    une lecture du champ, ce serait autre chose ;
//	(2) un identifiant d arme nul ou sentinelle (0 grenade, 1 melee, 2 vehicule dans
//	    `analysis`) : ces valeurs ne sont PAS des identifiants filmshell, les ecrire ici
//	    fabriquerait des jointures fausses avec `metadata.weapon_labels` ;
//	(3) un compte de tirs nul ou negatif : une ligne a zero n est pas une mesure, c est
//	    l ABSENCE de ligne qui porte le zero (et seulement quand la porte passe) ;
//	(4) un DOUBLON (indice, arme) dans la meme passe : deux lignes pour la meme cle feraient
//	    d un `SUM` un double comptage silencieux — exactement le defaut chiffre a 46,8 % qui a
//	    motive la refonte de `killer_victim_pairs` ;
//	(5) une reference `shots_fired` NEGATIVE : ce n est pas une reference, c est un defaut de
//	    lecture amont, et la porte le prendrait pour un ecart.
func validateWeaponShotsPlayer(pl *WeaponShotsPlayer, seen map[[2]uint64]bool) error {
	if pl.PlayerIndex < 0 || pl.PlayerIndex > 31 {
		return fmt.Errorf("PlayerIndex hors du champ 5 bits du fire-event (%d, attendu 0..31)",
			pl.PlayerIndex)
	}
	if pl.ShotsFired != nil && *pl.ShotsFired < 0 {
		return fmt.Errorf("ShotsFired negatif (%d) — ce n est pas une reference", *pl.ShotsFired)
	}
	for _, w := range pl.Weapons {
		if w.WeaponID <= weaponSentinelMax {
			return fmt.Errorf("WeaponID sentinelle ou nul (%d) — les identifiants 0/1/2 sont "+
				"les sentinelles grenade/melee/vehicule d `analysis`, pas des identifiants "+
				"filmshell joignables a metadata.weapon_labels", w.WeaponID)
		}
		if w.Shots <= 0 {
			return fmt.Errorf("nombre de tirs <= 0 pour l arme %d (%d) — une ligne a zero n est "+
				"pas une mesure, c est l absence de ligne qui porte le zero", w.WeaponID, w.Shots)
		}
		key := [2]uint64{uint64(pl.PlayerIndex), w.WeaponID}
		if seen[key] {
			return fmt.Errorf("doublon (indice %d, arme %d) dans la meme passe — un SUM le "+
				"compterait deux fois sans rien signaler", pl.PlayerIndex, w.WeaponID)
		}
		seen[key] = true
	}
	return nil
}
