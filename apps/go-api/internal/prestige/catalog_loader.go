package prestige

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// catalog_loader.go — chargeur des templates et preset arcs depuis TOML.
//
// Source : config/titles/{slug}/challenges/templates.toml
//          config/titles/{slug}/arcs/presets.toml
//
// Appelé au boot pour synchroniser le catalogue en DB. Le pattern de
// chargement est tout-ou-rien : si un fichier TOML est invalide, le
// catalogue n'est pas remplacé et les anciens templates restent en place
// (avec un slog.Warn).

// templatesTOML est la projection brute de templates.toml.
type templatesTOML struct {
	Meta      catalogMeta         `toml:"meta"`
	Templates []templateEntryTOML `toml:"templates"`
}

type catalogMeta struct {
	SchemaVersion int    `toml:"schema_version"`
	TitleSlug     string `toml:"title_slug"`
}

type templateEntryTOML struct {
	ID              string  `toml:"id"`
	Metric          string  `toml:"metric"`
	WindowType      string  `toml:"window_type"`
	WindowValue     string  `toml:"window_value"`
	Cadence         string  `toml:"cadence"`
	EvalType        string  `toml:"eval_type"`
	ModeFilter      string  `toml:"mode_filter"`
	LabelEN         string  `toml:"label_en"`
	LabelFR         string  `toml:"label_fr"`
	DescriptionEN   string  `toml:"description_en"`
	DescriptionFR   string  `toml:"description_fr"`
	NormalTarget    float64 `toml:"normal_target"`
	HeroicTarget    float64 `toml:"heroic_target"`
	LegendaryTarget float64 `toml:"legendary_target"`
	MythicTarget    float64 `toml:"mythic_target"`
}

// presetsTOML est la projection brute de presets.toml.
type presetsTOML struct {
	Meta catalogMeta          `toml:"meta"`
	Arcs []presetArcEntryTOML `toml:"arcs"`
}

type presetArcEntryTOML struct {
	ID            string                   `toml:"id"`
	TitleEN       string                   `toml:"title_en"`
	TitleFR       string                   `toml:"title_fr"`
	DescriptionEN string                   `toml:"description_en"`
	DescriptionFR string                   `toml:"description_fr"`
	Steps         []presetArcStepEntryTOML `toml:"steps"`
}

type presetArcStepEntryTOML struct {
	Position   int    `toml:"position"`
	TemplateID string `toml:"template_id"`
	TargetTier string `toml:"target_tier"`
}

// LoadTemplatesFromTOML charge un fichier templates.toml et remplace
// le catalogue du titre dans la DB.
//
// En cas d'erreur de lecture/parse, retourne l'erreur sans toucher la DB.
// Validation : tous les templates doivent avoir des paliers monotones.
func LoadTemplatesFromTOML(ctx context.Context, repo TemplateRepo, path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	var doc templatesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return 0, fmt.Errorf("%s: meta.title_slug manquant", path)
	}
	templates := make([]Template, 0, len(doc.Templates))
	now := time.Now().UTC()
	for _, t := range doc.Templates {
		if err := validateTemplateEntry(t); err != nil {
			return 0, fmt.Errorf("%s template %s: %w", path, t.ID, err)
		}
		templates = append(templates, Template{
			ID:              t.ID,
			TitleSlug:       doc.Meta.TitleSlug,
			Metric:          t.Metric,
			WindowType:      WindowType(t.WindowType),
			WindowValue:     t.WindowValue,
			Cadence:         Cadence(t.Cadence),
			EvalType:        EvalType(t.EvalType),
			ModeFilter:      defaultStr(t.ModeFilter, "universal"),
			LabelEN:         t.LabelEN,
			LabelFR:         t.LabelFR,
			DescriptionEN:   t.DescriptionEN,
			DescriptionFR:   t.DescriptionFR,
			NormalTarget:    t.NormalTarget,
			HeroicTarget:    t.HeroicTarget,
			LegendaryTarget: t.LegendaryTarget,
			MythicTarget:    t.MythicTarget,
			SchemaVersion:   doc.Meta.SchemaVersion,
			UpdatedAt:       now,
		})
	}

	if err := repo.Replace(ctx, doc.Meta.TitleSlug, templates); err != nil {
		return 0, fmt.Errorf("replace templates: %w", err)
	}
	slog.InfoContext(ctx, "prestige: templates loaded",
		"title_slug", doc.Meta.TitleSlug, "count", len(templates), "path", path)
	return len(templates), nil
}

