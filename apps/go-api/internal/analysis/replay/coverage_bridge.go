package replay

// coverage_bridge.go — LA SANTÉ DU PONT slot -> joueur, et le verdict qui en découle.
//
// Sorti de `coverage.go` le 2026-09-07 (constat C5 de la revue VIES-R1) : ce fichier-là
// franchissait les 500 lignes du dépôt en absorbant les cinq champs du nommage final. Le
// découpage suit la frontière naturelle du sujet — `coverage.go` porte ce que chaque CALQUE a
// rattaché, celui-ci ce que le PONT a su nommer. Déplacement PUR : aucune ligne de logique n'est
// modifiée, seul l'emplacement change.

import "log/slog"

// BridgeHealth résume la santé du pont slot -> joueur.
//
// POURQUOI LA PUBLIER PLUTÔT QUE LA GARDER POUR LE JOURNAL : la couverture d'un calque peut
// être excellente alors que le pont qui la porte est fragile. On publie donc, à côté du taux,
// de quoi juger la SOURCE : combien de vies le fil des morts a nommées, et combien de chunks
// ont confirmé la table d'index.
//
// UN CHAMP PAR LIGNE, ET C'EST DÉLIBÉRÉ : en Go, plusieurs champs déclarés sur une même
// ligne PARTAGENT leur tag, ce qui produirait ici plusieurs clés JSON identiques. Le
// compilateur ne dit rien ; seule la sortie sérialisée le révèle. Un test le verrouille
// (TestBridgeHealthJSONKeysAreDistinct).
type BridgeHealth struct {
	// Slots est le nombre d'entrées du pont.
	Slots int `json:"slots"`
	// FromReading : le nombre d'entrées issues de la LECTURE SEULE.
	//
	// CE N'EST PLUS LA SEULE SOURCE, et l'écrire ici l'aurait laissé croire : les FERMETURES
	// (closures.go, 2026-08-08) alimentent le pont par déduction. La règle que `verdictOfBridge`
	// applique réellement est `FromReading + ClosedByShot + ClosedByRespawn == Slots` — un écart
	// à cette somme, et non à `Slots` seul, signale une source non comptée.
	FromReading int `json:"fromReading"`
	// LivesNamed / LivesTotal : ce que le fil des morts a nommé.
	LivesNamed int `json:"livesNamed"`
	LivesTotal int `json:"livesTotal"`
	// IndexReadings : combien de chunks de réplication ont livré la MÊME table identité ->
	// index. Remplace la « marge » de l'ancienne résolution par choix.
	IndexReadings int `json:"indexReadings"`
	// IndexDisagreements : identités lues différemment d'un chunk à l'autre. Non nul, la
	// lecture est fausse.
	IndexDisagreements int `json:"indexDisagreements"`
	// SlotCollisions : slots dont les vies nommées désignent des joueurs différents. Non
	// nul, la table slot -> joueur n'est pas représentable et le verdict le dit.
	SlotCollisions int `json:"slotCollisions"`
	// NamedByPreviousLife / NamedByNextLife / NamedBySlotBridge : ce que le NOMMAGE FINAL a
	// réparé, par voie (cf. unnamed_lives.go). Depuis la décision produit du 2026-09-07 — « les
	// vies anonymes n'existent pas » — une piste publiée sans identité est un DÉFAUT du pont ;
	// ces trois compteurs disent par quelle preuve il a été comblé, et `UnnamedLives` ce qui a
	// résisté. Trois champs plutôt qu'un total : les voies n'ont pas la même force de preuve.
	NamedByPreviousLife int `json:"namedByPreviousLife"`
	NamedByNextLife     int `json:"namedByNextLife"`
	NamedBySlotBridge   int `json:"namedBySlotBridge"`
	// UnnamedLives : les pistes PUBLIÉES qu'aucune voie n'a su nommer. C'est le RÉSIDU, et il
	// doit tendre vers zéro — un artefact non nul porte un défaut de nommage à instruire, pas
	// une population « inconnue » à afficher. Doublé d'un `slog.Error` à la cuisson.
	UnnamedLives int `json:"unnamedLives"`
	// UnnamedLivesContested : la part d'`UnnamedLives` qui tombe sur une FRONTIÈRE entre deux
	// occupants nommés différents du même slot, que rien ne date. Comptée à part parce qu'elle
	// n'appelle pas le même chantier : l'identité n'y est pas absente, elle est INDÉCIDABLE —
	// nommer par « l'occupant précédent » serait un choix par l'ordre, celui-là même que
	// `SlotAmbiguous` existe pour signaler.
	UnnamedLivesContested int `json:"unnamedLivesContested"`
	// DeathOffsetMatched / DeathOffsetRunnerUp : la MARGE du calage du fil des morts — ce que
	// le calage retenu apparie, et ce que le meilleur des autres candidats aurait apparié.
	//
	// POURQUOI CETTE PAIRE EST PUBLIÉE. Depuis le 2026-09-07 le calage n'est plus cherché par un
	// balayage exhaustif de toute la plage mais par un vote qui localise quelques candidats,
	// puis par un affinage qui les mesure (cf. lives.go). Un vote peut se tromper de panier ; ce
	// qui l'empêche de le faire EN SILENCE — le défaut même que ce lot répare —, c'est de
	// publier le second compte à côté du premier. Un calage vrai écrase ses concurrents :
	// mesuré 71 contre 8 sur `51ebbc0f` et 157 contre 15 sur `d9781168`. Une marge sous
	// `deathOffsetMargeMin` est doublée d'un `slog.Warn` à la cuisson.
	DeathOffsetMatched  int `json:"deathOffsetMatched"`
	DeathOffsetRunnerUp int `json:"deathOffsetRunnerUp"`
	// DeathOffsetMs est LE CALAGE LUI-MÊME (lot M1b, 2026-09-08) : `horlogeFilm = horlogeMatch +
	// DeathOffsetMs` (cf. `OwnerReport.DeathOffsetMS`, `lives_export.go`). Publié pour que le
	// client puisse convertir un instant reçu sur l'horloge du MATCH (contrat
	// `TacticalContribution`, questions `morts`/`kills`/`gagne`/`isole`) en frame EXACTE du
	// rejeu, plutôt que l'approximation que `?frame=` servait jusqu'ici pour ces quatre
	// questions sur six (décalage mesuré de 3,6 à 50,8 s selon le match — cf.
	// `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md`).
	//
	// POINTEUR, PAS int64 : MÊME PIÈGE omitempty que `ReplayDocument.OriginMs`/`T0FilmMs`
	// (document.go). ZÉRO N'EST PAS UNE VALEUR PAR DÉFAUT ACCEPTABLE — un film dont le calage
	// mesuré vaut exactement 0 ms est un cas réel (horloges déjà alignées), et l'omettre le
	// ferait relire comme « calage inconnu ». ABSENT (nil) veut dire, et seulement : le pont
	// n'a pas été construit ou n'a apparié AUCUNE mort (`OwnerReport.DeathOffsetMatches == 0`,
	// cf. buildCoverage) — le calage n'existe alors pas, ce n'est pas une valeur.
	DeathOffsetMs *int64 `json:"deathOffsetMs,omitempty"`
	// ClosedByShot : entrées ajoutées par la fermeture A (le corps disponible).
	ClosedByShot int `json:"closedByShot"`
	// ClosedByRespawn : entrées ajoutées par la fermeture B (la réapparition).
	ClosedByRespawn int `json:"closedByRespawn"`
	// ClosedContested : déductions ABANDONNÉES faute d'unicité — deux corps possibles pour un
	// même tir, deux joueurs pour un même corps, deux corps pour un même joueur (un joueur n'a
	// qu'un DERNIER corps), ou deux corps pour une même mort.
	ClosedContested int `json:"closedContested"`
	// ClosedRefused : déductions REJETÉES alors qu'un seul candidat subsistait, pour l'une des
	// causes qui rendent l'attribution IMPOSSIBLE OU NON CORROBORÉE : le corps déduit ne
	// PROLONGE pas le tireur (fermeture A — aucun corps connu pour l'ancrer, ou un corps connu
	// qui ne s'achève pas avant lui ; le recouvrement en est le cas particulier), le
	// recouvrement (fermeture B — un joueur n'a qu'un corps), ou une identité que la table
	// d'index ne porte pas (on saurait quelle victime, pas quel index de film).
	//
	// LES DEUX COMPTEURS DE REFUS SONT PUBLIÉS AU MÊME TITRE QUE LES SUCCÈS, et ce n'est pas
	// de la décoration : un contrôle qui ne rejette jamais rien ne prouve rien. Mesuré sur
	// sept films, 33 attributions pour 17 refus.
	ClosedRefused int `json:"closedRefused"`
}

