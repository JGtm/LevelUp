// Package duckdb — killsource_weapon_kills_repo.go : L'UNIQUE lecteur de l'arme d'un kill
// pour les titres qui decodent leur film (`port.WeaponKillsRepository`).
//
// # Ce qu'il remplace, et pourquoi
//
// L'ancienne chaine reconstruisait l'arme en correlant les TIRS de l'attaquant avec
// l'instant du kill (table `weapon_kills`). Une epee, un marteau, un faisceau n'emettent
// pas de tir : leurs kills etaient perdus, ou recolles sur l'arme a feu tenue — ce qui
// gonflait l'AR, le BR et le Sidekick. Ce lecteur-ci lit la SOURCE DU DEGAT dans l'etat de
// mort de la VICTIME (`match_kill_events_latest.source_tag`), ou l'epee est une epee.
//
// Mesure du 2026-09-01 sur 200 matchs (.ai/V7.5/MESURE_BASCULE_ARME_2026-09-01.md) : le
// residu « Non attribue » passe de 14 453 (77,1 %) a 3 984 (21,2 %), et aucune classe
// d'arme a feu ne perd un seul frag.
//
// # Une seule voie, donc plus rien a departager
//
// Il rend AUSSI les classes hors arsenal (repulseur, bobines, environnement) que servait
// jusqu'ici un second chemin de chargement. Le filtre « cle sans identifiant numerique »
// qui empechait les deux voies de compter deux fois le meme kill n'a plus d'objet : il n'y
// a plus qu'une voie (decision D11 du plan).
//
// # Les frontieres, et qui les tient
//
//   - Source : la vue `_latest` et JAMAIS la table (regle ART n2 — une lecture brute
//     servirait les lignes de plusieurs passes de decodage superposees).
//   - Credit : `feed_killer_xuid`, le tueur tel que le kill-feed le designe, avec l'arme de
//     la SOURCE (decision D5).
//   - Totaux API : les classes melee / grenade / capacites spartanes gardent leur TOTAL des
//     compteurs natifs ; ce lecteur remonte leurs lignes pour la seule VENTILATION de
//     niveau 2, et c'est `fragdist` qui tient cette frontiere (decision D4).
//   - Une source qui ne resout aucune cle de registre reste dans « Non attribue » : on ne
//     devine pas, on ne proratise pas (decision D7). Le volume ecarte est JOURNALISE par
//     classe de source — un kill que le rejeu sait nommer ne disparait pas en silence (D13).
//
// La traduction tag -> cle se fait en Go, apres la requete : le SQL ne connait que des
// entiers, jamais une chaine a comparer (decision D12).
//
// Capability gating : `match_kill_events_latest` absente (titre sans decodeur de film,
// migration non appliquee) -> games.ErrCapabilityNotSupported, via isTableNotFoundErr.
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

// KillSourceWeaponKillsRepo implemente port.WeaponKillsRepository depuis la source de degat.
type KillSourceWeaponKillsRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de degat en cle de registre. Jamais nil : le cablage
	// ne construit ce repo que pour un titre qui en fournit un.
	classifier port.KillSourceClassifier
	// describer NOMME la classe d'une source, pour le journal seul. Peut etre nil.
	describer port.KillSourceDescriber
}

// NewKillSourceWeaponKillsRepo cree le lecteur d'arme adosse a la source de degat.
//
// describer est facultatif : il est decouvert par assertion sur le classificateur.
func NewKillSourceWeaponKillsRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *KillSourceWeaponKillsRepo {
	r := &KillSourceWeaponKillsRepo{pdb: pdb, classifier: classifier}
	if d, ok := classifier.(port.KillSourceDescriber); ok {
		r.describer = d
	}
	return r
}

// sourceTally : compteur intermediaire par (xuid, cle de registre).
type sourceTally struct {
	xuid      string
	weaponKey string
}

// LoadWeaponKillsAggregated charge les frags agreges par (xuid, arme), depuis la source
// de degat.
//
// L'appelant DOIT avoir valide les filtres ; le repo re-valide en defense.
func (r *KillSourceWeaponKillsRepo) LoadWeaponKillsAggregated(
	ctx context.Context,
	slug string,
	filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("KillSourceWeaponKillsRepo.LoadWeaponKillsAggregated: %w", err)
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "KillSourceWeaponKillsRepo: aucun classificateur pour ce titre",
			"slug", slug, "match_count", len(filters.MatchIDs))
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tally, err := r.queryTally(ctx, slug, filters)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "KillSourceWeaponKillsRepo: match_kill_events_latest absente",
				"slug", slug, "match_count", len(filters.MatchIDs))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillSourceWeaponKillsRepo: lecture echouee",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("KillSourceWeaponKillsRepo.LoadWeaponKillsAggregated: %w", err)
	}

	rows := r.resolveRows(ctx, slug, tally, filters.MinKills)
	if filters.IncludeGrenadeMelee {
		gm, err := r.queryGrenadeMelee(ctx, filters)
		if err != nil {
			// Les compteurs natifs sont une SECONDE source, independante du film : leur
			// absence n'invalide pas ce qui precede. On signale avant de degrader.
			slog.ErrorContext(ctx, "KillSourceWeaponKillsRepo: compteurs grenade/melee illisibles",
				"slug", slug, "err", err)
		} else {
			rows = append(rows, gm...)
		}
	}
	return rows, nil
}

