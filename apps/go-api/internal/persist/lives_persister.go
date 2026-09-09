// Package persist — lives_persister.go : ecriture INSERT-ONLY des DEUX tables des faits
// d'isolement, `match_lives` et `match_death_context` (plan Tactique, phase 7C).
//
// POURQUOI UN PERSISTER DEDIE, ET LE MEME PATRON QUE [KillSourcePersister] : ces deux tables
// naissent d'une PASSE DE DECODAGE d'un film, sur un match DEJA insere au registre. Le chemin
// builder (`SharedPersister.Persist`) est un no-op des que le match existe, et le film n'est
// jamais pret au sync primaire.
//
// ANTI-ART (ADR 0019/0026/0030) : INSERT purs, aucun UPDATE, aucun DELETE, aucun ON CONFLICT.
// « Remplacer » une passe consiste a en ecrire une nouvelle sous un `decode_pass` neuf ; ce sont
// les vues `_latest` qui ne rendent que la DERNIERE PASSE PAR MATCH. Ce fichier n'a donc rien a
// faire figurer dans l'allowlist de `internal/sync/no_art_patterns_test.go`.
//
// LES DEUX TABLES PARTAGENT LA MEME PASSE, ET C'EST LA RAISON D'UN SEUL PERSISTER : elles
// decrivent le meme decodage du meme film. Deux `decode_pass` differents rendraient les vues
// `_latest` incoherentes entre elles — le contexte d'une mort pourrait citer des vies que la vue
// des vies ne sert plus. Une transaction, un identifiant de passe, un instant.
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb (comme tous les
// persisters shared). `txBeginner` accepte aussi bien *sql.DB qu'un LeasedWriter.
package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// LivesBatch porte UNE passe de decodage : les vies nommees du film et le contexte de chacune
// de ses morts.
type LivesBatch struct {
	MatchID string
	// DecoderRev : version du decodeur. Requis, meme raison que pour le journal des morts —
	// c'est la seule facon de savoir QUELS matchs redecoder apres un changement de decodeur,
	// au lieu de tout redecoder.
	DecoderRev string
	Lives      []LifeInsert
	Contexts   []DeathContextInsert
}

// LifeInsert — une vie nommee, sur l'horloge du MATCH.
type LifeInsert struct {
	XUID    string `json:"xuid"`
	StartMS int64  `json:"start_ms"`
	EndMS   int64  `json:"end_ms"`
	// EndCause : comment la vie s'est terminee (`death` | `film_end` | `cut`).
	EndCause string `json:"end_cause"`
	// NamedBy : comment on sait a qui elle appartient (`death` | `closure` | `biped_creation` |
	// `biped_creation_propagee` | `elimination` | `exclusion_temporelle`). ORTHOGONAL a
	// EndCause — les confondre fait compter mort un survivant nomme par fermeture.
	NamedBy string `json:"named_by"`
}

// DeathContextInsert — qui entourait la victime a l'instant d'une mort du journal.
//
// La cle (MatchID, VictimXUID, TimeMS) est celle de `match_kill_events` : c'est ce qui rend les
// deux tables joignables sans rapprocher deux horloges.
type DeathContextInsert struct {
	VictimXUID string `json:"victim_xuid"`
	TimeMS     int64  `json:"time_ms"`
	// NearestTeammateM : nil = AUCUN coequipier visible. Une absence de mesure, jamais une
	// distance infinie ni un zero.
	NearestTeammateM    *float64 `json:"nearest_teammate_m,omitempty"`
	TeammatesVisible    int      `json:"teammates_visible"`
	TeammatesWaiting    int      `json:"teammates_waiting"`
	TeammatesOutOfSight int      `json:"teammates_out_of_sight"`
	TeammatesLeft       int      `json:"teammates_left"`
	TeammatesTotal      int      `json:"teammates_total"`
}

// LivesPersister ecrit une passe de faits d'isolement dans shared_matches_v2.duckdb.
type LivesPersister struct {
	db txBeginner
}

// NewLivesPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewLivesPersister(db txBeginner) *LivesPersister {
	return &LivesPersister{db: db}
}

