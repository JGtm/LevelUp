package replay

// e0_gachis_research_test.go — INSTRUMENT JETABLE de l'etape E0 du plan
// `.ai/PLAN_EQUIPEMENT_GACHIS_2026-09-09.md`. Il MESURE le canal `equipmentChanges` sur le
// parc d'artefacts cuits, il ne publie rien et il est SUPPRIME des que ses chiffres sont au
// document de reference (patron `filmdec/r11_*_research_test.go`).
//
// CE QUI EST MESURE, ET DANS L'ORDRE DU PLAN :
//
//	E0.1  volumes par `Kind`, slots distincts, taux d'emissions MANQUEES lu dans la couverture
//	E0.2  jointure rang de palette -> famille par `abilityLabels` ; taux de rangs NON nommes
//	E0.3  jointure `Slot` -> joueur par le REGISTRE D'IDENTITE publie (`identity.bipedSlots`)
//	E0.4  identite `taken ~ utilise + lache + garde`, par joueur et par famille
//
// LA JOINTURE D'IDENTITE PASSE PAR L'ARTEFACT, PAS PAR LE FILM. `buildPlayers` / `indexBySlot`
// travaillent sur des positions de film ; ici l'entree est un artefact CUIT, et le registre y
// est deja publie avec ses BORNES (`identity.bipedSlots[].link.from/to`). Un slot est recycle a
// chaque reapparition : une ligne par VIE, jamais une par slot — c'est ce couple de bornes qui
// interdit de crediter le premier occupant (defaut P0-2).
//
// DEUX DEFINITIONS DE « UTILISE » (decision P2 du plan, §1 bis du document de reference) :
// un equipement d'ACTIVATION sert quand il est ACTIVE (camouflage et surbouclier ->
// `equipmentEpisodes`, grappin -> `grappleLines`, translocateur -> `translocations`,
// propulseur -> `abilityImpulses`) ; un DEPLOYABLE sert quand il est POSE
// (`equipmentPlacements` `origin: deployed`, poseur = la vie). Le REPULSEUR n'a AUCUN canal
// d'usage — negatif mesure, 9 canaux fouilles : ses fenetres ne peuvent donc jamais etre
// classees « utilise », et l'instrument le dit au lieu de le taire.
//
// COMMENT LA FERMETURE EST TESTEE, ET POURQUOI ELLE N'EST PAS CIRCULAIRE. Chaque `taken`
// ouvre une FENETRE qui se ferme au `taken` ou au `spent` suivant de la MEME vie, a defaut a
// la fin de la vie. Une fenetre porte UNE issue :
//
//	utilise       >= 1 evenement du canal d'usage de la famille, sur la vie, dans la fenetre
//	lache         sinon, >= 1 pose `dropped` de la famille attribuee a la vie dans la fenetre
//	garde         sinon, et la fenetre ne se ferme PAS par un `spent`
//	NON EXPLIQUE  sinon : le film ANNONCE la consommation, aucun canal d'usage ne la voit
//
// Les trois premieres sont les trois issues de la decision P1 ; la quatrieme est l'ECART, et
// c'est le seul chiffre que la decision de sortie du plan interroge. Compter « lache » sur les
// poses BRUTES gonflerait le total (une mort libere aussi l'equipement de reapparition, qui
// n'a jamais ete `taken`) : ces poses hors fenetre sont comptees a part, jamais dans l'identite.
//
// UNE POSE EST UNE CHARGE, PAS UN OBJET (piege d'unite n° 1 du document de reference) : un
// capteur pris une fois et lance quatre fois donne 4 poses pour 1 objet, et le mur en publie
// DEUX par pose (l'appareil et ses panneaux). Classer PAR FENETRE — et non par evenement —
// dedoublonne les deux d'un coup, sans table d'identifiants a maintenir.
//
// LECTURE SEULE, AUCUNE BASE. Les artefacts sont des FICHIERS ; une passe de recuisson peut
// tenir les DuckDB du parc, et cet instrument n'en ouvre aucune.
//
// USAGE (depuis apps/go-api) :
//
//	EQ_GACHIS_ART=<depot>/data/cache/replays/halo_infinite \
//	  go test ./internal/analysis/replay/ -run '^TestE0EquipementGachisResearch$' -v
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// eqE0ArtEnv porte le repertoire des artefacts cuits. Defaut : le chemin du depot.
const eqE0ArtEnv = "EQ_GACHIS_ART"

