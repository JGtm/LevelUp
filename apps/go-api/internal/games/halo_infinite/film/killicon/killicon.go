// Package killicon fait LE PONT entre la source de degat d une mort et la vignette de
// l atlas KILL FEED qui la represente. Il est 100 % HORS LIGNE : il ne lit que deux
// tables versionnees et embarquees (`data/rules.tsv` ici, `damagetag/data/labels.tsv`
// a cote), jamais le jeu, jamais le disque, jamais la base.
//
// CE QU IL RESOUT. Le decodeur de film rend, par mort, un tag `jpt!`
// (`match_kill_events.source_tag`). `damagetag` le traduit deja en CLASSE + NOM a 97,6 %.
// Il manquait le dernier maillon : quelle IMAGE montrer. C est ce paquet.
//
// POURQUOI LA RESOLUTION EST INDEXEE PAR TAG. Les regles s ECRIVENT par nom (c est la
// seule quantite qui traverse toutes les lignes de labels.tsv : 11 lignes de classe ARME
// sur 114 portent un tag `weap`, les autres portent un `proj`, un `hlmt` ou un `weme`),
// mais elles se RESOLVENT une fois pour toutes a l initialisation, en une table
// `tag -> vignette`. Consequence : aucune comparaison de chaine au moment de servir une
// page, et aucune sensibilite d execution aux divergences de nommage (le depot en porte
// une, connue : << Mk51 Sidekick >> en base contre << Mk50 Sidekick >> au registre).
//
// LA GARANTIE. Une source dont le nom est une alternative (<< A / B >>) n obtient une
// vignette QUE si toutes ses branches designent la meme. Sinon elle n en obtient AUCUNE :
// le produit se replie sur le libelle. Une icone absente est un repli, une icone fausse
// est un mensonge — et un mensonge sur l arme d un kill est indetectable a l oeil.
//
// PEREMPTION. Comme `damagetag`, la table GRANDIT a chaque saison. Une regle dont le nom
// ne correspond plus a aucune ligne de labels.tsv fait ECHOUER la suite de tests
// (`TestChaqueRegleTrouveSaSource`) : la peremption se paie en rouge, jamais en silence.
package killicon

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
)

//go:embed data/rules.tsv
var rawRules string

// Genre : sur quelle quantite une regle s applique.
type Genre string

// Les trois genres publies, par priorite decroissante de resolution.
const (
	// GenreNom : la cle est le champ `nom` de labels.tsv, verbatim.
	GenreNom Genre = "NOM"
	// GenreGGGL : la cle est l entree de la liste des grenades du jeu (`gggl entree N/4`).
	GenreGGGL Genre = "GGGL"
	// GenreBanque : la cle est la racine de banque sonore citee par le champ detail
	// (`sb_010_veh_cv_ghost` -> `veh_cv_ghost`). C est ainsi que le CHASSIS d un vehicule
	// se nomme : la classe VEHICULE ne porte pas de nom propre, mais 55 de ses 89 lignes
	// citent UNE racine, et une seule. Ne s applique JAMAIS quand la ligne en cite
	// plusieurs — un effet partage par le Chopper, la Banshee et le Wasp ne designe aucun
	// des trois.
	GenreBanque Genre = "BANQUE"
	// GenreClasse : la cle est la classe damagetag. Repli le plus large.
	GenreClasse Genre = "CLASSE"
)

// Icon : la vignette qui represente une source de degat.
type Icon struct {
	// Sprite : stem du PNG sous static/weapons-assets/{slug}/jeu/, atlas kill feed.
	Sprite string
	// WeaponKey : cle canonique du registre d armes ("" si l objet n y figure pas —
	// Mutilator et Sandwich sont dans ce cas, mesure, pas oubli).
	WeaponKey string
	// Genre : par quelle regle la vignette a ete obtenue. Sert aux garde-rails et a
	// l artefact de revue, jamais a une branche de comportement produit.
	Genre Genre
}

// Rule : une ligne de `data/rules.tsv`.
type Rule struct {
	Genre         Genre
	Key           string
	Sprite        string
	WeaponKey     string
	Justification string
}

// Provenance : d ou vient la table embarquee.
type Provenance struct {
	Date        string
	RuleCount   int
	IconedTags  int
	AnnouncedNb int // valeur `regles=` de l en-tete, pour le garde-rail de coherence
}

var (
	rules    []Rule
	byTag    map[uint32]Icon
	provided Provenance
)

// ggglRe extrait l entree de la liste des grenades du champ `detail` de labels.tsv.
// La forme AMBIGUE (<< entrees `gggl` atteintes : 0+1 sur 4 >>) ne matche PAS, et c est
// voulu : plusieurs entrees atteintes = aucune grenade designable.
var ggglRe = regexp.MustCompile(`gggl entree (\d+)/`)

// banqueRe extrait les racines de banque sonore citees par le champ `detail`
// (`sb_010_veh_cv_ghost` -> `veh_cv_ghost`). Le prefixe `sb_NNN_` est un numero de
// paquet, pas une information : deux paquets peuvent porter la meme racine.
var banqueRe = regexp.MustCompile(`sb_\d+_([a-z]+_[a-z]+_[a-z0-9]+)`)

// uniqueBanque rend la racine de banque d une ligne, et SEULEMENT si elle est unique.
//
// C est toute la surete de ce genre de regle. Une ligne comme
// « vehi 3d4a8a5a, veh_bt_chopper / veh_cv_banshee / veh_cv_ghost / veh_un_wasp » decrit un
// effet PARTAGE par quatre chassis : lui donner l icone de l un des quatre serait faux trois
// fois sur quatre. Meme regle que pour les noms alternatifs << A / B >>.
func uniqueBanque(detail string) (string, bool) {
	seen := ""
	for _, m := range banqueRe.FindAllStringSubmatch(detail, -1) {
		if seen == "" {
			seen = m[1]
			continue
		}
		if m[1] != seen {
			return "", false
		}
	}
	return seen, seen != ""
}

