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
2. **Sons** : BLOQUE sur la source audio — le zip Downloads est ABSENT, l'audio du jeu est
   non-extractible (verdict double-mesure du pont sonore, chantier parque, « n'y revenez
   pas sans element nouveau »). Le lot 5 ne demarre qu'a reception de fichiers sons
   fournis par l'utilisateur.
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

- [ ] 3.1 MATIERE PREMIERE — CORRIGE 13/08 (user) : le jeu EST installe sur
      C:/Program Files (x86)/Steam/steamapps/common/Halo Infinite (verifie : deploy/ds/
      levels/multi = 31 dossiers, deploy/any present). `himap.LevelsDir`/deploy_root.go
      resout deja cette racine (preuve : lot bornes). Source PRIMAIRE = l'installation ;
      les copies LevelUp-re/jeu_deploy_* et la clef PNY ne sont que des secours. L'agent
      d'inventaire avait teste l'ancien chemin D:/SteamLibrary — conclusion invalidee.
- [ ] 3.2 OUTIL Go `cmd/mapcallouts-build` (porte de callouts_all.py + callouts.py,
      offsets documentes au champ pres : tag levl, named locations root+0x91C stride 0x28,
      volumes root+0x3BC stride 0xD0, polygone bloc enfant @0x6C, top/bottom @0x64/0x68)
      sur internal/himap (sait deja ouvrir modules et tags). Libelles : jointure par
      string_id sur callouts_i18n.csv EXISTANT (816/816 resolus, copier le CSV en
      reference versionnee) — PAS de re-extraction uslg.
- [ ] 3.3 Referentiel versionne data/titles/halo_infinite/reference/map_callouts.json
      (22 cartes, 816 zones, polygones complets + FR/EN + top/bottom) +
      PathResolver.MapCalloutsPath + garde-fou de catalogue (patron
      TestCatalogueLivreEstExploitable).
- [ ] 3.4 Service : servir les callouts de la carte du match (modele
      replay_map_background : resolution au service par map-module, PAS de re-cuisson des
      artefacts). Forge = 0 callouts PAR CONSTRUCTION (mesure) → champ absent, propre.
- [ ] 3.5 Web : couche canvas portee du POC (l.1033-1094 : zones fines pointillees sans
      remplissage, 11 grandes zones pair-impair avec parts/holes, libelles 25px MAJUSCULES
      FR blanc cerne noir — pixels d'ECRAN, piege du canevas documente), toggle « Zones »
      (i18n FR/EN), zone courante du joueur sur la fiche (zoneAt 3D — les etages se
      confondent en 2D).
- [ ] 3.6 Decoupage sur sol praticable : Ridgeline SEULE est decoupee (dump existant,
      versionne) — la conserver ; les 21 autres livrent le polygone designer BRUT, et le
      decoupage universel va au registre (depend du chantier cartes, en pause user).

Gate 3 : catalogue garde-fou vert ; go test cibles + web verts ; verif visuelle user sur
000d5950 (Cliffhanger/Ridgeline = la reference du POC).

## Lot 4 — Objectifs statiques par mode

- [ ] 4.1 Table mode→roles EN DONNEE (TOML mappings du titre, pas en dur) : CTF →
      flag_spawn + flag_delivery ; Strongholds → strongholds_zone ; Oddball →
      oddball_spawn ; Stockpile → stockpile_socket/navpoint ; Extraction →
      extraction_zone ; Assault → assault_bomb. Le SERVEUR choisit les roles servis
      (pair_name normalise cote Go) — le front n'affiche que ce qui arrive (title-agnostic,
      degradation par absence).
- [ ] 4.2 Service : MapKeysForMatch(matchID).MapID → LoadMapObjectives → ZonesOfRole
      (lecteur existant objectives_catalog.go, seuls appelants = CLI/tests) → champ servi
      avec le rejeu. map_id vide / carte inconnue = champ absent, jamais d'erreur. MAJ du
      commentaire perime d'objectives_catalog.go (34 cartes → 72) dans le meme commit.
- [ ] 4.3 Web : couche zones (boites orientees + cylindres projetes, meme transform monde
      que structure/tracks) + marqueurs spawns/livraisons colores par teamIndex (les zones
      Bastion sont neutres team_index -1). INTERDIT d'inventer les lettres A/B/C
      (SpatialRank ≠ lettre du jeu, garde documentee).
