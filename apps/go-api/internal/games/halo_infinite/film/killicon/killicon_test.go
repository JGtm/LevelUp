package killicon

import (
	"os"
	"path/filepath"
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
	for _, l := range damagetag.Labels() {
		if l.Name != "" {
			noms[l.Name] = true
		}
		classes[string(l.Class)] = true
		if m := ggglRe.FindStringSubmatch(l.Detail); m != nil {
			gggl[m[1]] = true
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
		damagetag.ClassArme:    {114, 105},
		damagetag.ClassMelee:   {14, 14},
		damagetag.ClassGrenade: {15, 15},
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
	// Les classes sans regle ne doivent JAMAIS recevoir d icone (vehicule, bidon, chute,
	// inconnu) : servir l image d un chassis au hasard serait le defaut que ce lot evite.
	for _, cl := range []damagetag.Class{
		damagetag.ClassVehicule, damagetag.ClassObjet, damagetag.ClassGlobal, damagetag.ClassInconnu,
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
