// Package persist — bomb_stats_persister.go : ecriture INSERT-ONLY des STATISTIQUES D'ASSAUT
// reconstruites du film, et des FAITS DATES de la bombe.
//
// Deux destinations, une seule transaction :
//
//	shared.match_bomb_stats               les 5 statistiques par joueur (append-only, vue
//	                                      match_bomb_stats_latest — ADR 0026)
//	shared.match_objective_events         les faits dates (objective_type = 'bomb',
//	  + match_objective_event_players     event_type bomb_armed | bomb_detonated)
//
// ─── POURQUOI UN PERSISTER DEDIE ──────────────────────────────────────────────────────────
//
// Meme raison que `KillSourcePersister` et `WeaponShotsPersister` : `SharedPersister.Persist`
// est un no-op si `batch.Shared.Match == nil` et un skip si le match existe deja dans
// `match_registry` — or une passe de decodage de film arrive TOUJOURS sur un match deja insere
// (le film n'est pas pret au sync primaire, il arrive un cycle plus tard).
//
// ─── ANTI-ART (ADR 0019/0026/0030) ────────────────────────────────────────────────────────
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT — rien a faire figurer dans
// l'allowlist de `no_art_patterns_test.go`, et `match_bomb_stats` y entre au contraire dans les
// tables PROTEGEES. « Remplacer » une passe de statistiques consiste a en ecrire une nouvelle :
// la vue `match_bomb_stats_latest` ne rend que la derniere ligne par (match_id, xuid).
//
// ⚠ `ObjectiveEventsRepo.WriteMatch` (platform/duckdb) fait, LUI, un DELETE-then-INSERT, et son
// en-tete le documente comme « hors chemin live ». Il n'est NI appele NI modifie ici.
//
// ─── LES FAITS DATES S'ECRIVENT UNE FOIS PAR MATCH, ET C'EST UN CHOIX ─────────────────────
//
// `match_objective_events` a une PK NATURELLE `(match_id, seq)` et AUCUNE vue `_latest` : elle
// n'a pas de mecanique de generation. Un INSERT-only qui re-ecrirait les memes faits ne pourrait
// donc que (a) violer la PK, ou (b) empiler deux generations que tout lecteur compterait deux
// fois. Ce persister prend la seule troisieme voie disponible sans DELETE : il ECRIT SI LE MATCH
// N'A PAS DEJA DE FAITS `bomb`, et sinon il ne touche a rien et le dit au journal. Les
// statistiques, elles, sont ecrites dans tous les cas — la table qui les porte, elle, SAIT
// remplacer une generation.
//
// Consequence assumee : apres un changement de decodeur, les statistiques d'un match se
// rafraichissent et ses faits dates NON. Remplacer les faits demande une semantique de
// generation que le schema de `match_objective_events` ne porte pas — c'est un chantier de
// schema, pas une ligne de code a glisser ici.
//
// ─── UN ARMEMENT SANS ACTEUR RESOLU S'ECRIT SANS ACTEUR ───────────────────────────────────
//
// Le noyau (E2) publie l'armement meme quand la jointure ne nomme personne (slot non ponte, ou
// aucun lacher dans la fenetre) : 4 des 13 armements du corpus. Une telle ligne entre dans
// `match_objective_events` SANS aucune ligne dans `match_objective_event_players`. Jamais un
// acteur devine, jamais un armement escamote.
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

// Vocabulaire des faits dates de la bombe. Ces quatre valeurs sont des RECOPIES de constantes
// qui appartiennent a `analysis` (`replay.BombEventArmed` / `replay.BombEventDetonated`,
// `objectiveevents.ObjectiveTypeBomb` / `objectiveevents.RoleScorer`). La recopie est volontaire
// — faire dependre `persist` du decodeur de film pour quatre chaines serait un couplage
// disproportionne — et elle est TENUE par un garde-rail : bomb_stats_sentinels_test.go echoue le
// jour ou l'une des quatre diverge de sa source (meme dispositif que weaponSentinelMax).
const (
	// BombEventArmed : la bombe a ete ARMEE (anneau `ti=12 i14` du film).
	BombEventArmed = "bomb_armed"
	// BombEventDetonated : la bombe a EXPLOSE (statborg `comp 0` canal A).
	BombEventDetonated = "bomb_detonated"
	// bombObjectiveType : la valeur `objective_type` de la famille bombe, deja prevue au schema
	// de `match_objective_events`.
	bombObjectiveType = "bomb"
	// bombEventRoleScorer : le role de l'acteur d'un fait de bombe.
	bombEventRoleScorer = "scorer"
)