// VerdictNominal est le verdict d'un calque publiable sans réserve. Nommé plutôt que répété :
// c'est la valeur que les consommateurs comparent, et une faute de frappe dans l'une des six
// occurrences produirait un calque silencieusement « non nominal ».
const VerdictNominal = "nominal"

// Seuils de la porte de publication. Ils sont ÉCRITS ICI, pas dispersés dans les appelants,
// et chacun porte la mesure qui l'a fixé.
const (
	// publishMinRatio : en deçà, le calque publie moins des deux tiers de ce qui existe et
	// se lira comme exhaustif alors qu'il ne l'est pas. Le critère de succès du plan est à
	// 85 % ; 0,66 est le seuil du REFUS, pas celui de la satisfaction.
	publishMinRatio = 0.66
	// bridgeMinReadings : nombre minimal de chunks concordants pour tenir la table d'index
	// pour lue. Mesuré sur 000d5950 : 26 chunks de réplication, tous d'accord. En deçà de 2,
	// on n'a pas de confirmation du tout.
	bridgeMinReadings = 2
)

// verdictOf rend le verdict d'un calque : « nominal », « partiel » ou « non publiable ».
//
// LA PORTE REFUSE PLUTÔT QUE D'AVERTIR. Un calque « non publiable » doit être retiré de
// l'écran, pas affiché avec une note en bas de page : un tir posé sur le mauvais joueur est
// pire qu'un tir absent, et c'est la leçon que ce chantier a déjà payée.
func verdictOf(c LayerCoverage) string {
	switch {
	case c.Available == 0:
		return "aucune donnée"
	case !c.Balanced():
		return "non publiable : fuite dans le comptage"
	case float64(c.Attached)/float64(c.Available) < publishMinRatio:
		return "partiel : moins des deux tiers rattachés"
	default:
		return VerdictNominal
	}
}

