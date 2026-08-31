// replay_contract_test.go — LE CONTRAT DU DOCUMENT DE REJEU, CHAMP PAR CHAMP.
//
// CE QUE CE FICHIER FERME. L artefact de rejeu publie des dizaines de champs (le compte du
// jour et son historique : wantReplayDocumentFields). Le contrat OpenAPI n en a
// longtemps decrit que 6, et les types TypeScript etaient ECRITS A LA MAIN hors du fichier
// genere : trois verites parallelles, dont deux se corrigeaient a la main. Le cout de cette
// divergence a ete paye — l interface manuelle nommait `weapon` le champ que le contrat nomme
// `w`, et pendant toute la vie du rejeu 2D les huit familles d effet de tir sont restees
// inatteignables sans que rien ne le signale.
//
// LES TROIS SONT ALIGNEES DEPUIS LE 2026-07-31 (le document est tire du contrat genere, et le
// contrat est genere depuis les types Go). CE QUI MANQUAIT ETAIT LE TEST : sans lui, rien
// n empeche la divergence de revenir, et elle reviendra silencieusement.
//
// CE QUE CE TEST AJOUTE AU GOLDEN OPENAPI EXISTANT. `openapi_golden_test.go` compare le contrat
// regenere au fichier commite, octet pour octet — il attrape TOUTE derive, mais il exige CGO et
// il ne dit pas CE QUI a change. Celui-ci est CGO-free, il ne regarde que la famille du rejeu, et
// il nomme le champ qui manque. Les deux se completent : l un est exhaustif, l autre est lisible.
//
// LE VOLET TypeScript est teste ailleurs, et il le doit : la frontiere de nullabilite vit dans
// `apps/web/src/features/match-replay/replayNormalize.ts`, et c est la que se verifie qu aucun
// tableau du contrat ne lui echappe (cf. replayContract.test.ts).
//
// LES CHAMPS QUE PERSONNE NE LIT — INVENTAIRE CONSIGNE, NON TRAITE (leur sort est le lot 3.6 du
// plan de finalisation ; ce jalon-ci VERROUILLE, il ne supprime pas). Re-mesure du 2026-07-31,
// par grep sur `features/match-replay/` et la route du rejeu, en excluant la frontiere et les
// tests — DOUZE champs publies dont aucun consommateur :
//
//	schemaVersion      le client ne branche sur aucune version ; il lit ce qu il trouve
//	structureBounds    le cadrage se fait sur `bounds`, jamais sur l etendue de la structure
//	MapObject.typeId   aucun style par famille d objet Forge n est rendu
//	Surface.zb         la face INFERIEURE ; seule la superieure sert de sol
//	RosterEntry.filmIndex  le client joint par xuid, jamais par index — c est la regle
//	BridgeHealth.livesNamed / livesTotal / indexReadings   l ecran affiche les verdicts, pas
//	                   les compteurs qui les fondent
//	Inventory.cand     le nombre de lectures possibles du bloc de munitions
//	Projectile.rest    le seul champ qui CERTIFIE une fin de vol, et il n est pas dessine
//	Point.hp           publie a 0,6 % de couverture, et volontairement pas mis en barre
//	Track.name         cas a part : il n est JAMAIS ECRIT par le Go (le film ne porte aucun
//	                   gamertag sur les traces). Il n a donc pas de lecteur parce qu il n a
//	                   rien a lire — c est le seul des douze dont la suppression est acquise.
//
// AUCUN DE CES CHAMPS N EST SUPPRIME ICI. Trois familles s y melangent — jamais ecrit,
// deliberement publie sans etre affiche (hp, rest), et simplement pas encore consomme — et les
// trancher demande un arbitrage produit, pas un test.

package contracttest_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"levelup/go-api/internal/analysis/replay"
)

