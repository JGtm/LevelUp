# Plan — Parite fonctionnelle du rejeu 2D + generation pour tous les matchs

> Demande utilisateur du 2026-08-13 soir. Sources qui font foi : le POC (artefact claude.ai
> eb7b8af2 — LA spec de rendu), la section REPLAY 2D du Backlog Notion (cahier des charges,
> lu en entier le 13/08), et l'inventaire sur pieces wf_6986830d (4 volets, résultats
> détaillés dans le task-output de la session superviseur). Execution sous le contrat
> `plan-execution` : lots ordonnés, gates par lot, statuts [x]/[~]/[!], zero hors-perimetre
> (les decouvertes se notent en §Decouvertes). Branche feat/v75, commits par lot, PAS de
> push (orchestrateur). JAMAIS deux commandes Go en parallele ; UN SEUL decodage filmdec a
> la fois par process (globaux mutables + map compWidthObs sans verrou — race reelle).

## Decisions produit (tranchees avant execution)

1. **Couleur des effets** : l'arbitrage acte du lot 3.2 est CONSERVE (famille = FORME,
   couleur = TIREUR — regle color-tokens, documente en tete de shotEffects.ts). La palette
   hsla par famille du POC n'est PAS reintroduite. Si le gate visuel du user la redemande,
   c'est un echange de palette isole, pas une refonte.
2. **Sons** : DEBLOQUE le 13/08 par la source fournie par l'utilisateur
   (`D:/Halo Infinite Gun Sounds.zip`, 76 WAV) — le lot 5 est livre (2026-08-14).
   L'extraction audio DU JEU reste parquee (« n'y revenez pas sans element nouveau ») :
   ce lot n'y touche pas, il range et joue un pack fourni.
