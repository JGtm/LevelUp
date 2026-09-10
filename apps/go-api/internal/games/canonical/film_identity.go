package canonical

// film_identity.go — LES TYPES DU REGISTRE D'IDENTITE DES ENTITES D'UN FILM.
//
// # POURQUOI ILS SONT CANONIQUES (inventaire P1, axe (h))
//
// `internal/games/canonical` ne portait AUCUN type d'entite de film : ni slot de bipede, ni
// objet d'objectif, ni vehicule. Le registre nait donc canonique parce qu'il n'y a rien a
// casser — ce sont des types de DONNEE inter-titres, pas des adapters. `internal/analysis/replay`
// les remplit pour Halo Infinite ; un autre titre les remplirait par son propre adapter sans
// toucher au consommateur. Aucune comparaison de slug nulle part : la porte reste la capability
// `film.replay_artifact`.
//
// # LA PROVENANCE EXISTAIT DEJA EN CINQ EXEMPLAIRES INCOMPATIBLES
//
// `NomParMort`/`NomParFermeture`, `BridgeHealth`, `vehicleResolvedBy`,
// `EquipmentPlacement.Origin`, `Pickup.Origin`, `ScoreIdentity*` : six facons de dire « d'ou
// vient ce lien », dont une seule n'etait meme pas publiee. [LinkSource] et [Link] les
// remplacent par UNE forme, et la regle « <= 2 copies » impose que toute nouvelle enumeration
// de provenance passe par ici.
//
// # CE QUE LA FORME GARANTIT
//
// Un lien PORTE SES BORNES. Un slot de bipede ou d'entite statborg est REATTRIBUE (a une
// reapparition, d'une manche a l'autre) : un lien aplati sur tout le match credite le premier
// occupant, et c'est exactement le defaut P0-2 du chantier. `From`/`To` interdisent de le
// rejouer.

// EntityKind nomme le TYPE d'entite d'un film qu'un lien identifie. Liste FERMEE : toute autre
// valeur est un defaut de producteur, pas une categorie.
type EntityKind string

const (
	// EntityPlayerIndex : l'index de joueur du film (5 bits precedant le xuid).
	EntityPlayerIndex EntityKind = "player_index"
	// EntityBipedSlot : le slot de bipede des trajectoires (ti=35).
	EntityBipedSlot EntityKind = "biped_slot"
	// EntityStatborgSlot : le slot d'entite statborg d'un joueur (10..24 pairs).
	EntityStatborgSlot EntityKind = "statborg_slot"
	// EntityTeamSlot : le slot d'entite statborg d'une equipe (6 et 8).
	EntityTeamSlot EntityKind = "team_slot"
	// EntityObjectiveObject : un objet d'objectif (drapeau, crane, bombe) — archetype ti=42.
	EntityObjectiveObject EntityKind = "objective_object"
	// EntityZone : une zone ou colline — archetype ti=13.
	EntityZone EntityKind = "zone"
	// EntityVehicle : un vehicule — archetype ti=40.
	EntityVehicle EntityKind = "vehicle"
	// EntityWorldItem : une arme au sol, un equipement, un socle — ti=42 et ti=37.
	EntityWorldItem EntityKind = "world_item"
	// EntityBot : un bot declare par BOT_METADATA (paquet type 12).
	EntityBot EntityKind = "bot"
)

// AllEntityKinds retourne la liste exhaustive des types d'entite du registre.
func AllEntityKinds() []EntityKind {
	return []EntityKind{EntityPlayerIndex, EntityBipedSlot, EntityStatborgSlot, EntityTeamSlot,
		EntityObjectiveObject, EntityZone, EntityVehicle, EntityWorldItem, EntityBot}
}

// IsKnownEntityKind valide qu'un EntityKind est dans l'enum canonique.
func IsKnownEntityKind(k EntityKind) bool {
	for _, known := range AllEntityKinds() {
		if k == known {
			return true
		}
	}
	return false
}

// LinkSource dit D'OU vient un lien du registre. Valeurs FERMEES ; toute autre valeur est un
// defaut de producteur, pas une categorie.
type LinkSource string

