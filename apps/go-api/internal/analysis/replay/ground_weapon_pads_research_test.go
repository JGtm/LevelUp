package replay

// ground_weapon_pads_research_test.go — SOCLES par RECURRENCE SPATIALE, et CYCLE de
// reapparition (phase 1 du plan .ai/V7.5/replay2d/PLAN_ARMES_AU_SOL_2E_LECTURE.md).
//
// L'ENTREE EST LE RECORD DE CREATION, PAS LA DISPERSION DES DELTAS. C'est l'arbitrage
// superviseur du 2026-08-17 apres le gate 0 : le critere de dispersion mesurait l'immobilite
// d'une arme au sol sur ses seuls echantillons delta, c'est-a-dire sur sa phase MOBILE — un
// objet qui se pose CESSE d'emettre sa position. La position publiable d'une arme au sol est
// celle de son record de CREATION (`ScanFilmGroundWeaponCreations`), validee par deux oracles
// independants : atterrissage 282/289 et identite 937/947.
//
// LE FILTRE D'IDENTITE EST LE GARDE-FOU, ET IL VIENT D'UNE DECOUVERTE. Le temoin FANTOME du
// balayage de creations n'est PAS discriminant a lui seul (decouverte 2 du plan : sur
// `00162144`, la bande fantome rend 398 creations acceptees contre 366 pour la bande reelle).
// Une creation n'est donc RETENUE que si son mot MPP de 32 bits se resout dans le catalogue
// d'armes de production (`weaponv3.KnownWeaponHigh32`). La seconde formulation de l'arbitrage
// — « egal a la famille high-32 du meme slot a l'image-cle » — est un SOUS-ENSEMBLE de
// celle-ci, parce que `ScanFilmKeyframeGroundWeapons` ne rend que des familles deja connues :
// elle ne s'ajoute donc pas au filtre, elle le CONTROLE (accord publie ci-dessous).
//
// CE QUE L'INSTRUMENT MESURE, UN FILM PAR PROCESSUS :
//
//	1.0  creations `ti=42` acceptees / croisees / ecartees, temoin fantome par le MEME code,
//	     et la part des creations SANS vie delta — les candidats « apparus au repos » ;
//	1.1  classement `dropped` / `spawned` par la regle du 2026-08-18, aux MEMES constantes de
//	     production (`originDropWindowUS` = 2 frames, `originDropMaxDist` = 1,5 m) ;
//	1.2  grappes des apparitions `spawned` (famille + position, rayon 1 m), et le TEMOIN
//	     NEGATIF : les memes grappes sur les apparitions `dropped` (les morts) ;
//	1.3  cycle par grappe : mediane, p10, p90, ecart-type, « etabli » ou non.
//
// POWER-UPS : memes regles sur les objets `ti=37` dont la famille du manifeste commence par
// `powerup_` (`ScanFilmEquipmentPlacements`, LA fonction de production, calibration MPP
// comprise). Le corpus des 12 films du 2026-08-18 n'en portait aucun de socle ; ce qui suit le
// MESURE au lieu de le supposer.
//
// LECTURE SEULE, aucune base (les cartes se lisent hors de cet instrument, dans l'instantane
// parquet du registre). UN SEUL decodage filmdec par process (`LockProcessDecode`), largeurs
// d'axe restaurees (`installWorldObjectPrecision`).
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 GW_PADS=<repo>/data/cache/film_chunks/01e1f945 GW_PADS_MAP=Catalyst \
//	  go test ./internal/analysis/replay/ -run '^TestGroundWeaponPads$' -timeout 30m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
)

const (
	gwPadsEnv       = "GW_PADS"        // repertoire des chunks du film
	gwPadsMapEnv    = "GW_PADS_MAP"    // nom de carte (bornes de dequantification)
	gwPadsBoundsEnv = "GW_PADS_BOUNDS" // catalogue map_quant_bounds.json (defaut : PathResolver)
)

// LES SEUILS, LES CLASSES ET LA REGLE DE GRAPPE SONT EN PRODUCTION (`ground_weapon_rules.go`)
// depuis la phase 3 : `gwPadRadiusM`, `gwPadMinHits`, `gwPadCycleMinGaps`, `gwPadCycleMaxCV`,
// `gwPadApparition`, `gwPadsClass`, `gwPadsIdentity`, `gwPadsWeaponFamily`. Ils ont ete ECRITS
// AVANT la mesure (plan, items 1.2-1.4 et decision 3) et n'ont pas bouge ; cet instrument les
// APPELLE, il n'en garde aucune copie.

