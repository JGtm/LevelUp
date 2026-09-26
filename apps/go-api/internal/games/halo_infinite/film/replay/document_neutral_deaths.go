package replay

// document_neutral_deaths.go — LES MORTS QUE PERSONNE NE REVENDIQUE.
//
// (Extrait de document.go le 2026-08-18, revue R1 du lot A : le fichier depassait sa taille
// d'avant le lot. Aucune ligne de ce type n'a change — c'est un DEPLACEMENT, sur le modele de
// document_ground_weapons.go et document_score.go.)

// NeutralDeath est une mort que PERSONNE ne revendique — et de quoi le joueur est mort.
//
// # CE QU'ELLE EST, ET CE QU'ELLE N'EST PAS
//
// Le kill feed du match porte cette MORT sans aucun kill en face, et la source du dégât fatal
// désigne la victime elle-même : chute, sortie de zone, ou sa propre arme. Ce n'est donc pas un
// kill sans tueur, c'est une mort SANS TUEUR — la distinction n'est pas rhétorique, elle
// interdit d'inventer un responsable pour faire tenir la ligne dans le moule d'un kill.
//
// # POURQUOI L'ARTEFACT LA PORTE
//
// Le client déduit déjà ces morts de ses pistes (une fin de vie qu'aucun kill ne consomme) et
// leur donne une ligne grise. Ce qu'il ne peut PAS déduire, c'est DE QUOI le joueur est mort :
// cela se lit dans le composant dead-state du film, hors ligne, par le décodeur de source de
// dégât. Cette table est ce pont-là, et rien d'autre.
//
// # LA RÈGLE QUI GOUVERNE LE CHAMP `Kind`
//
// Une mort dont la nature n'est pas établie N'ENTRE PAS dans cette table : le fil garde son
// repère neutre. Jamais l'icône d'une autre mort — même faute que servir l'icône d'une autre
// arme, déjà refusée au chantier des icônes de kill feed.
type NeutralDeath struct {
	// XUID identifie le joueur mort. C'est la SEULE clé de jointure avec les pistes : un
	// pseudo change, un xuid non.
	XUID string `json:"xuid"`
	// FeedMs est l'instant de la mort SUR L'HORLOGE DU FIL, pas sur l'axe du rejeu — la même
	// horloge que les kills du feed une fois `t0_ms` ajouté, celle dont `OriginMs` donne le
	// décalage. Le client applique le MÊME recalage qu'à ses autres lignes de fil ; publier
	// ici un instant déjà recalé figerait dans l'artefact un décalage que le client sait
	// mesurer autrement quand l'origine n'est pas établie.
	FeedMs int `json:"feedMs"`
	// Kind est le TYPE de mort établi, en identifiant stable (jamais un libellé traduit — la
	// règle i18n du dépôt : les libellés vivent côté affichage) :
	//
	//	environment  chute, hors-limites, dégât de monde — la nature `DEGAT_GLOBAL` du décodeur
	//	suicide      le joueur s'est tué avec sa propre source de dégât (sa roquette, sa grenade)
	//
	// Un type inconnu ne s'écrit pas : l'entrée est omise.
	Kind string `json:"kind"`
	// Img est l'URL du pictogramme du jeu qui représente ce type de mort. Vide = le titre n'en
	// sert pas : le client garde son repère neutre. Tinted dit si le visuel est un masque à
	// teindre (même contrat que les icônes d'arme du fil).
	Img    string `json:"img,omitempty"`
	Tinted bool   `json:"tinted,omitempty"`
}
