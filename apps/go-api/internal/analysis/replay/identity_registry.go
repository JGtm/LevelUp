package replay

// identity_registry.go — LE REGISTRE D'IDENTITE DU FILM : LE SEUL PRODUCTEUR DE LIENS.
//
// # CE FICHIER EST L'UNIQUE ENTREE AUTORISEE DU PONT (garde-rail date, lot P2, 2026-09-08)
//
// Jusqu'ici, quinze calques et deux lecteurs hors rejeu appelaient `buildOwners` /
// `ResolveSlotXUID` ou lisaient `OwnerReport.SlotXUID` / `.Owner` directement. Chacun avait sa
// propre garde (ou pas) contre les slots ambigus, son propre repli, sa propre facon de traiter
// un slot recycle. Le registre reprend cette responsabilite : il compose les lectures, applique
// les replis DANS UN ORDRE ECRIT, et n'expose que des accesseurs qui portent deja leurs gardes.
// `internal/archlint/no_identity_bridge_outside_registry_test.go` interdit toute autre entree.
//
// # « L'INDEX EST L'INDEX » (doctrine du plan v2 §0.7, amendee par D11)
//
// Le film porte des liens DIRECTS : l'index de joueur ecrit dans les chunks de replication
// (`PlayerIndexTable`), et le `BotID` du paquet BOT_METADATA qui EST le N de `bid(N.0)`. Ils
// sont poses D'ABORD et a 100 %, jamais remplaces par une deduction. Le pont par morts ne vient
// qu'ENSUITE, pour le seul lien que le film ne donne pas (slot de bipede -> joueur : rien dans
// `BipedPosition` ne porte d'identite), et il VERIFIE les liens directs — un desaccord alarme et
// se compte, il n'invente jamais un nom.
//
// # LA FONCTION EST PURE, ET C'EST UNE CONTRAINTE D'ARCHITECTURE (decision D11)
//
// Aucune I/O, aucune base, aucun `filmsource.Film` : l'entree est ce que le DECODAGE a deja
// rendu. C'est ce qui permet aux DEUX producteurs de l'appeler — `replaybuild` a la cuisson, et
// `sync/killcollector` au sync, ou les donnees d'un match sont deja completes. Une seule
// fonction, donc un seul nommage : deux tables du meme film ne peuvent plus diverger.

import (
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/objectiveevents"
)

// IdentityClock est l'axe de frames du document, quand l'appelant en a un.
//
// LE COLLECTEUR DE SYNC N'EN A PAS, et c'est normal : il ne publie pas d'artefact, il ecrit des
// vies et des contextes de mort sur l'horloge du MATCH. `StepUS` a zero signifie « aucun axe de
// frames » : le registre construit alors ses ponts sans publier de section — les liens n'auraient
// pas de bornes exprimables, et un lien sans bornes est exactement ce que le defaut P0-2 a coute.
type IdentityClock struct {
	// OriginUS est l'instant, sur l'horloge du film, de la frame 0.
	OriginUS uint64
	// StepUS est la duree d'une frame en microsecondes. Zero = pas d'axe.
	StepUS uint64
	// FrameCount est le nombre de frames publiees.
	FrameCount int
}

