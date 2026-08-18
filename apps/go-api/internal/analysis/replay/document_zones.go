package replay

// document_zones.go — L'ETAT DES ZONES : la forme que « qui tient quelle zone » prend dans
// l'artefact, et ce que la mesure a refuse d'y mettre.
//
// CHRONIQUE — v15 (2026-08-18, plan `.ai/V7.5/replay2d/PLAN_EXPLOITATION_REGISTRE_FILM.md`,
// lot C-bis phase 2b). Le document publie `zoneStates` — L'ETAT DE CHAQUE ZONE sur toute la
// partie, en intervalles — et `coverage.zones`, ses denominateurs. Le champ est optionnel, mais
// la version monte : la teinte de propriete cote client N'EXISTE que si l'artefact la porte, et
// la reprise du backfill se fait par SchemaVersion — un artefact v14 doit se voir « a re-cuire »,
// pas « a jour ».
//
// D'OU VIENT CE QUI EST PUBLIE, ET DE QUOI C'EST FAIT :
//
//	le CANAL       l'archetype `ti=13` du film (`managed-object-property-*`), porte au lot C-bis
//	               phase 1 et balaye par `filmdec.ScanFilmManagedProperties`. UN SLOT EST UNE
//	               PROPRIETE RESEAU NOMMEE, pas une zone : trois familles coexistent par zone —
//	               la JAUGE de capture (tag 3), le PROPRIETAIRE (tag 4) et un canal neutre.
//	la ZONE        le catalogue versionne d'objectifs de carte, fourni par l'appelant DANS
//	               L'ORDRE OU LE SERVICE LE SERT (cf. ZoneInput.Zones).
//	l'APPARIEMENT  slot -> zone PAR MATCH, par la coincidence d'un sommet de jauge avec une
//	               capture nommee attribuee geometriquement (`AttributeZones` sur la position du
//	               capteur). Coherence mesuree 93,1 % et 98,4 % (seuil 90 %), temoins a 41-48 %
//	               (permutation des slots) et 51-57 % (sommet decale de 20 s).
//	le PROPRIETAIRE la VALEUR du tag 4 : `0xFFFFFFFF` (personne), `0x0`, `0x1`. Concordance avec
//	               l'index d'equipe du capteur 100,0 % (48/48) et 91,1 % (51/56) hors emissions
//	               neutres, sur les trois slots canoniques — un par zone.
//	l'EQUIPE       le ROSTER (`ZoneInput.TeamByXUID`), JAMAIS le film : `game-engine-team-mapping`
//	               lit ses bits sans les publier (lot C-bis phase 1).
//
// CE QUE LA MESURE A REFUSE DE PUBLIER, ET C'EST LA MOITIE DU RESULTAT :
//
//	« CONTESTE ». La piste demandait la valeur du tag 4 pendant les rampes non abouties ; or les
//	slots de rampe ne portent PAS de tag 4 (les deux familles sont disjointes). La question est
//	VIDE sur ce corpus, et c'est ecrit plutot que contourne.
//
//	LA CLE DE NOMMAGE (tag 5) COMME IDENTITE DE ZONE. Elle est absente des slots de jauge de deux
//	zones sur trois et DIFFERE entre deux matchs de la meme carte sur la troisieme. La carte
//	slot -> zone tient par le NUMERO DE SLOT du match, pas par cette cle : `key` est donc publiee
//	quand elle existe, et n'est jamais la cle de jointure.
//
//	UNE CARTE SLOT -> ZONE DE CARTE. L'appariement se refait A CHAQUE MATCH, parce qu'il se fonde
//	sur les captures nommees de CE match. Un mode sans oracle nomme (KOTH, Oddball) n'a pas de
//	carte par captures — le volet colline passe par la GRAPPE des positions, et le dit
//	(`coverage.zones.method`).
//
// LA LIMITE QUI COMPTE POUR LE RENDU, ecrite ici parce qu'elle se verrait sinon comme un bug :
// `zoneRef` indexe `mapObjectives.zones`, que le SERVICE sert a la requete d'apres la table de
// roles du titre. En KOTH cette table ne sert AUCUN role (le catalogue de formes ne connait
// aucun role de colline — mesure de la phase 2a sur 6 cartes) : les intervalles de colline sont
// donc publies dans l'artefact et le client n'a, aujourd'hui, aucune zone ou les poser.
// `coverage.zones.roles` publie les roles employes pour que la jointure soit VERIFIABLE plutot
// que supposee.