// BombPlayerStatsRow porte LES CINQ STATISTIQUES D'UN JOUEUR pour un match.
//
// ⚠ TOUS LES CHAMPS DE MESURE SONT DES POINTEURS, ET C'EST LE COEUR DU DESIGN : `nil` veut dire
// NON MESURE et s'ecrit NULL. Ecrire 0 a la place dirait « mesure : rien ne s'est passe » la ou
// la verite est « on n'a pas regarde » — et un agregat sommerait ces faux zeros sans rien
// signaler. Le noyau d'extraction pilote ces pointeurs par des temoins de LECTURE ; ce persister
// ne fait que les transporter, il ne comble aucun trou.
type BombPlayerStatsRow struct {
	// XUID du joueur, en decimal (la clef de match_participants). Obligatoire : un bot n'a pas
	// de XUID, donc pas de ligne — son absence n'est pas un zero.
	XUID string `json:"xuid"`
	// Detonations : explosions creditees au joueur (statborg comp 0 canal A).
	Detonations *int `json:"bomb_detonations,omitempty"`
	// Arms : armements attribues par la jointure lacher <-> armement (E2).
	Arms *int `json:"bomb_arms,omitempty"`
	// Grabs : ramassages de la bombe (periodes du canal des armes tenues).
	Grabs *int `json:"bomb_grabs,omitempty"`
	// TimeAsCarrierSeconds : temps bombe en main, periodes FERMEES seulement.
	TimeAsCarrierSeconds *float64 `json:"time_as_bomb_carrier_seconds,omitempty"`
	// CarriersKilled : porteurs de bombe tues.
	CarriersKilled *int `json:"bomb_carriers_killed,omitempty"`
}

// BombEventRow est UN FAIT DATE de la bombe, destine a `match_objective_events`.
type BombEventRow struct {
	// EventType : BombEventArmed ou BombEventDetonated. Aucune autre valeur n'est acceptee —
	// cette table est mode-agnostique, et y laisser entrer un type inconnu la rendrait
	// illisible pour ses autres lecteurs.
	EventType string `json:"event_type"`
	// TimeMS : l'instant, sur l'HORLOGE DU FILM — la meme que celle des `ObjectiveAction`, donc
	// celle qu'ecrivent deja les autres producteurs de cette table. Ce persister ne recale
	// rien : la jointure d'E2 a fait son recalage en amont, sur ses propres entrees.
	TimeMS int `json:"time_ms"`
	// XUID de l'acteur. VIDE = aucun acteur resolu : le fait est publie quand meme, sans ligne
	// dans `match_objective_event_players`.
	XUID string `json:"xuid,omitempty"`
	// TeamID : l'equipe creditee. nil = inconnue (colonne NULL).
	TeamID *int `json:"team_id,omitempty"`
	// Source : la provenance du decodage, en toutes lettres (vocabulaire de
	// `objectiveevents.Source*`). Obligatoire : un fait qui ne dit pas d'ou il vient laisse un
	// lecteur lui preter la precision qu'il veut.
	Source string `json:"source"`
	// Confidence : la precision temporelle (vocabulaire de `objectiveevents.Confidence*`).
	// Obligatoire, pour la meme raison.
	Confidence string `json:"confidence"`
}

// BombStatsBatch est LE RESULTAT D'UNE PASSE DE DECODAGE D'UN FILM, cote Assaut.
//
// L'unite de production est le MATCH ENTIER, comme pour `KillSourceBatch` et
// `WeaponShotsBatch` : c'est ce qui rend la vue `_latest` capable de retenir une generation
// entiere plutot qu'un melange.
type BombStatsBatch struct {
	MatchID string `json:"match_id"`
	// Players : les joueurs nommes par au moins une source. Un joueur qu'aucune source ne nomme
	// n'a pas de ligne — le noyau ne connait pas le roster et n'invente pas de ligne a zero.
	Players []BombPlayerStatsRow `json:"players,omitempty"`
	// Events : les faits dates (armements, explosions).
	Events []BombEventRow `json:"events,omitempty"`
}

