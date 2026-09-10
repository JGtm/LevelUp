package killicon

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/testfixtures"
)

// atlasPrefix : toutes les vignettes de ce pont viennent de l atlas KILL FEED. L atlas
// `contour` ne porte ni grenade, ni melee, ni pictogramme — s en servir ici laisserait
// 12 % des morts (la melee) sans image.
const atlasPrefix = "killfeed-"

func spritePath(stem string) string {
	return filepath.Join(testfixtures.RepoRoot(),
		"static", "weapons-assets", "halo_infinite", "jeu", stem+".png")
}

// registryEN lit le registre versionne des noms d armes : weapon_key -> nom EN.
func registryEN(t *testing.T) map[string]string {
	t.Helper()
	p := filepath.Join(testfixtures.RepoRoot(),
		"config", "titles", "halo_infinite", "mappings", "weapon_names.toml")
	raw, err := os.ReadFile(p) //nolint:gosec // chemin construit depuis la racine du depot
	if err != nil {
		t.Fatalf("weapon_names.toml : %v", err)
	}
	var doc struct {
		Weapons map[string]struct {
			EN string `toml:"en"`
		} `toml:"weapons"`
	}
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("weapon_names.toml : %v", err)
	}
	out := make(map[string]string, len(doc.Weapons))
	for k, v := range doc.Weapons {
		out[k] = v.EN
	}
	return out
}

