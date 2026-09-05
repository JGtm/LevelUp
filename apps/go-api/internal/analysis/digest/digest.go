// Package digest — L'EMPREINTE D'UNE VALEUR GO : STABLE, SANS NOM DE TYPE, SANS CHAMP INVISIBLE.
//
// # POURQUOI PAS `encoding/json`
//
// Le harnais d'equivalence de la cuisson (PLAN_CUISSON_PERF §3 D4) hache la sortie de CHAQUE
// balayage pour prouver qu'un refacto ne change RIEN. Un digest JSON aurait trois trous, et
// chacun laisserait passer une regression :
//   - il ne voit que les champs EXPORTES — le struct embarque non exporte `componentDirs` de
//     `filmdec.BipedPosition` serait invisible, et une derive dedans passerait le gate ;
//   - `encoding/json` REFUSE NaN et les infinis : un film pathologique rendrait une erreur la
//     ou le harnais doit justement mesurer ;
//   - il porte les balises `json:"..."` et le nom des types : un renommage purement cosmetique
//     ferait rougir tous les digests figes, et l'operateur apprendrait a les regenerer sans
//     lire — exactement ce que ce gate doit empecher.
//
// # CE QUE LE RENDU GARANTIT
//
// Deux valeurs de meme forme rendent la MEME suite d'octets, quel que soit le nom du type qui
// les porte. Le rendu est une grammaire, pas du JSON : `{nom=valeur;...}` pour un struct
// (champs tries par nom, exportes ou non), `{cle=valeur;...}` pour une map (paires triees par le
// RENDU de la cle, PUIS par le rendu de la valeur — cf. ci-dessous), `[a,b]` pour une tranche.
// Les flottants passent par FormatFloat('g', -1, bits) : NaN, `+Inf` et `-Inf` se rendent au
// lieu d'echouer.
//
// L'INDEPENDANCE A L'ORDRE D'INSERTION D'UNE MAP N'EST PAS INCONDITIONNELLE, et la condition
// est ecrite ici parce qu'elle a ete violee en silence (mesure du 2026-09-02) : les paires se
// trient par le rendu de leur cle, or la grammaire ne porte pas les noms de type — deux cles
// DISTINCTES peuvent donc rendre les MEMES octets. `map[any]int{int32(1):100, int64(1):200}` en
// est un cas mesure (les deux cles rendent `1`), comme deux cles POINTEUR visant des valeurs
// egales (le rendu est celui de la valeur pointee). Le tri departage alors par le rendu de la
// VALEUR ; deux entrees dont la cle ET la valeur rendent les memes octets sont indiscernables
// PAR CONSTRUCTION, et l'ordre entre elles ne change pas le flux.
//
// # LE PREFIXE DE LONGUEUR, ET POURQUOI IL EST NON NEGOCIABLE
//
// Les delimiteurs seuls NE SUFFISENT PAS : une donnee qui contient le delimiteur imite une
// structure. Quatre collisions etaient CONSTRUCTIBLES avant le 2026-09-02, mesurees par la revue
// du lot 0 — `[]string{"a,b"}` rendait `[a,b]`, comme `[]string{"a","b"}` ; une map dont une cle
// portait `=` fusionnait avec sa voisine ; `[][]byte{{1,2},{3}}` rendait les memes octets que
// `[][]byte{{1,2,44,3}}` (44 = la virgule) ; et `[]byte("<nil>")` rendait le marqueur du nul.
// Un harnais d'equivalence qui accepte des collisions constructibles ne prouve rien.
//
// La grammaire porte donc, pour toute donnee de longueur variable IMBRIQUEE, un prefixe de
// longueur : une chaine s'ecrit `s:<len>:<octets>`, une tranche d'octets `b:<len>:<octets>`, et
// le nul `n:`. La longueur dit exactement combien d'octets suivent : plus aucune donnee ne peut
// se faire passer pour un delimiteur.
//
// # LE CAS SPECIAL DE LA RACINE : UNE TRANCHE D'OCTETS RESTE BRUTE
//
// Une tranche d'octets passee DIRECTEMENT a Of (profondeur 0) est hachee TELLE QUELLE, sans
// prefixe : c'est une propriete voulue et utilisee — l'empreinte de l'artefact de rejeu DOIT
// etre le sha256 de ses octets, celui qu'un operateur retrouve avec `sha256sum` sur le fichier
// range. La contrepartie est ecrite ici : a la racine, une tranche d'octets peut imiter
// n'importe quel rendu (elle EST des octets libres), et `[]byte(nil)` s'y confond avec
// `[]byte{}` — zero octet dans les deux cas. Des la profondeur 1, le prefixe s'applique et la
// confusion disparait.
//
// # LA MARCHE EST BORNEE EN PROFONDEUR
//
// La garde de cycle ne suit que les POINTEURS : `m := map[string]any{}; m["soi"] = m` n'en
// porte aucun et faisait deborder la pile. Au-dela de profondeurMaxRendu, le rendu ecrit
// `<profondeur-max>` et s'arrete — un digest, jamais un plantage.
//
// # LE RENDU EST VERSIONNE
//
// Toute evolution de cette grammaire rend les references figees incomparables. La version vit
// dans `GrammarVersion` (cf. grammar.go) et s'ecrit en tete de chaque fichier de digests :
// changer le rendu SANS la monter ferait lire une regression du decodeur la ou seule la
// grammaire a bouge.
//
// Paquet FEUILLE : il n'importe que la bibliotheque standard, et rien du depot.
package digest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"reflect"
	"sort"
	"strconv"
)

