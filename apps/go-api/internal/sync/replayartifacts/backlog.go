package replayartifacts

// backlog.go — LE RATTRAPAGE DE L'ÉTAPE 1.58.
//
// # LE DÉFAUT QUE CE FICHIER CORRIGE
//
// L'étape ne voyait que `insertedIDs` : UNE tentative, à l'instant de l'insertion du match.
// Or le film Theater se publie APRÈS la partie — il n'est pas rare qu'il manque encore au
// moment où le cycle insère le match. Une tentative unique, ratée, n'était jamais reprise :
// le match n'ayant plus jamais le statut « inséré », plus rien ne le regardait.
//
// Mesure du 2026-09-01 : UN artefact sur les 222 matchs des 90 derniers jours, et les 50
// artefacts locaux sont tous d'anciens matchs cuits à la main. L'étape 1.57, elle, a un
// rattrapage depuis le 2026-08-29 (`killcollector.backlogAJour` + `ordonnancer`) ; ce fichier
// est son pendant.
//
// # CE QUE LE RATTRAPAGE REGARDE, ET CE QU'IL NE REGARDE PAS
//
// La QUEUE RÉCENTE du registre, et elle seule : les [BacklogHorizon] matchs les plus récents
// de la fenêtre de rétention, dont l'artefact est ABSENT du disque. Trois bornes, trois
// raisons :
//
//	horizon de lecture      Une fois la queue récente couverte, le retard est VIDE et l'étape
//	                        s'arrête d'elle-même. Sans cette borne, un dépôt de plusieurs
//	                        milliers de matchs verrait le fil de l'eau réclamer indéfiniment
//	                        des films expirés depuis des années. Le solde historique relève de
//	                        `levelup backfill-replay`, et c'est déjà ce que dit `maxPerCycle`.
//	présence du FICHIER     Le prédicat le moins cher qui existe : un `os.Stat`. Le prédicat
//	                        complet (version de schéma + compteurs de joueur) lit et
//	                        désérialise l'artefact ENTIER — acceptable sur les quelques matchs
//	                        insérés d'un cycle, ruineux sur soixante-quatre à chaque cycle.
//	                        Le rattrapage répond donc à « ce match n'a AUCUN rejeu », qui est
//	                        exactement le défaut mesuré ; la réparation des artefacts appauvris
//	                        reste au chemin des matchs insérés, qui a les faits sous la main.
//	marqueur terminal       Un film expiré côté serveur Halo (~29 % du corpus) ne reviendra
//	                        jamais. Le registre le SAIT et le dit dans `backfill_completed`.
//
// # RÉSIDU ASSUMÉ
//
// Un match récent dont le film n'existera jamais mais que rien n'a encore marqué est retenté à
// chaque cycle, jusqu'à ce que soixante-quatre matchs plus récents l'aient chassé de
// l'horizon. Le coût d'une tentative est UN appel qui rend « absent », et le lot est plafonné :
// le pire cas est cinq appels stériles par cycle.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/sync/matchflags"
)

// BacklogHorizon borne la LECTURE du retard, pas le travail (c'est maxPerCycle qui borne le
// travail). Même valeur que `killcollector.PostSyncBacklogHorizon`, pour la même raison :
// ramener des centaines d'identifiants pour en traiter cinq est du gaspillage.
const BacklogHorizon = 64

// BudgetParCycle : LE GARDE-FOU DE DURÉE, et il n'est pas négociable.
//
// `maxPerCycle` compte des matchs, pas des secondes — or un film coûte de quelques secondes à
// plus d'une minute de décodage, et le pont disque télécharge en plus des dizaines de Mo. Sans
// budget, cinq gros films suffiraient à rallonger un cycle de synchronisation, et tout ce qui
// vient après (PSA, agrégats, médias) attendrait derrière. Le budget arrête la passe ENTRE
// deux matchs ; le solde repart au cycle suivant, l'étape étant idempotente (l'artefact déjà
// à jour n'est pas reconstruit). Valeur alignée sur `killcollector.PostSyncBudget`.
const BudgetParCycle = 5 * time.Minute

