package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempStructure(t *testing.T, ms MapStructure) string {
	t.Helper()
	blob, err := json.Marshal(ms)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	p := filepath.Join(t.TempDir(), "ridgeline.json")
	if err := os.WriteFile(p, blob, 0o644); err != nil {
		t.Fatalf("écriture: %v", err)
	}
	return p
}

func TestLoadMapStructure(t *testing.T) {
	p := writeTempStructure(t, MapStructure{
		SchemaVersion: MapStructureSchemaVersion,
		Module:        "ridgeline",
		Surfaces:      []Surface{{X0: -1, Y0: -2, X1: 3, Y1: 4, Z: 1.5, ZB: 0}},
	})
	ms, err := LoadMapStructure(p)
	if err != nil {
		t.Fatalf("LoadMapStructure: %v", err)
	}
	if ms.Module != "ridgeline" || len(ms.Surfaces) != 1 {
		t.Fatalf("document inattendu: %+v", ms)
	}
	if got := ms.Surfaces[0].Area(); got != 24 {
		t.Fatalf("aire = %v, attendu 24", got)
	}
}

// Une face inférieure à exactement 0 doit SURVIVRE à l'aller-retour JSON : un omitempty sur
// ZB la relirait comme 0 par défaut — invisible ici, mais il déplacerait d'un étage toute
// surface dont la face haute serait, elle, omise. Le test fige l'absence d'omitempty.
func TestSurfaceZeroAltitudeSurvivesJSON(t *testing.T) {
	blob, err := json.Marshal(Surface{X0: 1, Y0: 1, X1: 2, Y1: 2, Z: 0, ZB: 0})
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"z", "zb"} {
		if _, ok := back[k]; !ok {
			t.Fatalf("champ %q omis du JSON (%s) — une altitude nulle serait perdue", k, blob)
		}
	}
}

func TestLoadMapStructureRejectsOtherSchemaVersion(t *testing.T) {
	p := writeTempStructure(t, MapStructure{SchemaVersion: MapStructureSchemaVersion + 1})
	if _, err := LoadMapStructure(p); err == nil {
		t.Fatal("une version de schéma inconnue doit être refusée")
	}
}

func TestSurfaceBounds(t *testing.T) {
	if surfaceBounds(nil) != nil {
		t.Fatal("aucune surface -> pas de bornes")
	}
	b := surfaceBounds([]Surface{
		{X0: 0, Y0: 0, X1: 2, Y1: 2},
		{X0: -5, Y0: 1, X1: -1, Y1: 9},
	})
	if b == nil || b.MinX != -5 || b.MinY != 0 || b.MaxX != 2 || b.MaxY != 9 {
		t.Fatalf("bornes inattendues: %+v", b)
	}
}