func TestGroundWeaponPads(t *testing.T) {
	dir := os.Getenv(gwPadsEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", gwPadsEnv)
	}
	entry := mapQuantEntryFromEnv(t, gwPadsMapEnv, gwPadsBoundsEnv)
	release := filmdec.LockProcessDecode()
	defer release()
	defer installWorldObjectPrecision(entry, dir)()

	wr := entry.Range()
	film, mapName := filepath.Base(dir), os.Getenv(gwPadsMapEnv)
	t.Logf("FILM %s · carte %q (module %s) · largeurs %v", film, mapName, entry.Module, entry.AxisWidths)

	positions, err := filmdec.ScanFilmBipedPositions(dir, gwPadsScanOptions(&wr))
	if err != nil {
		t.Fatalf("nuage des bipedes indisponible : %v", err)
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].TimestampUS < positions[j].TimestampUS
	})
	lives := equipmentLives(positions)
	t.Logf("BIPEDES — %d positions · %d slots · %d vies (fins de vie = les morts)",
		len(positions), len(lives), gwPadsCountLives(lives))

	app := gwPadsPowerups(t, dir, &wr, lives)
	app = append(app, gwPadsWeapons(t, dir, &wr, lives)...)
	for _, a := range app {
		t.Logf("APPAR\t%s\t%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%d\t%s\t%t",
			film, mapName, a.Kind, a.Family, a.X, a.Y, a.Z, a.TUS, a.Class, a.HasDelta)
	}
	gwPadsReport(t, film, mapName, app)
}

// gwPadsScanOptions rend les options de balayage du nuage des bipedes : les bornes de la carte,
// rien de plus. Le CAP n'est PAS capture — cet instrument ne mesure aucune orientation, et
// `CaptureDirs` couterait un decodage supplementaire par record.
func gwPadsScanOptions(wr *filmdec.Vec3Range) filmdec.ScanFilmOptions {
	o := filmdec.DefaultScanFilmOptions()
	o.WorldRange = wr
	return o
}

// gwPadsWeapons rend les apparitions d'ARMES AU SOL retenues, et publie les denominateurs de
// l'item 1.0 : ancres, acceptees, croisees, ecartees, temoin fantome, part sans vie delta.
func gwPadsWeapons(
	t *testing.T, dir string, wr *filmdec.Vec3Range, lives map[uint32][]equipLife,
) []gwPadApparition {
	t.Helper()
	band := filmdec.GroundWeaponSlotBand(dir)
	cre, st, err := filmdec.ScanFilmGroundWeaponCreationsForBand(dir, wr, band)
	if err != nil {
		t.Fatalf("creations ti=42 : %v", err)
	}
	// TEMOIN FANTOME par le MEME code, meme cardinalite : sans lui, un compte d'acceptations
	// ne dit pas si le balayage lit des records ou en invente (decouverte 2 du plan).
	census := gwKeyframeCensus(t, dir)
	pcre, pst, err := filmdec.ScanFilmGroundWeaponCreationsForBand(
		dir, wr, gwPhantomBand(band, census.allSlots))
	if err != nil {
		t.Fatalf("creations ti=42 (fantome) : %v", err)
	}
	known := loadoutFamilies()
	fam := gwPadsKeyframeFamilies(t, dir, known)
	tracks, err := filmdec.ScanFilmWorldObjects(dir, wr, filmdec.GroundWeaponTypeIndex)
	if err != nil {
		t.Fatalf("vies delta ti=42 : %v", err)
	}
	spans := filmdec.EquipmentLifeSpans(tracks)

	out := make([]gwPadApparition, 0, len(cre))
	crossed, pairs, agree, noDelta := 0, 0, 0, 0
	for _, c := range cre {
		w, ok := gwPadsIdentity(c)
		if !ok || !known[w] {
			continue
		}
		crossed++
		if f, seen := fam[filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}]; seen {
			pairs++
			if f == w {
				agree++
			}
		}
		a := gwPadApparition{
			Kind: gwPadKindWeapon, Family: gwPadsWeaponFamily(w),
			X: c.X, Y: c.Y, Z: c.Z, TUS: c.TimestampUS,
			HasDelta: len(spans[filmdec.EquipmentLifeKey{Slot: c.Slot, Gen: c.Gen}]) > 0,
		}
		if !a.HasDelta {
			noDelta++
		}
		a.Class = gwPadsClass(lives, a)
		out = append(out, a)
	}
	pcrossed := 0
	for _, c := range pcre {
		if w, ok := gwPadsIdentity(c); ok && known[w] {
			pcrossed++
		}
	}
	t.Logf("1.0 ARMES ti=42 — bande %d slots · ancres %d · acceptees %d · CROISEES %d"+
		" · ecartees %d || FANTOME ancres %d · acceptees %d · croisees %d",
		st.Slots, st.Anchors, st.Accepted, crossed, st.Accepted-crossed,
		pst.Anchors, pst.Accepted, pcrossed)
	t.Logf("1.0 CONTROLE d'identite (chaine independante : familles high-32 des images-cles)"+
		" — paires croisees %d · accord %d · %s", pairs, agree, gwPadsPart(agree, pairs))
	t.Logf("1.0 SANS VIE DELTA (« apparues au repos ») — %d sur %d retenues (%s) · vies delta"+
		" ti=42 connues %d", noDelta, crossed, gwPadsPart(noDelta, crossed), len(spans))
	return out
}

