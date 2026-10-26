# Audit decodeur de film + generateur d'artefacts de rejeu — 2026-09-24

Skill `adversarial-audit`. Demande utilisateur (conversationnelle) : « ce que tu penses de mon
decodeur de film et generateur d'artefact de replay — code, archi, conventions modernes ; les
erreurs sont du bonus ». L'audit ne corrige rien : il produit ce registre. Toute correction est
un chantier ulterieur (`plan-review`, `plan-execution`, `adversarial-review`).

## Cadrage

- **Arbre audite** : `9d335ea43` (feat/v75), fige dans un worktree detache par relecteur
  (`LevelUp-wt-revue-*`). Chaque constat retenu a ete re-verifie sur `feat/retours-rejeu`
  (campagne en vol, 66+ commits, +11 000 lignes dans ce perimetre) : colonne « rr ».
- **Perimetre** : `apps/go-api/internal/games/halo_infinite/film/**` (hors `research/`),
  `internal/replaybuild`, `internal/filmproc`, `internal/sync/killcollector` ; ~119 000 lignes Go
  hors tests dont ~55 000 de commentaires (46 %), ~270 000 lignes de tests.
- **Axes passes** :
  - CORRECTION (bugs), 11 relecteurs en fan-out aveugle par sous-perimetre : SRC (source,
    profile, revision, types, filmcache, decfilm, catalogues), GRAM-A1 (lecteurs, registre,
    trames, images-cles), GRAM-A2 (composants, aiguillages, balayages hors ligne), GRAM-B
    (scanners de domaine + positions/weaponscan/weaponv3), FACTS-K (killsource), FACTS-O
    (objectives + fallback), REP-A1 (assemblage, codec des faits, couverture), REP-A2 (identites,
    vies, morts, synthese d'usage, catalogues), REP-B1 (calques d'objectifs + mapvar), REP-B2
    (vehicules, equipement, armes au sol, inventaire), OPS (orchestrateurs). Couvre de fait A6
    (erreurs avalees) et A2 (invariants ART de `killcollector`).
  - ARCHITECTURE (1 relecteur) et CONVENTIONS GO MODERNES / QUALITE (1 relecteur) : axes ajoutes
    pour la demande ; leur recevabilite exigeait preuve mesuree + source citee + consequence.
- **Axes NON passes** : A3 multi-titre (le decodeur est Halo Infinite par nature, ADR 0012),
  A10 front (hors perimetre), A9 correction des KPI (hors perimetre).