// eqE0MinArtefacts est le plancher du plan (E0.1 : « sur au moins 20 artefacts locaux »).
const eqE0MinArtefacts = 20

// Familles de POWER-UP. Le manifeste NOMME les rangs 8 et 9 sans leur donner de famille
// (`replay_labels.toml` : leur objet de manifeste n'est pas l'emplacement de capacite). Cet
// instrument leur en donne une — LE MEME VOCABULAIRE que les poses (`powerup_camo`,
// `powerup_overshield`) — pour pouvoir joindre les deux canaux ; il ne modifie pas le manifeste.
const (
	eqE0FamCamo       = "powerup_camo"
	eqE0FamOvershield = "powerup_overshield"
)

// eqE0Deployables : les familles dont « utilise » veut dire POSE (famille B du §1 bis).
var eqE0Deployables = map[string]bool{
	"wall": true, "sensor": true, "shroud_screen": true,
	"threat_seeker": true, "repair_field": true, "translocator_beacon": true,
}

// eqE0SansCanal : les familles dont AUCUN canal ne mesure l'usage. Le repulseur, et lui seul
// (negatif mesure des rapports R8/R9/R11).
var eqE0SansCanal = map[string]bool{"repulsor": true}

// eqE0Issue nomme l'issue d'une fenetre de `taken`.
type eqE0Issue int

const (
	eqE0Utilise eqE0Issue = iota
	eqE0Lache
	eqE0Garde
	eqE0NonExplique
)

// eqE0Compte porte les quatre issues d'un couple (joueur, famille).
type eqE0Compte struct{ Taken, Utilise, Lache, Garde, NonExplique int }

// eqE0Totaux agrege tout le corpus.
type eqE0Totaux struct {
	Artefacts, AvecChangements, SansIdentite, SansPalette int
	Taken, Spent, Autres, SlotsDistincts                  int
	Publies, Manquees, SautsCompteur, Reapparitions       int
	Recuperees, PremiereHorsNorme, Repetitions            int
	RangsLus, RangsNommes                                 int
	MuetsFilmSansPalette, MuetsRangConnuAilleurs          int
	MuetsRangNonEtabli                                    int
	RangsMuets                                            map[int]int
	EvtsRattaches, EvtsNonRattaches                       int
	SlotsRattaches, SlotsNonRattaches                     int
	ParFamille                                            map[string]*eqE0Compte
	ParCouple                                             map[string]*eqE0Compte
	PosesHorsFenetre, UsagesHorsFenetre                   int
	// PosesParFamille : deployees, lachees, origine inconnue.
	PosesParFamille map[string][3]int
}

func TestE0EquipementGachisResearch(t *testing.T) {
	dir := eqE0Dir(t)
	files := eqE0Fichiers(t, dir)
	if len(files) < eqE0MinArtefacts {
		t.Skipf("%d artefacts dans %s : le plan en exige %d — instrument saute",
			len(files), dir, eqE0MinArtefacts)
	}
	noms, familles := eqE0Palette(t)
	tot := &eqE0Totaux{
		RangsMuets: map[int]int{}, ParFamille: map[string]*eqE0Compte{},
		ParCouple: map[string]*eqE0Compte{}, PosesParFamille: map[string][3]int{},
	}
	for _, f := range files {
		doc := eqE0Lire(t, f)
		eqE0UnArtefact(doc, noms, familles, tot)
	}
	t.Logf("CORPUS : %d artefacts lus dans %s", tot.Artefacts, dir)
	eqE0RapportVolumes(t, tot)
	eqE0RapportRangs(t, tot)
	eqE0RapportIdentite(t, tot)
	eqE0RapportIssues(t, tot)
}

// eqE0Dir rend le repertoire des artefacts, ou saute l'instrument.
func eqE0Dir(t *testing.T) string {
	t.Helper()
	if v := os.Getenv(eqE0ArtEnv); v != "" {
		return v
	}
	dir := filepath.Join(repoRootForTest(t), "data", "cache", "replays", "halo_infinite")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("%s absent et %s introuvable : instrument saute", eqE0ArtEnv, dir)
	}
	return dir
}

