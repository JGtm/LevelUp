// Package persist — flag_grabs_net_persister.go : ecriture INSERT-ONLY des PRISES DE DRAPEAU
// BRUTES ET NETTES lues du film.
//
// Une destination, une transaction :
//
//	shared.match_flag_grabs_net   les deux comptes par joueur, plus la fenetre de jonglage
//	                              appliquee (append-only, vue match_flag_grabs_net_latest —
//	                              ADR 0026)
//
// ─── POURQUOI UN PERSISTER DEDIE ──────────────────────────────────────────────────────────
//
// Meme raison que `BombStatsPersister` : `SharedPersister.Persist` est un no-op si
// `batch.Shared.Match == nil` et un skip si le match existe deja dans `match_registry` — or
// une passe de lecture d'artefact arrive TOUJOURS sur un match deja insere (le film n'est pas
// pret au sync primaire, il arrive un cycle plus tard).
//
// ─── ANTI-ART (ADR 0019/0026/0030) ────────────────────────────────────────────────────────
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT — rien a faire figurer dans
// l'allowlist de `no_art_patterns_test.go`, et `match_flag_grabs_net` y entre au contraire
// dans les tables PROTEGEES (ainsi que dans `sync/append_only_state_guard_test.go` et la
// regexp de `duckdb/no_raw_rating_reads_test.go` — recette ADR 0026 etape 5). « Remplacer »
// une passe consiste a en ecrire une nouvelle : la vue `match_flag_grabs_net_latest` rend la
// DERNIERE PASSE ENTIERE par match, jamais la derniere ligne par cle.
//
// ─── L UNITE D ECRITURE EST LA PASSE, ET `decode_pass` EST CE QUI LA NOMME ────────────────
//
// Toutes les lignes d une passe portent le MEME `decode_pass` — tire une seule fois par
// [newDecodePassID], meme doctrine que kill_events_persister.go et kill_opening_persister.go.
// C est lui, et non `written_at`, qui fait gagner une generation ENTIERE : un joueur que la
// nouvelle passe ne nomme plus est RETRACTE, et sa ligne ne survit pas avec l ANCIENNE
// fenetre de jonglage a cote des nouvelles.
//
// ─── UNE PASSE VIDE N'EST PAS UNE PASSE A ZERO ────────────────────────────────────────────
//
// Un match sans film, un film qui n'est pas du CTF, un titre sans fenetre declaree : la passe
// n'arrive pas jusqu'ici. Si elle arrive vide malgre tout, elle est IGNOREE et LOGGUEE —
// ecrire zero ligne serait indistinguable d'un match sans drapeau, et la vue continue de
// servir la passe precedente.
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

// FlagGrabsNetRow porte LES DEUX COMPTES D'UN JOUEUR pour un match.
//
// PAS DE POINTEURS ICI, contrairement a `BombPlayerStatsRow`, ET C'EST VOULU : les deux
// comptes viennent de LA MEME lecture du meme calque. Ils sont mesures ensemble ou pas du
// tout — il n'existe pas de cas « on a lu les brutes mais pas les nettes ». Le « non mesure »
// se dit donc par l'ABSENCE DE LIGNE, jamais par un NULL de colonne.
type FlagGrabsNetRow struct {
	// XUID du joueur, en decimal (la clef de match_participants). Obligatoire.
	//
	// UN JOUEUR DU ROSTER QUI N A JAMAIS TOUCHE LE DRAPEAU A UNE LIGNE A ZERO, et c est une
	// mesure : sur un match LU, « il n a rien pris » et « on n a pas regarde » sont deux
	// choses, et seule la presence de la ligne les separe.
	XUID string `json:"xuid"`
	// Raw : les prises BRUTES lues sur le calque de drapeau — le compteur officiel tel que le
	// film le rend, jonglage compris.
	Raw int `json:"flag_grabs_raw"`
	// Net : les memes prises, jonglage replie.
	Net int `json:"flag_grabs_net"`
}

// FlagGrabsNetBatch est LE RESULTAT D'UNE PASSE DE LECTURE D'UN ARTEFACT, cote drapeau.
//
// L'unite de production est le MATCH ENTIER, comme pour `BombStatsBatch` : c'est ce qui rend
// la vue `_latest` capable de retenir une generation entiere plutot qu'un melange.
type FlagGrabsNetBatch struct {
	MatchID string `json:"match_id"`
	// WindowMS est la fenetre de jonglage appliquee, en millisecondes. Elle est ECRITE SUR
	// CHAQUE LIGNE : sans elle, deux passes cuites sous deux fenetres differentes seraient
	// indistinguables dans la meme colonne. Obligatoire et strictement positive — une passe
	// sans fenetre n'a pas de regle, donc pas de prise nette.
	WindowMS int `json:"juggle_window_ms"`
	// Openings est le nombre d OUVERTURES DE PORTAGE que l oracle du film a comptees sur ce
	// match (`coverage.flagCarries.openings`). C est le DENOMINATEUR de `Raw` : les pistes ne
	// portent que les prises que le pont a su nommer ET situer. Valeur de MATCH, ecrite sur
	// chaque ligne de la passe. Peut valoir zero sur un artefact qui ne la publie pas — elle
	// se lit alors « non renseigne », jamais « aucune ouverture ».
	Openings int `json:"openings"`
	// Players : les joueurs ayant au moins une prise brute. Un joueur qui n'a jamais touche
	// le drapeau n'a pas de ligne — le noyau ne connait pas le roster et n'invente pas de
	// ligne a zero.
	Players []FlagGrabsNetRow `json:"players,omitempty"`
}

