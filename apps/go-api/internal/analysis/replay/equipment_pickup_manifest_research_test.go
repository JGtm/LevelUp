package replay

// equipment_pickup_manifest_research_test.go — LOT 5, ÉTAPE 1 : NOMMER les ramassages
// non-arme PAR LES FICHIERS DU JEU, et non plus par corrélation statistique.
//
// ## LE FAIT QUI OUVRE LA VOIE, ET POURQUOI LE LOT 4 NE L'AVAIT PAS VU
//
// Le lot 4 a cherché à nommer le `R(32)` des classes 2/3 par CORRÉLATION (rang i48, puis état
// des images-clés) et a plafonné à 19-29 % de couverture. Les deux voies mesuraient la bonne
// chose ; elles n'avaient simplement pas assez de matière.
//
// Or le dépôt porte DÉJÀ une table id -> famille construite par un chemin tout autre : la
// STRUCTURE DES FICHIERS DU JEU. Le chantier `PLAN_NOMMAGE_EQIP_TRANSLOCATEUR` (2026-08-18) a
// remonté la chaîne `sofd -> sofa -> {string_id, eqip}` dans les modules installés et cassé les
// `string_id` par dictionnaire murmur3, produisant les 21 lignes `[[equipment_objects]]` de
// `config/titles/halo_infinite/mappings/replay_labels.toml`. Cette table est keyée par GlobalID
// de tag `eqip` — l'identifiant que le film écrit sur une CRÉATION d'objet ti=37.
//
// L'HYPOTHÈSE DE CE LOT, ET C'EST TOUT CE QU'ELLE EST : le `R(32)` de la charge d'un
// `biped_pickup` de classe 2/3 vit dans le MÊME espace d'identifiants que la création ti=37,
// exactement comme le `R(32)` des classes 0/1 vit dans l'espace des familles d'arme. Si elle
// tient, le nommage est acquis d'un coup, par le jeu et non par la statistique.
//
// ## LES QUATRE JUGES, ÉCRITS AVANT LA MESURE
//
//	M1 — COUVERTURE. Au moins 90 % des ramassages non-arme, et au moins 90 % de leurs
//	     identifiants DISTINCTS, se résolvent dans le manifeste. Sous ce seuil, l'espace
//	     d'identifiants n'est pas celui-là (ou le manifeste est incomplet, ce qui se
//	     distingue en regardant QUELS identifiants manquent).
//	M2 — SÉPARATION DES ESPACES (le témoin structurel). Aucun identifiant de classe ARME
//	     (0/1) ne doit se résoudre dans le manifeste d'équipement, et aucun identifiant de
//	     classe NON-ARME ne doit se résoudre dans le catalogue d'armes. Deux espaces qui se
//	     chevaucheraient rendraient la table ambiguë : c'est la « zéro collision » du cahier.
//	     C'est aussi le seul témoin honnête disponible ici — un manifeste de 21 entrées face
//	     à un espace de 2^32 ne peut pas couvrir 90 % d'un ensemble par hasard, mais il
//	     POURRAIT le faire si les deux espaces étaient en réalité confondus.
//	M3 — CONCORDANCE avec les deux étiquettes que le lot 4 a acquises par DEUX voies
//	     indépendantes : `eef5d48d` = Thruster/propulseur (rang 21) et `8e2dc574` = rang 19.
//	     Le manifeste doit dire `thruster` pour le premier et la famille du rang 19 pour le
//	     second, SANS avoir jamais consulté un film.
//	M4 — GAIN sur le lot 4 : la couverture nommée doit dépasser les 19,5 % / 25,0 % de la
//	     voie delta i48. Sinon la voie ne vaut pas d'être publiée.
//
// Garde `BIPED_PICKUP_FILM` (constante `pickupsBridgeEnv`), comme tout le chantier. Recherche
// pure : aucun fichier de production n'est touché, aucune cuisson n'est lancée.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/weapons"
)

