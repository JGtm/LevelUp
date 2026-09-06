// Package duckdb — tactical_repo_univers.go : L'UNIVERS des lectures tactiques.
//
// Fichier separe de tactical_repo.go (phase 4 bis, 2026-09-06) : celui-la portait
// deja 486 lignes, et le perimetre par liste blanche l'aurait pousse au-dela du
// seuil de 500 (CLAUDE.md n 5). La coupure suit la seule frontiere naturelle du
// lecteur : ICI l'ensemble des matchs retenus et la composition de leurs equipes ;
// LA-BAS les trois lectures qui s'y accrochent.
//
// ─── LE PERIMETRE EST UNE LISTE BLANCHE, PLUS UN JEU D'AXES ────────────────────
//
// Jusqu'a la phase 4, l'univers etait filtre par `analysis.BuildNeighborsWhereClause`
// (playlist, mode, dates, issue). Ce vocabulaire-la ne sait pas lire les SESSIONS :
// elles vivent dans la base JOUEUR (`player_match_enrichment`), que ces requetes
// shared ne joignent pas — le filtre de session etait donc range dans les filtres
// IGNORES, et l'onglet servait la periode entiere sans le dire.
//
// Depuis la phase 4 bis, le client fait resoudre sa selection par le endpoint de
// filtres (`service.FilteredMatchIDs`, base joueur — periode OU sessions epinglees,
// contexte solo/escouade, cascade) et passe des match_id. Une seule definition du
// perimetre dans l'app, et le filtre de session MARCHE (arbitrage utilisateur du
// 2026-09-06).
package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"levelup/go-api/internal/domain"
)

// QTacticalUnivers est le SELECT des matchs RETENUS : ceux du joueur, sur la
// carte demandee quand il y en a une.
//
// LA CARTE EST OPTIONNELLE, ET C'EST UN PARAMETRE, PAS UNE CONCATENATION (ajout
// 2026-09-06, phase 3) : `? = ” OR mr.map_id = ?` neutralise le predicat quand
// l'appelant ne vise aucune carte (page Escouade). Assembler la clause en Go
// aurait fait DEUX chaines SQL pour un seul univers — et le garde-rail structurel
// campaign_exclusion_guard_test ne balaye qu'une constante, pas un assemblage.
//
// LE PERIMETRE (liste blanche de match_id, composition) est ajoute par
// `clausePerimetre` : sa longueur depend de l'appel, donc il ne peut pas vivre dans
// une constante. Ses valeurs sont des PARAMETRES LIES, jamais des litteraux.
//
// LE DRAPEAU `mesure` (ajout 2026-09-06, correction G2) dit si le journal des morts
// de ce match est LISIBLE : au moins une ligne publiable dans
// `match_kill_events_latest`. Un match dont le film n'a jamais ete decode — ou dont le
// film Theater a EXPIRE cote serveur — n'est pas un match a zero mort, c'est un match
// ILLISIBLE : il ne peut alimenter aucun numerateur, et le laisser au denominateur
// « par match » ferait varier la grandeur avec la couverture de film au lieu du jeu.
//
// `publishable` est exige ICI comme dans les deux lectures : sans lui, un match dont
// toutes les lignes sont ecartees compterait comme mesure sur cette page et pas sur la
// page Escouade, qui lit le meme journal filtre pareil.
//
// PREFIXE `Q` ET TOKEN CAMPAGNE (correction R2, revue du 2026-09-06) : le
// garde-rail structurel campaign_exclusion_guard_test ne balaye QUE les constantes
// nommees `Q<...>`. Sans le prefixe, un lecteur per-player passait sous son radar ;
// sans le token, les ~287 matchs Campagne d'un joueur Halo 5 entraient dans
// l'univers des rasters alors que l'Explorateur les masque. Le token est resolu au
// call site par resolveCampaignExclusion, qui connait le titre du joueur (no-op
// pour Infinite, qui n'a aucun match Campagne au registre).
const QTacticalUnivers = `
SELECT mr.match_id, COALESCE(mp.outcome, ?) AS outcome,
       EXISTS (SELECT 1 FROM match_kill_events_latest e
               WHERE e.match_id = mr.match_id AND e.publishable) AS mesure
FROM match_registry mr
JOIN match_participants mp ON mp.match_id = mr.match_id
WHERE mp.xuid = ? AND (? = '' OR mr.map_id = ?)` + campaignExclusionToken

