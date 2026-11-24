package migrations

// steps_shared_highlight_events_dedupe.go — shared_highlight_events_dedupe_v1 : retire de
// highlight_events les lignes écrites deux fois (forme sans code doublée par sa jumelle avec code,
// match recopié en bloc par un rejeu). Règle, mesures et garde-fous :
// migration/highlight_events_dedupe.go.
//
// BORNÉ À CE TITRE (`OnlyTitles`) : la règle n'a été mesurée que sur Halo Infinite, et la base
// partagée de Halo 5 porte des milliers de lignes identiques légitimes (deux assistances ou deux
// médailles du même joueur au même instant). Halo 5 hérite du registre pour la cible `shared`
// (cf. title_set.go) : sans la borne, le step y tournerait aussi.

import "levelup/go-api/internal/migration"

func stepHighlightEventsDedupe() migration.Migration {
	return migration.Migration{
		Name:     "shared_highlight_events_dedupe_v1",
		TargetDB: migration.TargetShared,
		Description: "Halo Infinite : retire de highlight_events (swap sans DELETE) les lignes sans code " +
			"doublées par leur jumelle avec code et la seconde moitié des matchs recopiés en bloc",
		OnlyTitles:  []string{migration.DefaultSlug},
		ApplySchema: migration.ApplyHighlightEventsDedupe,
	}
}