// queryTally lit les morts creditees et les agrege par (xuid, cle de registre). La
// traduction du tag se fait ICI, en Go (D12).
func (r *KillSourceWeaponKillsRepo) queryTally(
	ctx context.Context,
	slug string,
	filters port.WeaponKillFilters,
) (map[sourceTally]int, error) {
	q, args := buildKillSourceWeaponQuery(filters, r.pdb.TitleSlug)

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

	out := map[sourceTally]int{}
	ecartes := map[string]int{}
	for dbRows.Next() {
		var (
			xuid      string
			sourceTag uint32
			kills     int
		)
		if err := dbRows.Scan(&xuid, &sourceTag, &kills); err != nil {
			// Une ligne illisible est une anomalie de schema, pas un cas nominal : on la
			// signale avant de degrader, on ne l'avale pas en silence.
			slog.ErrorContext(ctx, "KillSourceWeaponKillsRepo: scan echoue, ligne sautee", "err", err)
			continue
		}
		key, ok := r.classifier.KillSourceRegistryKey(sourceTag)
		if !ok {
			ecartes[r.classeSource(sourceTag)] += kills
			continue
		}
		out[sourceTally{xuid: xuid, weaponKey: key}] += kills
	}
	if err := dbRows.Err(); err != nil {
		return nil, err
	}
	r.journaliserEcartes(ctx, slug, ecartes, len(filters.MatchIDs))
	return out, nil
}

// classeSource nomme la classe d'une source, pour le journal seul. « inconnue » si le
// titre ne sait pas la nommer (interface optionnelle, cf. port.KillSourceDescriber).
func (r *KillSourceWeaponKillsRepo) classeSource(sourceTag uint32) string {
	if r.describer == nil {
		return "inconnue"
	}
	if name, ok := r.describer.KillSourceClassName(sourceTag); ok && name != "" {
		return name
	}
	return "inconnue"
}

// journaliserEcartes publie le volume de morts qui restent dans « Non attribue » faute de
// cle de registre, ventile par classe de source (decision D13 : un kill que le rejeu sait
// nommer ne doit pas disparaitre en silence).
func (r *KillSourceWeaponKillsRepo) journaliserEcartes(
	ctx context.Context, slug string, ecartes map[string]int, matchCount int,
) {
	if len(ecartes) == 0 {
		return
	}
	total := 0
	classes := make([]string, 0, len(ecartes))
	for c, n := range ecartes {
		total += n
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool { return ecartes[classes[i]] > ecartes[classes[j]] })
	detail := make([]string, 0, len(classes))
	for _, c := range classes {
		detail = append(detail, fmt.Sprintf("%s=%d", c, ecartes[c]))
	}
	slog.InfoContext(ctx, "arme du kill: morts sans cle de registre, laissees en « Non attribue »",
		"title", slug, "match_count", matchCount, "morts_ecartees", total,
		"par_classe", strings.Join(detail, " "))
}

// buildKillSourceWeaponQuery : morts creditees aux joueurs demandes, sur les matchs
// demandes, dont la source est mesuree. GROUP BY (tueur, source).
//
// `feed_killer_xuid IS NOT NULL` exclut les tueurs BOTS (un bot n'a pas de xuid) — le
// filtre sur les joueurs le ferait de toute facon, la clause le dit.
func buildKillSourceWeaponQuery(f port.WeaponKillFilters, titleSlug string) (string, []any) {
	var sb strings.Builder
	args := make([]any, 0, len(f.MatchIDs)+len(f.XUIDs)+1)
	for _, id := range f.MatchIDs {
		args = append(args, id)
	}
	sb.WriteString(`
SELECT
    k.feed_killer_xuid,
    k.source_tag,
    COUNT(*)::INTEGER AS kills
FROM match_kill_events_latest k
WHERE k.match_id IN (`)
	sb.WriteString(Placeholders(len(f.MatchIDs)))
	sb.WriteString(`)`)
	sb.WriteString(excludeCampaignByMatchID(titleSlug, "k.match_id"))
	sb.WriteString(`
  AND k.source_tag IS NOT NULL
  AND k.feed_killer_xuid IS NOT NULL`)
	appendKillerXUIDFilter(&sb, &args, f)
	sb.WriteString(`
GROUP BY k.feed_killer_xuid, k.source_tag`)
	return sb.String(), args
}