// verdictOfBridge rend le verdict du pont lui-même.
//
// LA RÈGLE DE PROVENANCE A CHANGÉ LE 2026-08-08, ET LE COMMENTAIRE AVEC ELLE. Elle exigeait
// `FromReading == Slots` — « une source autre que la lecture a alimenté le pont ». Cette règle
// est née du retrait du repli VOTÉ (2026-07-28) et elle visait juste : rien ne devait entrer au
// pont par un choix. Elle deviendrait FAUSSE telle quelle depuis l'ajout des FERMETURES
// (closures.go), qui ne sont pas des lectures mais des déductions par élimination.
//
// Ce qui est conservé, et c'est l'esprit de la règle : **toute entrée du pont doit être
// justifiée par une source nommée**. La somme lecture + fermetures doit donc rendre EXACTEMENT
// le nombre d'entrées ; un écart signalerait qu'une troisième source, elle non comptée, s'est
// glissée dans le pont — précisément ce que la règle d'origine interdisait.
func verdictOfBridge(b BridgeHealth) string {
	switch {
	case b.Slots == 0:
		return "non publiable : aucun pont"
	case b.SlotCollisions > 0:
		return "non publiable : un slot change de porteur"
	case b.FromReading+b.ClosedByShot+b.ClosedByRespawn != b.Slots:
		return "non publiable : une source non comptée a alimenté le pont"
	case b.IndexDisagreements > 0:
		return "non publiable : une identité est lue de deux façons"
	case b.IndexReadings < bridgeMinReadings:
		return "partiel : la table d'index n'est confirmée par aucun second chunk"
	default:
		return VerdictNominal
	}
}

// warnIfCalageEtroit alarme quand le calage du fil des morts n'écrase pas franchement son
// meilleur concurrent.
//
// LE CALAGE EST DÉSORMAIS LOCALISÉ PAR UN VOTE (cf. lives.go), et une heuristique peut se
// tromper de panier. Le compte du candidat suivant est le seul témoin qui le dise : quand le
// calage retenu ne fait pas au moins `deathOffsetMargeMin` fois mieux, il n'est plus
// distinguable du bruit et TOUT ce qui en dépend est suspect — le nommage des vies, et
// l'origine du document dont il est le témoin (`resolveOriginMs`). Mesuré x8,9 et x10,5 sur
// les deux témoins du parc : ce seuil n'est pas atteint aujourd'hui, et c'est bien pourquoi
// l'atteindre doit se voir.
func (b BridgeHealth) warnIfCalageEtroit() {
	if b.DeathOffsetMatched == 0 ||
		b.DeathOffsetMatched >= deathOffsetMargeMin*b.DeathOffsetRunnerUp {
		return
	}
	slog.Warn("rejeu : calage du fil des morts trop peu distinct du bruit — nommage et origine suspects",
		"apparies", b.DeathOffsetMatched, "second_candidat", b.DeathOffsetRunnerUp,
		"marge_minimale", deathOffsetMargeMin, "vies", b.LivesTotal, "slots", b.Slots)
}