func init() {
	var err error
	if rules, provided.Date, provided.AnnouncedNb, err = parseRules(rawRules); err != nil {
		panic("killicon: data/rules.tsv illisible: " + err.Error())
	}
	provided.RuleCount = len(rules)
	byTag = resolve(rules, damagetag.Labels())
	provided.IconedTags = len(byTag)
}

// Lookup : la vignette d un tag `jpt!`, si le pont en connait une.
//
// Faux en second retour = repli assume : le produit affiche le libelle sans icone. Ce
// n est PAS une erreur, c est le comportement nominal pour un vehicule, un bidon, une
// chute, un tag inconnu ou un nom alternatif dont les branches divergent.
func Lookup(tag uint32) (Icon, bool) {
	ic, ok := byTag[tag]
	return ic, ok
}

// Rules : les regles embarquees, dans l ordre du fichier. La tranche est une copie.
func Rules() []Rule { return append([]Rule(nil), rules...) }

// ResolvedTags : tous les tags qui obtiennent une vignette, tries. Copie.
func ResolvedTags() []uint32 {
	out := make([]uint32, 0, len(byTag))
	for t := range byTag {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Source : la provenance de la table embarquee.
func Source() Provenance { return provided }

// ─────────────────────────────────────────────────────────────────────────────
// RESOLUTION
// ─────────────────────────────────────────────────────────────────────────────

// resolve croise les regles et les etiquettes et rend la table `tag -> vignette`.
//
// Priorite NOM > GGGL > CLASSE : la premiere qui repond gagne. Une etiquette non
// publiable (statut AMBIGU, ou INCONNU sans nom) n obtient jamais de vignette — le
// meme critere que celui qui interdit de publier son libelle.
func resolve(rs []Rule, labels []damagetag.Label) map[uint32]Icon {
	idx := map[Genre]map[string]Rule{
		GenreNom: {}, GenreGGGL: {}, GenreBanque: {}, GenreClasse: {},
	}
	for _, r := range rs {
		if m, ok := idx[r.Genre]; ok {
			m[r.Key] = r
		}
	}
	out := make(map[uint32]Icon, len(labels))
	for _, l := range labels {
		if !l.Publishable() {
			continue
		}
		if r, ok := matchRule(l, idx); ok {
			out[l.Tag] = Icon{Sprite: r.Sprite, WeaponKey: r.WeaponKey, Genre: r.Genre}
		}
	}
	return out
}

// matchRule applique les quatre genres dans l ordre de priorite : le nom propre d abord
// (le plus specifique), puis les deux quantites qui designent un objet precis sans le
// nommer, puis la classe (le plus large).
func matchRule(l damagetag.Label, idx map[Genre]map[string]Rule) (Rule, bool) {
	if l.Name != "" {
		if r, ok := idx[GenreNom][l.Name]; ok {
			return r, true
		}
	}
	if m := ggglRe.FindStringSubmatch(l.Detail); m != nil {
		if r, ok := idx[GenreGGGL][m[1]]; ok {
			return r, true
		}
	}
	if racine, ok := uniqueBanque(l.Detail); ok {
		if r, ok := idx[GenreBanque][racine]; ok {
			return r, true
		}
	}
	if r, ok := idx[GenreClasse][string(l.Class)]; ok {
		return r, true
	}
	return Rule{}, false
}

// ─────────────────────────────────────────────────────────────────────────────
// LECTURE DU FICHIER EMBARQUE
// ─────────────────────────────────────────────────────────────────────────────

const ruleColumns = 5

// headerInts lit `date=` et `regles=` d une ligne de commentaire d en-tete.
func headerInts(line string) (date string, count int) {
	for _, f := range strings.Fields(line) {
		if v, ok := strings.CutPrefix(f, "date="); ok {
			date = v
		}
		if v, ok := strings.CutPrefix(f, "regles="); ok {
			if n, err := strconv.Atoi(v); err == nil {
				count = n
			}
		}
	}
	return date, count
}

func parseRules(raw string) ([]Rule, string, int, error) {
	var out []Rule
	date, announced := "", 0
	for n, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if d, c := headerInts(line); d != "" || c != 0 {
				if d != "" {
					date = d
				}
				if c != 0 {
					announced = c
				}
			}
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != ruleColumns {
			return nil, "", 0, fmt.Errorf("ligne %d: %d colonnes au lieu de %d", n+1, len(f), ruleColumns)
		}
		if f[0] == "genre" { // ligne d en-tete de colonnes
			continue
		}
		r := Rule{Genre: Genre(f[0]), Key: f[1], Sprite: f[2], WeaponKey: f[3], Justification: f[4]}
		if err := validate(r, n+1); err != nil {
			return nil, "", 0, err
		}
		out = append(out, r)
	}
	return out, date, announced, nil
}

// validate refuse a la LECTURE ce qu aucun test ne rattraperait a l usage : une regle
// sans vignette ne dit rien, une regle sans justification n est pas auditable.
func validate(r Rule, line int) error {
	switch r.Genre {
	case GenreNom, GenreGGGL, GenreBanque, GenreClasse:
	default:
		return fmt.Errorf("ligne %d: genre %q inconnu", line, r.Genre)
	}
	if r.Key == "" || r.Sprite == "" {
		return fmt.Errorf("ligne %d: cle ou sprite vide", line)
	}
	if r.Justification == "" {
		return fmt.Errorf("ligne %d: justification vide (regle non auditable)", line)
	}
	return nil
}