// PersistPass ecrit les deux tables d'UN match en 1 transaction, en INSERT purs.
//
// Toutes les lignes portent le MEME `decode_pass` et le MEME `written_at` — c'est ce qui rend
// les vues `_latest` capables de retenir une generation entiere plutot qu'un melange.
//
// UNE PASSE SANS VIE N'ECRIT RIEN, PAS MEME SES CONTEXTES. Sans vies, le pont slot->xuid du film
// n'a pas ete construit, et un contexte calcule sans lui ne reposerait sur rien. La passe est
// ignoree et LOGGUEE : ecrire zero ligne en silence serait indistinguable d'un match sans mort.
func (p *LivesPersister) PersistPass(ctx context.Context, in LivesBatch) error {
	if err := validateLivesBatch(in); err != nil {
		return err
	}
	if len(in.Lives) == 0 {
		slog.WarnContext(ctx, "persist: passe des vies vide, aucune ligne ecrite "+
			"(pont slot->xuid non construit : le contexte des morts ne reposerait sur rien)",
			"match_id", in.MatchID, "decoder_rev", in.DecoderRev)
		return nil
	}

	pass, err := newDecodePassID()
	if err != nil {
		return fmt.Errorf("persist: %s: %w", in.MatchID, err)
	}
	return p.insertLivesPass(ctx, in, pass, time.Now().UTC())
}

// insertLivesPass ouvre la transaction et ecrit les deux tables.
func (p *LivesPersister) insertLivesPass(ctx context.Context, in LivesBatch, pass string, now time.Time) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_lives %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertLifeRows(ctx, tx, in, pass, now); err != nil {
		return err
	}
	if err := insertDeathContextRows(ctx, tx, in, pass, now); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_lives %s: %w", in.MatchID, err)
	}
	return nil
}

// insertLifeRows ecrit les vies. INSERT pur, table append-only.
func insertLifeRows(ctx context.Context, tx *sql.Tx, in LivesBatch, pass string, now time.Time) error {
	for i := range in.Lives {
		l := &in.Lives[i]
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_lives (
				match_id, decode_pass, decoder_rev, written_at,
				xuid, start_ms, end_ms, end_cause, named_by
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.MatchID, pass, in.DecoderRev, now,
			l.XUID, l.StartMS, l.EndMS, l.EndCause, l.NamedBy)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_lives %s/%s/%d: %w",
				in.MatchID, l.XUID, l.StartMS, err)
		}
	}
	return nil
}

// insertDeathContextRows ecrit le contexte des morts. INSERT pur, table append-only.
func insertDeathContextRows(ctx context.Context, tx *sql.Tx, in LivesBatch, pass string, now time.Time) error {
	for i := range in.Contexts {
		c := &in.Contexts[i]
		var proche any
		if c.NearestTeammateM != nil {
			proche = *c.NearestTeammateM
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO match_death_context (
				match_id, decode_pass, decoder_rev, written_at,
				victim_xuid, time_ms, nearest_teammate_m,
				teammates_visible, teammates_waiting, teammates_out_of_sight,
				teammates_left, teammates_total
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.MatchID, pass, in.DecoderRev, now,
			c.VictimXUID, c.TimeMS, proche,
			c.TeammatesVisible, c.TeammatesWaiting, c.TeammatesOutOfSight,
			c.TeammatesLeft, c.TeammatesTotal)
		if err != nil {
			return fmt.Errorf("persist: INSERT match_death_context %s/%s/%d: %w",
				in.MatchID, c.VictimXUID, c.TimeMS, err)
		}
	}
	return nil
}

// validateLivesBatch refuse ce qui ne peut pas etre une passe.
//
// LES REFUS SONT DES ERREURS, PAS DES CORRECTIONS. Un xuid vide, un `end_cause` inconnu ou un
// total de coequipiers incoherent sont des symptomes d'un producteur casse : les corriger ici
// (mettre un zero, choisir une cause par defaut) publierait une donnee inventee sous l'autorite
// de la base.
func validateLivesBatch(in LivesBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: LivesPersister.PersistPass: matchID vide")
	}
	if in.DecoderRev == "" {
		return fmt.Errorf("persist: LivesBatch.DecoderRev vide (%s) — sans elle, aucun "+
			"redecodage cible n'est possible apres un changement de decodeur", in.MatchID)
	}
	for i := range in.Lives {
		if err := validateLife(&in.Lives[i], in.MatchID, i); err != nil {
			return err
		}
	}
	for i := range in.Contexts {
		if err := validateDeathContext(&in.Contexts[i], in.MatchID, i); err != nil {
			return err
		}
	}
	return nil
}

