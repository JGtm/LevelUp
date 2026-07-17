// Package duckdb — relations_csr_repo.go : lecture best-effort du CSR courant d'une
// relation, pour le contexte « bête noire » du hub Communauté > Relations (lot
// relations-G). Lecture seule sur le catalogue shared via SharedReader.
package duckdb

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// relationsCSRTimeout : la lecture CSR est un enrichissement best-effort de
// l'aperçu Relations — un délai court évite qu'un shared lent ne pénalise la page.
const relationsCSRTimeout = 5 * time.Second

// Q31LatestCSRByXUID : snapshot CSR le PLUS RÉCENT d'un joueur, depuis la vue
// append-only match_csrs_latest (règle ART n°2) jointe à match_registry pour la
// chronologie. Ordonné par le fragment timezone canonique (règle CLAUDE.md n°8 :
// jamais start_time brut), le match le plus récent en premier. Paramètre : ?1 = xuid.
// Exécutée sur SharedReader (match_csrs et match_registry sont dans la shared DB —
// pas de préfixe `shared.`).
var Q31LatestCSRByXUID = `
SELECT c.tier, c.sub_tier, c.rating_value
FROM match_csrs_latest c
JOIN match_registry r ON r.match_id = c.match_id
WHERE c.xuid = ?
ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC NULLS LAST
LIMIT 1`

// GetLatestCSR retourne le snapshot CSR le plus récent du joueur `xuid` (par
// start_time canonique du match), lu depuis match_csrs_latest. Best-effort strict :
// retourne (nil, nil) — jamais d'erreur — si xuid vide, table/colonne absente,
// aucune ligne CSR, ou snapshot non significatif (palier absent ou sentinelle
// « Placement »). Alimente le contexte CSR de la bête noire (dégradation gracieuse :
// nil ⇒ rien affiché côté front). Voir port.RelationsRepository.GetLatestCSR.
func (r *CareerRepo) GetLatestCSR(ctx context.Context, xuid string) (*domain.RelationCSR, error) {
	if strings.TrimSpace(xuid) == "" {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, relationsCSRTimeout)
	defer cancel()

	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, nil //nolint:nilerr — enrichissement best-effort : shared indisponible ⇒ pas de CSR
	}
	defer release()

	var (
		tier    sql.NullString
		subTier sql.NullInt16
		rating  sql.NullFloat64
	)
	row := db.QueryRowContext(ctx, Q31LatestCSRByXUID, xuid)
	if err := row.Scan(&tier, &subTier, &rating); err != nil {
		// sql.ErrNoRows (aucune ligne CSR : relation non classée) OU table/colonne
		// absente (avant migration) ⇒ best-effort, pas de CSR, pas d'erreur.
		return nil, nil //nolint:nilerr
	}

	// Palier significatif : présent et différent de la sentinelle « Placement »
	// (mêmes critères que la sonde de couverture G0). Un palier absent/placement
	// n'est pas affichable comme rang courant → dégradation gracieuse.
	tierName := strings.TrimSpace(tier.String)
	if !tier.Valid || tierName == "" || strings.EqualFold(tierName, "Placement") {
		return nil, nil
	}

	csr := &domain.RelationCSR{Tier: &tierName}
	if subTier.Valid {
		v := int(subTier.Int16)
		csr.SubTier = &v
	}
	if rating.Valid {
		v := rating.Float64
		csr.RatingValue = &v
	}
	return csr, nil
}
