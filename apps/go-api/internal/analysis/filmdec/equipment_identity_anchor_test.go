package filmdec

// equipment_identity_anchor_test.go — L'ANCRAGE PAR CATALOGUE DE TAGS (lot R3, phase 2).
//
// POURQUOI CET INSTRUMENT EXISTE. La phase 1 a établi que le corps d'un record `ti=37`
// d'image-clé ne se lit PAS comme un record NEW du chemin delta : 2 marches bit-exactes sur
// 1 226. Lire un champ « à sa position supposée » est donc exclu — il faut un ANCRAGE qui ne
// dépende d'aucune grammaire.
//
// LA MÉTHODE EST DÉJÀ ÉPROUVÉE DANS CE PAQUET, deux fois. `keyframe_loadout.go` trouve les
// armes portées en balayant la charge utile bit à bit à la recherche d'identifiants de
// FAMILLE connus, puis attribue chaque occurrence au record qui la CONTIENT — et l'ancrage
// anti-hasard est la répartition PAR ARCHÉTYPE de ces occurrences (biped 495, arme au sol
// 397, divers 19 : une répartition sémantiquement juste, imposée par rien dans la méthode).
// `grenade_events.go` établit en outre que les identifiants du flux sont des **GlobalID de
// tag du jeu, décalés d'un bit à gauche**.
//
// CE QUE CET INSTRUMENT FAIT DONC : il construit le catalogue des GlobalID des tags du jeu
// (modules `.module` de l'installation locale), et mesure combien d'entre eux apparaissent
// dans les records d'image-clé, PAR ARCHÉTYPE PORTEUR. Si les tags d'un groupe donné se
// concentrent dans les records `ti=37`, l'identité de l'objet équipement est trouvée — sans
// avoir eu à résoudre la grammaire du record.
//
// LE TÉMOIN EST LA RÉPARTITION ELLE-MÊME : la méthode ne sait pas ce qu'est un `ti=37`, elle
// compte des occurrences. Une concentration sur le bon archétype ne peut pas être un artefact
// du balayage. Le second témoin est le tirage DÉCALÉ (`<<1`), qui doit se comporter comme le
// tirage direct si la lecture est réelle, et comme du bruit sinon.
//
// LECTURE SEULE. Gardé par EQUIP_ID_FILM (le film) ET par la présence de l'installation du
// jeu (`himap.DeployRoot`). Sauté partout ailleurs, CI comprise.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 EQUIP_ID_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestEquipmentIdentityTag' -timeout 30m -v

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/himodule"
)

// eqidTagCatalog est le catalogue des GlobalID des tags du jeu, par groupe fourCC.
type eqidTagCatalog struct {
	byGroup map[string]map[uint32]bool
	total   int
}

// eqidMaxModuleBytes borne la taille d'un module ouvert. `himodule.Open` fait un
// `os.ReadFile` : ouvrir les 132 modules de l'installation (dont un de 7,78 Go) serait une
// BOMBE MÉMOIRE. Les modules sont donc ouverts UN PAR UN, et les trop gros sont annoncés
// puis sautés — un saut annoncé est un résultat, un saut silencieux est un mensonge.
const eqidMaxModuleBytes = 700 << 20

// eqidLoadTagCatalog indexe les tags des modules GLOBAUX DE JEU (`any/globals/*.module`) de
// l'installation locale. C'est le dépôt `any` qu'il faut, et pas `pc` : `pc` ne porte que les
// assets de RENDU (mesuré : 8 groupes, tous graphiques ou sonores — bitm, mode, shbc...),
// alors que les définitions de jeu (armes, projectiles, équipements) vivent dans `any`.
//
// Le glob n'est PAS récursif, ce qui écarte `any/globals/forge/forge_objects` (397 Mo de
// pièces de Forge, sans rapport avec un équipement de joueur).
func eqidLoadTagCatalog(t *testing.T) eqidTagCatalog {
	t.Helper()
	root, err := himap.DeployRoot()
	if err != nil {
		t.Skipf("installation Halo Infinite absente : %v", err)
	}
	globs, err := filepath.Glob(filepath.Join(root, "any", "globals", "*.module"))
	if err != nil || len(globs) == 0 {
		t.Skipf("aucun module global de jeu sous %s (err=%v)", root, err)
	}
	cat := eqidTagCatalog{byGroup: map[string]map[uint32]bool{}}
	for _, p := range globs {
		if fi, err := os.Stat(p); err == nil && fi.Size() > eqidMaxModuleBytes {
			t.Logf("  module SAUTÉ (%.0f Mo > borne) : %s", float64(fi.Size())/(1<<20), filepath.Base(p))
			continue
		}
		eqidIndexModule(t, p, &cat)
	}
	return cat
}