// gwPadsPowerups rend les apparitions de POWER-UP retenues (`ti=37`, familles `powerup_*` du
// manifeste du titre) en passant par `ScanFilmEquipmentPlacements` — LA fonction de production,
// calibration du bloc MPP comprise. Un film non calibre ne rend AUCUNE pose : c'est le contrat
// de cette fonction, et le publier est ce qui distingue « pas de power-up » de « pas de lecture ».
func gwPadsPowerups(
	t *testing.T, dir string, wr *filmdec.Vec3Range, lives map[uint32][]equipLife,
) []gwPadApparition {
	t.Helper()
	raw, st, err := filmdec.ScanFilmEquipmentPlacements(dir, wr)
	if err != nil {
		t.Logf("1.0 POWER-UPS — balayage ti=37 impossible : %v", err)
		return nil
	}
	familles := origineFamilles(t)
	out, byFamily := []gwPadApparition{}, map[string]int{}
	for _, p := range raw {
		f := familles[p.GlobalID]
		if !strings.HasPrefix(f, "powerup_") {
			continue
		}
		byFamily[f]++
		a := gwPadApparition{
			Kind: gwPadKindPowerup, Family: f,
			X: p.X, Y: p.Y, Z: p.Z, TUS: p.T0US, HasDelta: p.Points > 0,
		}
		a.Class = gwPadsClass(lives, a)
		out = append(out, a)
	}
	t.Logf("1.0 POWER-UPS ti=37 — %s · poses lues %d · dont power-up %d %s",
		st.Calibration, len(raw), len(out), gwPadsFamilyLine(byFamily))
	return out
}

// gwPadsKeyframeFamilies indexe la famille high-32 lue aux IMAGES-CLES par vie (slot, gen) —
// la chaine independante qui CONTROLE le filtre d'identite.
func gwPadsKeyframeFamilies(
	t *testing.T, dir string, known map[uint32]bool,
) map[filmdec.EquipmentLifeKey]uint32 {
	t.Helper()
	out := map[filmdec.EquipmentLifeKey]uint32{}
	kf, err := filmdec.ScanFilmKeyframeGroundWeapons(dir, known)
	if err != nil {
		t.Logf("familles d'image-cle illisibles (%v) : le controle d'identite sera vide", err)
		return out
	}
	for _, g := range kf {
		if len(g.Families) > 0 {
			out[filmdec.EquipmentLifeKey{Slot: g.Slot, Gen: g.Gen}] = g.Families[0]
		}
	}
	return out
}

// gwPadsSetSpawned / gwPadsSetAtRest : les DEUX jeux de candidats compares a chaque etape.
//
// POURQUOI DEUX. L'item 1.1 definit `spawned` par le NEGATIF de la regle de lacher (aucune mort
// a proximite). L'item 1.0 designe, lui, un autre candidat : les creations SANS VIE DELTA, dites
// « apparues au repos ». Les deux ne coincident pas, et l'ecart est instructif — une arme
// echangee par son porteur (l'AR de depart, laché en ramassant autre chose) n'a PAS de mort a
// proximite, donc elle est `spawned`, mais elle TOMBE, donc elle emet des positions. Publier les
// deux jeux est la seule facon de ne pas trancher en douce : le premier est le critere ecrit, le
// second est celui que le plan lui-meme designe comme « socles attendus ».
const (
	gwPadsSetSpawned = "spawned"
	gwPadsSetAtRest  = "at_rest"
)