// appendKillerXUIDFilter restreint aux joueurs demandes, par xuid ou par gamertag.
// Au moins un des deux est garanti par Validate().
func appendKillerXUIDFilter(sb *strings.Builder, args *[]any, f port.WeaponKillFilters) {
	if len(f.XUIDs) > 0 {
		sb.WriteString(`
  AND k.feed_killer_xuid IN (`)
		sb.WriteString(Placeholders(len(f.XUIDs)))
		sb.WriteString(`)`)
		for _, x := range f.XUIDs {
			*args = append(*args, x)
		}
		return
	}
	sb.WriteString(`
  AND k.feed_killer_xuid IN (
      SELECT xuid FROM xuid_aliases WHERE gamertag = ?
  )`)
	*args = append(*args, f.Gamertag)
}

// resolveRows habille les compteurs de leurs dimensions de registre. Une cle inconnue du
// registre n'est PAS remontee : elle reste dans « Non attribue » (D7).
func (r *KillSourceWeaponKillsRepo) resolveRows(
	ctx context.Context,
	slug string,
	tally map[sourceTally]int,
	minKills int,
) []port.WeaponKillRow {
	if len(tally) == 0 {
		return nil
	}
	keys := make([]string, 0, len(tally))
	vu := map[string]bool{}
	for k := range tally {
		if !vu[k.weaponKey] {
			vu[k.weaponKey] = true
			keys = append(keys, k.weaponKey)
		}
	}
	meta := resolveWeaponKeyDimensions(ctx, r.pdb.Metadata, slug, keys)

	out := make([]port.WeaponKillRow, 0, len(tally))
	horsRegistre := 0
	for k, kills := range tally {
		if minKills > 0 && kills < minKills {
			continue
		}
		m, ok := meta[k.weaponKey]
		if !ok {
			horsRegistre += kills
			continue
		}
		out = append(out, port.WeaponKillRow{
			XUID:      k.xuid,
			WeaponID:  m.numericID,
			Kills:     kills,
			Label:     m.label,
			LabelEN:   m.labelEN,
			Role:      m.role,
			Class:     m.class,
			Family:    m.family,
			WeaponKey: k.weaponKey,
			// La ligne a ete MESUREE dans le film : c'est ce qui autorise les classes hors
			// arsenal (equipement, environnement) a etre servies. Cf. le champ dans
			// port/weapon_kills.go — sans lui, le bucket `h5_environmental` de Halo 5
			// remonterait par le meme chemin.
			FromDamageSource: true,
		})
	}
	if horsRegistre > 0 {
		slog.WarnContext(ctx, "arme du kill: cle resolue mais absente du registre",
			"title", slug, "morts", horsRegistre)
	}
	sortWeaponKillRows(out)
	return out
}

// sortWeaponKillRows rend la sortie DETERMINISTE : la map d'agregation ne l'est pas, et une
// sortie qui change d'ordre a chaque appel casse les goldens et fait clignoter le sunburst.
func sortWeaponKillRows(rows []port.WeaponKillRow) {
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

// queryGrenadeMelee rend les deux lignes sentinelles grenade / melee, lues des compteurs
// NATIFS de `match_participants` — exactement la meme source qu'avant la bascule. Le film
// ne les sert pas : leur total est autoritatif (D4).
func (r *KillSourceWeaponKillsRepo) queryGrenadeMelee(
	ctx context.Context, filters port.WeaponKillFilters,
) ([]port.WeaponKillRow, error) {
	q, args := buildParticipantMechanicQuery(filters, r.pdb.TitleSlug)
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
	var out []port.WeaponKillRow
	for dbRows.Next() {
		var (
			xuid          string
			weaponID      UBigint
			kills         int
			mechanicKills int
			isGM          bool
		)
		if err := dbRows.Scan(&xuid, &weaponID, &kills, &mechanicKills, &isGM); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if filters.MinKills > 0 && kills < filters.MinKills {
			continue
		}
		out = append(out, port.WeaponKillRow{
			XUID: xuid, WeaponID: weaponID.Int64(), Kills: kills, IsGrenadeMelee: isGM,
		})
	}
	return out, dbRows.Err()
}