// L'ajout de la structure ne doit PAS incrémenter SchemaVersion : les champs sont
// optionnels (omitempty), un client existant qui les ignore reste correct.
func TestStructureIsOptionalInDocument(t *testing.T) {
	doc := ReplayDocument{SchemaVersion: SchemaVersion}
	blob, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("sérialisation: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(blob, &back); err != nil {
		t.Fatalf("désérialisation: %v", err)
	}
	for _, k := range []string{"structure", "structureBounds"} {
		if _, ok := back[k]; ok {
			t.Fatalf("champ %q présent sur un document sans structure — omitempty manquant", k)
		}
	}
	// LA RÈGLE TENUE ICI : un champ OPTIONNEL de plus n'incrémente pas la version — sans
	// quoi chaque enrichissement casserait les clients. La version ne bouge que sur un
	// changement de FORME, et chaque incrément doit avoir sa raison écrite :
	//   v1 -> v2 (2026-08-02, lot 3.1/3.2) : les tables de libellés passent de `string`
	//   à `{en, fr}` et le type d'un lancer de grenade devient son RANG. Deux formes
	//   changées, donc une version — la structure, elle, n'y est pour rien.
	//   v2 -> v3 (2026-08-13, plan parité lot 2) : Grenade.proj (lien lancer -> projectile)
	//   et killEffects. Champs omitempty, mais la version monte parce qu'elle est la CLÉ DE
	//   REPRISE du backfill (lot 6) : les effets de repos de grenade n'existent que sur un
	//   artefact qui porte le lien, un v2 doit se lire « à re-cuire », pas « à jour ».
	//   v3 -> v4 (2026-08-14, plan parité lot 7.2) : originMs, l'instant de la frame 0 sur
	//   l'horloge du fil des éliminations. Champ omitempty, même raison de monter : c'est la
	//   CLÉ DE REPRISE du backfill, et sans lui le client ne peut caler le fil que par
	//   appariement statistique — un v3 doit se lire « à re-cuire », pas « à jour ».
	//   v4 -> v5 (2026-08-14, plan parité lot 7.1) : neutralDeaths, le TYPE des morts que
	//   personne ne revendique (chute / hors-limites, ou sa propre source de dégât). Champ
	//   omitempty, même raison de monter : c'est la CLÉ DE REPRISE du backfill, et sans lui
	//   le fil ne peut poser sur ces lignes qu'un repère générique — un v4 doit se lire
	//   « à re-cuire », pas « à jour ».
	//   v5 -> v6 (2026-08-14, plan PLAN_RANG_CAPACITE_I48 étape 1.2) : la capacité d'armure
	//   CHANGE DE GRANDEUR. `Inventory.a` portait `rang − 16` — l'ancre du canal d'image-clé
	//   se termine par `010`, les bits de poids fort du rang, et ce canal ne voit donc que la
	//   fenêtre 16..23. Le document publie désormais `abilities`, le RANG complet, alimenté
	//   par deux canaux. Le champ `a` est RETIRÉ, pas réinterprété : republier une autre
	//   grandeur sous la même clé aurait laissé tout client non mis à jour lire un nombre qui
	//   ne veut plus dire la même chose. Ce n'est donc PAS un champ optionnel de plus — c'est
	//   un retrait plus un changement de sens, et `abilityLabels` change de clé avec.
	//   v6 -> v7 (2026-08-16, plan PLAN_EQUIPEMENT_TI37 phase 1) : equipmentEpisodes, l'état
	//   ACTIF du camouflage et du surbouclier en épisodes datés par vie. Champ omitempty,
	//   même raison de monter que v3/v4/v5 : c'est la CLÉ DE REPRISE du backfill — l'effet
	//   plein-fiche et les sons d'équipement n'existent que sur un artefact qui porte les
	//   épisodes, un v6 doit se lire « à re-cuire », pas « à jour ».
	//   v7 -> v8 (2026-08-16, plan PLAN_GRAPPIN_LIGNE phase 1) : grappleLines, les tractions
	//   de grappin datées par vie avec leur point d'accroche en coordonnées monde. Champ
	//   omitempty, même raison de monter que v3/v4/v5/v7 : c'est la CLÉ DE REPRISE du
	//   backfill — la ligne joueur -> ancre n'existe que sur un artefact qui porte les
	//   tractions, un v7 doit se lire « à re-cuire », pas « à jour ».
	//   v8 -> v9 (2026-08-18, plan PLAN_POSES_EQUIPEMENT_PUBLICATION phase 2) :
	//   equipmentPlacements, les POSES d'objets d'équipement (mur, capteur, et les objets du
	//   monde qui partagent l'archétype) avec leur position monde, leur fenêtre [t0, t1],
	//   l'identifiant `eqip` du jeu, le poseur mesuré et son cap de visée. Champ omitempty,
	//   même raison de monter que v3/v4/v5/v7/v8 : c'est la CLÉ DE REPRISE du backfill — les
	//   marqueurs d'équipement posé n'existent que sur un artefact qui porte les poses, un v8
	//   doit se lire « à re-cuire », pas « à jour ». La couverture monte avec le champ
	//   (`coverage.placements`), et elle porte le découpage de bloc CALIBRÉ sur le film :
	//   sans elle, zéro pose par absence d'équipement et zéro pose par calibration refusée
	//   seraient indistinguables.
	//   v9 -> v10 (2026-08-18, plan PLAN_ORIGINE_POSES_ET_FAMILLES phase G) : chaque pose porte
	//   son ORIGINE MESURÉE (`origin` : `deployed` / `dropped` / `unknown`), et
	//   `coverage.placements` la croise avec la famille. UN CHAMP S'AJOUTE À UN SOUS-OBJET, ce
	//   qui d'ordinaire ne monte pas la version — ici c'est le SENS DU CALQUE ENTIER qui change,
	//   et c'est la raison. Mesure sur les 11 films calibrés : 3 242 des 3 661 poses à poseur
	//   mesuré (88,6 %) naissent dans les 2 frames ET les 1,5 m du DERNIER POINT de leur poseur
	//   — ce sont les objets qu'il PORTAIT, relâchés quand sa vie s'achève, pas des poses sur la
	//   carte. Un client v9 dessine donc un mur là où personne n'en a déployé, et sans montée de
	//   version il continuerait : la reprise du backfill se fait par SchemaVersion.
	//   L'hypothèse inverse — une DOTATION AU SPAWN, écrite avant mesure — est RÉFUTÉE : 4 poses
	//   sur 3 661 (0,1 %), et les 4 sont des vies de 0,13 à 1,49 s où début et fin se confondent.
	//   v10 -> v11 (2026-08-17, plan PLAN_ARMES_AU_SOL_2E_LECTURE phase 3) : `weaponPads` et
	//   `padPickups`, les SOCLES D'ARME du match — position, famille, apparitions, intervalles de
	//   présence bornés par le recensement des images-clés, cycle de réapparition quand il est
	//   ÉTABLI, et les occupations qui se sont achevées. Champs omitempty, même raison de monter
	//   que v3/v4/v5/v7/v8/v9 : c'est la CLÉ DE REPRISE du backfill — le calque des socles
	//   n'existe que sur un artefact qui les porte, un v10 doit se lire « à re-cuire », pas
	//   « à jour ». La couverture monte avec les champs (`coverage.groundWeapons`) : un film sans
	//   socle, un film dont toutes les armes sont des lâchers (Super Fiesta : 82,3 % de lâchers,
	//   zéro socle) et un film qu'on n'a pas su balayer rendent tous trois zéro socle, et seuls
	//   ces compteurs les distinguent. Le RAMASSEUR n'est PAS publié (`padPickups[].xuid` vaut
	//   `null`) : l'oracle des loadouts donne 88,1 % par slot de vie et 79,7 % par joueur, contre
	//   >= 90 % exigé, et le seuil n'a pas été rebaissé.
	//   v11 -> v12 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot A phase 1) :
	//   `scoreTimeline`, LE SCORE DANS LE TEMPS — la courbe des deux camps (par manche et en
	//   total) et les compteurs vivants de chaque joueur, avec `coverage.score`. Champ omitempty,
	//   et DEUX raisons de monter plutôt qu'une. La première est la clé de reprise habituelle : le
	//   document ne portait aucun score, donc ni la courbe de l'onglet Dominance ni le score
	//   vivant du rejeu n'existent sur un artefact v11 — il doit se lire « à re-cuire ». La
	//   seconde est un CORRECTIF : `objectives[]` était vide en production (le pont d'identité
	//   exige les lignes de match, que personne ne fournissait) et, quand un outil de mesure le
	//   remplissait, il était DÉCALÉ de `originMs` — 3,6 s à 50,8 s selon le match, d'où des
	//   pulses posés sur la mauvaise zone (appartenance stricte 9,9 % sans correction, 40,9 %
	//   avec). Un client v11 lit donc des actions absentes ou mal datées, ce qu'aucun champ
	//   optionnel de plus ne dirait.
	//   v12 -> v13 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot E phase 1) :
	//   `Point.p`, l'ÉLÉVATION DE VISÉE en degrés (positif = vers le haut, absent = à plat).
	//   UN CHAMP OPTIONNEL S'AJOUTE À UN SOUS-OBJET, ce qui d'ordinaire ne monte pas la version
	//   — ici c'est le SENS DU CÔNE DE VISÉE qui change, comme pour v9 -> v10. Le client
	//   dessinait le cône à sa longueur maximale sur chaque point porteur de cap : cette
	//   longueur affirmait une visée HORIZONTALE. Le film dit le contraire — sur les trois
	//   films de la mesure E.0.1 le mode tombe au centre du champ (1024 / 1013 / 1006) mais le
	//   support s'étend sur [537, 1490], soit environ ±82°. Un artefact v12 fait donc dessiner
	//   à plat des visées qui plongent, et la reprise du backfill se faisant par
	//   SchemaVersion, il continuerait sans montée de version. La convention est MESURÉE et
	//   non supposée : quantum 360/2048 = 0,17578°/pas (le candidat 180/2048 réfuté à 3,34x et
	//   4,06x), positif = haut (56 accords de signe sur 58 kills à |dz| >= 1 m), échelle
	//   contrôlée par l'oracle du kill (r = 0,930 / 0,916 / 0,969 sur 164 kills, écart médian
	//   de bout en bout 0,82 / 0,66 / 0,67°). Réserve écrite : toutes les valeurs observées
	//   tiennent dans la MOITIÉ centrale du champ, donc « ±180° sur tout le champ » et « ±90°
	//   sur sa moitié » sont indistinguables sur ce corpus.
	//   v13 -> v14 (2026-08-18, plan PLAN_OBJECTIFS_VIVANTS_2E_LECTURE phase 1 item 1.3) :
	//   `flagCarries`, LA VIE DE CHAQUE DRAPEAU de CTF — porté, porté sans fin datée, au sol,
	//   à sa base — avec `coverage.flagCarries`. Champ omitempty, même raison de monter que
	//   v3/v4/v5/v7/v8 : c'est la CLÉ DE REPRISE du backfill, et le drapeau vivant n'existe que
	//   sur un artefact qui le porte — un v13 doit se lire « à re-cuire », pas « à jour ».
	//   CE QUI EST MESURÉ, ET CE QUI NE L'EST PAS. Les bornes viennent des évènements de
	//   statistique NOMMÉS du statborg et du fil des morts (aucune estimation) ; le porteur du
	//   pont par instants de mort, donc du film seul ; la position de la piste PUBLIÉE du
	//   porteur. Le contrôle indépendant du marqueur d'image-clé confirme 37/37 des portages
	//   FERMÉS. Mais le LÂCHER VOLONTAIRE n'est daté par rien : un portage que rien ne ferme
	//   court jusqu'à la fin de l'axe, et il est publié sous un ÉTAT DISTINCT
	//   (`carried_open`) plutôt que confondu avec un portage établi — 0/5 de ceux-là sont
	//   confirmés par le marqueur, et c'est exactement ce que l'état dit. Le CRÂNE d'Oddball
	//   n'est PAS publié (ni canal ni oracle), ni le RETOUR AUTOMATIQUE (de 1,3 s à 35,8 s
	//   entre p10 et p90 sur le même film : aucune minuterie ne s'en déduit).
	//   v14 -> v15 (2026-08-18, plan PLAN_DRAPEAU_OBJET phase 2) : AUCUN CHAMP NEUF, et la
	//   version monte quand même — c'est le CONTENU de `flagCarries` qui change, sans qu'aucune
	//   clé ne bouge. L'OBJET drapeau (même archétype `ti=42` que les armes au sol, identifié par
	//   le manifeste du titre) réplique sa position quand PERSONNE ne le porte, et cette lecture
	//   répare deux défauts que v14 déclarait explicitement irréparables : le LÂCHER VOLONTAIRE
	//   se date (un portage que rien ne fermait — `carried_open`, borne haute jusqu'à la fin de
	//   l'axe — se ferme à l'instant où l'objet réapparaît AUX PIEDS de son porteur : 2 portages
	//   sur le corpus), et l'état `dropped` prend la position du dernier point de la piste LIBRE
	//   au lieu de celle du porteur mort (31 / 17 / 4 lâchers déplacés). Un artefact v14 et un
	//   v15 du même match publient donc les mêmes champs avec des valeurs et des intervalles
	//   différents — le cas qu'un client ne peut pas distinguer sans la version, et la reprise du
	//   backfill se fait par SchemaVersion.
	//   CE QUI N'EST PAS PUBLIÉ, ET C'EST LA MOITIÉ DU RÉSULTAT : la PISTE elle-même
	//   (`flagObjects`). Le contrôle de provenance, écrit AVANT la mesure, exigeait que >= 90 %
	//   des vies libres naissent à moins de 1,5 m d'un `flag_spawn` ou du porteur qui vient de
	//   finir ; la mesure rend 149/197 = 75,6 %. Le témoin tient largement (armes ordinaires
	//   soumises à la MÊME règle : 12,8 %, seuil <= 20 %) — la piste discrimine d'un facteur six,
	//   mais un quart des vies reste inexpliqué. Les deux corrections ci-dessus ne touchent QUE
	//   les vies nées aux pieds d'un porteur, c'est-à-dire la sous-population que ce même
	//   contrôle VALIDE ; une vie née à un socle est explicitement écartée.
	//   v15 -> v16 (2026-08-18, plan PLAN_EXPLOITATION_REGISTRE_FILM lot C-bis phase 2b) :
	//   `zoneStates`, L'ÉTAT DE CHAQUE ZONE du mode — qui la tient, depuis quand, et jusqu'à
	//   quel niveau de jauge elle a été contestée — avec `coverage.zones`. Champ omitempty,
	//   même raison de monter que les précédentes : c'est la CLÉ DE REPRISE du backfill, et la
	//   teinte de propriété n'existe que sur un artefact qui la porte.
	//   CE QUI EST MESURÉ, ET CE QUI NE L'EST PAS. Le canal est l'archétype `ti=13`, dont un
	//   SLOT EST UNE PROPRIÉTÉ RÉSEAU NOMMÉE et non une zone : la jauge de capture (tag 3) et le
	//   propriétaire (tag 4) vivent sur des slots DISJOINTS, et un troisième canal reste
	//   constamment neutre. L'appariement slot -> zone se refait à CHAQUE MATCH par la
	//   coïncidence d'un sommet de jauge avec une capture nommée attribuée géométriquement —
	//   cohérence 93,1 % et 98,4 % pour un seuil de 90 %, contre des témoins à 41-48 %
	//   (permutation des slots) et 51-57 % (sommet décalé de 20 s). La VALEUR du tag 4 est
	//   l'index d'équipe du capteur à 100,0 % (48/48) et 91,1 % (51/56) hors émissions neutres.
	//   Ce qui n'est PAS publié : « contesté » (les slots de rampe ne portent aucun tag 4 — la
	//   question est vide sur ce corpus), la clé de nommage comme identité de zone (absente de
	//   deux slots de jauge sur trois, et différente entre deux matchs de la même carte sur la
	//   troisième), et le propriétaire d'une colline de KOTH (le canal ne parle que là où il y a
	//   des captures nommées — la colline ne rend que sa PÉRIODE ACTIVE).
	//   v16 -> v17 (2026-08-19, plan PLAN_POWERUP_SOCLE_CATALYST phase 8) : AUCUN CHAMP NEUF À
	//   LA RACINE — c'est le CONTENU de `weaponPads` qui change. Les SOCLES DE POWER-UP y
	//   entrent par une SECONDE voie (`ti=37`), adjointe à celle des armes et jamais substituée
	//   à elle. Un artefact v16 du même match ne peut PAS les porter : la chaîne de production
	//   ne retenait un record `ti=37` que si sa position retombait sur une vie DELTA, et un
	//   socle ne bouge jamais — il était écarté PAR CONSTRUCTION. La reprise du backfill se
	//   fait par SchemaVersion, donc la version monte.
	//   CE QUE LA MESURE ÉTABLIT (plan, phase 3) : sur les deux films KOTH de Catalyst, le
	//   balayage sans l'oracle de vie delta rend 9 et 7 créations de `powerup_overshield` à la
	//   MÊME position au centimètre — (0,257 ; -0,003 ; 21,36) —, à 0,19 m du point que la
	//   phase 1 avait localisé par le croisement des trajectoires des quatre porteurs, sans
	//   lire un seul bit de record de création. Deux mesures indépendantes, aucun code partagé.
	//   Les deux films CTF de la MÊME carte n'en portent aucun : le sous-mode arme le socle.
	//   CE QUI N'EST PAS PUBLIÉ : `t1`. Un objet sans vie delta n'en a pas — la présence se
	//   borne par le recensement des images-clés, comme celle des armes. Et le power-up LÂCHÉ à
	//   une mort n'est PAS un socle : il bouge (vie delta) et naît là où son porteur meurt
	//   (`dropped`) — il reste publié par `equipmentPlacements` avec son origine, inchangé.
	//   v17 -> v18 (2026-08-19, plan PLAN_EXPLOITATION_REGISTRE_FILM lot C-ter volet 3) :
	//   `zoneStates[].gauge`, LA JAUGE DE CAPTURE EN DIRECT — la série datée de la valeur de la
	//   jauge de chaque zone PENDANT ses rampes (allégée : un point par variation >= 0,02 ou par
	//   seconde de rampe, rien hors rampe, premier et dernier point de chaque rampe toujours
	//   présents, chaque rampe fermée par son retour à zéro), sur la même échelle que `progress`, sur
	//   les modes à zones SIMULTANÉES seulement (jamais sur une colline de KOTH, où le canal est un
	//   compteur de transfert d'une seconde — volet 1), et `coverage.zones.gaugePoints`. Un
	//   sous-champ optionnel, et la version monte quand même — POUR CE QUE LE CLIENT CESSE DE
	//   DESSINER : l'arc de v16 traçait le SOMMET de la jauge par intervalle de propriété, une
	//   valeur tenue pendant toute la durée de la propriété, et il se lisait comme « capture en
	//   cours » alors qu'il n'était que le maximum atteint. Le client ne dessine plus cet arc, et
	//   ne dessine la jauge que d'un artefact qui porte la série : un artefact ANTÉRIEUR À v18 —
	//   un v16 comme un v17, qui n'a pas davantage de série — n'a plus d'arc du tout tant qu'il
	//   n'est pas re-cuit, et la reprise du backfill se fait par SchemaVersion.
	//   `progress` est CONSERVÉ dans le contrat (aucune clé ne bouge) : le sommet reste lisible
	//   pour qui le lit — mais son ÉCHELLE change avec celle de la jauge : la fraction de capture
	//   du JEU (0 = repos, 1 = pleine) remplace l'excursion mesurée du match, qu'une seule
	//   émission aberrante sous zéro suffisait à fausser (deux zones sur trois de `7344d24f`).
	//   LE NUMÉRO EST 18 ET PAS 17 : le 17 est parti aux socles de power-up ci-dessus, fusionnés
	//   avant nous — un numéro par montée, dans l'ordre de fusion.
	//   CE QUI EST MESURÉ SUR LES TÉMOINS : le poids de la série (<= +2 % de l'artefact exigé),
	//   le nombre de points par zone, et « la jauge monte avant la bascule du propriétaire » sur
	//   >= 90 % des captures de Bastion — chiffres au journal du lot (LOTCTER_VOLET3.md).
	//   v18 -> v19 (2026-08-25, lot 1 « lecture vide ») : `inventory[].empty` — POURQUOI une
	//   lecture d'inventaire ne rend rien. Deux valeurs seulement : `dead` quand le FIL DES MORTS
	//   corrobore (l'instant de la lecture tombe dans les 8 s qui suivent une mort du porteur du
	//   slot), `unknown` sinon. Le champ est optionnel et ADDITIF, et la version monte quand même —
	//   POUR CE QUE L'ARTEFACT AFFIRMAIT SANS LE SAVOIR : une lecture vide sortait NUE
	//   (`{"t":N,"slot":S}`), le client retenait cette lecture-là comme la plus récente <= T, et la
	//   fiche du joueur DISPARAISSAIT pendant ~20 s alors qu'une lecture pleine la précédait.
	//   17,4 % des lectures publiées sont dans ce cas (mesure du 2026-08-24, 6 721 records sur
	//   24 films). Un artefact v18 ne porte AUCUN marqueur : le client ne peut pas y distinguer
	//   « vide parce que mort » de « jamais lu », donc il se lit « à re-cuire » — et la reprise du
	//   backfill se fait par SchemaVersion.
	//   CE QUI EST MESURÉ, ET SON TÉMOIN : sur 8 films (1 419 records, 247 lectures vides), 88,3 %
	//   des lectures VIDES tombent dans la fenêtre de 8 s, contre 1,1 % des lectures PLEINES
	//   soumises à la MÊME fenêtre — 82x. Sur le film de vérité terrain : 93,8 % contre 0,7 %
	//   (137x). La fenêtre de 8 s n'est pas choisie : c'est le point de séparation maximale du
	//   balayage 2..20 s, et c'est aussi la médiane de réapparition relevée par `lives.go`.
	//   CE QUI N'EST PAS PUBLIÉ : une étiquette « mort » sur les 11,7 % que le fil des morts
	//   n'explique pas. Elles gardent `unknown` — le décodeur n'a rien lu, et personne ne sait
	//   pourquoi.
	//   v19 -> v20 (2026-08-25, lot 4.4 « suivi delta de l'inventaire ») : `grenadeReads` — LES
	//   GRENADES PORTÉES SUR LEUR PROPRE AXE, alimentées par DEUX canaux qui n'ont pas la même
	//   cadence : le record de biped des images-clés (~toutes les 20 s) et les composants i22/i47
	//   des paquets DELTA (transmis AU CHANGEMENT). Chaque lecture publie sa SOURCE (`kf` /
	//   `delta`), exactement comme `abilities` — le remède éprouvé contre « deux canaux, une seule
	//   étiquette ».
	//   POURQUOI LA VERSION MONTE : le client CONSOMME cet axe pour la boîte de grenades. L'âge
	//   médian de la lecture affichée passe de 10,00 s à 8,09 s (mesure sur 70 films, 28
	//   confrontables, 12 454 lectures delta). Un artefact v19 ne porte AUCUNE lecture delta —
	//   il se lit donc « à re-cuire », et la reprise du backfill se fait par SchemaVersion.
	//   CE QUI EST MESURÉ, ET SON TÉMOIN : les deux canaux sont décodés par des chemins SANS
	//   étape commune (motif d'ancrage dans les images-clés, marche de composants par les désers
	//   de production dans les deltas) et ils concordent à 97,94 % (714 couples sur 729). Contrôle
	//   croisé interne au canal delta : le masque d'i47 égale le bitmap des compteurs d'i22 sur
	//   1 925 records sur 1 925 — 100,0 %. Rappel de l'ancre sur les transitions attestées par les
	//   images-clés : 95,17 % (138 sur 145).
	//   CE QUI N'EST PAS PUBLIÉ, ET POURQUOI : les MUNITIONS delta. Implémentées et mesurées, elles
	//   sont REFUSÉES par leur propre mesure — concordance plafonnant à 92,80 %, et qui DESCEND
	//   quand on rapproche les deux lectures (88,06 % à 0,10 s contre 93,19 % à 2 s), ce qu'une
	//   consommation réelle entre les deux mesures ferait à l'envers. `Inventory.Am` reste donc
	//   alimenté par les seules images-clés. Détail : .ai/V7.5/replay2d/LOT4_SUIVI_DELTA_2026-08-25.md.
	//   v20 -> v21 (2026-08-27, phase D5 du plan des objectifs vivants) : LE PROPRIÉTAIRE DE LA
	//   COLLINE. Sur la voie du désignateur (KOTH), une période de colline se SUBDIVISE aux
	//   changements de main et chaque morceau porte son camp dans `ZoneSpan.Owner`.
	//   AUCUNE CLÉ NE BOUGE, ET C'EST LE CAS LE PLUS DANGEREUX : le champ `Owner` existait déjà,
	//   seul son CONTENU change. Un artefact v20 est donc STRUCTURELLEMENT valide et
	//   SÉMANTIQUEMENT périmé — il porte un `Owner` de colline toujours nul. Sans ce bump, la
	//   reprise du backfill (qui se fait par SchemaVersion) ne le rattraperait jamais et aucun
	//   rejeu déjà cuit ne montrerait la possession. C'est exactement pourquoi la règle du dépôt
	//   dit « un artefact vN doit se lire à re-cuire, pas à jour ».
	//   NIVEAU DE PREUVE : 88-89 % d'accord contre un témoin à 56 %, canal jamais réfuté, élu 4
	//   films sur 4, erreur concentrée aux BASCULES. Accepté par décision utilisateur du
	//   2026-08-26 (précédent : la garde de l'ouvrier à 88 %).
	//   CE QUI N'EST PAS PUBLIÉ, ET POURQUOI : le PORTAGE du crâne d'Oddball. Son identité est
	//   établie (`0x0017592C`, élu 4 films sur 4, né à 0,0 m du socle, à 3-6 ms d'un événement
	//   `th=10`) et elle entre au manifeste du titre — mais l'oracle du portage a été REFUSÉ par
	//   sa propre mesure : 40,6 à 66,7 % de trous à porteur unique contre un seuil de 90 %, et un
	//   témoin placé HORS trou qui rend le même signal dans 66,7 et 71,4 % des cas. Les ZONES DE
	//   TOTAL CONTROL non plus : le désignateur y rend jusqu'à 77 désignations simultanées sur un
	//   mode à trois zones. Détail : .ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_2026-08.md.
	//   v21 -> v22 (2026-08-27, lot VIP COURONNE) : LA COURONNE VIP. `vipCrown` publie les
	//   PÉRIODES DE PORT — chaque sélection `vip_selected` (`comp 22 A` = `TimesSelectedAsVip`,
	//   résolu au gate corrigé : 100 % par joueur x3 films, témoin décalé 0) ouvre une période,
	//   fermée par la mort du VIP ou la sélection suivante. Le champ est optionnel, mais la
	//   version monte pour la raison exacte des montées v14/v16/v21 : la reprise du backfill se
	//   fait par SchemaVersion, et un artefact 21 doit se lire « à re-cuire » — sans quoi aucun
	//   rejeu VIP déjà cuit ne montrerait la couronne.
	//   NIVEAU DE PREUVE : les périodes somment, par joueur, à `TimeAsVip` de l'API au SUB-SECONDE
	//   (recouv 100 % 3/3, 24/24 joueurs à +0,2-0,3 s), contre un témoin d'attribution aléatoire
	//   effondré (exactitude 8/8 contre 0-1/8). GARDE DE MODE chez l'appelant : `comp 22 A` vaut
	//   `flag_grabs` en CTF, donc la couronne n'est lue que sur un film reconnu VIP par
	//   `game_variant_name`. Détail : .ai/V7.5/replay2d/registre_film/VIP_COURONNE_PROTOCOLE.md.
	// - v23 (2026-08-28, lot PORTEUR ODDBALL) : `skullCarries` — LES PÉRIODES DE PORTAGE DU CRÂNE.
	//   Le porteur est le joueur dont les TICS DE SCORE DE MODE montent (`comp 0 A` =
	//   `skull_scoring_ticks`), un TRAIN de tics étant une période de portage ; nommé par le pont
	//   d'INSTANTS DE MORT PAR MANCHE (le slot est réattribué d'une manche à l'autre). Le champ est
	//   optionnel, mais la version monte pour la raison exacte des montées v14/v16/v21/v22 : la
	//   reprise du backfill se fait par SchemaVersion, un artefact 22 doit se lire « à re-cuire ».
	//   NIVEAU DE PREUVE : le portage a résisté à CINQ campagnes (proximité, traversée, score
	//   personnel : négatifs) ; le canal des tics tient — gate oracle porteur PRINCIPAL correct
	//   7/7 films, gate terrain manche 1 de d9781168 prises 9/9 et porteurs 8/9 (seuil 8/9),
	//   emplacement identifié par l'oracle films confondus. GARDE DE MODE chez l'appelant : `comp
	//   0 A` est le score de mode de tout mode, donc le porteur n'est lu que sur un film reconnu
	//   Oddball par `game_variant_name`. Le crâne LIBRE (`objectiveObjects`, v21) reste la couche
	//   POSITION ; celle-ci est la couche PORTEUR. Détail :
	//   .ai/V7.5/replay2d/registre_film/ODDBALL_PORTEUR_PROTOCOLE.md.
	// - v24 (2026-08-30, PLAN_RETOURS_UTILISATEUR_2026-08-29 §LOT F.1) : `equipmentEpisodes[].k`/
	//   `.a` — LES FRAGS ET ASSISTANCES DU PORTEUR pendant l'épisode (camo, surbouclier), et
	//   `coverage.equipment.killsRead` qui dit si la mesure a été TENTÉE pour ce match. Champs
	//   optionnels sur un sous-objet déjà publié, et pourtant la version monte, pour la raison
	//   exacte des montées v13/v18/v19 : un artefact 23 n'a jamais pu porter ces compteurs, et
	//   la reprise du backfill se fait par SchemaVersion. Décision utilisateur 8a/8b (DEC-7
	//   révisée) : GO à petite population — camo 35,2 % (25/71), surbouclier 55,6 % (10/18),
	//   global 39,3 % (35/89) en lecture STRICTE (`LineByLinePublishable`, la population qui
	//   affiche réellement des chiffres) ; re-mesure obligatoire après la cuisson de masse.
	//
	// v25 — LES PRISES ET LES LÂCHERS D'ARME (`weaponChanges`). Le composant d'identité d'arme
	//   du bipède n'entre au masque du flux delta que lorsqu'un emplacement CHANGE : chaque
	//   émission est donc un ramassage, un lâcher ou un échange, daté à la milliseconde. Jusqu'ici
	//   le document ne portait que `padPickups` — « ce socle s'est vidé quelque part dans cet
	//   intervalle », sans le joueur. Le champ est optionnel, mais la version monte pour la raison
	//   exacte des montées v14/v16/v21/v22/v23/v24 : la reprise du backfill se fait par SchemaVersion,
	//   un artefact 24 doit se lire « à re-cuire » — il ne peut porter aucun ramassage.
	//   NIVEAU DE PREUVE, et il est INÉGAL, ce qui doit se lire ici : le canal est JUSTE (sur
	//   5 627 tirs de trois films il ne retire jamais une arme encore utilisée) ; sa COMPLÉTUDE
	//   n'est PAS établie, faute d'oracle hors ligne — les images-clés sont trop grossières (20 s)
	//   et l'union des inventaires sature à 98-100 % avant même le canal. Ce qui a été mesuré à la
	//   place est la PLAUSIBILITÉ : hors drapeaux, 22 et 21 ramassages par match sur deux CTF
	//   Arena, composés d'armes de socle et de râtelier et jamais d'armes de départ, pour 10 et
	//   13 socles sur ces cartes. TROIS RÉFUTATIONS écrites : le lien vers l'objet du monde ne
	//   passe ni par la suppression de l'entité (1/71), ni par son attachement (1/21), ni par un
	//   appariement par les armes (5-12 % contre 70 % exigés). Détail :
	//   internal/analysis/replay/document_weapon_changes.go.
	//
	// v26 — LES RAMASSAGES ET LES CONSOMMATIONS D'ÉQUIPEMENT (`equipmentChanges`). La capacité
	//   d'armure suit la règle de l'arme en main : son composant (i48) n'entre au masque du flux
	//   delta que lorsqu'elle CHANGE. Le document portait déjà `abilities` — ce qu'un joueur
	//   PORTE ; ce calque dit ce qui lui ARRIVE. Champ optionnel, mais la version monte pour la
	//   raison exacte des montées v14/v16/v21/v22/v23/v24/v25 : un artefact 25 doit se lire
	//   « à re-cuire », il ne peut porter aucun ramassage d'équipement.
	//   NIVEAU DE PREUVE — MEILLEUR QUE CELUI DE v24, et pour une raison de format qu'il faut
	//   lire ici : le compteur de rotation d'i48 avance de 1 à chaque émission (zéro répétition
	//   sur 50 transitions, 3 films) et repart à 5 à la première émission de chaque vie (264 cas
	//   sur 269). Ce calque porte donc son PROPRE TÉMOIN DE COMPLÉTUDE — un pas supérieur à 1
	//   dénonce les émissions manquées et les compte : environ 16 pour 319 vues, soit ~95 % de
	//   couverture LUE et non supposée, publiée dans la couverture (`missedEstimate`).
	//   DEUX AUTRES MESURES fondent la sémantique : la porte ouverte est la CONSOMMATION et
	//   jamais la mort (17 cas, zéro dans la dernière seconde de la vie, la plus tardive laissant
	//   8,8 s à vivre) ; et la première émission d'une vie n'a pas un sens unique — à 0 ms de la
	//   naissance c'est une réapparition équipée (83 % des vies d'un film), tardive c'est un
	//   ramassage (médiane 16-18 s sur deux films d'arène, 0 % sous la seconde). Les
	//   réapparitions sont ÉCARTÉES : les publier fausserait le décompte du simple au double.
	//   Détail : internal/analysis/replay/document_equipment_changes.go.
	//
	// v27 — LES ARMES AU SOL OBSERVÉES (`groundWeapons`), ET LE RETRAIT DE LA MINUTERIE. La borne
	//   `until` du schéma 25 était une durée de TABLE (10/20/30 s) — l'utilisateur l'a refusée :
	//   « je veux juste voir quand elle est au sol et quand elle disparaît ». Le calque publie
	//   l'OBJET : position de repos, origine mesurée (dropped/spawned, la règle des poses), fin
	//   observée. TROIS FINS, TROIS PREUVES : `pickup` — une prise du flux delta tombe dans la
	//   fenêtre de vie à moins de 1,5 m (mesure fondatrice du 2026-08-30 : l'objet le plus proche
	//   d'une prise est à 0,61-0,75 m en médiane, témoin 4-7 m ; c'est la condition de reprise du
	//   REGISTRE_REPORTS levée par le canal du schéma 25) ; `seen` — dernière image-clé qui le
	//   recense, la disparition est dans les ~20 s suivantes non observées ; `open` — rien ne
	//   prouve la disparition. Les armes de socle (jamais bougé) restent au calque `weaponPads` :
	//   deux vérités pour un même objet seraient pires qu'une. Le champ `until` de
	//   `weaponChanges` est RETIRÉ avec sa table — un artefact 26 porte encore la convention.
	//   Détail : internal/analysis/replay/document_ground_weapon_items.go.
	//
	// v28 — LA FIN OBSERVÉE DES POSES D'ÉQUIPEMENT (`until`/`untilMax`/`end` sur
	//   `equipmentPlacements`). La mécanique de v27 appliquée au recensement `ti=37` que la
	//   chaîne des socles lisait déjà : dernière image-clé qui recense l'objet, première qui ne
	//   le recense plus. `t1` (fin du mouvement) interdisait par contrat de servir de
	//   disparition — un artefact 27 n'a AUCUNE fin d'affichage d'équipement. PAS de fin
	//   `pickup` ici, et c'est une réfutation mesurée (mesure D du 2026-08-30) : l'équipement
	//   tombe à la mort AVEC les grenades du mort, le lien spatial prise i48 -> pose attrape le
	//   mauvais objet (matrice GlobalID x rang non diagonale ; à candidat unique, 0-2 paires par
	//   film, incohérentes). Détail : internal/analysis/replay/equipment_placements.go.
	// - v29 (2026-08-31, chantier VISÉE À LA LUNETTE) : `Point.S` — LE PALIER DE LUNETTE.
	//   La version monte pour la raison EXACTE du schéma 13 (l'élévation de visée) : ce n'est pas
	//   un champ de plus, c'est le SENS DU CÔNE DE VISÉE qui change une seconde fois. Jusqu'ici le
	//   client dessinait la même ouverture pour un joueur à la hanche et un joueur à la lunette, et
	//   affirmait donc, sans le dire, que la visée était toujours aussi large. La reprise du
	//   backfill se faisant par SchemaVersion, un artefact 28 doit se lire « à re-cuire ».
	//   SOURCE : les événements `unit_zoom` (type 21) de la liste d'événements en tête de paquet —
	//   PAS le record de position, contrairement à `H` et `P`. C'est un état à BASCULE.
	//   NIVEAU DE PREUVE — vérité terrain, pas inférence : l'utilisateur a relevé à la main dans
	//   Theater, en première personne, les six périodes de lunette d'un joueur sur 00162144 ; les
	//   événements décodés apparient les SIX débuts à moins de 1,2 s, et le contrôle par
	//   translation de la chronologie sur ~3 200 décalages témoins (maximum repris sur les
	//   58 unités à chaque décalage) n'atteint JAMAIS ce score : p = 0,00 %. Le pont vers le
	//   joueur est une fermeture : index de référence + 512 tombe sur un slot bipède existant
	//   dans 98 % des cas contre 0 % pour toute autre base. RÉSERVE : seuls les événements en
	//   TÊTE de liste sont lus, donc les sorties sont sous-comptées — d'où le maintien borné
	//   (`zoomHoldUS`), qui refuse d'affirmer au-delà du délai au lieu de prolonger.
	//   CE QUI L'A RENDU POSSIBLE : sept campagnes avaient conclu « aucun événement de zoom dans
	//   la bobine » en lisant le type d'événement décalé d'UN bit ; leurs chaînes prétendument
	//   indépendantes partageaient cette erreur. Le négatif est réfuté.
	// v30 — LE RAMASSAGE NATIF (`pickups`), ET LA DATATION DES OCCUPATIONS DE SOCLE. La bobine
	//   porte un événement `biped_pickup` que personne n'avait décodé : le type 9 de la liste
	//   d'événements en tête des paquets delta. Sa grammaire est lue dans l'exe (descripteur
	//   0x144724e18, domaines 2/8/7, charge `R(3) classe + R(1) porte + R(32) catalogue`) et son
	//   cadrage est jugé par l'ORACLE DE TRAME sur deux films : longueur 50 bits sur 160/160
	//   événements, contre 0,0 % de trames exactes à ±1, 2 ou 3 bits.
	//   IL DATE, IL ATTRIBUE, IL NOMME. La référence de l'événement vaut `512 + index` = le slot
	//   du bipède ramasseur : UNE SEULE valeur d'écart sur 32/32 paires de vérité terrain (les
	//   ramassages que le canal i43..i46 voit aussi), témoin d'appariement permuté à 14-18 %.
	//   L'identifiant de catalogue vit dans le MÊME espace que `Loadout.W` (100 % des familles
	//   d'i43..i46 y figurent). Le R(3) sépare armes (classes 0/1) et reste (2/3, à 0,0 % d'armes
	//   sur 118 événements).
	//   CE QU'IL LÈVE : `padPickups` sort de l'intervalle anonyme de vingt secondes — instant
	//   exact `t` et `xuid` quand un ramassage natif de la même famille tombe dans la fenêtre.
	//   C'est la condition de reprise écrite au contrat de `PadPickup.XUID` (« un oracle plus
	//   RAPPROCHÉ que 20 s »), et ce n'est plus une inférence. RIEN N'EST EFFACÉ : une occupation
	//   non couverte garde son intervalle et son `xuid` à `null`.
	//   CE QU'IL NE FAIT PAS : remplacer `weaponChanges` (les deux canaux disent des choses
	//   différentes et s'accordent là où ils se recouvrent), ni prétendre à la complétude (seuls
	//   les événements EN TÊTE de liste sont vus — `coverage.pickups.multiEvent` publie la
	//   borne). Les classes non-arme sont publiées SUR MESURE : 80,5 % et 72,2 % d'entre elles
	//   n'ont aucune émission i48 du même slot à moins de 500 ms (témoin décalé 0,0 %) — elles
	//   comblent un trou, elles ne doublonnent pas `equipmentChanges`.
	//   Détail : internal/analysis/replay/document_pickups.go, pad_pickup_dating.go,
	//   internal/analysis/filmdec/biped_pickups.go, .ai/V7.5/film_re/NOTE_BIPED_PICKUP_2026-08-31.md.
	// v31 — LE NOM DE L'OBJET RAMASSÉ (`pickups[].family`), ET UNE NATURE À TROIS VALEURS.
	//   Le schéma 30 publiait un identifiant BRUT que rien ne nommait pour les classes non-arme.
	//   LE NOM VIENT DES FICHIERS DU JEU, PAS D'UNE STATISTIQUE — et c'est la raison de la
	//   montée. Le `R(32)` de ces classes est un GlobalID de tag `eqip` : le manifeste
	//   `[[equipment_objects]]` du titre le nomme, bâti en remontant la chaîne
	//   `sofd -> sofa -> {string_id, eqip}` dans les modules installés (2026-08-18).
	//   MESURE DU 2026-09-01, deux films de référence : 82/82 et 36/36 des ramassages non-arme
	//   résolus (8/8 identifiants distincts, les MÊMES sur les deux films), ZÉRO chevauchement
	//   avec le catalogue d'armes dans les deux sens, et concordance 2/2 avec les deux étiquettes
	//   que la corrélation avait acquises de son côté par deux voies indépendantes
	//   (`eef5d48d` = propulseur, `8e2dc574` = mur — un rang que la palette ne nommait pas).
	//   La voie de corrélation du lot précédent plafonnait à 19,5 % / 25,0 % : elle n'est pas
	//   fausse pour autant, c'est elle qui VALIDE la table aujourd'hui.
	//   `kind` PASSE DE DEUX À TROIS VALEURS, et la séparation vient du NOM : une fois les
	//   identifiants résolus, la classe 2 est grenade dans 100,0 % de ses événements et la
	//   classe 3 dans 0,0 %, sur les deux films, sans un seul identifiant réparti sur les deux
	//   classes. `item` N'EST PAS RENOMMÉ — il reste le repli des classes non-arme dont la
	//   nature n'est pas établie (le R(3) porte huit valeurs, quatre sont observées).
	//   CE QUE v31 REFUSE : descendre un libellé (`family` est un SLUG, la traduction reste au
	//   client — règle multi-titre) ; inventer un nom qu'elle n'a pas (un identifiant hors
	//   catalogue sort SANS `family`, et `coverage.pickups.unknownFamilies` le compte — le
	//   manifeste ne déclare que 21 objets, les trous doivent se voir) ; et publier l'ORIGINE
	//   d'une prise (socle de la carte contre objet tombé au sol), mesurée non concluante le
	//   2026-09-01 — 25,6 % d'injectivité contre 50 % exigés, et le dépôt ne déclare aucun point
	//   d'apparition d'équipement. La réfutation reste en place.
	//   UN ARTEFACT 30 SE LIT SANS CHANGEMENT : il ne porte simplement ni `family` ni les deux
	//   natures fines, et son `kind` vaut `item` là où un 31 dirait `grenade` ou `equipment`.
	//   Détail : internal/analysis/replay/document_pickups.go,
	//   .ai/V7.5/film_re/NOTE_NOMMAGE_ORIGINE_2026-09-01.md.
	//   RISQUE DE COLLISION DE NUMÉRO, consigné comme au 29->30 : ce lot prend le 31 sur
	//   `wt/pickup-nommage` alors que le 30 vient d'arriver sur `feat/v75`. L'arbitrage se fait
	//   au merge, par renumérotation.
	// v32 — L'ORIGINE D'UN RAMASSAGE NON-ARME : `Pickup.origin`, `spawner` ou `ground`.
	//   CE NUMERO LEVE UNE REFUTATION QUE v31 AVAIT ECRITE JUSTE AU-DESSUS, et il faut lire les
	//   deux ensemble. v31 refusait l'origine pour DEUX raisons : 25,6 % d'injectivite du juge
	//   temporel, ET « le depot ne declare aucun point d'apparition d'equipement ». La seconde
	//   est tombee : le catalogue des socles porte desormais 1 662 POINTS D'APPARITION sur
	//   56 cartes, extraits des `.mvar` par une recette qui interroge le catalogue Forge du jeu
	//   (`himap.EstPointDApparition`) — 16 types retenus sur 4 235 tags `food`.
	//   LA PREMIERE RAISON N'EST PAS CONTOURNEE, ELLE EST DEVENUE SANS OBJET : le juge temporel
	//   cherchait a apparier un ramassage a une NAISSANCE du film. `origin` n'apparie plus rien —
	//   il demande si le ramassage a eu lieu SUR un point que la CARTE declare, au centimetre et
	//   des la premiere image. Quatre voies purement filmiques avaient echoue avant (distance
	//   seule, fin de vie, levitation, recurrence des naissances) : aucune n'est reprise.
	//   `ground` NE REFAIT AUCUNE MESURE : il reutilise `EquipmentPlacement.Origin == "dropped"`,
	//   deja en production, qui rattache une pose a une fin de vie.
	//   L'ABSENCE EST UNE ABSTENTION, JAMAIS UN REPLI. Trois causes la produisent et la
	//   couverture les separe par `spawnPointsState` (`map_absent` / `not_established` /
	//   `established`), puis, une fois les points etablis, par le fait que le ramasseur n'avait
	//   pas de position assez proche dans le temps ou que le ramassage n'etait ni sur un point
	//   ni sur une pose.
	//   `originSpawner + originGround + originUnknown == items`, et un test tient l'invariant.
	//   LE TROU DE CATALOGUE EST UN CHAMP, PAS UN SILENCE — decision produit : seize cartes sur
	//   72 n'ont pas de points ETABLIS (leur `.mvar` a derive en amont), et la cuisson ne
	//   telecharge RIEN pour y remedier. Elle reste hors ligne ; le trou se comble par la CLI ou le sync.
	//   CE QUE v32 NE FAIT PAS : poser une origine sur les ramassages d'ARME (ils ont deja
	//   `GroundWeapon` avec son `End`/`Picker` — deux reponses a une question valent moins
	//   qu'une), et TYPER les points en grenade/equipement dans la decision (la nature du point
	//   est portee par le catalogue, elle n'entre pas dans le classement).
	//   UN ARTEFACT 31 SE LIT SANS CHANGEMENT : ses ramassages n'ont pas de cle `origin`, ce qui
	//   est exactement ce qu'un client doit traiter comme « non etabli ».
	//   Detail : internal/analysis/replay/pickup_origin.go,
	//   .ai/V7.5/film_re/NOTE_ORIGINE_POSITIONS_2026-09-01.md.
	//
	// (v33/v34 : pris 29/30 sur wt/bombe-visuel, renumerotes au merge du 2026-09-01 —
	// les schemas 29-32 etaient pris sur feat/v75, arbitrage ecrit aux schemas 30/31.)
	//
	// v33 — L'ARMEMENT DE LA BOMBE d'Assaut (`bombArmings`) : début du hold, instant armé,
	//   mèche (4 930 ms). Source : l'anneau du marqueur `ti=12 i14`, prouvé jauge d'armement
	//   par le protocole du 2026-09-01 avec tirage nul (13/13 Neutral Bomb CV 0,016, 4/4 Husky
	//   Raid, 0/1000 tirages nuls aussi bien). Le compte à rebours côté client n'existe que si
	//   l'artefact porte le calque — un artefact 32 se lit « à re-cuire ». REFUS : One Bomb
	//   (canal réfuté, CV 0,725), gardé par le NOM chez l'appelant ET par la confrontation
	//   locale aux explosions du même film ; l'ARMEUR (le navpoint est un marqueur d'écran,
	//   pas un acteur) ; les montées SOUS LE PLEIN (q<254 : holds relâchés et animation de
	//   recharge du marqueur, plafonnée à 253). Détail : document_bomb_armings.go.
	//
	// v34 — LE PORTEUR DE LA BOMBE d'Assaut (`bombCarries`) : les périodes de portage en
	//   intervalles de frames nommés par le xuid — le patron de `skullCarries` (v23) sur le
	//   canal des ARMES TENUES (la bombe est répliquée dans le composant weapon-state-type-info
	//   du bipède, famille 0x3fee4fcf — B1 2026-09-01, unique candidate des 9 films d'Assaut).
	//   PRISE = transition VERS la famille, LÂCHER = transition DEPUIS, la MORT du porteur
	//   ferme SANS émission (fil des morts, `BuildHeldObjectCarry`). Mesures : témoin Oddball
	//   46/46, porteur à la pose = détonateur statborg 13/17 (3 des 4 désaccords penchent
	//   CANAL par la position), mèche libre 27/28. GARDE : TOUTES les variantes bomb, One Bomb
	//   COMPRISE — le négatif de v33 vise l'anneau, pas ce canal. La bombe AU SOL n'est PAS
	//   publiée (aucun canal mesuré) : le client la dérive des périodes et des pistes (dernier
	//   point du lâcheur). Un artefact 33 se lit « à re-cuire ». Détail :
	//   document_bomb_carries.go.
	// v35 — LE RETOUR DU DRAPEAU DE CTF, dans ses deux moitiés. (1) LE RETOUR AUTOMATIQUE EST
	//   DATÉ. Le jeu ramène chez lui un drapeau resté au sol ; aucun compteur du statborg ne le
	//   dit, puisque personne n'est crédité — et les états `dropped` couraient donc jusqu'à la
	//   reprise ou la fin de l'axe, des lâchers de plus de deux minutes qui n'ont jamais existé à
	//   l'écran. L'OBJET, lui, le dit : une vie libre du drapeau qui NAÎT À SON SOCLE est le
	//   drapeau qui rentre, et le socle LE NOMME (ce que `flag_returns` ne fait pas). CONTRÔLE
	//   écrit avant la mesure et TENU : sur les retours que le statborg CRÉDITE, les deux chaînes
	//   — disjointes — tombent à la même frame dans 15 cas sur 15 (100 %, seuil 80 %, écart
	//   médian 1 frame ; compté par ÉVÉNEMENT crédité DISTINCT, `flag_returns` ne nommant pas son
	//   drapeau). (2) `flagReturnZone` publie la RÈGLE du mode : rayon de la zone de
	//   retour, minuterie à vide, durée à un défenseur — la CONTESTATION en est écartée par la
	//   mesure (sur 72 lâchers où un ennemi entre dans la zone, 56 finissent par une REPRISE : à
	//   1,3 m un ennemi ne conteste pas, il RAMASSE) et par l'observation de l'utilisateur.
	//   (3) LA VARIANTE « DRAPEAU NEUTRE » est
	//   reconnue et ne publie plus qu'UN drapeau, d'équipe -1, au socle du centre — le mode n'est
	//   pas dans le film, c'est l'OBJET qui tranche par le socle où il renaît, et la couverture
	//   publie le verdict avec ses deux comptes (`neutralFlag`, `neutralBirths`, `teamBirths`).
	//   Le rayon (1,3) est LU dans le script du jeu
	//   (`innerAreaMonitorRadius`) et CORROBORÉ par l'ajustement sur les films (minimum de
	//   dispersion à 1,3-1,5 m) ; les durées sont MESURÉES. L'occupation, elle, se compte chez le
	//   client : l'équipe d'un joueur n'est pas dans le film. Un artefact 34 doit se lire « à
	//   re-cuire » — ses drapeaux au sol n'ont ni retour automatique ni zone. Détail :
	//   internal/analysis/replay/flag_objects.go et .ai/V7.5/PLAN_CTF_ZONE_RETOUR_2026-08-30.md.
	//   (Ce lot avait pris le 29 sur wt/ctf-zone-retour ; renumerote 35 au merge du 2026-09-02.)
	// v36 — L'IDENTITÉ DES VIES : une track = UNE VIE (découpe `lifeGapUS`, la règle de
	//   `buildLifeSpans`, appliquée aux tracks), nommage PAR VIE (un slot recyclé porte une
	//   identité par occupant — le cas Sylvanus du retour user 2026-09-02), et LES BOTS
	//   (`roster[].bot` sans xuid + `tracks[].bot` sur les vies que le pont attribue à un
	//   index de BOT_METADATA). Un artefact 35 se lit « à re-cuire ». REFUS : l'héritage de
	//   slot pour les segments anonymes (c'était le bug), et le compteur de respawn réel
	//   (unité jamais calibrée — condition de reprise au registre).
	// v37 — LE COUP D'ENVOI, DATÉ PAR LE FILM (`t0FilmMs`). Le T0 servi jusqu'ici est ESTIMÉ
	//   des `first_joined_time` de l'API : sur 10-15 % des matchs ces horodatages collent au
	//   `start_time`, le T0 tombe à ~0, et le rejeu démarre sur des joueurs statufiés pendant
	//   tout le décompte — le défaut que l'utilisateur voit à l'écran. Le film le MESURE : la
	//   grille se lève d'un coup, donc le PREMIER MOUVEMENT des pistes date le coup d'envoi.
	//   TÉMOINS INTERNES AU FILM (2026-09-02, 101 artefacts) : rafale de départ à 6 joueurs de
	//   médiane sur 8-9 ; écart 1er -> 3e partant de 100 ms de médiane ; marge depuis la frame 0
	//   à 22 700 ms d'écart-type 299 ms, CV 0,013 sur 83 matchs. CONTRÔLE contre l'étalon :
	//   t0_film a un écart-type de 9 752 ms contre 12 764 ms pour t0_api sur les 49 matchs au
	//   T0-API sain, et ne descend jamais sous 25 907 ms là où l'API tombe à 17 804 ms. Sur les
	//   11 matchs dégénérés, 10 décomptes sur 11 dans la plage plausible 15-45 s.
	//   POURQUOI LA VERSION MONTE alors que le champ est OPTIONNEL — la raison exacte des
	//   montées v4 (l'origine) et v22 : la reprise du backfill se fait par SchemaVersion, et
	//   sans bump aucun rejeu déjà cuit ne démarrerait jamais sur le coup d'envoi.
	//   CE QUE v37 REFUSE : un zéro ambigu. Pas de mouvement détectable, rafale à moins de deux
	//   partants, ou premier mouvement à plus de 120 s de la frame 0 -> champ ABSENT (pointeur
	//   nil, même piège omitempty qu'`originMs`), refus journalisé, raison publiée dans
	//   `coverage.t0Film`. Et AUCUNE CONSTANTE ABSOLUE : la marge de 22 700 ms est un témoin
	//   documentaire, jamais une valeur servie — le détecteur mesure par match.
	//   Détail : internal/analysis/replay/t0_film.go et .ai/V7.5/PLAN_T0_FILM_2026-09-02.md.
	//   (Ce lot avait pris le 36 sur wt/t0-film pendant que l'identité des vies prenait le 36
	//   sur feat/v75 : renuméroté 37 au merge du 2026-09-02, arbitrage écrit aux schémas
	//   30, 31, 33 et 35.)
	// v38 — LA LECTURE FIABLE DES USAGES D'ÉQUIPEMENT (lots P1, P1bis et P3 du chantier du
	//   2026-09-03, décisions user D1-D4). Quatre champs optionnels et un changement de
	//   contenu, et la
	//   version monte pour la raison exacte des montées v14/v22/v25 : la reprise du backfill
	//   se fait par SchemaVersion, et un artefact 37 ne porte ni `translocations[]` (les
	//   téléportations datées ET SITUÉES par l'ÉVÉNEMENT type 117 du film — précision 18/18,
	//   rappel 8/8, rapport R1 ; le VA-ET-VIENT `fx/fy/fz` -> `tx/ty/tz` vient de la CHARGE de
	//   l'événement, layout lu dans l'exécutable et validé 18/18 à 0,00-0,26 m des
	//   discontinuités de piste, rapport R6 §1 — les six champs sont SOLIDAIRES, absents en
	//   bloc quand la charge n'a pas pu être déquantifiée, et `coverage.translocations.
	//   positioned` compte ceux qui les portent), ni `equipmentChanges[].recovered`
	//   (l'émission manquée retrouvée par la récupération GATÉE PAR LE TÉMOIN DE COMPTEUR —
	//   jamais par relâchement des gardes, le
	//   relâchement inconditionnel étant réfuté par +800 fausses acceptations sur 10 films),
	//   ni `equipmentChanges[].gap` (le saut RÉSIDUEL de compteur : un `from` sous gap se lit
	//   comme inconnu, pas comme faux). Le CONTENU des pistes bouge aussi : le filtre de
	//   vitesse est levé à ±200 ms d'un événement 117 du même slot (51/51 rejets à tort
	//   couverts, 0 fausse exemption, invariance bit à bit prouvée contre une implémentation
	//   de RÉFÉRENCE figée — la sémantique d'avant l'exemption — sur film sans tête 117).
	//   Elle porte ENFIN `abilityImpulses[]` (lot P3) : l'USAGE MESURÉ DU PROPULSEUR, daté par
	//   le corps `tag == 1` des composants i57/i59 — le MÊME dont le tag 3 porte le grappin —
	//   et ATTRIBUÉ par le rang i48 lu dans la MÊME VIE et ANTÉRIEUREMENT. 0,361 impulsion par
	//   vie de propulseur contre 0,011 par vie de répulseur (plus porté) et 0,000 sur 132 vies
	//   de grappin (R8 §8.8) ; vérité terrain Theater sur `1cd3848a` : 5 usages relevés,
	//   5 rendus, écart ≤ 1 s. LE CALQUE NE COUVRE PAS TOUS LES ÉQUIPEMENTS et le publie
	//   (`coverage.abilityImpulses.otherFamily`) : le RÉPULSEUR n'est PAS dans ce canal,
	//   négatif MESURÉ (R9, trois portes fermées). Et quand la chaîne d'attribution n'a pas pu
	//   tourner (palette non classée, aucune famille déclarée, aucune vie), les gestes tombent
	//   dans `noResolver` — un compteur À PART, parce qu'une indisponibilité déguisée en
	//   « autre équipement » se lirait comme une mesure.
	//   CE QUI N'EST PAS PUBLIÉ : la position de la faille AVANT le premier échange (aucune
	//   entité répliquée lisible — négatif mesuré R1 §1-3 ; la charge du 117 ne porte que
	//   {effet, départ, arrivée}, R6 §1.4). APRÈS, la balise est au point de DÉPART du saut :
	//   le va-et-vient publié suffit à la dessiner. Ni aucun usage du RÉPULSEUR, sur aucun des
	//   huit canaux jugés.
	//   LE LOT P5 (2026-09-04) ENRICHIT LA MÊME v38 : `abilityCharges[]` — les CHARGES
	//   RESTANTES (i56, quartet haut, rapport R11 : série 4→0 validée 5/5 au Theater, 36/36
	//   accroches de grappin appariées) et `coverage.abilityCharges`. DES ARTEFACTS 38 CUITS
	//   EXISTENT DÉSORMAIS hors répertoires de test (`1b2d9e08`, `1cd3848a`, les deux témoins
	//   de gate visuel — vérifié sur pièces le 2026-09-04), et le schéma reste à 38 QUAND
	//   MÊME : les deux ajouts sont purement additifs et omitempty, un lecteur 38 existant
	//   reste correct, et une montée à 39 n'aurait protégé aucun lecteur de plus (le parc
	//   entier est déjà <= 38 et la reprise du backfill par SchemaVersion ne perd rien) —
	//   la justification complète est à la chronique de document.go.
	//
	// - v39 (2026-09-05, FUSION DE DEUX CHANTIERS) : les trois montées 29, 30 et 31 du chantier
	//   VÉHICULES ET TOURELLES, numérotées sur une base antérieure, et la montée 39 du chantier
	//   ASSAUT (armement de la bombe en One Bomb, numérotée sur le 38), arrivent toutes POSÉES SUR
	//   LE 38 et fondues en UNE seule (décisions D3 et D13 du plan d'intégration) — aucun artefact
	//   n'a jamais été cuit à ces numéros-là. Ce qu'elles apportent, en CINQ temps — le cinquième
	//   naît DANS ce commit, sur le même numéro : le 39 n'a encore servi aucun artefact :
	// v39 (1) — LES VÉHICULES (`vehicles`). La vie de chaque véhicule `ti=40` du match : naissance
	//   (position du record de création, à des emplacements mesurés FIXES au rayon 0,00 m),
	//   identité de châssis (`MPPWord32`) résolue en famille de sprite, trajectoire échantillonnée
	//   sur la grille du document avec son CAP, et les ÉPISODES D'OCCUPATION (qui est à bord, de
	//   quand à quand, sur quel siège). Champ omitempty, même raison de monter que
	//   v14/v16/v21/v22/v25/v26/v27 : c'est la CLÉ DE REPRISE du backfill — le calque des
	//   véhicules n'existe que sur un artefact qui le porte, un v38 doit se lire « à re-cuire ».
	//   NIVEAU DE PREUVE, INÉGAL, et il doit se lire ici. SÛR : les positions (la grammaire bipède
	//   rend 99,4-100 % de pas sous 35 m/s sur la bande `ti=40`, contre 21,2-41,8 % pour celle des
	//   objets du monde) ; le CAP par la vélocité `i1` (écart médian au déplacement 1,7-2,1 deg sur
	//   4 films, R = 0,992-0,997, témoin par mélange déterministe 51-88 deg) ; l'identité
	//   `MPPWord32` (constance 100 % par vie, 5 valeurs sur 7 survivent au changement de build ET
	//   de carte, donc c'est un GlobalID de tag). PARTIEL, et publié comme tel : l'occupation, dont
	//   la primitive du « début de trou de position » n'attribue que 15,6-21,1 % des vies — mais à
	//   x20,3 et x30,5 le hasard, TÉMOIN FANTÔME NUL (0 contre 12 et 14) ; et la table de familles,
	//   dont la couverture est publiée châssis par châssis (`coverage.vehicles.unknownChassis`).
	//   CE QUE LA VERSION REFUSE, et c'est une réfutation mesurée (V3_DESTRUCTION_DATEE_2026-09-02,
	//   460 vies / 12 films / 8 gates, 7 échouent) : la DESTRUCTION datée. Zéro vie avec un occupant
	//   encore à bord à la fin serrée de son flux ; mort à bord ANTI-corrélée (3/80 = 3,8 % contre
	//   17/80 = 21,3 % au témoin à occupant décalé sur le MÊME intervalle) ; véhicule qui réplique
	//   encore 13 à 36 s (médiane par lot) après avoir été quitté — la fin de trajectoire est une
	//   MISE AU REPOS, pas une disparition. `VehicleTrack.End` vaut donc `inconnue`, et la
	//   disparition du sprite ne dit PAS que le véhicule a explosé.
	//   Détail : internal/analysis/replay/document_vehicles.go.
	//
	// v39 (2) — LES TIRS DES JOUEURS EMBARQUÉS (`shots[].v`). Un occupant attaché cesse de répliquer
	//   la position de son bipède : la porte des tirs, qui pose chaque tir sur cette position,
	//   écartait TOUS les tirs partis d'un véhicule. Mesure du 2026-09-02 sur `0d76e8f1` :
	//   1 166 tirs publiés, 12 épisodes d'occupation, ZÉRO tir pendant un épisode. Une seconde
	//   porte (`vehicle_shots.go`) reprend ces orphelins, leur donne la position INTERPOLÉE du
	//   véhicule et les marque de son slot (`v`) — sans quoi le client chercherait un pion qui
	//   n'existe pas. Champ omitempty, même raison de monter que v25/v26/v27 : c'est la CLÉ
	//   DE REPRISE du backfill, un artefact 38 est MUET au volant.
	//   NIVEAU DE PREUVE — un ORACLE INDÉPENDANT le fonde, et il ne doit rien à la géométrie (le
	//   record de tir ne porte AUCUNE position monde ; le critère est l'IDENTITÉ, le film écrivant
	//   son tireur). Les identifiants d'arme se lisent en deux moitiés de 32 bits : les 1 166 tirs
	//   publiés portent TOUS la moitié basse `0x42C9679F` (19 familles personnelles) ; les 23
	//   événements à moitié basse NULLE sont TOUS écartés par la porte du bipède (23/23) — une
	//   arme qu'on ne porte pas à pied, tirée par un joueur qui ne réplique plus. 17 de ces 23
	//   (73,9 %) tombent dans un épisode publié, contre 6 des 229 orphelins d'arme personnelle
	//   (2,6 %) : enrichissement x28. Témoin temporel (6 décalages de ±30 à ±120 s) : 12,2 en
	//   moyenne contre 23.
	//   CE QUE LA VERSION REFUSE : les tirs AMBIGUS (deux slots du même joueur répliquant tous
	//   deux une position) — leur signature est celle d'un joueur qui n'est PAS embarqué.
	//   Détail : internal/analysis/replay/vehicle_shots.go.
	//
	// v39 (3) — LA VISÉE DE CHAQUE OCCUPANT DE VÉHICULE (`vehicles[].rides[].aim`). Un occupant
	//   attaché cesse de répliquer sa POSITION (primitive du « trou »), et le dépôt en concluait
	//   qu'il ne répliquait plus RIEN : le cône du conducteur était dessiné au CAP DU CHÂSSIS,
	//   l'artilleur et le passager n'avaient aucun cône. La faute était dans le DÉTECTEUR —
	//   `ScanBipedRecords` exige un `i0` absolu et un masque commençant par 0, quand la forme la
	//   plus fréquente de la bande bipède est `i21,i25`, un record de VISÉE SANS POSITION
	//   (22 963 lectures sur le seul `0d76e8f1`). Champ omitempty, même raison de monter que
	//   v25/v26/v27 : c'est la CLÉ DE REPRISE du backfill, un artefact 38 fait viser tous
	//   ses occupants dans l'axe du châssis.
	//   NIVEAU DE PREUVE (V11_ORIENTATION_TOURELLE_2026-09-03, 5 films). PRÉSENCE : 4 832 à
	//   24 050 lectures par film, 46,5 à 231,2 par slot bipède, contre 0,2 à 0,9 par slot sur une
	//   bande FANTÔME de même cardinalité (x155 à x925). JUSTESSE : appariée à la lecture `i21`
	//   AVEC position du même slot à moins de 200 ms, écart médian de cap 0,2 à 0,5 deg
	//   (R 0,979-0,989) contre 75,7 à 93,7 deg au témoin par mélange (R 0,011-0,134) — la
	//   référence étant `Point.H`, déjà publié et validé par l'oracle du kill. COUVERTURE :
	//   35 / 35 (100 %) des épisodes attestés par la sortie portent au moins une visée à bord, à
	//   5 à 46 lectures/s, quand le même épisode porte 0 ou 1 lecture `i21` avec position.
	//   UTILITÉ : la visée n'est PAS le cap du châssis — médiane 15,7-21,8 deg, q3 39,6-52,9 deg.
	//   CE QUE LA VERSION REFUSE, réfutation mesurée AVEC TÉMOIN : l'orientation de la TOURELLE en
	//   tant qu'objet. L'entité tourelle ne réplique rien (139,6 et 85,5 en-têtes par slot contre
	//   86,3 et 194,2 au FANTÔME, formes de masque plates) et `i31`/`i41`/`i42` de `ti=40` ne sont
	//   jamais émis. Le cône de l'artilleur vient de L'HOMME, pas de la tourelle.
	//   Détail : internal/analysis/replay/vehicle_rides_aim.go.
	//
	// v39 (4) — L'ARMEMENT DE LA BOMBE EN ONE BOMB (étape E2-ter du plan d'Assaut, 2026-09-04,
	//   arbitrage utilisateur). AUCUN CHAMP NEUF : ce qui change est le CONTENU du calque
	//   `bombArmings`, et la version monte pour la raison exacte des montées v14/v22/v25/v37 —
	//   la reprise du backfill se fait par SchemaVersion, et un artefact 38 d'un match One Bomb
	//   ne porte AUCUN armement là où il en porte désormais.
	//   CE QUI A CHANGÉ. Le calque était gouverné par une garde de mode DOUBLE, dont la
	//   première écartait One Bomb PAR SON NOM : sous la lecture SIMPLE (montée contiguë,
	//   mèche fixe de 4,93 s) le protocole du 2026-09-01 y avait RÉFUTÉ le signal (CV 0,725,
	//   87/1000 tirages nuls aussi bien). La lecture « MÈCHE PAUSABLE » du même jour l'explique
	//   — 9/9 explosions portées, médiane 16,18 s, CV 0,017, 0/1000 — et elle est maintenant EN
	//   PRODUCTION : segments contigus (le cycle de recharge du marqueur finit à son MINIMUM et
	//   sort de lui-même), armement = segment qui finit à son sommet PLEIN, TENUE DE
	//   DÉSARMEMENT qui SUSPEND la mèche (pente 14-26 quanta/s, contre 138 pour une chute
	//   d'explosion), et MÈCHE MESURÉE SUR LE FILM (médiane des délais corrigés) au lieu d'une
	//   constante unique — 4,93 s en Neutral Bomb, 5,1 s en Husky Raid, ~16,2 s en One Bomb
	//   sortent de la MÊME règle, sans qu'aucun code ne branche sur le nom de la variante.
	//   CE QUI N'A PAS CHANGÉ, ET C'ÉTAIT L'EXIGENCE : les témoins Neutral Bomb (13/13) et
	//   Husky Raid (4/4) au chiffre près. Et la GARDE 2 reste, seule et tout-ou-rien par film :
	//   une explosion sans armement dans la fenêtre de sens, ou des mèches du film qui se
	//   contredisent, retiennent le calque ENTIER.
	//   Détail : filmdec/navpoint_radial_segments.go, replay/bomb_armings.go,
	//   replaybuild/zones.go et .ai/V7.5/PLAN_ASSAUT_STATS_2026-09-04.md (E2-ter).
	//
	// v39 (5) — LES CINQ STATISTIQUES D'OBJECTIF DE L'ASSAUT (`bombStats`) et ses FAITS DATÉS
	//   (`bombEvents`), 2026-09-05, étape G.2 de l'intégration. DEUX CHAMPS NEUFS SUR LE MÊME
	//   NUMÉRO, et la raison est mesurée : le 39 n'a JAMAIS cuit un artefact hors répertoires
	//   de test (le 38, lui, en avait deux — cf. la politique du lot P5 ci-dessus), et cette
	//   intégration est le commit qui le met au monde. Une montée à 40 marquerait « à
	//   re-cuire » un parc qui l'est déjà tout entier : elle ne protégerait aucun lecteur.
	//   CE QUE C'EST. `bomb_detonations`, `bomb_arms`, `bomb_grabs`,
	//   `time_as_bomb_carrier_seconds`, `bomb_carriers_killed`, par joueur — les statistiques
	//   que l'API 343 NE PUBLIE PAS pour ce mode. La cause du silence est structurelle et
	//   mesurée (Ghidra, 2026-09-04) : la famille `BombStats` du moteur est de la TÉLÉMÉTRIE
	//   Bond, écrite par un sérialiseur d'événement, jamais un composant d'entité — le film ne
	//   peut pas la répliquer, par construction. Elles sont donc RECONSTRUITES : le statborg
	//   `comp 0` canal A pour les explosions, le canal des armes tenues pour le portage,
	//   l'anneau `ti=12 i14` pour l'armement, et une JOINTURE (lâcher, puis porteur actif) pour
	//   nommer l'armeur — que le Lua du moteur, lui, ne nomme jamais (`activatingTeam` seul).
	//   POURQUOI DANS L'ARTEFACT : les quatre sources ne vivent en pleine fidélité qu'à la
	//   cuisson — le document publie le portage en FRAMES, sans les périodes non pontées ni le
	//   recalage d'horloge. Les recalculer chez le consommateur en ferait un second décodeur
	//   du même fait, moins précis.
	//   LES CINQ SONT MESURÉES, `bomb_carriers_killed` COMPRIS (lot G.6, 2026-09-05) : ce
	//   commentaire a porté l'inverse, au motif que la paire tueur/victime n'existait pas dans
	//   la chaîne de cuisson — faux, seule la VICTIME n'était pas résolue. Source non lue =
	//   champ absent chez tous les joueurs, jamais zéro.
	//   Détail : internal/analysis/replay/bomb_stats.go et bomb_stats_document.go.
	// v40 — LE PONT D'IDENTITÉ COMPLÉTÉ PAR LE TRIPLET (2026-09-06). Aucun champ ajouté : c'est
	//   le CONTENU de `objectives` qui change. Depuis `d173b1a8c` (2026-08-28), le calque
	//   résolvait l'identité slot -> joueur par les seuls INSTANTS DE MORT, qui en exigent
	//   trois : un joueur qui meurt moins de trois fois — le meilleur du match, celui qui porte
	//   le drapeau — n'était plus nommé, et ses actions disparaissaient. Mesuré sur `c0a82e88` :
	//   17 actions avant, 12 après, les deux seules actions de famille `flag` perdues.
	//   POURQUOI LA VERSION MONTE SANS CHANGEMENT DE FORME : un artefact 39 cuit avant le
	//   correctif peut manquer des actions sans que rien ne le dise, et `backfill-replay` saute
	//   un artefact qui porte la version courante — il garderait son calque appauvri. C'est la
	//   règle des montées v3/v4/v5/v14/v22/v25/39 (« un artefact vN doit se voir comme à
	//   re-cuire »), pas l'exception du lot P5, qui ne valait que parce qu'aucun artefact 38
	//   n'existait alors hors témoins de gate.
	//   Détail : internal/analysis/objectiveevents/slotidentity_rounds.go (CompletedByLines).
	// v41 — TROIS CALQUES RATTRAPENT « UNE TRACK = UNE VIE » (2026-09-06). Aucun champ ajouté.
	//   `48cf4905d` a découpé les pistes à `lifeGapUS` ; trois consommateurs supposaient encore
	//   « un slot = une piste » et ne gardaient que la DERNIÈRE : le nommage des vies fermées
	//   (`145908d1` : 51 pistes nommées pour 53 slots au pont, 29 tirs sur des pistes anonymes),
	//   les tractions de grappin (`879a4dba` : 23 accroches lues, 15 tractions publiées) et les
	//   épisodes de camo/surbouclier (`82f29378` : son unique surbouclier perdu). La version
	//   monte pour la raison des montées v39/v40 : un artefact 36 à 40 est appauvri sans que sa
	//   forme le dise, et `backfill-replay` saute un artefact à la version courante.
	//   Détail : internal/analysis/replay/{closures.go, grapple_lines.go, equipment_episodes.go}
	//   et .ai/V7.5/v2/INSTRUCTION_REGRESSIONS_2_4.md.
	// v43 — UNE VIE ANONYME N'EST PAS UNE ABSENCE (2026-09-06). Aucun champ ajouté : c'est le
	//   CONTENU de `skullCarries` (et, latent, de `bombCarries`) qui change. Le gate de présence
	//   de `af89b091b` (2026-08-30) n'indexait que les vies NOMMÉES et lisait « aucune vie nommée
	//   de X ne couvre l'intervalle » comme « X est absent de la carte » — alors que le pont
	//   laisse des vies anonymes (18 slots sur 160 sur `d9781168` — 142 portent au moins une vie
	//   nommée, ces 18-là aucune). Mesure par une chaîne INDÉPENDANTE : en Oddball le score EST le
	//   temps de portage, et la feuille de match donne 191 s / 196 s par équipe sur `d9781168` ;
	//   l'artefact publiait 60,1 s / 147,4 s, contre 172,5 s / 158,8 s pour l'artefact du parc au
	//   schéma 23. Le gate écartait 6 portages sur 36 (32,6 s) et en rognait 4 autres (91,2 s).
	//   Touche aussi `51ebbc0f` (7 portages sur 14) et `24dbb67d` (3 sur 20). La version monte
	//   pour la raison des montées v39/v40/v41 : un artefact 23 à 42 est appauvri sans que sa
	//   forme le dise, et `backfill-replay` saute un artefact qui porte la version courante.
	//   42 EST RÉSERVÉ au complément des ports de drapeau, en cours sur une autre branche : deux
	//   chantiers parallèles ne peuvent pas revendiquer le même numéro, celui-ci prend 43.
	//   Détail : internal/analysis/replay/skull_carries.go (carrierPresence.gate) et
	//   .ai/V7.5/v2/INSTRUCTION_RESIDUS_2026-09-06.md.
	if SchemaVersion != 43 {
		t.Fatalf("SchemaVersion = %d, attendu 43 : incrémenter exige une raison écrite ci-dessus "+
			"(un champ optionnel de plus n'en est pas une)", SchemaVersion)
	}
}
