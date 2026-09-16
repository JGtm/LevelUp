package killcollector

// map_identity.go — LA CARTE D'UN MATCH VIENT DE SON NOM, ET IL N'Y A QU'UN SITE POUR LE DIRE
// (lot 1.9.4, D13 : la donnée décide, une signature ne décide jamais quand le nom existe).
//
// # LE DÉFAUT QUE CE FICHIER FERME
//
// Le collecteur avait DEUX règles pour la même question. La passe des POSITIONS lisait le nom de
// carte du match dans la base (`port.ReplayMapNameRepo`) et cherchait l'entrée du catalogue par
// ce nom. La passe des TOUCHES, elle, appelait `grammar.DetectFilmMapEntry(dir, catalogue, "")` :
// elle DEVINAIT la carte en lisant le découpage d'i0 du film et en retenant l'entrée du catalogue
// dont les largeurs d'axe coïncidaient — le paramètre `mapNameOverride` existait et lui était
// passé VIDE, alors que le même collecteur tenait le nom.
//
// # POURQUOI LA SIGNATURE A ÉTÉ SUPPRIMÉE ET NON RÉTROGRADÉE EN REPLI
//
// Le brief du lot prévoyait d'en faire un repli nommé (D14) pour le cas où le nom manque. LA
// MESURE L'INTERDIT, et elle est collée au §5 du plan :
//
//	les largeurs d'axe valent `W = min(26, ceilLog2(ceil(60*etendue)))` par axe — une grandeur
//	  TRÈS grossière. Sur les 79 cartes du catalogue, 68 tombent dans 5 classes d'équivalence,
//	  dont une de 59 cartes (`15/15/17`). La signature n'identifie une carte que pour 11 cartes
//	  sur 79 ; partout ailleurs elle rend plusieurs candidats et refuse (`len(hits) != 1`) ;
//	sur les 17 films mesurés (14 témoins du corpus gate + les 8 builds) : 2 accords, 13
//	  ambiguïtés, et 2 DÉSACCORDS — les deux films Live Fire, où la signature rend EXACTEMENT UN
//	  candidat, `aquarius`, qui n'est pas la carte jouée. Les distances y auraient été calculées
//	  dans l'AABB d'une autre carte, sans un mot.
//
// Un repli qui se déclenche à tort corrompt un fait que la lecture aurait donné juste (D14 d) :
// une mécanique mesurée juste 2 fois sur 17 n'est pas un filet, c'est une source de faux. La
// carte se lit au nom, ou elle ne se lit pas — et l'absence est une ERREUR TYPÉE COMPTÉE (D-4
// d'ADR 0034), jamais un « au plus proche ».

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/games/halo_infinite/film/grammar"
)

// ErrSansNomDeCarte : ce match n'a AUCUNE identité de carte exploitable — la base ne le connaît
// pas, ou sa ligne de registre ne porte aucun nom.
//
// ELLE EST DISTINCTE DE [grammar.ErrUnknownMapBounds], et la distinction est le point : « je ne
// sais pas quelle carte » et « je sais quelle carte, elle n'est pas au catalogue » appellent deux
// gestes différents — instruire le registre du match d'un côté, étendre le catalogue de bornes de
// l'autre. Les confondre ferait chercher la correction du mauvais côté de la frontière.
var ErrSansNomDeCarte = errors.New("killcollector: aucun nom de carte pour ce match")

// resolveMapBounds : le nom de carte du match (base), puis ses bornes de déquantification
// (catalogue). Les deux échecs restent distincts pour l'appelant qui veut les compter séparément
// ([nomsDeCarteDuMatch], [entreeDeCatalogueParNom]) ; ce raccourci sert la passe des positions,
// dont les deux causes mènent au même compteur et au même abandon.
//
// LE COLLECTEUR DOIT AVOIR REÇU `WithPositionCapture` : `mapNames` et `mapBounds` sont vérifiés
// par les appelants (positions et touches), chacun avec son propre compteur de câblage.
func (c *KillSourceCollector) resolveMapBounds(ctx context.Context, matchID string) (grammar.MapQuantEntry, error) {
	noms, err := c.nomsDeCarteDuMatch(ctx, matchID)
	if err != nil {
		return grammar.MapQuantEntry{}, err
	}
	return c.entreeDeCatalogueParNom(noms)
}

// nomsDeCarteDuMatch rend les identités de carte candidates du match, telles que la BASE les
// nomme, vidées de leurs entrées vides. [ErrSansNomDeCarte] quand il n'en reste aucune.
func (c *KillSourceCollector) nomsDeCarteDuMatch(ctx context.Context, matchID string) ([]string, error) {
	keys, err := c.mapNames.MapKeysForMatch(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("identite de carte: %w", err)
	}
	noms := make([]string, 0, len(keys.Names))
	for _, name := range keys.Names {
		if name != "" {
			noms = append(noms, name)
		}
	}
	if len(noms) == 0 {
		return nil, fmt.Errorf("identite de carte: %w", ErrSansNomDeCarte)
	}
	return noms, nil
}

// entreeDeCatalogueParNom rend l'entrée de la PREMIÈRE identité candidate qui résout au
// catalogue. [grammar.ErrUnknownMapBounds] quand aucune ne résout : le nom est connu, la carte
// n'est pas au catalogue (D-4 d'ADR 0034 — elle se COMPTE, elle ne se devine pas).
//
// `repli_carte_premier_nom_resolu` (registre des replis) : les candidats sont essayés DANS
// L'ORDRE et le premier qui résout gagne, sans arbitrage. Ce n'est pas un ordre arbitraire —
// `MapKeysForMatch` les rend « du plus fiable au moins fiable » et documente pourquoi il y en a
// plusieurs (nom d'asset canonique contre libellé brut, l'un ou l'autre pouvant manquer).
func (c *KillSourceCollector) entreeDeCatalogueParNom(noms []string) (grammar.MapQuantEntry, error) {
	for _, name := range noms {
		if entry, err := c.mapBounds.Lookup(name); err == nil {
			return entry, nil
		}
	}
	return grammar.MapQuantEntry{}, fmt.Errorf("%w (candidats: %v)", grammar.ErrUnknownMapBounds, noms)
}
