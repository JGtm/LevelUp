package port

import "context"

// MedalDefinitionRow est une ligne résolue depuis medal_definitions (metadata DB).
type MedalDefinitionRow struct {
	MedalID     int64
	Label       string
	Description string
}

// MedalDefinitionsRepository résout les labels et descriptions de médailles
// depuis la base metadata (medal_definitions + medal_translations).
type MedalDefinitionsRepository interface {
	LookupByIDs(ctx context.Context, ids []int64) (map[int64]MedalDefinitionRow, error)
}