// BombStatsPersister ecrit une passe d'Assaut dans shared_matches_v2.duckdb.
type BombStatsPersister struct {
	db txBeginner
}

// NewBombStatsPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewBombStatsPersister(db txBeginner) *BombStatsPersister {
	return &BombStatsPersister{db: db}
}

// Persist ecrit le sous-batch `batch.Shared.BombStats` s'il existe. No-op sinon.
//
// C'est le chemin BatchBuilder ; le chemin direct (completion tardive d'un film, backfill) passe
// par [BombStatsPersister.PersistPass].
func (p *BombStatsPersister) Persist(ctx context.Context, batch *MatchBatch) error {
	if batch == nil {
		return errors.New("persist: BombStatsPersister.Persist: batch nil")
	}
	if batch.Shared.BombStats == nil {
		return nil
	}
	return p.PersistPass(ctx, *batch.Shared.BombStats)
}

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
//
// Toutes les lignes de statistiques portent le MEME `written_at` : c'est ce qui rend la vue
// `match_bomb_stats_latest` capable de retenir une generation entiere.
//
// Une passe VIDE (ni joueur ni fait) est ignoree et LOGGUEE : ecrire zero ligne serait
// indistinguable d'un match sans Assaut, et la vue continue de servir la passe precedente.
func (p *BombStatsPersister) PersistPass(ctx context.Context, in BombStatsBatch) error {
	if err := validateBombStatsBatch(in); err != nil {
		return err
	}
	if len(in.Players) == 0 && len(in.Events) == 0 {
		slog.WarnContext(ctx, "persist: passe bomb_stats vide, aucune ligne ecrite",
			"match_id", in.MatchID)
		return nil
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_bomb_stats %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	if err := insertBombStatsRows(ctx, tx, in, time.Now().UTC()); err != nil {
		return err
	}
	ecrits, err := insertBombObjectiveEvents(ctx, tx, in)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_bomb_stats %s: %w", in.MatchID, err)
	}
	if len(in.Events) > 0 && ecrits == 0 {
		// Ni une erreur ni un silence : le match portait deja ses faits dates, la passe les a
		// laisses en place (cf. l'en-tete — cette table n'a pas de semantique de generation).
		slog.InfoContext(ctx, "persist: faits dates de bombe deja presents, non reecrits",
			"match_id", in.MatchID, "faits_proposes", len(in.Events))
	}
	return nil
}

