package domain

import "sort"

// GroupesDeCartes regroupe les identifiants de carte (map_id) qui désignent un même terrain :
// une carte de base, ses variantes déclarées (map_variants.toml : « - Ranked », « Heavies ») et
// les autres identifiants portant le nom de la base.
//
// C'est la table que toute statistique PAR CARTE interroge pour faire UNE ligne d'un groupe :
// `Groupe(id)` donne la clé de regroupement, `Membres(id)` la liste des identifiants à lire pour
// couvrir le groupe. Un identifiant qui n'appartient à aucun groupe est son propre groupe, de sorte
// que l'appelant n'a jamais à traiter l'absence de variante comme un cas à part.
//
// La valeur zéro est utilisable : aucun groupe, chaque identifiant reste seul.
type GroupesDeCartes struct {
	baseDe   map[string]string            // identifiant membre -> identifiant de base (base comprise)
	membres  map[string][]string          // identifiant de base -> membres triés (base comprise)
	libelles map[string]map[string]string // identifiant de base -> locale -> libellé de la base
}

// NewGroupesDeCartes construit la table depuis `baseParMembre` (identifiant membre -> identifiant
// de base). Une base peut figurer ou non comme membre d'elle-même. Une entrée qui ne regroupe rien
// (un identifiant seul dans son groupe) est ignorée : seuls les groupes d'au moins deux
// identifiants existent. Une base qui est elle-même rattachée à une autre est remontée jusqu'à la
// racine ; sur un cycle, la racine est le plus petit identifiant du cycle (résultat déterministe).
func NewGroupesDeCartes(baseParMembre map[string]string) GroupesDeCartes {
	membres := make(map[string]map[string]struct{})
	for id, base := range baseParMembre {
		if id == "" || base == "" {
			continue
		}
		racine := racineDeCarte(id, baseParMembre)
		if membres[racine] == nil {
			membres[racine] = map[string]struct{}{racine: {}}
		}
		membres[racine][id] = struct{}{}
	}
	g := GroupesDeCartes{}
	for racine, ensemble := range membres {
		if len(ensemble) < 2 {
			continue
		}
		if g.baseDe == nil {
			g.baseDe = make(map[string]string)
			g.membres = make(map[string][]string)
		}
		liste := make([]string, 0, len(ensemble))
		for id := range ensemble {
			liste = append(liste, id)
			g.baseDe[id] = racine
		}
		sort.Strings(liste)
		g.membres[racine] = liste
	}
	return g
}

// racineDeCarte remonte de `id` jusqu'à l'identifiant sans rattachement ; sur un cycle, rend le plus
// petit identifiant rencontré.
func racineDeCarte(id string, baseParMembre map[string]string) string {
	vus := map[string]struct{}{}
	courant := id
	for {
		vus[courant] = struct{}{}
		suivant, ok := baseParMembre[courant]
		if !ok || suivant == "" || suivant == courant {
			return courant
		}
		if _, deja := vus[suivant]; deja {
			plus := suivant
			for v := range vus {
				if v < plus {
					plus = v
				}
			}
			return plus
		}
		courant = suivant
	}
}

// Groupe rend l'identifiant de base du groupe de `id`, ou `id` lui-même quand il n'appartient à
// aucun groupe.
func (g GroupesDeCartes) Groupe(id string) string {
	if base, ok := g.baseDe[id]; ok {
		return base
	}
	return id
}

// Regroupe dit si `id` appartient à un groupe d'au moins deux identifiants.
func (g GroupesDeCartes) Regroupe(id string) bool {
	_, ok := g.baseDe[id]
	return ok
}

// Membres rend, triés, tous les identifiants du groupe de `id` (la base et ses variantes), ou `[id]`
// quand il n'appartient à aucun groupe. La liste est une copie : l'appelant peut la modifier.
func (g GroupesDeCartes) Membres(id string) []string {
	base, ok := g.baseDe[id]
	if !ok {
		return []string{id}
	}
	return append([]string(nil), g.membres[base]...)
}

// Vide dit si aucun groupe n'existe.
func (g GroupesDeCartes) Vide() bool { return len(g.baseDe) == 0 }

// LocaleLibelleParDefaut est la locale de repli de Libelle : le nom EN de la carte de base.
const LocaleLibelleParDefaut = "en"

// AvecLibelles rend une copie de la table qui porte, par identifiant de base, le libellé de la carte
// de base par locale (`libelles[base][locale]`). Les entrées dont la base n'est pas celle d'un groupe
// sont ignorées. La table d'origine n'est pas modifiée.
func (g GroupesDeCartes) AvecLibelles(libelles map[string]map[string]string) GroupesDeCartes {
	out := GroupesDeCartes{baseDe: g.baseDe, membres: g.membres}
	for base, parLocale := range libelles {
		if _, ok := g.membres[base]; !ok || len(parLocale) == 0 {
			continue
		}
		if out.libelles == nil {
			out.libelles = make(map[string]map[string]string)
		}
		out.libelles[base] = parLocale
	}
	return out
}

// Libelle rend le libellé de la carte de BASE du groupe de `id`, dans la `locale` demandée, à défaut
// en LocaleLibelleParDefaut. `ok` est faux quand `id` n'appartient à aucun groupe ou que la base n'a
// aucun libellé : l'appelant garde alors le libellé propre de la carte.
//
// C'est ce libellé, et jamais celui d'une variante, qu'une ligne regroupée affiche : aucune mention de
// la variante (décision produit du 2026-10-26).
func (g GroupesDeCartes) Libelle(id, locale string) (string, bool) {
	base, ok := g.baseDe[id]
	if !ok {
		return "", false
	}
	parLocale := g.libelles[base]
	if l := parLocale[locale]; l != "" {
		return l, true
	}
	if l := parLocale[LocaleLibelleParDefaut]; l != "" {
		return l, true
	}
	return "", false
}
