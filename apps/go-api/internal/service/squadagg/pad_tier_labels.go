package squadagg

// pad_tier_labels.go — LE NOM DES ARMES DU DETAIL PAR NIVEAU, en un seul exemplaire.
//
// # POURQUOI CE FICHIER EXISTE
//
// Trois pages publient la ventilation par niveau d'arme — Sessions, Escouade, Timeseries — et
// chacune charge le catalogue du titre à son propre endroit (c'est la seule chose qu'elles ne
// peuvent pas partager : le `repoRoot` n'arrive pas par le même chemin). Le NOMMAGE, lui, doit
// rester unique : une seconde table de correspondance dériverait au premier ajout du manifeste
// du titre, et deux pages afficheraient deux noms pour la même arme.
//
// # LA REGLE, LA MEME QUE PARTOUT AILLEURS DANS LE DEPOT
//
// Une famille absente du catalogue garde SA CLE à l'écran, jamais un nom approchant — « un nom
// approchant se lit comme une certitude ». Le catalogue vide ne nomme rien et ne casse rien.

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// NommerArmesDesNiveaux pose `FamilyLabel` sur chaque arme du détail par niveau.
//
// `frPreferred` : rendre le libellé FR quand il existe (défaut du produit) ; l'anglais sinon.
// `parseKey` traduit la clé du contrat en identifiant de famille — injectée pour que ce paquet
// n'ait pas sa propre lecture d'une clé dont la forme appartient à `replay`.
//
// Silencieuse par construction : bloc nil, catalogue vide, clé illisible ou famille inconnue
// laissent la ligne telle quelle.
func NommerArmesDesNiveaux(
	block *domain.SessionUsagePadTiersBlock, weapons map[uint32]replay.WeaponLabel,
	frPreferred bool, parseKey func(string) (uint32, bool),
) {
	if block == nil || len(weapons) == 0 || parseKey == nil {
		return
	}
	for i := range block.Tiers {
		armes := block.Tiers[i].Weapons
		for j := range armes {
			famille, ok := parseKey(armes[j].FamilyKey)
			if !ok {
				continue
			}
			label, ok := weapons[famille]
			if !ok {
				continue
			}
			name := label.En
			if frPreferred && label.Fr != "" {
				name = label.Fr
			}
			if name != "" {
				armes[j].FamilyLabel = name
			}
		}
	}
}