// eqidIndexModule indexe UN module puis rend sa mémoire. Le module est volontairement local à
// cette fonction : le garder vivant sur toute la boucle cumulerait les charges utiles.
func eqidIndexModule(t *testing.T, path string, cat *eqidTagCatalog) {
	t.Helper()
	m, err := himodule.Open(path)
	if err != nil {
		t.Logf("  module illisible, ignoré : %s (%v)", filepath.Base(path), err)
		return
	}
	for _, f := range m.Files("") {
		if f.GlobalID == 0 {
			continue
		}
		// Un groupe non imprimable garde sa valeur brute : l'écarter reviendrait à décider
		// avant de mesurer quels tags ont le droit d'exister.
		g := f.Group
		if g == "" {
			g = "????"
		}
		if cat.byGroup[g] == nil {
			cat.byGroup[g] = map[uint32]bool{}
		}
		if !cat.byGroup[g][f.GlobalID] {
			cat.byGroup[g][f.GlobalID] = true
			cat.total++
		}
	}
}

// TestEquipmentIdentityTagInventory publie le recensement des groupes de tags. Il existe pour
// que le choix du groupe interrogé par l'ancrage soit une DONNÉE, pas une supposition.
func TestEquipmentIdentityTagInventory(t *testing.T) {
	cat := eqidLoadTagCatalog(t)
	groups := make([]string, 0, len(cat.byGroup))
	for g := range cat.byGroup {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return len(cat.byGroup[groups[i]]) > len(cat.byGroup[groups[j]])
	})
	t.Logf("== CATALOGUE DE TAGS — %d GlobalID, %d groupes ==", cat.total, len(groups))
	for i, g := range groups {
		if i >= 40 {
			t.Logf("  ... et %d groupes plus petits", len(groups)-i)
			break
		}
		t.Logf("  %-6s %6d tags", g, len(cat.byGroup[g]))
	}
}

// eqidAnchorTIs est la liste des archétypes contre lesquels l'ancrage est mesuré. Le bipède
// et l'arme au sol servent de TÉMOINS : la méthode y trouve déjà des identifiants d'arme
// (cf. keyframe_loadout.go), donc un balayage qui ne rendrait RIEN nulle part serait suspect.
var eqidAnchorTIs = []int{keyframeBipedTI, EquipmentTypeIndex, 38, ProjectileTypeIndex, GroundWeaponTypeIndex}

// TestEquipmentIdentityTagAnchor mesure, pour chaque groupe de tags candidat, la répartition
// PAR ARCHÉTYPE des occurrences de ses GlobalID dans les records d'image-clé du film.
func TestEquipmentIdentityTagAnchor(t *testing.T) {
	dir := os.Getenv(equipIDFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipIDFilmEnv)
	}
	cat := eqidLoadTagCatalog(t)
	release := LockProcessDecode()
	defer release()

	for _, grp := range eqidAnchorGroups(cat) {
		ids := cat.byGroup[grp]
		if len(ids) == 0 {
			continue
		}
		eqidAnchorReport(t, dir, grp, ids)
	}
}

