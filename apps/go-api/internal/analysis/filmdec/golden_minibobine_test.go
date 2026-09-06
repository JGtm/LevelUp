package filmdec

// golden_minibobine_test.go — LE GOLDEN INCONDITIONNEL DES FAMILLES DE BALAYAGE
// (lot E, item E.6 du PLAN_V2_REJEU_FILM, 2026-09-06).
//
// # LE TROU QU'IL BOUCHE
//
// L'audit du 2026-09-05 (constat F3) l'a etabli : sur ~34 familles de balayage utilisees en
// production, QUATRE seulement etaient confrontees a des octets reels en CI — par la mini-bobine
// du rejeu (`analysis/replay/testdata/minifilm_000d5950`), qui n'a NI chunk_00 NI continuite de
// paquets. Les POINTS D'ENTREE de famille (`ScanCamoStates`, `ScanZoomEvents`,
// `ScanBipedPickups`, `ScanInventoryDeltas`, ...) n'etaient appeles par AUCUN test de `filmdec`,
// garde ou non ; seules les enveloppes `ScanFilm*(dir)` l'etaient, depuis des instruments sous
// garde de variable d'environnement, donc jamais en CI.
//
// # POURQUOI CETTE BOBINE-CI, ET PAS CELLE DU REJEU
//
// La mini-bobine de `killsource` est un PREFIXE CONTIGU du film 000d5950 — chunks 00 a 05, en-tete
// compris, plus le chunk highlight — donc elle porte le REGISTRE (chunk_00) et la continuite que
// le decodeur exige pour construire son monde par accumulation. Celle du rejeu est faite de
// paquets choisis : elle n'y decode aucun record de canal delta. Mesure a l'appui, cette
// bobine-ci rend 28 005 records delta, 17 slots bipedes et une population non vide dans 25
// familles sur 30.
//
// # CE QUE LE GOLDEN FIGE, ET SOUS QUELLE FORME
//
// Une ligne par famille : `nom <TAB> compte <TAB> digest <TAB> premier`.
//
//   - `compte` est la POPULATION (ou -1 quand la famille rend une erreur, cf. ci-dessous) ;
//   - `digest` est le sha256 d'un rendu `%+v` de TOUT le resultat : c'est lui qui attrape une
//     valeur qui bouge sans que le compte change (une largeur inversee, typiquement) ;
//   - `premier` est UNE VALEUR NOMMEE LISIBLE — le premier element, champs nommes, tronque.
//     Sans elle, un rouge se lirait « le digest a change » et il faudrait un outil pour savoir
//     quoi. Avec elle, l'instant, le slot et la valeur du premier evenement sont dans le diff.
//
// UNE POPULATION VIDE EST UNE INFORMATION, PAS UN ECHEC. Cinq familles rendent 0 sur ce prefixe
// (le film n'en porte pas), et deux rendent une ERREUR d'etat documentee (« aucun slot
// d'archetype ti=11 / ti=40 »). Le golden fige les deux cas TELS QUELS : si un jour l'une se
// remplit, le golden rougit et c'est exactement ce qu'on veut savoir. Le detail famille par
// famille est au journal du lot (`.ai/V7.5/v2/LOT_E.md`).
//
// # PAS DE SKIP
//
// La bobine est VERSIONNEE : son absence est une panne du depot, pas une condition d'execution.
// Le test `t.Fatal` — c'est la lecon du 2026-08-02, ou un `t.Skip` avait laisse passer une derive
// du decodeur sur quatre films en annoncant « golden vert ».
//
// # REGENERATION — UNE PORTE QUI NE SERT QU'A CE GOLDEN
//
//	go test ./internal/analysis/filmdec/ -run GoldenMiniBobineFamilles -update-golden-familles
//
// Un golden ne s'edite JAMAIS a la main. Il ne se regenere qu'apres un changement de decodage
// DECLARE, et le diff des comptes se relit dans le journal du lot.
//
// LE DRAPEAU EST DEDIE, ET IL L'EST DEPUIS LE 2026-09-06 (correction C5 de la revue E-R1). Ce
// golden partageait le `-update` du corpus de graines du fuzz (`fuzz_records_test.go`) : la
// commande `go test ./internal/analysis/filmdec/ -update` SANS `-run`, qui est celle qu'on tape
// pour regenerer les graines, le reecrivait au passage avec ce que le decodeur rendait a cet
// instant — et le paquet repondait `ok`. Mesure a l'appui : `br.Skip(2)` -> `br.Skip(3)` dans
// `readZoomRef` (une largeur de generation qu'aucun autre test n'epingle) suffisait a le faire
// partir en silence. Une porte de regeneration nomme desormais CE qu'elle regenere.

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
)