// IdentityInput est TOUT ce que le registre consomme : des LECTURES deja faites, jamais un
// document cuit ni un fichier.
//
// Une structure unique parce que le depot borne a cinq parametres, et parce qu'un appelant qui
// ajoute une source ne doit pas casser les autres.
type IdentityInput struct {
	// Positions : les positions de bipede DEJA decodees (`filmdec.ScanBipedPositions`).
	Positions []filmdec.BipedPosition
	// BipedCreations : les records de CREATION de bipede deja decodes
	// (`filmdec.ScanBipedCreations`). C'est le lien DIRECT corps -> joueur : le film ECRIT
	// l'index de participant du proprietaire dans le default-state du record.
	//
	// VIDE = LE REGISTRE N'A AUCUNE LECTURE DIRECTE, et il le publie
	// (`BridgeHealth.BridgeNamedLives` non nul). Ce n'est pas une option : c'est la degradation
	// declaree d'un producteur qui ne porte pas encore ce canal.
	BipedCreations []filmdec.BipedCreation
	// Deaths : le fil des morts du film — il nomme chaque vie par sa victime.
	Deaths []Death
	// PlayerIndices : le lien DIRECT identite -> index de joueur, lu dans les chunks de
	// replication (cf. player_index.go).
	PlayerIndices PlayerIndexTable
	// Bots : les bots declares par BOT_METADATA, avec leur `BotID` (le N de `bid(N.0)`).
	Bots []BotIdentity
	// Fire : les evenements de tir, pour la fermeture A (un joueur qui agit a un corps).
	Fire []FireEventRef
	// RosterXUIDs : le roster COMPLET du match, tel que la base le connait. Source EXTERNE :
	// c'est lui qui rend l'ELIMINATION possible pour un joueur qui ne meurt jamais.
	RosterXUIDs []uint64
	// Participants : le TABLEAU DE L'API — les participants du match sous l'identifiant de la
	// base (`bid(N.0)` pour un bot) et leurs bornes de participation. VIDE = le producteur n'a
	// pas de base : la voie `tableau_api` se tait entierement, et elle le publie (cf.
	// identity_registry_scoreboard.go).
	Participants []Participant
	// Statborg : l'identite des slots d'entite statborg, resolue par manche par l'appelant.
	// Le registre ne la RECALCULE pas — il la PUBLIE avec sa provenance.
	Statborg StatborgIdentityInput
	// Clock : l'axe de frames du document. Zero = le registre ne publie pas de section.
	Clock IdentityClock
	// MatchID sert aux journaux — jamais a une decision.
	MatchID string
}

// StatborgIdentityInput porte l'identite des slots d'entite statborg et les enregistrements qui
// en sont le DENOMINATEUR.
//
// POURQUOI ELLE ARRIVE RESOLUE. La resolution vit dans `objectiveevents`, feuille du decodage,
// et deux calques la partagent deja, memorisee (cf. `replaybuild.pontParManche`). La recalculer
// ici serait un second deroulage complet du compteur de morts par cuisson — le cout que la
// memorisation existe pour eviter.
type StatborgIdentityInput struct {
	Identity objectiveevents.RoundIdentity
	Records  []objectiveevents.StatRecord
}

// IdentityRegistry est la table d'identite d'UN film : les liens, leur provenance, et les
// accesseurs que les calques consomment.
//
// LES ACCESSEURS PORTENT LEURS GARDES. `PontEpure` retire les slots ambigus ; `XUIDAt` /
// `XUIDNumAt` preferent la vie qui couvre l'instant et s'abstiennent sur un siege ambigu ;
// `PontEtabli` ne dit QUE si un pont existe. Le pont APLATI n'a plus d'accesseur du tout depuis
// le lot 6.1. Un calque ne peut plus oublier une garde : il n'a plus de quoi l'enfreindre.
type IdentityRegistry struct {
	// own est le pont brut. INTERNE, et il le reste : c'est tout l'objet du garde-rail.
	own OwnerReport
	// Section est ce qui se PUBLIE — les liens et leur couverture (cf.
	// identity_registry_section.go). Vide quand l'appelant n'a pas d'axe de frames.
	Section IdentitySection
	// deducedLives : indices, dans `own.lives`, des vies dont l'identite est une DEDUCTION du
	// registre (elimination sur le roster). Elles etablissent qu'un joueur etait la, jamais
	// qu'un autre n'y etait pas — les lecteurs qui prouvent une ABSENCE doivent les distinguer.
	deducedLives map[int]bool
	// eliminated compte les vies nommees par elimination sur le roster.
	eliminated int
	// eliminatedSlot / eliminatedXUID : le couple retenu par l'elimination, pour le journal et
	// pour le lien publie. Zero quand l'elimination ne s'est pas appliquee.
	eliminatedSlot uint32
	eliminatedXUID uint64
	// excluded compte les vies nommees par EXCLUSION TEMPORELLE (cf.
	// identity_registry_exclusion.go) ; excludedContradictions compte celles qu AUCUN joueur ne
	// peut occuper — une contradiction de la lecture, pas une abstention ordinaire.
	excluded, excludedContradictions int
	// creation porte ce que la LECTURE DIRECTE du record de creation a pose et refuse, avec la
	// cause de chaque refus (cf. identity_registry_creation.go).
	creation creationReport
	// bridge porte la confrontation du pont par morts au lien direct : concordances,
	// discordances, et le compte des vies que le pont a NOMMEES — non nul seulement quand le
	// registre n'a recu aucune lecture directe (cf. identity_registry_bridge.go).
	bridge bridgeVerification
	// tableau porte ce que le TABLEAU DE L'API a nomme et refuse — la voie qui ferme les corps
	// dont l'index est lu mais hors de la table publiee (cf. identity_registry_scoreboard.go).
	tableau scoreboardReport
}

