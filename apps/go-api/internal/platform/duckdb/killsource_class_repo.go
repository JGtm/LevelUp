// Package duckdb — killsource_class_repo.go : implementation DuckDB du loader agrege
// des kills par SOURCE DE DEGAT (port.KillSourceClassRepository).
//
// Source : `match_kill_events_latest` — la vue `_latest` et JAMAIS la table (regle ART
// n2 : une lecture brute servirait les lignes de plusieurs passes de decodage
// superposees). Une ligne par mort ; on compte celles que le kill-feed CREDITE aux
// joueurs demandes.
//
// LE POINT DUR, ET SA GARANTIE : NE PAS DOUBLE-COMPTER. Ce repo lit une SECONDE voie de
// mesure, en parallele de `weapon_kills` (attribution arme-a-feu). Si les deux voyaient
// le meme kill, il compterait deux fois et l'invariant du sunburst (somme des classes ==
// total de kills) sauterait. Le filtre qui l'interdit est STRUCTUREL, pas une liste de
// classes ecrite a la main : on ne garde que les cles de registre qui n'ont AUCUN
// identifiant numerique dans `weapon_ids`. Une source sans id numerique est, par
// construction, invisible a l'attribution arme-a-feu — qui ne sait resoudre qu'un
// `weapon_id`. Le garde-rail qui empeche cette propriete de deriver vit dans le paquet
// du registre (TestHorsArsenalHINFSansIdNumerique) : il exige que l'ensemble des cles
// HINF sans id numerique soit EXACTEMENT les six entrees hors arsenal.
//
// La traduction source -> cle de registre est injectee (port.KillSourceClassifier) : la
// table qui la porte est propre au titre, ce paquet reste title-agnostic. Elle se fait en
// Go, apres la requete : le tag reste un ENTIER cote SQL, aucune comparaison de chaine.
//
// Capability gating : `match_kill_events_latest` absente (titre sans decodeur de film,
// migration non appliquee) -> games.ErrCapabilityNotSupported, via isTableNotFoundErr.
// Classificateur nil (titre qui n'en fournit pas) -> zero ligne, sans erreur : c'est
// l'etat NOMINAL d'un titre sans film, pas une panne.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// KillSourceClassRepo implemente port.KillSourceClassRepository.
type KillSourceClassRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de degat en cle de registre. nil = ce titre n'en
	// fournit pas -> le repo rend zero ligne (etat nominal, cf. en-tete).
	classifier port.KillSourceClassifier
}

// NewKillSourceClassRepo cree un KillSourceClassRepo lie a un PlayerDB.
//
// classifier peut etre nil : voir l'en-tete du fichier.
func NewKillSourceClassRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *KillSourceClassRepo {
	return &KillSourceClassRepo{pdb: pdb, classifier: classifier}
}

// killSourceTally : compteur intermediaire par (xuid, weapon_key).
type killSourceTally struct {
	kills          int
	nonPublishable int
}

// killSourceKey : cle du compteur intermediaire.
type killSourceKey struct {
	xuid      string
	weaponKey string
}

// LoadKillSourceClassesAggregated charge les kills agreges par (xuid, weapon_key).
//
// L'appelant DOIT avoir valide les filtres ; le repo re-valide en defense.
func (r *KillSourceClassRepo) LoadKillSourceClassesAggregated(
	ctx context.Context,
	slug string,
	filters port.KillSourceClassFilters,
) ([]port.KillSourceClassRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("KillSourceClassRepo.LoadKillSourceClassesAggregated: %w", err)
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "KillSourceClassRepo: no classifier for title, nothing to load",
			"slug", slug, "match_count", len(filters.MatchIDs))
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tally, err := r.queryTally(ctx, filters)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "KillSourceClassRepo: match_kill_events_latest missing",
				"slug", slug, "match_count", len(filters.MatchIDs))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillSourceClassRepo: query failed",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("KillSourceClassRepo.LoadKillSourceClassesAggregated: %w", err)
	}
	if len(tally) == 0 {
		return nil, nil
	}
	return r.resolveRows(ctx, slug, tally), nil
}