// bitFilmAbsent : le marqueur terminal « film 404/expiré, 0 chunk disponible » du registre.
//
// ⚠ IL EST PARTAGÉ, ET CE N'EST PAS UN DÉTAIL. Il est POSÉ par l'étape 1.55 (weapon kills) et
// LU par le rattrapage de l'étape 1.57 (`killcollector.backlogAJour`) ; ce fichier en est le
// troisième usager. Quiconque supprime l'étape 1.55 supprime le seul poseur de ce marqueur, et
// prive du même coup les rattrapages 1.57 et 1.58 de leur seule protection contre les films
// irrécupérables — la conséquence doit être traitée dans le même lot, pas découverte après.
const bitFilmAbsent = matchflags.MBitWeaponKillsNoFilm

// candidatARattraper : une ligne de la queue récente, telle que le registre la rend.
type candidatARattraper struct {
	matchID string
	rawName string
	mapID   string
}

// candidatsARattraper rend le travail de rattrapage du cycle (au plus `plafond` matchs) et le
// RETARD RESTANT dans l'horizon après ce prélèvement.
//
// `deja` porte les matchs que le chemin des insérés a déjà retenus : les reprendre ici les
// ferait traiter deux fois dans le même cycle.
func candidatsARattraper(
	ctx context.Context, sharedDB, metaDB *sql.DB, d Deps, deja map[string]bool, plafond int,
) (work []buildWork, restant int) {
	if sharedDB == nil {
		return nil, 0
	}
	enRetard := lireQueueRecente(ctx, sharedDB, d, deja)
	plafond = min(max(plafond, 0), len(enRetard))
	// La résolution du nom EN coûte une requête au catalogue PAR MATCH : elle ne se fait que
	// sur ce que le cycle prend réellement, jamais sur tout l'horizon.
	for _, c := range enRetard[:plafond] {
		work = append(work, buildWork{matchID: c.matchID, mapNames: nomsDeCarte(ctx, metaDB, c)})
	}
	return work, len(enRetard) - plafond
}

// nomsDeCarte rend les identités de carte candidates, dans l'ordre de préférence (nom EN du
// catalogue d'abord, nom brut du registre ensuite) — même règle que la sélection des insérés.
func nomsDeCarte(ctx context.Context, metaDB *sql.DB, c candidatARattraper) []string {
	var names []string
	if en := resolveMapNameEN(ctx, metaDB, strings.TrimSpace(c.mapID)); en != "" {
		names = append(names, en)
	}
	if raw := strings.TrimSpace(c.rawName); raw != "" {
		names = append(names, raw)
	}
	return names
}

// lireQueueRecente lit l'horizon au registre puis écarte, sur DISQUE, ceux qui ont déjà un
// artefact. L'ordre du registre (du plus récent au plus vieux) est conservé.
func lireQueueRecente(ctx context.Context, sharedDB *sql.DB, d Deps, deja map[string]bool) []candidatARattraper {
	args := []any{bitFilmAbsent}
	if d.RetentionMonths > 0 {
		args = append(args, time.Now().UTC().AddDate(0, -d.RetentionMonths, 0))
	}
	rows, err := sharedDB.QueryContext(ctx, requeteQueueRecente(d.RetentionMonths), append(args, BacklogHorizon)...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — retard illisible", "err", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	paths := titlePkg.NewPathResolver(d.RepoRoot)
	var out []candidatARattraper
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		if err := rows.Scan(&id, &rawName, &mapID); err != nil {
			slog.WarnContext(ctx, "post-sync: rejeu 2D — retard (scan)", "err", err)
			return out
		}
		if deja[id] || artefactPresent(paths.ReplayArtifactPath(d.TitleSlug, id)) {
			continue
		}
		out = append(out, candidatARattraper{matchID: id, rawName: rawName.String, mapID: mapID.String})
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: rejeu 2D — retard (rows)", "err", err)
	}
	return out
}

