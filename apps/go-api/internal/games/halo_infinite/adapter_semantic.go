package halo_infinite

import (
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/mappings"
)

// SemanticAdapter est l'implémentation HI de games.TitleSemanticAdapter.
//
// Il enveloppe un *mappings.FieldMappingSet chargé au boot depuis
// config/titles/halo_infinite/mappings/fields.toml.
type SemanticAdapter struct {
	fields *mappings.FieldMappingSet
}

// NewSemanticAdapter construit un semantic adapter HI.
//
// fields ne doit pas être nil — la validation au boot doit garantir qu'un
// FieldMappingSet a été chargé pour le slug HI avant d'instancier l'adapter.
// Si fields est nil, NewSemanticAdapter retourne nil (signal au boot que le
// titre n'a pas de mapping disponible).
func NewSemanticAdapter(fields *mappings.FieldMappingSet) *SemanticAdapter {
	if fields == nil {
		return nil
	}
	return &SemanticAdapter{fields: fields}
}

// TitleSlug retourne le slug HI canonique.
func (a *SemanticAdapter) TitleSlug() string { return titlePkg.DefaultSlug }

// SchemaVersion retourne la version du schéma TOML chargé.
func (a *SemanticAdapter) SchemaVersion() int { return a.fields.SchemaVersion() }

// Fields retourne le FieldMappingSet chargé pour HI.
func (a *SemanticAdapter) Fields() *mappings.FieldMappingSet { return a.fields }
