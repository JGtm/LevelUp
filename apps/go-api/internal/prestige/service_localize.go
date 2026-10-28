package prestige

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"levelup/go-api/internal/i18n"
)

// service_localize.go — résolution À LA LECTURE des textes stockés en une seule langue
// (Challenge.Label, Arc.Title, Arc.Description), ADR 0040 D-3.
//
// Un défi ou un arc copie le texte de son modèle / preset à la création dans la langue par défaut
// (Template.StoredLabel, PresetArc.StoredTitle). L'origine est enregistrée (Challenge.TemplateID,
// Arc.PresetID) : tant que le texte stocké est EXACTEMENT une copie, le service sert celui du
// modèle dans la langue de la requête (Template.ServedLabel et consorts) ; un texte modifié ou
// libre est servi tel quel. Ni le stockage ni le schéma ne changent. La copie peut être dans N'IMPORTE
// QUELLE langue du modèle (isStoredCopy : un défi créé depuis le web porte le texte de la langue de
// la requête de création) : la résolution se fait donc aussi sous la langue par défaut. Un défi ou un
// arc SANS origine enregistrée (TemplateID / PresetID vide, ou absent du catalogue) est servi tel quel.

// localizeChallenges remplace en place le libellé des défis issus d'un modèle par celui de la
// langue de la requête et rend la liste. Un modèle illisible laisse le libellé stocké (journalisé).
func (s *service) localizeChallenges(ctx context.Context, list []Challenge) []Challenge {
	loc := i18n.FromContext(ctx)
	if s.deps.Templates == nil {
		return list
	}
	cache := map[string]*Template{}
	for i := range list {
		id := list[i].TemplateID
		if id == "" {
			continue
		}
		tmpl, seen := cache[id]
		if !seen {
			tmpl = s.loadTemplateForLabel(ctx, id)
			cache[id] = tmpl
		}
		if tmpl != nil {
			list[i].Label = tmpl.ServedLabel(list[i].Label, loc)
		}
	}
	return list
}

// localizeChallenge est localizeChallenges pour un seul défi.
func (s *service) localizeChallenge(ctx context.Context, c Challenge) Challenge {
	return s.localizeChallenges(ctx, []Challenge{c})[0]
}

// localizeArcs remplace en place titre et description des arcs adoptés depuis un preset
// (IsPreset et PresetID) par ceux de la langue de la requête. Un arc libre n'est pas touché.
func (s *service) localizeArcs(ctx context.Context, arcs []Arc) []Arc {
	loc := i18n.FromContext(ctx)
	if s.deps.PresetArcs == nil {
		return arcs
	}
	cache := map[string]*PresetArc{}
	for i := range arcs {
		id := arcs[i].PresetID
		if !arcs[i].IsPreset || id == "" {
			continue
		}
		preset, seen := cache[id]
		if !seen {
			preset = s.loadPresetForTitle(ctx, id)
			cache[id] = preset
		}
		if preset != nil {
			arcs[i].Title = preset.ServedTitle(arcs[i].Title, loc)
			arcs[i].Description = preset.ServedDescription(arcs[i].Description, loc)
		}
	}
	return arcs
}

// localizeArc est localizeArcs pour un seul arc.
func (s *service) localizeArc(ctx context.Context, a Arc) Arc {
	return s.localizeArcs(ctx, []Arc{a})[0]
}

// loadTemplateForLabel lit le modèle d'un défi ; nil quand il est introuvable ou illisible (le
// libellé stocké est alors servi, la raison est journalisée).
func (s *service) loadTemplateForLabel(ctx context.Context, id string) *Template {
	tmpl, err := s.deps.Templates.GetByID(ctx, id)
	if err != nil {
		logUnresolvedOrigin(ctx, "template", id, err)
		return nil
	}
	return &tmpl
}

// loadPresetForTitle lit le preset d'un arc ; nil quand il est introuvable ou illisible.
func (s *service) loadPresetForTitle(ctx context.Context, id string) *PresetArc {
	preset, err := s.deps.PresetArcs.GetByID(ctx, id)
	if err != nil {
		logUnresolvedOrigin(ctx, "preset_arc", id, err)
		return nil
	}
	return &preset
}

// logUnresolvedOrigin journalise le repli sur le texte stocké : une origine absente du catalogue
// (modèle retiré) est un cas normal (Debug), une lecture en échec est une anomalie (Warn).
func logUnresolvedOrigin(ctx context.Context, kind, id string, err error) {
	level := slog.LevelWarn
	if errors.Is(err, ErrChallengeNotFound) || errors.Is(err, sql.ErrNoRows) {
		level = slog.LevelDebug
	}
	slog.Log(ctx, level, "prestige: origine du texte stocké illisible, texte stocké servi",
		"origin_kind", kind, "origin_id", id, "err", err)
}