// epmSeuilCouverture est le barreau de M1, en pourcentage.
const epmSeuilCouverture = 90.0

// epmAcquis — les deux étiquettes que le lot 4 a établies par corrélation, sur deux films et
// par deux voies indépendantes (delta i48 ET état des images-clés). Elles servent de CONTRÔLE
// au manifeste, pas l'inverse : elles ont été écrites avant que le manifeste soit consulté.
//
// `8e2dc574` avait reçu « rang 19 », que la palette `famille_b` ne nomme pas — la table du jeu
// est justement ce qui comble ce trou. La famille attendue vient de la lecture Theater du
// 2026-07-27 consignée dans PLAN_NOMMAGE_EQIP_TRANSLOCATEUR (rang 19 = mur).
var epmAcquis = map[uint32]string{
	0xeef5d48d: "thruster",
	0x8e2dc574: "wall",
}

// epmClasseFamille range une famille du manifeste dans l'une des deux natures que l'étape 2
// oppose. Le manifeste ne porte PAS ce classement : il est dérivé du seul préfixe `grenade_`,
// qui est une convention du manifeste lui-même (quatre entrées `gggl_entree`, la liste des
// grenades du jeu) et non une interprétation de ce lot.
func epmEstGrenade(famille string) bool { return strings.HasPrefix(famille, "grenade_") }