// BuildIdentityRegistry construit le registre d'identite du film. PURE : aucune I/O.
//
// L'ORDRE DES ETAPES EST LA DOCTRINE, ET IL N'EST PAS NEGOCIABLE :
//
//  1. les liens DIRECTS sont poses D'ABORD et a 100 % — index de joueur <-> xuid, `bid` <-> bot,
//     et depuis le lot E2 le lien CORPS <-> JOUEUR que le record de creation du bipede ECRIT
//     (`identity_registry_creation.go`), propage aux autres vies du meme corps ;
//  2. le PONT PAR MORTS ne nomme plus : il pose la CAUSE de fin et VERIFIE le lien direct — une
//     discordance s'inscrit et alarme, elle n'ecrase jamais (`identity_registry_bridge.go`) ;
//  2. bis. le TABLEAU DE L'API nomme les corps dont l'index est LU mais hors de la table publiee
//     — les bots que `BOT_METADATA` declare, et les sieges d'index qu'un arrivant en cours prend
//     a un bot (`identity_registry_scoreboard.go`). Il vient APRES le pont parce que sa fenetre
//     de participation a besoin du calage que le pont mesure ;
//  3. l'ELIMINATION SUR LE ROSTER ferme le cas d'unicite — un seul xuid sans vie, un seul slot
//     sans nom ;
//  4. l'EXCLUSION TEMPORELLE porte la meme elimination a l'echelle d'UNE VIE — un joueur
//     n'occupe qu'un slot a la fois, donc une vie dont un seul joueur du roster est libre sur
//     tout l'intervalle lui revient (lot P2-bis) ;
//  5. ce qui resiste est publie « non resolu » AVEC SA CAUSE et COMPTE, avec son alarme.
func BuildIdentityRegistry(in IdentityInput) IdentityRegistry {
	reg := IdentityRegistry{deducedLives: map[int]bool{}}
	reg.own, reg.creation, reg.bridge = buildOwners(in)
	reg.resolveByScoreboard(in)
	reg.resolveByRosterElimination(in)
	reg.resolveByTemporalExclusion(in)
	reg.Section = buildIdentitySection(reg, in)
	return reg
}

// Vies rend les vies decoupees et nommees, telles que le registre les a laissees.
func (r IdentityRegistry) Vies() []lifeSpan { return r.own.lives }

// PontEtabli dit si le registre a pu ponter AU MOINS UN siege — la question que se posaient les
// gardes qui testaient `len(PontParSlot()) == 0` avant le lot 6.1.
//
// ELLE NE SERT QU'A SE TAIRE, jamais a nommer : un calque qui la trouve fausse n'a rien a
// publier. Le nommage passe par [IdentityRegistry.XUIDAt] / [IdentityRegistry.XUIDNumAt] (a
// l'instant) ou [IdentityRegistry.PontEpure] (pour une piste entiere).
func (r IdentityRegistry) PontEtabli() bool { return len(r.own.SlotXUID) > 0 }

