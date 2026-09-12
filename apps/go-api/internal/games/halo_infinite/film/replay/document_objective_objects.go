package replay

// document_objective_objects.go — LES OBJETS D'OBJECTIF LIBRES : où se trouve le crâne quand
// PERSONNE ne le porte.
//
// # LE CANAL EST UNE PRÉSENCE, PAS UNE DÉDUCTION
//
// Un objet du monde réplique sa position tant qu'il est libre, et CESSE de la répliquer dès
// qu'un joueur le porte (principe établi et mesuré, `flag_objects.go` § « Le principe »). Ce
// calque publie donc EXACTEMENT ce que le film dit : les positions que l'objet a réellement
// émises. Rien n'y est interpolé, rien n'y est supposé — et surtout, rien n'y dit qui le porte
// pendant les trous.
//
// # POURQUOI LE PORTAGE N'EST PAS ICI, ET NE LE SERA PAS SANS UNE NOUVELLE MESURE
//
// La phase D4 a mesuré le portage du crâne et l'a REFUSÉ par son propre protocole : 40,6 à
// 66,7 % des trous ont un porteur unique contre un seuil de 90 %, et un intervalle témoin placé
// HORS trou rend le même signal dans 66,7 et 71,4 % des cas. L'oracle du score personnel ne
// discrimine pas le portage en Oddball. Ce calque publie donc l'objet LIBRE et se tait sur le
// reste : un trou dans la suite des vies est un portage, mais par QUI reste inconnu.
//
// # POURQUOI LE DRAPEAU N'Y EST PAS, ALORS QUE LE CANAL EST LE MÊME
//
// Ce n'est pas un report : c'est un REFUS MESURÉ, et il est antérieur. Le contrôle 3 du lot du
// drapeau exigeait que >= 90 % des vies libres naissent à moins de 1,5 m d'un `flag_spawn` ou
// du porteur qui vient de finir ; la mesure sur les trois films CTF rend 149/197 = 75,6 %, avec
// un témoin qui tient largement (12,8 %). La piste discrimine, mais un quart des vies reste
// inexpliqué. Publier les vies libres du drapeau demanderait de lever CE négatif-là, pas
// d'étendre ce fichier.
//
// LA FORME EST DÉLIBÉRÉMENT GÉNÉRIQUE (`family`) pour que le drapeau puisse la rejoindre le jour
// où son contrôle passera : aucune clé ne bougera alors, seul le contenu.

// ObjectiveObjectLife est UNE VIE LIBRE d'un objet d'objectif : l'objet apparaît, réplique sa
// position, puis se tait — parce qu'on l'a ramassé, ou parce qu'il s'est immobilisé.
//
// UNE VIE, PAS UN OBJET. Le crâne d'un match a autant de vies libres qu'il a été lâché puis
// repris (16 à 47 par film sur le corpus Oddball mesuré). Les regrouper par objet obligerait le
// client à retrouver lui-même les trous ; les publier vie par vie les lui donne.
type ObjectiveObjectLife struct {
	// Family est la nature de l'objet, telle que le manifeste du titre la déclare : `ball`
	// aujourd'hui, `flag` le jour où son contrôle passera. JAMAIS déduite du libellé.
	Family string `json:"family"`
	// En / Fr sont le nom de l'objet dans les deux langues du produit. Ils voyagent avec la vie
	// plutôt que dans une table à part : le client dessine ce que le document dit, et un libellé
	// rangé ailleurs se désynchronise (même règle que les armes et les grenades).
	En string `json:"en"`
	Fr string `json:"fr"`
	// T0 / T1 bornent la vie en frames (même axe que Point.T). T1 est INCLUS.
	//
	// T0 == T1 EST UNE VIE RÉELLE, PAS UN DÉFAUT : l'objet est né immobile — à son socle, le
	// plus souvent — et n'a jamais bougé. Il a donc UN point, et sa position est connue.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Pts est la trajectoire répliquée de la vie, triée, le premier point étant celui de la
	// CRÉATION. Au moins un point : une vie sans position ne serait pas publiable.
	Pts []ObjectiveObjectPoint `json:"pts"`
}

// ObjectiveObjectPoint est une position datée d'un objet d'objectif libre. Même axe et mêmes
// unités que `Point` — le client les dessine avec la même transformation, sans conversion.
type ObjectiveObjectPoint struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

// ObjectiveObjectsCoverage porte les dénominateurs du calque. Sans eux, « 16 vies » se lirait
// comme une exhaustivité, et un film sans aucune vie publiée serait indistinguable d'un film
// qu'on n'a pas su lire.
//
// ELLE EST PUBLIÉE MÊME QUAND AUCUNE VIE NE L'EST, pour la raison qui vaut partout ailleurs dans
// ce document : trois silences différents (pas de balayage, aucun objet déclaré au manifeste,
// objet déclaré mais absent du film) doivent rester distincts.
type ObjectiveObjectsCoverage struct {
	// Scanned dit que la chaîne des socles a bien balayé l'archétype `ti=42`. Faux : tout le
	// reste vaut zéro parce qu'on n'a rien lu, pas parce qu'il n'y avait rien.
	Scanned bool `json:"scanned"`
	// Declared est le nombre d'identifiants que le manifeste du titre déclare comme objets
	// d'objectif PUBLIABLES. Zéro = le titre n'en déclare aucun.
	Declared int `json:"declared"`
	// Lives / Points : vies publiées et points qu'elles portent.
	Lives  int `json:"lives"`
	Points int `json:"points"`
	// Motionless compte les vies réduites à UN point (nées immobiles). Publié parce que c'est le
	// cas qui ressemble le plus à un défaut sans en être un.
	Motionless int `json:"motionless"`
	// OutOfAxis compte les vies écartées parce que leur création tombe hors de l'axe de frames
	// publié. Une vie hors axe n'est pas dessinable ; la taire sans la compter ferait passer un
	// décalage d'horloge pour une absence d'objet.
	OutOfAxis int `json:"outOfAxis"`
}