- [ ] 4.4 Bonus quasi gratuit : pulse sur zone/marqueur au moment d'une ACTION d'objectif
      (doc.objectives deja servi et normalise, rendu nulle part).

Gate 4 : tests Go service + web verts ; verification sur un match CTF (64e8adfa Catalyst,
5 captures) et un Strongholds ; sur un Slayer : AUCUNE zone.

## Lot 5 — Sons (SOURCE RECUE 13/08 : D:/Halo Infinite Gun Sounds.zip)

- [ ] 5.1 Source verifiee (superviseur 13/08) : 76 WAV dans 22 dossiers d'armes + MISC —
      noms ANGLAIS propres (Assault Rifle, Battle Rifle, Bulldog, Cindershot, Commando,
      Disruptor, Energy Sword, Gravity Hammer, Heatwave, Hydra, Mangler, Needler, Plasma
      Pistol, Pulse Carbine, Ravager, S7 Sniper, Sentinel Beam, Shock Rifle, Sidekick,
      Skewer, Spanker, Stalker Rifle), variantes Equip/Reload/Shot/Spray/ADS par arme.
      Rangement dans static/ (nommage par weapon_key hinf_* via la table de noms EN —
      PAS les noms de fichiers FR des images, piege Cremator/Cindershot), variante
      « Shot » prioritaire, troncature ~1 s si besoin, inventorier MISC.
- [ ] 5.2 Web Audio : son sur les KILLS a minima (weapon_key du feed → fichier), bouton
      mute + volume, coupe par defaut ? (decision au gate), respect prefers-reduced-motion.
- [ ] 5.3 Sons de grenades si le pack en porte (lancer/explosion, Shock = nappe electrique).

Gate 5 : gate d'ecoute user. NE PAS rouvrir l'extraction audio du jeu.

## Lot 6 — Generation pour tous les matchs + jobs/monitoring local

- [ ] 6.1 CLI `levelup backfill-replay` (patron backfill-killsource : 100 % hors ligne,
      tri par cout croissant, --dry-run/--limit/--force, reprise par SchemaVersion) :
      enumere le cache films (951), joint match_registry en RO pour la carte, appelle la
      LIBRAIRIE replay.BuildFromFilm (jamais exec du CLI), rapport par categories —
      construits / deja a jour / carte hors catalogue (COMPTEE A PART, echec voulu) /
      erreurs de decodage. Puis LE RUN DE MASSE (~8 h, artefacts ~2 Go).
- [ ] 6.2 JobType « replay_build » (domain/job.go) + libelle FR/EN admin + worker
      serialise sur le patron admin_actions (goroutine bgCtx, mutex process filmdec
      PARTAGE avec killsource) → visible dans /admin/monitoring/jobs sans travail UI.
      Job manuel unitaire (semantique jobs.json suffit en local ; la file durable = piste F
      prod, registre).
- [ ] 6.3 FIL DE L'EAU post-sync : etape apres runWeaponKills (convergence.go:471-604,
      heritee par V2) + LE PONT DISQUE : writer de chunks DANS le paquet filmcache
      (garde-rail de disposition existant — bonus : persiste les films qui expirent,
      archive irremplacable) ; gate fenetre `replay_retention_months` dans AppSettings
      (+ PATCH + UI admin, patron scheduler qui relit a chaque tick) ; conditionne LOCAL
      (« le VPS web ne decode JAMAIS » — la garde replay_local_gate existe).
- [ ] 6.4 PURGE recurrente des artefacts hors fenetre (patron *_cron.go + ReportCronRun,
      visible sur /admin/monitoring/crons) — supprime des ARTEFACTS, JAMAIS des films.
- [ ] 6.5 DATA RUN : rejouer backfill-killsource (couverture arme-du-kill 0-5 % sur les
      matchs recents — les effets de mort par famille et le kill feed en dependent).

Gate 6 : tests Go (dont integration si sync/ touche) ; backfill pilote 25 films AVANT le
run complet (taux d'echec ≤ 20 % hors carte-hors-catalogue) ; couverture finale publiee
(artefacts / films / matchs par categorie) ; jobs visibles dans l'admin.

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

## Reprise

Avancement = statuts de ce fichier + git log feat/v75. Ordre des lots : 1 → 2 → 3 → 4 → 6,
le 5 des reception des sons. Gates visuels/d'ecoute = utilisateur, en une passe par lot.