const (
	// LinkDirect : lu dans le film — un champ, un identifiant, une reference. C'est LA lecture,
	// et la doctrine « l'index est l'index » exige qu'elle soit employee a 100 % quand elle
	// existe, jamais remplacee par une deduction.
	LinkDirect LinkSource = "direct"
	// LinkCatalog : fige par le catalogue du titre ou de la carte (map_id, type_id, TOML).
	LinkCatalog LinkSource = "catalogue"
	// LinkExternal : vient de la base ou de la feuille de match — equipe, arrivee en cours,
	// roster complementaire. Ni `direct` (ce n'est pas le film), ni `deduit` (rien n'a ete
	// devine).
	LinkExternal LinkSource = "externe"
	// LinkInferred : pont par morts, fermeture, consensus, vote, geometrie, elimination.
	LinkInferred LinkSource = "deduit"
	// LinkUnresolved : rien n'a nomme ce lien. PUBLIE et COMPTE, jamais jete — c'est la
	// doctrine §0.2 : ce qui ne se resout pas se dit, il ne s'invente pas et il ne disparait pas.
	LinkUnresolved LinkSource = "non_resolu"
)

// AllLinkSources retourne la liste exhaustive des provenances.
func AllLinkSources() []LinkSource {
	return []LinkSource{LinkDirect, LinkCatalog, LinkExternal, LinkInferred, LinkUnresolved}
}

// IsKnownLinkSource valide qu'une LinkSource est dans l'enum canonique.
func IsKnownLinkSource(s LinkSource) bool {
	for _, known := range AllLinkSources() {
		if s == known {
			return true
		}
	}
	return false
}

// LinkMethod nomme la VOIE EXACTE a l'interieur d'une source. Il n'y a pas deux facons de
// deduire qui se valent : publier « deduit » sans dire par quoi rendrait la provenance
// inexploitable — c'est le defaut de `vehicleResolvedBy`, qui reste interne faute de forme.
//
// Vide pour un lien `direct` dont la source suffit a le decrire.
type LinkMethod string