// LoadPresetArcsFromTOML charge un fichier presets.toml et remplace le
// catalogue d'arcs du titre dans la DB.
func LoadPresetArcsFromTOML(ctx context.Context, repo PresetArcRepo, path string) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	var doc presetsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return 0, fmt.Errorf("%s: meta.title_slug manquant", path)
	}

	arcs := make([]PresetArc, 0, len(doc.Arcs))
	steps := make([]PresetArcStep, 0)
	now := time.Now().UTC()
	for _, a := range doc.Arcs {
		if a.ID == "" || a.TitleEN == "" || a.TitleFR == "" {
			return 0, fmt.Errorf("preset arc %q: id/title_en/title_fr requis", a.ID)
		}
		arcs = append(arcs, PresetArc{
			ID:            a.ID,
			TitleSlug:     doc.Meta.TitleSlug,
			TitleEN:       a.TitleEN,
			TitleFR:       a.TitleFR,
			DescriptionEN: a.DescriptionEN,
			DescriptionFR: a.DescriptionFR,
			SchemaVersion: doc.Meta.SchemaVersion,
			UpdatedAt:     now,
		})
		for _, st := range a.Steps {
			if st.Position <= 0 || st.TemplateID == "" {
				return 0, fmt.Errorf("preset arc %q step %d: position/template_id requis", a.ID, st.Position)
			}
			tier := Tier(st.TargetTier)
			if !tier.Valid() {
				return 0, fmt.Errorf("preset arc %q step %d: target_tier invalide (%q)",
					a.ID, st.Position, st.TargetTier)
			}
			steps = append(steps, PresetArcStep{
				PresetArcID: a.ID,
				Position:    st.Position,
				TemplateID:  st.TemplateID,
				TargetTier:  tier,
			})
		}
	}

	if err := repo.Replace(ctx, doc.Meta.TitleSlug, arcs, steps); err != nil {
		return 0, fmt.Errorf("replace preset arcs: %w", err)
	}
	slog.InfoContext(ctx, "prestige: preset arcs loaded",
		"title_slug", doc.Meta.TitleSlug, "arcs", len(arcs), "steps", len(steps), "path", path)
	return len(arcs), nil
}

// validateTemplateEntry vérifie la cohérence d'un template TOML.
func validateTemplateEntry(t templateEntryTOML) error {
	if t.ID == "" || t.Metric == "" || t.LabelEN == "" || t.LabelFR == "" {
		return fmt.Errorf("id/metric/label_en/label_fr requis")
	}
	if !Cadence(t.Cadence).Valid() {
		return fmt.Errorf("cadence invalide: %q", t.Cadence)
	}
	if !EvalType(t.EvalType).Valid() {
		return fmt.Errorf("eval_type invalide: %q", t.EvalType)
	}
	if !WindowType(t.WindowType).Valid() {
		return fmt.Errorf("window_type invalide: %q", t.WindowType)
	}
	if t.NormalTarget >= t.HeroicTarget ||
		t.HeroicTarget >= t.LegendaryTarget ||
		t.LegendaryTarget >= t.MythicTarget {
		return fmt.Errorf("paliers non monotones (N=%v H=%v L=%v M=%v)",
			t.NormalTarget, t.HeroicTarget, t.LegendaryTarget, t.MythicTarget)
	}
	return nil
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
