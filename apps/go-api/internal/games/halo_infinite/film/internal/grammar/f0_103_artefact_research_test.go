package grammar

// f0_103_artefact_research_test.go — LOT F.0 : L ARTEFACT COMME VERITE D ORIGINE, et la
// JOINTURE avec le balayage du film.
//
// # POURQUOI L ARTEFACT ET PAS UN RECLASSEMENT MAISON
//
// La question 2 du lot demande « pour chaque pose classee `dropped` … ». Ce classement est
// une DECISION DE PRODUCTION (`replay.equipmentOrigin` — fenetre de 200 ms ET distance de
// 1,5 m au moment ou F.0 a mesure ; la clause de distance a ete retiree par l'item F.1 le
// 2026-09-13, APRES cette mesure, ce que les comptes de ce lot refletent donc bien) : le
// recalculer ici en ferait une seconde ecriture, qui divergerait au premier correctif — et
// c est precisement ce correctif que le lot instruit. L instrument lit donc `origin` dans
// l artefact deja cuit, tel que le client le recoit.
//
// # LA JOINTURE, ET CE QUI LA REND EXACTE
//
// Les deux cotes viennent du MEME balayage : `grammar.ScanEquipmentPlacements` rend
// `(Life{Slot,Gen}, T0US, GlobalID)`, et `replay.buildEquipmentPlacements` publie la MEME
// liste projetee sur l axe de frames — `t0 = (T0US - originUS) / stepUS`, avec
// `originUS = premier paquet du chunk 1 + originMs·1000` et `stepUS = frameIntervalMs·1000`.
// La cle de jointure est donc `(t0, GlobalID)`, et les doublons de cle se departagent DANS
// L ORDRE (les deux cotes conservent l ordre du balayage). Le taux de jointure est publie :
// c est LE controle de la mesure, pas un detail d implementation.
//
// LECTURE SEULE. Voir `f0_103_contexte_research_test.go` pour les gardes et la commande.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/mappings"
)

// f0ArtPlacement est une pose PUBLIEE : ce que le client lit.
type f0ArtPlacement struct {
	T0     int    `json:"t0"`
	Family string `json:"family"`
	ID     string `json:"id"`
	Owner  int    `json:"owner"`
	Origin string `json:"origin"`
}

// f0ArtChange est un changement d equipement PUBLIE (`taken` / `spent`).
type f0ArtChange struct {
	T    int    `json:"t"`
	Slot uint32 `json:"slot"`
	Kind string `json:"kind"`
	R    int    `json:"r"`
	From int    `json:"from"`
	Gap  int    `json:"gap"`
}

// f0ArtLabel est une entree de la palette PUBLIEE par le document.
type f0ArtLabel struct {
	Family string `json:"family"`
}

// f0ArtTrack est une piste : un slot et la derniere frame de sa vie.
type f0ArtTrack struct {
	Slot     uint32 `json:"slot"`
	EndFrame int    `json:"endFrame"`
}

// f0Art est le sous-ensemble de l artefact dont ce lot a besoin.
type f0Art struct {
	SchemaVersion       int                   `json:"schemaVersion"`
	FrameCount          int                   `json:"frameCount"`
	OriginMs            *int64                `json:"originMs"`
	FrameIntervalMs     int64                 `json:"frameIntervalMs"`
	Tracks              []f0ArtTrack          `json:"tracks"`
	EquipmentPlacements []f0ArtPlacement      `json:"equipmentPlacements"`
	EquipmentChanges    []f0ArtChange         `json:"equipmentChanges"`
	AbilityLabels       map[string]f0ArtLabel `json:"abilityLabels"`
}

// f0LitArtefact charge l artefact d un film. Rend `false` quand il n existe pas : le film
// reste mesurable pour la question 1 (les references ne dependent d aucun artefact) et sort
// des questions 2 a 4, ce qui est dit au rapport plutot que masque.
func f0LitArtefact(t *testing.T, id string) (f0Art, bool) {
	t.Helper()
	dir := os.Getenv(f0ArtsEnv)
	if dir == "" {
		return f0Art{}, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		t.Logf("film %s : aucun artefact (%v) — hors des questions 2 a 4", id, err)
		return f0Art{}, false
	}
	var a f0Art
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("film %s : artefact indecodable : %v", id, err)
	}
	if a.FrameIntervalMs == 0 {
		a.FrameIntervalMs = 100
	}
	return a, true
}

// f0OrigineUS rend l origine de l axe de frames sur l horloge MOTEUR, et le pas en µs.
// `originMs` absent : l artefact ne porte pas de recalage, la jointure est impossible.
func f0OrigineUS(f f0Film, a f0Art) (origin, step uint64, ok bool) {
	if a.OriginMs == nil || a.FrameIntervalMs <= 0 {
		return 0, 0, false
	}
	return f.BaseUS + uint64(*a.OriginMs)*1000, uint64(a.FrameIntervalMs) * 1000, true
}

// f0Pose est une pose apres jointure : ce que le film en dit (cle de vie, instant moteur) ET
// ce que l artefact en publie (famille, origine, poseur).
type f0Pose struct {
	Cle    f0CleVie
	T0US   uint64
	T0     int
	ID     uint32
	Family string
	Origin string
	Owner  int
}