// LE PONT APLATI N'A PLUS D'ACCESSEUR (lot 6.1, 2026-09-10). `PontParSlot` rendait
// `OwnerReport.SlotXUID` tel quel — le PREMIER occupant d'un siege recycle, quel que soit
// l'instant demande — sous une exemption ecrite « ramassages, marques de portage, frags sous
// equipement actif ». Ses six lecteurs connaissaient tous l'instant de leur lecture ; ils sont
// passes a `XUIDNumAt`. Mesure qui a commande le retrait :
// `.ai/V7.5/RAPPORT_PONT_APLATI_2026-09-10.md` (un siege ambigu sur 74 films, deux ramassages
// publies sous le nom d'un joueur que le film place ailleurs). Garde-rail contre la
// reintroduction : `internal/archlint/no_identity_bridge_outside_registry_test.go`.

// IndexParSlot rend le pont slot -> INDEX DE JOUEUR du film. C'est la forme qu'attendent les
// rattachements d'EVENEMENTS, qui portent un index et non un xuid.
func (r IdentityRegistry) IndexParSlot() map[uint32]int { return r.own.Owner }

// PontEpure rend le pont slot -> xuid DEBARRASSE DES SLOTS AMBIGUS — celui que doit employer
// tout lecteur qui s'en sert pour NOMMER une piste (cf. `OwnerReport.NamingBridge`).
func (r IdentityRegistry) PontEpure() map[uint32]uint64 { return r.own.NamingBridge() }

// SlotsAmbigus rend les slots dont les vies nommees designent des joueurs DIFFERENTS.
func (r IdentityRegistry) SlotsAmbigus() map[uint32]bool { return r.own.SlotAmbiguous }

// PontDeSlot rend le xuid que le pont aplati donne a ce slot — VIDE si le slot est AMBIGU.
//
// Le pont garde le PREMIER occupant nomme d'un slot que deux joueurs se partagent : le servir
// publierait un nom arbitraire sur une vie que la lecture n'a pas nommee. C'est la meme
// abstention que [IdentityRegistry.XUIDAt], et pour la meme raison.
func (r IdentityRegistry) PontDeSlot(slot uint32) string {
	if r.own.SlotAmbiguous[slot] {
		return ""
	}
	if x, ok := r.own.SlotXUID[slot]; ok && x != 0 {
		return strconv.FormatUint(x, 10)
	}
	return ""
}

// scoreRecordsOf rend les enregistrements de statborg que l'appelant a deja decodes, ou rien.
// Le registre ne decode jamais : il PUBLIE ce que la lecture a rendu.
func scoreRecordsOf(in *ScoreInput) []objectiveevents.StatRecord {
	if in == nil {
		return nil
	}
	return in.Records
}

// XUIDAt rend le joueur qui OCCUPE ce slot a cet instant du film (microsecondes) : la vie qui
// couvre l'instant si elle est nommee, sinon le pont par slot — et rien du tout sur un slot
// ambigu. Chaine vide = personne ne le nomme.
func (r IdentityRegistry) XUIDAt(slot uint32, tUS uint64) string {
	x := r.own.xuidNumAt(slot, tUS)
	if x == 0 {
		return ""
	}
	return strconv.FormatUint(x, 10)
}

// XUIDNumAt est [IdentityRegistry.XUIDAt] EN NUMERIQUE — zero quand personne ne nomme le slot a
// cet instant.
//
// POURQUOI LES DEUX FORMES. Les calques qui PUBLIENT une identite l'ecrivent en chaine (c'est le
// contrat de l'artefact) ; ceux qui la JOIGNENT — un frag a son episode, une lecture d'inventaire
// aux morts de son porteur, une position a son corps — travaillent sur `uint64`, comme le fil des
// morts et le pont. Leur faire formater puis reparser une chaine par lecture serait un aller-retour
// pur, et c'est exactement ce que les lecteurs du pont aplati evitaient en le lisant a nu.
func (r IdentityRegistry) XUIDNumAt(slot uint32, tUS uint64) uint64 {
	return r.own.xuidNumAt(slot, tUS)
}

// DeathOffsetMS rend le calage du fil des morts sur l'horloge du film
// (`horlogeFilm = horlogeMatch + DeathOffsetMS`).
func (r IdentityRegistry) DeathOffsetMS() int64 { return r.own.DeathOffsetMS }