// eqE0Fichiers liste les artefacts, DERIVES EXCLUS (ce ne sont pas des documents de rejeu).
func eqE0Fichiers(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("%s illisible (%v) : instrument saute", dir, err)
	}
	var out []string
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".json") || strings.Contains(n, "derived") {
			continue
		}
		out = append(out, filepath.Join(dir, n))
	}
	sort.Strings(out)
	return out
}

// eqE0Lire deserialise un artefact.
func eqE0Lire(t *testing.T, path string) ReplayDocument {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact %s illisible : %v", path, err)
	}
	var doc ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("artefact %s indeserialisable : %v", path, err)
	}
	return doc
}

// eqE0Palette lit le manifeste du titre — LA MEME table que la production — et rend le nom EN
// de chaque rang etabli, plus sa famille. Les rangs des deux palettes sont disjoints
// (`markers` disjoints, invariant du loader) : une table unique est donc sure.
func eqE0Palette(t *testing.T) (map[int]string, map[int]string) {
	t.Helper()
	noms, familles := map[int]string{}, map[int]string{}
	for _, p := range goldenReplayLabels(t).AbilityPalettes() {
		for r, lab := range p.Ranks {
			if prev, dup := noms[r]; dup && prev != lab.En {
				t.Fatalf("rang %d nomme %q et %q : les palettes ne sont pas disjointes", r, prev, lab.En)
			}
			noms[r] = lab.En
			switch fam := p.Families[r]; {
			case fam != "":
				familles[r] = fam
			case lab.En == "Active Camouflage":
				familles[r] = eqE0FamCamo
			case lab.En == "Overshield":
				familles[r] = eqE0FamOvershield
			}
		}
	}
	return noms, familles
}

// eqE0UnArtefact mesure UN artefact et verse dans les totaux.
func eqE0UnArtefact(doc ReplayDocument, noms, familles map[int]string, tot *eqE0Totaux) {
	tot.Artefacts++
	if len(doc.AbilityLabels) == 0 {
		tot.SansPalette++
	}
	if doc.Identity == nil || len(doc.Identity.BipedSlots) == 0 {
		tot.SansIdentite++
	}
	eqE0Volumes(doc, tot)
	if len(doc.EquipmentChanges) == 0 {
		return
	}
	tot.AvecChangements++
	eqE0Rangs(doc, noms, tot)
	vies := eqE0Vies(doc)
	eqE0Rattachement(doc, vies, tot)
	eqE0Issues(doc, familles, vies, tot)
}

// eqE0Volumes joue E0.1 : les volumes par `Kind`, les slots distincts, et le TEMOIN DE
// COMPLETUDE de la couverture — le seul calque du rejeu qui sache dire ce qu'il a MANQUE.
func eqE0Volumes(doc ReplayDocument, tot *eqE0Totaux) {
	slots := map[uint32]bool{}
	for _, c := range doc.EquipmentChanges {
		slots[c.Slot] = true
		switch c.Kind {
		case EquipmentTaken:
			tot.Taken++
		case EquipmentSpent:
			tot.Spent++
		default:
			tot.Autres++
		}
	}
	tot.SlotsDistincts += len(slots)
	for _, p := range doc.EquipmentPlacements {
		c := tot.PosesParFamille[p.Family]
		switch p.Origin {
		case "deployed":
			c[0]++
		case "dropped":
			c[1]++
		default:
			c[2]++
		}
		tot.PosesParFamille[p.Family] = c
	}
	if doc.Coverage == nil || doc.Coverage.EquipmentChanges == nil {
		return
	}
	cov := doc.Coverage.EquipmentChanges
	tot.Publies += cov.Published
	tot.Manquees += cov.MissedEstimate
	tot.SautsCompteur += cov.CounterJumps
	tot.Reapparitions += cov.Spawned
	tot.Recuperees += cov.Recovered
	tot.PremiereHorsNorme += cov.LivesFirstOffSpec
	tot.Repetitions += cov.Repeats
}