// TestEquipmentPickupManifestNaming — ÉTAPE 1. Résoudre le `R(32)` des ramassages non-arme
// dans le manifeste d'objets d'équipement du titre, construit à partir des fichiers du jeu.
func TestEquipmentPickupManifestNaming(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, stats, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	familles := goldenReplayLabels(t).EquipmentObjects()
	armes := weapons.FilmshellWeaponKeysByFamily()
	t.Logf("== ÉTAPE 1 — NOMMER PAR LES FICHIERS DU JEU · %s ==", dir)
	t.Logf("ramassages natifs : %d (publiés %d) · manifeste d'équipement : %d entrée(s) · catalogue d'armes : %d famille(s)",
		len(pickups), stats.Published, len(familles), len(armes))

	// Recensement par identifiant, en gardant la ventilation par classe : c'est elle qui sert
	// l'étape 2 sans re-décoder le film.
	type compte struct {
		total int
		class map[uint8]int
	}
	nonArme := map[uint32]*compte{}
	arme := map[uint32]*compte{}
	nNonArme, nArme := 0, 0
	for _, p := range pickups {
		cible, n := nonArme, &nNonArme
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			cible, n = arme, &nArme
		}
		*n++
		e := cible[p.CatalogID]
		if e == nil {
			e = &compte{class: map[uint8]int{}}
			cible[p.CatalogID] = e
		}
		e.total++
		e.class[p.Class]++
	}

	// M1 — couverture, en événements ET en identifiants distincts.
	couverts, distinctsCouverts := 0, 0
	var manquants []uint32
	for id, e := range nonArme {
		if _, ok := familles[id]; ok {
			couverts += e.total
			distinctsCouverts++
			continue
		}
		manquants = append(manquants, id)
	}
	sort.Slice(manquants, func(i, j int) bool { return manquants[i] < manquants[j] })

	// M2 — séparation des espaces, dans les deux sens.
	var armeDansEquipement, nonArmeDansArmes []uint32
	for id := range arme {
		if _, ok := familles[id]; ok {
			armeDansEquipement = append(armeDansEquipement, id)
		}
	}
	for id := range nonArme {
		if _, ok := armes[id]; ok {
			nonArmeDansArmes = append(nonArmeDansArmes, id)
		}
	}
	sort.Slice(armeDansEquipement, func(i, j int) bool { return armeDansEquipement[i] < armeDansEquipement[j] })
	sort.Slice(nonArmeDansArmes, func(i, j int) bool { return nonArmeDansArmes[i] < nonArmeDansArmes[j] })

	// La table, événement par événement, avec la ventilation de classe.
	ids := make([]uint32, 0, len(nonArme))
	for id := range nonArme {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return nonArme[ids[i]].total > nonArme[ids[j]].total })
	t.Log("TABLE identifiant -> famille (manifeste du titre) · ventilation de classe :")
	for _, id := range ids {
		e := nonArme[id]
		fam, ok := familles[id]
		if !ok {
			fam = "<< ABSENT DU MANIFESTE >>"
		}
		t.Logf("  %08x  n=%-3d  classes %s  ->  %s", id, e.total, epmClassHist(e.class), fam)
	}

	t.Logf("COUVERTURE : %d/%d événements non-arme (%.1f %%) · %d/%d identifiants distincts (%.1f %%)",
		couverts, nNonArme, pct100(couverts, nNonArme),
		distinctsCouverts, len(nonArme), pct100(distinctsCouverts, len(nonArme)))
	if len(manquants) > 0 {
		t.Logf("IDENTIFIANTS NON RÉSOLUS : %s", epmHex(manquants))
	}
	t.Logf("SÉPARATION DES ESPACES : identifiants de classe ARME présents dans le manifeste d'équipement : %d %s · identifiants NON-ARME présents dans le catalogue d'armes : %d %s",
		len(armeDansEquipement), epmHex(armeDansEquipement),
		len(nonArmeDansArmes), epmHex(nonArmeDansArmes))

	// M3 — concordance avec les deux étiquettes acquises par corrélation.
	concorde, vues := 0, 0
	for id, attendu := range epmAcquis {
		if _, present := nonArme[id]; !present {
			t.Logf("CONCORDANCE %08x : identifiant absent de CE film (rien à conclure)", id)
			continue
		}
		vues++
		got := familles[id]
		if got == attendu {
			concorde++
			t.Logf("CONCORDANCE %08x : manifeste = %q, corrélation attendait %q — ACCORD", id, got, attendu)
			continue
		}
		t.Logf("CONCORDANCE %08x : manifeste = %q, corrélation attendait %q — DÉSACCORD", id, got, attendu)
	}

	t.Logf("VERDICT M1 (>= %.0f %% couverts, événements ET distincts) : %v",
		epmSeuilCouverture,
		pct100(couverts, nNonArme) >= epmSeuilCouverture && pct100(distinctsCouverts, len(nonArme)) >= epmSeuilCouverture)
	t.Logf("VERDICT M2 (zéro chevauchement d'espaces) : %v", len(armeDansEquipement) == 0 && len(nonArmeDansArmes) == 0)
	t.Logf("VERDICT M3 (concordance) : %d/%d des étiquettes acquises présentes dans ce film", concorde, vues)
	t.Logf("VERDICT M4 (gain sur la voie delta i48, 19,5 %% / 25,0 %%) : couverture nommée = %.1f %%",
		pct100(couverts, nNonArme))
}

