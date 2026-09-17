package types

// grammar_abilities.go — LES TYPES DE CONTRAT DES CANAUX DE CAPACITE (couche `grammar`).
//
// Deplaces depuis `film/internal/grammar` au lot 2.6.2 (volet grammaire / rejeu) SANS
// REECRITURE : les commentaires sont ceux des declarations d origine, et un renvoi de godoc y
// designe encore un symbole de la couche qui produit. Les reecrire aurait fait passer un
// deplacement pur pour un changement.

// AbilityCharge est UNE lecture d'emplacement de charge ARMÉ, localisée dans le film.
type AbilityCharge struct {
	// Slot est l'identifiant bas du bipède porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Emplacement est l'index (0..2) du bit de masque R(3) qui a armé cette lecture.
	// C'est une donnée de DÉBOGAGE : la spécialisation mesurée par R11 (e0 propulseur,
	// e2 grappin) est une observation, jamais une identité — l'identité vient d'i48.
	Emplacement int
	// Charges est le quartet HAUT de la valeur 7 bits : le compte de charges ENTIÈRES
	// restantes (lecture discrète du consommateur de l'exe, validée R11 §2).
	Charges int
	// Low est le quartet bas : la recharge fractionnaire. Publié pour que la mesure reste
	// relisible (les témoins de R11 l'avaient à zéro sur toute la série validée).
	Low int
}

// AbilityChargeStats compte ce que la marche a rencontré. Sans ces dénominateurs, une
// liste de lectures ne se juge pas : « 12 lectures armées » ne dit rien sans « sur combien
// de records annonçant le composant ».
type AbilityChargeStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI56 : records dont le masque annonce le composant d'énergie.
	WithI56 int
	// Read / Unread : lectures i56 abouties, et records dont la marche n'a pas atteint la
	// cible (un composant intermédiaire non porté, ou un débordement du payload).
	Read, Unread int
	// Armed est le nombre d'emplacements ARMÉS publiés — la sortie. Une lecture aboutie au
	// masque 000 compte dans Read et pas ici : « le composant a parlé, aucun emplacement
	// n'est armé » est le zéro que R11 §4 mesure sur les films sans grappin ni propulseur.
	Armed int
	// Absent dit que le composant d'énergie n'est déclaré par AUCUNE des deux étiquettes
	// dans l'archétype biped du film. C'est une information, pas une erreur : le film ne
	// transmet alors pas ce canal, et une liste vide sans ce témoin serait indistinguable
	// d'un film où personne n'use ses charges.
	Absent bool
	// Scanned dit que LE BALAYAGE A TOURNÉ. Faux = il n'a jamais commencé (une des quatre
	// portes de résolution a refusé : aucun chunk, aucun slot biped aux images-clés,
	// découpage i0 indétectable, registre illisible) — l'appelant reçoit alors une erreur,
	// et tout ce qui suit dans cette structure est un zéro SANS SIGNIFICATION.
	//
	// POURQUOI UN TÉMOIN PLUTÔT QUE L'ERREUR SEULE : l'erreur meurt chez l'appelant
	// immédiat, et le zéro qu'il laisse derrière voyage jusqu'à l'artefact. Sans ce champ,
	// une couverture de zéros affirmerait « le balayage a tourné, personne n'a usé de
	// charge » sur un film où rien n'a jamais été lu — la faute exacte que la doctrine de
	// coverage.go interdit (leçon H1 de la seconde passe de revue P3, recopiée d'
	// AbilityImpulseStats.Scanned). Un balayage qui aboutit le pose, `Absent` compris.
	Scanned bool
}

// AbilityImpulse est UNE lecture d'impulsion de capacité, localisée dans le film.
type AbilityImpulse struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires,
	// donc UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Predicted dit que la lecture vient du composant PRÉDIT i57 plutôt que de son jumeau
	// non prédit i59. LES DEUX SONT CO-TRANSMIS : un même geste apparaît souvent dans les
	// deux, et c'est à l'assembleur de les replier en épisodes plutôt que de compter deux
	// fois. Le témoin est publié pour que la couverture puisse dire lequel a parlé.
	Predicted bool
}

// AbilityImpulseStats compte ce que la marche a rencontré. Sans ces dénominateurs, une
// liste d'impulsions ne se juge pas : « 60 lectures » ne dit rien sans « sur combien de
// records annonçant le composant ».
type AbilityImpulseStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI57 / WithI59 : records dont le masque annonce le composant prédit / non prédit.
	WithI57, WithI59 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint la cible
	// (un composant intermédiaire non porté, ou un débordement du payload).
	Read, Unread int
	// Tag1 est le nombre de lectures dont le tag externe vaut abilityImpulseTag — les
	// seules publiées.
	Tag1 int
	// Absent dit qu'AUCUN des deux composants n'est déclaré par l'archétype biped du film.
	// C'est une information, pas une erreur : le film ne transmet alors pas ce canal, et
	// une liste vide sans ce témoin serait indistinguable d'un film sans propulseur.
	Absent bool
	// Scanned dit que LE BALAYAGE A TOURNÉ. Faux = il n'a jamais commencé (une des quatre
	// portes de résolution a refusé : aucun chunk, aucun slot biped aux images-clés, découpage
	// i0 indétectable, registre illisible) — l'appelant reçoit alors une erreur, et tout ce qui
	// suit dans cette structure est un zéro SANS SIGNIFICATION.
	//
	// POURQUOI UN TÉMOIN PLUTÔT QUE L'ERREUR SEULE : l'erreur meurt chez l'appelant immédiat,
	// et le zéro qu'il laisse derrière voyage jusqu'à l'artefact. Sans ce champ, une couverture
	// de zéros affirmerait « le balayage a tourné, le composant est là, personne ne s'en est
	// servi » sur un film où rien n'a jamais été lu — la faute exacte que la doctrine de
	// coverage.go interdit (cf. `attachInventoryCoverage`). Un balayage qui aboutit le pose,
	// `Absent` compris : « aucun composant déclaré » EST un résultat de balayage.
	Scanned bool
}

// AbilityRank est UNE transmission d'identité de capacité, localisée dans le film.
type AbilityRank struct {
	// Slot est l'identifiant bas du biped porteur — le même que celui des trajectoires, donc
	// UNE VIE et non un joueur (le slot migre aux réapparitions).
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur — MÊME horloge que BipedPosition.
	TimestampUS uint64
	// Counter est le compteur de rotation R(3). Il n'identifie rien à lui seul ; il est
	// conservé parce qu'il BORNE l'interprétation (une valeur hors 0..7 dirait que la lecture
	// est mal placée).
	Counter uint32
	// Rank est le rang dans la palette du match. Jamais AbilitySetNoRank : les lectures dont
	// la porte est ouverte ne sont pas émises — une identité non transmise n'est pas une
	// identité nulle.
	Rank int
}

// AbilityRankStats compte ce que la marche a rencontré. Sans ces dénominateurs, un
// histogramme de rangs ne se juge pas.
type AbilityRankStats struct {
	// Records est le nombre de records delta biped reconnus.
	Records int
	// WithI48 est le nombre de ces records dont le masque annonce i48.
	WithI48 int
	// Read / Unread : lectures abouties, et records dont la marche n'a pas atteint i48 (un
	// composant intermédiaire non porté, ou un débordement du payload).
	Read, Unread int
	// Gated est le nombre de lectures abouties SANS identité (porte à 1).
	Gated int
}
