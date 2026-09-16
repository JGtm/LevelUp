package filmdec

// f0_103_contexte_research_test.go — LOT F.0 (plan .ai/PLAN_FINITIONS_2026-09-13.md) :
// LE CONTEXTE DE MESURE du type 103 `EquipmentSpawnedObject`.
//
// # LA QUESTION DU LOT
//
// Decision utilisateur du 2026-09-13 : « jamais un equipement ne doit avoir un evenement sur
// une regle arbitraire ; le film dit s il est utilise ou deploye ». Ce qui est PROUVE a ce
// jour est seulement que la lecture EN TETE DE LISTE du type 103 (rapport R5 §3.2) ne separe
// pas le deploiement du lacher a la mort. Trois choses n avaient jamais ete faites :
//
//  1. marcher le 103 dans la LISTE COMPLETE (le marcheur de R7 existe) ;
//  2. RESOUDRE ses references (R5 §3.1 : « ref0 ≈ objet 13 bits, ref1 ≈ objet 13 bits »,
//     jamais resolues contre quoi que ce soit) ;
//  3. recenser les creations `ti=37` TOUTES CATEGORIES autour d une consommation de charge.
//
// Ce fichier porte le CONTEXTE (calibration de carte, balayages, marche) ; les VERDICTS sont
// dans `f0_103_verdicts_research_test.go`.
//
// # CE QUE LA RESOLUTION DES REFERENCES SIGNIFIE ICI
//
// Une reference gardee est `[1 porte][R(w) index][R(2) generation]` (grammaire R7 §1.2). Pour
// le type 103 les domaines sont {0,0,7}, soit TROIS references de 13 bits — exactement la
// largeur de `FrameConfig.IDLowBits`, celle de l en-tete NEW d un record de creation
// (`woNewSlotBits`). L hypothese testee est donc mecanique : `(index, generation)` d une
// reference du 103 est la cle de vie d une entite, la MEME paire que `EquipmentLifeKey` et que
// `EquipmentCreation.Slot/Gen`. Le test est un APPARIEMENT EXACT, pas une fenetre de temps.
//
// # POURQUOI LES COORDONNEES NE SONT PAS JUGEES ICI
//
// Le balayage des poses est INVARIANT D ECHELLE : la position dequantifiee vaut
// `min + (q+0,5)·etendue/2^w` et le rayon d accord de l oracle vaut `mppCalibPosEps·etendue`
// (EquipmentPosEps) — les deux sont proportionnels a l etendue, et `min` se simplifie dans la
// difference. Changer les bornes d une carte pour celles d une autre de MEMES largeurs d axe
// ne change donc NI le jeu de records acceptes NI leur (slot, gen, GlobalID, instant) : seules
// les coordonnees en metres changent. Ce lot ne lit que la premiere liste. Le controle est
// mesure par `TestF0InvarianceEchelle`.
//
// LECTURE SEULE : aucun fichier de production touche, aucune DuckDB, aucune ecriture sous
// `data/`. Skip par defaut (gardes d environnement), `CGO_ENABLED=0`.
//
//	CGO_ENABLED=0 \
//	  F0_ROOT=<depot>/data/cache/film_chunks \
//	  F0_ARTS=<depot>/data/cache/replays/halo_infinite \
//	  F0_CAT=<depot>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  F0_IDS=000d5950,1cd3848a,215e7022 F0_MAPS=000d5950=Cliffhanger,... \
//	  go test ./internal/games/halo_infinite/film/filmdec/ -run '^TestF0' -count=1 -timeout 90m -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	f0RootEnv   = "F0_ROOT"   // racine des chunks de film
	f0ArtsEnv   = "F0_ARTS"   // racine des artefacts de rejeu
	f0IDsEnv    = "F0_IDS"    // films a mesurer, separes par des virgules
	f0CatEnv    = "F0_CAT"    // map_quant_bounds.json
	f0MapsEnv   = "F0_MAPS"   // "id8=NomDeCarte,..." — sortie de TestF0CalibreCarte
	f0LabelsEnv = "F0_LABELS" // replay_labels.toml du titre
)