// PontPubliable dit si le NOMMAGE des vies est assez sur pour qu'on en tire des faits ECRITS EN
// BASE. Cf. death_context.go pour les deux criteres et ce qu'ils refusent.
func (r IdentityRegistry) PontPubliable() bool {
	return r.own.IndexDisagreements == 0 && len(r.own.SlotXUID) > 0
}

// VieDeduite dit que la vie d'indice `i` porte une identite DEDUITE par le registre. Les
// lecteurs qui prouvent une ABSENCE (le gate de presence des portages) doivent s'en abstenir :
// une deduction ajoute une presence, elle ne retire jamais celle d'un autre.
func (r IdentityRegistry) VieDeduite(i int) bool { return r.deducedLives[i] }

// TracesDeduites rend les INDICES des pistes que le registre a nommees par deduction, en
// appariant chaque piste a la vie deduite du meme slot qui la recouvre.
//
// POURQUOI ICI ET PAS CHEZ L'APPELANT : c'est le registre qui sait laquelle de ses vies est une
// lecture et laquelle une deduction. Le faire ailleurs demanderait de republier `deducedLives`,
// et une seconde lecture de la meme table est exactement ce que ce fichier existe pour empecher.
func (r IdentityRegistry) TracesDeduites(tracks []Track, origin, step uint64) map[int]bool {
	if len(r.deducedLives) == 0 {
		return map[int]bool{}
	}
	return r.tracesDontLaVie(tracks, origin, step, func(li int, _ lifeSpan) bool {
		return r.deducedLives[li]
	})
}

// TracesCloturesParMort rend les INDICES des pistes dont la vie se termine par une MORT LUE
// (`CauseVieMort`) — la seule fin que le fil des morts date.
//
// POURQUOI ELLE EXISTE (correctif E2-bis). Les lecteurs de durée qui recousent un silence de
// réplication (`equipment_episodes.spanFor`) doivent s'arrêter à une mort et à elle seule : un
// état actif qui enjambe une mort est une mesure FAUSSE, un état coupé à un simple trou est une
// mesure incomplète. Ils lisaient cette frontière dans « la vie porte un nom », proxy exact tant
// que le fil des morts était la SEULE voie de nommage — `nameLivesByDeaths` posait le xuid de la
// victime sur la vie que sa mort achève. Depuis le lot E2, le FILM nomme les vies à leur
// CRÉATION : toutes portent un nom, et le proxy déclare une mort à chaque trou de réplication.
// Mesure : `084a804d` slot 620, camo `[3105..3672]` retombé à `[3105..3120]` — 552 frames sur un
// épisode que rien n'interrompt. La cause de fin est DANS le registre : c'est lui qui la sert.
func (r IdentityRegistry) TracesCloturesParMort(tracks []Track, origin, step uint64) map[int]bool {
	return r.tracesDontLaVie(tracks, origin, step, func(_ int, l lifeSpan) bool {
		return l.cause == CauseVieMort
	})
}

// tracesDontLaVie apparie chaque piste à une vie du MÊME slot qui la recouvre et retient les
// pistes dont cette vie satisfait le prédicat. Le corps commun de [IdentityRegistry.TracesDeduites]
// et [IdentityRegistry.TracesCloturesParMort] : deux appariements identiques à vingt lignes d'écart
// divergeraient.
func (r IdentityRegistry) tracesDontLaVie(tracks []Track, origin, step uint64,
	garde func(int, lifeSpan) bool) map[int]bool {
	out := map[int]bool{}
	for i := range tracks {
		from, to := trackSpanUS(tracks[i], origin, step)
		for li, l := range r.own.lives {
			if l.slot != tracks[i].Slot || !garde(li, l) {
				continue
			}
			if minI64(to, l.to) >= maxI64(from, l.from) {
				out[i] = true
				break
			}
		}
	}
	return out
}

// DeathOffsetMatches rend le nombre de morts que le calage apparie — le DENOMINATEUR sans lequel
// un calage ne se juge pas (zero = le pont n'a pas ete construit, ou l'affinage n'a rien apparie).
func (r IdentityRegistry) DeathOffsetMatches() int { return r.own.DeathOffsetMatches }