// replaySchemas apparie chaque type Go publie dans l artefact au schema qui doit le decrire.
//
// LA LISTE EST ECRITE, PAS DERIVEE : un parcours automatique des types atteignables depuis
// ReplayDocument aurait le defaut de suivre le code — si un type sortait du document, il
// sortirait aussi du test, et personne ne le verrait.
var replaySchemas = []struct {
	schema string
	value  any
}{
	{"ReplayDocument", replay.ReplayDocument{}},
	{"Track", replay.Track{}},
	{"Point", replay.Point{}},
	{"Shot", replay.Shot{}},
	{"Grenade", replay.Grenade{}},
	{"Projectile", replay.Projectile{}},
	{"Loadout", replay.Loadout{}},
	{"Inventory", replay.Inventory{}},
	{"InventoryCoverage", replay.InventoryCoverage{}},
	{"AbilityRead", replay.AbilityRead{}},
	{"EquipmentEpisode", replay.EquipmentEpisode{}},
	{"EquipmentCoverage", replay.EquipmentCoverage{}},
	{"GrappleLine", replay.GrappleLine{}},
	{"GrappleCoverage", replay.GrappleCoverage{}},
	{"EquipmentPlacement", replay.EquipmentPlacement{}},
	{"EquipmentPlacementCoverage", replay.EquipmentPlacementCoverage{}},
	{"WeaponPad", replay.WeaponPad{}},
	{"PadPresence", replay.PadPresence{}},
	{"PadCycle", replay.PadCycle{}},
	{"PadPickup", replay.PadPickup{}},
	{"GroundWeaponCoverage", replay.GroundWeaponCoverage{}},
	{"AmmoSlot", replay.AmmoSlot{}},
	{"Surface", replay.Surface{}},
	{"MapObject", replay.MapObject{}},
	{"Bounds", replay.Bounds{}},
	{"RosterEntry", replay.RosterEntry{}},
	{"ObjectiveAction", replay.ObjectiveAction{}},
	{"ScoreTimeline", replay.ScoreTimeline{}},
	{"TeamScore", replay.TeamScore{}},
	{"PlayerScore", replay.PlayerScore{}},
	{"ScoreSeries", replay.ScoreSeries{}},
	{"ScoreRound", replay.ScoreRound{}},
	{"ScoreTick", replay.ScoreTick{}},
	{"ScoreCoverage", replay.ScoreCoverage{}},
	{"FlagCarry", replay.FlagCarry{}},
	{"FlagSpan", replay.FlagSpan{}},
	{"FlagCarriesCoverage", replay.FlagCarriesCoverage{}},
	{"VipPeriod", replay.VipPeriod{}},
	{"VipCrownCoverage", replay.VipCrownCoverage{}},
	{"SkullCarry", replay.SkullCarry{}},
	{"SkullCarriesCoverage", replay.SkullCarriesCoverage{}},
	{"ZoneState", replay.ZoneState{}},
	{"ZoneSpan", replay.ZoneSpan{}},
	{"GaugePoint", replay.GaugePoint{}},
	{"ZonesCoverage", replay.ZonesCoverage{}},
	{"Coverage", replay.Coverage{}},
	{"LayerCoverage", replay.LayerCoverage{}},
	{"BridgeHealth", replay.BridgeHealth{}},
}