// f0Ev103 est UNE occurrence du type 103 lue dans la liste, avec de quoi la juger.
type f0Ev103 struct {
	Film   string
	Chunk  int
	Paquet int
	// TsUS est l horodatage MOTEUR du paquet : la meme horloge que
	// `EquipmentCreation.TimestampUS` et `EquipmentPlacement.T0US`. Aucun recalage n est
	// donc necessaire entre l evenement et le record de creation.
	TsUS uint64
	// Pos est le rang dans la liste (1 = tete, cadrage CERTAIN).
	Pos int
	// ListePropre dit que la liste qui porte cet evenement s est fermee sur son bit de
	// continuation. Faux : la marche a derive APRES cet evenement — les evenements de
	// position 1 restent certains, les suivants sont a lire avec la reserve.
	ListePropre bool
	Refs        [3]r7RefVal
}

// f0Film porte tout ce qu un film rend a la mesure.
type f0Film struct {
	ID    string
	Dir   string
	Carte string
	Entry MapQuantEntry
	// BaseUS est l horodatage du PREMIER paquet du chunk 1 : le zero de l horloge FILM, celui
	// auquel `originMs` de l artefact se rapporte (conversion etablie par R1 §0, reprise par
	// R5). `ms_film = (ts_paquet - BaseUS)/1000`.
	BaseUS uint64
	// Ev103 : toutes les occurrences du type 103 de la LISTE COMPLETE.
	Ev103 []f0Ev103
	// Listes / ListesPropres / Ev103Tete : les denominateurs de la marche sur ce film.
	Listes, ListesPropres, Ev103Tete int
	// Creations est le balayage BRUT des records `ti=37` (sans l oracle de vie delta) :
	// TOUTES les apparitions d objet d equipement, y compris celles que l artefact ne
	// publie pas. C est le denominateur de la question 4.
	Creations []EquipmentCreation
	CreStats  EquipmentCreationStats
	// Places est la chaine de PRODUCTION : les poses confirmees par l oracle de vie, celles
	// que l artefact publie.
	Places     []EquipmentPlacement
	PlaceStats EquipmentPlacementStats
	// Projectiles est le balayage `ti=41` : la SECONDE nature d objet que le 103 peut
	// engendrer. Sans elle, une reference qui designe un projectile se lirait « non resolue »,
	// et le taux de resolution du 103 serait sous-estime sans qu on sache pourquoi.
	Projectiles []ProjectileTrack
}

// f0Films rend la racine et les films demandes, ou skip l instrument.
func f0Films(t *testing.T) (string, []string) {
	t.Helper()
	root, ids := os.Getenv(f0RootEnv), os.Getenv(f0IDsEnv)
	if root == "" || ids == "" {
		t.Skipf("instrument F.0 : definir %s et %s", f0RootEnv, f0IDsEnv)
	}
	var out []string
	for _, id := range strings.Split(ids, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return root, out
}

// f0Catalogue charge le catalogue de bornes de production.
func f0Catalogue(t *testing.T) *MapQuantCatalog {
	t.Helper()
	path := os.Getenv(f0CatEnv)
	if path == "" {
		t.Skipf("instrument F.0 : definir %s (map_quant_bounds.json)", f0CatEnv)
	}
	cat, err := LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	return cat
}

// f0Cartes lit `F0_MAPS` ("id8=NomDeCarte,...") et resout chaque nom dans le catalogue.
func f0Cartes(t *testing.T, cat *MapQuantCatalog) map[string]MapQuantEntry {
	t.Helper()
	spec := os.Getenv(f0MapsEnv)
	if spec == "" {
		t.Skipf("instrument F.0 : definir %s — le lancer d abord par TestF0CalibreCarte", f0MapsEnv)
	}
	out := map[string]MapQuantEntry{}
	for _, kv := range strings.Split(spec, ",") {
		i := strings.IndexByte(kv, '=')
		if i <= 0 {
			continue
		}
		id, nom := strings.TrimSpace(kv[:i]), strings.TrimSpace(kv[i+1:])
		e, err := cat.Lookup(nom)
		if err != nil {
			t.Fatalf("film %s : carte %q hors catalogue (%v)", id, nom, err)
		}
		out[id] = e
	}
	return out
}

// f0Ctx construit le contexte de marche (etendues et largeur d index de region) depuis une
// entree de catalogue. Meme forme que `r7CtxDeCarte`, sur le type de PRODUCTION.
func f0Ctx(e MapQuantEntry) r7Ctx {
	return r7Ctx{
		etendues: [3]float64{
			float64(e.Max[0] - e.Min[0]),
			float64(e.Max[1] - e.Min[1]),
			float64(e.Max[2] - e.Min[2]),
		},
		regionBits: uint(e.EffectiveRegionIndexBits()),
		hasMap:     true,
	}
}

// f0BaseUS rend l horodatage du premier paquet du chunk 1 — le zero de l horloge FILM.
func f0BaseUS(t *testing.T, dir string) uint64 {
	t.Helper()
	raw, err := ReadFilmChunk(dir, 1)
	if err != nil {
		t.Fatalf("chunk 1 illisible (%s) : %v", dir, err)
	}
	pks := WalkPackets(raw)
	if len(pks) == 0 {
		t.Fatalf("aucun paquet dans le chunk 1 de %s", dir)
	}
	return pks[0].TimestampUS
}

// f0Marche103 marche la liste COMPLETE de tous les paquets delta du film et rend les
// occurrences du type 103, plus les denominateurs de la marche.
func f0Marche103(t *testing.T, id, dir string, ctx r7Ctx) ([]f0Ev103, int, int) {
	t.Helper()
	n := CountFilmChunks(dir)
	var out []f0Ev103
	listes, propres := 0, 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0]&0x40 == 0 { // bit de continuation a 0 : liste vide
				continue
			}
			listes++
			evs, stop, _, _ := r7Marche(pay, ctx)
			propre := stop == r7StopFin
			if propre {
				propres++
			}
			for _, e := range evs {
				if e.Typ != 103 {
					continue
				}
				out = append(out, f0Ev103{
					Film: id, Chunk: c, Paquet: pk.Index, TsUS: pk.TimestampUS,
					Pos: e.Pos, ListePropre: propre, Refs: e.Refs,
				})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TsUS < out[j].TsUS })
	return out, listes, propres
}