// requeteQueueRecente : les matchs les plus récents du registre que le marqueur terminal
// n'écarte pas, dans la fenêtre de rétention (0 = illimitée).
//
// ORDRE DESCENDANT, comme `killcollector.requeteBacklog` et pour la même raison : un film déjà
// expiré ne se sauve pas, et les matchs récents sont à la fois les seuls récupérables et ceux
// que l'utilisateur regarde. `start_time` brut trierait faux (règle 8) — d'où le COALESCE
// canonique.
func requeteQueueRecente(months int) string {
	fenetre := ""
	if months > 0 {
		fenetre = " AND " + analysis.SQLStartTimeCanonical("r") + " >= ?"
	}
	return `SELECT r.match_id, r.map_name, r.map_id
		FROM match_registry r
		WHERE COALESCE(r.backfill_completed, 0) & ? = 0` + fenetre + `
		ORDER BY ` + analysis.SQLStartTimeCanonical("r") + ` DESC, r.match_id
		LIMIT ?`
}

// artefactPresent : le prédicat le moins cher qui existe. Il répond à « ce match a-t-il UN
// rejeu », pas à « son rejeu est-il au bon schéma » — cf. l'en-tête du fichier.
func artefactPresent(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

// budgetDuCycle rend le budget de durée effectif.
//
// 0 = le contrat de production, [BudgetParCycle]. Une valeur NÉGATIVE veut dire « déjà
// épuisé » : c'est ainsi que les tests prouvent que la garde existe et qu'elle s'applique AVANT
// le premier match, sans attendre cinq minutes ni décoder quoi que ce soit.
func budgetDuCycle(d Deps) time.Duration {
	if d.Budget != 0 {
		return d.Budget
	}
	return BudgetParCycle
}

// selectBuildWork lit les identités de carte des matchs insérés et applique la fenêtre de
// rétention (months <= 0 = illimité). metaDB peut être nil (pas de résolution EN : map_name
// brut seul, même dégradation que le backfill CLI).
func selectBuildWork(
	ctx context.Context, sharedDB, metaDB *sql.DB, insertedIDs []string, months int,
) []buildWork {
	if len(insertedIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(insertedIDs)), ",")
	args := make([]any, 0, len(insertedIDs))
	for _, id := range insertedIDs {
		args = append(args, id)
	}
	q := fmt.Sprintf(`SELECT match_id, map_name, map_id, %s AS start_canonical
		FROM match_registry WHERE match_id IN (%s)`,
		analysis.SQLStartTimeCanonical("match_registry"), placeholders)
	rows, err := sharedDB.QueryContext(ctx, q, args...)
	if err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu échouée", "err", err)
		return nil
	}
	defer func() { _ = rows.Close() }()

	var cutoff time.Time
	if months > 0 {
		cutoff = time.Now().UTC().AddDate(0, -months, 0)
	}
	var out []buildWork
	for rows.Next() {
		var id string
		var rawName, mapID sql.NullString
		var start sql.NullTime
		if err := rows.Scan(&id, &rawName, &mapID, &start); err != nil {
			slog.WarnContext(ctx, "post-sync: sélection rejeu (scan)", "err", err)
			return out
		}
		if months > 0 && start.Valid && start.Time.Before(cutoff) {
			continue // hors fenêtre de rétention : le backfill CLI reste libre de le faire
		}
		out = append(out, buildWork{matchID: id, mapNames: nomsDeCarte(ctx, metaDB,
			candidatARattraper{matchID: id, rawName: rawName.String, mapID: mapID.String})})
	}
	if err := rows.Err(); err != nil {
		slog.WarnContext(ctx, "post-sync: sélection rejeu (rows)", "err", err)
	}
	return out
}

// resolveMapNameEN résout le nom EN d'une carte par son asset UGC (asset_translations).
// Best-effort : metaDB nil ou nom absent → "" (le candidat brut reste).
func resolveMapNameEN(ctx context.Context, metaDB *sql.DB, mapID string) string {
	if metaDB == nil || mapID == "" {
		return ""
	}
	var en string
	err := metaDB.QueryRowContext(ctx,
		`SELECT name FROM asset_translations WHERE asset_type = 'map' AND asset_id = ? AND lang = 'en-US'`,
		mapID).Scan(&en)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(en)
}
