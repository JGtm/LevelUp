package games

// JournalDesMortsFiable dit si `match_kill_events` de ce titre nomme le tueur de chaque
// mort de facon exploitable LIGNE A LIGNE — ce qu'exige la mesure de l'echange (« qui a
// venge qui »), et rien de moins.
//
// CE PREDICAT VIT ICI, ET PAS DANS UN SERVICE. Il a DEUX lecteurs depuis le 2026-09-06 :
// l'onglet Tactique (KPI d'echange par carte) et la page Escouade (matrice, delais, KPI).
// Deux copies auraient donne deux verdicts differents au premier titre ajoute, et donc deux
// taux d'echange sous le meme nom sur deux pages voisines. Le predicat est PUR : une lecture
// de CapabilityMap, aucune I/O, aucune comparaison de slug.
//
// LES DEUX PROVENANCES, et elles ne se lisent PAS de la meme facon :
//
//	film.kill_source              la source du degat fatal, decodee du film
//	                              (Halo Infinite : supported). `Has` suffit.
//	match.killfeed.per_kill       le kill-feed natif de l'API du titre. Exige ici
//	                              `supported` STRICTEMENT, pas `Has`.
//
// POURQUOI `supported` STRICTEMENT SUR LA SECONDE. `CapabilityMap.Has` accepte aussi
// `degraded`, et Halo Infinite declare justement `match.killfeed.per_kill = degraded`
// (kills simultanes possiblement omis, cf. capabilities.toml) — soit exactement le defaut
// qui fabriquerait de faux echanges : une mort omise dans la fenetre de 5 s se lit comme
// « non vengee ». Infinite passe deja par `film.kill_source` ; l'exiger `supported` ici
// n'ote donc rien a personne, et protege le jour ou un titre ne declarerait QUE ce
// kill-feed la, en degrade. Halo 5 declare `supported` (mesure sur pieces, capabilities.toml
// du titre) et remplit `match_kill_events` par la reprise de `killer_victim_pairs`.
//
// UNE CAPABILITY ABSENTE N'EST PAS UN ZERO. Le titre qui echoue a cette porte ne recoit
// AUCUNE section d'echange (ni KPI, ni matrice, ni distribution) : publier un taux nul se
// lirait comme une contre-performance, quand la verite est « ce titre ne sait pas mesurer
// ca ».
func JournalDesMortsFiable(caps CapabilityMap) bool {
	return caps.Has(CapFilmKillSource) || caps[CapMatchKillfeedPerKill] == CapSupported
}