// profondeurMax borne la traversee des pointeurs par le comptage. Un type auto-referent
// (`type P *P` avec `p = &p`) la ferait boucler sans jamais rendre.
const profondeurMax = 64

// profondeurMaxRendu borne la MARCHE du rendu (cf. l'en-tete). Elle est large — aucune valeur
// legitime du decodeur n'imbrique mille niveaux — parce que son role n'est pas de filtrer mais
// d'empecher un debordement de pile sur une valeur auto-referente SANS pointeur.
const profondeurMaxRendu = 1000

// Les marqueurs du rendu. `marqueNil` porte les deux points comme les prefixes de longueur :
// c'est la meme grammaire, un mot-clef suivi de `:`, et aucune chaine ni tranche d'octets
// imbriquee ne peut l'imiter puisqu'elles portent la leur.
const (
	marqueNil        = "n:"
	marqueCycle      = "<cycle>"
	marqueProfondeur = "<profondeur-max>"
)

// Of rend le nombre d'elements de PREMIER NIVEAU de v et l'empreinte SHA-256 hexadecimale de
// son rendu canonique.
//
// count vaut la longueur pour une tranche, un tableau, une map ou une chaine — les pointeurs
// et les interfaces sont traverses —, 0 pour toute forme nulle, et 1 pour tout le reste.
func Of(v any) (count int, sum string) {
	rv := reflect.ValueOf(v)
	h := sha256.New()
	e := &encodeur{w: h, chemin: map[reference]bool{}}
	e.valeur(rv, 0)
	return compte(rv, 0), hex.EncodeToString(h.Sum(nil))
}

// reference identifie un pointeur DEJA OUVERT sur le chemin courant. Le type en fait partie :
// deux pointeurs de types differents partagent une adresse des que l'un vise le premier champ
// de l'autre, et les confondre inventerait un cycle qui n'existe pas.
type reference struct {
	ptr uintptr
	typ reflect.Type
}

// encodeur ecrit le rendu canonique d'une valeur dans w.
type encodeur struct {
	w io.Writer
	// chemin porte les pointeurs ouverts entre la racine et la valeur courante — pas ceux
	// deja refermes : un graphe qui repasse deux fois par le meme noeud sur DEUX branches
	// n'est pas un cycle, et se rend en entier.
	chemin map[reference]bool
}

// ecrire pousse une chaine dans le condensat.
//
// L'ERREUR EST ECARTEE ICI, ET SEULEMENT ICI : la cible est un [hash.Hash], dont le contrat
// dit que Write ne rend jamais d'erreur. Le sous-encodeur des cles de map ecrit dans un
// bytes.Buffer, qui n'en rend pas davantage.
func (e *encodeur) ecrire(s string) { _, _ = io.WriteString(e.w, s) }

