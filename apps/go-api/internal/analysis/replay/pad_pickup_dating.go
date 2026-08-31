package replay

// pad_pickup_dating.go — DATER LES OCCUPATIONS DE SOCLE AVEC L'ÉVÉNEMENT NATIF.
//
// LE PROBLÈME QU'IL RÈGLE. `padPickups` publiait « ce socle s'est vidé quelque part entre
// `tLow` et `tHigh` », un intervalle de vingt secondes, et `xuid` valait `null` PARTOUT. Ce
// n'était pas un oubli : le contrat de `PadPickup.XUID` porte la mesure qui l'a refusé
// (88,1 % en suivant le slot de vie, 79,7 % en suivant le joueur, contre >= 90 % exigé) et il
// nomme ce qui le lèverait — « un oracle plus RAPPROCHÉ que 20 s ».
//
// L'ÉVÉNEMENT NATIF EST CET ORACLE, et il ne fait aucune inférence : il est daté à la
// milliseconde et il porte son ramasseur (`512 + référence` = le slot, exact sur 32/32 paires
// de vérité terrain, deux films). Quand un ramassage natif de la MÊME FAMILLE tombe dans la
// fenêtre d'une occupation, on publie l'instant exact et le joueur.
//
// CE QU'ON NE FAIT PAS, ET C'EST LA RÈGLE : ON N'EFFACE RIEN. L'intervalle `[tLow, tHigh]`
// reste publié dans tous les cas — daté ou non. Une occupation que l'événement natif ne
// couvre pas garde exactement ce qu'elle avait avant ce lot ; le canal AJOUTE, il ne remplace
// pas. Le rappel du canal natif est une borne inférieure (il ne voit que les événements en
// tête de liste) : substituer serait échanger une donnée sûre contre une donnée partielle.
//
// AMBIGUÏTÉ : si PLUSIEURS ramassages natifs de la même famille tombent dans la même fenêtre,
// on ne date PAS. Deux joueurs ont pu prendre la même arme ailleurs sur la carte pendant ces
// vingt secondes, et rien dans l'événement ne dit de quel socle il vient (l'instance de
// l'objet n'est pas dans l'événement — hypothèse mesurée et réfutée). Choisir au hasard
// nommerait un ramasseur faux ; on s'abstient et on le compte.
//
// ## LES DEUX CÔTÉS N'ÉCRIVENT PAS LA FAMILLE PAREIL, ET LA JOINTURE DOIT LE SAVOIR
//
// Revue adversariale du 2026-08-31 : la première version de ce fichier comparait `Pickup.W`
// (`fmt.Sprintf("%08x", …)` — huit hexa MINUSCULES, sans préfixe) à `WeaponPad.Weapon`
// (`formatWeaponFamily` — `"0x"` + huit hexa MAJUSCULES). Les deux espaces ne coïncidaient
// JAMAIS : `hits` était toujours vide, aucun `padPickups[].t` n'était jamais écrit, aucun
// `xuid` posé — et `coverage.padDating` publiait `{dated: 0, uncovered: N}` qui SE LISAIT
// COMME UNE MESURE alors que c'était un défaut de format. La cuisson pilote n'a pas pu le
// révéler : son film ne porte aucun socle.
//
// LA NORMALISATION SE FAIT ICI, AU POINT DE JOINTURE, ET NULLE PART AILLEURS. Les formes
// publiées ne bougent PAS : `weaponChanges[].w` s'écrit ainsi depuis le schéma 25 et des
// clients peuvent déjà le lire ; `weaponPads[].weapon` depuis bien plus tôt. Changer l'une des
// deux pour faire plaisir à une jointure interne casserait un contrat public pour un confort
// privé.
//
// LES SOCLES DE POWER-UP NE SONT PAS JOIGNABLES, et ce n'est pas un échec de couverture : leur
// identité (`gwPadWeaponID` -> `Appar.Family`) est un NOM CANONIQUE, pas un identifiant de
// famille. Aucun ramassage natif ne peut donc s'y apparier, jamais. Ils sont comptés à part
// (`PowerupOccupations`) au lieu d'être noyés dans `Uncovered`, qui laisserait croire que le canal
// natif a cherché et n'a pas trouvé.