// f0Charge assemble le contexte complet d un film : marche, poses de production, balayage
// brut des creations `ti=37`.
//
// LES GLOBAUX DE PAQUET SONT POSES ET RESTAURES : `WorldObjectPrecision` (largeurs de la
// carte) et les largeurs du bloc MPP (mesurees sur CE film par la chaine de production, que
// `ScanFilmEquipmentPlacements` restaure en sortant — il faut les REPOSER avant le balayage
// brut, sans quoi aucune identite ne se resout et rien ne le dit).
func f0Charge(t *testing.T, root, id string, e MapQuantEntry) f0Film {
	t.Helper()
	dir := filepath.Join(root, id)
	if CountFilmChunks(dir) == 0 {
		t.Fatalf("film %s : aucun chunk dans %s", id, dir)
	}
	release := LockProcessDecode()
	defer release()
	prevPrec := WorldObjectPrecisionActuelle()
	SetWorldObjectPrecisionFromLayout(e.Layout())
	defer func() { PoserWorldObjectPrecision(prevPrec) }()

	f := f0Film{ID: id, Dir: dir, Entry: e, BaseUS: f0BaseUS(t, dir)}
	f.Ev103, f.Listes, f.ListesPropres = f0Marche103(t, id, dir, f0Ctx(e))
	for _, ev := range f.Ev103 {
		if ev.Pos == 1 {
			f.Ev103Tete++
		}
	}
	wr := e.Range()
	pl, pst, err := ScanFilmEquipmentPlacements(dir, &wr)
	if err != nil {
		t.Fatalf("film %s : poses illisibles : %v", id, err)
	}
	f.Places, f.PlaceStats = pl, pst
	prevMPP := SetMPPWidths(pst.Calibration.Widths)
	defer SetMPPWidths(prevMPP)
	cre, cst, err := ScanFilmEquipmentCreations(dir, &wr)
	if err != nil {
		t.Fatalf("film %s : creations illisibles : %v", id, err)
	}
	f.Creations, f.CreStats = cre, cst
	proj, err := ScanFilmProjectiles(dir, &wr)
	if err != nil {
		t.Fatalf("film %s : projectiles illisibles : %v", id, err)
	}
	f.Projectiles = proj
	return f
}

// f0CleVie est la cle d une VIE d entite : (index, generation). C est la paire que l en-tete
// NEW d un record de creation ecrit, et celle qu une reference gardee porte.
type f0CleVie struct {
	Index uint64
	Gen   uint32
}