// wantReplayDocumentFields : le nombre de champs que l artefact publie. Ecrit ici pour que le
// chiffre du chantier soit verifiable et pas seulement affirme.
//
// CHRONIQUE DU COMPTE (un champ n entre au document que par cette ligne) :
//
//	22 -> 23  2026-08-05  `objectives`, le calque d actions d objectif, entre a l integration
//	                      de `feat/re-mode-score`.
//	23 -> 25  2026-08-13/14  DEUX champs, un par lot de la v7.5 :
//	                      - `killEffects` (lot 2) : la table qui donne leur famille de RENDU
//	                        aux effets de MORT. Les kills du feed portent un weapon_key resolu
//	                        cote base, jamais un identifiant d arme film — sans cette table le
//	                        client ne peut joindre aucun effet a un kill.
//	                      - `mapObjectives` (lot 4) : le calque STATIQUE des objectifs du mode
//	                        joue (zones, apparitions et livraisons de drapeau, socles), rempli
//	                        A LA REQUETE par le service et jamais ecrit dans l artefact —
//	                        l artefact ne connait ni sa carte ni son mode.
//	25 -> 27  2026-08-14  DEUX champs, un par item du lot 7 :
//	                      - `originMs` (7.2) : l ORIGINE de la frame 0 sur l horloge du fil.
//	                        L horodatage de paquet du film est une horloge MOTEUR (des milliers
//	                        de secondes depuis le demarrage du jeu) : sans cette origine publiee,
//	                        le client n a aucun zero commun avec les events de la Match View, et
//	                        le fil arrivait 3,6 s a 40 s apres le flash des fiches.
//	                      - `neutralDeaths` (7.1) : les morts que personne ne revendique
//	                        (suicide, environnement...), avec la NATURE du degat fatal — c est
//	                        elle qui choisit l icone du type de mort au fil.
//
//	27 -> 28  2026-08-14  `abilities` (plan PLAN_RANG_CAPACITE_I48, etape 1.2) : le RANG de
//	                      palette de la capacite d armure portee. UN CHAMP ENTRE, mais un
//	                      autre CHANGE DE SENS en meme temps et c est le vrai evenement :
//	                      `Inventory.a` portait `rang - 16` (l ancre du canal d image-cle se
//	                      termine par `010`, les bits de poids fort du rang, si bien que ce
//	                      canal ne voit que 16..23). Le champ `a` est RETIRE plutot que
//	                      reinterprete — republier une autre grandeur sous la meme cle aurait
//	                      laisse tout client non mis a jour lire un nombre qui ne veut plus dire
//	                      la meme chose. `abilityLabels` change de cle avec (rang, plus index).
//
//	28 -> 29  2026-08-16  `equipmentEpisodes` (plan PLAN_EQUIPEMENT_TI37, phase 1) : l etat
//	                      ACTIF du camouflage et du surbouclier, en episodes dates par vie
//	                      — les DEUX familles dont l etat est MESURE (i28 queue[1] binaire
//	                      0/4095 exclusif aux vies rang 8 ; i5 non clampe, regle q > 64, 0
//	                      faux positif sur ~150 000 mesures hors porteurs). Les autres
//	                      familles restent SANS etat : les deployables ne se datent pas par
//	                      les canaux mesures, la mobilite n a pas d instant d usage par i54.
//	                      `Coverage` gagne en meme temps son bloc `equipment` (vies
//	                      porteuses / vies publiees, par famille) : N episodes sans
//	                      denominateur se lirait comme une exhaustivite.
//
//	29 -> 30  2026-08-16  `grappleLines` (plan PLAN_GRAPPIN_LIGNE, phase 1) : les TRACTIONS
//	                      de grappin — fenetre datee par vie [t0, t1], du tir a l ARRIVEE
//	                      mesuree sur la trajectoire, et point d accroche en coordonnees
//	                      monde. Source : le corps tag==3 d i59, porte sur grammaire MESUREE
//	                      et prouve au gate 0 du plan (marche a l ecart zero sur 3 films,
//	                      ancre fixe a 0,05-0,07 u pres, distance joueur->ancre decroissante
//	                      contre temoins melanges effondres). `Coverage` gagne son bloc
//	                      `grapple` (tirs, accroches, tractions, rates, corps non
//	                      decodables) : N tractions sans ses rejets se lirait comme une
//	                      exhaustivite.
//
//	30 -> 31  2026-08-18  `equipmentPlacements` (plan PLAN_POSES_EQUIPEMENT_PUBLICATION,
//	                      phase 2) : les POSES d objets d equipement — position monde,
//	                      fenetre [t0, t1] de la creation a la disparition, famille
//	                      (`wall` / `sensor` / `other`), identifiant `eqip` du jeu, poseur
//	                      MESURE (bipede le plus proche a 250 ms et moins de 3 m ; mediane
//	                      0,52-0,60 m sur 11 films contre 11-36 m pour le temoin) et cap de
//	                      visee du poseur quand il a ete lu. Source : le mot de 32 bits du
//	                      bloc `object-multiplayer-properties` du record de CREATION de
//	                      l archetype 37 — 21 valeurs sur 21 resolues dans le groupe `eqip`
//	                      du jeu. `Coverage` gagne son bloc `placements`, qui porte le
//	                      DECOUPAGE de bloc calibre sur le film : sans lui, zero pose par
//	                      absence d equipement et zero pose par calibration refusee seraient
//	                      indistinguables.
//
//	31 -> 33  2026-08-17  DEUX champs, un seul calque (plan PLAN_ARMES_AU_SOL_2E_LECTURE,
//	                      phase 3) : les SOCLES D ARME du match.
//	                      - `weaponPads` : position monde du socle, famille d arme (meme
//	                        ecriture que `Loadout.W`, donc meme cle dans `weaponLabels`),
//	                        instants d apparition, intervalles de presence BORNES par le
//	                        recensement des images-cles, et cycle de reapparition SEULEMENT s il
//	                        est etabli. Source : le record de CREATION de l archetype 42, dont
//	                        le mot MPP de 32 bits est l identite de l arme — 282 atterrissages
//	                        exacts sur 289 pour l oracle de position, 937 accords sur 947 pour
//	                        l identite croisee avec les images-cles.
//	                      - `padPickups` : les occupations qui se sont ACHEVEES, publiees comme
//	                        un INTERVALLE et non un instant (le film ne porte aucun evenement de
//	                        ramassage, le recensement est espace de ~20 s). Le champ `xuid`
//	                        existe et vaut `null` PARTOUT : l oracle des loadouts donne 88,1 %
//	                        par slot de vie et 79,7 % par joueur, contre >= 90 % exige.
//	                      `Coverage` gagne son bloc `groundWeapons` : un film sans socle, un film
//	                      dont toutes les armes sont des lachers et un film qu on n a pas su
//	                      balayer rendent tous trois zero socle, et seuls ces compteurs les
//	                      distinguent.
//
//	33 -> 34  2026-08-18  `scoreTimeline` (plan PLAN_EXPLOITATION_REGISTRE_FILM, lot A phase 1) :
//	                      LE SCORE DANS LE TEMPS. Un seul champ de document, mais SEPT schemas
//	                      de plus, parce que la courbe a deux formes et deux porteurs :
//	                      `teams[]` (le camp, sa courbe par MANCHE et son total cumule) et
//	                      `players[]` (score personnel, frags, morts, assistances, chacun sous
//	                      les deux memes formes). Les manches ne sont pas un detail : le score
//	                      de mode repart de zero a chaque manche, et lire la derniere valeur
//	                      brute donnait 100/78 la ou l oracle affiche dit 200/121.
//	                      Source : les enregistrements d entite des paquets FRAME, dont la
//	                      grammaire a ete calibree sur 1 078 en-tetes et 2 708 lectures de
//	                      composant issus d une capture Cheat Engine. Oracle : le score
//	                      AFFICHE (16/16 exact sur les films ou l API le porte, 5 modes sur 5) —
//	                      et il est NOMME dans le document (`coverage.score.oracle`), parce que
//	                      l API compte autre chose en Strongholds (des ticks) et en KOTH (des
//	                      secondes de colline).
//	                      `Coverage` gagne son bloc `score` : identite des camps (`a` par le
//	                      score final, `b` par la somme des frags, `unresolved`), manches lues,
//	                      mode porte, lecture tronquee, points publies. Une courbe sans ces
//	                      compteurs se lirait comme une certitude — et l ABSENCE du bloc dit
//	                      encore autre chose : l appelant n a rien fourni a lire.
//
//	34 -> 34  2026-08-18  RIEN, ET C EST ECRIT (plan PLAN_EXPLOITATION_REGISTRE_FILM, lot E
//	                      phase 1). Le schema 13 publie `Point.p` — l ELEVATION DE VISEE — mais
//	                      `Point` n est pas un champ RACINE du document : il vit sous
//	                      `tracks[].points[]`. Le compte ci-dessous ne compte que la racine, il
//	                      ne bouge donc pas. La ligne existe quand meme parce que l absence de
//	                      ligne pour une montee de schema se lit comme un OUBLI ; le champ, lui,
//	                      est bel et bien verrouille — par
//	                      TestReplayContractDescribesEveryPublishedField, qui compare le type Go
//	                      `Point` au schema `Point` du contrat, dans les DEUX sens.
//
//	34 -> 35  2026-08-18  `flagCarries` (plan PLAN_OBJECTIFS_VIVANTS_2E_LECTURE, phase 1
//	                      item 1.3) : LA VIE DE CHAQUE DRAPEAU de CTF, en intervalles d etat sur
//	                      l axe de frames du rejeu. Un champ de document, deux schemas de plus
//	                      (`FlagCarry`, `FlagSpan`) et un bloc de couverture
//	                      (`FlagCarriesCoverage`).
//	                      Sources, toutes dans le film : les bornes viennent des evenements de
//	                      statistique NOMMES du statborg (`flag_grabs`, `flag_steals`,
//	                      `flag_captures`, `flag_returns`) et du fil des morts ; le porteur du
//	                      pont par INSTANTS DE MORT (aucune ligne de match, donc aucune base) ;
//	                      la position de la piste PUBLIEE du porteur — le drapeau porte EST a la
//	                      position de son porteur, rien de l objet n est decode ; le mode de
//	                      l accord de trois signaux du film (15 films de mode connu, 15 verdicts
//	                      justes). Seuls les SOCLES viennent d ailleurs : du catalogue versionne
//	                      d objectifs, joint par `map_id`.
//	                      QUATRE ETATS, ET LE QUATRIEME EST LE RESULTAT : `carried` (un fait date
//	                      a ferme le portage), `carried_open` (rien ne le ferme — borne haute a
//	                      la fin de l axe, incertitude publiee comme telle), `dropped`, `home`.
//	                      Le controle independant du marqueur d image-cle confirme 37/37 des
//	                      portages FERMES et 0/5 des ouverts : les confondre ferait juger la
//	                      justesse des bornes par des portages qui n en ont pas.
//	                      `Coverage` gagne son bloc `flagCarries` : un film d un autre mode et un
//	                      film CTF sans aucun portage publie rendent tous deux un calque vide, et
//	                      seuls ces compteurs les distinguent.
//
//	35 -> 36  2026-08-18  `zoneStates` (plan PLAN_EXPLOITATION_REGISTRE_FILM, lot C-bis phase 2b) :
//	                      L ETAT DE CHAQUE ZONE du mode, en intervalles de propriete sur l axe de
//	                      frames du rejeu. Un champ de document, deux schemas de plus
//	                      (`ZoneState`, `ZoneSpan`) et un bloc de couverture (`ZonesCoverage`).
//	                      Source, toute dans le film : l archetype `ti=13`
//	                      (`managed-object-property-*`), dont UN SLOT EST UNE PROPRIETE RESEAU
//	                      NOMMEE et non une zone — la JAUGE de capture (tag 3) et le PROPRIETAIRE
//	                      (tag 4) vivent sur des slots DISJOINTS. L appariement slot -> zone se
//	                      refait A CHAQUE MATCH, par la coincidence d un sommet de jauge avec une
//	                      capture nommee attribuee geometriquement : coherence 93,1 % et 98,4 %
//	                      (seuil 90 %) contre des temoins a 41-48 % (permutation des slots) et
//	                      51-57 % (sommet decale de 20 s). La VALEUR du tag 4 est l index d equipe
//	                      du capteur a 100,0 % (48/48) et 91,1 % (51/56) hors emissions neutres.
//	                      L EQUIPE vient du ROSTER, jamais du film : `game-engine-team-mapping`
//	                      lit ses bits sans les publier.
//	                      `zoneRef` est un INDEX dans `mapObjectives.zones` — le calque statique
//	                      servi a la requete —, et `coverage.zones.roles` publie les roles qui
//	                      composent cette liste pour que la jointure se VERIFIE au lieu d etre
//	                      supposee. `Coverage` gagne son bloc `zones` : un film d un autre mode,
//	                      un film a zones dont l appariement echoue et une carte hors du
//	                      catalogue rendent tous trois un calque vide, et seuls ces compteurs les
//	                      distinguent.
//
//	36 -> 36  2026-08-18  RIEN A LA RACINE, ET C EST ECRIT (plan PLAN_EXPLOITATION_REGISTRE_FILM,
//	                      lot C-ter volet 3). Le schema 18 publie `zoneStates[].gauge` — LA JAUGE
//	                      DE CAPTURE EN DIRECT (serie datee `[{t, v}]`, allegee : un point par
//	                      variation >= 0,02 ou par seconde de rampe, rien hors rampe, chaque rampe
//	                      fermee par son retour a zero, sur l echelle de `progress`, modes a zones
//	                      SIMULTANEES seulement — jamais sur une colline de KOTH, volet 1) et
//	                      `coverage.zones.gaugePoints` — mais ni l un ni l autre
//	                      n est un champ RACINE du document : le premier vit sous `zoneStates[]`,
//	                      le second sous `coverage.zones`. Le compte ci-dessous ne compte que la
//	                      racine, il ne bouge donc pas. La ligne existe pour la meme raison que
//	                      celle du schema 13 : une montee de schema sans ligne se lit comme un
//	                      OUBLI. Les champs, eux, sont verrouilles — `GaugePoint` entre dans
//	                      replaySchemas, et TestReplayContractDescribesEveryPublishedField compare
//	                      `ZoneState`, `ZonesCoverage` et `GaugePoint` a leur schema dans les DEUX
//	                      sens. `progress` reste tel quel : le sommet par intervalle est CONSERVE
//	                      dans le contrat, c est le client qui cesse de le dessiner.
//	                      RENUMEROTE LE 2026-08-19 : la jauge visait le 17, une autre session l a
//	                      pris en fusionnant avant nous (socles de power-up dans `weaponPads`,
//	                      eux aussi SANS champ racine neuf). Le compte reste donc 36 des DEUX
//	                      cotes de la fusion — aucun des deux lots n ajoute de champ a la racine.
//	                      PRECISION 2026-08-20 (re-fusion d origin) : le compte racine passe
//	                      bien a 37, mais par l entree SUIVANTE (`mapWeaponPads`, un champ servi
//	                      a la requete) — la jauge, elle, n y est toujours pour rien.
//
//	36 -> 37  2026-08-19  `mapWeaponPads` (plan PLAN_SOCLES_MVAR, section 8 ter) : LES
//	                      EMPLACEMENTS DE SOCLE de la carte, CROISES avec les socles du match.
//	                      Un champ de document et deux schemas de plus (`MapWeaponPads`,
//	                      `MapWeaponPadDTO`), rempli A LA REQUETE par le service comme
//	                      `mapObjectives` et jamais ecrit dans l artefact — d ou un SchemaVersion
//	                      d artefact INCHANGE a 17 : rien n a bouge dans l artefact.
//	                      Source : les trois type_id de socle du fichier de carte
//	                      (0x5F379533 pouvoir, 0x6253CFC0 ratelier, 0x5E86D110 power-up), figes
//	                      hors ligne en catalogue versionne — 72 cartes, 1 454 emplacements,
//	                      32 positions d oracle appariees sur 32 a une mediane de 0,01 m.
//	                      LE CHAMP NE PORTE QUE LES EMPLACEMENTS ALLUMES : le fichier de carte
//	                      POSE les socles, le mode les ALLUME, et Cliffhanger en porte dix-huit
//	                      dont dix servis en CTF et ZERO en Super Fiesta. Un emplacement qu aucun
//	                      socle du match ne confirme a moins d un metre reste au serveur
//	                      (decision utilisateur du 2026-08-19) ; `catalogN` dit combien la carte
//	                      en porte, pour que le calque avoue ce qu il n affiche pas.
//
//	38 -> 39  2026-08-27  `objectiveObjects` (plan PLAN_OBJECTIFS_ETAT_VIVANT, phase D5-bis) :
//	                      OU SE TROUVE L OBJET D OBJECTIF QUAND PERSONNE NE LE PORTE — les vies
//	                      LIBRES du crane d Oddball. Le canal est une PRESENCE, pas une
//	                      deduction : un objet du monde replique sa position tant qu il est
//	                      libre et CESSE de la repliquer des qu on le porte. Le champ publie
//	                      donc exactement les positions que le film a emises.
//	                      L IDENTITE DU CRANE EST MESUREE (phase D4) : mot MPP `0x0017592C`,
//	                      elu sur 4 films sur 4, ne a 0,0 m du socle `oddball_spawn` unique de
//	                      sa carte et coincidant a 3-6 ms d un evenement `th=10`. Le compteur
//	                      `coverage.groundWeapons.objectives` le corrobore a l unite pres :
//	                      23 / 16 / 21 / 47 creations, exactement les comptes de D4.
//	                      CE QUE LE CHAMP NE DIT PAS, ET C EST ECRIT DANS SON SCHEMA : QUI porte
//	                      l objet pendant les trous. L oracle du porteur a ete mesure et REFUSE
//	                      par son propre protocole (40,6 a 66,7 % de trous a porteur unique
//	                      contre un seuil de 90 %, temoin hors trou a 66,7 et 71,4 %).
//	                      LE DRAPEAU N Y EST PAS, et ce n est pas un report : le controle 3 de
//	                      son propre lot a ECHOUE sur ses vies libres (149/197 = 75,6 % pour un
//	                      seuil de 90 %). La forme publiee porte `family` pour qu il puisse la
//	                      rejoindre sans qu aucune cle ne bouge.
//	                      SCHEMA D ARTEFACT INCHANGE A 21 : le bump du lot a eu lieu dans le
//	                      meme lot et n a quitte ni le poste ni les temoins locaux — aucun
//	                      artefact 21 n existe ailleurs, le bump unique reste unique.
//
//	39 -> 40  2026-08-27  `vipCrown` (lot VIP COURONNE, `.ai/V7.5/replay2d/registre_film/
//	                      VIP_COURONNE_PROTOCOLE.md`) : LES PERIODES DE PORT DE LA COURONNE VIP,
//	                      en intervalles de frames nommes par le xuid du VIP. Un champ de
//	                      document, un schema de plus (`VipPeriod`) et un bloc de couverture
//	                      (`VipCrownCoverage`). Source, toute dans le film : les SELECTIONS
//	                      `vip_selected` (`comp 22 A` = `TimesSelectedAsVip`, resolu au gate
//	                      corrige — 100 % par joueur x3 films, temoin decale 0) et le fil des
//	                      morts ; le VIP nomme par le pont d INSTANTS DE MORT (aucune base). La
//	                      reconstruction a ete MESUREE : les periodes somment, par joueur, a
//	                      `TimeAsVip` de l API au SUB-SECONDE (recouv 100 % 3/3, 24/24 joueurs a
//	                      +0,2-0,3 s), contre un temoin d attribution aleatoire effondre
//	                      (exactitude 8/8 contre 0-1/8). GARDE DE MODE chez l appelant : `comp
//	                      22 A` vaut `flag_grabs` en CTF, donc la couronne n est lue que sur un
//	                      film reconnu VIP par `game_variant_name` — jamais devinee dans le film.
//	                      Le SchemaVersion d artefact monte a 22 (reprise du backfill).
//	                      `Coverage` gagne son bloc `vipCrown` : un film non-VIP (bloc absent) et
//	                      un film VIP sans periode publiee se distinguent par lui.
//
//	40 -> 41  2026-08-28  `skullCarries` (lot PORTEUR ODDBALL, `.ai/V7.5/replay2d/registre_film/
//	                      ODDBALL_PORTEUR_PROTOCOLE.md`) : LES PERIODES DE PORTAGE DU CRANE, en
//	                      intervalles de frames nommes par le xuid du porteur. Un champ de
//	                      document, un schema de plus (`SkullCarry`) et un bloc de couverture
//	                      (`SkullCarriesCoverage`). Source, toute dans le film : le porteur est le
//	                      joueur dont les TICS DE SCORE DE MODE montent (`comp 0 A` =
//	                      `skull_scoring_ticks`), un TRAIN de tics ETANT une periode de portage ;
//	                      le porteur nomme par le pont d INSTANTS DE MORT PAR MANCHE (le slot est
//	                      reattribue d une manche a l autre, aucune base). Le portage avait resiste
//	                      a CINQ campagnes (proximite, traversee, score personnel : negatifs) ; le
//	                      canal des tics tient — gate oracle porteur PRINCIPAL correct 7/7 films,
//	                      gate terrain manche 1 de d9781168 prises 9/9 et porteurs 8/9 (seuil 8/9),
//	                      emplacement identifie par l oracle films confondus. GARDE DE MODE chez l
//	                      appelant : `comp 0 A` est le score de mode de tout mode, donc le porteur
//	                      n est lu que sur un film reconnu Oddball par `game_variant_name`. Le
//	                      SchemaVersion d artefact monte a 23 (reprise du backfill). `Coverage`
//	                      gagne son bloc `skullCarries`. Le crane LIBRE (`objectiveObjects`, v21)
//	                      reste la couche POSITION ; celle-ci est la couche PORTEUR par-dessus.
//
//	41 -> 44  2026-08-30  TROIS champs, le chantier ramassage (schemas 25-28) :
//	                      - `weaponChanges` (v25) : les prises et lachers d arme, dates a la
//	                        milliseconde, re-annonces ecartees ;
//	                      - `equipmentChanges` (v26) : ramassages et consommations d equipement
//	                        (i48), avec temoin de completude au compteur de rotation ;
//	                      - `groundWeapons` (v27) : les armes au sol individuelles, bornees par
//	                        l observation (pickup date / census) — la minuterie `until` de v25
//	                        est retiree en meme temps. La v28 (fins des poses) n ajoute AUCUN
//	                        champ au document : `until`/`untilMax`/`end` vivent sur
//	                        EquipmentPlacement, pas a la racine.
//
// Les quatorze fois, ce test a ATTRAPE l ecart : une branche publiait le champ avant que le
// chiffre ne le dise. Contrat regenere (`make openapi-gen`), jamais ecrit a la main.
const wantReplayDocumentFields = 44

