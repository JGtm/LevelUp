package prestige

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/i18n"
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
	ID          string `toml:"id"`
	Metric      string `toml:"metric"`
	WindowType  string `toml:"window_type"`
	WindowValue string `toml:"window_value"`
	Cadence     string `toml:"cadence"`
	EvalType    string `toml:"eval_type"`
	ModeFilter  string `toml:"mode_filter"`
	// Label et Description : tables par langue (`label = { en = "...", fr = "..." }`).
	Label           i18n.Text `toml:"label"`
	Description     i18n.Text `toml:"description"`
	NormalTarget    float64   `toml:"normal_target"`
	HeroicTarget    float64   `toml:"heroic_target"`
	LegendaryTarget float64   `toml:"legendary_target"`
	MythicTarget    float64   `toml:"mythic_target"`
	// Tagging V1 PlayerProfile (cf. plan §5.1).
	LUSRComponents []string `toml:"lusr_components"`
	RadarAxes      []string `toml:"radar_axes"`
	IsLongTerm     bool     `toml:"is_long_term"`
}

// presetsTOML est la projection brute de presets.toml.
type presetsTOML struct {
	Meta catalogMeta          `toml:"meta"`
	Arcs []presetArcEntryTOML `toml:"arcs"`
}

type presetArcEntryTOML struct {
	ID          string                   `toml:"id"`
	Title       i18n.Text                `toml:"title"`
	Description i18n.Text                `toml:"description"`
	Steps       []presetArcStepEntryTOML `toml:"steps"`
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
			Labels:          t.Label,
			Descriptions:    t.Description,
			NormalTarget:    t.NormalTarget,
			HeroicTarget:    t.HeroicTarget,
			LegendaryTarget: t.LegendaryTarget,
			MythicTarget:    t.MythicTarget,
			LUSRComponents:  t.LUSRComponents,
			RadarAxes:       t.RadarAxes,
			IsLongTerm:      t.IsLongTerm,
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
		if a.ID == "" {
			return 0, fmt.Errorf("preset arc: id requis")
		}
		if err := a.Title.Trimmed().Validate(); err != nil {
			return 0, fmt.Errorf("preset arc %q: title: %w", a.ID, err)
		}
		arcs = append(arcs, PresetArc{
			ID:            a.ID,
			TitleSlug:     doc.Meta.TitleSlug,
			Titles:        a.Title,
			Descriptions:  a.Description,
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
	if t.ID == "" || t.Metric == "" {
		return fmt.Errorf("id/metric requis")
	}
	if err := t.Label.Trimmed().Validate(); err != nil {
		return fmt.Errorf("label: %w", err)
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
