package invariants

// invariants_lusr_chain.go — I14 : aucune chaîne LUSR étrangère au titre de la base.
//
// Détecteur de la corruption cross-titre du 2026-06-26 (spécification G3 du rapport
// .ai/V7.5/RAPPORT_VOLET1_LUSR_H5_2026-08-28.md §5.3). Le pipeline V2 tirait alors le
// titre de deux sources : handles DB du titre du CYCLE, moteur du titre du PROFIL. Les
// 4 joueurs déclarés sous deux titres ont reçu, dans leurs player DB halo_infinite, des
// lignes match_skill_rank de chaîne h5_arena — la chaîne LUSR de Halo 5.
//
// Les gardes STRUCTURELLES (stamp du titre du moteur, fail-loud au câblage V2) ferment
// le trou en amont ; cet invariant est le filet de données, en second.
//
// Fonction séparée plutôt qu'un checkFn de plus : la signature checkFn partagée par les
// 13 invariants existants ne porte pas l'ensemble autorisé. Le package reste sans
// dépendance interne (il ne peut donc pas énumérer les chaînes d'un titre) : l'appelant
// injecte la liste du titre de la base.
//
// LECTURE DE LA TABLE BRUTE — volontaire, et c'est le point : match_skill_rank est
// append-only, la vue _latest ne montre que la ligne gagnante par match. Une chaîne
// fausse écrite une fois SURVIT à tout replay correct sous la ligne gagnante, invisible
// de _latest. Un détecteur qui ne lirait que la vue aurait déclaré les 4 bases saines
// alors qu'elles portaient 2 479 lignes étrangères.

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// lusrChainRatingTypes — les rating_type dont playlist_group porte une chaîne LUSR.
// 'CSR' est exclu : son playlist_group relève du référentiel des playlists classées,
// pas de la partition TrueSkill.
var lusrChainRatingTypes = []string{"LUSR", "LUSR_V2"}

// CheckPlayerLUSRChains vérifie qu'aucune ligne LUSR de la player DB ne porte une
// chaîne absente de allowedChains — la liste des chaînes du TITRE de cette base, que
// l'appelant lit dans le package de titre (internal/games/{slug}/skillchain pour Halo
// Infinite). Violation `lusr_chain_foreign_title`, SeverityFail.
//
// allowedChains vide = harnais mal câblé : erreur, jamais un verdict « tout est
// étranger » ni un succès silencieux (un invariant invérifiable est un échec du
// harnais, cf. godoc du package).
func CheckPlayerLUSRChains(ctx context.Context, playerDB *sql.DB, allowedChains []string) (Report, error) {
	rep := Report{}
	if playerDB == nil {
		return rep, fmt.Errorf("invariants/lusr_chain_foreign_title: playerDB nil")
	}
	if len(allowedChains) == 0 {
		return rep, fmt.Errorf("invariants/lusr_chain_foreign_title: allowedChains vide — " +
			"l'appelant doit injecter les chaînes LUSR du titre de cette base")
	}

	args := make([]any, 0, len(lusrChainRatingTypes)+len(allowedChains))
	for _, rt := range lusrChainRatingTypes {
		args = append(args, rt)
	}
	for _, c := range allowedChains {
		args = append(args, c)
	}
	query := `
		SELECT playlist_group, COUNT(*) AS n
		FROM match_skill_rank
		WHERE rating_type IN (` + placeholders(len(lusrChainRatingTypes)) + `)
		  AND playlist_group IS NOT NULL
		  AND playlist_group <> ''
		  AND playlist_group NOT IN (` + placeholders(len(allowedChains)) + `)
		GROUP BY playlist_group
		ORDER BY n DESC, playlist_group ASC`

	rows, err := playerDB.QueryContext(ctx, query, args...)
	if err != nil {
		return rep, fmt.Errorf("invariants/lusr_chain_foreign_title: %w", err)
	}
	defer rows.Close()

	var total int
	var groups []string
	for rows.Next() {
		var group string
		var n int
		if err := rows.Scan(&group, &n); err != nil {
			return rep, fmt.Errorf("invariants/lusr_chain_foreign_title scan: %w", err)
		}
		groups = append(groups, group)
		total += n
	}
	if err := rows.Err(); err != nil {
		return rep, fmt.Errorf("invariants/lusr_chain_foreign_title rows: %w", err)
	}
	if total == 0 {
		return rep, nil
	}
	rep.Violations = append(rep.Violations, Violation{
		Key:      "lusr_chain_foreign_title",
		Severity: SeverityFail,
		Count:    total,
		Sample:   capSample(groups),
		Description: "lignes match_skill_rank portant une chaîne LUSR qui n'appartient pas au " +
			"titre de cette base (corruption cross-titre, incident 2026-06-26)",
	})
	return rep, nil
}

// placeholders retourne "?, ?, ?" pour n paramètres (n ≥ 1).
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
