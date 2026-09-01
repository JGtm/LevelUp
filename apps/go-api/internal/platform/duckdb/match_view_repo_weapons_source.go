// Package duckdb — match_view_repo_weapons_source.go : les armes d'un match lues dans la
// SOURCE DE DEGAT du film, pour les titres qui savent la traduire.
//
// # Pourquoi ce fichier double le lecteur SQL
//
// Les armes d'un match etaient lues par `Q28BulkWeaponKills` sur `v_weapon_kills` : un
// `GROUP BY effective_weapon_id`, puis une resolution des libelles en Go. Cette voie reste
// LA voie des titres dont l'arme est native de l'API (Halo 5). Elle ne peut pas servir un
// titre qui lit la source de degat : la traduction d'un tag `jpt!` en entree de registre
// vit dans le BINAIRE, pas dans la base — la porter en SQL obligerait a copier la table de
// correspondance dans DuckDB et a la maintenir synchronisee a chaque saison (decision D12
// du plan du 2026-09-01). La requete reste donc entiere en entiers, et la traduction se
// fait ici, en Go.
//
// # Le choix se fait sur une capability, jamais sur un slug
//
// `MatchViewRepo` recoit un `port.KillSourceClassifier` au cablage — nil pour un titre qui
// n'en fournit pas. Nil : on lit `v_weapon_kills`. Non nil : on lit
// `match_kill_events_latest`. Aucun `slug ==` nulle part.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"levelup/go-api/internal/domain"
)

// qMatchKillSourceWeapons : les morts d'un match dont la source est mesuree, agregees par
// (tueur credite, source). Le tag reste un ENTIER — aucune chaine a comparer cote SQL.
//
// `feed_killer_xuid IS NOT NULL` exclut les tueurs BOTS (un bot n'a pas de xuid).
const qMatchKillSourceWeapons = `
SELECT
    k.feed_killer_xuid,
    k.source_tag,
    COUNT(*)::INTEGER AS kills
FROM match_kill_events_latest k
WHERE k.match_id = ?
  AND k.source_tag IS NOT NULL
  AND k.feed_killer_xuid IS NOT NULL
GROUP BY k.feed_killer_xuid, k.source_tag`

// matchSourceKey : cle d'agregation intermediaire (joueur, cle de registre).
type matchSourceKey struct {
	xuid      string
	weaponKey string
}

// bulkWeaponKillsFromSource rend les armes de TOUS les joueurs du match, lues dans la
// source de degat.
//
// Meme contrat que le lecteur SQL qu'il remplace : une ligne par (joueur, arme), triee par
// joueur puis kills decroissants, dimensions du registre resolues. `MechanicKills` vaut 0 —
// il n'a de sens que pour `weapon_kills`, ou une melee est attribuee a l'arme TENUE ; la
// source de degat, elle, nomme l'effet qui a tue.
func (r *MatchViewRepo) bulkWeaponKillsFromSource(
	ctx context.Context, matchID string,
) ([]domain.BulkWeaponKillRaw, error) {
	tally, ordered, err := r.tallyMatchKillSources(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if len(ordered) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(ordered))
	vu := map[string]bool{}
	for _, k := range ordered {
		if !vu[k.weaponKey] {
			vu[k.weaponKey] = true
			keys = append(keys, k.weaponKey)
		}
	}
	meta := resolveWeaponKeyDimensions(ctx, r.pdb.Metadata, pdbTitleSlug(r.pdb), keys)

	out := make([]domain.BulkWeaponKillRaw, 0, len(ordered))
	for _, k := range ordered {
		m, ok := meta[k.weaponKey]
		if !ok {
			continue // cle inconnue du registre : reste dans « Non attribue » (D7)
		}
		out = append(out, domain.BulkWeaponKillRaw{
			XUID:        k.xuid,
			WeaponID:    m.numericID,
			Kills:       tally[k],
			WeaponLabel: m.label,
			Class:       m.class,
			Role:        m.role,
			Family:      m.family,
			WeaponKey:   k.weaponKey,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].XUID != out[j].XUID {
			return out[i].XUID < out[j].XUID
		}
		return out[i].Kills > out[j].Kills
	})
	return out, nil
}

