package games

// JournalDesMortsFiable dit si `match_kill_events` de ce titre nomme le tueur de chaque
// mort de facon exploitable LIGNE A LIGNE. C'est la porte du bloc de coordination (l'appui
// recu des pages Sessions et Series temporelles), dont le drapeau « match mesure » se lit
// dans ce journal.
//
// CE PREDICAT VIT ICI, ET PAS DANS UN SERVICE. Il est lu au cablage des deux pages
// (api/wire/registry_pages.go) et par le producteur du bloc (service/coordination_block.go) :
// deux copies donneraient deux verdicts differents au premier titre ajoute. Le predicat est
// PUR : une lecture de CapabilityMap, aucune I/O, aucune comparaison de slug.
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
// qu'une lecture ligne a ligne ne doit pas accepter. Infinite passe deja par
// `film.kill_source` ; l'exiger `supported` ici
// n'ote donc rien a personne, et protege le jour ou un titre ne declarerait QUE ce
// kill-feed la, en degrade. Halo 5 declare `supported` (mesure sur pieces, capabilities.toml
// du titre) et remplit `match_kill_events` par la reprise de `killer_victim_pairs`.
//
// UNE CAPABILITY ABSENTE N'EST PAS UN ZERO. Le titre qui echoue a cette porte recoit un
// bloc de coordination indisponible avec sa raison machine (`CoordinationUnsupported`) :
// publier un taux nul se lirait comme une contre-performance, quand la verite est « ce
// titre ne sait pas mesurer ca ».
func JournalDesMortsFiable(caps CapabilityMap) bool {
	return caps.Has(CapFilmKillSource) || caps[CapMatchKillfeedPerKill] == CapSupported
}
