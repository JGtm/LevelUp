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
	if SchemaVersion != 17 {
		t.Fatalf("SchemaVersion = %d, attendu 17 : incrémenter exige une raison écrite ci-dessus "+
			"(un champ optionnel de plus n'en est pas une)", SchemaVersion)
	}
}