// clauseAucunMatch : le predicat d'une liste blanche VIDE.
//
// UNE LISTE VIDE VEUT DIRE AUCUN MATCH, JAMAIS TOUS. C'est l'etat normal d'un filtre
// qui ne retient rien (une session sans match sur cette carte, une composition qui
// n'a jamais joue ensemble) ; le traduire par « pas de restriction » servirait
// l'historique ENTIER a qui vient d'en demander une tranche vide. `IN ()` n'est pas
// du SQL valide, d'ou ce predicat explicite plutot qu'une liste degeneree.
const clauseAucunMatch = "\n  AND FALSE"

// clauseCoequipier : UN xuid de la composition doit avoir joue DANS MON EQUIPE.
//
// Une clause par coequipier, toutes en AND : le match n'est retenu que si TOUS y
// etaient de mon cote. `c.team_id = mp.team_id` s'accroche a MA ligne de participant
// (l'alias `mp` de QTacticalUnivers / QTacticalMaps) — comparer a une equipe absolue
// n'aurait aucun sens, les numeros d'equipe se reattribuent a chaque partie.
const clauseCoequipier = `
  AND EXISTS (SELECT 1 FROM match_participants c
              WHERE c.match_id = mr.match_id AND c.xuid = ? AND c.team_id = mp.team_id)`

// clausePerimetre assemble le predicat de perimetre — liste blanche de match_id et
// composition — avec ses arguments LIES, dans l'ordre.
//
// Le fragment s'ajoute a la fin du WHERE : tous ses predicats sont des AND, l'ordre
// vis-a-vis de l'exclusion Campagne est donc indifferent.
func clausePerimetre(q domain.TacticalQuery) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(q.Matchs.IDs())+len(q.Coequipiers))

	if q.Matchs.Restreint() {
		ids := q.Matchs.IDs()
		if len(ids) == 0 {
			sb.WriteString(clauseAucunMatch)
		} else {
			sb.WriteString("\n  AND mr.match_id IN (" + Placeholders(len(ids)) + ")")
			args = append(args, ToAnySlice(ids)...)
		}
	}
	for _, xuid := range q.Coequipiers {
		sb.WriteString(clauseCoequipier)
		args = append(args, xuid)
	}
	return sb.String(), args
}

// universSQL assemble le SELECT de l'univers et ses arguments, token Campagne
// resolu pour le titre du joueur. `q.MapID` vide = toutes les cartes.
func (r *TacticalRepo) universSQL(q domain.TacticalQuery) (string, []any) {
	perim, perimArgs := clausePerimetre(q)
	args := append([]any{domain.OutcomeUnknown, q.PlayerXUID, q.MapID, q.MapID}, perimArgs...)
	return resolveCampaignExclusion(QTacticalUnivers, r.pdb.TitleSlug, "mr") + perim, args
}

// chargerUnivers lit les matchs retenus PUIS la composition de leurs equipes.
//
// Les equipes sont lues par une SECONDE requete qui re-selectionne l'univers en
// sous-requete, plutot que par une liste `IN (?, ?, ...)` construite en Go : le
// predicat serait alors ecrit a deux endroits au lieu d'un.
func (r *TacticalRepo) chargerUnivers(ctx context.Context, db *sql.DB, q domain.TacticalQuery) (domain.TacticalUnivers, error) {
	univ := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}

	selectSQL, args := r.universSQL(q)
	rows, err := db.QueryContext(ctx, selectSQL+" ORDER BY mr.match_id", args...)
	if err != nil {
		return univ, fmt.Errorf("univers: %w", err)
	}
	if err := scanRows(ctx, rows, "univers", func(sc rowScanner) error {
		var m domain.TacticalMatch
		if err := sc.Scan(&m.MatchID, &m.Outcome, &m.Mesure); err != nil {
			return err
		}
		univ.Matchs = append(univ.Matchs, m)
		return nil
	}); err != nil {
		return univ, err
	}
	if len(univ.Matchs) == 0 {
		return univ, nil
	}

	equipesSQL := `SELECT p.match_id, p.xuid, p.team_id FROM match_participants p
		WHERE p.match_id IN (SELECT u.match_id FROM (` + selectSQL + `) u)
		  AND p.xuid IS NOT NULL AND p.xuid <> '' AND p.team_id IS NOT NULL`
	rows, err = db.QueryContext(ctx, equipesSQL, args...)
	if err != nil {
		return univ, fmt.Errorf("equipes: %w", err)
	}
	err = scanRows(ctx, rows, "equipes", func(sc rowScanner) error {
		var matchID, xuid string
		var team int
		if err := sc.Scan(&matchID, &xuid, &team); err != nil {
			return err
		}
		parMatch := univ.Equipes[matchID]
		if parMatch == nil {
			parMatch = make(map[string]int)
			univ.Equipes[matchID] = parMatch
		}
		parMatch[xuid] = team
		return nil
	})
	return univ, err
}