// eqE0Rangs joue E0.2 : le rang de palette est-il NOMME ? Sur un `taken` c'est `R` ; sur un
// `spent`, `R` vaut la sentinelle et c'est `From` qui porte ce qui vient d'etre consomme.
func eqE0Rangs(doc ReplayDocument, noms map[int]string, tot *eqE0Totaux) {
	for _, c := range doc.EquipmentChanges {
		r := c.R
		if c.Kind == EquipmentSpent {
			r = c.From
		}
		if r == NoAbilityRank {
			continue
		}
		tot.RangsLus++
		// LA TABLE DU DOCUMENT FAIT FOI, pas le manifeste : elle est PROPRE A LA PALETTE du
		// match, et un film dont la palette n'est pas classee ne recoit AUCUN nom. Juger sur
		// le manifeste nommerait des rangs que l'ecran, lui, affichera nus.
		if _, ok := doc.AbilityLabels[strconv.Itoa(r)]; ok {
			tot.RangsNommes++
			continue
		}
		tot.RangsMuets[r]++
		// DEUX CAUSES, ET ELLES N'APPELLENT PAS LE MEME CORRECTIF. Ou le FILM n'a pas de
		// palette classee (aucun rang n'y est nomme, la table est absente en bloc) ; ou le
		// RANG lui-meme n'a pas de nom etabli dans le manifeste. La premiere se repare en
		// classant la palette du film, la seconde en inversant un identifiant de chaine.
		if len(doc.AbilityLabels) == 0 {
			tot.MuetsFilmSansPalette++
			continue
		}
		if _, connu := noms[r]; connu {
			tot.MuetsRangConnuAilleurs++
			continue
		}
		tot.MuetsRangNonEtabli++
	}
}

// eqE0Vie est UNE occupation de slot, bornee — donc une VIE.
type eqE0Vie struct {
	XUID     string
	From, To int
}

// eqE0Vies indexe le registre d'identite PUBLIE par slot. Les lignes sans xuid entrent
// quand meme : les jeter ferait de la couverture un compte de RESCAPES.
func eqE0Vies(doc ReplayDocument) map[uint32][]eqE0Vie {
	out := map[uint32][]eqE0Vie{}
	if doc.Identity == nil {
		return out
	}
	for _, b := range doc.Identity.BipedSlots {
		out[b.Slot] = append(out[b.Slot], eqE0Vie{XUID: b.XUID, From: b.Link.From, To: b.Link.To})
	}
	for s := range out {
		v := out[s]
		sort.SliceStable(v, func(i, j int) bool { return v[i].From < v[j].From })
		out[s] = v
	}
	return out
}

// eqE0Joueur rend le xuid de la vie qui occupe `slot` a la frame `t`, et si elle a ete nommee.
func eqE0Joueur(vies map[uint32][]eqE0Vie, slot uint32, t int) (eqE0Vie, bool) {
	for _, v := range vies[slot] {
		if t >= v.From && t <= v.To {
			return v, v.XUID != ""
		}
	}
	return eqE0Vie{}, false
}

// eqE0Rattachement joue E0.3, aux DEUX grains que le plan distingue : l'evenement et le slot.
func eqE0Rattachement(doc ReplayDocument, vies map[uint32][]eqE0Vie, tot *eqE0Totaux) {
	parSlot := map[uint32]bool{}
	for _, c := range doc.EquipmentChanges {
		_, ok := eqE0Joueur(vies, c.Slot, c.T)
		if ok {
			tot.EvtsRattaches++
			parSlot[c.Slot] = true
			continue
		}
		tot.EvtsNonRattaches++
		if _, deja := parSlot[c.Slot]; !deja {
			parSlot[c.Slot] = false
		}
	}
	for _, ok := range parSlot {
		if ok {
			tot.SlotsRattaches++
			continue
		}
		tot.SlotsNonRattaches++
	}
}

// eqE0Fenetre est une fenetre de `taken` : ce qui a ete pris, par qui, et jusqu'a quand.
type eqE0Fenetre struct {
	Slot           uint32
	XUID, Famille  string
	Debut, Fin     int
	FermeeParSpent bool
}