// queryTally lit les morts creditees et les agrege par (xuid, cle de registre).
//
// La traduction du tag se fait ICI, en Go : le SQL ne connait que des entiers.
// `publishable = FALSE` est COMPTE (les lignes sont justes en agregat, cf. l'en-tete de
// port/kill_source_class.go) mais tracé a part, pour que la surface puisse dire d'ou
// vient sa mesure.
func (r *KillSourceClassRepo) queryTally(
	ctx context.Context,
	filters port.KillSourceClassFilters,
) (map[killSourceKey]*killSourceTally, error) {
	q, args := buildKillSourceClassQuery(filters, r.pdb.TitleSlug)

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	dbRows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer dbRows.Close()

	out := map[killSourceKey]*killSourceTally{}
	for dbRows.Next() {
		var (
			xuid        string
			sourceTag   uint32
			publishable bool
			kills       int
		)
		if err := dbRows.Scan(&xuid, &sourceTag, &publishable, &kills); err != nil {
			// Une ligne illisible est une anomalie de schema, pas un cas nominal : on la
			// signale avant de degrader, on ne l'avale pas en silence.
			slog.ErrorContext(ctx, "KillSourceClassRepo: scan failed, row skipped", "err", err)
			continue
		}
		key, ok := r.classifier.KillSourceRegistryKey(sourceTag)
		if !ok {
			continue // source hors perimetre : reste dans « Non attribue » (decision D6)
		}
		k := killSourceKey{xuid: xuid, weaponKey: key}
		if out[k] == nil {
			out[k] = &killSourceTally{}
		}
		out[k].kills += kills
		if !publishable {
			out[k].nonPublishable += kills
		}
	}
	return out, dbRows.Err()
}

// buildKillSourceClassQuery : morts creditees aux joueurs demandes, sur les matchs
// demandes, dont la source est mesuree. GROUP BY (tueur, source, publiable).
//
// `feed_killer_xuid IS NOT NULL` exclut les tueurs BOTS (un bot n'a pas de xuid) — le
// filtre IN le ferait de toute facon, la clause le dit.
func buildKillSourceClassQuery(f port.KillSourceClassFilters, titleSlug string) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs))
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	sb.WriteString(`
SELECT
    k.feed_killer_xuid,
    k.source_tag,
    k.publishable,
    COUNT(*)::INTEGER AS kills
FROM match_kill_events_latest k
WHERE k.match_id IN (`)
	sb.WriteString(Placeholders(len(f.MatchIDs)))
	sb.WriteString(`)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "k.match_id"))
	sb.WriteString(`
  AND k.source_tag IS NOT NULL
  AND k.feed_killer_xuid IS NOT NULL
  AND k.feed_killer_xuid IN (`)
	sb.WriteString(Placeholders(len(f.XUIDs)))
	for _, x := range f.XUIDs {
		args = append(args, x)
	}
	sb.WriteString(`)
GROUP BY k.feed_killer_xuid, k.source_tag, k.publishable`)
	return sb.String(), args
}

// resolveRows habille les compteurs de leur classe et de leur libelle, et applique LE
// filtre anti-double-comptage (cle sans id numerique — cf. l'en-tete du fichier).
func (r *KillSourceClassRepo) resolveRows(
	ctx context.Context,
	slug string,
	tally map[killSourceKey]*killSourceTally,
) []port.KillSourceClassRow {
	keys := make([]string, 0, len(tally))
	seen := map[string]bool{}
	for k := range tally {
		if !seen[k.weaponKey] {
			seen[k.weaponKey] = true
			keys = append(keys, k.weaponKey)
		}
	}
	meta := resolveOffArsenalKeys(ctx, r.pdb.Metadata, slug, keys)

	out := make([]port.KillSourceClassRow, 0, len(tally))
	for k, t := range tally {
		m, ok := meta[k.weaponKey]
		if !ok {
			// Cle inconnue du registre, ou cle qui PORTE un id numerique : dans les deux
			// cas on ne la remonte pas. Le second cas est celui qui protege du
			// double-comptage — il ne devrait jamais arriver (garde-rail dans le paquet
			// weapons), d'ou le log.
			slog.WarnContext(ctx, "KillSourceClassRepo: registry key dropped",
				"slug", slug, "weapon_key", k.weaponKey, "kills", t.kills)
			continue
		}
		out = append(out, port.KillSourceClassRow{
			XUID:                k.xuid,
			WeaponKey:           k.weaponKey,
			Class:               m.class,
			Label:               m.label,
			LabelEN:             m.labelEN,
			Kills:               t.kills,
			NonPublishableKills: t.nonPublishable,
		})
	}
	sortKillSourceRows(out)
	return out
}

// sortKillSourceRows rend la sortie DETERMINISTE : la map d'agregation ne l'est pas, et
// une sortie qui change d'ordre a chaque appel casse les goldens et fait clignoter le
// sunburst. Tri par kills decroissants, puis xuid, puis cle — total, jamais ambigu.
func sortKillSourceRows(rows []port.KillSourceClassRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kills != rows[j].Kills {
			return rows[i].Kills > rows[j].Kills
		}
		if rows[i].XUID != rows[j].XUID {
			return rows[i].XUID < rows[j].XUID
		}
		return rows[i].WeaponKey < rows[j].WeaponKey
	})
}
