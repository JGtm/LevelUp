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
// LES ACCESSEURS PORTENT LEURS GARDES. `PontEpure` retire les slots ambigus, `XUIDAt` prefere la
// vie qui couvre l'instant au pont aplati, `PontParSlot` ne sert que les consommateurs dont
// l'exemption est ecrite. Un calque ne peut plus oublier une garde : il n'a plus de quoi
// l'enfreindre.
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
}

// BuildIdentityRegistry construit le registre d'identite du film. PURE : aucune I/O.
//
// L'ORDRE DES ETAPES EST LA DOCTRINE, ET IL N'EST PAS NEGOCIABLE :
//
//  1. les liens DIRECTS (index de joueur <-> xuid, `bid` <-> bot) sont poses tels quels ;
//  2. le PONT PAR MORTS nomme les vies que le film ne nomme pas, et VERIFIE les liens directs ;
//  3. l'ELIMINATION SUR LE ROSTER ferme le cas d'unicite — un seul xuid sans vie, un seul slot
//     sans nom ;
//  4. l'EXCLUSION TEMPORELLE porte la meme elimination a l'echelle d'UNE VIE — un joueur
//     n'occupe qu'un slot a la fois, donc une vie dont un seul joueur du roster est libre sur
//     tout l'intervalle lui revient (lot P2-bis) ;
//  5. ce qui resiste est publie « non resolu » et COMPTE, avec son alarme.
func BuildIdentityRegistry(in IdentityInput) IdentityRegistry {
	reg := IdentityRegistry{deducedLives: map[int]bool{}}
	reg.own = buildOwners(indexBySlot(in.Positions), in.Deaths, in.PlayerIndices, in.Fire)
	reg.resolveByRosterElimination(in)
	reg.resolveByTemporalExclusion(in)
	reg.Section = buildIdentitySection(reg, in)
	return reg
}

// Vies rend les vies decoupees et nommees, telles que le registre les a laissees.
func (r IdentityRegistry) Vies() []lifeSpan { return r.own.lives }