// Les DEUX methodes d'appariement slot -> zone. Elles ne valent pas la meme chose, et le
// document le dit plutot que de laisser le client le deviner.
const (
	// ZoneMethodCaptures : la zone d'un slot vient des CAPTURES NOMMEES du statborg, attribuees
	// geometriquement a la position de leur auteur. C'est la methode mesuree a 93-98 %.
	ZoneMethodCaptures = "captures+geometry"
	// ZoneMethodPositions : aucun oracle nomme (KOTH). La zone d'une periode de garde vient de
	// la GRAPPE des positions pendant la montee de la jauge. Methode plus faible : sa nettete
	// est excellente sur un film, moyenne sur un autre, NULLE sur un troisieme (phase 2a).
	ZoneMethodPositions = "positions+geometry"
)

// ZoneState est L'ETAT D'UNE ZONE sur toute la partie : une suite d'intervalles.
//
// UNE ZONE, PAS UNE CAPTURE. Le regroupement est par OBJET, comme pour `flagCarries` : publier
// un intervalle par capture obligerait le client a reconstituer lui-meme la continuite entre
// « prise ici » et « reprise la ».
type ZoneState struct {
	// ZoneRef est l'index de la zone dans `mapObjectives.zones` — le calque STATIQUE que le
	// service sert a la requete. C'est la SEULE cle de jointure du calque.
	//
	// POURQUOI UN INDEX ET PAS UN IDENTIFIANT. Le DTO des objectifs statiques ne publie
	// volontairement aucun identifiant (la lettre A/B/C n'existe dans aucune donnee decodee) ;
	// l'ordre, lui, est deterministe — role par role, puis rang spatial. L'artefact et le
	// service construisent donc la meme liste, et `coverage.zones.roles` publie de quoi le
	// verifier.
	ZoneRef int `json:"zoneRef"`
	// Key est la cle de nommage du slot (tag 5) quand il en emet une, 0 sinon. TRACABILITE
	// SEULEMENT : elle n'est ni stable entre deux matchs de la meme carte, ni presente partout
	// (cf. l'en-tete). Ne jamais joindre dessus.
	Key uint32 `json:"key,omitempty"`
	// Spans est l'etat de la zone, en intervalles tries par T0 et sans recouvrement.
	Spans []ZoneSpan `json:"spans"`
}

// ZoneSpan est UN intervalle d'etat d'une zone.
type ZoneSpan struct {
	// T0 / T1 bornent l'intervalle en frames (meme axe que Point.T). T1 est INCLUS.
	T0 int `json:"t0"`
	T1 int `json:"t1"`
	// Owner est l'equipe qui TIENT la zone, ou `null` quand personne ne la tient (valeur
	// neutre `0xFFFFFFFF` du canal).
	//
	// POINTEUR ET SANS `omitempty` : le camp 0 existe, et « personne » doit se VOIR a `null` —
	// sinon « zone neutre » et « artefact plus ancien » se confondraient (meme regle que
	// `FlagSpan.XUID`).
	Owner *int `json:"owner"`
	// Progress est le SOMMET de la jauge de capture atteint pendant l'intervalle, ramene a
	// [0, 1]. Absent quand la zone n'a pas de slot de jauge apparie sur ce match, ou quand
	// aucune emission de jauge ne tombe dans l'intervalle.
	//
	// LA CONVERSION EST UNE CONVENTION, PAS UNE MESURE : le deser declare la plage [-100, +100]
	// (constantes `0x143cd8f84` / `0x143cd84a8` du jeu) et la valeur est ramenee lineairement.
	// Le quantum brut n'est pas republie — il n'aurait de sens qu'avec la table de largeurs.
	Progress *float32 `json:"progress,omitempty"`
	// Active dit que la zone est LA ZONE ACTIVE du mode pendant l'intervalle (colline de KOTH).
	// Faux partout dans les modes a zones simultanees (Bastion) : c'est `owner` qui y parle.
	Active bool `json:"active"`
}