// gwPadsReport publie les items 1.1 a 1.3 : le classement, le croisement avec la vie delta, les
// grappes des deux jeux de candidats, le TEMOIN NEGATIF sur les `dropped`, et les cycles.
func gwPadsReport(t *testing.T, film, mapName string, app []gwPadApparition) {
	t.Helper()
	byClass := map[string][]gwPadApparition{}
	cross := map[string]int{}
	for _, a := range app {
		byClass[a.Class] = append(byClass[a.Class], a)
		cross[fmt.Sprintf("%s/delta=%t", a.Class, a.HasDelta)]++
	}
	t.Logf("1.1 CLASSEMENT — apparitions retenues %d · dropped %d (%s) · spawned %d (%s)",
		len(app), len(byClass[gwClassDropped]), gwPadsPart(len(byClass[gwClassDropped]), len(app)),
		len(byClass[gwClassSpawned]), gwPadsPart(len(byClass[gwClassSpawned]), len(app)))
	t.Logf("1.1 CROISEMENT classe x vie delta (deux criteres INDEPENDANTS) — %s",
		gwPadsFamilyLine(cross))

	atRest := make([]gwPadApparition, 0, len(byClass[gwClassSpawned]))
	for _, a := range byClass[gwClassSpawned] {
		if !a.HasDelta {
			atRest = append(atRest, a)
		}
	}
	spawnedPads := gwPadsLogSet(t, film, mapName, gwPadsSetSpawned, byClass[gwClassSpawned])
	atRestPads := gwPadsLogSet(t, film, mapName, gwPadsSetAtRest, atRest)

	ghost, _ := gwPadsClusterAssign(byClass[gwClassDropped])
	ghostPads, _ := gwPadsKeep(ghost)
	t.Logf("1.2 TEMOIN NEGATIF (grappes des MORTS, memes rayon et regle) — %d grappes,"+
		" dont %d a >= %d apparitions · %s", len(ghost), len(ghostPads),
		gwPadMinHits, gwPadsSpread(ghost))
	t.Logf("1.3 CYCLES %s — %s", gwPadsSetSpawned, gwPadsCycleSummary(spawnedPads))
	t.Logf("1.3 CYCLES %s — %s", gwPadsSetAtRest, gwPadsCycleSummary(atRestPads))
}

// gwPadsLogSet grappe un jeu de candidats, publie ses socles ligne a ligne, et les rend.
func gwPadsLogSet(
	t *testing.T, film, mapName, set string, app []gwPadApparition,
) []gwPadCluster {
	t.Helper()
	pads, _ := gwPadsClusterAssign(app)
	socles, _ := gwPadsKeep(pads)
	t.Logf("1.2 GRAPPES %s — apparitions %d · %d grappes, dont %d SOCLES (>= %d apparitions)"+
		" · %s", set, len(app), len(pads), len(socles), gwPadMinHits, gwPadsSpread(pads))
	for _, p := range socles {
		c := gwPadsCycle(p.TS)
		t.Logf("PAD\t%s\t%s\t%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%d\t%.1f\t%.1f\t%.1f\t%.1f\t%t",
			film, mapName, set, p.Kind, p.Family, p.X, p.Y, p.Z, len(p.TS),
			c.MedianS, c.P10S, c.P90S, c.SDS, c.Established)
	}
	return socles
}

// gwPadsCountLives compte les vies de bipede toutes tranches confondues.
func gwPadsCountLives(lives map[uint32][]equipLife) int {
	n := 0
	for _, v := range lives {
		n += len(v)
	}
	return n
}

// gwPadsFamilyLine rend le detail par famille, trie — un total sans detail ne se relit pas.
func gwPadsFamilyLine(m map[string]int) string {
	if len(m) == 0 {
		return "(aucune)"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func gwPadsPart(k, n int) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d=%.1f%%", k, n, 100*float64(k)/float64(n))
}

// mapQuantEntryFromEnv charge l'entree de catalogue de bornes de la carte nommee par `mapEnv`.
// Sans elle, pas de METRES — et tous les seuils de ce chantier sont en metres : on echoue
// plutot que de mesurer dans une unite muette.
//
// LE CHEMIN PASSE PAR `PathResolver`, jamais par un `filepath.Join(..., "data", ...)` a la main
// (regle du depot). `boundsEnv` ne sert qu'a pointer un catalogue de rechange.
func mapQuantEntryFromEnv(t *testing.T, mapEnv, boundsEnv string) filmdec.MapQuantEntry {
	t.Helper()
	nom := os.Getenv(mapEnv)
	if nom == "" {
		t.Fatalf("%s absent : la carte porte les bornes de dequantification, sans elle la"+
			" distance n'est pas en metres", mapEnv)
	}
	chemin := os.Getenv(boundsEnv)
	if chemin == "" {
		chemin = title.NewPathResolver(repoRootForTest(t)).MapQuantBoundsPath(title.DefaultSlug)
	}
	cat, err := filmdec.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatalf("catalogue de bornes illisible (%s) : %v", chemin, err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q absente du catalogue : %v", nom, err)
	}
	return e
}