// eqE0Issues joue E0.4 : les fenetres, leur issue, et ce qui tombe HORS de toute fenetre.
func eqE0Issues(
	doc ReplayDocument, familles map[int]string, vies map[uint32][]eqE0Vie, tot *eqE0Totaux,
) {
	fen := eqE0Fenetres(doc, familles, vies)
	usages := eqE0Usages(doc)
	poses := eqE0Poses(doc)
	usesVus, posesVues := map[int]bool{}, map[int]bool{}
	for _, f := range fen {
		eqE0Ajouter(tot, f, eqE0IssueDe(f, usages, poses, usesVus, posesVues))
	}
	tot.UsagesHorsFenetre += len(usages) - len(usesVus)
	tot.PosesHorsFenetre += len(poses) - len(posesVues)
}

// eqE0IssueDe classe UNE fenetre, et marque au passage les evenements consommes — c'est ce qui
// permet de compter ceux qui ne le sont par aucune fenetre.
func eqE0IssueDe(
	f eqE0Fenetre, usages, poses []eqE0Evt, usesVus, posesVues map[int]bool,
) eqE0Issue {
	trouve := false
	for i, u := range usages {
		if u.Slot == f.Slot && u.Famille == f.Famille && u.T >= f.Debut && u.T <= f.Fin {
			usesVus[i], trouve = true, true
		}
	}
	if trouve {
		return eqE0Utilise
	}
	for i, p := range poses {
		if p.Slot == f.Slot && p.Famille == f.Famille && p.T >= f.Debut && p.T <= f.Fin {
			posesVues[i], trouve = true, true
		}
	}
	if trouve {
		return eqE0Lache
	}
	if !f.FermeeParSpent {
		return eqE0Garde
	}
	return eqE0NonExplique
}

// eqE0Fenetres ouvre une fenetre par `taken` NOMME (rang connu, famille connue, vie nommee) et
// la ferme au changement suivant de la MEME vie, a defaut a la fin de la vie.
func eqE0Fenetres(
	doc ReplayDocument, familles map[int]string, vies map[uint32][]eqE0Vie,
) []eqE0Fenetre {
	parSlot := map[uint32][]EquipmentChange{}
	for _, c := range doc.EquipmentChanges {
		parSlot[c.Slot] = append(parSlot[c.Slot], c)
	}
	var out []eqE0Fenetre
	for slot, cs := range parSlot {
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].T < cs[j].T })
		for i, c := range cs {
			fam := familles[c.R]
			if c.Kind != EquipmentTaken || fam == "" {
				continue
			}
			v, nomme := eqE0Joueur(vies, slot, c.T)
			if !nomme {
				continue
			}
			fin, spent := v.To, false
			if i+1 < len(cs) && cs[i+1].T <= v.To {
				fin, spent = cs[i+1].T, cs[i+1].Kind == EquipmentSpent
			}
			out = append(out, eqE0Fenetre{Slot: slot, XUID: v.XUID, Famille: fam,
				Debut: c.T, Fin: fin, FermeeParSpent: spent})
		}
	}
	return out
}

// eqE0Evt est un evenement d'usage ou de lacher, ramene au triplet qui sert a la jointure.
type eqE0Evt struct {
	Slot    uint32
	Famille string
	T       int
}

// eqE0Usages rassemble les canaux d'USAGE des deux familles du §1 bis : activation pour le
// camouflage, le surbouclier, le grappin, le translocateur et le propulseur ; POSE pour les
// deployables. Le repulseur n'y figure pas — aucun canal ne le mesure.
func eqE0Usages(doc ReplayDocument) []eqE0Evt {
	var out []eqE0Evt
	for _, e := range doc.EquipmentEpisodes {
		fam := eqE0FamCamo
		if e.Fam == EquipFamilyOvershield {
			fam = eqE0FamOvershield
		}
		out = append(out, eqE0Evt{Slot: e.Slot, Famille: fam, T: e.T0})
	}
	for _, g := range doc.GrappleLines {
		out = append(out, eqE0Evt{Slot: g.Slot, Famille: "grapple", T: g.T0})
	}
	for _, tr := range doc.Translocations {
		out = append(out, eqE0Evt{Slot: tr.Slot, Famille: "translocator_beacon", T: tr.T})
	}
	for _, im := range doc.AbilityImpulses {
		out = append(out, eqE0Evt{Slot: im.Slot, Famille: im.Family, T: im.T})
	}
	for _, p := range doc.EquipmentPlacements {
		if p.Origin == "deployed" && p.Owner >= 0 && eqE0Deployables[p.Family] {
			out = append(out, eqE0Evt{Slot: uint32(p.Owner), Famille: p.Family, T: p.T0})
		}
	}
	return out
}