// TestEquipmentPickupClassByManifest — ÉTAPE 2. Trancher « classe 2 = grenades, classe 3 =
// équipement » PAR LE NOM, une fois que l'étape 1 a nommé les identifiants.
//
// POURQUOI CE JUGE EST DIFFÉRENT DES DEUX PRÉCÉDENTS. Le lot 4 opposait la classe au RANG i48
// (J1) et au COMPTEUR de grenades (J2) : deux juges pris DANS LE FILM, donc tributaires de la
// densité des émissions et de fenêtres temporelles. Ici la nature de l'objet vient des
// FICHIERS DU JEU et ne dépend d'aucune fenêtre : le croisement classe × nature est un
// tableau de contingence pur.
//
// SEUILS ÉCRITS AVANT LA MESURE :
//
//	C1 — la classe 2 est GRENADE dans >= 90 % de ses événements résolus, la classe 3 dans
//	     <= 10 %. Une séparation franche, sur les deux films.
//	C2 — TÉMOIN de structure : le même tableau calculé en PERMUTANT les classes (2 <-> 3)
//	     doit s'effondrer. C'est mécanique, pas informatif — il est calculé pour être dit,
//	     pas pour convaincre.
//	C3 — le vrai témoin est ailleurs : la nature ne doit pas être une PROPRIÉTÉ DE LA
//	     FRÉQUENCE. On vérifie donc que la séparation tient identifiant par identifiant (un
//	     identifiant ne se répartit pas sur les deux classes) et pas seulement en volume.
func TestEquipmentPickupClassByManifest(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, _, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	familles := goldenReplayLabels(t).EquipmentObjects()
	t.Logf("== ÉTAPE 2 — CLASSE x NATURE, PAR LE MANIFESTE · %s ==", dir)

	// Contingence classe -> {grenade, non-grenade, non résolu}.
	type ligne struct{ grenade, autre, inconnu int }
	tab := map[uint8]*ligne{}
	// Et la répartition de classe PAR IDENTIFIANT, pour C3.
	parID := map[uint32]map[uint8]int{}
	for _, p := range pickups {
		if filmdec.BipedPickupIsWeaponClass(p.Class) {
			continue
		}
		l := tab[p.Class]
		if l == nil {
			l = &ligne{}
			tab[p.Class] = l
		}
		fam, ok := familles[p.CatalogID]
		switch {
		case !ok:
			l.inconnu++
		case epmEstGrenade(fam):
			l.grenade++
		default:
			l.autre++
		}
		if parID[p.CatalogID] == nil {
			parID[p.CatalogID] = map[uint8]int{}
		}
		parID[p.CatalogID][p.Class]++
	}

	classes := make([]uint8, 0, len(tab))
	for c := range tab {
		classes = append(classes, c)
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i] < classes[j] })
	t.Log("CONTINGENCE classe x nature (résolus seulement pour les taux) :")
	taux := map[uint8]float64{}
	for _, c := range classes {
		l := tab[c]
		res := l.grenade + l.autre
		taux[c] = pct100(l.grenade, res)
		t.Logf("  classe %d : grenade %d · autre %d · non résolu %d  ->  %.1f %% de GRENADE sur %d résolus",
			c, l.grenade, l.autre, l.inconnu, taux[c], res)
	}

	// C3 — pureté par identifiant.
	pur, melange := 0, 0
	var melanges []uint32
	for id, m := range parID {
		if len(m) <= 1 {
			pur++
			continue
		}
		melange++
		melanges = append(melanges, id)
	}
	sort.Slice(melanges, func(i, j int) bool { return melanges[i] < melanges[j] })
	t.Logf("PURETÉ par identifiant : %d pur(s) sur une seule classe · %d réparti(s) sur deux classes %s",
		pur, melange, epmHex(melanges))

	c2, ok2 := taux[2]
	c3, ok3 := taux[3]
	t.Logf("VERDICT C1 (classe 2 >= 90 %% grenade ET classe 3 <= 10 %%) : %v (classe 2 = %.1f %% · classe 3 = %.1f %%)",
		ok2 && ok3 && c2 >= 90 && c3 <= 10, c2, c3)
	t.Logf("VERDICT C2 (témoin permuté 2<->3 s'effondre) : %v — permuté, la classe 2 rendrait %.1f %% de grenade",
		ok2 && ok3 && c3 < 90, c3)
	t.Logf("VERDICT C3 (aucun identifiant réparti sur deux classes) : %v", melange == 0)
}

// epmClassHist rend un histogramme classe -> compte, trié.
func epmClassHist(m map[uint8]int) string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%d:%d", k, m[uint8(k)]))
	}
	return strings.Join(parts, " ")
}