// PontParSlot rend le pont APLATI slot -> xuid.
//
// IL GARDE LE PREMIER OCCUPANT D'UN SLOT RECYCLE, et c'est pourquoi il ne doit servir qu'aux
// consommateurs dont l'exemption est ecrite (ramassages, marques de portage, frags sous
// equipement actif). Tout ce qui NOMME une piste passe par [IdentityRegistry.PontEpure] ; tout
// ce qui interroge un INSTANT passe par [IdentityRegistry.XUIDAt].
func (r IdentityRegistry) PontParSlot() map[uint32]uint64 { return r.own.SlotXUID }

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
	return r.own.xuidAt(slot, tUS)
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
	out := map[int]bool{}
	if len(r.deducedLives) == 0 {
		return out
	}
	for i := range tracks {
		from, to := trackSpanUS(tracks[i], origin, step)
		for li, l := range r.own.lives {
			if !r.deducedLives[li] || l.slot != tracks[i].Slot {
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

// buildOwners construit le pont a partir du seul fil des morts.
//
// PAS DE REPLI. Si le film ne porte pas son fil des morts, le pont est VIDE et aucun tir n'est
// publie — c'est le comportement voulu. Un rejeu muet se voit ; un rejeu qui pose des tirs sur
// le mauvais joueur ne se voit pas, et c'est bien pire.
//
// APPELANT UNIQUE : [BuildIdentityRegistry]. Le garde-rail `archlint` l'exige.
func buildOwners(tracks map[uint32]slotTrack, deaths []Death, idx PlayerIndexTable,
	fire []FireEventRef) OwnerReport {
	rep := OwnerReport{Owner: map[uint32]int{}, SlotXUID: map[uint32]uint64{}}
	if len(deaths) == 0 || len(tracks) == 0 || len(idx.ByXUID) == 0 {
		return rep
	}
	lives := buildLifeSpans(tracks)
	rep.LivesTotal = len(lives)
	off, matched, second := bestDeathOffset(lives, deaths)
	rep.DeathOffsetMS, rep.DeathOffsetMatches = off, matched
	rep.DeathOffsetRunnerUp = second
	rep.DeathsNamed = nameLivesByDeaths(lives, deaths, off)
	rep.lives = lives
	if rep.DeathsNamed == 0 {
		return rep
	}
	rep.IndexReadings = idx.Readings
	rep.IndexDisagreements = idx.Disagreements
	owners, byXUID, ambigus := ownersFromLives(lives, idx.ByXUID)
	rep.SlotAmbiguous = ambigus
	rep.SlotCollisions = len(ambigus)
	rep.FromDeaths = len(owners)
	// LES FERMETURES VIENNENT APRES LA LECTURE, JAMAIS A SA PLACE (cf. closures.go). Elles ne
	// touchent que les vies que le fil des morts n'a pas nommees, et elles s'abstiennent des que
	// deux candidats subsistent. `FromDeaths` est fige AVANT, pour que l'ecart entre lui et
	// `len(Owner)` reste lisible : c'est exactement ce que les fermetures ont ajoute.
	rep.Owner, rep.Closures = closeBridge(tracks, owners, lives, deaths, off, idx.ByXUID, fire)
	rep.SlotXUID = extendSlotXUID(byXUID, rep.Owner, idx.ByXUID)
	// LES FERMETURES NOMMENT AUSSI LA VIE (lot identite des vies, 2026-09-02) : le nommage des
	// tracks se fait desormais PAR VIE, et une vie fermee sans identite redeviendrait anonyme a
	// l'ecran alors que le pont la connait. C'est LA VIE QUE LA FERMETURE A DESIGNEE qui est
	// nommee (`closureReport.closedLife`), pas « l'unique vie anonyme du slot ».
	nameClosedLives(rep.lives, rep.Owner, rep.Closures.closedLife, idx.ByXUID)
	return rep
}

// nameClosedLives pose l'identite d'une fermeture sur LA VIE QU'ELLE A DESIGNEE.
//
// `closed` vient des fermetures elles-memes (slot -> indice de vie ; -1 = deux vies designees,
// donc abstention). Une vie deja nommee par le fil des morts n'est jamais reecrite : la lecture
// prime sur la deduction, comme partout dans ce pont.
func nameClosedLives(lives []lifeSpan, after, closed map[uint32]int, xuidToIndex map[uint64]int) {
	if len(closed) == 0 {
		return
	}
	indexToXUID := indexToXUIDOf(xuidToIndex)
	for slot, life := range closed {
		if life < 0 || life >= len(lives) || lives[life].xuid != 0 {
			continue
		}
		pi, known := after[slot]
		if !known {
			continue
		}
		if x, ok := indexToXUID[pi]; ok {
			lives[life].xuid = x
			// LA FERMETURE NOMME, ELLE NE TERMINE PAS. Elle dit « un autre corps est reapparu,
			// donc celui-ci etait celui-la » — rien sur la facon dont la vie s'est terminee.
			// `cause` reste donc ce que la decoupe a etabli (fin du film, ou coupure), et c'est
			// exactement ce qui empeche de refabriquer une mort pour un survivant.
			lives[life].nomPar = NomParFermeture
		}
	}
}

// extendSlotXUID pose l'identite sur les slots que les fermetures ont attribues. Sans cela, un
// slot deduit porterait des tirs sans que le client puisse nommer son joueur — les deux tables
// diraient deux choses differentes du meme pont, ce que `ownersFromLives` interdit deja.
func extendSlotXUID(byXUID map[uint32]uint64, owner map[uint32]int,
	xuidToIndex map[uint64]int) map[uint32]uint64 {
	indexToXUID := indexToXUIDOf(xuidToIndex)
	out := make(map[uint32]uint64, len(owner))
	for s, x := range byXUID {
		out[s] = x
	}
	for s, pi := range owner {
		if _, ok := out[s]; ok {
			continue
		}
		if x, ok := indexToXUID[pi]; ok {
			out[s] = x
		}
	}
	return out
}

// indexToXUIDOf renverse la table identite -> index. Un helper plutot que deux boucles
// identiques a vingt lignes d'ecart : la troisieme copie derive.
func indexToXUIDOf(xuidToIndex map[uint64]int) map[int]uint64 {
	out := make(map[int]uint64, len(xuidToIndex))
	for x, i := range xuidToIndex {
		out[i] = x
	}
	return out
}

// xuidAt rend le joueur qui OCCUPE ce slot à cet instant : la vie qui couvre l'instant si elle
// est nommée, sinon le pont par slot. Chaîne vide = ni l'une ni l'autre ne le nomme.
//
// POURQUOI L'INSTANT COMPTE (correctif du 2026-09-06, constat P1-7). `SlotXUID` est une identité
// UNIQUE PAR SLOT pour tout le match : `ownersFromLives` garde la PREMIÈRE vie nommée et jette
// les suivantes en collision, et `buildLifeSpans` trie par slot puis chronologiquement — c'est
// donc le PREMIER occupant, quel que soit l'instant demandé. Sur un slot de biped recyclé entre
// deux joueurs nommés (9 artefacts du parc portent `slotCollisions > 0`), tout lecteur qui
// interroge le pont sans son instant crédite le premier occupant.
//
// LE MOTIF EST CELUI DU DÉPÔT — « par vie d'abord, pont en repli » (cf. `tracksByXUID`) — et
// c'est ici qu'il vit pour tous ses lecteurs : la table par vie est déjà DANS cet objet.
func (r OwnerReport) xuidAt(slot uint32, tUS uint64) string {
	t := int64(tUS)
	for _, l := range r.lives {
		if l.slot != slot || l.xuid == 0 || t < l.from || t > l.to {
			continue
		}
		return strconv.FormatUint(l.xuid, 10)
	}
	// LE REPLI PAR SLOT S'ABSTIENT SUR UN SLOT AMBIGU (2026-09-07). `SlotXUID` y garde le
	// PREMIER occupant nommé, par ordre des vies : le servir à un instant que sa vie ne couvre
	// pas reviendrait à publier un nom arbitraire, et c'est exactement ce que cette méthode
	// existe pour éviter. Sans vie couvrante ET sur un slot à plusieurs occupants, on se tait.
	if r.SlotAmbiguous[slot] {
		return ""
	}
	if x, ok := r.SlotXUID[slot]; ok && x != 0 {
		return strconv.FormatUint(x, 10)
	}
	return ""
}

// NamingBridge rend le pont slot -> joueur DÉBARRASSÉ DES SLOTS AMBIGUS — celui que doit
// employer tout lecteur qui s'en sert pour NOMMER une piste.
//
// POURQUOI IL EXISTE (constat C2 de la revue VIES-R1, 2026-09-07). `SlotXUID` garde le PREMIER
// occupant nommé d'un slot que deux joueurs se partagent : c'est un choix par l'ORDRE DES VIES.
// `xuidAt` et `bridgeOfSlot` s'en abstiennent déjà, mais le helper partagé `xuidOfPublishedTrack`
// ne le pouvait pas — il ne reçoit qu'une map. Résultat mesuré sur `084a804d` slot 734 : la passe
// de nommage REFUSE (`contested = 1`) et le helper servait quand même `2535430265968559`, si bien
// que `samplesByXUID` indexait les positions de la piste contestée sous le premier occupant —
// une capture de zone pouvait être géolocalisée sur la trajectoire d'un AUTRE joueur.
//
// PLUTÔT QUE DE FAIRE DESCENDRE `SlotAmbiguous` DANS QUATRE CHAÎNES d'appel (les zones, les
// pistes de porteur de drapeau, les actions d'objectif, les morts neutres), on retire les slots
// ambigus À LA SOURCE : le lecteur ne peut plus oublier la garde, puisqu'il n'a plus de quoi
// l'enfreindre.
//
// `SlotXUID` RESTE INCHANGÉ pour ses autres consommateurs (ramassages, marques de portage, frags
// sous équipement actif) : leur exemption est explicite au cadrage de l'audit, et la modifier
// sortirait du périmètre de cette revue.
func (r OwnerReport) NamingBridge() map[uint32]uint64 {
	if len(r.SlotAmbiguous) == 0 {
		return r.SlotXUID
	}
	out := make(map[uint32]uint64, len(r.SlotXUID))
	for s, x := range r.SlotXUID {
		if !r.SlotAmbiguous[s] {
			out[s] = x
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