// tallyMatchKillSources agrege les morts du match par (joueur, cle de registre) et rend
// AUSSI l'ordre de premiere apparition, pour que la sortie ne depende pas d'une map.
func (r *MatchViewRepo) tallyMatchKillSources(
	ctx context.Context, matchID string,
) (map[matchSourceKey]int, []matchSourceKey, error) {
	sharedDB, release, err := r.sharedRead().Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	rows, err := sharedDB.QueryContext(ctx, qMatchKillSourceWeapons, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("match kill sources: %w", err)
	}
	defer rows.Close()

	tally := map[matchSourceKey]int{}
	ordered := make([]matchSourceKey, 0, 32)
	ecartes := 0
	for rows.Next() {
		var (
			xuid      string
			sourceTag uint32
			kills     int
		)
		if err := rows.Scan(&xuid, &sourceTag, &kills); err != nil {
			// Une ligne illisible est une anomalie de schema, pas un cas nominal.
			slog.ErrorContext(ctx, "match view: source de degat illisible, ligne sautee",
				"match_id", matchID, "err", err)
			continue
		}
		key, ok := r.killSourceClassifier.KillSourceRegistryKey(sourceTag)
		if !ok {
			ecartes += kills
			continue
		}
		k := matchSourceKey{xuid: xuid, weaponKey: key}
		if _, seen := tally[k]; !seen {
			ordered = append(ordered, k)
		}
		tally[k] += kills
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if ecartes > 0 {
		// Un kill que le rejeu 2D sait nommer et que le tableau d'armes ne montre pas ne
		// doit pas disparaitre en silence (decision D13 du plan).
		slog.InfoContext(ctx, "match view: morts sans cle de registre, absentes du detail des armes",
			"match_id", matchID, "morts_ecartees", ecartes)
	}
	return tally, ordered, nil
}

// topWeaponByXUID rend, pour chaque joueur, l'arme qui l'a le plus servi sur le match.
//
// Elle remplace la sous-requete `top_weapons` que portait Q12 : celle-ci lisait
// `v_weapon_kills`, table qui disparait du fichier des titres a decodeur de film. La
// calculer ici sert les DEUX titres depuis la meme source que le detail des armes — un
// tableau et un resume qui se contredisent seraient pires que pas de resume du tout.
func topWeaponByXUID(rows []domain.BulkWeaponKillRaw) map[string]int64 {
	best := map[string]int{}
	out := map[string]int64{}
	for _, w := range rows {
		if w.WeaponID == 0 || w.Kills <= 0 {
			continue // objet hors arsenal : aucun identifiant a poser dans top_weapon_id
		}
		if w.Kills > best[w.XUID] {
			best[w.XUID] = w.Kills
			out[w.XUID] = w.WeaponID
		}
	}
	return out
}

// attachTopWeapons renseigne `TopWeaponID` de chaque ligne de scoreboard depuis LE meme
// lecteur d'armes que le detail du match.
//
// POURQUOI EN GO, ET PLUS DANS LA REQUETE. Q12 portait une sous-requete `top_weapons` sur
// `v_weapon_kills` — table qui disparait du fichier des titres a decodeur de film. La
// laisser aurait fait echouer TOUTE la requete (Catalog Error) et vide le scoreboard, pas
// seulement l'arme favorite : c'est exactement l'incident de production du 25/07 que
// l'en-tete de Q12 documente. La calculer ici la sert aux DEUX titres depuis la source qui
// alimente deja le detail des armes — un resume et un tableau qui se contredisent seraient
// pires que pas de resume.
//
// Best-effort : une lecture en echec laisse `TopWeaponID` nil, et la colonne disparait de
// la page. Jamais d'erreur remontee — le scoreboard, lui, est deja la.
func (r *MatchViewRepo) attachTopWeapons(ctx context.Context, matchID string, rows []domain.ScoreboardRaw) {
	if len(rows) == 0 {
		return
	}
	bulk, err := r.GetMatchBulkWeaponKills(ctx, matchID)
	if err != nil {
		slog.WarnContext(ctx, "match view: arme favorite du scoreboard non calculee",
			"match_id", matchID, "err", err)
		return
	}
	top := topWeaponByXUID(bulk)
	for i := range rows {
		if id, ok := top[rows[i].XUID]; ok {
			v := id
			rows[i].TopWeaponID = &v
		}
	}
}