// eqE0Poses rassemble les LACHERS a la mort — et eux seuls.
func eqE0Poses(doc ReplayDocument) []eqE0Evt {
	var out []eqE0Evt
	for _, p := range doc.EquipmentPlacements {
		if p.Origin == "dropped" && p.Owner >= 0 {
			out = append(out, eqE0Evt{Slot: uint32(p.Owner), Famille: p.Family, T: p.T0})
		}
	}
	return out
}

// eqE0Ajouter verse une issue dans les DEUX grains : la famille (le tableau du rapport) et le
// couple (joueur, famille) — le grain que la decision de sortie du plan interroge. Le couple
// porte le xuid du joueur, jamais le slot : un slot est une vie, un joueur en a plusieurs.
func eqE0Ajouter(tot *eqE0Totaux, f eqE0Fenetre, issue eqE0Issue) {
	if tot.ParFamille[f.Famille] == nil {
		tot.ParFamille[f.Famille] = &eqE0Compte{}
	}
	cle := f.XUID + "|" + f.Famille
	if tot.ParCouple[cle] == nil {
		tot.ParCouple[cle] = &eqE0Compte{}
	}
	for _, c := range []*eqE0Compte{tot.ParFamille[f.Famille], tot.ParCouple[cle]} {
		c.Taken++
		switch issue {
		case eqE0Utilise:
			c.Utilise++
		case eqE0Lache:
			c.Lache++
		case eqE0Garde:
			c.Garde++
		default:
			c.NonExplique++
		}
	}
}

// eqE0RapportVolumes publie E0.1.
func eqE0RapportVolumes(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	t.Log("== E0.1 VOLUMES DU CANAL `equipmentChanges` ==")
	t.Logf("  artefacts portant des changements %s · sans section `identity` %d"+
		" · sans table `abilityLabels` %d", eqE0Pct(tot.AvecChangements, tot.Artefacts),
		tot.SansIdentite, tot.SansPalette)
	total := tot.Taken + tot.Spent + tot.Autres
	t.Logf("  changements publies %d · `taken` %s · `spent` %s · AUTRE Kind %d",
		total, eqE0Pct(tot.Taken, total), eqE0Pct(tot.Spent, total), tot.Autres)
	t.Logf("  slots distincts porteurs (somme sur les artefacts, un slot = une VIE) %d",
		tot.SlotsDistincts)
	t.Log("== E0.1 TEMOIN DE COMPLETUDE (couverture du canal — le compteur de rotation) ==")
	t.Logf("  emissions MANQUEES %d sur %d publiees + manquees = TAUX D'EMISSIONS MANQUEES %s",
		tot.Manquees, tot.Publies+tot.Manquees, eqE0Pct(tot.Manquees, tot.Publies+tot.Manquees))
	t.Logf("  sauts de compteur %d · reapparitions ecartees %d · recuperees (schema 38) %d"+
		" · premiere emission hors norme %d · repetitions %d", tot.SautsCompteur,
		tot.Reapparitions, tot.Recuperees, tot.PremiereHorsNorme, tot.Repetitions)
}