const insertBombStatsSQL = `
	INSERT INTO match_bomb_stats (
		match_id, xuid, written_at,
		bomb_detonations, bomb_arms, bomb_grabs,
		time_as_bomb_carrier_seconds, bomb_carriers_killed
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

// insertBombStatsRows ecrit les lignes de statistiques. `now` est partage par toutes les lignes
// de la passe — c'est l'arbitre de la vue `_latest`.
func insertBombStatsRows(ctx context.Context, tx *sql.Tx, in BombStatsBatch, now time.Time) error {
	if len(in.Players) == 0 {
		return nil
	}
	stmt, err := tx.PrepareContext(ctx, insertBombStatsSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_bomb_stats %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	for _, pl := range in.Players {
		// Les pointeurs partent tels quels : database/sql ecrit NULL pour un pointeur nil, et
		// c'est exactement la distinction « non mesure » que le schema porte.
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pl.XUID, now,
			pl.Detonations, pl.Arms, pl.Grabs,
			pl.TimeAsCarrierSeconds, pl.CarriersKilled,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_bomb_stats %s/%s: %w", in.MatchID, pl.XUID, err)
		}
	}
	return nil
}

const insertObjectiveEventSQL = `
	INSERT INTO match_objective_events (
		match_id, seq, time_ms, objective_type, event_type,
		team_id, objective_id, value, source, confidence, details
	) VALUES (?, ?, ?, ?, ?, ?, NULL, NULL, ?, ?, NULL)`

const insertObjectiveEventPlayerSQL = `
	INSERT INTO match_objective_event_players (match_id, seq, xuid, role)
	VALUES (?, ?, ?, ?)`

// insertBombObjectiveEvents ecrit les faits dates SI le match n'en porte pas deja de la famille
// bombe. Retourne le nombre de faits reellement ecrits (0 = deja presents, ou aucun propose).
//
// `seq` est alloue APRES le maximum deja present sur le match : la PK naturelle
// `(match_id, seq)` est partagee avec les autres producteurs de cette table, et repartir de 0
// entrerait en collision avec eux au lieu de s'ajouter a leur suite.
func insertBombObjectiveEvents(ctx context.Context, tx *sql.Tx, in BombStatsBatch) (int, error) {
	if len(in.Events) == 0 {
		return 0, nil
	}
	deja, err := countBombObjectiveEvents(ctx, tx, in.MatchID)
	if err != nil {
		return 0, err
	}
	if deja > 0 {
		return 0, nil
	}
	seq, err := nextObjectiveEventSeq(ctx, tx, in.MatchID)
	if err != nil {
		return 0, err
	}
	for _, ev := range in.Events {
		if err := insertOneBombEvent(ctx, tx, in.MatchID, seq, ev); err != nil {
			return 0, err
		}
		seq++
	}
	return len(in.Events), nil
}

// insertOneBombEvent ecrit le fait et, SI un acteur est resolu, sa ligne d'acteur. Un fait sans
// acteur reste un fait : il s'ecrit seul.
func insertOneBombEvent(ctx context.Context, tx *sql.Tx, matchID string, seq int, ev BombEventRow) error {
	if _, err := tx.ExecContext(ctx, insertObjectiveEventSQL,
		matchID, seq, ev.TimeMS, bombObjectiveType, ev.EventType,
		ev.TeamID, ev.Source, ev.Confidence,
	); err != nil {
		return fmt.Errorf("persist: INSERT match_objective_events %s seq=%d (%s): %w",
			matchID, seq, ev.EventType, err)
	}
	if ev.XUID == "" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertObjectiveEventPlayerSQL,
		matchID, seq, ev.XUID, bombEventRoleScorer,
	); err != nil {
		return fmt.Errorf("persist: INSERT match_objective_event_players %s seq=%d: %w",
			matchID, seq, err)
	}
	return nil
}

// countBombObjectiveEvents compte les faits de la famille bombe deja presents sur le match.
func countBombObjectiveEvents(ctx context.Context, tx *sql.Tx, matchID string) (int, error) {
	var n int
	err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM match_objective_events WHERE match_id = ? AND objective_type = ?`,
		matchID, bombObjectiveType).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("persist: lecture match_objective_events %s: %w", matchID, err)
	}
	return n, nil
}

