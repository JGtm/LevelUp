// Package duckdb — killsource_weapon_scope.go : LE foyer unique des lectures « armes d'un
// joueur » adossees a la source de degat du film.
//
// # Pourquoi un foyer, et pas une requete par surface
//
// Trois surfaces posent la meme question a des perimetres differents : le top armes de la
// cible de l'Explorer (borne a un lot de matchs), l'arme favorite de l'Accueil (tout
// l'historique du joueur) et la statistique d'arme du moteur de citations. Recopier trois
// fois « agreger par source, traduire le tag, resoudre le registre, trier » aurait garanti
// que les trois copies divergent (regle CLAUDE.md n6 : a la 3e copie, on centralise).
//
// La traduction du tag se fait en Go, jamais en SQL (decision D12 du plan du 2026-09-01) :
// la table de correspondance vit dans le binaire et change a chaque saison ; la copier dans
// DuckDB obligerait a la resynchroniser a chaque fois.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// weaponScopeRow : une arme et son volume de frags sur le perimetre demande.
type weaponScopeRow struct {
	weaponID  int64 // 0 = objet hors arsenal (aucun identifiant numerique au registre)
	weaponKey string
	label     string
	kills     int
}

// weaponKillsFromSourceForPlayer agrege les morts CREDITEES a un joueur par arme, lues
// dans la source de degat du film.
//
// matchIDs vide = tout l'historique du joueur (l'arme favorite de l'Accueil). Le
// garde-fou anti-scan-complet reste tenu par le filtre sur le xuid, qui est obligatoire.
//
// Best-effort de resolution : une cle que le registre ne connait pas n'est PAS remontee
// (elle reste dans « Non attribue » — decision D7), les autres le sont.
func weaponKillsFromSourceForPlayer(
	ctx context.Context,
	pdb *PlayerDB,
	classifier port.KillSourceClassifier,
	xuid string,
	matchIDs []string,
) ([]weaponScopeRow, error) {
	if pdb == nil || classifier == nil || strings.TrimSpace(xuid) == "" {
		return nil, nil
	}
	tally, err := tallyPlayerKillSources(ctx, pdb, classifier, xuid, matchIDs)
	if err != nil {
		return nil, err
	}
	if len(tally) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	meta := resolveWeaponKeyDimensions(ctx, pdb.Metadata, pdbTitleSlug(pdb), keys)

	out := make([]weaponScopeRow, 0, len(tally))
	for k, kills := range tally {
		m, ok := meta[k]
		if !ok {
			continue
		}
		out = append(out, weaponScopeRow{
			weaponID: m.numericID, weaponKey: k, label: m.label, kills: kills,
		})
	}
	// Tri TOTAL : kills decroissants puis cle — une sortie qui change d'ordre a chaque
	// appel ferait clignoter « arme favorite ».
	sort.Slice(out, func(i, j int) bool {
		if out[i].kills != out[j].kills {
			return out[i].kills > out[j].kills
		}
		return out[i].weaponKey < out[j].weaponKey
	})
	return out, nil
}

// tallyPlayerKillSources compte les morts creditees au joueur, par cle de registre.
func tallyPlayerKillSources(
	ctx context.Context,
	pdb *PlayerDB,
	classifier port.KillSourceClassifier,
	xuid string,
	matchIDs []string,
) (map[string]int, error) {
	q, args := buildPlayerKillSourceQuery(pdbTitleSlug(pdb), xuid, matchIDs)

	db, release, err := pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var (
			sourceTag uint32
			kills     int
		)
		if err := rows.Scan(&sourceTag, &kills); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if key, ok := classifier.KillSourceRegistryKey(sourceTag); ok {
			out[key] += kills
		}
	}
	return out, rows.Err()
}