// f0BaseRef est la BASE de l index d une reference gardee : la valeur lue dans le flux est
// relative, le slot d entite est `base + index`.
//
// 512 N EST PAS UN REGLAGE, C EST UNE MESURE. R1 l avait deja etabli sur le type 117
// (« index 8 bits base 512 »), et la sonde de ce lot le retrouve a l identique sur le
// type 103 : sur le film `9e8fb31b`, les cinq premieres occurrences portent ref1 = 1011,
// 1013, 1071, 1125, 1182 pendant qu une creation `ti=37` de PANNEAU DE MUR (`0x528fce46`)
// nait 33 ms plus tot aux slots 1523, 1525, 1583, 1637, 1694 — ecart CONSTANT de 512, cinq
// fois de suite. Les deux bases sont mesurees cote a cote par le rapport (0 et 512), avec
// leur temoin de hasard : c est la mesure qui tranche, jamais ce commentaire.
const f0BaseRef = 512

// f0CleDe rend la cle de vie d une reference gardee, base appliquee.
func f0CleDe(r r7RefVal, base uint64) (f0CleVie, bool) {
	if !r.Present {
		return f0CleVie{}, false
	}
	return f0CleVie{Index: r.Index + base, Gen: r.Gen}, true
}

// TestF0CalibreCarte identifie la carte de chaque film par l ORACLE DE TRAME, seul juge du
// cadrage (temoin 3 de R7). Les candidats sont les PROFILS D ETENDUE DISTINCTS du catalogue
// de production : des cartes reelles, jamais un balayage libre.
//
// Sortie : la ligne `F0_MAPS=...` a reporter dans les lancers suivants.
func TestF0CalibreCarte(t *testing.T) {
	root, ids := f0Films(t)
	cat := f0Catalogue(t)
	profils := f0Profils(cat)
	if len(profils) == 0 {
		t.Skipf("catalogue vide (%s)", f0CatEnv)
	}
	release := LockProcessDecode()
	defer release()
	t.Logf("%d profils d etendue distincts sur %d cartes du catalogue", len(profils), len(cat.Maps))
	garde := func(evs []r7Ev) bool {
		for _, e := range evs {
			if e.Typ == 117 || e.Typ == 82 {
				return true
			}
		}
		return false
	}
	var retenues []string
	for _, id := range ids {
		dir := filepath.Join(root, id)
		reg, chunks, err := r7Chargements(dir)
		if err != nil || len(chunks) == 0 {
			t.Logf("film %s : illisible (%v) — ignore", id, err)
			continue
		}
		// PREMIER FILTRE, ET IL NE VIENT PAS DU CATALOGUE : le decoupage d i0 se LIT dans le
		// film (DetectI0Layout, profil de bascule par position de bit). Ne restent candidates
		// que les cartes dont les largeurs d axe du catalogue s y accordent — c est le
		// controle que reclame le commentaire d `AxisWidths` (R7 : 7 films sur 7).
		lay, _, lerr := detectI0Layout(dir)
		cands := profils
		if lerr == nil && lay.Valid() {
			cands = nil
			for _, p := range profils {
				if p.Entry.AxisWidths == lay.AxisW {
					cands = append(cands, p)
				}
			}
		}
		cfg := DefaultFrameConfig()
		cfg.IDLowBits, _ = r7CalibreIDLow(reg, chunks)
		best, bestProf, second := "", -1.0, -1.0
		for _, p := range cands {
			st, _ := r7OracleFilm(reg, chunks, f0Ctx(p.Entry), cfg, garde, 0)
			prof := st.profondeur()
			if prof > bestProf {
				second, bestProf, best = bestProf, prof, p.Nom
			} else if prof > second {
				second = prof
			}
		}
		t.Logf("film %s : decoupage i0 LU %v (err %v) · %d candidats de meme largeur · "+
			"carte retenue %-22s profondeur %.3f (2e %.3f) · IDLowBits=%d",
			id, lay.AxisW, lerr, len(cands), best, bestProf, second, cfg.IDLowBits)
		retenues = append(retenues, id+"="+best)
	}
	t.Logf("F0_MAPS=%s", strings.Join(retenues, ","))
}