3. **Equipements actifs** (effet SUR TOUTE LA FICHE — precision user 13/08 : surbouclier =
   fiche doree, camouflage = effet de verre, translocateur = bordure animee) : l'etat
   « actif » n'a aujourd'hui AUCUNE source decodable etablie (i57 refute comme interrupteur,
   compteur d'usages = adresse Cheat Engine). Le lot 2 contient une INVESTIGATION bornee
   (events MobilityAction* de filmdec) ; si rien n'est decodable offline → registre en RE,
   pas de promesse, pas d'effet simule.
4. **« rangees » / « aucune »** : ce sont deux etats MESURES distincts (armes rangees D=2 ;
   munitions non ecrites = plein). Decision : les rendre muets pour l'utilisateur final —
   pictogrammes discrets, un seul tooltip simple FR/EN chacun, zero jargon de flux.

## Lot 1 — Fiches joueur (produit, rapide, tres visible)

- [x] 1.1 HAUTEUR CONSTANTE vivant/mort : reserver la place (min-height ou rangees
      fantomes) — aujourd'hui ReplayWeaponsRow/InventoryRow rendent null a la mort et la
      fiche se compacte (ReplayTeams.tsx l.140-209, rosterLogic l.239-246).
      FAIT (2026-08-13) : PlayerCard rend TOUJOURS 4 rangees — zone vitalite/retour a
      hauteur FIXE (h-3.5), zone armes min-h-[18px], zone inventaire min-h-4 — le contenu
      change a la mort, jamais la place. Test structurel « meme nombre de rangees
      vivant/mort » ajoute (ReplayTeams.test.tsx) ; l'egalite au pixel = gate visuel user.
- [x] 1.2 TOOLTIPS dev supprimes : ammoSlotHint, abilityUnknownHint, respawnUnknownHint,
      weaponSwapHint, grenadeSelUnknownHint, ammoNoneHint, drawnUnknownHint,
      weaponInHandHint (+ les title= porteurs). CONSERVES : gamertag, Bouclier/Sante,
      aria-labels a11y (barre respawn, jauge charge).
      FAIT (2026-08-13) : les 8 cles retirees d'i18n.ts + leurs title= ; les infobulles
      composites gardent leur partie INFORMATIVE (detail du swap + age, age de lecture) et
      perdent la phrase de methode. Conserves intacts : title gamertag, title+aria des
      barres Bouclier/Sante, aria barre respawn + jauge charge. Decouverte : weaponInHand
      (sans Hint) etait deja orphelin — retire avec (0 code mort).
- [x] 1.3 « rangees »/« aucune » remplaces selon la decision 4 (i18n FR+EN).
      FAIT (2026-08-13) : deux pictogrammes discrets en currentColor (HolsterMark : fleche
      vers l'etui ; AmmoFullMark : chargeur rempli), UN tooltip simple chacun, FR+EN
      (holsteredLabel « Armes rangees »/« Weapons holstered », ammoFullLabel « Munitions
      pleines »/« Ammo full »), role=img + aria-label ; tests mis a jour + 1 nouveau
      (pictogramme plein). « degainee ? » (selecteur non lu, hors decision 4) garde son
      jeton texte, sans tooltip de methode (1.2).
- [!] 1.4 INVESTIGATION bornee equipement actif : les events MobilityAction*
      (filmdec/components_biped_ability.go) donnent-ils un USAGE date de capacite,
      offline-pur, sur 000d5950 ? Si oui : effet plein-fiche a l'usage (duree fixe courte),
      les 3 rendus de la decision 3. Si non : ligne registre RE, item [!] justifie.
      INVESTIGATION FAITE, verdict NON-DECIDABLE (2026-08-13) : instrument
      `internal/analysis/filmdec/i54_research_test.go` (garde I54_FILM, lecture seule).
      Mesure sur 000d5950 : 2 819 records delta portent i54, 67 EPISODES discrets
      (~0,6 s, 1-3 par vie, 39/99 vies, 3/67 seulement pres du spawn) — c'est un
      EVENEMENT date en cours de vie, pas du bruit de mouvement. MAIS rien offline
      n'etablit que c'est un usage d'EQUIPEMENT (vs escalade/mantle) et aucune IDENTITE
      d'equipement ne voyage avec — la decision 3 exige l'identite pour choisir le rendu.
      Pas d'effet simule. Ligne au REGISTRE_REPORTS avec deux conditions de reprise
      (croisement i56 energie de capacite, ou verite Theater datee).

Gate 1 : typecheck purge + lint + vitest verts ; capture des fiches avant/apres pour le
gate visuel user (mort ET vivant cote a cote, hauteur identique).
PASSE (2026-08-13) : tsc -b OK (cache .tmp purge), eslint 0 erreur (19 warnings
preexistants hors match-replay), vitest 408 fichiers / 3627 tests verts. Go : vet OK,
instrument i54 SKIP proprement sans I54_FILM (CI sure). La capture visuelle est remise
au user (instructions dans le compte rendu du lot — la session ne juge pas l'aspect).

## Lot 2 — Effets de mort, grenades, lancers

- [x] 2.1 EFFET DE MORT par famille d'arme (portage drawKillFx du POC, l.2884 de
      l'artefact) : oriente tueur→victime quand le couple est complet (regle POC 89/93),
      marqueur non oriente sinon ; extremite reelle (cible=true) ; melee = arc SANS eclair
      (seuil 8 m mesure) ; formes = shotEffects existant, duree KFX_HOLD 1,4 s. Donnees :
      kills du feed (killer/victim/weapon_key, horloge recalee) + positions via doc.tracks
      (fenetre death, patron posOfNameAt). Manque Go : la famille fx par weapon_key sur les
      events du feed (aujourd'hui [shot_effects] n'est joignable que par id d'arme film) —
      petite modif (servir fx avec weapon_key, ou table cote client depuis weaponLabels).
      FAIT (2026-08-13) : cote Go, le document v3 publie `killEffects` (weapon_key →
      famille, table [shot_effects] servie telle quelle — option « servir fx avec
      weapon_key » retenue via l'artefact, l'API ne lit aucun TOML en requete) ;
      [shot_effects] gagne les cles kill-only (grenades frag/plasma=explosive,
      dynamo=shock, bandit et ma5k_avenger=ballistic ; Spike SANS weapon_key = neutre,
      mesure killicon). Cote web : killFx.ts pur (precalcul monde, horloge du fil
      reutilisee, positions vivant=positionAt / vie close=fenetre DEATH 1,5 s), orientation
      SEULEMENT sur couple complet, marqueur pointille sinon (drawDeathMarker), extremite
      reelle (`target` : explosif et aiguilles a l'extremite), melee = arcs sans eclair +
      arc de liaison sous 8 m monde (`meleeLink`), couleur = tueur (arbitrage 3.2).
      8 tests killFx + 1 test melee liaison. LIMITE (decouverte) : la melee GENERIQUE
      (classe sans weapon_key) tombe sur le rendu neutre — le feed ne porte pas la classe.
- [x] 2.2 TIRS : l'effet par famille sur TOUS les tirs publies EXISTE (drawShotsLayer, 475
      tirs) — verifier pourquoi il est invisible a l'oeil (durees/intensites vs POC :
      SHOT_HOLD 0,6 s, longueurs) et recaler sur le POC. Pas de re-conception.
      FAIT (2026-08-13) : deux causes identifiees — remanence partagee 1,4 s (trait
      trainant dim) et longueur 26 px. Recale : SHOT_HOLD_MS 600 ms distinct, SHOT_LENGTH
      62 px, courbes d'extinction POC (balistique deux horloges eclat²/trait^1,5 largeur
      2,2 ; plasma racine ; choc ^1,6 largeur 1,7 ; lumiere 6,5/2,0 ; melee 2e arc).
      canvasRecording.test.ts mis a jour (il figeait l'ancienne geometrie).
- [x] 2.3 GRENADES : publier le lien grenade↔projectile dans l'artefact (l'appariement
      existe deja dans grenades.go, fenetre 200 ms) + effet par TYPE au point Rest :
      Shock/Dynamo (rang 2) = effet ELECTRIQUE persistant ~2-3 s (geometrie drawShock),
      autres = halo « derniere position connue » — JAMAIS « impact » (aucun event de
      detonation dans le film, seul Rest certifie 78/79 ; frag : la replication cesse
      ~1,4 s avant la meche).
      FAIT (2026-08-13) : `Grenade.proj` (*int — piege omitempty, l'index 0 est valide)
      pointe le projectile PUBLIE (buildProjectiles rend la table brut→publie, construit
      avant les lancers) ; temoin : 65/70 lancers lies (= mesure POC). Web : l'effet se
      pose au DERNIER point replique du vol lie — MESURE sur le temoin : at-rest n'existe
      QUE sur les Spike (15/21), jamais frag/plasma/Dynamo (0/44, l'objet detone au lieu
      de s'immobiliser) — l'exiger eteignait trois types sur quatre (premiere version
      corrigee au gate) ; la certification reste portee (champ rest). Dynamo rang 2 =
      nappe electrique 2,5 s (geometrie brisee, germe stable), autres = halo discret
      1,4 s ; note i18n FR/EN sous le canvas « derniere position connue ». SchemaVersion
      2→3 (cle de reprise du backfill lot 6) + openapi-gen + generate-types meme commit.
- [x] 2.4 LANCERS mis en evidence : type distingue sur la carte (vignette
      GrenadeLabels[rank].img deja publiee, ou forme par rang) + pulse ephemere sur la
      fiche au lancer (badge .gic du POC, l.534-543).
      FAIT (2026-08-13) : carte — vignette du type au-dessus de l'anneau (masque HUD teint
      a l'encre du theme UNE fois par theme via source-in, cache par rang ; rang sans
      visuel = anneau seul). Fiche — badge .gic : premiere place de la rangee d'armes,
      token warning, pop 0,18 s (reduced-motion: none), remanence 1,4 s, HORS estompage
      d'age (bloc separe — les opacites CSS se multiplient) ; jointure par l'index de FILM
      du roster (l'auteur est ecrit). 3 tests grenadeThrowActive.

Gate 2 : tests web verts ; si modif du contrat artefact : bump SchemaVersion + openapi-gen
+ generate-types meme commit ; re-cuisson du temoin 000d5950 + verification visuelle a
remettre au user (liste des instants a regarder : un kill balistique, un plasma, une melee,
un lancer Shock).
PASSE (2026-08-13) : tsc -b OK (cache purge), eslint 0 erreur (19 warnings preexistants),
vitest 410 fichiers / 3644 tests verts (14 skips preexistants), go test replay + mappings +
replaylabels + handlers + service OK, go vet OK, golden assembly regenere (seule la ligne
schema change). Temoin 000d5950 re-cuit en v3 (`--map cliffhanger` — la cle du catalogue
est le NOM de carte, ridgeline n'est que le module) : 65/70 lancers lies (dont 15 vols
at-rest, tous Spike), 29 cles killEffects. La verification visuelle est remise au user
(liste des instants dans le compte rendu du lot — la session ne juge pas l'aspect).

## Lot 3 — Callouts (regression POC, tout existe hors depot)

- [x] 3.1 MATIERE PREMIERE — CORRIGE 13/08 (user) : le jeu EST installe sur
      C:/Program Files (x86)/Steam/steamapps/common/Halo Infinite (verifie : deploy/ds/
      levels/multi = 31 dossiers, deploy/any present). `himap.LevelsDir`/deploy_root.go
      resout deja cette racine (preuve : lot bornes). Source PRIMAIRE = l'installation ;
      les copies LevelUp-re/jeu_deploy_* et la clef PNY ne sont que des secours. L'agent
      d'inventaire avait teste l'ancien chemin D:/SteamLibrary — conclusion invalidee.
      VERIFIE (2026-08-13) : installation presente (31 dossiers ds, any present),
      callouts.py/callouts_all.py/callouts_i18n.csv (816 lignes) presents dans
      LevelUp-re/scratchpad_recherche, dump decoupe Ridgeline versionne dans .ai/V7.5/dumps.
- [x] 3.2 OUTIL Go `cmd/mapcallouts-build` (porte de callouts_all.py + callouts.py,
      offsets documentes au champ pres : tag levl, named locations root+0x91C stride 0x28,
      volumes root+0x3BC stride 0xD0, polygone bloc enfant @0x6C, top/bottom @0x64/0x68)
      sur internal/himap (sait deja ouvrir modules et tags). Libelles : jointure par
      string_id sur callouts_i18n.csv EXISTANT (816/816 resolus, copier le CSV en
      reference versionnee) — PAS de re-extraction uslg.
      FAIT (2026-08-13) : lecteur `internal/himap/callouts.go` (navigation struct-table
      de sddt.go ; temoins gamefiles : Horseshoe au champ pres contre le dump, balayage
      22 cartes / 816 zones / liaison nom<->volume verifiee a chaque lecture) + outil
      `cmd/mapcallouts-build` (jointure CSV par carte+volumeIndex avec verification du
      string_id contre le tag, invariants 22/816 bloquants). CSV copie en reference
      versionnee (data/titles/halo_infinite/reference/callouts_i18n.csv). DECOUVERTES
      mesurees en cours de portage : sommets du polygone RELATIFS a pos (cf. Decouvertes)
      + classement grandes/fines par recouvrement etalonne sur le POC (classify_test.go).
- [x] 3.3 Referentiel versionne data/titles/halo_infinite/reference/map_callouts.json
      (22 cartes, 816 zones, polygones complets + FR/EN + top/bottom) +
      PathResolver.MapCalloutsPath + garde-fou de catalogue (patron
      TestCatalogueLivreEstExploitable).
      FAIT (2026-08-13) : catalogue genere (733 Ko, schema_version 1, cle = module
      installe — celle de map_quant_bounds), loader `replay/callouts_catalog.go`
      (verrou de version, ErrCalloutsUnknownMap = cas nominal Forge),
      `TestCatalogueCalloutsLivreEstExploitable` (22 cartes / 816 zones / libelles
      816/816 / tranches ordonnees / ridgeline 28 zones dont 11 grandes = POC).
- [x] 3.4 Service : servir les callouts de la carte du match (modele
      replay_map_background : resolution au service par map-module, PAS de re-cuisson des
      artefacts). Forge = 0 callouts PAR CONSTRUCTION (mesure) → champ absent, propre.
      FAIT (2026-08-13) : `MapCallouts` sur port.ReplayService (sentinelle
      ErrMapCalloutsNotAvailable), `service/replay_map_callouts.go` (match -> noms ->
      module via catalogue de bornes -> entree callouts ; PAS d'essai map_id, voulu : le
      canevas Forge n'a aucune zone, absence propre testee sur Dynasty), endpoint Huma
      GET /matches/{id}/replay/callouts (404 nomme, garde local herite du montage),
      openapi-gen + generate-types DANS LE MEME lot de commit. Oracle reel :
      Cliffhanger -> ridgeline 28 zones / 11 grandes servies.
- [x] 3.5 Web : couche canvas portee du POC (l.1033-1094 : zones fines pointillees sans
      remplissage, 11 grandes zones pair-impair avec parts/holes, libelles 25px MAJUSCULES
      FR blanc cerne noir — pixels d'ECRAN, piege du canevas documente), toggle « Zones »
      (i18n FR/EN), zone courante du joueur sur la fiche (zoneAt 3D — les etages se
      confondent en 2D).
      FAIT (2026-08-13) : `calloutsLayer.ts` (fines pointillees teinte neutre, grandes
      pair-impair avec parts/holes, une couleur de serie par grande zone = la rotation de
      teinte du POC en tokens, libelles 25 px MAJUSCULES blanc cerne noir dedoublonnes
      par nom sur la zone la plus haute — encre structurelle documentee ; libelle dans la
      LANGUE DE L'UI, fr et en servis). Le piege du canevas ne s'applique PAS ici : le
      contexte est transforme au devicePixelRatio, une unite de dessin EST un pixel
      d'ecran (documente au fichier). Calque STATIQUE cuit hors ecran (regle du sol),
      toggle « Zones » FR/EN visible seulement si la carte a des zones, zone courante sur
      la fiche par zoneAt 3D (position interpolee x/y/z de la vie courante), ligne a
      hauteur RESERVEE vivant/mort (regle 1.1). 10 tests calloutsLayer + 3 tests fiches.
- [x] 3.6 Decoupage sur sol praticable : Ridgeline SEULE est decoupee (dump existant,
      versionne) — la conserver ; les 21 autres livrent le polygone brut (ecrit dans le
      sidecar/catalogue).
      FAIT (2026-08-13) : l'outil ingere le dump versionne
      (.ai/V7.5/dumps/callout_zones_ridgeline_clipped.json) pour ridgeline — contour
      decoupe + parties + trous, choix de fiabilite PAR ZONE respecte (21 decoupees,
      7 replis brut declares par le dump), jointure verifiee par libelle. Champ
      `provenance` au catalogue (decoupe/brut). Le decoupage universel est au
      REGISTRE_REPORTS (depend du chantier cartes, en pause user).

Gate 3 : catalogue garde-fou vert ; go test cibles + web verts ; verif visuelle user sur
000d5950 (Cliffhanger/Ridgeline = la reference du POC).
PASSE (2026-08-13) : garde-fou catalogue vert (22 cartes / 816 zones / ridgeline 28 dont
11 grandes) ; go test service + handlers + replay + title + mapcallouts-build OK, himap
par -run ancre (TestCallouts*) OK, go vet ./... OK, golangci-lint cible : 0 issue sur les
fichiers du lot ; web : tsc -b OK (cache .tmp purge), eslint 0 erreur sur les fichiers
touches, vitest COMPLET 411 fichiers / 3657 tests verts (14 skips preexistants) — la
suite complete a attrape le garde-rail titleSlug des query keys (V72-29), la cle
matchReplayCallouts y est classee (commit dedie). La verification visuelle sur 000d5950
est remise au user (instructions dans le compte rendu — la session ne juge pas
l'aspect).

## Lot 4 — Objectifs statiques par mode

- [x] 4.1 Table mode→roles EN DONNEE (TOML mappings du titre, pas en dur) : CTF →
      flag_spawn + flag_delivery ; Strongholds → strongholds_zone ; Oddball →
      oddball_spawn ; Stockpile → stockpile_socket/navpoint ; Extraction →
      extraction_zone ; Assault → assault_bomb. Le SERVEUR choisit les roles servis
      (pair_name normalise cote Go) — le front n'affiche que ce qui arrive (title-agnostic,
      degradation par absence).
      FAIT (2026-08-13) : config/titles/halo_infinite/mappings/objective_roles.toml —
      jetons de mode (mot entier : « CTF » attrape One Flag/Fiesta/Neutral Flag CTF,
      mesure sur les 435 pair_name du registre) + roles mapvar + drapeau `neutral` par
      entree (cf. Decouvertes : 95/158 zones Bastion portent un camp de FICHIER, la
      possession est dynamique et non decodee → Strongholds et Extraction s'affichent
      neutres, le drapeau garde ses camps). Loader strict games/mappings (liste fermee
      mapvar, tout-ou-rien) ; matching au service (analysis.ExtractKnownMode — mappings
      ne peut pas importer analysis, cycle). Variantes bombe (Neutral/One Bomb) → assaut.
- [x] 4.2 Service : MapKeysForMatch(matchID).MapID → LoadMapObjectives → ZonesOfRole
      (lecteur existant objectives_catalog.go, seuls appelants = CLI/tests) → champ servi
      avec le rejeu. map_id vide / carte inconnue = champ absent, jamais d'erreur. MAJ du
      commentaire perime d'objectives_catalog.go (34 cartes → 72) dans le meme commit.
      FAIT (2026-08-13) : port.MatchMapKeys.PairName (meme ligne match_registry, servi
      brut) ; ReplayDocument.MapObjectives (omitempty) rempli A LA REQUETE par GetReplay —
      pair_name → NormalizeModeLabel → specs de roles → jointure map_id SEUL →
      BuildMapObjectives (zones = volumes via Zone.Shape expose, marqueurs = nouveaux
      PointsOfRole — flag_delivery est MIXTE : 36 volumes + 138 points). Pas de bump
      SchemaVersion (champ jamais ecrit dans l'artefact, aucune re-cuisson) ; commentaire
      34→72 corrige (+ la claim « Bastion toutes neutres », fausse sur le catalogue 72) ;
      openapi-gen + generate-types meme commit. Oracle reel Catalyst : 5 marqueurs
      (dont le drapeau central neutre) + 2 cylindres, au champ pres.
- [x] 4.3 Web : couche zones (boites orientees + cylindres projetes, meme transform monde
      que structure/tracks) + marqueurs spawns/livraisons colores par teamIndex (les zones
      Bastion sont neutres team_index -1). INTERDIT d'inventer les lettres A/B/C
      (SpatialRank ≠ lettre du jeu, garde documentee).
      FAIT (2026-08-13) : objectivesLayer.ts — boites aux 4 coins monde par le Forward
      servi, cylindres au rayon monde ; marqueurs en losange, anneau en plus pour une
      livraison ; couleur par teamIndex via le referentiel d'identite du jeu (lib/halo),
      encre neutre pour -1 (Bastion/Extraction arrivent deja neutralises du serveur).
      AUCUN texte — garde TESTEE (ni fillText ni strokeText). Calque statique cuit hors
      ecran (regle du sol), entre les callouts et les projectiles.
- [x] 4.4 Bonus quasi gratuit : pulse sur zone/marqueur au moment d'une ACTION d'objectif
      (doc.objectives deja servi et normalise, rendu nulle part).
      FAIT (2026-08-13) : buildObjectivePulses — l'action est posee sur l'element servi le
      plus proche de son AUTEUR a l'instant T (position relue par posOfPlayerAt, fenetre
      apres-mort du calque des morts ; action sans position ECARTEE, jamais posee au
      hasard) ; anneau qui s'ouvre 1,4 s a la couleur de l'element, statique sous
      mouvement reduit. 13 tests objectivesLayer.

Gate 4 : tests Go service + web verts ; verification sur un match CTF (64e8adfa Catalyst,
5 captures) et un Strongholds ; sur un Slayer : AUCUNE zone.
PASSE (2026-08-13) : go build + go test replay/mapvar/service/mappings/handlers OK,
integration TestReplayMapRepo (pair_name) OK, go vet ./... OK, golangci-lint 0 issue sur
les fichiers du lot ; web tsc -b OK (cache .tmp purge), eslint 0 erreur fichiers touches,
vitest COMPLET 412 fichiers / 3670 tests verts (14 skips preexistants). Artefact
Strongholds construit (696a9d7c, Vagabond, 110 vies / 5 337 frames). Verification API sur
le serveur local (match_id COMPLETS — cf. Decouvertes) : 64e8adfa CTF = 2 zones cylindre
livraison (camps 0/1) + 5 marqueurs (3 spawns dont le central neutre + 2 livraisons) ;
696a9d7c Strongholds = 3 boites TOUTES neutres (la regle `neutral` ecrase les camps de
fichier 0/1/-1 de Vagabond) ; 000d5950 Slayer et 01e1f945 KOTH = champ ABSENT. La
verification VISUELLE est remise au user (instructions au compte rendu — la session ne
juge pas l'aspect).

## Lot 5 — Sons (SOURCE RECUE 13/08 : D:/Halo Infinite Gun Sounds.zip)

- [x] 5.1 Source verifiee (superviseur 13/08) : 76 WAV dans 22 dossiers d'armes + MISC —
      noms ANGLAIS propres (Assault Rifle, Battle Rifle, Bulldog, Cindershot, Commando,
      Disruptor, Energy Sword, Gravity Hammer, Heatwave, Hydra, Mangler, Needler, Plasma
      Pistol, Pulse Carbine, Ravager, S7 Sniper, Sentinel Beam, Shock Rifle, Sidekick,
      Skewer, Spanker, Stalker Rifle), variantes Equip/Reload/Shot/Spray/ADS par arme.
      Rangement dans static/ (nommage par weapon_key hinf_* via la table de noms EN —
      PAS les noms de fichiers FR des images, piege Cremator/Cindershot), variante
      « Shot » prioritaire, troncature ~1 s si besoin, inventorier MISC.
      FAIT (2026-08-14) : 27 fichiers sous `static/sounds/halo_infinite/` — 22 armes +
      `explosion` + 4 `throw_*`. RANGEMENT VERIFIE PAR EMPREINTE (SHA-256 de chaque fichier
      contre l'entree du zip) : chacun porte bien la variante « Shot » de SON arme — les
      tailles identiques Disruptor / Gravity Hammer, qui ressemblaient a une copie fautive,
      sont une coincidence de duree (WAV PCM non compresse), les empreintes different.
      MISC INVENTORIE : `Explosion` + les 4 lancers de grenade exploites ; Active Camo,
      Drop Wall, Grapple(x2), Overshield, Repulsor, Slide, Threat Sensor, Generic ADS,
      pas/sprint laisses de cote — AUCUN de ces gestes n'est date par le document de rejeu
      (pas d'evenement d'equipement ni de deplacement dans le film), un son sans instant ne
      peut pas sonner juste. TRONCATURE FAITE a 1,2 s : 16,9 Mo -> 5,9 Mo, parce que le
      moteur ne joue que la premiere seconde (enveloppe SOUND_CUT_S) et que
      `Dockerfile:183 COPY static /app/static` embarque tout dans l'image (~23 Mo
      aujourd'hui). Les octets gardes sont IDENTIQUES a la source (verifie octet par octet
      contre le zip), la marge de 0,2 s au-dela du `stop` garantit que la coupe entendue
      reste celle de l'enveloppe et jamais une fin de fichier. Le zip source reste intact
      sur D: (reversible). ARMES DU REFERENTIEL SANS SON, silence assume : `hinf_bandit`,
      `hinf_ma5k_avenger`, `hinf_fuel_rod_spnkr`, `hinf_vestige_carbine` (absentes du pack).
- [x] 5.2 Web Audio : son sur les KILLS a minima (weapon_key du feed → fichier), bouton
      mute + volume, coupe par defaut ? (decision au gate), respect prefers-reduced-motion.
      FAIT (2026-08-14) : trois briques etanches, zero logique dans le composant.
      (a) `replaySound.ts` — REGLES PURES : manifeste weapon_key -> stem, piste precalculee
      sur l'horloge DU FIL (`alignFeedToTracks`, la meme qui date le flash des fiches : une
      horloge brute sonnerait a cote de son image), curseur qui ne rejoue rien deux fois
      (recalage silencieux au-dela de `SOUND_RESYNC_JUMP_MS` = 1 s, donc scrub, rebouclage
      et retour d'onglet ne rejouent pas ce qui a ete enjambe), et `soundPlaysAtSpeed`
      (`SOUND_MAX_SPEED` = 2 : a 4x une seconde de son couvre 4 s de match, les echanges se
      recouvrent en continu — le son raconterait le lecteur, pas le match).
      (b) `replayAudio.ts` — LECTURE : AudioContext ne naissant QUE dans le geste
      d'activation (politique d'autoplay), chargement paresseux des seuls sons du match,
      404/indecodable memorise comme ABSENT (silence propre, une seule trace, jamais de
      re-tentative par kill, jamais un son de remplacement), enveloppe de gain (tenue puis
      fondu 0,75 -> 1,0 s : un `stop()` sec claquerait), 8 voix simultanees max, volume
      maitre en rampe de 20 ms (poser `gain.value` au curseur crepiterait).
      (c) `useReplaySound.ts` — CABLAGE : COUPE PAR DEFAUT (decision tenue : rien ne sonne
      NI ne se telecharge avant le clic), preference + volume persistes (localStorage,
      patron du repo), battement appele DANS la boucle rAF et nulle part ailleurs (pause,
      onglet en arriere-plan, redessin de theme = pas de battement = pas de son, par
      construction), premier battement apres activation = recalage silencieux (sinon
      activer a 2 100 ms deverserait tout le debut), coupure immediate au clic (maitre a
      zero, pas d'attente d'une seconde), contexte ferme au demontage. UI :
      `ReplaySoundControls.tsx` dans la barre des calques (patron du bouton Zones : pas de
      commande quand la piste n'a aucun son ; bouton estompe + infobulle quand la vitesse
      le tait), i18n FR+EN (`sound`, `soundHint`, `soundVolume`, `soundFastHint`).
      PREFERS-REDUCED-MOTION : rien n'a ete ajoute qui bouge — les animations du calque
      (tirs, morts, grenades, pulses) continuent de la respecter, et le son etant coupe par
      defaut il n'y a aucune stimulation non demandee. Garde-rail d'assets
      (`replaySoundAssets.guard.test.ts`) : manifeste et dossier sont la MEME liste.
      Tests : 12 (moteur : declenchement, coupure a 1 s, silence si absent, voix, volume),
      10 (regles + curseur), 9 (cablage : rien avant le clic, pas de deversement, avance
      rapide muette, persistance) — double Web Audio partage `test/fakeAudio.ts`.
- [x] 5.3 Sons de grenades si le pack en porte (lancer/explosion, Shock = nappe electrique).
      FAIT (2026-08-14) : les LANCERS sonnent par TYPE (rang -> `throw_frag`/`plasma`/
      `dynamo`/`spike`, l'auteur et l'instant sont ecrits dans le film) et un kill A LA
      grenade sonne `explosion` (c'est elle qui a tue, pas le geste du lancer).
      EXPLOSION EN FIN DE VOL REFUSEE, et c'est le lot 2 qui tranche : `grenadeFx.ts` dit
      qu'AUCUN evenement de detonation n'existe dans le film et que la fin de vol est une
      « derniere position connue » — pour une frag la replication cesse ~1,4 s AVANT la
      meche. Un son d'explosion la-dessus affirmerait une detonation que la donnee ne porte
      pas, au mauvais instant, en contradiction avec ce que l'ecran ecrit deja. La nappe
      electrique Shock/Dynamo reste un effet VISUEL (pas de son de nappe dans le pack).

Gate 5 : gate d'ecoute user. NE PAS rouvrir l'extraction audio du jeu.
PASSE cote automatique (2026-08-14) : tsc -b OK (cache .tmp purge), eslint 0 sur les
fichiers touches, vitest COMPLET 416 fichiers / 3720 tests verts (14 skips preexistants).
Le GATE D'ECOUTE reste au user (instructions au compte rendu — la session ne juge jamais
le rendu sonore).

## Lot 6 — Generation pour tous les matchs + jobs/monitoring local

- [x] 6.1 CLI `levelup backfill-replay` (patron backfill-killsource : 100 % hors ligne,
      tri par cout croissant, --dry-run/--limit/--force, reprise par SchemaVersion) :
      enumere le cache films (951), joint match_registry en RO pour la carte, appelle la
      LIBRAIRIE replay.BuildFromFilm (jamais exec du CLI), rapport par categories —
      construits / deja a jour / carte hors catalogue (COMPTEE A PART, echec voulu) /
      erreurs de decodage. Puis LE RUN DE MASSE (~8 h, artefacts ~2 Go).
      FAIT (2026-08-14) : trois briques. (1) VERROU PROCESS filmdec unique
      (`filmdec.LockProcessDecode`, decode_gate.go) — killsource.decodeMu MIGRE dessus et
      replay.BuildFromFilm l'acquiert : la course compWidthObs killsource/rejeu est
      fermee au niveau du paquet qui possede les globaux. (2) LIBRAIRIE
      `internal/replaybuild` (Builder : catalogue de bornes + labels charges UNE fois,
      cache de structure par module, ArtifactUpToDate = cle de reprise SchemaVersion,
      ErrMapNotInCatalog = echec voulu) — cmd/replay-build refactore dessus (3e
      consommateur = centralisation obligatoire, 0 copie restante). (3) CLI
      `levelup backfill-replay` : enumeration par filmcache.ListShortIDs (disposition
      centralisee), jointure registre en LECTURE COURTE relachee avant tout decodage
      (OpenReadForQuery ; metadata EN best-effort — un serveur qui la tient RW degrade
      sur map_name brut), filtre deja-a-jour AVANT tri/limit (un pilote --limit 25 livre
      25 constructions reelles), rapport 5 categories (+ hors-registre). Tests :
      replaybuild (reprise + resolution sur le catalogue livre), filmcache, killsource,
      replay verts ; vet + golangci-lint cible 0 issue. PILOTE 25 films au gate 6 ; RUN
      DE MASSE = orchestrateur (borne dure de la session).
- [x] 6.2 JobType « replay_build » (domain/job.go) + libelle FR/EN admin + worker
      serialise sur le patron admin_actions (goroutine bgCtx, mutex process filmdec
      PARTAGE avec killsource) → visible dans /admin/monitoring/jobs sans travail UI.
      Job manuel unitaire (semantique jobs.json suffit en local ; la file durable = piste F
      prod, registre).
      FAIT (2026-08-14) : JobTypeReplayBuild + POST /admin/actions/replay-build/run
      {match_id} (patron convergence : body decode maison 400, 409 avec job_id actif par
      match, goroutine bgCtx + recover, 202 AsyncJobStatus) ; runner
      ServiceRegistry.RunReplayBuild (registry_replay_build.go) : single-flight
      replayBuildMu, handles base relaches AVANT le decodage (dataQualityHandles),
      forme courte ACCEPTEE (resolution LIKE + refus des prefixes ambigus — piege lot 4),
      candidats de carte = asset EN puis map_name brut (ordre ReplayMapRepo), LIBRAIRIE
      replaybuild (jamais exec CLI, verrou filmdec partage). Libelle FR/EN
      statusDisplay.ts (« Construction du rejeu 2D » / « 2D replay build ») + test.
      openapi-gen + generate-types + typecheck purge OK dans le meme commit. 4 tests
      handler (503/400/202/409) verts.
- [x] 6.3 FIL DE L'EAU post-sync : etape apres runWeaponKills (convergence.go:471-604,
      heritee par V2) + LE PONT DISQUE : writer de chunks DANS le paquet filmcache
      (garde-rail de disposition existant — bonus : persiste les films qui expirent,
      archive irremplacable) ; gate fenetre `replay_retention_months` dans AppSettings
      (+ PATCH + UI admin, patron scheduler qui relit a chaque tick) ; conditionne LOCAL
      (« le VPS web ne decode JAMAIS » — la garde replay_local_gate existe).
      FAIT (2026-08-14) : (a) PONT DISQUE `filmcache.Write` (chunks d'abord, manifeste
      EN DERNIER = marqueur de commit ; manifeste historique JAMAIS reecrit — il porte le
      blob_prefix CDN ; chunks immuables jamais reecrits ; 3 tests). (b) Etape 1.58
      `runReplayArtifacts` (engine_postsync_replay.go) apres runWeaponKills — heritee par
      V2 : GetFilmChunks (assertion OPTIONNELLE sur le client, les mocks ne bougent pas),
      persistance cache, construction replaybuild, borne 5 matchs/cycle (solde au CLI),
      fenetre par SQLStartTimeCanonical, compteurs expvar. (c)
      `replay_retention_months` (0 = illimite, defaut) : AppSettings + Apply/ToResponse
      + validation PATCH >= 0 + types.ts + carte « Rejeu 2D » AnalyseTab (i18n FR+EN) —
      le GET /settings est un body non type, openapi.yaml inchange (verifie). (d)
      CONDITIONNE LOCAL aux 3 sites de wiring (BuildEngine scheduler via
      wireReplayArtifacts, factory V2, handler HTTP) : IsProduction() -> hook absent ;
      fabrique unique NewReplayArtifactsHook. Gates : go test sync/scheduler/handlers/
      settings/filmcache verts, integration -p 1 -run ancre OK, tsc -b (purge) OK,
      eslint 0, vitest settings 69 verts, lint 0 issue sur les fichiers du lot.
- [x] 6.4 PURGE recurrente des artefacts hors fenetre (patron *_cron.go + ReportCronRun,
      visible sur /admin/monitoring/crons) — supprime des ARTEFACTS, JAMAIS des films.
      FAIT (2026-08-14) : scheduler/replay_purge_cron.go (patron world_leaderboard :
      RunOnce + ReportCronRun "replay_purge", tick quotidien, fenetre relue a CHAQUE
      tick ; 0 = illimite -> no-op rapporte en SUCCES). L'age d'un artefact est celui de
      SON MATCH (SQLStartTimeCanonical du registre, jamais le mtime — re-cuire ne
      rajeunit pas) ; un artefact SANS ligne de registre n'est JAMAIS detruit
      (indatable) ; seuls les {short8}.json du dossier d'artefacts sont touches —
      film_chunks/film_manifests jamais. Lecture par OpenReadForQuery (mono-process
      safe). PathResolver.ReplayArtifactsDir ajoute (ReplayArtifactPath derive).
      Cable dans cmd/server (settingsStore -> retention). 2 tests (purge selective
      vieux/recent/indatable sur vraie DuckDB temporaire ; dossier absent nominal).
- [!] 6.5 DATA RUN : rejouer backfill-killsource (couverture arme-du-kill 0-5 % sur les
      matchs recents — les effets de mort par famille et le kill feed en dependent).
      PILOTE TENTE ET BLOQUE (2026-08-14) : `backfill-killsource --limit 5 --films-only`
      echoue au premier pas sur le lock DuckDB — la commande exige OpenReadWrite sur
      shared (SERVEUR ARRETE, contrat documente en tete du fichier), et le serveur air
      du user tient le fichier (server.exe~ PID 27448) avec INTERDICTION de session de
      le redemarrer. L'erreur est propre et actionnable (« serveur arrete ? »). Ligne au
      REGISTRE_REPORTS : orchestrateur, pilote --limit PUIS run complet dans une fenetre
      serveur arrete (accord user).

Gate 6 : tests Go (dont integration si sync/ touche) ; backfill pilote 25 films AVANT le
run complet (taux d'echec ≤ 20 % hors carte-hors-catalogue) ; couverture finale publiee
(artefacts / films / matchs par categorie) ; jobs visibles dans l'admin.
PASSE (2026-08-14, hors 6.5 bloque ci-dessus) : tests Go verts par lot (sync 51 s,
scheduler, handlers, settings, filmcache, replaybuild, replay, killsource, filmdec par
-run ancre) + integration -p 1 -run ancre sur sync/ OK. PILOTE 25 FILMS (14 min 51 s,
un a la fois, avant-plan) : 19 construits / 2 deja a jour / 6 carte hors catalogue
(echec VOULU : Outlook, Live Fire, Disciple, Merchant's Square... — cartes communautaires
sans bornes) / 0 hors registre / 0 ERREUR DE DECODAGE → taux d'echec hors
carte-hors-catalogue = 0 % ≤ 20 %. COUVERTURE au moment de la cloture : 951 films en
cache, 21 artefacts a jour (schema 3), 2 artefacts v2 a re-cuire, 928 restants — le run
de masse (~8 h) revient a l'orchestrateur (registre ; de preference SERVEUR ARRETE pour
la resolution EN metadata). Jobs admin : JobType replay_build + libelle FR/EN livres,
route montee (tests 202/409/400/503) — verification VISUELLE /admin/monitoring/jobs
remise au user. Verification visuelle user (gate) : 1) ouvrir un match du pilote (ex.
e94163af CTF Bazaar ou 606d9844 Chasm) → l'onglet Rejeu 2D doit servir le nouvel
artefact ; 2) Reglages → Analyse → carte « Rejeu 2D » : champ « Fenetre de conservation
des rejeux » (0 par defaut) ; 3) /admin/monitoring/crons : ligne replay_purge en succes.

## Lot 7 — Deux directives utilisateur du 2026-08-14 (verifiees sur pieces)

> Elles REMPLACENT deux choix faits en cours de route par des donnees qui existaient deja.
> A executer APRES le lot 5, AVANT tout run de masse (le 7.2 change le contrat d'artefact :
> generer 928 artefacts avant serait a re-cuire).

- [ ] 7.1 MORTS NEUTRES : la ligne grise « mort » doit porter L'ICONE DU TYPE DE MORT, pas un
      repere generique. Les icones EXISTENT deja, extraites et versionnees
      (`static/weapons-assets/halo_infinite/jeu/index.json`) : `suicide` (killfeed-61),
      `splatter` (60, ecrasement), `environment` (55), `fusion_coil` (27), `waterfall` (78),
      `ricochet` (57), `player_left` (69). La donnee qui distingue = la NATURE du degat fatal
      de killsource (« arme / melee / grenade / vehicule / objet explosif / environnement » —
      GUIDE_KILLSOURCE §194 : elle vient de la structure des archives du jeu, pas d'une
      heuristique, et « reste juste meme quand le nom sort en Autres »). Regle : nature ->
      icone, une nature inconnue reste le repere neutre (jamais l'icone d'une autre mort).
      MESURER d'abord la couverture des morts sans tueur par la nature (le guide note qu'elles
      sont RARES : 0 sur 4 films de reference, 1 suicide sur le BTB) — si la donnee manque sur
      la majorite, le dire et livrer ce qui est mesure.
- [ ] 7.2 SYNCHRO PAR LE T0 REEL, pas par appariement statistique. Directive utilisateur :
      « pas besoin d'algo alambique, on a le first joined time en data API qui donne le debut
      reel du match sans la phase d'attente/chargement ». VERIFIE : ce T0 EXISTE et est deja
      persiste — `match_registry.real_start_time` + `t0_quality`, calcule par
      `analysis/timeline.ComputeT0` (`cmd/backfill_t0` : T0 = MIN(first_joined_time des joueurs
      present_at_beginning, hors bots) − start_time_utc) ; MatchView s'en sert deja
      (`meta.T0Ms` -> `correctMatchViewEventsT0`). LE DEFAUT EST DANS L'ARTEFACT : `build.go`
      cale sa frame 0 sur le PREMIER PAQUET DE POSITION (`origin = sorted[0].TimestampUS`),
      un referentiel qui n'existe nulle part ailleurs — d'ou l'ecart mesure de 3,7 s a 40 s
      selon le match. CORRECTIF : l'artefact PUBLIE son origine (timestamp absolu du premier
      paquet, ou son decalage au `start_time_utc` du match) ; le client calcule un decalage
      EXACT (origine artefact − T0) au lieu d'apparier des morts. Bump SchemaVersion +
      re-cuisson des ~23 artefacts existants. GAIN par rapport au contournement livre en
      `edb0e723c` : marche aussi sur les matchs SANS couverture killer_victim_pairs (limite
      documentee du correctif actuel), et ne depend plus d'un appariement a +-2 s.
      PIEGE CONNU : dette TZ sur `first_joined_time` (cf. `cmd/backfill_first_joined_tz`,
      memoire `project_data_quality_first_joined_tz`) — lire `t0_quality` et degrader
      proprement quand il vaut autre chose que la valeur nominale, jamais servir un T0 douteux.
      L'appariement de `edb0e723c` reste comme REPLI quand le T0 est absent/rejete.

## Hors perimetre (registre)

Ouvrier distant + file durable + heartbeat (piste F, activation prod post-tag) ; decoupage
callouts universel (depend cartes) ; palette POC des effets (si le user la redemande) ;
etat VIVANT des objectifs (qui porte le drapeau — ti=11, reverse supplementaire) ; heatmap
(cahier item 13) ; sons hors kills.

## Decouvertes

- (lot 1, 2026-08-13) `weaponInHand` (i18n match-replay) etait deja orphelin AVANT le
  lot — aucune reference hors i18n.ts. Retire avec les 8 cles de l'item 1.2 (0 code mort).
- (lot 1, 2026-08-13) i54 « biped-mobility-action » est un SIGNAL D'EVENEMENT date
  exploitable (67 episodes discrets ~0,6 s sur 000d5950, en cours de vie — pas du bruit de
  mouvement, pas du spawn), mesure par l'instrument versionne
  `internal/analysis/filmdec/i54_research_test.go`. Non identifiable comme usage
  d'equipement aujourd'hui (item 1.4 [!]) ; la piste de reprise la plus courte est le
  croisement i56 (chute d'energie de capacite) — cf. REGISTRE_REPORTS.
- (lot 1, 2026-08-13) la reservation de hauteur des fiches (1.1) est en min-height pour
  les zones armes/inventaire : une rangee VIVANTE qui passerait sur deux lignes (colonne
  tres etroite, flex-wrap) depasserait la reserve. Cas non observe aux largeurs de la
  page ; a surveiller au gate visuel.
- (lot 2, 2026-08-13) la MELEE GENERIQUE (coups sans arme nommee) n'a PAS de weapon_key
  (regle killicon CLASSE MELEE, mesure) : son effet de mort tombe sur le rendu neutre, pas
  sur l'arc de melee. Le POC la rangeait par la CLASSE (e.w.cl), que le feed ne sert pas.
  Piste si le gate visuel le reclame : servir la classe damagetag sur l'event kill (modif
  Go match_view) — hors perimetre du lot, note au registre si demande.
- (lot 2, 2026-08-13) cmd/replay-build : la cle `--map` est le NOM de carte du catalogue
  (`cliffhanger`), PAS le module (`ridgeline`) — l'instruction d'orchestration disait
  ridgeline, l'erreur du catalogue est explicite et la correction evidente.
- (lot 2, 2026-08-13) le badge .gic ne peut pas vivre DANS le bloc estompe par l'age de
  lecture : les opacites CSS se multiplient. Il vit dans un bloc frere, hors estompage.
- (lot 2, 2026-08-13) `projectile-at-rest-state` est PROPRE AU TYPE : sur le temoin, 15/21
  vols Spike le portent, 0/44 frag+plasma+Dynamo — coherent avec un objet qui detone (il
  disparait, il ne s'immobilise pas). Le « 78/79 » du POC comptait les vols TERMINES tous
  types projectiles confondus, pas les grenades liees. Toute logique future qui exigerait
  at-rest sur une grenade non-Spike s'eteindra en silence.

- (lot 6, 2026-08-14) killsource serialisait son decodage par un verrou LOCAL de paquet
  (decodeMu) : deux verrous locaux (killsource + rejeu) ne se protegent pas l'un de
  l'autre — la serialisation est remontee au paquet qui POSSEDE les globaux
  (filmdec.LockProcessDecode), et les deux consommateurs l'empruntent.
- (lot 6, 2026-08-14) GET /settings sert un body NON TYPE (settingsJSONOutput{Body any})
  : un champ ajoute a SettingsResponse ne change PAS openapi.yaml — le contrat web vit
  dans lib/api/types.ts (manuel), a mettre a jour a la main. Verifie sur pieces
  (openapi-gen re-joue, 0 diff).
- (lot 6, 2026-08-14) un serveur qui tourne tient metadata.duckdb RW : la resolution EN
  des noms de carte du backfill-replay DEGRADE alors sur map_name brut (les map_name
  UUID tombent en « hors catalogue » au lieu de se resoudre). Le run de masse gagne a
  tourner SERVEUR ARRETE — note au REGISTRE_REPORTS avec le run.
- (lot 3, 2026-08-13) les sommets du polygone d'un volume levl sont RELATIFS a pos, et
  c'est mesure deux fois : pos+rel reproduit a 0,0000 m les polygones monde du dump de
  Ridgeline (16 zones dessinees), et l'AABB @0x94 du record ([minX maxX minY maxY minZ
  maxZ]) est la boite des sommets RELATIFS sur toutes les cartes — donc pas de rotation
  supplementaire. L'invariant est verifie A CHAQUE extraction (verifieAABBRelative) : un
  record qui ne le porte plus est refuse, pas translate au petit bonheur.
- (lot 3, 2026-08-13) les volumes SANS forme propre portent un polygone par defaut
  (sommets ±0,5, AABB nulle) : ce n'est pas une forme dessinee — le lecteur ne publie le
  polygone que sur has_shape. Ils restent au catalogue SANS contour : ce sont les etages
  secondaires d'un meme nom, et ils rendent zoneAt juste (l'etage bas de « Tuyaux » vit
  dans un volume secondaire).