// TestReplayContractDescribesEveryPublishedField : AUCUN CHAMP PUBLIE SANS DESCRIPTION, ET
// AUCUNE DESCRIPTION SANS CHAMP.
//
// Les deux sens comptent, et pas pour la meme raison. Un champ publie que le contrat ignore est
// une donnee que le client ne peut pas typer — c est ce qui a produit l interface manuelle. Une
// propriete decrite que le Go ne publie plus est une promesse morte : un client la lit comme
// optionnelle et ne saura jamais qu elle ne viendra plus.
func TestReplayContractDescribesEveryPublishedField(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range replaySchemas {
		t.Run(c.schema, func(t *testing.T) {
			sch, ok := schemas[c.schema]
			if !ok {
				t.Fatalf("le contrat ne decrit AUCUN schema %q, alors que l artefact publie le "+
					"type Go correspondant", c.schema)
			}
			got := jsonFieldsOf(reflect.TypeOf(c.value))
			want := propertyNamesOf(sch)
			for _, f := range got {
				if !contains(want, f) {
					t.Errorf("champ %q publie par le Go et ABSENT du contrat — un client ne peut "+
						"pas le typer, et c est exactement ainsi qu une interface ecrite a la "+
						"main reapparait", f)
				}
			}
			for _, p := range want {
				if !contains(got, p) {
					t.Errorf("propriete %q decrite par le contrat et PLUS publiee par le Go — une "+
						"promesse morte que le client lira comme optionnelle", p)
				}
			}
		})
	}
}