// buildPlayerKillSourceQuery : morts creditees au joueur, agregees par source. Le tag reste
// un ENTIER cote SQL — aucune chaine a comparer.
func buildPlayerKillSourceQuery(titleSlug, xuid string, matchIDs []string) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(matchIDs)+1)
	sb.WriteString(`
SELECT k.source_tag, COUNT(*)::INTEGER AS kills
FROM match_kill_events_latest k
WHERE k.feed_killer_xuid = ?
  AND k.source_tag IS NOT NULL`)
	args = append(args, xuid)
	if len(matchIDs) > 0 {
		sb.WriteString(`
  AND k.match_id IN (`)
		sb.WriteString(Placeholders(len(matchIDs)))
		sb.WriteString(`)`)
		for _, id := range matchIDs {
			args = append(args, id)
		}
	}
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "k.match_id"))
	sb.WriteString(`
GROUP BY k.source_tag`)
	return sb.String(), args
}

// topWeaponsFromSource rend le top `limit` armes du joueur sur un lot de matchs, lues dans
// la source de degat. Meme contrat que la voie SQL qu'elle remplace : libelle resolu, tri
// par frags decroissants, best-effort (nil sur erreur — le top armes est une information
// secondaire, jamais fatale a la page).
//
// Les objets hors arsenal (repulseur, bobines, environnement) sont ECARTES : cette surface
// affiche une vignette d'arme, et un objet sans identifiant numerique n'en a pas. Ils
// restent comptes partout ou le sunburst les sert.
func (r *ExplorerRepo) topWeaponsFromSource(
	ctx context.Context, xuid string, matchIDs []string, limit int,
) []domain.WeaponHighlight {
	rows, err := weaponKillsFromSourceForPlayer(ctx, r.pdb, r.killSourceClassifier, xuid, matchIDs)
	if err != nil {
		slog.DebugContext(ctx, "ExplorerRepo.topWeaponsFromSource: lecture (best-effort)", "err", err)
		return nil
	}
	out := make([]domain.WeaponHighlight, 0, limit)
	for _, w := range rows {
		if w.weaponID == 0 {
			continue
		}
		label := w.label
		if label == "" {
			label = strconv.FormatUint(uint64(w.weaponID), 10) //nolint:gosec
		}
		out = append(out, domain.WeaponHighlight{
			WeaponID: w.weaponID, Kills: w.kills, LabelFR: label, LabelEN: label,
		})
		if len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// favoriteWeaponFromSource rend l'arme la plus meurtriere du joueur sur TOUT son
// historique — l'encart « arme favorite » de l'Accueil. Second retour faux : aucune arme
// resolue (film jamais decode sur ce joueur), l'encart disparait.
func favoriteWeaponFromSource(
	ctx context.Context, pdb *PlayerDB, classifier port.KillSourceClassifier, xuid string,
) (weaponScopeRow, bool) {
	rows, err := weaponKillsFromSourceForPlayer(ctx, pdb, classifier, xuid, nil)
	if err != nil {
		slog.WarnContext(ctx, "arme favorite: lecture de la source de degat echouee", "err", err)
		return weaponScopeRow{}, false
	}
	for _, w := range rows {
		if w.weaponID != 0 {
			return w, true
		}
	}
	return weaponScopeRow{}, false
}

// favoriteWeaponFromDamageSource sert l'encart « arme favorite » de l'Accueil depuis la
// source de degat. Meme contrat externe que la voie SQL : ("", 0, nil) quand rien n'est
// resolu — le front affiche « — » comme avant, et la panne est visible au journal.
func (r *HomeRepo) favoriteWeaponFromDamageSource(ctx context.Context, locale string) (string, int, error) {
	w, ok := favoriteWeaponFromSource(ctx, r.pdb, r.killSourceClassifier, r.pdb.XUID)
	if !ok {
		return "", 0, nil
	}
	nom := w.label
	if locale == "en" {
		// Le libelle EN vient de la MEME passe registre que le FR (weapon_name_labels) ;
		// on le redemande ici plutot que de l'entasser dans weaponScopeRow, qui sert trois
		// surfaces dont deux n'en ont pas l'usage.
		if m, found := resolveWeaponKeyDimensions(ctx, r.pdb.Metadata, pdbTitleSlug(r.pdb),
			[]string{w.weaponKey})[w.weaponKey]; found && m.labelEN != "" {
			nom = m.labelEN
		}
	}
	if nom == "" {
		nom = "Inconnue"
		if locale == "en" {
			nom = "Unknown"
		}
	}
	return nom, w.kills, nil
}