// epmHex rend une liste d'identifiants en hexadécimal, ou « — » si elle est vide.
func epmHex(ids []uint32) string {
	if len(ids) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%08x", id))
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// TestBuildPickupsFamilyCoverageOnRealFilms — LA COUVERTURE DE `family` PAR LA CHAINE DE
// PRODUCTION, sur les deux films de reference.
//
// POURQUOI CE TEST EXISTE A COTE DES MESURES DU LOT 5. Les mesures de recherche resolvaient les
// identifiants a la main, contre le manifeste. Celui-ci appelle `buildPickups` — LA fonction de
// production — avec les catalogues que la couche titre lui donne reellement, et publie le taux
// que le document portera. Sans lui, « 100 % » serait une propriete de mon instrument, pas du
// produit.
//
// AUCUNE CUISSON : un film est decode en memoire, aucun artefact n est ecrit.
func TestBuildPickupsFamilyCoverageOnRealFilms(t *testing.T) {
	dir := os.Getenv(pickupsBridgeEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument de mesure saute", pickupsBridgeEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	pickups, st, err := filmdec.ScanFilmBipedPickups(dir)
	if err != nil {
		t.Fatalf("ramassages natifs illisibles : %v", err)
	}
	// LES MEMES TABLES QUE LA COUCHE TITRE POSE dans LabelCatalog (cf. replaylabels.Load) :
	// `Keys` vient de FilmshellWeaponKeysByFamily, `EquipmentFamilies` du manifeste.
	equipement := goldenReplayLabels(t).EquipmentObjects()
	armes := weapons.FilmshellWeaponKeysByFamily()

	got, cov := buildPickups(pickups,
		replayClock{origin: 0, step: 100_000, families: equipement}, nil, st, armes)

	parNature := map[PickupKind]struct{ total, nomme int }{}
	for _, p := range got {
		e := parNature[p.Kind]
		e.total++
		if p.Family != "" {
			e.nomme++
		}
		parNature[p.Kind] = e
	}
	t.Logf("== COUVERTURE `family` PAR LA CHAINE DE PRODUCTION · %s ==", dir)
	t.Logf("publies : %d · sans famille : %d", cov.Published, cov.UnknownFamilies)
	for _, k := range []PickupKind{PickupWeapon, PickupGrenade, PickupEquipment, PickupItem} {
		e := parNature[k]
		if e.total == 0 {
			continue
		}
		t.Logf("  %-10s : %d/%d nommes (%.1f %%)", k, e.nomme, e.total,
			100*float64(e.nomme)/float64(e.total))
	}
	// LE SEUIL PORTE SUR LES NON-ARMES, et sur elles seules : c est ce que ce lot a resolu.
	// Le catalogue d ARMES ne couvre pas tout le jeu et n est pas l objet de cette mesure —
	// son taux est publie, pas juge.
	nonArme := parNature[PickupGrenade]
	eq := parNature[PickupEquipment]
	totalNA, nommeNA := nonArme.total+eq.total, nonArme.nomme+eq.nomme
	if totalNA == 0 {
		t.Fatal("aucun ramassage non-arme dans ce film : la mesure n a pas de denominateur")
	}
	if nommeNA != totalNA {
		t.Errorf("non-armes nommees : %d/%d — le manifeste du titre ne couvre plus tout le corpus "+
			"de ce film ; publier le trou plutot que de baisser le seuil", nommeNA, totalNA)
	}
	var inconnues []uint32
	vu := map[uint32]bool{}
	for i, p := range got {
		if p.Family != "" || vu[pickups[i].CatalogID] {
			continue
		}
		vu[pickups[i].CatalogID] = true
		inconnues = append(inconnues, pickups[i].CatalogID)
	}
	sort.Slice(inconnues, func(a, b int) bool { return inconnues[a] < inconnues[b] })
	t.Logf("IDENTIFIANTS SANS FAMILLE (distincts) : %s", epmHex(inconnues))
	t.Logf("VERDICT : non-armes nommees %d/%d (%.1f %%)", nommeNA, totalNA,
		100*float64(nommeNA)/float64(totalNA))
}
