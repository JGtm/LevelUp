//go:build research

package replay

// powerup_socle_research_test.go — LE POWER-UP DE SOCLE AU CENTRE DE CATALYST.
//
// CE QUE CET INSTRUMENT CHERCHE. L'utilisateur decrit (2026-08-18) un power-up pose sur un
// socle au centre de Catalyst, sur un pont — camouflage ou surbouclier selon le sous-mode.
// La chaine de production ne le voit NULLE PART : `equipmentPlacements` des artefacts cuits
// de Catalyst ne porte qu'un seul `powerup_overshield`, `dropped`, avec porteur.
//
// POURQUOI CE NEGATIF NE PROUVE RIEN, et c'est le point de depart du lot. `ScanFilmEquipment
// Placements` ne retient un record de creation `ti=37` que si sa position retombe sur le
// PREMIER POINT d'une vie decodee des paquets delta (`confirmPlacements` ->
// `MatchEquipmentLife`). Or un objet POSE cesse d'emettre sa position. Un objet de socle qui
// ne bouge JAMAIS n'a donc aucune vie delta, donc aucun record confirmable : il est invisible
// a cette chaine PAR CONSTRUCTION. La chaine `ti=42` (armes au sol), elle, retient les
// creations SANS vie delta et filtre par IDENTITE — c'est l'asymetrie que ce lot exploite.
//
// LECTURE SEULE, aucune base, aucune ecriture. Plan : `.ai/V7.5/replay2d/
// PLAN_POWERUP_SOCLE_CATALYST.md` (hypotheses et seuils ECRITS AVANT la mesure).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 OBJ_FILM_ART=<depot>/data/cache/replays/halo_infinite \
//	  go test ./internal/games/halo_infinite/film/replay/ -run '^TestPowerupSocleAncrage$' -v
//
//	CGO_ENABLED=0 OBJ_FILM=<depot>/data/cache/film_chunks/01e1f945 OBJ_FILM_MAP=Catalyst \
//	  go test ./internal/games/halo_infinite/film/replay/ -run '^TestPowerupSocle' -timeout 60m -v

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"os"
	"testing"
)

// LA GARDE DU FILM EST CELLE DE TOUT LE PAQUET : `OBJ_FILM` porte la RACINE du cache film
// (le repertoire qui contient `film_chunks/`), et `objRequireRoot` la lit. Ce fichier ne la
// redeclare pas — une seconde definition de la meme garde divergerait au premier correctif.

// TestPowerupSocleAncrage — PHASE 0 du plan : le centre de Catalyst, et l'etat du corpus.
//
// Ne decode AUCUN film : il ne lit que les artefacts cuits. C'est ce qui en fait l'ancrage —
// la cible spatiale des phases suivantes se pose avant qu'un seul bit de film soit lu.
func TestPowerupSocleAncrage(t *testing.T) {
	dir := psArtDir(t)
	var docs []ReplayDocument
	t.Log("=== 0.1 CORPUS ===")
	for _, f := range psFilmsCatalyst {
		doc, ok := psLoadDoc(t, dir, f.ID)
		if !ok {
			t.Logf("  %s (%s) : artefact ABSENT", f.ID, f.Mode)
			continue
		}
		docs = append(docs, doc)
		t.Logf("  %s (%s) : %d images, %d socles d'arme, %d poses, %d episodes"+
			" | surbouclier %d vies / %d episodes, camo %d vies",
			f.ID, f.Mode, doc.FrameCount, len(doc.WeaponPads), len(doc.EquipmentPlacements),
			len(doc.EquipmentEpisodes), doc.Coverage.Equipment.OvershieldLives,
			doc.Coverage.Equipment.OvershieldEpisodes, doc.Coverage.Equipment.CamoLives)
	}
	if len(docs) == 0 {
		t.Skipf("aucun artefact Catalyst dans %s : instrument saute", dir)
	}

	pads := psSoclesUniques(docs)
	ct := psCentreDesSocles(pads)
	t.Log("=== 0.1 CENTRE DE CATALYST (socles publies) ===")
	for _, p := range pads {
		t.Logf("  socle (%.3f ; %.3f)", p.X, p.Y)
	}
	t.Logf("  %d socles uniques | %d paires miroir -> axe y = %.4f", len(pads), ct.Paires, ct.YAxe)
	t.Logf("  %d socles sur l'axe, x de %.3f a %.3f -> milieu %.4f",
		ct.SurAxe, ct.XMin, ct.XMax, float64(ct.C.X))
	t.Logf("  CENTRE RETENU = (%.3f ; %.3f)", ct.C.X, ct.C.Y)

	t.Log("=== 0.1 CONTROLE : milieu des bornes des positions JOUEES ===")
	for i, d := range docs {
		mx := (d.Bounds.MinX + d.Bounds.MaxX) / 2
		my := (d.Bounds.MinY + d.Bounds.MaxY) / 2
		t.Logf("  film %d : milieu (%.3f ; %.3f), ecart au centre retenu %.3f m",
			i, mx, my, psDist(psPoint{X: mx, Y: my}, ct.C))
	}
	if ct.Paires < 2 || ct.SurAxe < 2 {
		t.Fatalf("centre non etabli : %d paires miroir, %d socles sur l'axe", ct.Paires, ct.SurAxe)
	}
}

// psMapEnv porte le nom de carte ; psBoundsEnv le catalogue de bornes. Les deux ont un
// defaut : les quatre films du lot sont tous Catalyst, et le catalogue vit dans le depot.
const (
	psMapEnv    = "OBJ_FILM_MAP"
	psBoundsEnv = "OBJ_FILM_BOUNDS"
	psCarte     = "Catalyst"
)

// psEntreeCarte rend les bornes de dequantification de la carte du lot. Elle passe par le
// MEME helper que l'instrument des socles d'arme (`mapQuantEntryFromEnv`) : une seconde
// lecture du catalogue divergerait au premier correctif.
func psEntreeCarte(t *testing.T) profile.MapQuantEntry {
	t.Helper()
	if os.Getenv(psMapEnv) == "" {
		t.Setenv(psMapEnv, psCarte)
	}
	return mapQuantEntryFromEnv(t, psMapEnv, psBoundsEnv)
}