const (
	// MethodFilmFooter : le pied de film nomme l'entite.
	MethodFilmFooter LinkMethod = "pied_de_film"
	// MethodPlayerIndexTable : les 5 bits precedant le xuid dans les chunks de replication.
	MethodPlayerIndexTable LinkMethod = "PlayerIndexTable"
	// MethodBotID : le BotID du paquet BOT_METADATA, qui EST le N de `bid(N.0)`.
	MethodBotID LinkMethod = "bid"
	// MethodDeathBridge : le pont par morts — la mort qui TERMINE une vie la nomme.
	MethodDeathBridge LinkMethod = "pont_par_morts"
	// MethodClosure : une fermeture de slot (le corps disponible, la reapparition).
	MethodClosure LinkMethod = "fermeture"
	// MethodPreviousLife / MethodNextLife : l'occupant nomme du MEME slot le plus proche dans
	// le temps, avant ou apres.
	MethodPreviousLife LinkMethod = "vie_precedente"
	MethodNextLife     LinkMethod = "vie_suivante"
	// MethodSlotBridge : le pont aplati par slot, en dernier repli.
	MethodSlotBridge LinkMethod = "pont_par_slot"
	// MethodSuccession : un relais de bot, date par l'instant de bascule lu dans la base.
	MethodSuccession LinkMethod = "relais"
	// MethodScoreboard : le TABLEAU DE L'API nomme le participant que le film ne table pas.
	//
	// Elle va avec [LinkExternal], jamais avec [LinkInferred] : rien n'est devine. L'index de
	// participant est LU dans le film ; ce que le tableau apporte est l'IDENTIFIANT publiable
	// (`bid(N.0)` d'un bot, que la table d'index ne peut pas porter faute de xuid) et les
	// BORNES DE PARTICIPATION qui departagent un siege d'index que deux participants successifs
	// occupent (un bot, puis l'humain qui le remplace en cours de partie).
	MethodScoreboard LinkMethod = "tableau_api"
	// MethodRosterElimination : il ne reste qu'une affectation possible entre un xuid libre et
	// un slot sans nom. Ce n'est pas une deduction par ressemblance, c'est un appariement force.
	MethodRosterElimination LinkMethod = "elimination_roster"
	// MethodTemporalExclusion : la MEME elimination, portee de « tout le match » a
	// « l'intervalle d'une vie ». Un joueur n'occupe qu'un slot a la fois : les joueurs qu'une
	// vie nommee place ailleurs pendant l'intervalle sont exclus, et s'il n'en reste qu'un,
	// c'est lui. Deux candidats, ou zero (la lecture se contredit) : on se tait.
	MethodTemporalExclusion LinkMethod = "exclusion_temporelle"
	// MethodDeathInstants : les instants de mort apparies au fil (pont statborg par manche).
	MethodDeathInstants LinkMethod = "instants_de_mort"
	// MethodSheetTriplet : le triplet K/D/A confronte a la feuille de match.
	MethodSheetTriplet LinkMethod = "triplet_feuille"
	// MethodContested : deux candidats subsistent, rien ne les departage. Va toujours avec
	// [LinkUnresolved] : l'identite n'est pas absente, elle est INDECIDABLE.
	MethodContested LinkMethod = "conteste"
	// MethodBipedCreation : le record de CREATION du bipede porte l'index de participant de son
	// proprietaire (`ECS_ReadEntityRefIndex5`, +67 bits apres l'en-tete NEW `ti=35`). C'est une
	// LECTURE du film, pas une deduction : elle va avec [LinkDirect].
	MethodBipedCreation LinkMethod = "creation_bipede"
	// MethodBipedCreationPropagated : le MEME record, applique a une autre vie du MEME corps.
	//
	// CE N'EST PAS UNE DEDUCTION, et c'est pourquoi elle va aussi avec [LinkDirect] : un film
	// porte UN SEUL record de creation par slot de bipede (mesure du lot E2 : 538 vies sur cinq
	// films, 0 slot a deux records, generation invariante). Une vie sans record propre est donc
	// le MEME corps qu'une decoupe a `lifeGapUS` a separee d'un sejour continu — pas un autre
	// occupant qu'on devinerait.
	MethodBipedCreationPropagated LinkMethod = "creation_bipede_propagee"
	// Les TROIS CAUSES de non-resolution d'un slot de bipede. Elles vont toujours avec
	// [LinkUnresolved] et ventilent [BipedLinkCounts.UnresolvedByCause] : « non resolu » sans sa
	// cause ne se corrige pas, il se contemple.
	//
	//	MethodIndexOutOfTable    l'index LU n'est pas dans la table publiee (participant que
	//	                         `PlayerIndexTable` ne nomme pas — bot non declare, verdict I0).
	//	MethodNoCreationRecord   aucun record de creation n'a ete lu sur ce corps.
	//	MethodDivergentReadings  deux records du meme slot portent des index DIFFERENTS : la
	//	                         propagation deviendrait un choix, donc on se tait.
	MethodIndexOutOfTable   LinkMethod = "index_hors_table"
	MethodNoCreationRecord  LinkMethod = "sans_record"
	MethodDivergentReadings LinkMethod = "lectures_divergentes"
	// MethodNone : aucune voie — le lien n'a jamais eu de candidat.
	MethodNone LinkMethod = ""
)

// Link est UN lien du registre : d'ou il vient, par quelle voie, sur quelle preuve, et entre
// quelles bornes il vaut.
type Link struct {
	// Source dit d'ou vient le lien. Jamais vide : un lien sans provenance est un lien qu'on
	// ne peut pas juger.
	Source LinkSource `json:"source"`
	// Method nomme la voie exacte a l'interieur de la source.
	Method LinkMethod `json:"method,omitempty"`
	// Readings porte le DENOMINATEUR de la voie quand elle en a un — lectures concordantes
	// pour `direct`, morts appariees pour le pont, votes pour un modal. Zero quand la voie
	// n'en produit pas.
	Readings int `json:"readings,omitempty"`
	// Metric porte la grandeur continue de la voie quand elle en a une (distance en metres
	// pour la geometrie, ecart en millisecondes pour un appariement). Nul sinon.
	Metric *float64 `json:"metric,omitempty"`
	// From / To bornent la validite du lien sur l'axe du rejeu (index de frame, bornes
	// INCLUSIVES). Un lien valable tout le match porte [0, frameCount-1].
	//
	// C'EST CE COUPLE QUI INTERDIT DE REJOUER LE DEFAUT P0-2 : un slot recycle porte DEUX
	// liens bornes, jamais un lien aplati qui crediterait son premier occupant.
	From int `json:"from"`
	To   int `json:"to"`
}

// EntityRef designe une entite du film : son type, et son identifiant DANS CE FILM.
//
// L'identifiant est une chaine parce que les espaces de cles different d'un type a l'autre
// (un slot est un entier, un objet d'objectif un couple `slot:gen`, un bot un `bid(N.0)`) et
// qu'aucun d'eux ne se compare a un autre. Les melanger dans un int serait la meme faute que
// confondre un slot de bipede et un slot de statborg.
type EntityRef struct {
	Kind EntityKind `json:"kind"`
	ID   string     `json:"id"`
}

