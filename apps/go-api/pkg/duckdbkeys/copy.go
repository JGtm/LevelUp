package duckdbkeys

import (
	"context"
	"fmt"
	"strings"
)

// CopyStats rend compte d'une recopie vers une table à clés.
type CopyStats struct {
	Source            int64 // lignes lues dans la source
	Copied            int64 // lignes écrites dans la destination
	DuplicatesDropped int64 // lignes écartées : doublon d'une ligne gardée sur une clé
	DivergentKeys     int64 // valeurs de clé dont les doublons ne sont pas identiques
}

// DedupSelect rend le SELECT des colonnes cols de la relation src qui ne garde qu'une ligne par
// valeur de chaque clé de keys (clés appliquées l'une après l'autre, dans l'ordre donné).
//
// Règle de départage, déterministe et documentée : la ligne dont l'horodatage `recency` est le
// plus récent est gardée (NULL en dernier) ; à horodatage égal ou sans horodatage (`recency`
// vide), la plus petite selon l'ordre de toutes les colonnes, prises dans l'ordre de cols — le
// résultat ne dépend donc que du contenu, jamais de l'ordre physique des lignes.
//
// Aucune ligne n'est écartée pour une clé primaire NULL : elle reste dans le SELECT, et la
// clé primaire de la destination la refuse (l'appelant écarte ces tables en amont). Pour une
// contrainte UNIQUE, une ligne dont une colonne de la clé est NULL n'entre en
// conflit avec aucune autre : elle est gardée. Les colonnes de chaque clé doivent
// figurer dans cols.
func DedupSelect(src string, cols []string, keys []Key, recency string) string {
	colList := identList(cols)
	order := make([]string, 0, len(cols)+1)
	if recency != "" {
		order = append(order, Ident(recency)+" DESC NULLS LAST")
	}
	for _, c := range cols {
		order = append(order, Ident(c)+" ASC NULLS LAST")
	}
	orderBy := strings.Join(order, ", ")

	cur := fmt.Sprintf("SELECT %s FROM %s", colList, src)
	for _, k := range keys {
		rn := fmt.Sprintf("ROW_NUMBER() OVER (PARTITION BY %s ORDER BY %s)", identList(k.Columns), orderBy)
		if !k.Primary {
			rn = fmt.Sprintf("CASE WHEN %s THEN %s ELSE 1 END", notNullAll(k.Columns), rn)
		}
		cur = fmt.Sprintf("SELECT %s FROM (SELECT %s, %s AS __dk_rn FROM (%s) AS __dk) AS __dk_f WHERE __dk_rn = 1",
			colList, colList, rn, cur)
	}
	return cur
}

// CopyInto recopie les colonnes cols de la relation src (un nom de table cité ou une fonction
// table, ex. `read_parquet('…')`) dans la table dst, une ligne par valeur de chaque clé de keys
// (règle de DedupSelect), et rend le compte de ce qui a été gardé et écarté. Une ligne à clé
// primaire NULL fait échouer la recopie : rien n'est écarté en silence. La destination
// doit exister ; q est typiquement une transaction de l'appelant.
func CopyInto(ctx context.Context, q Querier, dst, src string, cols []string, keys []Key, recency string) (CopyStats, error) {
	var st CopyStats
	if err := q.QueryRowContext(ctx, "SELECT count(*) FROM "+src).Scan(&st.Source); err != nil {
		return st, fmt.Errorf("comptage de la source: %w", err)
	}
	res, err := q.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (%s) %s",
		Ident(dst), identList(cols), DedupSelect(src, cols, keys, recency)))
	if err != nil {
		return st, fmt.Errorf("recopie vers %s: %w", dst, err)
	}
	if st.Copied, err = res.RowsAffected(); err != nil {
		return st, fmt.Errorf("recopie vers %s: lignes écrites: %w", dst, err)
	}
	st.DuplicatesDropped = st.Source - st.Copied
	if st.DuplicatesDropped <= 0 {
		return st, nil
	}
	for _, k := range keys {
		var n int64
		div := fmt.Sprintf(`SELECT count(*) FROM (SELECT 1 FROM %s WHERE %s GROUP BY %s
			HAVING count(DISTINCT row(%s)) > 1)`, src, notNullAll(k.Columns), identList(k.Columns), identList(cols))
		if err := q.QueryRowContext(ctx, div).Scan(&n); err != nil {
			return st, fmt.Errorf("doublons divergents de %s: %w", dst, err)
		}
		st.DivergentKeys += n
	}
	return st, nil
}

// identList cite et joint des noms de colonnes.
func identList(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = Ident(c)
	}
	return strings.Join(quoted, ", ")
}

// notNullAll rend le prédicat « aucune des colonnes n'est NULL ».
func notNullAll(cols []string) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = Ident(c) + " IS NOT NULL"
	}
	return strings.Join(parts, " AND ")
}
