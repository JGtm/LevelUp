// Package duckdb — citation_texts.go : lecture des citations (metadata.citation_mappings pour la
// structure, asset_translations pour les textes par langue). Les noms et descriptions ne sont plus
// des colonnes appariées : une langue de plus est un jeu de lignes `citation` de asset_translations
// (ADR 0040 D-15), lu ici sans changer ce code.
package duckdb

import (
	"context"
	"fmt"
	"strings"

	"levelup/go-api/internal/i18n"
)

// citationMeta regroupe ce que les pages lisent d'une citation : le nom et la description DANS LA
// LANGUE DE LA REQUÊTE, l'image et les paliers.
type citationMeta struct {
	display     string
	imagePath   string
	tierTargets string
	description string
}

// pickCitationTexts choisit le nom et la description d'une citation dans la langue `loc`. Le nom
// replie en croisé (pickAcrossChain : la langue demandée, la langue de repli, la langue par
// défaut). La description suit Text.PickOptional : la langue demandée, puis la langue de repli ; une
// description absente reste absente ("" : la carte affiche le nom seul), jamais le texte d'une autre
// langue (principe GH-5b « EN n'injecte jamais de FR »).
func pickCitationTexts(loc i18n.Locale, t DefinitionTexts) (name, desc string) {
	name = pickAcrossChain(loc, t.Name)
	if loc == i18n.Fallback {
		// Pick sous la langue de repli ne replie sur rien : lecture directe, sans compter
		// comme manquante une description facultative absente.
		return name, t.Description[loc]
	}
	return name, t.Description.PickOptional(loc)
}

// loadCitationMeta lit pour les citations `norms` (actives) l'image, les paliers et les textes par
// langue, choisis dans la langue du contexte. Une citation absente de citation_mappings est absente
// du résultat. L'erreur est rendue : l'appelant la journalise avant de dégrader.
func loadCitationMeta(ctx context.Context, meta *DB, norms []string) (map[string]citationMeta, error) {
	out := make(map[string]citationMeta, len(norms))
	if len(norms) == 0 || meta == nil {
		return out, nil
	}
	args := make([]any, len(norms))
	for i, n := range norms {
		args[i] = n
	}
	rows, err := meta.Query(ctx, fmt.Sprintf(Q26jCitationMappingsForNormsTemplate, Placeholders(len(norms))), args...)
	if err != nil {
		return out, fmt.Errorf("citation_mappings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var norm string
		var m citationMeta
		if err := rows.Scan(&norm, &m.imagePath, &m.tierTargets); err != nil {
			return out, fmt.Errorf("citation_mappings scan: %w", err)
		}
		out[norm] = m
	}
	if err := rows.Err(); err != nil {
		return out, fmt.Errorf("citation_mappings: %w", err)
	}
	texts, err := LoadAssetTexts(ctx, meta, AssetTypeCitation, norms)
	if err != nil {
		return out, err
	}
	loc := i18n.FromContext(ctx)
	for norm, m := range out {
		m.display, m.description = pickCitationTexts(loc, texts[norm])
		out[norm] = m
	}
	return out, nil
}

// loadCitationMetaOrLog est loadCitationMeta pour les appelants qui dégradent (citation sans
// métadonnées) : l'échec est journalisé et consigné AVANT la dégradation, ce qui a été lu est rendu.
func loadCitationMetaOrLog(ctx context.Context, meta *DB, norms []string) map[string]citationMeta {
	out, err := loadCitationMeta(ctx, meta, norms)
	if err != nil {
		bestEffortFailed(ctx, "citation_meta", err)
	}
	return out
}

// citationDisplayNames retourne, pour les citations `norms`, leur nom dans la langue du contexte
// (repli croisé de pickAcrossChain). Une citation sans nom est absente du résultat.
func citationDisplayNames(ctx context.Context, meta *DB, norms []string) (map[string]string, error) {
	texts, err := LoadAssetNames(ctx, meta, AssetTypeCitation, norms)
	if err != nil {
		return nil, err
	}
	loc := i18n.FromContext(ctx)
	out := make(map[string]string, len(texts))
	for norm, t := range texts {
		if name := strings.TrimSpace(pickAcrossChain(loc, t.Name)); name != "" {
			out[norm] = name
		}
	}
	return out, nil
}