// bobineFamilles est la mini-bobine versionnee de `killsource`, vue depuis `filmdec`. Elle n'est
// pas dupliquee ici : deux verites binaires du meme film seraient deux verites a maintenir.
const bobineFamilles = "../../games/halo_infinite/film/killsource/testdata/minibobine_000d5950"

// bobineChunks est le nombre de chunks de replication de la bobine (chunk_01 a chunk_06).
const bobineChunks = 6

// goldenFamillesPath est la sortie figee.
const goldenFamillesPath = "testdata/golden_minibobine_familles.tsv"

// updateGoldenFamilles est LA porte de regeneration de ce golden, et elle ne regenere que lui.
var updateGoldenFamilles = flag.Bool("update-golden-familles", false,
	"reecrire testdata/golden_minibobine_familles.tsv apres un changement de decodage DECLARE")

// premierMax borne la valeur lisible d'une ligne : assez pour l'instant, le slot et les premiers
// champs, pas assez pour qu'une trace de projectile noie le fichier.
const premierMax = 140

// resultatFamille est UNE ligne du golden.
type resultatFamille struct {
	Nom     string
	Compte  int
	Digest  string
	Premier string
}

// recueil accumule les lignes dans l'ordre d'appel.
type recueil struct{ lignes []resultatFamille }

// ajouter fige une famille : son compte, le digest de TOUT son contenu, et sa premiere valeur.
// Une erreur devient un compte de -1 et une premiere valeur qui la porte en clair : l'etat
// « cette famille ne s'applique pas a ce film » est une donnee, pas un trou.
func (r *recueil) ajouter(nom string, compte int, contenu, premier any, err error) {
	l := resultatFamille{Nom: nom, Compte: compte}
	switch {
	case err != nil:
		l.Compte, l.Premier = -1, "ERREUR: "+err.Error()
		l.Digest = digestDe(l.Premier)
	case compte == 0:
		l.Premier, l.Digest = "(population vide)", digestDe(rendreStable(reflect.ValueOf(contenu)))
	default:
		l.Premier = tronquer(rendreStable(reflect.ValueOf(premier)))
		l.Digest = digestDe(rendreStable(reflect.ValueOf(contenu)))
	}
	r.lignes = append(r.lignes, l)
}

// ajouterSlice est la forme courante : une famille qui rend une tranche.
func ajouterSlice[T any](r *recueil, nom string, v []T, err error) {
	var premier any
	if len(v) > 0 {
		premier = v[0]
	}
	r.ajouter(nom, len(v), v, premier, err)
}