// TestReplayDocumentFieldCountIsFrozen : le chiffre du chantier, verifie des deux cotes.
//
// LE NOM NE PORTE PLUS LE CHIFFRE (il a dit « TwentyTwo » jusqu au 2026-08-14 alors que le
// document en publiait 25) : un compte qui bouge se lit dans wantReplayDocumentFields et sa
// chronique, pas dans un identifiant que personne ne pense a renommer.
func TestReplayDocumentFieldCountIsFrozen(t *testing.T) {
	got := jsonFieldsOf(reflect.TypeOf(replay.ReplayDocument{}))
	if len(got) != wantReplayDocumentFields {
		t.Errorf("%d champ(s) publie(s) par l artefact, attendu %d : %v",
			len(got), wantReplayDocumentFields, got)
	}
	props := propertyNamesOf(loadReplaySchemas(t)["ReplayDocument"])
	if len(props) != wantReplayDocumentFields {
		t.Errorf("%d propriete(s) decrite(s) par le contrat, attendu %d", len(props),
			wantReplayDocumentFields)
	}
}

// TestReplayContractCarriesTupleArity : L ARITE D UN TUPLE SE DIT, ELLE NE SE DEVINE PAS.
//
// Le Go ecrit `Poly [][2]float32` et `P [][3]float32` : des sommets XY et des pas [dt, x, y].
// JSON Schema n a pas de type « tuple », mais il a `minItems` = `maxItems`, et c est exactement
// ce que cela veut dire. Sans cette paire, le contrat ne dit plus que « un tableau de nombres »,
// et la frontiere web qui retablit l arite le ferait sur une supposition au lieu d une lecture.
func TestReplayContractCarriesTupleArity(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range []struct {
		schema, prop string
		goType       reflect.Type
		goField      string
	}{
		{"Surface", "poly", reflect.TypeOf(replay.Surface{}), "Poly"},
		{"Projectile", "p", reflect.TypeOf(replay.Projectile{}), "P"},
	} {
		t.Run(c.schema+"."+c.prop, func(t *testing.T) {
			f, ok := c.goType.FieldByName(c.goField)
			if !ok {
				t.Fatalf("champ Go %s.%s introuvable", c.goType.Name(), c.goField)
			}
			// [][N]float32 : la longueur du tuple est celle du tableau interieur.
			wantLen := f.Type.Elem().Len()
			tuple := tupleSchemaOf(t, schemas[c.schema], c.prop)
			mn, mx := intAt(tuple, "minItems"), intAt(tuple, "maxItems")
			if mn != wantLen || mx != wantLen {
				t.Errorf("arite decrite minItems=%d maxItems=%d, attendu %d des deux cotes — le Go "+
					"ecrit un [%d]float32, et c est la SEULE facon de le dire en JSON Schema",
					mn, mx, wantLen, wantLen)
			}
		})
	}
}