func validateLife(l *LifeInsert, matchID string, i int) error {
	if l.XUID == "" {
		return fmt.Errorf("persist: match_lives %s ligne #%d: xuid vide — une vie anonyme "+
			"ne s'ecrit pas (elle fabriquerait un joueur fantome)", matchID, i)
	}
	switch l.EndCause {
	case CauseFinMort, CauseFinFilm, CauseFinCoupure:
	default:
		return fmt.Errorf("persist: match_lives %s ligne #%d: end_cause %q inconnue", matchID, i, l.EndCause)
	}
	switch l.NamedBy {
	case NommeParMort, NommeParFermeture, NommeParCreation, NommeParCreationPropagee,
		NommeParElimination, NommeParExclusionTemporelle:
	default:
		return fmt.Errorf("persist: match_lives %s ligne #%d: named_by %q inconnu", matchID, i, l.NamedBy)
	}
	if l.EndMS < l.StartMS {
		return fmt.Errorf("persist: match_lives %s ligne #%d: fin (%d) avant debut (%d)",
			matchID, i, l.EndMS, l.StartMS)
	}
	return nil
}

func validateDeathContext(c *DeathContextInsert, matchID string, i int) error {
	if c.VictimXUID == "" {
		return fmt.Errorf("persist: match_death_context %s ligne #%d: victim_xuid vide — "+
			"la ligne ne pourrait se joindre a aucune mort du journal", matchID, i)
	}
	somme := c.TeammatesVisible + c.TeammatesWaiting + c.TeammatesOutOfSight + c.TeammatesLeft
	if somme != c.TeammatesTotal {
		return fmt.Errorf("persist: match_death_context %s ligne #%d: visible(%d) + "+
			"waiting(%d) + out_of_sight(%d) + left(%d) = %d, mais total = %d — un etat non "+
			"nomme se cacherait dans l'ecart", matchID, i, c.TeammatesVisible,
			c.TeammatesWaiting, c.TeammatesOutOfSight, c.TeammatesLeft, somme, c.TeammatesTotal)
	}
	if c.NearestTeammateM != nil && c.TeammatesVisible == 0 {
		return fmt.Errorf("persist: match_death_context %s ligne #%d: une distance (%v) sans "+
			"aucun coequipier visible — seule une position repliquee autorise une distance",
			matchID, i, *c.NearestTeammateM)
	}
	return nil
}

// Les valeurs autorisees des deux colonnes de `match_lives`. Elles DOUBLENT les constantes du
// paquet `replay` (CauseVie*, NomPar*) et ce n'est pas un oubli : `persist` ne doit pas importer
// un paquet d'analyse pour valider une colonne, et le garde-rail
// `internal/archlint/no_life_cause_divergence_test.go` interdit aux deux jeux de diverger.
const (
	CauseFinMort    = "death"
	CauseFinFilm    = "film_end"
	CauseFinCoupure = "cut"

	NommeParMort      = "death"
	NommeParFermeture = "closure"
	// Voies du registre d'identite (lot E2, 2026-09-08) : le lien DIRECT corps -> joueur lu dans
	// le record de creation du bipede (et propage aux autres sejours du meme corps), l'elimination
	// sur le roster et l'exclusion temporelle. Oubliees a la livraison d'E2 : la passe
	// `backfill-killsource` du 2026-09-09 a refuse les 736 films (« named_by "biped_creation"
	// inconnu ») — le garde-rail `no_life_cause_divergence_test` ne lisait que lives.go.
	NommeParCreation            = "biped_creation"
	NommeParCreationPropagee    = "biped_creation_propagee"
	NommeParElimination         = "elimination"
	NommeParExclusionTemporelle = "exclusion_temporelle"
)