// digestDe rend le sha256 hexadecimal d'une chaine.
func digestDe(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// tronquer borne une valeur lisible et neutralise les tabulations et sauts de ligne, qui
// casseraient le format TSV.
func tronquer(s string) string {
	s = strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
	if len(s) > premierMax {
		return s[:premierMax] + "..."
	}
	return s
}

// TestGoldenMiniBobineFamilles — LE GOLDEN. Il appelle les POINTS D'ENTREE de famille, jamais les
// enveloppes `ScanFilm*(dir)` : ce sont les points d'entree que la cuisson appelle, et une
// enveloppe qui relit le disque testerait le chargement, pas le decodage.
func TestGoldenMiniBobineFamilles(t *testing.T) {
	film, err := filmsource.LoadDir(bobineFamilles, nil)
	if err != nil {
		t.Fatalf("mini-bobine versionnee illisible (%s) : %v — elle est dans le depot, "+
			"son absence est une panne, pas une condition d'execution", bobineFamilles, err)
	}
	release := LockProcessDecode()
	defer release()

	fc := NewFilmContext(film)
	var r recueil
	famillesDeltaBipede(t, &r, fc)
	famillesObjetsDuMonde(&r, fc, film)
	famillesEvenementsEtImagesCles(&r, fc, film)

	lignes := make([]string, 0, len(r.lignes))
	for _, l := range r.lignes {
		lignes = append(lignes, fmt.Sprintf("%s\t%d\t%s\t%s", l.Nom, l.Compte, l.Digest, l.Premier))
	}
	comparerGoldenFamilles(t, lignes)
}

// famillesDeltaBipede : les canaux qui marchent les records delta du bipede.
func famillesDeltaBipede(t *testing.T, r *recueil, fc *FilmContext) {
	t.Helper()
	charges, _, err := ScanAbilityCharges(fc)
	ajouterSlice(r, "abilityCharges", charges, err)
	impulses, _, err := ScanAbilityImpulses(fc)
	ajouterSlice(r, "abilityImpulses", impulses, err)
	ranks, _, err := ScanAbilityRanks(fc)
	ajouterSlice(r, "abilityRanks", ranks, err)
	aims, err := ScanBipedAimOnly(fc)
	ajouterSlice(r, "bipedAim", aims, err)
	camo, _, err := ScanCamoStates(fc)
	ajouterSlice(r, "camoStates", camo, err)
	grapple, _, err := ScanGrappleReads(fc)
	ajouterSlice(r, "grappleReads", grapple, err)
	held, _, err := ScanHeldWeaponChanges(fc, nil)
	ajouterSlice(r, "heldWeaponChanges", held, err)
	inv, _, err := ScanInventoryDeltas(fc)
	ajouterSlice(r, "inventoryDeltas", inv, err)
	unitEq, err := ScanUnitEquipment(fc)
	ajouterSlice(r, "unitEquipment", unitEq, err)
	eqChanges, _, err := ScanEquipmentChanges(fc, nil)
	ajouterSlice(r, "equipmentChanges", eqChanges, err)
	eqState, _, err := ScanEquipmentState(fc)
	ajouterSlice(r, "equipmentState", eqState, err)
}

// famillesObjetsDuMonde : les entites du monde (positions, projectiles, creations, zones).
func famillesObjetsDuMonde(r *recueil, fc *FilmContext, film *filmsource.Film) {
	wr := QuantRangeCEBiped // bornes MESUREES du film 000d5950 (cf. quantize.go) — zero fixture

	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanBipedPositions(film, opt)
	ajouterSlice(r, "bipedPositions", pos, err)

	proj, err := ScanProjectiles(film, &wr)
	ajouterSlice(r, "projectiles", proj, err)
	objets, err := ScanWorldObjects(film, &wr, GroundWeaponTypeIndex)
	ajouterSlice(r, "worldObjects_ti42", objets, err)
	eqCre, _, err := ScanEquipmentCreations(fc, &wr)
	ajouterSlice(r, "equipmentCreations", eqCre, err)
	gwCre, _, err := ScanGroundWeaponCreations(fc, &wr)
	ajouterSlice(r, "groundWeaponCreations", gwCre, err)
	vehCre, _, err := ScanVehicleCreations(fc, &wr)
	ajouterSlice(r, "vehicleCreations", vehCre, err)
	places, _, err := ScanEquipmentPlacements(fc, &wr)
	ajouterSlice(r, "equipmentPlacements", places, err)

	kf := ScanWorldObjectKeyframes(film, BipedTypeIndex)
	r.ajouter("worldObjectKeyframes_ti35", len(kf.Band), kf, kf.TimesUS, nil)

	props, err := ScanManagedProperties(fc)
	premier := any(nil)
	if len(props.Reads) > 0 {
		premier = props.Reads[0]
	}
	r.ajouter("managedProperties", len(props.Reads), props, premier, err)

	obj, err := ScanObjectives(fc)
	r.ajouter("objectives_ti11", len(obj.Reads), obj, nil, err)

	nav, err := ScanNavpointRadial(fc, nil)
	if err == nil {
		var p any
		if len(nav.Reads) > 0 {
			p = nav.Reads[0]
		}
		r.ajouter("navpointRadial", len(nav.Reads), *nav, p, nil)
	} else {
		r.ajouter("navpointRadial", 0, nil, nil, err)
	}
}

// famillesEvenementsEtImagesCles : la liste d'evenements en tete de paquet, et les images-cles.
func famillesEvenementsEtImagesCles(r *recueil, fc *FilmContext, film *filmsource.Film) {
	fire, err := ScanFireEvents(film)
	ajouterSlice(r, "fireEvents", fire, err)
	gren, err := ScanGrenadeThrows(film)
	ajouterSlice(r, "grenadeThrows", gren, err)
	pickups, _, err := ScanBipedPickups(fc)
	ajouterSlice(r, "bipedPickups", pickups, err)
	ajouterSlice(r, "zoomEvents", ScanZoomEvents(film), nil)
	ajouterSlice(r, "translocatorTeleports", ScanTranslocatorTeleports(film, nil), nil)
	veh, err := ScanVehicleEvents(fc)
	ajouterSlice(r, "vehicleEvents", veh, err)

	marks, err := ScanCarrierMarks(film)
	var pm any
	if len(marks.Marks) > 0 {
		pm = marks.Marks[0]
	}
	r.ajouter("carrierMarks", len(marks.Marks), marks, pm, err)

	// LE CATALOGUE DE FAMILLES D'ARMES EST DERIVE DU FILM LUI-MEME. Les deux balayages d'images
	// cles filtrent sur un catalogue de familles connues ; sans lui ils rendent zero, et un zero
	// de mauvais appel n'est pas une population vide. Le catalogue est donc construit ici a
	// partir des identifiants d'arme que le film porte deja (evenements de tir + changements
	// d'arme portee) : deterministe, sans fixture, et de l'ordre de grandeur que la grammaire
	// attend (quelques dizaines).
	known := catalogueDuFilm(fc, film)
	r.ajouter("catalogueFamillesDuFilm", len(known), known, known, nil)
	load, err := ScanKeyframeLoadouts(film, known)
	ajouterSlice(r, "keyframeLoadouts", load, err)
	gw, err := ScanKeyframeGroundWeapons(film, known)
	ajouterSlice(r, "keyframeGroundWeapons", gw, err)

	// Les tirs et degats n'ont PAS de forme `film` : leur point d'entree prend le repertoire.
	shots, err := ScanFilmWeaponShots(bobineFamilles, bobineChunks)
	ajouterSlice(r, "weaponShots", shots, err)
	reg, errReg := fc.Registry()
	if errReg != nil {
		r.ajouter("weaponDamages", 0, nil, nil, errReg)
		return
	}
	dmg, base, err := ScanFilmWeaponDamages(bobineFamilles, reg, bobineChunks)
	ajouterSlice(r, "weaponDamages", dmg, err)
	r.ajouter("weaponDamagesBaseSlot", base, base, base, nil)
}

// catalogueDuFilm rend les identifiants de famille d'arme que le film porte, tous chemins
// confondus. Trie a la lecture par la carte, donc stable au digest (le rendu `%+v` d'une carte Go
// est trie par cle depuis Go 1.12).
func catalogueDuFilm(fc *FilmContext, film *filmsource.Film) map[uint32]bool {
	known := map[uint32]bool{}
	if fire, err := ScanFireEvents(film); err == nil {
		for _, e := range fire {
			known[uint32(e.WeaponID>>32)] = true
			known[uint32(e.WeaponID)] = true
		}
	}
	if hw, _, err := ScanHeldWeaponChanges(fc, nil); err == nil {
		for _, c := range hw {
			known[c.Family] = true
		}
	}
	return known
}

// comparerGoldenFamilles confronte les lignes mesurees au fichier fige, ou le reecrit sous
// `-update`.
func comparerGoldenFamilles(t *testing.T, lignes []string) {
	t.Helper()
	contenu := strings.Join(lignes, "\n") + "\n"
	if *updateGoldenFamilles {
		if err := os.WriteFile(goldenFamillesPath, []byte(contenu), 0o600); err != nil {
			t.Fatalf("ecriture de %s : %v", goldenFamillesPath, err)
		}
		t.Logf("golden des familles fige : %s (%d familles)", goldenFamillesPath, len(lignes))
		return
	}
	brut, err := os.ReadFile(goldenFamillesPath)
	if err != nil {
		t.Fatalf("golden illisible (%s) : %v — le figer avec -update", goldenFamillesPath, err)
	}
	figees := strings.Split(strings.TrimRight(string(brut), "\n"), "\n")
	if len(figees) != len(lignes) {
		t.Fatalf("%d familles mesurees contre %d figees : une famille est apparue ou a disparu.\n"+
			"mesure :\n%s", len(lignes), len(figees), strings.Join(lignes, "\n"))
	}
	for i := range lignes {
		if lignes[i] != figees[i] {
			t.Errorf("famille %d a change :\n  fige  : %s\n  obtenu: %s", i+1, figees[i], lignes[i])
		}
	}
}

// rendreStable rend une valeur sous une forme DETERMINISTE d'un processus a l'autre.
//
// POURQUOI PAS `%+v`, QUI SEMBLAIT SUFFIRE. Il imprime l'ADRESSE d'un pointeur, et plusieurs
// structures du decodeur en portent (`InventoryDelta.Ammo[].Mag`, par exemple). Le digest
// changeait alors a chaque execution : un golden qui rougit au hasard ne verrouille rien et
// finit desactive. Mesure faite en figeant ce golden : deux passes consecutives donnaient deux
// empreintes differentes pour `inventoryDeltas`.
//
// CE RENDU DESCEND DANS LES CHAMPS NON EXPORTES, et c'est voulu : la moitie de ce qui distingue
// deux decodages y vit (`componentDirs`, `componentVitals` d'une position bipede). `reflect`
// interdit `.Interface()` sur eux, mais pas la lecture de leur VALEUR — c'est ce que fait ce
// rendu, cas par cas.
func rendreStable(v reflect.Value) string {
	if !v.IsValid() {
		return "nil"
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return "nil"
		}
		return "&" + rendreStable(v.Elem())
	case reflect.Struct:
		return rendreStruct(v)
	case reflect.Slice, reflect.Array:
		return rendreSuite(v)
	case reflect.Map:
		return rendreCarte(v)
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Float32, reflect.Float64:
		// 'g' avec la precision native : la meme valeur binaire rend toujours la meme chaine.
		return strconv.FormatFloat(v.Float(), 'g', -1, v.Type().Bits())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	default:
		return v.Kind().String()
	}
}