- **Doctrine de reference** : CLAUDE.md (regles 3, 5, 6, 7, 11, anti-patterns, regles DuckDB),
  ADR 0034 (D-1 a D-10, etats M2-M4), regles produit (places finies, un partant libere sa place,
  aucune vie anonyme, manches sans mi-temps, « absent n'est pas zero »).
- **Verification adverse** : chaque lot de constats est passe devant un refutateur frais (« en
  cas de doute, refute »). Le superviseur a en plus verifie sur pieces : `solo.go`, l'unite de
  `vehicle_rides_seat.go:55`, `zones.go:224`, l'absence de l'etape 1.57 sur `main`, les comptes
  `CompteurBranche`, l'import de `film/replay` par 26 paquets, `-race` en CI, le chemin `.ai/`
  cite par CLAUDE.md.
- **Contraintes respectees** : aucune commande `go` par les relecteurs, aucun decodage de film,
  aucune lecture sous `data/`, aucune ecriture dans les worktrees (un fichier cree par erreur par
  le relecteur architecture a ete supprime aussitot, `git status` vide).

## Constats retenus

Gravite : P0 = donnee fausse servie / corruption / crash en production ; P1 = bug reel a portee
limitee ou violation d'une regle ecrite avec consequence ; P2 = defaut latent. Chemins relatifs a
`apps/go-api/internal/`, `F/` = `games/halo_infinite/film/`.

### [P0] GB-1 — Les positions et les canaux delta bipede ne lisent que la GENERATION 1 du handle
- Ou : `F/internal/grammar/offline_biped.go:170` (`RequireTag1: true` par defaut) et `:295` ;
  `F/internal/grammar/delta_biped_walk.go:108` (`true` en dur) ; copie `equipment_recovery.go:234` ;
  appelants `F/replay/build_from_film.go:186-188` (`DefaultScanFilmOptions`),
  `sync/killcollector/positions.go:235`, `F/internal/grammar/weapon_hit_distance_resolver.go:71`.
  Seul `F/replay/build_vehicles.go:253-254` desarme le filtre (« arme, la bande ne rendrait qu'un
  quart du nuage »).
- Regle : le code dit lui-meme que le tag est la generation (`frame_records.go:171-172`,
  `projectiles.go:346-348`, `offline_biped_band.go:67` : « RequireTag1 est a DESARMER ») ;
  ADR 0034 D-10 (heuristique qui decide, non inscrite, non comptee).
- Consequence : quand le pool de slots bipede reboucle (BTB long), les corps de generation >= 2
  n'ont plus de positions ni de lectures delta. Mesure du depot
  (`.ai/V7.5/RAPPORT_PONT_APLATI_2026-09-10.md:97-98,125-137,200-202`) : `084a804d` (BTB 16 min)
  = 379 records de creation, 257 corps avec positions, **122 corps sans aucune position** ; le
  second corps du siege 603 tient en une frame ; `durationMs` du rejeu tronque ; `kill_positions`
  et distances perdues.
- Reproduction : `git grep -n "p+14, 2) != 1\|lay, true, func\|RequireTag1:    true" -- apps/go-api/internal/games/halo_infinite/film/internal/grammar/`
- Verification adverse : tient ; P0 sur les films qui rebouclent (un seul film mesure :
  `1c4c63c2`, `a349fea8` non mesures).
- rr : present (`positions_porte.go` ne relit aucun delta de generation >= 2).
- Traitement propose : mesurer le parc (corps sans positions par film), desarmer le filtre ET
  chainer par (slot, generation) les canaux qui chainent par slot (`equipment_changes.go:99-101,
  176-189`, `held_weapon_changes.go:80-84`), lot de comportement + corpus gate + recuisson.
  Decision : a trancher.
- **Couplage (verification adverse REP-A2)** : RA2-1, RA2-2 et RA2-3 (identites raisonnees par
  slot) ne sont latents QUE parce que les corps de generation >= 2 n'ont pas de positions.
  Corriger GB-1 seul les reveillerait : les deux doivent partir dans le meme lot, avec une cle
  (slot, generation) typee.

### [P1] OPS-3 — L'etape killsource post-sync (1.57) tourne une fois par joueur, en parallele, sur le meme arriere
- Ou : `sync/killcollector/postsync.go:250-275,402-421` (arriere global, `LIMIT`, sans filtre
  joueur) ; `sync/v2/post_sync.go:104-127` (une goroutine par joueur, `PostSyncParallelism=0`) ;
  `sync/v2/cycle.go:259` (un match d'escouade est dans la liste de chaque participant).
- Regle : `sync/killcollector/collector.go:42-43` (« le post-sync du serveur garde la boucle en
  serie ») ; re-controle sous writer impose a l'etape voisine 1.54 (`convergence.go:61-75`).
- Consequence : N telechargements + N decodages du meme film dans le serveur web, N passes
  append-only ecrites (les vues `_latest` restent justes : charge, pas donnee fausse). VPS web
  2 vCPU / 2 Go sans swap.
- **ABSENT de `main`** (`cf333a388`) : `killcollector/postsync.go` n'y existe pas. Le defaut
  arrive en production avec la fusion v7.5.
- Verification adverse : tient, P1, « risque de P0 sur le VPS ».
- rr : present.
- Traitement propose : `singleflight` par `match_id` (golang.org/x/sync est deja une
  dependance) ou semaphore global sur 1.57. Decision : **a trancher avant la fusion v7.5 vers
  main**.

### [P1] SRC-2 / OPS-4 — Le cache de films ecrit ses chunks sans atomicite et tient « present » pour « complet »
- Ou : `F/filmcache/write.go:69-77` (`os.Stat` puis `continue` ; `os.WriteFile` direct), `:79-82`
  et `:96` (manifeste).
- Regle : `write.go:13-16` (« le manifeste est le marqueur de commit […] jamais un film a moitie
  lisible ») ; `cmd/levelup/cmd_archive_films.go:44-46` (interruptible « a tout instant », mais
  sur `context.Background()` sans gestion de signal).
- Consequence : un chunk tronque (disque plein, `TASKKILL /F /T` d'air en dev, SIGKILL/OOM en
  prod, Ctrl+C) est adopte au passage suivant puis consacre par le manifeste ; `appendPackets`
  s'arrete sans signal au paquet tronque ; film ampute definitivement (le film expire cote
  serveur). Le `ChunkSize` de l'API est lu (`sync/haloclient/halo_client_film.go:67`) et jamais
  compare. Manifeste dechire jamais repare -> `MBitFilmAbsent` terminal alors que les chunks
  sont sur disque.
- Verification adverse : tient, P1 (le maillon « cache-first » depend de
  `LEVELUP_LEGACY_FILM_CACHE_DIR`, mais le `continue` suffit seul).
- rr : modifie — manifeste atomique et erreur si illisible (corrige pour l'avenir) ; boucle des
  chunks inchangee (present).
- Traitement propose : temp + rename par chunk (`platform/atomicfile`), comparaison a la taille
  annoncee, reparation d'un manifeste illisible.

### [P1] FK-1 — Un remplacant humain fait perdre son epinglage au bot de relais, sans signal
- Ou : `F/internal/facts/killsource/roster.go:212-213` (`nHumans: len(kf.names)`), `:255-257` ;
  aucun journal ni compteur sur `r.unpinned` ; `games/halo_infinite/replayidentity/bot_identities.go:56-63`
  retire le bot en silence.
- Regle : `roster.go:12-15,40-41` (« Non vide = anomalie a regarder, pas a ignorer ») ;
  CLAUDE.md regle 3.
- Consequence : morts du bot et morts qu'il inflige perdues ; assistances en `?N` ; avec un
  second indice libre, `publishable=FALSE` pour tout le match. `b1ad85eb` a la structure exacte
  et n'y a echappe qu'a un joueur pres ; le BTB est deja mesure (8 bots declares, 2 epingles,
  `RE_LOG_KILLWEAPON.md:9816-9817`). Le lot 5.2b.1 a ouvert le 4v4 aux remplacants sans revoir
  la regle ; le temoin `index_motif_test.go:20-21` place le bot au slot 9 et contourne le cas.
- Verification adverse : tient, P1. rr : present.

### [P1] FK-2 — Un nom de remplissage `?N` est publie comme assistant nomme, ecrit en base et affiche
- Ou : `F/internal/facts/killsource/assist.go:406-411` (ne rejette que `"?"`) contre
  `roster.go:226-228` (fabrique `?N`) et `roster.go:405` (traite le prefixe `?` comme silence).
- Regle : `assist.go:44` ; contradiction interne au paquet.
- Consequence : `?10` reel sur `b1ad85eb` ; ecrit par le collecteur
  (`sync/killcollector/collector_batch.go:75-77,94-96`), accepte par le persister ; Q21c
  (`queries_match.go:529-548`) ne filtre pas et `decorateAssist` pose `AssistState=named` :
  « ?10 » s'affiche comme assistant sur la page match.
- Verification adverse : tient (P2 declare) ; **releve en P1 par le superviseur** (donnee fausse
  visible, film reel). rr : present.

### [P1] GA2-1 — Portage divergent de `FUN_14076e524` : le corps d'i54 lit l'index a la largeur du mot de poignee et les axes a 6/6/6
- Ou : `F/internal/grammar/components_world.go:139-149` contre `traverse.go:180-192`,
  `components_position_i0.go:146-153,316-324`, `transloc_events.go:207-230` (valide 18/18).
- Regle : `components_position_i0.go:309-314` (« lue par TOUS les chemins de FUN_14076e524 ») ;
  ADR 0034 Contexte (la classe « deux lecteurs decident differemment » que l'ADR dit fermee) ;
  ADR M3 D-3 (Traversal = descripteur des deltas, pas la largeur absolue).
- Consequence : desynchronisation du record bipede (i55..i63 puis la suite du paquet delta),
  records perdus pour killsource. Corroboration : `.ai/V7.5/film_re/NOTE_5_19_RECORD_AVANT_LE_REJET_2026-09-22.md:130`
  (i54 lu a 344 bits = lecture « traversee » ; dernier record avant rejet dans 90,3 % des cas).
  Le lot 3.4.1 a corrige les deux autres lecteurs absolus du bipede, pas celui-ci.
- Verification adverse : tient, P1 ; non connu, non decide. Candidat marginal (~0,9 % des rejets
  de `bfecd02b`) au residu de film dense **clos par decision utilisateur du 23/09 : non rouvert
  ici**. rr : present.

### [P1] GA2-2 — `flock-position` lit `FUN_14076e524` au niveau du registre (0) au lieu de l'immediat 0x10
- Ou : `F/internal/grammar/components_flock.go:56-62` (appel `dispatch_biped.go:82-84`) ; meme
  motif probable pour asset-transform (`dispatch_biped.go:236-251`, niveau 30).
- Regle : `F/internal/profile/loi_largeurs.go:29-32` (« le NIVEAU est un IMMEDIAT du site
  d'appel ») ; la doc de la fonction elle-meme (`components_flock.go:47`, `0x10`).
- Consequence : sous-lecture de 22 a 48 bits par record ti=21 (golden : « ti=21 sous-lit », 0
  ferme sur 5 bobines). Verification adverse : tient, P1 portee limitee. rr : present.

### [P1] GA1-2 — Le repli « liaison par anticipation » (lot 5.23) est hors registre et compte nulle part
- Ou : `F/internal/grammar/world.go:93-144` (`LierParAnticipation` ; `AnticipationsParArchetype`
  sans appelant, `w.anticipations` jamais lu), `keyframe_anticipe.go:21` (« C est un REPLI »),
  `frame_infer.go:152-170`.
- Regle : ADR 0034 D-10 regles 1, 3, 4 et D-10 bis.
- Consequence : sur `bfecd02b`, stances 616 -> 841 dues a ce repli sans que l'artefact le dise ;
  ratchet aveugle (vocabulaire « anticipation », pas « repli ») ; regle de retrait inapplicable.
- Verification adverse : tient, P1. rr : present.

### [P1] RA1-1 — Les faits persistes figent des balayages commandes par l'appelant, que la fraicheur ne compare pas
- Ou : `F/replay/film_scan.go:376,425,463-467` ; `F/replay/film_inputs.go:204-212` (`applyTo`) ;
  `F/replay/filmfacts_fichier.go:323-335` (`Utilisable`) ; `replaybuild/filmfacts_cuisson.go:178-182`.
- Regle : `F/replay/build_from_facts.go:31-33` (« les entrees de l'APPELANT […] ne sont pas dans
  les faits ») ; ADR 0034 M4 (« these are facts of the film »).
- Consequence : une cuisson sans faits de base (cas documentes reels : ouvrier, `replay-build`
  sans `--facts`, base indisponible au backfill) ecrit des faits avec `ZoneScanned=false`,
  `BombReads=nil`, roster vide ; la reparation suivante les rejoue : KOTH/Bastions sans
  `zoneStates`, `coverage.bombArmings {scanned:true, reads:0}` faux, puis artefact « complet »
  fige jusqu'a la prochaine montee de revision. Les faits degrades s'ecrivent meme quand
  l'artefact appauvri est refuse.
- Verification adverse : tient, P1. rr : present.

### [P1] RB2-1 — Sieges : un arrivant precoce arrete l'appariement de tout son camp
- Ou : `F/replay/sieges.go:263-283` (`break`), `:322-325` (un arrivant qui repart n'est jamais
  partant).
- Regle : en-tete `sieges.go:31-33`, godoc `:257-258` ; regles produit (places finies, un
  partant libere sa place).
- Consequence : mesuree par la campagne rr (`RAPPORT_equipes_b1ad85eb.md` §C1/C2 sur la
  branche) : un camp a 5 sieges en 4v4 ; au parc, 14/111 documents depassent la taille d'equipe,
  22/111 montrent des partis.
- Verification adverse : tient, P1. rr : present ; **correctif existant sur `feat/rr-vague-d`
  (`placeParChainage`), non fusionne**.

### [P1] RB2-4 — `spawnSetFrom` compare la premiere emission d'arme a une image-cle FUTURE
- Ou : `F/replay/document_weapon_changes.go:164-189` (`pick := list[0]`).
- Regle : godoc `:158-159` (« le dernier releve qui PRECEDE ») ; D-10 regle 1 (non inscrit).
- Consequence : prise reelle classee `restated`, retiree de `weaponChanges`, l'arme au sol perd
  sa fin `pickup`. Mesuree par la campagne rr (`81c02726` : Hydra de JGtm ; `b1f01a33` : 3/5).
- Verification adverse : tient, P1. rr : present ; **correctif sur `feat/rr-m3` /
  `feat/rr-vague-d`, non fusionne**.

### [P1] RB2-5 — Un meme ramassage natif date et nomme plusieurs occupations de socle
- Ou : `F/replay/pad_pickup_dating.go:145-180` (aucun marquage du ramassage consomme).
- Regle : en-tete `:22-26` (« on s'abstient et on le compte ») ; D-10 regle 1 (jointure non
  inscrite).
- Consequence : `F/replay/usage_summary.go:351-354` credite le joueur deux fois (stats de
  session et d'escouade) ; chevauchements reels dans le golden `assembly_a521164d`.
- Verification adverse : tient, P1. rr : present.

### [P1] RB1-1 — `flag_carriers_killed` peut fermer le portage d'un porteur vivant et effacer sa capture
- Ou : `F/replay/flag_carries.go:362-389` (aucun filtre d'equipe : `scan.TeamOf` existe mais
  n'est pas passe ; un tueur non identifie n'exclut personne), `flag_carries_close.go:81-84`.
- Regle : `flag_carries.go:52-55` ; ADR 0034 D-10.
- Consequence : capture effacee, drapeau publie au sol pendant le portage ; mauvais ciblage deja
  observe sur `b8a44fe8` (cas `carried_open`).
- Verification adverse : tient ; P0 declare ramene a P1 (aucune fausse publication mesuree sur
  un portage ferme). rr : present.

### [P1] FO-1 / RA2-4 — La provenance `residu_de_manche` est publiee « deduit » sans voie
- Ou : `F/replay/identity_registry_section.go:318-328` (`methodeStatborg`) ; producteur
  `F/internal/facts/objectives/slotidentity_residue.go:53,66`.
- Regle : `games/canonical/film_identity.go:108,188-189`.
- Consequence : faible portee (le web ne lit pas `method`), mais le compteur de retrait de cette
  voie d'inference est aveugle. **Connu depuis l'audit 0.E (13/09), jamais route vers un lot.**
  Trouve independamment par deux relecteurs.
- Verification adverse : tient, P1. rr : present.

### [P2] Constats latents ou de portee faible (tous verifies « tient » ; tous presents sur rr sauf mention)

| ID | Constat | Ou | Note |
|---|---|---|---|
| SRC-1 | Les revisions ne hachent ni `F/types`, ni `F/damagetag` (donnees + seuil `Strong`), ni `games/weapons/filmshell`, qui decident la sortie | `F/revision/equivalence_test.go:73-104`, `F/types/grammar_mouvement.go:111-126`, `killsource/label.go:55`, `damagetag.go:154` | faits jugés frais sur un decodage perime ; lignes `match_kill_events` hors backlog |
| RA1-3 | Du code de balayage vit dans `F/replay` (`inventory_decode.go` : lecteur de bits prive, `deaths_source.go`, `player_index.go`, `origin.go`…) et aucune revision ne le hache | `F/replay/layers.go:114-118,142` | exemple REEL sur rr : `ScanDeaths` change de chunk sans montee (plan rr §8.20, faits d'`ab526724` renommes a la main) |
| RA1-4 | La cle de cuisson ne compare que module et `AxisW` (ni `Region`, ni `RegionIndexBits`, ni bornes) | `F/replay/filmfacts_decode.go:96-114,74` | correction de catalogue = faits « frais » incoherents |
| RA1-6 | La branche des faits efface l'erreur du fil des morts (premisse fausse) | `replaybuild/filmfacts_cuisson.go:91-97` vs `F/replay/film_scan.go:400-409` | `ab526724` reel ; rr le reconnait (§8.4, « breche S8 »), correctif M8 non livre |
| RA1-5 | Un fichier de faits corrompu fait paniquer au lieu de rendre une erreur | `F/replay/filmfacts_flux.go:144-176`, `filmfacts_fichier.go:285-297`, ~30 `make(…, int(r.u()))` | precedent reel (panique du 18/09) ; aucun `recover` |
| RA1-2 | Inventaire nil -> tranche vide au passage par les faits | `F/replay/filmfacts_decode.go:162-163` | couverture et calque divergent entre branches |
| RA1-7 | `LoadGeometry` avale les erreurs d'analyse CSV | `F/replay/geometry.go:81-85,171-178` | regle 3 |
| OPS-1 | Verrou solo sans proprietaire (`beat` et `Release` ne verifient rien ; vol concurrent) | `filmproc/solo.go:99-185` | P0 declare -> P2 : aucune prod ne le prend, chaque detenteur est borne ; `soloWrite` reecrit `StartedAt` a chaque battement |
| OPS-2 | L'action admin `replay-build/run` decode dans le serveur, hors verrou, sans plafond | `api/wire/registry_replay_build.go:31-66` | P1 -> P2 : la prod refuse (placement) ; 3 commentaires invoquent un verrou SUPPRIME |
| OPS-5 | Un film ecarte (cle inconnue) est rapporte « carte hors catalogue » par le parent | `replaychild/replaychild.go:126-132,232-233` | **etendu sur rr** : le classement par texte de `ErrFilmNonFinalise` est lui aussi inatteignable (compte en echec, l'inverse du commentaire) |
| FO-3 | Le pont par instants de mort lit le compteur sans les filtres de la serie publiee | `F/internal/facts/objectives/slotidentity_deaths.go:184-209` | silence, pas mauvaise attribution |
| FO-4 | `repli_emission_du_compteur_de_morts_jetee` omet un site ; branche `len(kept)==0` morte | `F/internal/facts/fallback/registre_objectifs.go:159-173` | le reste du constat est refute |
| GA2-3 | `DAT_144632be0` cable a 1 dans 4 lecteurs (dont `unit_control.go:258`, `default_state.go:297`) | `F/internal/grammar/components_position_i0.go:218,418` | doc « comme partout ailleurs » perimee depuis 3.4.1 |
| GA2-4 | Etat par defaut ti=13 : `FUN_140ce59bc` porte en R(4) seul, son jumeau lit la charge du variant | `F/internal/grammar/default_state_arch.go:176-190` | a trancher dans Ghidra |
| GA2-5 | Chemin world-object : largeurs de carte meme quand idx=-1 | `F/internal/grammar/dispatch_object.go:158-163` | jumeaux conformes via `absAxisWFor` |
| GA1-1 / GB-2 | Lecteurs de references d'evenement sans borne : panique sur payload corrompu | `F/internal/grammar/event_list.go:190,212`, `equipment_spawn_events.go:120-136` | P1 -> P2 : `appendPackets` n'emet jamais de paquet tronque ; absent du fuzz |
| GA1-3 | Un record NEW publie ses etats de mouvement sous le slot du record precedent | `F/internal/grammar/frame_infer.go:178-200,256-262` | P1 -> P2 : non mesure ; D13 « NON TRAITE » deja consigne |
| GA1-4 | La LIS de la table de datums garde la coincidence la plus tardive | `F/internal/grammar/keyframe_datums.go:106-137` | `ambigus` ne mesure pas ce que dit son commentaire |
| GA1-5 | Le recul sur les vacants de tete est neutralise par le bourrage | `F/internal/grammar/player_table.go:272-292` | la mesure « HeadVacant=0 » ne prouve rien |
| GB-3 | Ordre publie d'`equipmentChanges[]` non deterministe sur ex aequo (map + `sort.Slice`) | `F/internal/grammar/equipment_changes.go:134,166,290-302` | contredit « un ordre TOTAL » |
| GB-4 | `_, _ = DecodeFrameRecords(...)` dans un calcul mort en production | `F/internal/grammar/weapon_hits.go:258` | regle 7 plus que regle 3 |
| FK-3 | Le motif du xuid ne cherche pas les joueurs qui tuent sans mourir | `killsource/index_motif.go:173-191`, `feed.go:150-158` | |
| FK-4 | Le temps 4 peut reecrire un instant publie ; `Covered > RealPairs` possible | `killsource/hybrid.go:211-231`, `match.go:204-211` | masque une mort manquee |
| FK-5 | Double comptage au numerateur de sante | `killsource/hybrid.go:183-196,364-373` | viole `killhealth.go:34-35` |
| FK-6 | Doublons d'enregistrement : couple faux fabrique (le champ `bit` « pour dedoublonner » n'est jamais lu) | `killsource/feed_couples.go:160-188`, `assist.go:128-202` | compte en `Contradiction` (0 sur 21 films) |
| FK-7 | Faux WARN « carte absente » a chaque decodage sur Cliffhanger | `killsource/decode.go:131-139` | diagnostic seulement |
| RB2-3 | Vie d'une cle bornee par le seul successeur retenu | `F/replay/ground_weapon_objects.go:151-192`, `equipment_placement_ends.go:35-51` | P1 -> P2 : non observe |
| RB2-6 | Tolerance de siege : 2 000 ms divises par un pas en us -> 0 frame | `F/replay/vehicle_rides_seat.go:55` | verifie par le superviseur ; effet quasi nul |
| RB2-7 | Lien prise -> arme au sol ignore `LowUS` | `F/replay/document_ground_weapon_items.go:323-362` | |
| RB2-8 | Origine `dropped` decidee par une fenetre 200 ms / 1,5 m non inscrite | `F/replay/ground_weapon_rules.go:362-381` | D-10 regle 1 |
| RB1-2 | Reprise dans la meme frame que le lacher triee avant lui | `F/replay/flag_carries_lives.go:191,241,364` | P1 -> P2 : fins arrondies protegees |
| RB1-3 | Origine d'horloge illisible -> `bomb_arms` persiste comme zero mesure | `F/replay/bomb_stats_document.go:88-90` | P1 -> P2 : telechargement tout-ou-rien |
| RB1-4 | Lecteur Bond `.mvar` : comptes sans plafond (panique ou OOM fatal dans le serveur) | `F/replay/mapvar/cb2.go:86,150-158,229,253` | |
| RB1-5 | Hors catalogue, « un seul drapeau » suppose (decide et teste, premisse fausse) | `F/replay/flag_carries_handoff.go:45` | le rapport du depot mesure 57 portages / 775 s retires a tort ; latent |
| RB1-6 | Fermoirs indexes par slot sans la manche | `F/replay/flag_carries.go:322-357` | attenue par `closeByHomecoming` |
| RB1-7 | Portage de bombe ferme au lacher meme si la mort est anterieure | `F/replay/held_object_carry.go:135-139` | |
| RB1-8 | Allegement de jauge : descente non publiee dans la seconde | `F/replay/zone_states_gauge.go:117` | |
| CONV-1 | `os.IsNotExist` sur une erreur enveloppee `%w` : branche « table absente » morte | `replaybuild/zones.go:224` | verifie par le superviseur ; journal faux (WARN au lieu de DEBUG) |
| RA2-1 | Sur un slot recycle, un record de creation « ouvre » la vie du corps SUIVANT, publiee `direct` sous le joueur precedent | `F/replay/identity_registry_creation.go:242-254,185-197` | P1 -> P2 : l'ordre observe (084a804d) est l'ordre benin ; latent a cause de GB-1 |
| RA2-2 | Le nommage final « par occupation du slot » franchit une frontiere de corps (bot, `index_hors_table`) ; non inscrit au registre | `F/replay/unnamed_lives.go:86-114,159-184`, `identity.go:291-301` | P1 -> P2 : l'occurrence citee est refutee ; mecanisme reel ; rr tarit le cas `lectures_divergentes` |
| RA2-3 | Tirs et lancers rattaches par le pont aplati slot -> premier occupant | `F/replay/shots.go:86`, `grenades.go:190`, `coverage_decoder.go:132-151` | meme defaut deja corrige pour les seules equipes (`player_teams.go:94-99`) ; latent a cause de GB-1 |
| RA2-5 | `buildRoster` : comparateur non total sur une entree en ordre de map (deux bots d'un meme index) | `F/replay/identity.go:246-269` | ordre du roster non reproductible au-dela de 12 entrees |
| RA2-6 | `nameTracksByLives` exige un recouvrement > 0 : une vie d'un echantillon n'est jamais nommee par sa propre vie | `F/replay/identity.go:44-62` | effet OBSERVE : compteurs sur 17 artefacts ; `084a804d` `unnamedLives` 0 -> 1 (vie publiee anonyme, contraire a la regle produit) |

## Constats ecartes

| Constat | Axe | Motif d'ecart |
|---|---|---|
| FO-2 (manche courte toleree dans `contiguousRounds`) | correction | refute : lecture litterale du contrat, aucun cas au parc (mesure e1911 sur 1 351 films) |
| FO-4, partie `repli_emission_hors_domaine_jetee` | correction | refute : le « second site » est dans le fichier deja cite |
| OPS-6 (plafond suspendu 30 s pendant le compte rendu HTTP) | correction | refute : choix documente (`memguard.go:13-16`), borne, VM dediee |
| RB2-2 (deux vies vehicule de meme (slot, gen) fusionnees) | correction | refute : la generation change a chaque reutilisation ; toutes les vies ti=40 mesurees a gen=1 |
| RB1-9 (repli compte deux fois par cuisson CTF) | correction | refute en tant que defaut : l'unite publiee est le declenchement |
| OPS-1 en P0, RB1-1 en P0, OPS-2 / GA1-1 / GA1-3 / RB1-2 / RB1-3 / RB2-3 / RA2-1 / RA2-2 en P1 | gravite | ramenes (voir tableau P2 et fiches) |
| RA2-2, occurrence citee sur `084a804d` | correction | refutee : la vie `lectures_divergentes` precede le premier record de son siege, aucune frontiere de corps franchie |

## Architecture et conventions (axes non-bugs, synthese des deux relecteurs, chiffres verifies)

Forces mesurees : couche `source` exemplaire (lecture par mot de 64 bits inlinable, quatre
conventions de bord nommees, tests differentiels) ; provenance par valeur (33 lignes de profil
avec source/preuve/date, `ecs_table.tsv` 1 082 lignes) ; D-5 reellement exploite
(killcollector a 3 ouvriers) ; codec des faits bien concu (en-tete de 110 o, erreurs typees,
95x a 442x) ; isolation memoire par processus nee d'incidents ; erreurs `%w` 161/161, 0 `==`
sur erreur, atomics types, 0 `reflect`/`unsafe`/`interface{}` ; fonctions courtes (mediane 9 a
14 lignes, 8 fonctions > 80 lignes, toutes des aiguillages) ; 1 330 liens de doc `[Nom]`.

Faiblesses mesurees :
1. **Fraicheur et revisions mal calees sur les sorties** : SRC-1, RA1-1, RA1-3, RA1-4, RA1-6
   (la republication peut servir un decodage perime sans signal) ; a l'inverse l'empreinte hache
   les commentaires (46 % des lignes) et `facts.Rev` hache `objectives/` et `fallback/` que
   `killsource` n'importe pas (fausses alertes, backlog sans objet).
2. **Identite (slot, generation) non typee** : GB-1, RA2-1/2/3, RB2-1, FK-1 — plusieurs lecteurs
   raisonnent sur le slot seul alors que le registre connait les corps successifs.
3. **Portages jumeaux d'une meme fonction du jeu** : GA2-1..5 ; 7 lectures de bits artisanales
   hors `source` subsistent (ratchet nominatif).
4. **D-10 appliquee par un ratchet de vocabulaire** : GA1-2, RB2-4/5/8, filtre de GB-1 hors
   registre ; **81 entrees sur 99 ont `CompteurBranche: false`**, 55 visent un jalon clos.
5. **`F/replay` paquet-dieu** : 189 fichiers, 957 declarations non exportees, `Options` ~70
   champs en entree ET en sortie (`applyTo`), importe par **26 paquets de production** dont
   `persist`, `api/handlers`, `service` ; une seconde sequence de balayage dans
   `sync/killcollector/positions.go:259-321` avec d'autres entrees.
6. **Decodage multi-passes** : ~40 traversees du film par cuisson ; 8 canaux refont chacun
   `walkDeltaBipedRecords` (ancrage bit a bit). Impose en partie par le format, en partie choisi.
7. **CI** : elle protege l'assemblage, pas le decodage ; `-race` ne tourne jamais sur le film
   (seulement `shared-social-gate.yml`) ; 329 `*_research_test.go` (95 000 lignes, 36 % des
   tests) dans les paquets de production dont 218 sans tag ; 683 `t.Skip`, 3 `t.Parallel`.
8. **Contexte et journalisation** : `BuildBytes` cree son `context.Background()` ; aucune des 129
   fonctions exportees de `replay` ne prend de `ctx` ; 219 appels `slog` sans contexte (les
   attributs du `ContextHandler` sont perdus) ; D-4 tenue a moitie (6 appels dans `grammar`, 12
   dans `facts`, 174 + expvar dans `replay`) ; 22 variables de paquet exportees modifiables.
9. **Determinisme par chance** : 43 `sort.Slice` a cle unique sur des structs (pdqsort n'est
   stable par accident que sous 13 elements : les petites fixtures ne le revelent pas).
10. **Stdlib moderne quasi absente** : 1 appel `slices`/`maps`/`cmp`/`iter` contre 248 `sort.*` ;
    1 `range` sur entier contre 220 boucles a trois clauses ; 0 `iter`, `sync.OnceValue`,
    `synctest`, `b.Loop`. Palier neutre : ~350 sites par `go fix` (Go 1.26) + stdlib. PIEGES
    non neutres : `sort.Slice` -> `slices.SortFunc` si le comparateur n'est pas total,
    `slices.Clone` (vide vs nil en JSON), `omitzero`, et **surtout ne pas migrer `math/rand` vers
    `math/rand/v2`** (`rng.Perm` a graine fixe fait partie de la sortie de la bijection,
    `killsource/options.go:141`).
11. **Documentation perimee** : 13 affirmations d'etat fausses sur 16 verifiees (dont la doc de
    paquet de `killsource`, qui decrit le verrou supprime) ; 12 chemins `.ai/` morts sur 79 ; 170
    commandes `go test` visent `film/filmdec/` qui n'existe plus ; 10 doc comments colles a la
    mauvaise declaration ; `CLAUDE.md` prescrit `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`
    (le fichier est sous `.ai/V7.5/`) ; l'ADR 0034 est a 61 % un journal d'etat fige a M4
    (SchemaVersion 62 et revisions de septembre, l'arbre est a 68).

Ecarts ADR 0034 / arbre (releves par le relecteur architecture, commandes dans son rapport) :
« replay ne decode rien » (faux : etage de balayage + decodeur de bits) ; « seule porte aux
octets » (7 lectures artisanales, facade qui re-expose `LecteurSur`/`Paquets`/`Inflate`) ; « D-5
prouve sous -race » (pas en CI) ; « un seul etage de balayage » (second dans killcollector) ;
« verrou pris par les SIX points d'entree » (un septieme ne le prend pas) ; « D-10 Reached as
written » (82 % non comptes).

## Axes sans constat

- Invariants DuckDB (ART) de `killcollector` : respectes (persisters INSERT-only sous bail, vues
  `_latest`, tri `SQLStartTimeCanonical`, CLI exigeant le serveur arrete).
- Aucun `os.Getenv` dans le code de production du perimetre (configuration injectee).
- Aucun `reflect`, `unsafe`, `interface{}` ; aucune comparaison `err == ErrX`.

## Suite

**Escalade utilisateur (decisions, pas d'action proposee par l'audit)** :
1. OPS-3 avant la fusion v7.5 vers `main` (le defaut n'existe pas en production aujourd'hui).
2. GB-1 : P0 sur les films longs ; mesurer le parc puis lot de comportement.
3. Modele de revision / fraicheur (SRC-1, RA1-1, RA1-3, RA1-4, RA1-6) : touche l'ADR 0034 D-6/D-7.
4. Identite (slot, generation) comme type de premiere classe : touche plusieurs calques et
   `killcollector`.
5. GA2-1 / GA2-2 : candidats (marginaux) au residu de film dense, clos par decision du 23/09.
6. Decoupe de `F/replay` et facade : touche le ratchet de surface de la decision V25.
7. Politique de commentaires (contrat dans le code, histoire dans l'ADR/journal) et tag
   `research` pour les instruments.

**Deja en vol ailleurs** : RB2-1 et RB2-4 ont un correctif sur `feat/rr-vague-d` / `feat/rr-m3`
(non fusionne) ; RA1-6 est prevu au lot M8 de la campagne rr ; OPS-4 est corrige cote manifeste
sur `feat/retours-rejeu`.

**Candidats a un plan** (a cadrer sous `plan-review`) : robustesse E/S (SRC-2/OPS-4, OPS-1 par
verrou OS `LockFileEx`/`flock`, RA1-5, RB1-4, GA1-1) ; replis D-10 (GA1-2, RB2-5, RB2-8, cablage
des 81 compteurs) ; drapeau (RB1-1, RB1-5, RB1-6) ; killsource (FK-1, FK-2, FK-3..7) ;
modernisation neutre (`go fix` + `errors.Is` + commentaires perimes + `ctx`, jugee a
`replay-equiv` zero difference, goldens d'empreinte regeneres a revision constante).
