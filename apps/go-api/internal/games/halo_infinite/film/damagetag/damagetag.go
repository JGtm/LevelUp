// Package damagetag porte, EMBARQUEES DANS LE BINAIRE, les deux tables qui permettent de
// decoder la source de degat d un kill dans un film Theater Halo Infinite sans que la machine
// ait le jeu installe.
//
// POURQUOI CE PAQUET EXISTE. Le decodeur lit, dans le composant dead-state de la victime, un
// identifiant de tag `jpt!` (damage_effect). Deux tables lui sont necessaires :
//
//  1. le CATALOGUE des ids `jpt!` du jeu, qui sert de test d acceptation au scan direct
//     (un candidat dont le tag n est pas un `jpt!` est rejete ; faux positif mesure
//     ~1 sur 9 000 000 pour les valeurs >= 0x10000) ;
//  2. la TABLE `tag -> nom d arme / classe`, qui transforme cet identifiant en etiquette.
//
// Les deux se CALCULAIENT en lisant les archives `.module` du jeu (132 archives, 37 Go) et les
// banques Wwise. Le serveur de production n a pas Halo installe : elles sont donc PRECALCULEES,
// versionnees dans `data/`, et embarquees par `go:embed`. Ce paquet ne touche JAMAIS le disque
// du jeu, ni aucun cache temporaire.
//
// REGENERATION — QUAND, OU, COMMENT, ET COMMENT VERIFIER.
//
// QUAND. A chaque mise a jour de contenu du jeu (nouvelle saison, nouvelle arme, nouveau
// vehicule). Ces tables GRANDISSENT : mesure comparable sur la table des medailles, 118
// identifiants sur 167 en mars, 121 sur 167 en juillet. Un tag inconnu du catalogue est
// INVISIBLE au scan (le test d acceptation le rejette), donc une table perimee se paie en
// COUVERTURE, silencieusement.
//
// OU. Sur une machine qui a Halo Infinite INSTALLE — la generation lit les en-tetes des
// archives `Content/deploy/**/*.module` et les noms de banques `Content/Sound/win/**/*.pck`.
// Jamais sur le serveur de production, qui n a pas le jeu. Il faut aussi cgo (decompression
// Kraken) : `CGO_ENABLED=1` et le compilateur C sur le PATH.
//
// COMMENT (depuis `apps/go-api`) :
//
//	go run ./cmd/tmp_tagname pojptgame   # 1. le catalogue embarque est-il perime, et de combien ?
//	go run ./cmd/tmp_tagname pogen       # 2. reecrit data/jpt_ids.txt et data/labels.tsv
//	go test ./internal/games/halo_infinite/film/damagetag/...   # 3. ancres Theater + repartition
//	go run ./cmd/tmp_tagname poscan all  # 4. non-regression sur les 4 films de reference
//
// L etape 4 doit rendre, au chiffre pres : 363 apparies, 14 rejetes, gate (b) 96.3 %,
// couverture 363/372 = 97.6 % (RE_LOG 7ter.60 (C4), verifie 7ter.61). REJOUE LE 2026-07-28 :
// `poscan all` rend TOUJOURS ces quatre chiffres, a l unite.
//
// CE CHIFFRE EST CELUI DE L OUTIL DE RE, PAS CELUI DU PAQUET `killsource`, ET IL NE FAUT
// SURTOUT PAS LE CORRIGER EN 371/371. Les deux instruments sont distincts et n ont pas recu les
// memes correctifs : `cmd/tmp_tagname poscan` est reste a la ligne de base de 7ter.60, tandis que
// le paquet a recu depuis les morts auto-infligees (7ter.63), la correction de denominateur
// (7ter.66) et les morts infligees PAR un bot (7ter.79) - il rend, lui, 371/371 couples REELS et
// 380/380 morts de l API. MEME PIEGE QUE LA RESERVE 7ter.74bis (outil 224 lignes / paquet 225 en
// BTB). Ici 363/372 est la BONNE valeur attendue : cette etape teste la PEREMPTION DU CATALOGUE a
// instrument constant, et un ecart signale un catalogue regenere de travers, rien d autre.
//
// PREUVE QUE LE DECODAGE EST BIEN OFFLINE. Renommer `Content/deploy` et `Content/Sound`, puis
// rejouer `poscan all` et `polabel all` : les sorties doivent etre IDENTIQUES AU BIT, et
// `pojptgame` doit echouer. C est fait, et les deux sorties ont le meme SHA-256.
//
// CE QUE LE STATUT VEUT DIRE, ET POURQUOI IL EST OBLIGATOIRE DE LE REGARDER :
//
//	VALIDE        etiquette publiable telle quelle.
//	SOUS_RESERVE  la CLASSE est sure, le NOM PROPRE ne l est pas : un nom precis peut designer
//	              une autre arme de la meme classe. C est le statut de toutes les armes nommees.
//	AMBIGU        NE PAS PUBLIER. L etiquette serait affirmative et fausse (effet generique
//	              partage par plusieurs entrees, dont on ne peut pas designer laquelle).
//	INCONNU       le tag est un `jpt!` valide mais aucune source ne lui est remontee.
package damagetag

