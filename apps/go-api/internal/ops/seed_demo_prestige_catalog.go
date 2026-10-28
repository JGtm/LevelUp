// Package ops — seed_demo_prestige_catalog.go : catalogue Prestige du titre, lu pour la démo.
//
// Un objectif ou un arc n'est servi dans la langue de la requête que s'il a une ORIGINE connue
// (Challenge.TemplateID, Arc.PresetID) et que son texte stocké est la copie de départ du modèle
// ou du preset (prestige.Template.ServedLabel, PresetArc.ServedTitle ; ADR 0040 D-3). Le plan
// démo s'adosse donc au catalogue réel du titre, lu ici depuis les TOML par le chargeur du
// serveur (prestige.LoadTemplatesFromTOML / LoadPresetArcsFromTOML) : le serveur de démo seede
// son metadata depuis ces mêmes fichiers au boot, les identifiants et les textes concordent.
package ops

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/prestige"
)

// demoPrestigeCatalog est le catalogue de modèles de défis et d'arcs preset d'un titre.
// Les identifiants du catalogue sont préfixés par le slug du titre ("<slug>.weekly.x") : les
// spécifications démo ne portent que le suffixe, ce qui garde le plan indépendant du titre.
// Un catalogue vide (titre sans catalogue) fait retomber le plan sur ses textes de repli.
type demoPrestigeCatalog struct {
	Slug      string
	Templates map[string]prestige.Template
	Presets   map[string]prestige.PresetArc
}

// template rend le modèle d'identifiant "<slug>.<suffix>".
func (c demoPrestigeCatalog) template(suffix string) (prestige.Template, bool) {
	t, ok := c.Templates[c.Slug+"."+suffix]
	return t, ok
}

// preset rend le preset d'arc d'identifiant "<slug>.<suffix>".
func (c demoPrestigeCatalog) preset(suffix string) (prestige.PresetArc, bool) {
	p, ok := c.Presets[c.Slug+"."+suffix]
	return p, ok
}

// empty : aucun modèle ni preset (titre sans catalogue, ou catalogue illisible).
func (c demoPrestigeCatalog) empty() bool {
	return len(c.Templates) == 0 && len(c.Presets) == 0
}

// captureTemplateRepo et capturePresetRepo réutilisent les chargeurs TOML du serveur sans base :
// ils ne retiennent que ce que le chargeur leur remet via Replace. Les autres méthodes de
// l'interface ne sont jamais appelées par un chargeur (interface embarquée nil).
type captureTemplateRepo struct {
	prestige.TemplateRepo
	got []prestige.Template
}

func (r *captureTemplateRepo) Replace(_ context.Context, _ string, ts []prestige.Template) error {
	r.got = ts
	return nil
}

type capturePresetRepo struct {
	prestige.PresetArcRepo
	got []prestige.PresetArc
}

func (r *capturePresetRepo) Replace(_ context.Context, _ string, arcs []prestige.PresetArc,
	_ []prestige.PresetArcStep) error {
	r.got = arcs
	return nil
}

// loadDemoPrestigeCatalog lit le catalogue du titre sous <repoRoot>/config/titles/<slug>/
// (repoRoot vide = racine résolue par title.FindRepoRoot). Un titre sans fichier de catalogue
// (Halo 5 : pas de défis) rend un catalogue vide SANS erreur ; une racine introuvable ou un
// TOML invalide rend une erreur, que l'appelant journalise avant de dégrader.
func loadDemoPrestigeCatalog(ctx context.Context, repoRoot, titleSlug string) (demoPrestigeCatalog, error) {
	cat := demoPrestigeCatalog{Slug: titleSlug}
	if repoRoot == "" {
		root, err := titlePkg.FindRepoRoot()
		if err != nil {
			return cat, fmt.Errorf("racine du dépôt introuvable: %w", err)
		}
		repoRoot = root
	}
	titleDir := filepath.Dir(titlePkg.NewPathResolver(repoRoot).TitleMappingsDir(titleSlug))
	// Un dossier de titre absent trahit une racine mal pointée (pas un titre sans défis) : erreur,
	// pas un catalogue vide silencieux.
	if _, err := os.Stat(titleDir); err != nil {
		return cat, fmt.Errorf("dossier de configuration du titre: %w", err)
	}

	tr := &captureTemplateRepo{}
	if _, err := prestige.LoadTemplatesFromTOML(ctx, tr,
		filepath.Join(titleDir, "challenges", "templates.toml")); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			slog.InfoContext(ctx, "seed-demo prestige: titre sans catalogue de défis",
				"title_slug", titleSlug)
			return cat, nil
		}
		return cat, fmt.Errorf("catalogue de modèles: %w", err)
	}
	cat.Templates = make(map[string]prestige.Template, len(tr.got))
	for _, t := range tr.got {
		cat.Templates[t.ID] = t
	}

	pr := &capturePresetRepo{}
	if _, err := prestige.LoadPresetArcsFromTOML(ctx, pr,
		filepath.Join(titleDir, "arcs", "presets.toml")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cat, fmt.Errorf("catalogue de presets: %w", err)
	}
	cat.Presets = make(map[string]prestige.PresetArc, len(pr.got))
	for _, p := range pr.got {
		cat.Presets[p.ID] = p
	}
	return cat, nil
}

// demoCatalogGaps liste les origines (modèles et presets) que les spécifications démo
// attendent et que le catalogue ne porte pas : chaque manque fait retomber l'objectif ou l'arc
// concerné sur son texte de repli. Vide pour un catalogue complet.
func demoCatalogGaps(cat demoPrestigeCatalog) []string {
	var gaps []string
	for _, s := range demoObjectiveSpecs() {
		if _, ok := cat.template(s.TemplateID); !ok {
			gaps = append(gaps, cat.Slug+"."+s.TemplateID)
		}
	}
	for _, a := range demoArcSpecs() {
		if _, ok := cat.preset(a.PresetID); !ok {
			gaps = append(gaps, cat.Slug+"."+a.PresetID)
		}
	}
	return gaps
}