// LinkCounts compte les liens d'un type d'entite PAR PROVENANCE. C'est le recapitulatif que le
// gate corpus compare : il echoue si un compte `direct` BAISSE, ou si un compte `deduit` monte
// a `direct` constant.
type LinkCounts struct {
	Direct     int `json:"direct"`
	Catalog    int `json:"catalogue"`
	External   int `json:"externe"`
	Inferred   int `json:"deduit"`
	Unresolved int `json:"non_resolu"`
}

// Add incremente le compteur de la provenance. Une provenance inconnue n'est PAS silencieuse :
// elle n'incremente rien et [LinkCounts.Total] cesse alors d'egaler le nombre de liens, ce que
// l'invariant du producteur verifie.
func (c *LinkCounts) Add(s LinkSource) {
	switch s {
	case LinkDirect:
		c.Direct++
	case LinkCatalog:
		c.Catalog++
	case LinkExternal:
		c.External++
	case LinkInferred:
		c.Inferred++
	case LinkUnresolved:
		c.Unresolved++
	}
}

// Total rend le nombre de liens comptes, toutes provenances confondues.
func (c LinkCounts) Total() int {
	return c.Direct + c.Catalog + c.External + c.Inferred + c.Unresolved
}

// BipedLinkCounts est [LinkCounts] pour la famille du SLOT DE BIPEDE, avec les deux
// ventilations que cette famille — et elle seule — sait produire.
//
// # POURQUOI UNE FORME A PART PLUTOT QUE DEUX CHAMPS DE PLUS DANS `LinkCounts`
//
// `direct_propage` et `non_resolu_par_cause` n'ont de sens que la ou UN MEME record couvre
// plusieurs liens et ou l'echec porte une cause instruite. L'index de joueur (une table lue une
// fois) et le slot de statborg (un pont par manche) n'ont ni l'un ni l'autre : leur servir deux
// objets vides ferait croire a une ventilation qui n'existe pas.
//
// [LinkCounts] est EMBARQUE, donc `direct` / `deduit` / `non_resolu` restent au meme niveau dans
// le JSON et le gate corpus continue de les comparer sans rien savoir de ce type.
type BipedLinkCounts struct {
	LinkCounts
	// DirectPropagated est la PART de `Direct` obtenue en appliquant le record de creation d'un
	// corps a une autre vie du MEME corps. Sous-compte, jamais un total a part : le brief du lot
	// E2 dit « direct (dont propages) », et un gate qui surveille `direct` doit voir la somme.
	DirectPropagated int `json:"direct_propage"`
	// UnresolvedByCause ventile `Unresolved`. La somme de ses champs EGALE `Unresolved` : un
	// non-resolu sans cause serait exactement le silence que le registre existe pour interdire.
	UnresolvedByCause UnresolvedCauses `json:"non_resolu_par_cause"`
}

// UnresolvedCauses ventile les liens non resolus d'un slot de bipede par CAUSE PROUVEE.
type UnresolvedCauses struct {
	// IndexOutOfTable : l'index est LU, il n'est simplement pas dans la table publiee.
	IndexOutOfTable int `json:"index_hors_table"`
	// NoCreationRecord : aucun record de creation n'a ete lu sur ce corps.
	NoCreationRecord int `json:"sans_record"`
	// DivergentReadings : deux records du meme slot portent des index differents.
	DivergentReadings int `json:"lectures_divergentes"`
}

// Total rend le nombre de causes comptees — l'invariant que le producteur verifie contre
// `Unresolved`.
func (c UnresolvedCauses) Total() int {
	return c.IndexOutOfTable + c.NoCreationRecord + c.DivergentReadings
}

// AddCause incremente la cause que la voie designe. Une voie qui n'est pas une cause de
// non-resolution n'incremente RIEN : la somme cesse alors d'egaler `Unresolved`, et c'est
// l'invariant du producteur qui le dit — pas un silence.
func (c *UnresolvedCauses) AddCause(m LinkMethod) {
	switch m {
	case MethodIndexOutOfTable:
		c.IndexOutOfTable++
	case MethodNoCreationRecord:
		c.NoCreationRecord++
	case MethodDivergentReadings:
		c.DivergentReadings++
	}
}
