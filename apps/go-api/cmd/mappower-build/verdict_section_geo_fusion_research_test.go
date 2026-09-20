//go:build research

package main

// verdict_section_geo_fusion_research_test.go — LA SECTION 9 DU VERDICT : le diagnostic
// geometrique quand la lignee jugee est la geometrie seule ; sinon, ou trouver les
// documents des autres lignees et comment les regenerer.

import (
	"fmt"
	"strings"
)

// documentsDesLignees : le document de verdict de chaque fichier de positions.
var documentsDesLignees = map[string]string{
	positionsGeoNom:       "VERDICT_GEO_2026-09-20.md",
	positionsFusionNom:    "VERDICT_FUSION_2026-09-20.md",
	verdictV2PositionsNom: verdictV2SortieNom,
}

// sectionGeoEtFusion ecrit la section 9.
func sectionGeoEtFusion(b *strings.Builder, v verdictV2Rendu) {
	fmt.Fprintf(b, "## 9. Geometrie et fusion\n\n")
	if v.DiagnosticsGeo != nil || strings.HasSuffix(v.CheminPositions, positionsGeoNom) {
		tableDiagnosticGeo(b, v.DiagnosticsGeo)
	}
	fmt.Fprintf(b, "Chaque lignee a son document, genere par la meme commande avec son fichier de positions"+
		" (depuis `apps/go-api`, `%s=<racine contenant data/>`) :\n\n", verdictDataRootEnv)
	for _, l := range ligneesConnues {
		nomDoc, ok := documentsDesLignees[l.Fichier[strings.LastIndexAny(l.Fichier, `/\`)+1:]]
		if !ok {
			continue
		}
		fmt.Fprintf(b, "- **%s** : `%s=%s/%s %s=%s go test -tags research ./cmd/mappower-build/ -run VerdictV2`\n",
			l.Nom, verdictV2PositionsEnv, verdictDossier, strings.ReplaceAll(l.Fichier, `\`, "/"),
			verdictV2SortieEnv, nomDoc)
	}
	fmt.Fprintf(b, "\nLe reglage de fusion (`fusion.ReglageFusionV1`) a ete choisi par balayage sur les cartes de"+
		" calibrage seulement : `%s/%s`. Le diagnostic v2 (section 7) ne se rejoue que sur un fichier"+
		" empirique v2 ; le diagnostic geometrique ci-dessus, que sur le fichier geometrique.\n\n",
		verdictFusion, calibrageNom)
}