import (
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed data/jpt_ids.txt
var rawIDs string

//go:embed data/labels.tsv
var rawLabels string

// Class : la nature de la source de degat.
type Class string

// Les classes publiees. Elles correspondent aux familles etablies par le chantier RE.
const (
	ClassArme     Class = "ARME"           // arme tenue en main, nommee
	ClassMelee    Class = "MELEE"          // effet partage par tout l arsenal
	ClassGrenade  Class = "GRENADE"        // projectile d un couple de la liste `gggl`
	ClassVehicule Class = "VEHICULE"       // armement declare par un chassis
	ClassObjet    Class = "OBJET_EXPLOSIF" // objet de decor porte qui blesse en explosant
	ClassGlobal   Class = "DEGAT_GLOBAL"   // chute, environnement, hors-limites
	ClassInconnu  Class = "INCONNU"        // `jpt!` valide, aucune source remontee
)

// Status : ce que le consommateur a le droit de faire de la ligne.
type Status string

// Les statuts publies.
const (
	StatusValide  Status = "VALIDE"
	StatusReserve Status = "SOUS_RESERVE"
	StatusAmbigu  Status = "AMBIGU"
	StatusInconnu Status = "INCONNU"
)

// Label : l etiquette precalculee d un tag `jpt!`.
type Label struct {
	Tag     uint32
	Class   Class
	Status  Status
	Name    string // nom d arme, vide pour les classes qui n en portent pas
	Detail  string // qualification mesuree (entree de grenade, chassis, etat de degat, effet)
	Reserve string // ce que l etiquette NE dit PAS ; vide si aucune reserve
}

// Publishable : la ligne peut-elle etre affichee a un utilisateur ?
func (l Label) Publishable() bool { return l.Status == StatusValide || l.Status == StatusReserve }

// Provenance : d ou viennent les tables embarquees.
type Provenance struct {
	IDsDate    string
	IDsCount   int
	LabelsDate string
	LabelCount int
}

var (
	catalog  map[uint32]struct{}
	ids      []uint32
	labels   map[uint32]Label
	provided Provenance
)

func init() {
	var err error
	if catalog, ids, provided.IDsDate, err = parseIDs(rawIDs); err != nil {
		panic("damagetag: data/jpt_ids.txt illisible: " + err.Error())
	}
	if labels, provided.LabelsDate, err = parseLabels(rawLabels); err != nil {
		panic("damagetag: data/labels.tsv illisible: " + err.Error())
	}
	provided.IDsCount = len(ids)
	provided.LabelCount = len(labels)
}

// IsDamageEffect : `tag` est-il une entree du groupe `jpt!` du jeu ? C est le test
// d acceptation du scan direct.
func IsDamageEffect(tag uint32) bool {
	_, ok := catalog[tag]
	return ok
}

// Strong : le regime ou le taux de faux positif mesure (~1/9 000 000) s applique. Sous 0x10000
// il vaut 1.15 % — trois tags REELS y vivent, donc on ne peut pas exclure ce regime du scan,
// mais on ne doit jamais melanger les deux dans une mesure.
func Strong(tag uint32) bool { return tag >= 0x10000 }

// IDs : le catalogue, trie. La tranche rendue est une copie.
func IDs() []uint32 { return append([]uint32(nil), ids...) }

// Size : nombre d ids `jpt!` embarques.
func Size() int { return len(ids) }

// Lookup : l etiquette precalculee d un tag, si elle existe.
func Lookup(tag uint32) (Label, bool) {
	l, ok := labels[tag]
	return l, ok
}

// Labels : TOUTES les etiquettes, triees par tag. La tranche rendue est une copie.
//
// Sert aux tables DERIVEES de celle-ci (pont vers les icones du kill feed : paquet
// `killicon`) : elles doivent enumerer les etiquettes pour s indexer par tag a
// l initialisation, sans relire le fichier embarque une seconde fois — deux lectures,
// c est deux verites.
func Labels() []Label {
	out := make([]Label, 0, len(labels))
	for _, l := range labels {
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

// Source : la provenance des tables embarquees.
func Source() Provenance { return provided }

// ─────────────────────────────────────────────────────────────────────────────
// LECTURE DES FICHIERS EMBARQUES
// ─────────────────────────────────────────────────────────────────────────────

// headerDate : la valeur `date=` d une ligne de commentaire d en-tete, ou "".
func headerDate(line string) string {
	for _, f := range strings.Fields(line) {
		if v, ok := strings.CutPrefix(f, "date="); ok {
			return v
		}
	}
	return ""
}

func parseIDs(raw string) (map[uint32]struct{}, []uint32, string, error) {
	set := map[uint32]struct{}{}
	var list []uint32
	date := ""
	for n, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if d := headerDate(line); d != "" {
				date = d
			}
			continue
		}
		v, err := strconv.ParseUint(line, 16, 32)
		if err != nil {
			return nil, nil, "", fmt.Errorf("ligne %d: %q: %w", n+1, line, err)
		}
		if _, dup := set[uint32(v)]; dup {
			continue
		}
		set[uint32(v)] = struct{}{}
		list = append(list, uint32(v))
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })
	return set, list, date, nil
}

func parseLabels(raw string) (map[uint32]Label, string, error) {
	out := map[uint32]Label{}
	date := ""
	for n, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			if d := headerDate(line); d != "" {
				date = d
			}
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 6 {
			return nil, "", fmt.Errorf("ligne %d: %d colonnes au lieu de 6", n+1, len(f))
		}
		v, err := strconv.ParseUint(f[0], 16, 32)
		if err != nil {
			return nil, "", fmt.Errorf("ligne %d: tag %q: %w", n+1, f[0], err)
		}
		out[uint32(v)] = Label{Tag: uint32(v), Class: Class(f[1]), Status: Status(f[2]),
			Name: f[3], Detail: f[4], Reserve: f[5]}
	}
	return out, date, nil
}
