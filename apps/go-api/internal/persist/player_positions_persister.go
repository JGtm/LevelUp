// Package persist — player_positions_persister.go : ecriture INSERT-ONLY d une PASSE DE
// POSITIONS JOUEURS dans `shared.match_player_positions`, projetee de l ARTEFACT DE REJEU.
//
// ─── POURQUOI UN PERSISTER DEDIE ──────────────────────────────────────────────────────────
//
// Meme raison que [BombStatsPersister], [UsageSummaryPersister] et [KillPositionPersister] :
// `SharedPersister.Persist` est un no-op des que le match existe deja dans `match_registry`, et
// une projection d artefact arrive TOUJOURS sur un match deja insere (l artefact n existe qu
// APRES le sync primaire).
//
// ─── CE QUI A CHANGE, ET POURQUOI (decision utilisateur 1, plan v2) ───────────────────────
//
// La table etait remplie par un OUTIL DE DIAGNOSTIC (`cmd/diag_weapons_v3 -positions -write`)
// en DELETE-then-INSERT, sur le handle de LECTURE du pool. Elle est desormais une PROJECTION DE
// L ARTEFACT, ecrite par les derivations post-rangement (`replayartifacts.Deriver`), dans le
// cycle de sync — donc sous le regime anti-ART comme toutes les autres.
//
// ─── ANTI-ART (ADR 0019/0026/0030) ────────────────────────────────────────────────────────
//
// INSERT purs. Aucun DELETE, aucun UPDATE, aucun ON CONFLICT — rien a faire figurer dans l
// allowlist de `no_art_patterns_test.go`, et `match_player_positions` y entre au contraire dans
// les tables PROTEGEES. « Remplacer » une projection consiste a en ecrire une nouvelle : la vue
// `match_player_positions_latest` ne rend que la DERNIERE PASSE PAR MATCH.
//
// ─── LA REPRISE SUR UN MATCH DEJA PROJETE ─────────────────────────────────────────────────
//
// Elle ecrit UNE NOUVELLE PASSE, sans rien lire ni effacer — c est exactement ce que fait
// [BombStatsPersister.PersistPass] pour `match_bomb_stats`. La difference avec les faits dates
// de la bombe (qui, eux, ne se reecrivent pas) tient au schema : `match_objective_events` a une
// PK naturelle et aucune vue `_latest`, cette table-ci a une semantique de generation. Rien a
// arbitrer ici.
//
// ─── UNE PASSE VIDE N EST PAS ECRITE ──────────────────────────────────────────────────────
//
// Ecrire zero ligne serait indistinguable d un match sans positions decodables, et la vue
// continuerait de servir la passe precedente sans qu on sache pourquoi. Une passe vide est donc
// ignoree et JOURNALISEE — meme doctrine que [KillPositionPersister.PersistPass].
//
// PRE-REQUIS : le caller doit tenir le lease RW sur shared_matches_v2.duckdb (ADR 0013).
// `txBeginner` accepte aussi bien *sql.DB qu un LeasedWriter.

package persist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// PlayerPositionRow est UNE position joueur d une passe.
//
// PAS DE XUID, ET CE N EST PAS UN OUBLI : la table est MATCH-LEVEL depuis sa creation (la
// delta-compression du film bloque l index par joueur, cf. `analysis/positions`). La projection
// de l artefact POURRAIT nommer le porteur — les trajectoires du document portent un xuid —
// mais l ajouter changerait la forme d une table que la carte de chaleur lit deja, ce qui n est
// pas ce que la decision 1 demande. Consigne en decouverte.
type PlayerPositionRow struct {
	// TimeMS : l instant, sur l AXE DU REJEU (frame x frameIntervalMs), et non sur l horloge du
	// film. La carte de chaleur ne lit que x/y ; l axe est ecrit pour que la ligne reste
	// datable, et sa nature est dite ici plutot que devinee plus tard.
	TimeMS int `json:"time_ms"`
	// X, Y, Z : la position monde. Z peut valoir 0 sur un document qui ne la publie pas.
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
	// Team : L IDENTIFIANT D EQUIPE DE LA BASE (`match_participants.team_id`), ou -1 quand
	// l equipe n est pas etablie. -1 est une valeur PLEINE (« pas attribuee »), jamais un
	// trou : le film ne porte pas l equipe, elle est JOINTE par le xuid du porteur.
	//
	// PAS SEULEMENT 0 ET 1, et c est une mesure : 4 matchs sur 1 959 de la base locale portent
	// plus de deux `team_id` distincts, avec des valeurs allant jusqu a 30 (modes a plus de
	// deux camps). La colonne transporte ce que la base dit, sans borne ni normalisation — la
	// carte de chaleur, elle, n offre aujourd hui que deux filtres : decision produit
	// consignee, hors perimetre de ce lot.
	Team int `json:"team"`
}

// PlayerPositionsBatch est UNE passe de positions pour UN match.
type PlayerPositionsBatch struct {
	// MatchID : l identite CANONIQUE du match (celle du registre), obligatoire.
	MatchID string `json:"match_id"`
	// Rows : les positions de la passe. Vide = rien a ecrire (cf. l en-tete).
	Rows []PlayerPositionRow `json:"rows"`
}

// PlayerPositionsPersister ecrit une passe de positions dans shared_matches_v2.duckdb.
type PlayerPositionsPersister struct {
	db txBeginner
}

// NewPlayerPositionsPersister construit un persister. `db` doit tenir le lease RW sur shared.
func NewPlayerPositionsPersister(db txBeginner) *PlayerPositionsPersister {
	return &PlayerPositionsPersister{db: db}
}

// PersistPass ecrit UNE passe en 1 transaction, en INSERT purs.
//
// Toutes les lignes portent le MEME `positions_pass` et le MEME `written_at` : c est ce qui rend
// la vue `match_player_positions_latest` capable de retenir une generation entiere.
func (p *PlayerPositionsPersister) PersistPass(ctx context.Context, in PlayerPositionsBatch) error {
	if in.MatchID == "" {
		return errors.New("persist: PlayerPositionsPersister.PersistPass: matchID vide")
	}
	if len(in.Rows) == 0 {
		slog.WarnContext(ctx, "persist: passe de positions vide, aucune ligne ecrite",
			"match_id", in.MatchID)
		return nil
	}
	pass, err := newDecodePassID()
	if err != nil {
		return err
	}

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("persist: BeginTx match_player_positions %s: %w", in.MatchID, err)
	}
	defer func() { _ = tx.Rollback() }() // no-op apres Commit

	stmt, err := tx.PrepareContext(ctx, insertPlayerPositionSQL)
	if err != nil {
		return fmt.Errorf("persist: prepare match_player_positions %s: %w", in.MatchID, err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now().UTC()
	for i := range in.Rows {
		r := in.Rows[i]
		if _, err := stmt.ExecContext(ctx,
			in.MatchID, pass, now, r.TimeMS, r.X, r.Y, r.Z, r.Team,
		); err != nil {
			return fmt.Errorf("persist: INSERT match_player_positions %s (#%d): %w", in.MatchID, i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("persist: Commit match_player_positions %s: %w", in.MatchID, err)
	}
	return nil
}

const insertPlayerPositionSQL = `
	INSERT INTO match_player_positions
		(match_id, positions_pass, written_at, time_ms, x, y, z, team)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