// f0Profil est un candidat de carte pour la calibration.
type f0Profil struct {
	Nom   string
	Entry MapQuantEntry
}

// f0Profils rend UN representant par profil d etendue distinct du catalogue. Deux cartes de
// memes largeurs d axe ET de memes etendues sont indiscernables par l oracle : en garder deux
// ferait croire a une ambiguite la ou il n y en a pas.
func f0Profils(cat *MapQuantCatalog) []f0Profil {
	vus := map[string]bool{}
	noms := make([]string, 0, len(cat.Maps))
	for nom := range cat.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var out []f0Profil
	for _, nom := range noms {
		e := cat.Maps[nom]
		cle := fmt.Sprintf("%.3f/%.3f/%.3f/%d", e.Max[0]-e.Min[0], e.Max[1]-e.Min[1],
			e.Max[2]-e.Min[2], e.EffectiveRegionIndexBits())
		if vus[cle] {
			continue
		}
		vus[cle] = true
		out = append(out, f0Profil{Nom: nom, Entry: e})
	}
	return out
}

// TestF0InvarianceEchelle mesure la reserve annoncee en tete de fichier : deux entrees de
// catalogue de MEMES largeurs d axe rendent le MEME jeu de (slot, gen, GlobalID, instant),
// et seules les coordonnees en metres changent. C est ce qui autorise a juger les references
// du 103 sans avoir prouve la carte de chaque film.
func TestF0InvarianceEchelle(t *testing.T) {
	root, ids := f0Films(t)
	cat := f0Catalogue(t)
	cartes := f0Cartes(t, cat)
	id := ids[0]
	e, ok := cartes[id]
	if !ok {
		t.Skipf("film %s absent de %s", id, f0MapsEnv)
	}
	jumelle, nom, trouve := f0Jumelle(cat, e)
	if !trouve {
		t.Skipf("aucune carte du catalogue ne partage les largeurs d axe %v sans partager ses bornes",
			e.AxisWidths)
	}
	t.Logf("film %s : bornes de reference %v..%v — jumelle d echelle %q %v..%v",
		id, e.Min, e.Max, nom, jumelle.Min, jumelle.Max)
	a := f0Signatures(t, root, id, e)
	b := f0Signatures(t, root, id, jumelle)
	if len(a) == 0 {
		t.Fatalf("film %s : aucune creation lue — l invariance ne se mesure pas sur le vide", id)
	}
	ecart := 0
	for k := range a {
		if !b[k] {
			ecart++
		}
	}
	for k := range b {
		if !a[k] {
			ecart++
		}
	}
	t.Logf("INVARIANCE D ECHELLE : %d signatures (slot,gen,GlobalID,instant) de reference, "+
		"%d avec la jumelle, %d ecarts — attendu 0", len(a), len(b), ecart)
	if ecart != 0 {
		t.Errorf("invariance d echelle REFUTEE : %d ecarts", ecart)
	}
}

// f0Jumelle cherche une entree de MEMES largeurs d axe et de MEMES bits de region que `e`,
// mais de bornes DIFFERENTES : le temoin de l invariance d echelle.
func f0Jumelle(cat *MapQuantCatalog, e MapQuantEntry) (MapQuantEntry, string, bool) {
	noms := make([]string, 0, len(cat.Maps))
	for nom := range cat.Maps {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	for _, nom := range noms {
		c := cat.Maps[nom]
		if c.AxisWidths != e.AxisWidths || c.Region != e.Region ||
			c.EffectiveRegionIndexBits() != e.EffectiveRegionIndexBits() {
			continue
		}
		if c.Min != e.Min || c.Max != e.Max {
			return c, nom, true
		}
	}
	return MapQuantEntry{}, "", false
}

// f0Signatures rend les signatures (slot, gen, GlobalID, instant) des creations `ti=37` lues
// avec les bornes donnees — tout ce que ce lot juge, coordonnees exclues.
func f0Signatures(t *testing.T, root, id string, e MapQuantEntry) map[string]bool {
	t.Helper()
	f := f0Charge(t, root, id, e)
	out := make(map[string]bool, len(f.Creations))
	for _, c := range f.Creations {
		out[fmt.Sprintf("%d/%d/%08x/%d", c.Slot, c.Gen,
			uint32(c.MPPVal[MPPWord32]), c.TimestampUS)] = true
	}
	return out
}