// PadDatingStats dit ce que la datation a pu faire, et ce qu'elle n'a pas pu.
type PadDatingStats struct {
	// Occupations est le nombre d'occupations achevées examinées.
	Occupations int `json:"occupations"`
	// Dated est le nombre d'occupations dont l'instant exact a été publié.
	Dated int `json:"dated"`
	// Named est le nombre dont le RAMASSEUR a pu être nommé (sous-ensemble de Dated : un
	// ramassage natif peut être daté sans que le pont slot -> joueur nomme sa vie).
	Named int `json:"named"`
	// Ambiguous compte les fenêtres où PLUSIEURS ramassages natifs de la même famille
	// tombaient : on s'abstient plutôt que de nommer un ramasseur au hasard.
	Ambiguous int `json:"ambiguous"`
	// Uncovered compte les fenêtres qu'aucun ramassage natif ne couvre — elles gardent leur
	// intervalle, intact.
	Uncovered int `json:"uncovered"`
	// PowerupOccupations compte les occupations de socle de POWER-UP, structurellement non
	// joignables : l'identité d'un tel socle est un NOM CANONIQUE, pas une famille d'arme.
	// Elles sont sorties d'`Uncovered` À DESSEIN — les y laisser ferait lire « le canal natif
	// a cherché et n'a pas trouvé » là où il n'y avait rien à chercher.
	//
	// LE NOM N'EST PAS `powerupPads`, ET C'EST DÉLIBÉRÉ (correctif de revue, ronde 2) :
	// `coverage.groundWeapons.powerupPads` existe déjà et compte des SOCLES PUBLIÉS. Deux clés
	// homonymes à dénominateurs différents dans le même document se liraient l'une pour
	// l'autre. Ici on compte des OCCUPATIONS écartées de la jointure — le nom le dit.
	PowerupOccupations int `json:"powerupOccupations"`
}

// padFamilyKey rend la clé de comparaison d'une famille d'arme, quelle que soit la convention
// d'écriture de la source, et dit si la valeur EST une famille.
//
// Les deux écritures rencontrées : `fmt.Sprintf("%08x", fam)` (canaux `pickups` et
// `weaponChanges`) et `formatWeaponFamily(fam)` = `"0x" + huit majuscules` (`loadouts`,
// `weaponPads`). Le second retour est FAUX pour tout ce qui n'est pas huit chiffres
// hexadécimaux — c'est ainsi qu'un socle de power-up (nom canonique) se distingue d'un socle
// d'arme, sans avoir à connaître la liste des noms.
func padFamilyKey(s string) (string, bool) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if len(s) != 8 {
		return "", false
	}
	buf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			buf[i] = c
		case c >= 'A' && c <= 'F':
			buf[i] = c + ('a' - 'A')
		default:
			return "", false
		}
	}
	return string(buf), true
}

// datePadPickups pose l'instant exact et le ramasseur sur les occupations que l'événement
// natif couvre. Modifie `picks` en place et rend les compteurs.
//
// `pads` sert à retrouver la FAMILLE d'arme du socle : `PadPickup.Pad` est un index dans
// `pads`, et c'est la famille qui apparie une occupation à un ramassage natif.
func datePadPickups(pads []WeaponPad, picks []PadPickup, pickups []Pickup) PadDatingStats {
	st := PadDatingStats{Occupations: len(picks)}
	if len(picks) == 0 {
		return st
	}
	// PAS DE RETOUR ANTICIPÉ QUAND LE CANAL NATIF EST VIDE, et c'est un correctif de revue
	// (ronde 2) : la première version versait alors TOUTES les occupations dans `Uncovered`,
	// power-ups compris — c'est-à-dire exactement la lecture mensongère (« le canal a cherché
	// et n'a pas trouvé ») que ce fichier prétend éliminer. La boucle ci-dessous classe
	// TOUJOURS, y compris avec zéro ramassage : un socle de power-up reste hors jointure, un
	// socle d'arme reste non couvert.
	//
	// Index par famille NORMALISÉE : une occupation ne s'apparie qu'à un ramassage de LA MÊME
	// arme, et les deux côtés ne l'écrivent pas pareil (cf. l'en-tête).
	byFamily := map[string][]Pickup{}
	for _, p := range pickups {
		if p.Kind != PickupWeapon {
			continue // un socle d'arme ne rend pas de l'équipement
		}
		key, ok := padFamilyKey(p.W)
		if !ok {
			continue
		}
		byFamily[key] = append(byFamily[key], p)
	}
	for i := range picks {
		k := &picks[i]
		if k.Pad < 0 || k.Pad >= len(pads) {
			st.Uncovered++
			continue
		}
		key, ok := padFamilyKey(pads[k.Pad].Weapon)
		if !ok {
			// Socle de POWER-UP (nom canonique) : rien à chercher, et on ne fait pas passer
			// ça pour une recherche infructueuse.
			st.PowerupOccupations++
			continue
		}
		var hits []Pickup
		for _, p := range byFamily[key] {
			if p.T >= k.TLow && p.T <= k.THigh {
				hits = append(hits, p)
			}
		}
		switch len(hits) {
		case 0:
			st.Uncovered++
		case 1:
			t := hits[0].T
			k.T = &t
			st.Dated++
			if hits[0].XUID != "" {
				x := hits[0].XUID
				k.XUID = &x
				st.Named++
			}
		default:
			// Plusieurs candidats : on s'abstient. Voir l'en-tête de ce fichier.
			st.Ambiguous++
		}
	}
	return st
}