// ViesNommeesParLaLecture rend le nombre de vies que le FIL DES MORTS a nommées — la lecture
// seule, avant toute déduction. Zéro = le pont n'a pas été construit.
func (r IdentityRegistry) ViesNommeesParLaLecture() int { return r.own.DeathsNamed }

// FermeturesParTir / FermeturesParReapparition / FermeturesContestees / FermeturesRefusees :
// ce que les fermetures ont ajouté et refusé (cf. closures.go). Publiés par la santé du pont.
func (r IdentityRegistry) FermeturesParTir() int          { return r.own.Closures.byShot }
func (r IdentityRegistry) FermeturesParReapparition() int { return r.own.Closures.byRespawn }
func (r IdentityRegistry) FermeturesContestees() int      { return r.own.Closures.contested }
func (r IdentityRegistry) FermeturesRefusees() int        { return r.own.Closures.refused }

// SlotsParLaLecture rend le nombre de slots que la LECTURE SEULE a nommés (avant fermetures).
func (r IdentityRegistry) SlotsParLaLecture() int { return r.own.FromDeaths }

// ViesParCreation / ViesParCreationPropagee : les vies que le RECORD DE CREATION nomme, selon
// qu'il OUVRE cette vie ou un AUTRE sejour du meme corps. La seconde est un
// SOUS-COMPTE de la couverture directe, jamais un total a part.
func (r IdentityRegistry) ViesParCreation() int         { return r.creation.Direct }
func (r IdentityRegistry) ViesParCreationPropagee() int { return r.creation.Propagated }
func (r IdentityRegistry) CorpsAvecCreation() int       { return r.creation.Slots }
func (r IdentityRegistry) LecturesDIndexDeBot() int     { return r.creation.IndexBot }

// ViesNommeesParLeTableau / ViesConflitAuTableau / ViesSansCandidatAuTableau : ce que le TABLEAU
// DE L'API a nomme et ce qu'il a refuse, par cause (cf. identity_registry_scoreboard.go).
func (r IdentityRegistry) ViesNommeesParLeTableau() int   { return r.tableau.Nommees() }
func (r IdentityRegistry) ViesConflitAuTableau() int      { return r.tableau.Conflits }
func (r IdentityRegistry) ViesSansCandidatAuTableau() int { return r.tableau.SansCandidat }
func (r IdentityRegistry) PontConcordant() int            { return r.bridge.Concordant }
func (r IdentityRegistry) PontDiscordant() int            { return r.bridge.Discordant }
func (r IdentityRegistry) ViesNommeesParLePont() int      { return r.bridge.NamedByBridge }

// ViesTotal / LecturesIndex / DesaccordsIndex / CollisionsDeSlot / CalageSecond : les
// dénominateurs et les témoins que la couverture publie.
func (r IdentityRegistry) ViesTotal() int        { return r.own.LivesTotal }
func (r IdentityRegistry) LecturesIndex() int    { return r.own.IndexReadings }
func (r IdentityRegistry) DesaccordsIndex() int  { return r.own.IndexDisagreements }
func (r IdentityRegistry) CollisionsDeSlot() int { return r.own.SlotCollisions }
func (r IdentityRegistry) CalageSecond() int     { return r.own.DeathOffsetRunnerUp }

// CalageSiConnu rend le calage du fil des morts, ou nil quand il n'est pas CONNU — pas seulement
// quand il vaut zéro (cf. BridgeHealth.DeathOffsetMs, lot M1b).
//
// LE TÉMOIN DE CONNAISSANCE EST `DeathOffsetMatches > 0`, PAS `DeathOffsetMS != 0`. Un calage à
// zéro exact (horloges déjà alignées) est une mesure valide qu'il ne faut pas confondre avec son
// absence.
func (r IdentityRegistry) CalageSiConnu() *int64 {
	if r.own.DeathOffsetMatches <= 0 {
		return nil
	}
	v := r.own.DeathOffsetMS
	return &v
}
