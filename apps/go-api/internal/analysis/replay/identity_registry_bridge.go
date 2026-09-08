package replay

// identity_registry_bridge.go — LE PONT PAR MORTS, DEVENU UNE VERIFICATION (lot E2, 2026-09-08).
//
// # CE QU'IL FAISAIT, ET POURQUOI IL NE LE FAIT PLUS
//
// Il NOMMAIT : la mort qui termine une vie donnait a cette vie l'identite de sa victime. C'etait
// le seul lien disponible tant que rien dans le film ne rattachait un corps a un joueur. Le
// record de creation du bipede porte cette identite (`identity_registry_creation.go`) : le pont
// est passe de PRODUCTEUR a TEMOIN.
//
// # POURQUOI IL FALLAIT LE DECLASSER, ET PAS SEULEMENT LE DOUBLER
//
// Son appariement est glouton par ecart croissant, et il DEPARTAGE PAR L'ORDRE DES SLOTS quand
// deux fins de vie tombent au meme instant — fin de manche, fin de film, double kill. Le sondage
// E2 a mesure la consequence : sur `d9781168` et `64e8adfa`, **sept paires EXACTEMENT ECHANGEES**
// entre deux vies qui se terminent a la meme image. Ce n'est pas un desaccord a arbitrer, c'est
// le pont qui se trompe et la lecture qui tranche. Le garder en repli aurait laisse ces sept
// paires vivre partout ou la lecture directe n'aurait rien dit.
//
// # LE DIRECT L'EMPORTE, TOUJOURS, ET LE DESACCORD SE COMPTE
//
// Une discordance n'ecrase JAMAIS le nom direct : elle s'inscrit (`Discordant`) et elle alarme.
// Un compteur muet ferait disparaitre exactement ce que ce declassement corrige.
//
// # LA SEULE SITUATION OU IL NOMME ENCORE
//
// Quand le registre ne recoit AUCUN record de creation — le film n'a pas ete lu par ce canal, ou
// l'appelant est un producteur qui ne le porte pas encore. Sans lecture directe, toute vie serait
// `sans_record` et le rejeu perdrait d'un coup son pont, ses tirs et ses pistes nommees. C'est
// une DEGRADATION COMPLETE ET DECLAREE (`bridgeNamedLives` publie, non nul = le direct n'a pas
// tourne), pas un repli par vie : sur un film dont les creations sont lues, le pont ne nomme
// RIEN.
//
//	bascule du defaut : 2026-09-08 (lot E2)
//	retrait cible     : quand `testdata/inputs_000d5950.bin.gz` portera les creations — il est
//	                    fige AVANT ce canal et son film (`000d5950`) n'est plus au parc (466
//	                    films, celui-la absent), donc il ne peut pas etre regenere aujourd'hui.
//	critere mesurable : `bridgeNamedLives == 0` sur toute cuisson de film du parc.

import "log/slog"

// bridgeVerification est ce que la confrontation du pont au lien direct etablit.
type bridgeVerification struct {
	// Matched : les vies dont une mort du fil apparie la fin. C'est le DENOMINATEUR, et il ne
	// depend pas du nommage — il vaut ce que valait `DeathsNamed` avant le declassement.
	Matched int
	// Concordant / Discordant : parmi les vies appariees ET nommees par la lecture directe,
	// celles dont la victime EST le joueur lu, et celles dont elle ne l'est pas.
	Concordant, Discordant int
	// NamedByBridge : les vies que le pont a NOMMEES. Non nul = le registre n'a recu aucune
	// lecture directe (cf. l'en-tete de ce fichier).
	NamedByBridge int
}

// marquerCauseDeMort pose [CauseVieMort] sur les vies que le fil des morts apparie.
//
// LA CAUSE N'EST PAS LE NOM, et les confondre est ce qui a coute une lecture d'isolement entiere
// (P0 de la ronde 2, 2026-09-07). Le fil des morts reste la SEULE source qui dise « ce joueur est
// mort » : ce marquage-la ne se declasse pas, il n'a jamais nomme personne.
func marquerCauseDeMort(lives []lifeSpan, pairs []deathPair) {
	for _, p := range pairs {
		lives[p.li].cause = CauseVieMort
	}
}

// verifierParLesMorts confronte la victime de chaque mort appariee au joueur que la LECTURE
// DIRECTE a pose sur la meme vie. Elle ne modifie AUCUNE vie.
func verifierParLesMorts(lives []lifeSpan, deaths []Death, pairs []deathPair,
	matchID string) bridgeVerification {
	v := bridgeVerification{Matched: len(pairs)}
	for _, p := range pairs {
		l := lives[p.li]
		if l.xuid == 0 || !nomParLecture(l.nomPar) {
			continue
		}
		if l.xuid == deaths[p.di].XUID {
			v.Concordant++
			continue
		}
		v.Discordant++
		slog.Warn("rejeu : le pont par morts CONTREDIT le lien direct — le film fait foi",
			"match_id", matchID, "slot", l.slot, "de", l.from, "a", l.to,
			"xuidDirect", l.xuid, "xuidDuPont", deaths[p.di].XUID, "voie", l.nomPar)
	}
	return v
}

// nomParLecture dit si cette voie est une LECTURE du film — la seule qu'il y ait lieu de
// confronter au pont. Confronter le pont a une deduction ne mesurerait que la deduction.
func nomParLecture(nomPar string) bool {
	return nomPar == NomParCreation || nomPar == NomParCreationPropagee
}

// nommerParLesMorts pose l'identite de la victime sur la vie que sa mort termine. APPELEE DANS
// UN SEUL CAS — aucune lecture directe recue (cf. l'en-tete du fichier) — et l'appelant publie
// son compte pour que le cas ne passe pas inapercu.
func nommerParLesMorts(lives []lifeSpan, deaths []Death, pairs []deathPair) int {
	n := 0
	for _, p := range pairs {
		if lives[p.li].xuid != 0 {
			continue
		}
		lives[p.li].xuid = deaths[p.di].XUID
		lives[p.li].nomPar = NomParMort
		n++
	}
	return n
}