- (lot 3, 2026-08-13) le classement grandes/fines du POC se DERIVE du recouvrement 2D
  (raster 0,25 m) : sur Ridgeline les 11 grandes sont recouvertes a 0,00 par les autres
  (un pavage ne se recouvre pas), les 5 fines a 0,56-1,00. Seuil de majorite (0,5),
  etalonnage rejoue en continu par classify_test.go contre le classement du POC.
- (lot 3, 2026-08-13) le piege « libelles en pixels d'ecran » du POC ne s'applique pas au
  canvas du rejeu : le POC dessinait dans un canevas de 1600 px affiche a ~840 px (d'ou
  son facteur K) ; notre contexte est transforme au devicePixelRatio et le canvas
  s'affiche a sa taille CSS — une unite de dessin EST un pixel d'ecran, 25 px se dessine
  25, sans K.

- (lot 4, 2026-08-13) le catalogue d'objectifs 72 cartes CONTREDIT deux claims du
  lecteur : 95/158 zones de Bastion portent un team_index 0/1 (Vagabond incl. — la
  version 34 cartes n'en avait aucune), et flag_delivery est un role MIXTE (36 volumes +
  138 points ; Catalyst porte les deux au meme endroit). La possession d'une zone etant
  dynamique et non decodee, la regle d'affichage (neutre ou pas) est portee par la
  DONNEE (`neutral` dans objective_roles.toml), jamais par le role.