// rendreStruct rend les champs dans l'ordre de declaration, nommes.
func rendreStruct(v reflect.Value) string {
	t := v.Type()
	var b strings.Builder
	b.WriteByte('{')
	for i := 0; i < v.NumField(); i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(t.Field(i).Name)
		b.WriteByte(':')
		b.WriteString(rendreStable(v.Field(i)))
	}
	b.WriteByte('}')
	return b.String()
}

// rendreSuite rend une tranche ou un tableau. Les tranches d'octets sont rendues en hexadecimal :
// une trame brute de plusieurs kilo-octets rendue element par element etoufferait le calcul.
func rendreSuite(v reflect.Value) string {
	if v.Type().Elem().Kind() == reflect.Uint8 && v.Kind() == reflect.Slice {
		return "0x" + hex.EncodeToString(v.Bytes())
	}
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(rendreStable(v.Index(i)))
	}
	b.WriteByte(']')
	return b.String()
}

// rendreCarte rend une carte AVEC SES CLES TRIEES : l'ordre d'iteration de Go est aleatoire.
func rendreCarte(v reflect.Value) string {
	cles := make([]string, 0, v.Len())
	valeurs := map[string]string{}
	iter := v.MapRange()
	for iter.Next() {
		k := rendreStable(iter.Key())
		cles = append(cles, k)
		valeurs[k] = rendreStable(iter.Value())
	}
	sort.Strings(cles)
	var b strings.Builder
	b.WriteString("map[")
	for i, k := range cles {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(k)
		b.WriteByte(':')
		b.WriteString(valeurs[k])
	}
	b.WriteByte(']')
	return b.String()
}