// f0Joint apparie les poses de l artefact aux poses du balayage sur `(t0, GlobalID)`. Rend les
// poses appariees, le nombre de poses d artefact non appariees et le nombre de poses de film
// non appariees — les trois se lisent ensemble.
func f0Joint(f f0Film, a f0Art) ([]f0Pose, int, int) {
	origin, step, ok := f0OrigineUS(f, a)
	if !ok {
		return nil, len(a.EquipmentPlacements), len(f.Places)
	}
	type cle struct {
		t0 int
		id uint32
	}
	seaux := map[cle][]EquipmentPlacement{}
	for _, p := range f.Places {
		t0 := f0Frame(p.T0US, origin, step)
		if t0 < 0 {
			t0 = 0
		}
		if t0 >= a.FrameCount && a.FrameCount > 0 {
			t0 = a.FrameCount - 1
		}
		seaux[cle{t0, p.GlobalID}] = append(seaux[cle{t0, p.GlobalID}], p)
	}
	var out []f0Pose
	orphelines := 0
	apparies := 0
	for _, ap := range a.EquipmentPlacements {
		var id uint32
		if _, err := fmt.Sscanf(ap.ID, "0x%08x", &id); err != nil {
			orphelines++
			continue
		}
		k := cle{ap.T0, id}
		v := seaux[k]
		if len(v) == 0 {
			orphelines++
			continue
		}
		p := v[0]
		seaux[k] = v[1:]
		apparies++
		out = append(out, f0Pose{
			Cle: f0CleVie{Index: uint64(p.Life.Slot), Gen: p.Life.Gen}, T0US: p.T0US,
			T0: ap.T0, ID: id, Family: ap.Family, Origin: ap.Origin, Owner: ap.Owner,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].T0US < out[j].T0US })
	return out, orphelines, len(f.Places) - apparies
}

// f0Frame projette un instant moteur sur l axe de frames du document — la MEME arithmetique
// que `replay.frameOf`, division entiere comprise.
func f0Frame(ts, origin, step uint64) int {
	if ts >= origin {
		return int((ts - origin) / step)
	}
	return -int((origin - ts + step - 1) / step)
}

// f0RangFamille rend la table rang -> famille applicable au film : celle que le DOCUMENT
// publie quand il en a une, sinon celle du MANIFESTE du titre.
//
// LE REPLI EST LICITE ET IL EST DIT. Huit artefacts sur soixante-quatre ne portent aucune
// table (mesure E0 n°3) : la palette du film n a pas ete classee. Les deux palettes du
// manifeste ont des rangs DISJOINTS — c est ce qui rend leur fusion sans ambiguite, et la
// fusion echoue bruyamment si un rang venait a figurer des deux cotes avec deux familles.
func f0RangFamille(t *testing.T, a f0Art) (map[int]string, string) {
	t.Helper()
	if len(a.AbilityLabels) > 0 {
		out := map[int]string{}
		for k, v := range a.AbilityLabels {
			var r int
			if _, err := fmt.Sscanf(k, "%d", &r); err == nil && v.Family != "" {
				out[r] = v.Family
			}
		}
		return out, "document"
	}
	return f0ManifesteRangs(t), "manifeste (document sans table de palette)"
}

// f0ManifesteRangs fusionne les tables rang -> famille des palettes du manifeste.
func f0ManifesteRangs(t *testing.T) map[int]string {
	t.Helper()
	path := os.Getenv(f0LabelsEnv)
	if path == "" {
		t.Skipf("instrument F.0 : definir %s (replay_labels.toml) — aucun nom de famille n est "+
			"ecrit dans ce fichier, la table vient du manifeste", f0LabelsEnv)
	}
	set, err := mappings.LoadReplayLabelsFromFile(path)
	if err != nil {
		t.Fatalf("manifeste de libelles : %v", err)
	}
	out := map[int]string{}
	for _, p := range set.AbilityPalettes() {
		for rang, fam := range p.Families {
			if prev, vu := out[rang]; vu && prev != fam {
				t.Fatalf("manifeste : le rang %d porte deux familles (%q et %q) — la fusion "+
					"des palettes n est plus sans ambiguite", rang, prev, fam)
			}
			out[rang] = fam
		}
	}
	return out
}

// f0ManifesteObjets rend la table GlobalID -> famille du manifeste (les objets `eqip`).
func f0ManifesteObjets(t *testing.T) map[uint32]string {
	t.Helper()
	path := os.Getenv(f0LabelsEnv)
	if path == "" {
		t.Skipf("instrument F.0 : definir %s (replay_labels.toml)", f0LabelsEnv)
	}
	set, err := mappings.LoadReplayLabelsFromFile(path)
	if err != nil {
		t.Fatalf("manifeste de libelles : %v", err)
	}
	return set.EquipmentObjects()
}

// f0FinDeVie rend, par slot, la DERNIERE frame de chacune de ses vies publiees. C est ce qui
// permet de reconnaitre les huit cas de D12 : une pose d appareil classee `deployed` alors
// qu elle tombe a l image EXACTE de la fin de vie de son poseur.
func f0FinDeVie(a f0Art) map[uint32][]int {
	out := map[uint32][]int{}
	for _, tr := range a.Tracks {
		out[tr.Slot] = append(out[tr.Slot], tr.EndFrame)
	}
	for s := range out {
		sort.Ints(out[s])
	}
	return out
}

// f0EstFinDeVie dit si la frame `t0` est exactement la derniere frame d une vie du slot.
func f0EstFinDeVie(fins map[uint32][]int, slot uint32, t0 int) bool {
	for _, f := range fins[slot] {
		if f == t0 {
			return true
		}
	}
	return false
}
