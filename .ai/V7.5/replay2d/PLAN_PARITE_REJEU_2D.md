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

- [ ] 1.1 HAUTEUR CONSTANTE vivant/mort : reserver la place (min-height ou rangees
      fantomes) — aujourd'hui ReplayWeaponsRow/InventoryRow rendent null a la mort et la
      fiche se compacte (ReplayTeams.tsx l.140-209, rosterLogic l.239-246).
- [ ] 1.2 TOOLTIPS dev supprimes : ammoSlotHint, abilityUnknownHint, respawnUnknownHint,
      weaponSwapHint, grenadeSelUnknownHint, ammoNoneHint, drawnUnknownHint,
      weaponInHandHint (+ les title= porteurs). CONSERVES : gamertag, Bouclier/Sante,
      aria-labels a11y (barre respawn, jauge charge).
- [ ] 1.3 « rangees »/« aucune » remplaces selon la decision 4 (i18n FR+EN).
- [ ] 1.4 INVESTIGATION bornee equipement actif : les events MobilityAction*
      (filmdec/components_biped_ability.go) donnent-ils un USAGE date de capacite,
      offline-pur, sur 000d5950 ? Si oui : effet plein-fiche a l'usage (duree fixe courte),
      les 3 rendus de la decision 3. Si non : ligne registre RE, item [!] justifie.

Gate 1 : typecheck purge + lint + vitest verts ; capture des fiches avant/apres pour le
gate visuel user (mort ET vivant cote a cote, hauteur identique).

## Lot 2 — Effets de mort, grenades, lancers

- [ ] 2.1 EFFET DE MORT par famille d'arme (portage drawKillFx du POC, l.2884 de
      l'artefact) : oriente tueur→victime quand le couple est complet (regle POC 89/93),
      marqueur non oriente sinon ; extremite reelle (cible=true) ; melee = arc SANS eclair
      (seuil 8 m mesure) ; formes = shotEffects existant, duree KFX_HOLD 1,4 s. Donnees :
      kills du feed (killer/victim/weapon_key, horloge recalee) + positions via doc.tracks
      (fenetre death, patron posOfNameAt). Manque Go : la famille fx par weapon_key sur les
      events du feed (aujourd'hui [shot_effects] n'est joignable que par id d'arme film) —
      petite modif (servir fx avec weapon_key, ou table cote client depuis weaponLabels).
- [ ] 2.2 TIRS : l'effet par famille sur TOUS les tirs publies EXISTE (drawShotsLayer, 475
      tirs) — verifier pourquoi il est invisible a l'oeil (durees/intensites vs POC :
      SHOT_HOLD 0,6 s, longueurs) et recaler sur le POC. Pas de re-conception.
- [ ] 2.3 GRENADES : publier le lien grenade↔projectile dans l'artefact (l'appariement
      existe deja dans grenades.go, fenetre 200 ms) + effet par TYPE au point Rest :
      Shock/Dynamo (rang 2) = effet ELECTRIQUE persistant ~2-3 s (geometrie drawShock),
      autres = halo « derniere position connue » — JAMAIS « impact » (aucun event de
      detonation dans le film, seul Rest certifie 78/79 ; frag : la replication cesse
      ~1,4 s avant la meche).
- [ ] 2.4 LANCERS mis en evidence : type distingue sur la carte (vignette
      GrenadeLabels[rank].img deja publiee, ou forme par rang) + pulse ephemere sur la
      fiche au lancer (badge .gic du POC, l.534-543).

Gate 2 : tests web verts ; si modif du contrat artefact : bump SchemaVersion + openapi-gen
+ generate-types meme commit ; re-cuisson du temoin 000d5950 + verification visuelle a
remettre au user (liste des instants a regarder : un kill balistique, un plasma, une melee,
un lancer Shock).

## Lot 3 — Callouts (regression POC, tout existe hors depot)

- [ ] 3.1 VERIFIER LA MATIERE PREMIERE d'abord : le jeu n'est PLUS installe. Sources =
      C:/Users/Guillaume/Projects/LevelUp-re/jeu_deploy_ds/ (modules cartes) +
      jeu_deploy_any/ (modules uslg) + clef PNY en secours. Inventorier les 31 modules
      AVANT de coder ; s'il en manque → le dire, pas contourner.
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

## Lot 5 — Sons (demarre a reception des fichiers sons user)

- [ ] 5.1 Reception + rangement des sons (static/, nommage par famille fx ou weapon_key),
      troncature ~1 s si besoin.
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

(a remplir en cours d'execution)

## Reprise

Avancement = statuts de ce fichier + git log feat/v75. Ordre des lots : 1 → 2 → 3 → 4 → 6,
le 5 des reception des sons. Gates visuels/d'ecoute = utilisateur, en une passe par lot.
