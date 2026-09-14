package fallback

// compteur.go — LE COMPTE DES DÉCLENCHEMENTS, PAR CUISSON.
//
// # PAR CUISSON, ET NON PAR PROCESSUS
//
// Un compteur de paquet mélangerait deux films décodés en parallèle et ferait mentir les deux.
// Le critère S2 du plan exige précisément cela (« deux films décodés en parallèle sous `-race`
// sans course »), et le critère S1 supprime les variables de paquet mutables du décodeur : un
// compteur global serait une dette posée à l'endroit exact où le chantier en retire.
//
// Le compteur se crée donc par cuisson, voyage avec elle, et son rapport est publié dans le
// document que cette cuisson produit (`coverage.fallbacks[]`).
//
// # NIL EST UN COMPTEUR VALIDE
//
// Toutes les méthodes acceptent un récepteur nil et ne font rien. C'est ce qui rend
// l'instrumentation d'un site SANS RISQUE : un appelant qui ne fournit pas de compteur (un test
// unitaire, un outil de diagnostic, un chemin de cuisson qui n'en porte pas encore) traverse le
// site exactement comme avant. Le lot 1.9.0 est à équivalence zéro différence ; un compteur qui
// paniquerait sur nil l'aurait rendu impossible.
//
// # POURQUOI UN VERROU ET PAS `atomic`
//
// La table est une `map` : elle n'est pas sûre en écriture concurrente, et un compteur par
// repli en `atomic` demanderait une table figée à la compilation. Le verrou coûte une paire
// d'instructions sur un chemin qui se déclenche au plus quelques milliers de fois par film,
// face à un décodage qui pèse des dizaines de secondes.

import (
	"log/slog"
	"sort"
	"sync"
)

// Compteur compte les déclenchements de replis d'UNE cuisson. Le zéro n'est pas utilisable :
// passer par [NouveauCompteur], ou laisser nil (qui ne compte rien).
type Compteur struct {
	mu sync.Mutex
	n  map[Nom]int
}

// NouveauCompteur rend un compteur vide, prêt à recevoir des déclenchements.
func NouveauCompteur() *Compteur { return &Compteur{n: map[Nom]int{}} }

// Declenche note UN déclenchement du repli `nom`. Sûr sur un récepteur nil.
//
// UN NOM HORS REGISTRE EST UNE ERREUR JOURNALISÉE, PAS UNE PANIQUE ET PAS UN SILENCE. Le
// décodage d'un film ne s'interrompt pas pour un compteur ; mais taire l'écart laisserait un
// repli se compter sous un nom que personne ne pourrait relier à une entrée — exactement le
// repli anonyme que ce paquet existe pour interdire. Le garde-rail `archlint` attrape le cas
// à la compilation du test, celui-ci le rattrape à l'exécution.
func (c *Compteur) Declenche(nom Nom) { c.DeclencheN(nom, 1) }

// DeclencheN note `k` déclenchements d'un coup — pour un site qui compte une population après
// coup (une boucle qui rejette N éléments) plutôt qu'un par un. `k <= 0` ne fait rien.
func (c *Compteur) DeclencheN(nom Nom, k int) {
	if c == nil || k <= 0 {
		return
	}
	if _, ok := Lire(nom); !ok {
		slog.Error("repli hors registre declenche — le compte ne se rattache a aucune entree",
			"repli", string(nom), "declenchements", k)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.n == nil {
		c.n = map[Nom]int{}
	}
	c.n[nom] += k
}

// Compte rend le nombre de déclenchements d'un repli sur cette cuisson. Zéro sur un récepteur
// nil, comme sur un repli jamais déclenché — c'est pourquoi [Repli.CompteurBranche] existe.
func (c *Compteur) Compte(nom Nom) int {
	if c == nil {
		return 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n[nom]
}

// Declenchement est une ligne du rapport : un repli et son compte.
type Declenchement struct {
	// Nom : l'identifiant stable du repli, tel qu'il se relit au registre.
	Nom Nom
	// Declenchements : combien de fois il s'est déclenché sur cette cuisson.
	Declenchements int
}

// Rapport rend les replis DÉCLENCHÉS, triés par nom.
//
// LES ZÉROS N'Y SONT PAS, et c'est la forme la plus simple qui reste lisible : un artefact qui
// porterait les ~100 lignes du registre à zéro pèserait le même poids sur tous les films et
// n'apprendrait rien. Ce qui se lit dans un document, c'est ce qui S'EST déclenché ; ce que le
// registre porte, c'est ce qui PEUT se déclencher. Les deux se croisent par le nom.
func (c *Compteur) Rapport() []Declenchement {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Declenchement, 0, len(c.n))
	for nom, n := range c.n {
		if n > 0 {
			out = append(out, Declenchement{Nom: nom, Declenchements: n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nom < out[j].Nom })
	return out
}