// octets pousse des octets bruts dans le condensat (meme contrat que ecrire).
func (e *encodeur) octets(b []byte) { _, _ = e.w.Write(b) }

// valeur ecrit le rendu canonique de v. `prof` est sa profondeur d'imbrication : 0 pour la
// valeur passee a Of, et c'est cette profondeur-la qui donne son cas special a la tranche
// d'octets racine (cf. l'en-tete).
func (e *encodeur) valeur(v reflect.Value, prof int) {
	if prof > profondeurMaxRendu {
		// Valeur auto-referente sans pointeur (une map qui se contient) : on rend un marqueur
		// plutot que de deborder la pile.
		e.ecrire(marqueProfondeur)
		return
	}
	switch v.Kind() {
	case reflect.Invalid:
		e.ecrire(marqueNil)
	case reflect.Pointer:
		e.pointeur(v, prof)
	case reflect.Interface:
		// UNE INTERFACE EST TRANSPARENTE : c'est la valeur portee qui compte, jamais la boite.
		if v.IsNil() {
			e.ecrire(marqueNil)
			return
		}
		e.valeur(v.Elem(), prof+1)
	case reflect.Struct:
		e.structure(v, prof)
	case reflect.Map:
		e.table(v, prof)
	case reflect.Slice, reflect.Array:
		e.suite(v, prof)
	default:
		e.scalaire(v)
	}
}