// TestReplayContractDoesNotRequireOptionalFields : `omitempty` et `required` ne se contredisent pas.
//
// Un champ omis a la serialisation ne peut pas etre requis a la lecture. La contradiction ne
// casse rien au premier abord — elle casse le jour ou le champ vaut zero.
func TestReplayContractDoesNotRequireOptionalFields(t *testing.T) {
	schemas := loadReplaySchemas(t)
	for _, c := range replaySchemas {
		omit := omitemptyFieldsOf(reflect.TypeOf(c.value))
		for _, r := range requiredOf(schemas[c.schema]) {
			if contains(omit, r) {
				t.Errorf("%s : le contrat exige %q alors que le Go l omet quand il est vide — "+
					"la contradiction se revele le jour ou le champ vaut zero", c.schema, r)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Lecture du contrat et reflexion
// ---------------------------------------------------------------------------

// loadReplaySchemas lit components.schemas de api/openapi.yaml.
func loadReplaySchemas(t *testing.T) map[string]map[string]any {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(thisFile), "..", "api", "openapi.yaml")
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fige, relatif au fichier de test
	if err != nil {
		t.Fatalf("lecture du contrat %s : %v", path, err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]map[string]any `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("contrat illisible : %v", err)
	}
	if len(doc.Components.Schemas) == 0 {
		t.Fatal("le contrat ne porte aucun schema : la lecture est fausse")
	}
	return doc.Components.Schemas
}

// jsonFieldsOf rend les noms JSON des champs serialises d une struct.
func jsonFieldsOf(rt reflect.Type) []string {
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		if name, ok := jsonNameOf(rt.Field(i)); ok {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// omitemptyFieldsOf rend les champs que le Go OMET quand ils sont vides.
func omitemptyFieldsOf(rt reflect.Type) []string {
	var out []string
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		name, ok := jsonNameOf(f)
		if !ok || !strings.Contains(f.Tag.Get("json"), ",omitempty") {
			continue
		}
		out = append(out, name)
	}
	return out
}

// jsonNameOf rend le nom JSON d un champ, et false s il n est pas serialise.
func jsonNameOf(f reflect.StructField) (string, bool) {
	if f.PkgPath != "" { // champ non exporte
		return "", false
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		name = f.Name
	}
	return name, true
}

func propertyNamesOf(schema map[string]any) []string {
	props, _ := schema["properties"].(map[string]any)
	out := make([]string, 0, len(props))
	for k := range props {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func requiredOf(schema map[string]any) []string {
	raw, _ := schema["required"].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// tupleSchemaOf descend `properties.<prop>.items` — l ELEMENT de la liste, c est-a-dire le
// tuple lui-meme. C est lui qui porte l arite ; le niveau en dessous ne decrit qu un nombre.
func tupleSchemaOf(t *testing.T, schema map[string]any, prop string) map[string]any {
	t.Helper()
	props, _ := schema["properties"].(map[string]any)
	p, _ := props[prop].(map[string]any)
	tuple, ok := p["items"].(map[string]any)
	if !ok {
		t.Fatalf("propriete %q : aucun schema d element — l arite ne peut plus s exprimer", prop)
	}
	if _, hasInner := tuple["items"]; !hasInner {
		t.Fatalf("propriete %q : l element n est plus un tableau — ce n est plus un tuple", prop)
	}
	return tuple
}

func intAt(m map[string]any, key string) int {
	switch v := m[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		return -1
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