// eqE0RapportRangs publie E0.2.
func eqE0RapportRangs(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	muets := tot.RangsLus - tot.RangsNommes
	t.Log("== E0.2 JOINTURE RANG DE PALETTE -> FAMILLE (`abilityLabels` du document) ==")
	t.Logf("  rangs lus %d · NOMMES %s · NON NOMMES %s", tot.RangsLus,
		eqE0Pct(tot.RangsNommes, tot.RangsLus), eqE0Pct(muets, tot.RangsLus))
	t.Logf("  causes du silence : film SANS palette classee %s · rang connu du manifeste mais"+
		" absent de la table du film %s · rang NON ETABLI nulle part %s",
		eqE0Pct(tot.MuetsFilmSansPalette, muets), eqE0Pct(tot.MuetsRangConnuAilleurs, muets),
		eqE0Pct(tot.MuetsRangNonEtabli, muets))
	rangs := make([]int, 0, len(tot.RangsMuets))
	for r := range tot.RangsMuets {
		rangs = append(rangs, r)
	}
	sort.Slice(rangs, func(i, j int) bool { return tot.RangsMuets[rangs[i]] > tot.RangsMuets[rangs[j]] })
	for i, r := range rangs {
		if i == 12 {
			t.Logf("  … %d rangs muets distincts au total", len(rangs))
			break
		}
		t.Logf("  rang muet %3d : %d lectures", r, tot.RangsMuets[r])
	}
}

// eqE0RapportIdentite publie E0.3.
func eqE0RapportIdentite(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	evts := tot.EvtsRattaches + tot.EvtsNonRattaches
	slots := tot.SlotsRattaches + tot.SlotsNonRattaches
	t.Log("== E0.3 JOINTURE `Slot` -> JOUEUR (registre d'identite publie, bornes par VIE) ==")
	t.Logf("  evenements rattaches %s · NON RATTACHES %s", eqE0Pct(tot.EvtsRattaches, evts),
		eqE0Pct(tot.EvtsNonRattaches, evts))
	t.Logf("  slots porteurs rattaches %s · NON RATTACHES %s",
		eqE0Pct(tot.SlotsRattaches, slots), eqE0Pct(tot.SlotsNonRattaches, slots))
}

// eqE0RapportIssues publie E0.4 : le tableau par famille, puis l'ecart par couple
// (joueur, famille) — mediane et pire cas, les deux chiffres de la decision de sortie.
func eqE0RapportIssues(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	t.Log("== E0.4 IDENTITE `taken ~ utilise + lache + garde`, PAR FAMILLE ==")
	fams := make([]string, 0, len(tot.ParFamille))
	for f := range tot.ParFamille {
		fams = append(fams, f)
	}
	sort.Slice(fams, func(i, j int) bool {
		return tot.ParFamille[fams[i]].Taken > tot.ParFamille[fams[j]].Taken
	})
	var glob eqE0Compte
	for _, f := range fams {
		c := tot.ParFamille[f]
		glob.Taken, glob.Utilise = glob.Taken+c.Taken, glob.Utilise+c.Utilise
		glob.Lache, glob.Garde = glob.Lache+c.Lache, glob.Garde+c.Garde
		glob.NonExplique += c.NonExplique
		note := ""
		if eqE0SansCanal[f] {
			note = "  <- AUCUN CANAL D'USAGE (negatif mesure) : « utilise » y est impossible"
		}
		t.Logf("  %-20s pris %4d · utilise %4d · lache %4d · garde %4d · NON EXPLIQUE %s%s",
			f, c.Taken, c.Utilise, c.Lache, c.Garde, eqE0Pct(c.NonExplique, c.Taken), note)
	}
	t.Logf("  %-20s pris %4d · utilise %4d · lache %4d · garde %4d · NON EXPLIQUE %s",
		"TOUTES FAMILLES", glob.Taken, glob.Utilise, glob.Lache, glob.Garde,
		eqE0Pct(glob.NonExplique, glob.Taken))
	t.Logf("  hors de toute fenetre de `taken` : usages %d · lachers %d — equipement de"+
		" REAPPARITION (jamais `taken`) et manques du canal, comptes a part, JAMAIS dans"+
		" l'identite", tot.UsagesHorsFenetre, tot.PosesHorsFenetre)
	eqE0RapportEcart(t, tot)
	eqE0RapportPosesBrutes(t, tot)
}

