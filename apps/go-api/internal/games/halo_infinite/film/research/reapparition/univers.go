//go:build research

package reapparition

// univers.go — L'UNIVERS DES NOMS DE COMPOSANT, BALAYE DANS L'IMAGE.
//
// POURQUOI UN NEGATIF A BESOIN DE CECI. Dire « le film n'ecrit pas le minuteur de reapparition
// d'un vehicule » a partir de `ecs_table.tsv` ne prouve rien : la table ne connait que les
// archetypes rencontres dans les films decodes. Le balayage de l'IMAGE, lui, rend TOUS les noms
// de composant que le moteur embarque, qu'un film les ait portes ou non. Un vocabulaire absent
// de cet univers est absent du moteur, pas seulement du corpus.
//
// LA DEFINITION EST ETROITE ET ECRITE : une chaine C isolee d'une section de donnees, faite de
// minuscules, de chiffres et de tirets, terminee par `-component` ou par `-component-<n>` (les
// instances multiples que la note de methode decrit, § 6). Elle laisse dehors les noms ECS
// ecrits sans le suffixe (`biped-action`, `simulation-state`) : le rapport le dit, et le
// vocabulaire se cherche alors dans le pool complet des chaines, pas ici.

import (
	"sort"
	"strings"
)

const suffixeComposant = "-component"

// NomsDeComposants rend, triee et dedupliquee, la liste des noms de composant de l'image.
func (e *Executable) NomsDeComposants() []string {
	vus := map[string]bool{}
	for i := range e.sections {
		s := &e.sections[i]
		if s.execCode {
			continue
		}
		c := s.contenu
		deb := 0
		for off := 0; off < len(c); off++ {
			if c[off] != 0 {
				continue
			}
			if n := off - deb; n >= len(suffixeComposant) {
				if mot := string(c[deb:off]); estNomDeComposant(mot) {
					vus[mot] = true
				}
			}
			deb = off + 1
		}
	}
	out := make([]string, 0, len(vus))
	for n := range vus {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// estNomDeComposant applique la definition ecrite en tete de fichier.
func estNomDeComposant(s string) bool {
	if len(s) < len(suffixeComposant)+2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
		if !ok {
			return false
		}
	}
	if finit(s, suffixeComposant) {
		return true
	}
	// forme instanciee : `...-component-<n>`
	j := len(s)
	for j > 0 && s[j-1] >= '0' && s[j-1] <= '9' {
		j--
	}
	if j == len(s) || j == 0 || s[j-1] != '-' {
		return false
	}
	return finit(s[:j-1], suffixeComposant)
}

func finit(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// ---------------------------------------------------------------------------
// LE POOL COMPLET DES CHAINES — ce que l'en-tete de ce fichier annonce.
//
// `NomsDeComposants` ne rend que les noms en `-component`. Un vocabulaire de MOTEUR
// (« sprint », « slide », « crouch », « jump », « clamber ») n'y figure pas forcement : il vit
// dans des noms de tag, de propriete d'animation, de script de mode, ou nulle part. Le lot 5.3
// a besoin de la forme FORTE du negatif — « ce mot n'est dans AUCUNE chaine de l'image » — et
// c'est ce que ce balayage mesure.
// ---------------------------------------------------------------------------

// Chaine est une chaine C isolee d'une section de donnees, avec son adresse virtuelle.
type Chaine struct {
	VA    uint64
	Texte string
}

// ChainesContenant rend, par mot cherche, les chaines de l'image qui le contiennent. La
// comparaison est faite en minuscules des deux cotes : le moteur ecrit indifferemment
// `biped-slide-component`, `SlideState` et `slide_speed`.
//
// La definition d'une chaine est ETROITE ET ECRITE : au moins `minLongueurChaine` octets
// imprimables ASCII, terminee par un NUL, dans une section SANS code. Elle laisse dehors les
// chaines larges (UTF-16) — un mot absent ici peut vivre en UTF-16 ; le rapport le dit.
func (e *Executable) ChainesContenant(mots []string) map[string][]Chaine {
	bas := make([]string, len(mots))
	for i, m := range mots {
		bas[i] = strings.ToLower(m)
	}
	out := map[string][]Chaine{}
	for _, m := range bas {
		out[m] = nil
	}
	for i := range e.sections {
		s := &e.sections[i]
		if s.execCode {
			continue
		}
		e.balayerSection(s, bas, out)
	}
	for m := range out {
		sort.Slice(out[m], func(a, b int) bool { return out[m][a].Texte < out[m][b].Texte })
	}
	return out
}

const minLongueurChaine = 4

// balayerSection decoupe une section en chaines C imprimables et classe chacune sous les mots
// qu'elle contient. Une meme chaine peut repondre a plusieurs mots : elle est classee sous
// chacun, car le rapport se lit mot par mot.
func (e *Executable) balayerSection(s *section, mots []string, out map[string][]Chaine) {
	c := s.contenu
	deb := 0
	for off := 0; off <= len(c); off++ {
		if off < len(c) && estImprimable(c[off]) {
			continue
		}
		if off-deb >= minLongueurChaine {
			texte := string(c[deb:off])
			bas := strings.ToLower(texte)
			for _, m := range mots {
				if strings.Contains(bas, m) {
					out[m] = append(out[m], Chaine{VA: s.va + uint64(deb), Texte: texte})
				}
			}
		}
		deb = off + 1
	}
}

// estImprimable garde l'ASCII lisible, espace compris : un nom de tag du moteur y tient
// entierement, et le NUL comme tout octet binaire coupe la chaine.
func estImprimable(b byte) bool { return b >= 0x20 && b < 0x7f }