// eqidAnchorGroups rend les groupes interrogés par l'ancrage. `eqip` est le groupe des
// définitions d'équipement ; `proj` et `weap` sont les TÉMOINS POSITIFS (la méthode y a déjà
// fait ses preuves) ; `sofd` porte la palette de capacités du match.
func eqidAnchorGroups(cat eqidTagCatalog) []string {
	want := []string{"eqip", "proj", "weap", "sofd"}
	out := make([]string, 0, len(want))
	for _, g := range want {
		if len(cat.byGroup[g]) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// eqidAnchorReport balaye toutes les images-clés du film pour UN groupe de tags, dans les
// deux lectures (directe et décalée d'un bit), et publie la répartition par archétype.
func eqidAnchorReport(t *testing.T, dir, group string, ids map[uint32]bool) {
	t.Helper()
	direct := make(map[uint32]bool, len(ids))
	shifted := make(map[uint32]bool, len(ids))
	for id := range ids {
		direct[id] = true
		shifted[id<<1] = true
	}
	hitsDirect := eqidCountByTI(dir, direct)
	hitsShifted := eqidCountByTI(dir, shifted)
	t.Logf("== ANCRAGE groupe `%s` (%d tags) ==", group, len(ids))
	for _, ti := range eqidAnchorTIs {
		t.Logf("    ti=%-3d · direct %5d occurrences · décalé<<1 %5d",
			ti, hitsDirect[ti], hitsShifted[ti])
	}
}

// TestEquipmentIdentityTagClasses mesure la PARTITION que les tags `eqip` induisent sur les
// entités `ti=37` : cardinalité (C1), stabilité par vie d'objet (C2), et distribution des
// valeurs. La globalité inter-films (C3) se lit en rejouant ce test sur plusieurs films.
func TestEquipmentIdentityTagClasses(t *testing.T) {
	dir := os.Getenv(equipIDFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure sauté", equipIDFilmEnv)
	}
	cat := eqidLoadTagCatalog(t)
	eqip := cat.byGroup["eqip"]
	if len(eqip) == 0 {
		t.Skip("groupe `eqip` absent du catalogue local")
	}
	release := LockProcessDecode()
	defer release()

	perLife := map[eqidLifeKey]map[uint32]bool{}
	hist := map[uint32]int{}
	records, withTag, multi := 0, 0, 0
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, r := range WalkKeyframeWorld(pay) {
				if r.TI == EquipmentTypeIndex {
					records++
				}
			}
			for _, rf := range familiesByRecord(pay, eqip, EquipmentTypeIndex) {
				withTag++
				if len(rf.Families) > 1 {
					multi++
				}
				k := eqidLifeKey{uint32(rf.Rec.Slot), uint32(rf.Rec.Gen)}
				if perLife[k] == nil {
					perLife[k] = map[uint32]bool{}
				}
				for _, f := range rf.Families {
					perLife[k][f] = true
					hist[f]++
				}
			}
		}
	}

	t.Logf("== PARTITION PAR TAG `eqip` — %s ==", dir)
	t.Logf("  records ti=%d %d · PORTEURS d'au moins un tag %d (%.1f %%) · records à >1 tag %d",
		EquipmentTypeIndex, records, withTag, 100*float64(withTag)/float64(maxInt(records, 1)), multi)
	t.Logf("  C1 — vies d'objet identifiées %d · classes DISTINCTES %d",
		len(perLife), len(hist))
	stable := 0
	for _, vs := range perLife {
		if len(vs) == 1 {
			stable++
		}
	}
	t.Logf("  C2 — vies à classe UNIQUE %d / %d (%.1f %%)",
		stable, len(perLife), 100*float64(stable)/float64(maxInt(len(perLife), 1)))
	for _, line := range eqidTopValues(eqidWiden(hist), len(hist)) {
		t.Logf("      %s", line)
	}
}

// eqidWiden élargit un histogramme 32 bits vers la forme attendue par eqidTopValues.
func eqidWiden(h map[uint32]int) map[uint64]int {
	out := make(map[uint64]int, len(h))
	for k, v := range h {
		out[uint64(k)] = v
	}
	return out
}

// maxInt évite une division par zéro dans les pourcentages d'un corpus vide.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// eqidCountByTI compte, par archétype porteur, les occurrences de `known` dans les records
// des images-clés du film. Il réutilise `familiesByRecord` — LE balayage du paquet, celui qui
// a déjà servi aux armes portées et aux armes au sol : pas de second balayeur écrit à côté.
func eqidCountByTI(dir string, known map[uint32]bool) map[int]int {
	out := map[int]int{}
	n := CountFilmChunks(dir)
	for c := 1; c <= n; c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(chunk) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			pay := p.Payload(chunk)
			for _, ti := range eqidAnchorTIs {
				for _, rf := range familiesByRecord(pay, known, ti) {
					out[ti] += len(rf.Families)
				}
			}
		}
	}
	return out
}