// branches decoupe un nom alternatif << A / B >> en ses branches.
func branches(name string) []string {
	parts := strings.Split(name, " / ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// spriteByName : les regles NOM indexees, pour verifier la coherence des alternatives.
func spriteByName() map[string]string {
	out := map[string]string{}
	for _, r := range Rules() {
		if r.Genre == GenreNom {
			out[r.Key] = r.Sprite
		}
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// LE PONT TIENT-IL A SES DEUX SOURCES ?
// ─────────────────────────────────────────────────────────────────────────────

// TestChaqueRegleTrouveSaSource est le garde-rail de PEREMPTION. `labels.tsv` grandit a
// chaque saison et peut voir ses noms corriges. Une regle dont la cle ne designe plus
// rien serait un trou d icone SILENCIEUX : ici elle est rouge.
func TestChaqueRegleTrouveSaSource(t *testing.T) {
	noms := map[string]bool{}
	classes := map[string]bool{}
	gggl := map[string]bool{}
	banques := map[string]bool{}
	for _, l := range damagetag.Labels() {
		if l.Name != "" {
			noms[l.Name] = true
		}
		classes[string(l.Class)] = true
		if m := ggglRe.FindStringSubmatch(l.Detail); m != nil {
			gggl[m[1]] = true
		}
		// LES DEUX FORMES DE RACINE, comme le resolveur : la courte (trois segments, celle
		// des chassis) ET la longue (le nom de banque entier, celle qui distingue les quatre
		// bobines). Ne construire que la courte rendait ce garde-rail ROUGE sur des regles
		// pourtant justes — il aurait dit `aucune ligne ne cite cette racine` alors que la
		// ligne la cite, mais plus loin que le troisieme segment.
		for _, re := range []*regexp.Regexp{banqueRe, banqueLongueRe} {
			for _, m := range re.FindAllStringSubmatch(l.Detail, -1) {
				banques[m[1]] = true
			}
		}
	}
	for _, r := range Rules() {
		switch r.Genre {
		case GenreNom:
			if !noms[r.Key] {
				t.Errorf("regle NOM %q : aucune ligne de labels.tsv ne porte ce nom", r.Key)
			}
		case GenreClasse:
			if !classes[r.Key] {
				t.Errorf("regle CLASSE %q : classe absente de labels.tsv", r.Key)
			}
		case GenreGGGL:
			if !gggl[r.Key] {
				t.Errorf("regle GGGL %q : aucune ligne ne porte cette entree de grenade", r.Key)
			}
		case GenreBanque:
			if !banques[r.Key] {
				t.Errorf("regle BANQUE %q : aucune ligne ne cite cette racine de banque", r.Key)
			}
		}
	}
}

// TestChaqueVignetteExisteSurDisque ferme le trou par lequel un renommage de PNG
// passerait la CI en cassant l UI : le pont ne peut pas designer un fichier absent.
func TestChaqueVignetteExisteSurDisque(t *testing.T) {
	for _, r := range Rules() {
		if _, err := os.Stat(spritePath(r.Sprite)); err != nil {
			t.Errorf("regle %s %q -> %q : PNG absent (%v)", r.Genre, r.Key, r.Sprite, err)
		}
	}
}

// TestToutesLesVignettesViennentDeLAtlasKillFeed verrouille la decision du lot : le feed
// se sert du format bandeau, pas des grandes images de l atlas `contour`.
func TestToutesLesVignettesViennentDeLAtlasKillFeed(t *testing.T) {
	if len(Rules()) == 0 {
		t.Fatal("aucune regle : le test ne prouve rien")
	}
	for _, r := range Rules() {
		if !strings.HasPrefix(r.Sprite, atlasPrefix) {
			t.Errorf("regle %s %q -> %q : hors atlas %q", r.Genre, r.Key, r.Sprite, atlasPrefix)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AUCUNE ARME SUR LA MAUVAISE ICONE
// ─────────────────────────────────────────────────────────────────────────────

// aliasAssumes : les SEULES divergences de nommage tolerees entre le nom de labels.tsv
// et le nom EN du registre d armes. Chacune est une divergence CONNUE et documentee du
// depot, pas un a-peu-pres. Toute autre divergence fait echouer le test.
//
//	Mk51 Sidekick : `weapon_labels.name_en` (d ou vient labels.tsv) dit Mk51, le registre
//	                dit Mk50. Meme arme, divergence relevee par le plan d integration des
//	                icones (decouvertes, 2026-08-09) et non corrigee a ce jour.
var aliasAssumes = map[string]string{
	"Mk51 Sidekick": "Mk50 Sidekick",
}

// TestChaqueRegleEstCorroboreeParLeRegistre est LE test qui interdit l icone fausse.
//
// Il ne croit pas la table sur parole : pour chaque regle qui declare un `weapon_key`, il
// exige que le nom EN du registre versionne (`weapon_names.toml`) soit bien celui de la
// cle — directement, par alias assume, ou par l une des branches d un nom alternatif.
// Une transcription de travers (la vignette du Cindershot donnee au Heatwave, par
// exemple) sort ici, mecaniquement.
func TestChaqueRegleEstCorroboreeParLeRegistre(t *testing.T) {
	reg := registryEN(t)
	corrobore := 0
	for _, r := range Rules() {
		if r.Genre != GenreNom || r.WeaponKey == "" {
			continue
		}
		en, ok := reg[r.WeaponKey]
		if !ok {
			t.Errorf("regle NOM %q : weapon_key %q absent du registre", r.Key, r.WeaponKey)
			continue
		}
		accepte := en == r.Key || aliasAssumes[r.Key] == en
		for _, b := range branches(r.Key) {
			if b == en || aliasAssumes[b] == en {
				accepte = true
			}
		}
		if !accepte {
			t.Errorf("regle NOM %q -> weapon_key %q : le registre dit %q — "+
				"soit la regle designe la mauvaise arme, soit l alias manque", r.Key, r.WeaponKey, en)
			continue
		}
		corrobore++
	}
	if corrobore == 0 {
		t.Fatal("aucune regle corroboree : le test ne prouve rien")
	}
}

// TestUneAlternativeNeSertQuUneVignetteUnanime : un nom alternatif (<< A / B >>) ne peut
// porter une icone que si toutes ses branches CONNUES designent la meme. C est la regle
// de surete du lot, verifiee sur la table plutot que crue sur commentaire.
func TestUneAlternativeNeSertQuUneVignetteUnanime(t *testing.T) {
	spr := spriteByName()
	verifiees := 0
	for _, r := range Rules() {
		if r.Genre != GenreNom || !strings.Contains(r.Key, " / ") {
			continue
		}
		for _, b := range branches(r.Key) {
			s, ok := spr[b]
			if !ok {
				continue // branche sans regle propre : rien a contredire
			}
			verifiees++
			if s != r.Sprite {
				t.Errorf("alternative %q -> %q, mais la branche %q -> %q : "+
					"une des deux est fausse", r.Key, r.Sprite, b, s)
			}
		}
	}
	if verifiees == 0 {
		t.Fatal("aucune branche confrontee : le test ne prouve rien")
	}
}

// alternativesEcartees : les noms alternatifs volontairement SANS regle. Le test verifie
// deux choses a la fois — qu ils ne recoivent aucune icone, et que l ecart est MOTIVE
// (au moins deux branches connues qui designent des vignettes differentes).
var alternativesEcartees = []string{
	"CQS48 Bulldog / Mutilator",
	"Disruptor / Shock Rifle / Shock Rifle (Ranked)",
	"Gravity Hammer / Mutilator",
	"Mangler / Ravager / Shock Rifle / Skewer",
	"Needler / Plasma Pistol",
}

func TestLesAlternativesContradictoiresNObtiennentAucuneIcone(t *testing.T) {
	spr := spriteByName()
	for _, nom := range alternativesEcartees {
		if s, ok := spr[nom]; ok {
			t.Errorf("%q a recu la vignette %q : ses branches ne designent pas la meme arme", nom, s)
		}
		vues := map[string]bool{}
		for _, b := range branches(nom) {
			if s, ok := spr[b]; ok {
				vues[s] = true
			}
		}
		if len(vues) < 2 {
			t.Errorf("%q : ecart NON motive — %d vignette(s) distincte(s) parmi ses branches, "+
				"il en faut au moins 2 pour justifier le repli", nom, len(vues))
		}
	}
	// Non-vacuite : ces noms doivent exister dans labels.tsv, sinon le test protege du vide.
	noms := map[string]bool{}
	for _, l := range damagetag.Labels() {
		noms[l.Name] = true
	}
	for _, nom := range alternativesEcartees {
		if !noms[nom] {
			t.Errorf("%q n existe plus dans labels.tsv : la liste des ecarts est perimee", nom)
		}
	}
}

// TestUnEffetPartageParPlusieursChassisNObtientAucuneIcone : le pendant, cote vehicules, de
// la regle des noms alternatifs. Une ligne qui cite plusieurs racines de banque decrit un
// effet PARTAGE — par exemple le Chopper, la Banshee, le Ghost et le Wasp sur le meme tag :
// lui donner l icone de l un des quatre serait faux trois fois sur quatre.
//
// Le test verifie les deux sens : ces lignes n ont pas d icone, ET il en existe vraiment
// (sinon il ne protegerait rien).
func TestUnEffetPartageParPlusieursChassisNObtientAucuneIcone(t *testing.T) {
	partagees, uniques := 0, 0
	for _, l := range damagetag.Labels() {
		if l.Class != damagetag.ClassVehicule || !l.Publishable() {
			continue
		}
		racines := map[string]bool{}
		for _, m := range banqueRe.FindAllStringSubmatch(l.Detail, -1) {
			racines[m[1]] = true
		}
		switch {
		case len(racines) > 1:
			partagees++
			if ic, ok := Lookup(l.Tag); ok {
				t.Errorf("tag %08x cite %d chassis mais recoit %q", l.Tag, len(racines), ic.Sprite)
			}
		case len(racines) == 1:
			uniques++
		}
	}
	if partagees == 0 || uniques == 0 {
		t.Fatalf("le test ne prouve rien : %d lignes partagees, %d a chassis unique", partagees, uniques)
	}
}

// TestAucuneSourceNonPubliableNObtientDIcone : les statuts AMBIGU (effet generique qui
// traverse plusieurs entrees de grenade) ne peuvent pas recevoir d image, exactement
// comme ils ne peuvent pas recevoir de libelle.
func TestAucuneSourceNonPubliableNObtientDIcone(t *testing.T) {
	ambigus := 0
	for _, l := range damagetag.Labels() {
		if l.Publishable() {
			continue
		}
		if l.Status == damagetag.StatusAmbigu {
			ambigus++
		}
		if _, ok := Lookup(l.Tag); ok {
			t.Errorf("tag %08x (statut %s) a une icone alors qu il n est pas publiable", l.Tag, l.Status)
		}
	}
	if ambigus == 0 {
		t.Fatal("aucune ligne AMBIGU dans labels.tsv : le test ne prouve rien")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// COUVERTURE ET COHERENCE DE LA TABLE
// ─────────────────────────────────────────────────────────────────────────────

// TestEnteteAnnonceLeBonNombreDeRegles : l en-tete `regles=` doit suivre le contenu.
// Sans ce controle, une ligne supprimee par megarde passerait inapercue.
func TestEnteteAnnonceLeBonNombreDeRegles(t *testing.T) {
	p := Source()
	if p.AnnouncedNb != p.RuleCount {
		t.Errorf("en-tete regles=%d, fichier %d lignes", p.AnnouncedNb, p.RuleCount)
	}
	if p.Date == "" {
		t.Error("en-tete sans date=")
	}
}

// TestCouvertureParClasse mesure ce que le pont couvre, classe par classe, et FIGE le
// resultat. Le chiffre n est pas decoratif : c est la promesse produit (une icone sur le
// kill feed) rendue verifiable. Une baisse = une regression, une hausse = une decision a
// consigner.
func TestCouvertureParClasse(t *testing.T) {
	attendu := map[damagetag.Class][2]int{ // classe -> {publiables, avec icone}
		// ARME 114->115 publiables, 105->106 avec icone (2026-08-25) : le repulseur
		// (jpt! 07104b31, RE himap eqip 7ca85adc -> sofa 6845f2b3 -> eqip frere 1e79ebda)
		// rejoint la table, cf. killicon/data/rules.tsv "NOM Repulsor".
		damagetag.ClassArme:    {115, 106},
		damagetag.ClassMelee:   {14, 14},
		damagetag.ClassGrenade: {15, 15},
		// VEHICULE 46 -> 48 avec icone le 2026-09-10 (genre PORTEUR). Le +2 vient
		// ENTIEREMENT du Gungoose (`00426796`, `00426797`) : ces deux tags ne citent aucune
		// banque sonore et n avaient donc jamais pu recevoir d icone.
		//
		// LE WARTHOG N APPARAIT PAS DANS CE CHIFFRE, et c est la lecon de ce compteur : son
		// tag `382cafaf` avait DEJA une icone — la mauvaise, celle de la tourelle generique.
		// Il a change de vignette, pas de statut. Un compte global ne voit pas un echange :
		// c est `TestPorteurPrimeBanque` qui l epingle, tag par tag.
		damagetag.ClassVehicule: {89, 48},
		// OBJET_EXPLOSIF ENTRE DANS LA TABLE LE 2026-08-27, et c est une DECISION, pas une
		// derive : les quatre bobines ont chacune leur vignette dans l atlas (42 Shock,
		// 43 Blast, 44 UNSC fusion, 45 Plasma, passe humaine du 2026-08-09). Ce qui les
		// tenait dehors etait technique — la coupe a trois segments de banqueRe les rendait
		// toutes identiques (exp_single_small) — pas doctrinal.
		// 8 sur 19 SEULEMENT, et le reste n est pas un oubli : les onze autres lignes
		// d OBJET_EXPLOSIF ne citent AUCUNE banque sonore (elles designent un hlmt, un effe
		// ou un weap). Sans flaveur d energie, rien ne dit LAQUELLE des quatre bobines c est,
		// et servir une icone au hasard serait le defaut que ce pont existe pour eviter.
		damagetag.ClassObjet: {19, 8},
	}
	got := map[damagetag.Class][2]int{}
	for _, l := range damagetag.Labels() {
		if !l.Publishable() {
			continue
		}
		c := got[l.Class]
		c[0]++
		if _, ok := Lookup(l.Tag); ok {
			c[1]++
		}
		got[l.Class] = c
	}
	for cl, want := range attendu {
		if got[cl] != want {
			t.Errorf("classe %s : %v publiables/avec icone, attendu %v", cl, got[cl], want)
		}
	}
	// Les classes sans regle ne doivent JAMAIS recevoir d icone (bidon, chute, inconnu) :
	// servir une image au hasard serait le defaut que ce lot evite.
	for _, cl := range []damagetag.Class{
		damagetag.ClassGlobal, damagetag.ClassInconnu,
	} {
		if n := got[cl][1]; n != 0 {
			t.Errorf("classe %s : %d icones alors qu aucune regle ne la couvre", cl, n)
		}
	}
}

// TestLookupEstIndexeParTag verifie le contrat d execution : la resolution passe par le
// tag, jamais par une chaine. Un tag connu rend sa vignette, un tag absent du catalogue
// rend faux sans paniquer.
func TestLookupEstIndexeParTag(t *testing.T) {
	tags := ResolvedTags()
	if len(tags) == 0 {
		t.Fatal("aucun tag resolu")
	}
	ic, ok := Lookup(tags[0])
	if !ok || ic.Sprite == "" {
		t.Fatalf("Lookup(%08x) = %+v, %v", tags[0], ic, ok)
	}
	if _, ok := Lookup(0xffffffff); ok {
		t.Error("Lookup(0xffffffff) : un tag hors catalogue ne doit pas avoir d icone")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// GENRE PORTEUR (2026-09-10)
// ─────────────────────────────────────────────────────────────────────────────

// Les deux `jpt!` qui citent la MEME banque `tur_un_machinegun` et que SEUL le porteur
// distingue. Ils sont le coeur du lot : avant le genre PORTEUR ils rendaient la meme
// vignette generique, alors que 173 des 184 morts mesurees venaient du Warthog (vue
// `match_kill_events_latest` — la table brute, ou cinq revisions de decodeur coexistent,
// annoncait 1002 sur 1072 : meme rapport, effectifs gonfles six fois).
const (
	tagWarthogLAAG  = 0x382cafaf // vehi dd7f9102 — mitrailleuse du Warthog
	tagTourelleFixe = 0x00015cd1 // vehi 0000d500 — tourelle fixe posee sur la carte
)

// TestChaqueReglePorteurTrouveSonVehi : le pendant de TestChaqueRegleTrouveSaSource pour le
// nouveau genre.
//
// POURQUOI IL EXISTE. Une regle PORTEUR dont le `vehi` a disparu de labels.tsv (saison
// suivante, table regeneree) serait INERTE : pas d erreur, pas de compteur, juste une icone
// qui cesse de sortir. C est trait pour trait le silence que ce lot repare — la table SAVAIT
// nommer le Warthog depuis le 2026-09-02 et personne ne l a su pendant huit jours. La
// peremption de ce genre doit donc se payer en rouge, jamais en discretion.
func TestChaqueReglePorteurTrouveSonVehi(t *testing.T) {
	reg := registryEN(t)
	verifiees := 0
	for _, r := range Rules() {
		if r.Genre != GenrePorteur {
			continue
		}
		if !unVehiDeclareCePorteur(r.Key) {
			t.Errorf("regle PORTEUR %q : aucune ligne de labels.tsv ne declare ce vehi comme "+
				"porteur UNIQUE — regle inerte", r.Key)
			continue
		}
		// Un chassis SANS weapon_key perdrait sa ligne de statistiques : le kill retomberait
		// dans « Non attribue » alors qu il en avait une avant. Une icone gagnee ne doit pas
		// se payer d une mesure perdue.
		if r.WeaponKey == "" {
			t.Errorf("regle PORTEUR %q : weapon_key vide", r.Key)
			continue
		}
		if _, ok := reg[r.WeaponKey]; !ok {
			t.Errorf("regle PORTEUR %q : weapon_key %q absent du registre", r.Key, r.WeaponKey)
			continue
		}
		verifiees++
	}
	if verifiees == 0 {
		t.Fatal("aucune regle PORTEUR verifiee : le test ne prouve rien")
	}
}

func unVehiDeclareCePorteur(cle string) bool {
	for _, l := range damagetag.Labels() {
		if p, ok := uniquePorteur(l.Detail); ok && p == cle {
			return true
		}
	}
	return false
}

// TestPorteurPrimeBanque : la priorite, CONSTATEE sur le cas reel — pas re-implementee.
//
// Le temoin est la seconde moitie du test et il n est pas decoratif : la tourelle FIXE cite
// la MEME banque que le Warthog. Si elle basculait elle aussi sur la vignette du Hog, la
// regle PORTEUR aurait deborde de son porteur, et un test sans temoin serait reste vert.
func TestPorteurPrimeBanque(t *testing.T) {
	ic, ok := Lookup(tagWarthogLAAG)
	if !ok {
		t.Fatalf("tag %08x : aucune icone", uint32(tagWarthogLAAG))
	}
	if ic.Genre != GenrePorteur || ic.Sprite != "killfeed-26" {
		t.Errorf("Warthog %08x : %q par %q — attendu killfeed-26 par PORTEUR",
			uint32(tagWarthogLAAG), ic.Sprite, ic.Genre)
	}
	fixe, ok := Lookup(tagTourelleFixe)
	if !ok {
		t.Fatalf("tourelle fixe %08x : aucune icone", uint32(tagTourelleFixe))
	}
	if fixe.Genre != GenreBanque || fixe.Sprite != "killfeed-05" {
		t.Errorf("tourelle fixe %08x : %q par %q — attendu killfeed-05 par BANQUE (elle n a "+
			"pas de regle PORTEUR et doit garder la vignette generique)",
			uint32(tagTourelleFixe), fixe.Sprite, fixe.Genre)
	}
}

// TestPorteurRefuseUnePluraliteDeclaree : la garde du marqueur `+N`, point de rupture du genre.
//
// `racineUnique` ne suffit PAS ici : sur « vehi 003f00c7 +1 » la regex ne voit qu un seul tag
// et le declarerait unique, alors que la ligne annonce N+1 porteurs. Sans le rejet explicite,
// une regle PORTEUR deborderait en silence sur les autres.
func TestPorteurRefuseUnePluraliteDeclaree(t *testing.T) {
	if _, ok := uniquePorteur("ARME DE VEHICULE (vehi 003f00c7 +1, sb_010_tur_un_machinegun, classe turret)"); ok {
		t.Error("`vehi 003f00c7 +1` declare plusieurs porteurs : uniquePorteur doit refuser")
	}
	if _, ok := uniquePorteur("(vehi 3d4a8a5a, x) puis (vehi b857fb95, y)"); ok {
		t.Error("deux vehi differents sur la ligne : uniquePorteur doit refuser")
	}
	got, ok := uniquePorteur("ARME DE VEHICULE (vehi dd7f9102, sb_010_tur_un_machinegun, classe turret)")
	if !ok || got != "dd7f9102" {
		t.Errorf("porteur unique : (%q, %v) — attendu (dd7f9102, true)", got, ok)
	}
}

// chassisMongoose : le `vehi` du Mongoose, PARTAGE entre le Mongoose nu et le Gungoose.
const chassisMongoose = "000025aa"

// tagsGungooseConnus : les deux seules lignes de labels.tsv portees par ce chassis au
// 2026-09-10. Toutes deux sont des degats de PROJECTILE, et le Mongoose nu n a pas d arme.
var tagsGungooseConnus = map[uint32]bool{0x00426796: true, 0x00426797: true}

// TestChassisMongooseNePorteQueDesTagsGungoose : la contrepartie du seul compromis du lot.
//
// LE COMPROMIS. La regle Gungoose est indexee sur le chassis MONGOOSE, faute de mieux : le
// Gungoose n a pas de `vehi` propre, c est un Mongoose auquel une arme est accrochee. Elle
// n est donc juste que TANT QUE tout ce que porte ce chassis vient de cette arme — vrai
// aujourd hui (deux lignes, deux projectiles ; le Mongoose nu ne porte ni `uwfa` ni `scen`,
// CONTACT_ARMES_GUNGOOSE_2026-09-02 section 2), pas garanti demain.
//
// CE QUE CE TEST EMPECHE. Un troisieme tag apparaissant sur ce chassis — un ecrasement de
// Mongoose, typiquement — heriterait EN SILENCE de la vignette du Gungoose, et afficherait
// un Gungoose la ou un Mongoose a ecrase. Exactement l icone fausse que cette table existe
// pour interdire. Ici, il rougit.
func TestChassisMongooseNePorteQueDesTagsGungoose(t *testing.T) {
	vus := 0
	for _, l := range damagetag.Labels() {
		p, ok := uniquePorteur(l.Detail)
		if !ok || p != chassisMongoose {
			continue
		}
		vus++
		if !tagsGungooseConnus[l.Tag] {
			t.Errorf("tag %08x : NOUVEAU porteur sur le chassis Mongoose %s (%q). La regle "+
				"PORTEUR lui donnerait la vignette du Gungoose. Verifier ce qu il est AVANT "+
				"de l ajouter a tagsGungooseConnus : si ce n est pas l arme du Gungoose, la "+
				"regle doit etre reindexee, pas la liste allongee.",
				l.Tag, chassisMongoose, l.Detail)
		}
	}
	if vus == 0 {
		t.Fatal("aucune ligne portee par le chassis Mongoose : le test ne prouve rien")
	}
}