// FlagGrabsNetPersister ecrit une passe de prises de drapeau dans shared_matches_v2.duckdb.
type FlagGrabsNetPersister struct {
	db txBeginner
}

// NewFlagGrabsNetPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewFlagGrabsNetPersister(db txBeginner) *FlagGrabsNetPersister {
	return &FlagGrabsNetPersister{db: db}
}

// Persist ecrit le sous-batch `batch.Shared.FlagGrabsNet` s'il existe. No-op sinon.
//
// C'est le chemin BatchBuilder ; le chemin direct (lecture tardive d'un artefact, backfill)
// passe par [FlagGrabsNetPersister.PersistPass].
func (p *FlagGrabsNetPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: FlagGrabsNetPersister.Persist: batch nil")
	}
	if batch.Shared.FlagGrabsNet == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.FlagGrabsNet)
}

const insertFlagGrabsNetSQL = `
	INSERT INTO match_flag_grabs_net (
		match_id, decode_pass, xuid, written_at,
		flag_grabs_raw, flag_grabs_net, openings, juggle_window_ms
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
//
// Toutes les lignes portent le MEME `written_at` : c'est ce qui rend la vue
// `match_flag_grabs_net_latest` capable de retenir une generation entiere.
func (p *FlagGrabsNetPersister) PersistPass(ctx context.Context, in FlagGrabsNetBatch) error {
	if err := validateFlagGrabsNetBatch(in); err != nil {
		return err
	}
	if len(in.Players) == 0 {
		slog.WarnContext(ctx, "persist: passe flag_grabs_net vide, aucune ligne ecrite",
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
		return fmt.Errorf("persist: BeginTx match_flag_grabs_net %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertFlagGrabsNetRows(ctx, tx, in, pass, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_flag_grabs_net %s: %w", in.MatchID, err)
	}
	return nil
}

// insertFlagGrabsNetRows ecrit les lignes. `pass` et `now` sont partages par toutes les lignes
// de la passe : c'est `pass` qui arbitre la vue `_latest`, `now` ne fait que l'ordonner.
func insertFlagGrabsNetRows(ctx context.Context, tx *sql.Tx, in FlagGrabsNetBatch, pass string, now time.Time) error {
	stmt, err := tx.PrepareContext(ctx, insertFlagGrabsNetSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_flag_grabs_net %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	for _, pl := range in.Players {
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, pl.XUID, now, pl.Raw, pl.Net, in.Openings, in.WindowMS,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_flag_grabs_net %s/%s: %w", in.MatchID, pl.XUID, err)
		}
	}
	return nil
}

// validateFlagGrabsNetBatch : ce que le persister REFUSE. La validation passe AVANT la
// transaction, donc un refus ne laisse aucune ligne derriere lui.
//
//	(1) un match sans identifiant ;
//	(2) une fenetre nulle ou negative : sans regle il n'y a pas de prise nette, et ecrire la
//	    colonne a zero ferait croire a une mesure sous une fenetre instantanee ;
//	(3) un XUID vide : la ligne n'aurait pas de proprietaire, et la vue `_latest` partitionne
//	    justement sur (match_id, xuid) ;
//	(4) un DOUBLON de xuid dans la meme passe : la vue choisirait arbitrairement l'une des
//	    deux lignes ;
//	(5) un compte NEGATIF, ou des prises nettes SUPERIEURES aux brutes : ce n'est pas une
//	    mesure, c'est un defaut de lecture amont — le meme controle que le rapport d'etape 0
//	    a passe sur 522 comparaisons.
func validateFlagGrabsNetBatch(in FlagGrabsNetBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: FlagGrabsNetBatch.MatchID vide")
	}
	if in.WindowMS <= 0 {
		return fmt.Errorf("persist: %s: juggle_window_ms = %d — une passe sans fenetre n'a pas "+
			"de regle, donc pas de prise nette", in.MatchID, in.WindowMS)
	}
	if in.Openings < 0 {
		return fmt.Errorf("persist: %s: openings = %d — ce n'est pas un compte", in.MatchID, in.Openings)
	}
	vus := make(map[string]bool, len(in.Players))
	for i := range in.Players {
		if err := validateFlagGrabsNetRow(&in.Players[i], vus); err != nil {
			return fmt.Errorf("persist: %s joueur #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validateFlagGrabsNetRow applique les controles (3), (4) et (5) a UNE ligne.
func validateFlagGrabsNetRow(pl *FlagGrabsNetRow, vus map[string]bool) error {
	if pl.XUID == "" {
		return errors.New("XUID vide — une ligne de prises sans proprietaire ne se lit pas")
	}
	if vus[pl.XUID] {
		return fmt.Errorf("doublon de xuid %q dans la meme passe — la vue _latest choisirait "+
			"arbitrairement l'une des deux lignes", pl.XUID)
	}
	vus[pl.XUID] = true
	if pl.Raw < 0 || pl.Net < 0 {
		return fmt.Errorf("compte negatif pour %q (brut=%d net=%d) — ce n'est pas une mesure",
			pl.XUID, pl.Raw, pl.Net)
	}
	if pl.Net > pl.Raw {
		return fmt.Errorf("prises nettes > brutes pour %q (%d > %d) — la regle ne cree jamais "+
			"de prise", pl.XUID, pl.Net, pl.Raw)
	}
	return nil
}