// nextObjectiveEventSeq rend le premier `seq` libre du match, TOUS objectifs confondus.
func nextObjectiveEventSeq(ctx context.Context, tx *sql.Tx, matchID string) (int, error) {
	var seq int
	err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), -1) + 1 FROM match_objective_events WHERE match_id = ?`,
		matchID).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("persist: seq libre match_objective_events %s: %w", matchID, err)
	}
	return seq, nil
}

// validateBombStatsBatch : ce que le persister REFUSE. La validation passe AVANT la transaction,
// donc un refus ne laisse aucune ligne derriere lui.
func validateBombStatsBatch(in BombStatsBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: BombStatsBatch.MatchID vide")
	}
	vus := make(map[string]bool, len(in.Players))
	for i := range in.Players {
		if err := validateBombPlayerRow(&in.Players[i], vus); err != nil {
			return fmt.Errorf("persist: %s joueur #%d: %w", in.MatchID, i, err)
		}
	}
	faits := make(map[[3]string]bool, len(in.Events))
	for i := range in.Events {
		if err := validateBombEventRow(&in.Events[i], faits); err != nil {
			return fmt.Errorf("persist: %s fait #%d: %w", in.MatchID, i, err)
		}
	}
	return nil
}

// validateBombPlayerRow : ce qui est REFUSE au niveau de la ligne de statistiques.
//
//	(1) un XUID vide : la ligne n'aurait pas de proprietaire, et la vue `_latest` partitionne
//	    justement sur (match_id, xuid) — deux lignes sans xuid s'ecraseraient l'une l'autre ;
//	(2) un DOUBLON de xuid dans la meme passe : deux lignes pour la meme clef feraient de la
//	    vue `_latest` un tirage arbitraire entre elles ;
//	(3) un compte NEGATIF : ce n'est pas une mesure, c'est un defaut de lecture amont ;
//	(4) une ligne dont AUCUN champ n'est mesure : elle n'affirmerait rien tout en occupant la
//	    clef (match_id, xuid) dans la vue, masquant une passe precedente qui, elle, mesurait.
func validateBombPlayerRow(pl *BombPlayerStatsRow, vus map[string]bool) error {
	if pl.XUID == "" {
		return errors.New("XUID vide — une ligne de statistiques sans proprietaire ne se lit pas")
	}
	if vus[pl.XUID] {
		return fmt.Errorf("doublon de xuid %q dans la meme passe — la vue _latest choisirait "+
			"arbitrairement l'une des deux lignes", pl.XUID)
	}
	vus[pl.XUID] = true
	compteurs := map[string]*int{
		"bomb_detonations": pl.Detonations, "bomb_arms": pl.Arms,
		"bomb_grabs": pl.Grabs, "bomb_carriers_killed": pl.CarriersKilled,
	}
	for nom, v := range compteurs {
		if v != nil && *v < 0 {
			return fmt.Errorf("%s negatif (%d) — ce n'est pas une mesure", nom, *v)
		}
	}
	if pl.TimeAsCarrierSeconds != nil && *pl.TimeAsCarrierSeconds < 0 {
		return fmt.Errorf("time_as_bomb_carrier_seconds negatif (%v)", *pl.TimeAsCarrierSeconds)
	}
	if pl.Detonations == nil && pl.Arms == nil && pl.Grabs == nil &&
		pl.TimeAsCarrierSeconds == nil && pl.CarriersKilled == nil {
		return fmt.Errorf("aucune mesure pour %q — une ligne integralement NULL n'affirme rien "+
			"et masquerait dans la vue _latest une passe qui, elle, mesurait", pl.XUID)
	}
	return nil
}

// validateBombEventRow : ce qui est REFUSE au niveau du fait date.
//
//	(1) un `event_type` hors des deux valeurs connues : cette table est partagee avec les
//	    autres modes a objectif, y laisser entrer un type inconnu la rend illisible ;
//	(2) un instant NEGATIF : l'horloge du film part de zero ;
//	(3) une `source` ou une `confidence` vide : un fait qui ne dit pas d'ou il vient laisse un
//	    lecteur lui preter la precision qu'il veut ;
//	(4) un DOUBLON (type, instant, acteur) : deux lignes pour le meme fait le compteraient
//	    deux fois, et rien dans la table ne permettrait de s'en apercevoir.
func validateBombEventRow(ev *BombEventRow, faits map[[3]string]bool) error {
	if ev.EventType != BombEventArmed && ev.EventType != BombEventDetonated {
		return fmt.Errorf("event_type %q inconnu (attendu %q ou %q)",
			ev.EventType, BombEventArmed, BombEventDetonated)
	}
	if ev.TimeMS < 0 {
		return fmt.Errorf("time_ms negatif (%d) — l'horloge du film part de zero", ev.TimeMS)
	}
	if ev.Source == "" || ev.Confidence == "" {
		return fmt.Errorf("source (%q) et confidence (%q) obligatoires — un fait qui ne dit pas "+
			"d'ou il vient laisse un lecteur lui preter la precision qu'il veut",
			ev.Source, ev.Confidence)
	}
	cle := [3]string{ev.EventType, fmt.Sprint(ev.TimeMS), ev.XUID}
	if faits[cle] {
		return fmt.Errorf("doublon (%s, %d ms, acteur %q) dans la meme passe",
			ev.EventType, ev.TimeMS, ev.XUID)
	}
	faits[cle] = true
	return nil
}