// eqE0RapportPosesBrutes publie le RECENSEMENT BRUT des poses par famille et par origine — le
// chiffre que la decouverte ouverte du plan (§6 : capteur 49 deploiements pour 302 lachers,
// mur 295/251) demande de rafraichir. Il est publie, PAS instruit : ce lot ne l'enquete pas.
func eqE0RapportPosesBrutes(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	t.Log("== RECENSEMENT BRUT DES POSES (rapporte au §6 du plan, non instruit) ==")
	fams := make([]string, 0, len(tot.PosesParFamille))
	for f := range tot.PosesParFamille {
		fams = append(fams, f)
	}
	sort.Slice(fams, func(i, j int) bool {
		a, b := tot.PosesParFamille[fams[i]], tot.PosesParFamille[fams[j]]
		return a[0]+a[1]+a[2] > b[0]+b[1]+b[2]
	})
	var tt [3]int
	for _, f := range fams {
		p := tot.PosesParFamille[f]
		tt[0], tt[1], tt[2] = tt[0]+p[0], tt[1]+p[1], tt[2]+p[2]
		t.Logf("  %-20s deployees %5d · lachees %5d · origine inconnue %5d", f, p[0], p[1], p[2])
	}
	t.Logf("  %-20s deployees %5d · lachees %5d · origine inconnue %5d — RESERVE DE COUVERTURE"+
		" des poses %s", "TOUTES FAMILLES", tt[0], tt[1], tt[2],
		eqE0Pct(tt[2], tt[0]+tt[1]+tt[2]))
}

// eqE0RapportEcart publie la mediane et le pire cas de l'ecart par couple (joueur, famille).
// L'ECART EST LA PART NON EXPLIQUEE : la fenetre se ferme par un `spent` — le film ANNONCE la
// consommation — et aucun canal d'usage ne la voit. C'est le seul ecart non circulaire : les
// trois autres issues partitionnent les `taken` par construction.
func eqE0RapportEcart(t *testing.T, tot *eqE0Totaux) {
	t.Helper()
	type paire struct {
		cle   string
		ecart float64
	}
	var ps []paire
	for cle, c := range tot.ParCouple {
		if c.Taken == 0 {
			continue
		}
		ps = append(ps, paire{cle, float64(c.NonExplique) / float64(c.Taken)})
	}
	if len(ps) == 0 {
		t.Log("  ECART : aucun couple (joueur, famille) mesurable")
		return
	}
	// TRI TOTAL, jamais partiel : a ecart egal, le plus grand denominateur d'abord, puis la
	// cle. Sans ce departage, deux executions de l'instrument nomment deux « pires cas »
	// differents (l'ordre d'iteration d'une map Go n'est pas reproductible) et le chiffre
	// consigne au rapport ne se retrouve plus.
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].ecart != ps[j].ecart {
			return ps[i].ecart < ps[j].ecart
		}
		if a, b := tot.ParCouple[ps[i].cle].Taken, tot.ParCouple[ps[j].cle].Taken; a != b {
			return a < b
		}
		return ps[i].cle < ps[j].cle
	})
	med := ps[len(ps)/2].ecart
	if len(ps)%2 == 0 {
		med = (ps[len(ps)/2-1].ecart + ps[len(ps)/2].ecart) / 2
	}
	pire := ps[len(ps)-1]
	t.Logf("  ECART sur %d couples (joueur, famille) · MEDIANE %.2f %% · PIRE CAS %.2f %% (%s,"+
		" %d pris dont %d non expliques)", len(ps), 100*med, 100*pire.ecart,
		pire.cle, tot.ParCouple[pire.cle].Taken, tot.ParCouple[pire.cle].NonExplique)
	// LE PIRE CAS D'UN COUPLE A UNE SEULE PRISE VAUT 0 % OU 100 %, jamais autre chose : le
	// publier seul donnerait un chiffre de bruit. Le meme pire cas sur un denominateur
	// substantiel dit ce que le modele coute vraiment.
	for i := len(ps) - 1; i >= 0; i-- {
		c := tot.ParCouple[ps[i].cle]
		if c.Taken < 5 {
			continue
		}
		t.Logf("  PIRE CAS a >= 5 prises : %.2f %% (%s, %d pris dont %d non expliques)",
			100*ps[i].ecart, ps[i].cle, c.Taken, c.NonExplique)
		return
	}
}

// eqE0Pct rend un pourcentage lisible avec son denominateur.
func eqE0Pct(n, d int) string {
	if d == 0 {
		return fmt.Sprintf("%d/0 (non calculable)", n)
	}
	return fmt.Sprintf("%d/%d (%.2f %%)", n, d, 100*float64(n)/float64(d))
}