// ZonesCoverage porte les denominateurs du calque. Sans eux, « 3 zones » se lirait comme une
// exhaustivite, et un film d'un autre mode serait indistinguable d'un film dont l'appariement a
// echoue.
//
// ELLE EST PUBLIEE MEME QUAND AUCUNE ZONE NE L'EST, pour la meme raison que `placements`,
// `groundWeapons`, `score` et `flagCarries`. Son ABSENCE dit encore autre chose : l'appelant n'a
// rien fourni a lire (pas de catalogue de zones, ou film non balaye).
type ZonesCoverage struct {
	// Method nomme l'appariement employe : [ZoneMethodCaptures] ou [ZoneMethodPositions].
	Method string `json:"method"`
	// Roles nomme les roles du catalogue qui composent `mapObjectives.zones`, DANS L'ORDRE et
	// separes par une virgule (`strongholds_zone`, ou `strongholds_zone,extraction_zone`).
	// C'est ce qui rend `zoneRef` verifiable au lieu d'etre suppose.
	//
	// UNE CHAINE ET NON UN TABLEAU, deliberement : ce champ est un TEMOIN de jointure que rien
	// ne parcourt. Le publier en tableau ferait entrer un tableau NULLABLE de plus dans le
	// contrat, donc une entree de plus dans la frontiere de nullabilite du client — un cout
	// reel (garde `replayContract.test.ts`) pour une donnee qui se lit d'un coup d'oeil.
	Roles string `json:"roles,omitempty"`
	// Catalog est le nombre de zones du catalogue (le denominateur de `paired`).
	Catalog int `json:"catalog"`
	// Slots est le nombre de slots `ti=13` qui emettent une valeur scalaire sur ce film.
	Slots int `json:"slots"`
	// Paired / Unpaired partagent les slots PORTEURS D'UNE JAUGE : ceux qu'une zone du
	// catalogue a recus, et ceux qu'aucune capture n'a permis de rattacher. Les seconds ne sont
	// PAS publies — un intervalle pose sur une zone devinee serait invisible et credible.
	Paired   int `json:"paired"`
	Unpaired int `json:"unpaired"`
	// Captures / Attributed : les captures nommees du film, et celles qu'une position a permis
	// d'attribuer a une zone. C'est le denominateur de l'appariement lui-meme.
	Captures   int `json:"captures"`
	Attributed int `json:"attributed"`
	// OwnerChecked / OwnerAgreed : LE CONTROLE INDEPENDANT du proprietaire. Pour chaque capture
	// attribuee, la valeur du tag 4 de la zone juste apres la capture est confrontee a l'equipe
	// du capteur (roster). `Checked` compte les confrontations possibles (valeur non neutre
	// dans la fenetre), `Agreed` celles ou la valeur EST l'index d'equipe.
	//
	// DEUX INTS PLUTOT QU'UN TAUX, regle du depot : un taux sans son denominateur ne se
	// verifie pas. La phase 2a a mesure 48/48 et 51/56.
	OwnerChecked int `json:"ownerChecked"`
	OwnerAgreed  int `json:"ownerAgreed"`
	// Spans est le nombre d'intervalles publies, toutes zones confondues.
	Spans int `json:"spans"`
	// HillPeriods est le nombre de periodes de COLLINE publiees (methode par positions). Zero
	// dans les modes a zones simultanees.
	HillPeriods int `json:"hillPeriods"`
	// UnknownOwner compte les emissions du canal de propriete dont la valeur n'est ni neutre ni
	// un index d'equipe connu. Elles n'ouvrent aucun intervalle : publier un camp qu'aucun
	// joueur n'occupe serait une invention, et la taire empecherait de la voir arriver.
	UnknownOwner int `json:"unknownOwner"`
}