// scalaire ecrit le rendu d'une valeur sans structure interne.
func (e *encodeur) scalaire(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		e.ecrire(strconv.FormatBool(v.Bool()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.ecrire(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.ecrire(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32:
		e.ecrire(flottant(v.Float(), 32))
	case reflect.Float64:
		e.ecrire(flottant(v.Float(), 64))
	case reflect.Complex64:
		e.complexe(v, 32)
	case reflect.Complex128:
		e.complexe(v, 64)
	case reflect.String:
		e.chaine(v.String())
	case reflect.Chan:
		e.opaque(v, "<chan>")
	case reflect.Func:
		e.opaque(v, "<func>")
	default:
		// UnsafePointer et tout ce qui viendrait apres : l'adresse est un alea de course, la
		// hacher rendrait le digest non reproductible. Seule la presence est retenue.
		e.ecrire("<opaque>")
	}
}

// chaine ecrit une chaine PREFIXEE DE SA LONGUEUR (`s:<len>:<octets>`). C'est ce prefixe qui
// interdit a une chaine de se faire passer pour un delimiteur de structure — sans lui,
// `[]string{"a,b"}` et `[]string{"a","b"}` ont la meme empreinte (cf. l'en-tete).
func (e *encodeur) chaine(s string) {
	e.ecrire("s:" + strconv.Itoa(len(s)) + ":" + s)
}

// flottant rend un flottant SANS jamais echouer : NaN, `+Inf` et `-Inf` sont des rendus
// ordinaires (c'est tout l'objet de ce paquet, cf. l'en-tete).
func flottant(f float64, bits int) string { return strconv.FormatFloat(f, 'g', -1, bits) }

// complexe rend un complexe comme un couple de flottants.
func (e *encodeur) complexe(v reflect.Value, bits int) {
	c := v.Complex()
	e.ecrire("(" + flottant(real(c), bits) + "," + flottant(imag(c), bits) + ")")
}

// opaque rend une valeur dont seule la nullite est reproductible.
func (e *encodeur) opaque(v reflect.Value, marque string) {
	if v.IsNil() {
		e.ecrire(marqueNil)
		return
	}
	e.ecrire(marque)
}

// pointeur suit le pointeur UNE fois ; un pointeur deja ouvert sur le chemin courant se rend
// `<cycle>` — c'est ce qui fait terminer une structure auto-referente PAR POINTEUR. Un cycle
// sans pointeur (une map qui se contient) est arrete par la borne de profondeur.
func (e *encodeur) pointeur(v reflect.Value, prof int) {
	if v.IsNil() {
		e.ecrire(marqueNil)
		return
	}
	ref := reference{ptr: v.Pointer(), typ: v.Type()}
	if e.chemin[ref] {
		e.ecrire(marqueCycle)
		return
	}
	e.chemin[ref] = true
	e.valeur(v.Elem(), prof+1)
	delete(e.chemin, ref)
}

// structure rend un struct : champs TRIES PAR NOM, exportes ou non.
//
// Les champs non exportes sont lus par reflexion SANS passer par Interface() — qui paniquerait
// dessus. C'est la raison d'etre du paquet : ce que `encoding/json` ne voit pas est justement
// ce qu'un refacto peut casser en silence.
func (e *encodeur) structure(v reflect.Value, prof int) {
	t := v.Type()
	rangs := make([]int, t.NumField())
	for i := range rangs {
		rangs[i] = i
	}
	sort.Slice(rangs, func(a, b int) bool {
		return t.Field(rangs[a]).Name < t.Field(rangs[b]).Name
	})
	e.ecrire("{")
	for n, i := range rangs {
		if n > 0 {
			e.ecrire(";")
		}
		// Le nom d'un champ est un IDENTIFIANT Go : il ne peut porter ni `;` ni `=`, donc il
		// n'a pas besoin d'un prefixe de longueur pour rester sans ambiguite.
		e.ecrire(t.Field(i).Name)
		e.ecrire("=")
		e.valeur(v.Field(i), prof+1)
	}
	e.ecrire("}")
}

// paire : une entree de map, le rendu de sa cle, et — SEULEMENT en cas d'egalite de cles — le
// rendu de sa valeur (cf. departager).
type paire struct {
	cle      string
	val      reflect.Value
	valRendu string
	rendue   bool
}

// table rend une map : paires TRIEES PAR LE RENDU DE LEUR CLE, puis departagees par le rendu de
// leur VALEUR quand deux cles rendent les memes octets.
//
// L'ordre d'iteration d'une map Go est volontairement aleatoire : sans ce tri, deux executions
// sur la MEME valeur rendraient deux digests, et le harnais crierait a la regression a chaque
// passage.
func (e *encodeur) table(v reflect.Value, prof int) {
	if v.IsNil() {
		e.ecrire(marqueNil)
		return
	}
	cles := v.MapKeys()
	paires := make([]paire, 0, len(cles))
	for _, k := range cles {
		paires = append(paires, paire{cle: e.rendu(k, prof+1), val: v.MapIndex(k)})
	}
	sort.Slice(paires, func(i, j int) bool { return paires[i].cle < paires[j].cle })
	e.departager(paires, prof+1)
	e.ecrire("{")
	for i, p := range paires {
		if i > 0 {
			e.ecrire(";")
		}
		e.ecrire(p.cle)
		e.ecrire("=")
		if p.rendue {
			// Rendu deja calcule par le departage, avec la MEME grammaire et la meme
			// profondeur : le reecrire evite de marcher la valeur une seconde fois.
			e.ecrire(p.valRendu)
			continue
		}
		e.valeur(p.val, prof+1)
	}
	e.ecrire("}")
}

// departager ordonne, par le rendu de leur VALEUR, les entrees dont les CLES rendent les memes
// octets. `paires` est deja trie par cle ; `prof` est la profondeur des valeurs.
//
// POURQUOI CE SECOND TOUR EXISTE (defaut mesure le 2026-09-02). La grammaire ne porte pas les
// noms de type : deux cles DISTINCTES peuvent rendre la meme chaine — `int32(1)` et `int64(1)`
// dans un `map[any]int`, ou deux cles pointeur visant des valeurs egales. Le tri par cle seule
// laissait alors ces entrees dans l'ordre ou `MapKeys` les avait servies, c'est-a-dire l'ordre
// ALEATOIRE d'iteration d'une map Go : la MEME valeur rendait DEUX empreintes.
//
// LE RENDU DE VALEUR N'EST CALCULE QUE POUR LES ENTREES EN CONFLIT — il coute un tampon par
// valeur, et le conflit est rare — puis il est REUTILISE a l'ecriture, jamais recalcule.
func (e *encodeur) departager(paires []paire, prof int) {
	for i := 0; i < len(paires); {
		j := i + 1
		for j < len(paires) && paires[j].cle == paires[i].cle {
			j++
		}
		if j-i > 1 {
			groupe := paires[i:j]
			for k := range groupe {
				groupe[k].valRendu = e.rendu(groupe[k].val, prof)
				groupe[k].rendue = true
			}
			// Deux entrees dont la cle ET la valeur rendent les memes octets restent
			// indiscernables : l'ordre entre elles ne change pas le flux.
			sort.Slice(groupe, func(a, b int) bool { return groupe[a].valRendu < groupe[b].valRendu })
		}
		i = j
	}
}

// suite rend une tranche ou un tableau. Les octets font exception : ils portent leurs octets
// eux-memes, prefixes de leur longueur — sauf a la RACINE, ou ils sont bruts (cf. l'en-tete).
func (e *encodeur) suite(v reflect.Value, prof int) {
	octets := v.Type().Elem().Kind() == reflect.Uint8
	// LE CAS SPECIAL DE LA RACINE : une tranche d'octets passee directement a Of EST l'artefact,
	// et son empreinte doit etre celle que `sha256sum` rend sur le fichier. Nil comprise : zero
	// octet est zero octet, il n'y a pas de marqueur a inventer par-dessus.
	if prof == 0 && octets && v.Kind() == reflect.Slice {
		e.octetsBruts(v, prof)
		return
	}
	if v.Kind() == reflect.Slice && v.IsNil() {
		e.ecrire(marqueNil)
		return
	}
	if octets {
		e.octetsBruts(v, prof)
		return
	}
	e.ecrire("[")
	for i := 0; i < v.Len(); i++ {
		if i > 0 {
			e.ecrire(",")
		}
		e.valeur(v.Index(i), prof+1)
	}
	e.ecrire("]")
}

// octetsBruts pousse les octets, precedes de `b:<len>:` des qu'ils sont IMBRIQUES. Sans ce
// prefixe, `[][]byte{{1,2},{3}}` et `[][]byte{{1,2,44,3}}` rendent la meme suite d'octets — 44
// est le code de la virgule qui separe les elements.
//
// La voie directe (Bytes) est refusee a une valeur issue d'un champ non exporte et aux
// tableaux non adressables : la recopie element par element est le repli, et elle rend
// exactement les memes octets.
func (e *encodeur) octetsBruts(v reflect.Value, prof int) {
	var buf []byte
	if v.Kind() == reflect.Slice && v.CanInterface() {
		buf = v.Bytes()
	} else {
		buf = make([]byte, v.Len())
		for i := range buf {
			buf[i] = byte(v.Index(i).Uint())
		}
	}
	if prof > 0 {
		e.ecrire("b:" + strconv.Itoa(len(buf)) + ":")
	}
	e.octets(buf)
}

// rendu rend le rendu canonique d'une valeur sous forme de chaine — pour les CLES de map, qui
// se trient par leur rendu. Le chemin des pointeurs ouverts est PARTAGE avec l'encodeur pere :
// une cle qui pointe vers son propre conteneur reste un cycle.
func (e *encodeur) rendu(v reflect.Value, prof int) string {
	var buf bytes.Buffer
	sous := &encodeur{w: &buf, chemin: e.chemin}
	sous.valeur(v, prof)
	return buf.String()
}

// compte rend le nombre d'elements de premier niveau, pointeurs et interfaces traverses.
func compte(v reflect.Value, profondeur int) int {
	if profondeur > profondeurMax {
		return 1
	}
	switch v.Kind() {
	case reflect.Invalid:
		return 0
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return 0
		}
		return compte(v.Elem(), profondeur+1)
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			return 0
		}
		return v.Len()
	case reflect.Array, reflect.String:
		return v.Len()
	case reflect.Chan, reflect.Func:
		if v.IsNil() {
			return 0
		}
		return 1
	default:
		return 1
	}
}