- (lot 4, 2026-08-13) PIEGE DE VERIFICATION API : match_registry est indexe par match_id
  COMPLET ; seule ReplayArtifactPath normalise la forme courte (FilmShortMatchID). Une
  verif curl a l'id COURT rend le document SANS fond/callouts/objectifs (MapKeysForMatch
  ne trouve rien, degradation silencieuse en debug) — verifier toujours avec l'UUID
  complet, celui des routes de l'app.
- (lot 4, 2026-08-13) un server.exe ZOMBIE (demarre 22:17, air ne l'avait pas tue) tenait
  metadata.duckdb : le fils reconstruit par air mourait a l'ouverture (« utilise par un
  autre processus ») et :8000 servait un binaire d'avant le lot. Remede documente
  (memoire metadata-fatal) applique : kill du doublon, air a relance un fils sain.
- (lot 4, 2026-08-13) Total Control (110 matchs du registre) et Land Grab sont des modes
  a zones NON couverts par la table : leurs roles d'objets ne sont pas etablis (aucune
  mesure .mvar — utilisent-ils strongholds_zone ?). Le jour ou c'est mesure, l'extension
  est UNE entree TOML, zero code. KOTH reste hors v7.5 (registre).
- (lot 5, 2026-08-14) DEUX FICHIERS DU PACK ONT LA MEME TAILLE SANS ETRE LE MEME SON :
  Disruptor Shot et Gravity Hammer Hit font 857 942 octets a l'octet pres (WAV PCM non
  compresse, memes duree et format), de meme que MISC/Explosion et MISC/Springing on metal.
  Un controle de rangement par taille conclut donc a une copie fautive la ou il n'y en a
  pas — c'est l'empreinte qui tranche. A garder en tete pour tout futur pack d'assets.
- (lot 5, 2026-08-14) le test « saut long en avant » ecrit avec le curseur sonore sautait a
  `SOUND_RESYNC_JUMP_MS + 1` = 1 001 ms et attendait 2 evenements enjambes alors que le
  second est a 1 200 ms : il n'avait jamais tourne. Corrige (saut a 1 500 ms) et complete
  par le cas symetrique (un saut JUSTE sous le seuil reste une lecture continue).

## Reprise

Avancement = statuts de ce fichier + git log feat/v75. Ordre des lots : 1 → 2 → 3 → 4 → 6,
le 5 des reception des sons. Gates visuels/d'ecoute = utilisateur, en une passe par lot.
Lot 5 CLOS cote code le 2026-08-14 (5.1/5.2/5.3 [x]) : il ne reste que le gate d'ecoute.
