# SCOUT S1 — Sons de véhicules et tourelles (2026-08-31)

> Exécuteur : agent scout Sonnet, lot S1 du `PLAN_VEHICULES_TOURELLES.md`. Worktree
> `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`). Aucun commit, aucune écriture
> hors ce fichier et le dossier `scout_sons_preuve/`. Aucune compilation Go (contrainte du
> plan — un autre agent compile en parallèle), aucun Python exécuté.
>
> Gate du lot (`PLAN_VEHICULES_TOURELLES.md` tableau des lots, ligne S1) : « Rapport options
> + reco, preuve minimale ». Les deux sont livrés ci-dessous.

## Résumé en une page

| Question | Verdict |
|---|---|
| 1. Outillage existant | `apps/go-api/cmd/weapon-sounds/` (Go) + `vgmstream-cli.exe` (décodage Vorbis Wwise) + `ffmpeg` (égalisation finale seulement). Mode d'emploi exact retrouvé et **une lacune comblée** : la chaîne `.pck → .wem` documentée comme « script Python hors dépôt » est en fait un format trivial (`AKPK`) déjà décrit en clair dans `pck.go` — je l'ai rejouée en PowerShell sans Go ni Python, à titre de preuve. |
| 2. Chemin de tags véhicules | `vehi` → 46/62 armements directs ou chaîne `vcdd→sofd→sofa→uwfa→weap` (16/62) — **acquis, aucune mesure neuve nécessaire**. Nouveau : **les banques `_veh_`/`_tur_` existent comme fichiers `.pck` NOMMÉS EN CLAIR sur le disque du jeu** (37 fichiers), trouvées par une simple liste de dossier — pas besoin de la voie « hash + dictionnaire » développée pour les équipements/modes. |
| 3. Boucles moteur | Extractibles (preuve faite : un clip de 0,88 s du Ghost décodé sans erreur). Le drapeau de bouclage Wwise (`sLoopCount`) est déjà lu par l'outil pour d'autres familles de banques ; son application aux banques véhicule est une extension directe, non encore exécutée (contrainte : pas de compilation Go pendant ce scout). |
| 4. Tirs des 62 armements | **Oui, même chaîne** : ce sont des tags `weap` ordinaires, le champ nommé « Weapon Fire Sound » de `weap.xml` s'applique sans modification. Preuve directe : le pack `sb_010_tur_un_machinegun.pck` (un des 62) est un `.pck` AKPK standard portant du Wwise-Vorbis, format identique à celui des armes de joueur. |
| 5. Recommandation | Lot V3 **plus petit et moins risqué** que le lot armes déjà livré (55 armes traitées) : environ 12 couples véhicule/tourelle, outillage réutilisé à l'identique, un seul ajout Go trivial (lecteur AKPK, ~40 lignes, déjà à moitié écrit dans `pck.go`). Détail §5. |

Fichiers de preuve écrits (aucun commit) :

```
.ai/V7.5/film_re/scout_sons_preuve/tur_un_machinegun_506708444.wem   (4 416 o, source brute)
.ai/V7.5/film_re/scout_sons_preuve/tur_un_machinegun_506708444.wav   (38 444 o, écoutable)
.ai/V7.5/film_re/scout_sons_preuve/veh_cv_ghost_146409670.wem        (16 013 o, source brute)
.ai/V7.5/film_re/scout_sons_preuve/veh_cv_ghost_146409670.wav        (168 844 o, écoutable)
```

---

## 1. L'outillage existant

### 1.1 Trois chantiers sons antérieurs, tous vivants dans ce dépôt

| Chantier | Où | Ce qu'il livre |
|---|---|---|
| Sons de fin de partie | `apps/web/src/features/match-replay/endMatchSound.ts` + `.test.ts` ; plan `.ai/V7.5/PLAN_REPLAY_CADRAGE_VICTOIRE.md` (§ « CS1 », copie de 14 wav depuis `_fin_partie/livraison/`) | 14 wav UI (voix/musique de fin), extraits une fois, sans rapport structurel avec les sons d'armes/véhicules (pas de tag `.module`, pas de Wwise). Mentionné pour mémoire, non réutilisable ici. |
| Sons d'armes classiques | `apps/go-api/cmd/weapon-sounds/` (Go) ; `.ai/V7.5/PLAN_EXTRACTION_SONS_ARMES.md`, `.ai/V7.5/RECETTE_SONS_ARMES.md`, `.ai/V7.5/HANDOFF_SONS_ARMES_2026-08-15.md` | La chaîne complète `.pck`/`sbnk` → événement Wwise → `.wem` → coup reconstitué. **C'est la chaîne directement réutilisable pour les véhicules** (§4). |
| Sons d'objectifs/équipements/modes | même outil, modes ajoutés en continu ; `.ai/V7.5/RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`, `HANDOFF_SONS_RECONSTITUTION_2026-08-27.md`, `RE_GESTES_SONORES_2026-08-27.md`, `RE_SONS_RATELIER_ET_KOTH_2026-08-30.md` | La technique de **nommage d'une banque par hachage FNV-1** de son nom de fichier probable, et la lecture fine du format Wwise (boucles, délais, modes d'enchaînement). |

### 1.2 L'outil Go : où il vit réellement, et lequel exécuter

`apps/go-api/cmd/weapon-sounds/` existe en **trois états différents** selon le worktree, ce qui n'est documenté nulle part et vaut la peine d'être consigné :

| Worktree | Branche | Dernier commit touchant l'outil | Modes disponibles |
|---|---|---|---|
| `LevelUp-wt-replay2d` | `feat/extraction-sons-armes` | (2026-08-16) | `probe/map/audit/noms/lien/banks/sndscan/qui/deps/remonter/tir/melee/meleefx/cadence/arbre/embarques/lot/lot-tir/final` — **`weapon-sounds.exe` déjà COMPILÉ y est présent** (vérifié `ls`, 2026-08-16) |
| `LevelUp` (checkout principal, `main`) | `main` | après le merge equip-sons | ajoute `eqip.go/eqip_arbre.go/eqip_banks.go/eqip_durees.go/wemduree.go` — pas de `.exe` présent |
| **`LevelUp-wt-vehicules`** (ce worktree, base `feat/v75`) | `wt/vehicules-tourelles` | `05a150fda` (le plus récent, 2026-08-30/31) | tout ce qui précède **+** `banks_noms.go/banks_dico.go/audit_actions.go/audit_boucles.go/conteneurs_mode.go/orphelins.go/blend_dump.go/actions_delai.go/variantes_conditionnelles.go/chaines.go` — **aucun `.exe` compilé nulle part** |

Conséquence pratique pour la suite du chantier (V3) : la version la **plus complète** de l'outil est déjà ici, sur la branche de base (`feat/v75`), donc accessible sans merge. Mais je n'ai **pas pu la compiler** (contrainte du plan — bâtir en Go pendant ce scout corromprait le cache partagé avec l'agent qui compile en parallèle), et **aucun exécutable compilé de cette version la plus récente n'existe nulle part sur la machine** (vérifié dans les 6 worktrees pertinents + `.claude/worktrees/*` + `Downloads/Halo Infinite - Sons v75/` — recherché explicitement, rien trouvé). L'exécutable trouvé dans `LevelUp-wt-replay2d` est **antérieur** aux modes `banks-noms`/`eqip-arbre`/`chaines` — je ne l'ai donc pas utilisé (il n'aurait pas pu nommer une banque véhicule par hachage) ; **je ne l'ai pas non plus eu besoin**, ma preuve (§6) a pris un autre chemin.

Dépendance de build : CGO (`apps/go-api/internal/ooz`, décompression Kraken/Oodle) — gcc msys64 ucrt64, déjà résolu pour tout le chantier son (`PLAN_EXTRACTION_SONS_ARMES.md` §0).

### 1.3 Dépendances externes, vérifiées sur cette machine

| Outil | Chemin constaté | Rôle |
|---|---|---|
| `vgmstream-cli.exe` (r2117) | `C:\Users\Guillaume\Desktop\Halo Infinite - Sons armes\_outils\vgmstream\vgmstream-cli.exe` | **Seul décodeur** du Vorbis Wwise custom (`fmt = 0xFFFF`). Testé directement par moi : voir §6. |
| `ffmpeg` (8.1.2) | `C:\Users\Guillaume\AppData\Local\Microsoft\WinGet\Packages\Gyan.FFmpeg_Microsoft.Winget.Source_8wekyb3d8bbwe\ffmpeg-8.1.2-full_build\bin\ffmpeg.exe` (winget) | **Ne décode PAS** le Vorbis Wwise (confirmé deux fois dans `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md` §7 : rend `unknown codec`). Sert uniquement à la mesure de crête et à l'égalisation finale (LUFS/dBTP). |
| Scripts Python hors dépôt | `Desktop/Halo Infinite - Sons armes/_outils/*.py` (`akpk_unpack.py`, `conv_lot.py`, `coups_lot.py`, `manifeste2.py`, `livraison.py`, `reinjecte.py`) | Non exécutés par moi (contrainte de ce scout : « pas de Python »). Documentés dans `RECETTE_SONS_ARMES.md` §1-§7 comme le chemin **habituel** pour l'étape `.pck → .wem` et l'assemblage final. |

### 1.4 Découverte : la chaîne `.pck → .wem` n'a pas besoin de Python

**C'est la lacune la plus utile que ce scout comble.** `RECETTE_SONS_ARMES.md` §1 présente l'extraction `.pck → .wem` comme un script Python (`akpk_unpack.py`), documentation qui laisse penser que Go/vgmstream ne suffisent pas. En pratique :

1. `vgmstream-cli.exe` ne sait **pas** ouvrir un `.pck` directement — testé : `vgmstream-cli.exe -m -i sb_010_tur_un_rocketturret.pck` échoue avec `failed opening`.
2. Mais le format du conteneur (`AKPK`, Wwise File Package) est **déjà documenté en clair, à l'octet près**, dans l'en-tête de `apps/go-api/cmd/weapon-sounds/pck.go` (lignes 1-16) : magic `AKPK`, trois tables (banks/streams/externes), chaque entrée `{id u32, tailleBloc u32, tailleFichier u32, blocDepart u32, idLangue u32}`, offset réel = `blocDepart × tailleBloc`.
3. J'ai rejoué ce format en ~90 lignes de PowerShell (aucun Go compilé, aucun Python), et il fonctionne à l'identique de la lecture Go (`lirePck`/`lireTablePck` dans `pck.go`) — preuve faite en §6.

Conséquence pour V3 : le chemin `.pck → .wem` peut rester **100 % Go** si on ajoute un mode de dump à `cmd/weapon-sounds` (quelques dizaines de lignes, réutilisant `lirePck` déjà écrit) — ou rester scripté hors dépôt comme aujourd'hui. Les deux options sont désormais posées avec leur coût réel (§5), alors qu'avant ce scout, l'hypothèse implicite était « il faut Python ».

---

## 2. Le chemin de tags véhicules

### 2.1 Ce qui est déjà acquis (killweapon, aucune mesure neuve)

Source : `.ai/ETAT_DE_L_ART_KILLWEAPON.md` §2.5 (lignes 155-171) et `.ai/V7.5/killweapon/RE_LOG_KILLWEAPON.md` (sections `7ter.45` à `7ter.50`).

- **Règle R-VÉHICULE** : un tag `weap` est un armement de véhicule si un `vehi` le référence directement (**46/62**) ou s'il pend à la chaîne `vcdd → sofd → sofa → uwfa → weap` (**16/62**). Disjonction totale mesurée avec le catalogue d'armes joueur (0/62, p = 1,1e-06).
- **Distinction tourelle / fixe**, par deux signaux indépendants : la classe ASCII du `weap` (`turret` vs `fixed`, lisible dans les modules — `stringsSize` = 0 partout ailleurs, seule cette classe survit) et **la nomenclature de la banque du châssis** (`_tur_` vs `_veh_`), à 22/23 et 17/22. Confirmé en Theater : le Ghost (arme intégrée, pas de siège arrière) est `fixed`, une tourelle UNSC arrachée à la main est `turret`.
- L'ancre nommée de toute la règle est la **Gungoose**, exclusivement par la chaîne `vcdd`.
- Ce que le §2.5bis ferme explicitement comme **hors périmètre de ce scout** : le véhicule comme entité répliquée dans le film (positions, occupant, lien joueur-véhicule) — c'est le lot V1/V2, pas S1.

### 2.2 Ce que ce scout ajoute : les banques existent, nommées en clair, sur le disque

Le doc killweapon établit la règle de nomenclature (`_tur_`/`_veh_` comme **sous-chaîne** d'un nom de banque) mais ne liste aucune banque par son nom complet — la mesure originale (`7ter.47`/`7ter.48`) travaillait par comptage statistique, pas par inventaire. J'ai fait l'inventaire, par une simple lecture de dossier (`Get-ChildItem` sur `Sound/win/SFX`, **aucun outil, aucune hypothèse** — ce sont des noms de fichiers réels sur le disque du jeu installé) :

**Sur 841 `.pck` au total, 73 vivent dans la même plage numérique que les armes (`sb_009_*`/`sb_010_*`), répartis SANS RESTE en quatre familles : `_wea_` (armes), `_grn_` (grenades), `_veh_` (15 fichiers), `_tur_` (8 fichiers).**

Châssis / véhicules (`_veh_`), 15 packs :

| Pack | Taille | Véhicule probable | Confiance |
|---|---|---|---|
| `sb_010_veh_cv_ghost.pck` | 1,45 Mo | Ghost (Covenant) | **Haute** — nom sans ambiguïté |
| `sb_010_veh_cv_ghostboss02.pck` | 4,70 Mo | Ghost (variante boss/campagne) | Haute |
| `sb_010_veh_cv_banshee.pck` | 4,93 Mo | Banshee | **Haute** |
| `sb_010_veh_cv_wraith.pck` | 5,15 Mo | Wraith | **Haute** |
| `sb_009_veh_cv_phantom.pck` + `sb_010_veh_cv_phantom.pck` | 0,27 + 2,24 Mo | Phantom (deux banques distinctes — rôles à distinguer) | Haute |
| `sb_010_veh_un_scorpion.pck` | 5,69 Mo | Scorpion | **Haute** |
| `sb_010_veh_un_wasp.pck` | 6,62 Mo | Wasp | **Haute** |
| `sb_010_veh_un_pelican.pck` | 0,41 Mo | Pelican | Haute (véhicule non pilotable en multi standard — à vérifier utilité) |
| `sb_010_veh_bt_chopper.pck` + `chopperboss01.pck` | 3,32 + 0,99 Mo | Chopper (Bannis) | **Haute** |
| `sb_010_veh_un_falcongrenadelauncher.pck` + `falconlmgturret.pck` | 3,70 + 2,25 Mo | Falcon (présence en multijoueur Halo Infinite **à vérifier**, hors périmètre son) | Moyenne |
| `sb_010_veh_un_rockethog.pck` | 3,70 Mo | **Warthog** — nom interne, identité exacte à confirmer par la chaîne de tags (piège déjà documenté : « un nom interne de pack ne dit pas quelle arme c'est », `HANDOFF_SONS_ARMES_2026-08-15.md` §5 sur `heatwave`/`cindershot`) | **Basse-Moyenne**, cf. réserve ci-dessous |
| `sb_010_veh_un_wargoose.pck` | 4,06 Mo | Gungoose ou Mongoose (hypothèse par élimination — aucun pack `mongoose`/`gungoose` n'existe par ce nom, vérifié explicitement) | **Basse**, cf. réserve |

Tourelles (`_tur_`), 8 packs :

| Pack | Taille | Arme probable |
|---|---|---|
| `sb_010_tur_un_machinegun.pck` | 2,29 Mo | Mitrailleuse/chaingun UNSC — **extrait et vérifié en §6** |
| `sb_010_tur_un_gausscannon.pck` | 4,59 Mo | Canon Gauss UNSC |
| `sb_010_tur_un_rocketturret.pck` | 0,20 Mo | Tourelle roquette UNSC |
| `sb_010_tur_cv_plasmacannon.pck` | 2,69 Mo | Canon plasma Covenant |
| `sb_010_tur_cv_shadeturret.pck` | 2,94 Mo | Tourelle Shade Covenant |
| `sb_010_tur_fr_beamautoturret.pck` | 0,55 Mo | Tourelle à faisceau Forerunner |
| `sb_010_tur_bt_gatlingmortar.pck` + `_sentinelminiboss.pck` | 2,37 + 0,81 Mo | Mortier/chaingun Bannis |

**Réserve explicite** (à ne pas sauter en V3) : `rockethog` et `wargoose` sont des **noms de code internes**, pas des noms en jeu confirmés. Le chantier armes a déjà payé cette erreur une fois (`fr_heatwave` menait à `hinf_cindershot`, pas à l'arme « Heatwave »). La bonne méthode, déjà écrite et reproductible, est de croiser le pack par le **nom de la banque `sbnk`** (hash FNV-1, `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md` §0) ou par la **chaîne de tags** `vehi → weap` déjà établie pour R-VÉHICULE — jamais par ressemblance de nom.

**Aucun pack séparé pour Razorback ou Mongoose/Gungoose** (recherché explicitement : `mongoose|gungoose|razorback|recon`, zéro résultat). Cohérent avec le fait que le Razorback est une variante cargo du Warthog (probablement même châssis sonore que `rockethog`) et qu'un pack de **mode** existe déjà à part (`sb_004_mod_mp_shared_razorback`, 2 événements — `RE_SONS_RATELIER_ET_KOTH_2026-08-30.md` §8.2 — vraisemblablement spawn/despawn, pas le moteur).

**Nuance sur le clivage `_veh_`/`_tur_`** : il n'est pas parfaitement propre au niveau des noms de fichiers — `sb_010_veh_un_falconlmgturret.pck` porte le mot « turret » dans un pack préfixé `veh_`. Cohérent avec la mesure killweapon elle-même (22/23 et 17/22, pas 100 %) : la règle est un signal fort, pas une garantie absolue.

**Trouvaille annexe, hors périmètre strict mais utile au lot V1/V2 (« état détruit »)** : six packs `sb_008_exp_vehicle_{large,med,small}_{covenant,unsc}.pck` — des explosions de véhicule catégorisées par taille et faction, distinctes du catalogue « objets explosifs » (bobines) déjà résolu dans `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md` §5. Non creusé ici (hors des 5 questions posées), signalé pour mémoire.

### 2.3 Le tag `vehi` lui-même n'a pas de définition XML nommée (contrairement à `weap`)

Vérifié par grep dans `apps/go-api/internal/himap` et dans tout `.ai/` : il n'existe **aucun équivalent de `weap.xml`** pour `vehi` (pas de « vehi.xml », pas de champs nommés type « Engine Sound »). Le seul endroit où `vehi` est traité comme groupe de tag est `.ai/V7.5/outillage/forge_palette/tmp_forgename/chain.go` ligne 124, qui le classe simplement au même rang que `weap`/`eqip`/`proj` (un objet placeable de palette Forge) — aucune information sur ses champs internes.

**Conséquence pratique, et c'est une bonne nouvelle** : ça n'empêche rien. Contrairement aux banques d'équipement/mode (qui n'avaient **aucun** `.pck` nommé et ont dû être trouvées par hachage — 3 banques sur 17 seulement touchaient un pack nommé, `PLAN_EQUIPEMENTS_MANQUANTS_SONS.md`, découverte 4 en fin de plan), **les banques véhicules ont directement leur `.pck` nommé sur disque** (§2.2). On peut donc retrouver le pack châssis d'un véhicule sans jamais avoir besoin de décoder la structure interne du tag `vehi` — la voie directe (nom de fichier) suffit, la voie par tag (`vehi → …`) ne sert qu'à **confirmer** l'identité en cas d'ambiguïté (rockethog, wargoose).

---

## 3. Les boucles moteur

### 3.1 Extractibles : prouvé

Le clip `veh_cv_ghost_146409670.wav` (§6, 0,88 s, stéréo 48 kHz) montre que le pipeline d'extraction ne bute sur aucun obstacle propre aux véhicules — même format Wwise-Vorbis, même conteneur `.pck`/AKPK, même décodeur.

### 3.2 Le bouclage lui-même : technique déjà écrite, pas encore appliquée aux véhicules

`RE_GESTES_SONORES_2026-08-27.md` §2 établit, sur les banques d'équipement/mode, que le nombre de lectures d'un conteneur Wwise est lisible directement dans la structure `AkRanSeqCntrInitialValues` (`sLoopCount` — 0 = boucle infinie), avec le mode d'enchaînement (`eTransitionMode` — bout à bout, cadence de déclenchement, etc.). C'est déjà codé (`conteneurs_mode.go`, mode `audit-boucles`, `chaines.go`) et validé sur 1 753 conteneurs de `common`. **Rien dans cette lecture n'est spécifique à la famille de banque** (mode/équipement) : c'est un champ générique du format Wwise, donc son application aux banques `_veh_` est une extension directe — non exécutée dans ce scout (contrainte : pas de compilation Go), mais sans obstacle prévisible.

### 3.3 Un candidat par véhicule : oui, avec une réserve à trancher en V3

Mesure faite sur `sb_010_veh_cv_ghost.pck` (248 entrées) : distribution de tailles de 1,4 Ko à 16 Ko, soit environ 0,08 s à 0,88 s au bitrate observé (144-171 kbps). **Aucun clip de plusieurs secondes** — cohérent avec la pratique Wwise standard pour un moteur de véhicule : le jeu ne stocke pas un long enregistrement, il boucle un court cycle et pilote sa hauteur/son intensité en direct par un paramètre de jeu (RTPC, typiquement la vitesse), exactement le mécanisme déjà rencontré et résolu **au niveau méthode** pour le translocateur (`RE_GESTES_SONORES_2026-08-27.md` §4, H4 « Blend piloté par RTPC »).

Pour le rejeu 2D — une seule boucle simple par véhicule, sans simulation de régime moteur — le candidat naturel est un clip de taille **moyenne à grande** du pack `_veh_` (pas les plus petits, probablement des à-coups/impacts de suspension ; pas nécessairement le plus grand non plus, qui peut être un événement rare). La désignation précise **doit passer par une écoute**, comme pour tous les gestes sonores déjà livrés dans ce chantier (`RECETTE_SONS_ARMES.md` §5 : « les votes priment sur tout critère » ; aucune exception trouvée qui permette de sauter cette étape).

### 3.4 Réserve produit à acter avant V3

Le rejeu 2D actuel ne pilote aucun paramètre de jeu en direct pour les sons d'armes (doctrine actée : rendu « au point de référence », `RECETTE_SONS_ARMES.md` §4 et §8bis) — jouer une boucle moteur à vitesse fixe, sans crossfade de régime, est cohérent avec cette doctrine existante et ne demande **aucune** décision produit nouvelle.

---

## 4. Les tirs des 62 armements de véhicule : même chaîne que les armes classiques ?

**Oui, sans aucune adaptation du format.** R-VÉHICULE (§2.1) établit que ces 62 armements sont des tags `weap` ordinaires — la seule différence est *comment on y accède* (`vehi` direct ou chaîne `vcdd`), pas *ce qu'ils sont*. Le champ nommé « Weapon Fire Sound » de `weap.xml`, qui donne le son de tir pour les armes de joueur (`RECETTE_SONS_ARMES.md` §3), est un champ du tag `weap` lui-même : il s'applique donc identiquement, quel que soit le chemin qui a mené à ce tag.

**Preuve directe apportée par ce scout** (§6) : `sb_010_tur_un_machinegun.pck` — un des 62 armements de véhicule selon la classification killweapon (tourelle UNSC) — est un `.pck` AKPK standard (même en-tête, mêmes tables) portant des `.wem` au format **« Custom Vorbis », en-tête RIFF Wwise**, format rigoureusement identique à celui mesuré sur `sb_010_wea_un_assaultrifle.pck` dans `PLAN_EXTRACTION_SONS_ARMES.md` (étape 1). Le clip extrait (ID `506708444`, 0,2 s, stéréo 48 kHz, 178 variantes dans le pack) a exactement la forme d'un tir automatique en salve courte — comparable aux « 296 micro-échantillons de 0,08 s » du fusil d'assaut cités dans le même plan (problème initial du chantier).

**Deux adaptations réelles, mais mineures, à prévoir pour le pipeline existant** :

1. **Le nommage en jeu ne passe pas par le catalogue standard.** Le pipeline `lot-tir` actuel joint un `weap` à son nom en jeu via `jeu/index.json` → `weapon_names.toml` (`RECETTE_SONS_ARMES.md` §3). Or R-VÉHICULE mesure une disjonction **totale** (0/62) entre les armements de véhicule et ce catalogue (`ETAT_DE_L_ART_KILLWEAPON.md` §2.5, ligne « Quelle est la force de la règle ? »). Il faudra nommer ces 62 `weap` par leur **véhicule porteur** (`vehi` parent), pas par une entrée catalogue qui n'existe pas.
2. **Le rattachement passe parfois par un chemin plus long.** 16/62 armements n'ont pas de lien direct `vehi → weap` mais la chaîne à quatre maillons `vcdd → sofd → sofa → uwfa → weap`. C'est déjà mesuré et vérifié (killweapon), pas à redécouvrir — juste à réutiliser tel quel dans le lot V3.

---

## 5. Recommandation pour le lot V3

### 5.1 Chaîne retenue

1. **Réutiliser tel quel** l'acquis killweapon (R-VÉHICULE) pour la liste des 62 `weap` et leur véhicule porteur — zéro mesure neuve.
2. **Retrouver le pack châssis/tourelle par nom direct** dans `Sound/win/SFX/*.pck` (§2.2, 23 candidats déjà repérés) — plus rapide que la voie « hachage + dictionnaire » développée pour les équipements, puisque ces packs sont déjà nommés en clair. Confirmer les identités ambiguës (`rockethog`, `wargoose`) par la chaîne de tags `vehi → …` ou par le nom de banque `sbnk` (FNV-1, méthode déjà écrite dans `RE_BANQUES_SONORES_NOMMEES_2026-08-26.md`).
3. **Étendre `cmd/weapon-sounds`** d'un mode de dump AKPK (reprendre `lirePck`/`lireTablePck`, déjà écrits dans `pck.go` — il ne manque qu'une boucle d'écriture de fichier, quelques lignes) pour rester 100 % Go sur cette étape, plutôt que de dépendre du script Python hors dépôt.
4. **Décoder en wav** via `vgmstream-cli.exe` (déjà présent, aucune dépendance nouvelle).
5. **Désigner à l'oreille** — par véhicule — la boucle moteur (parmi les clips moyens/longs du pack `_veh_`) et le ou les tirs (parmi les clips courts, filtrés comme pour les armes par le champ « Weapon Fire Sound »). Même méthode, mêmes garde-fous (garde-fou de durée, distinction 1P/3P le cas échéant) que `RECETTE_SONS_ARMES.md`.
6. **Livrer** avec la même discipline que les lots précédents (égalisation −16 LUFS / −1 dBTP, manifeste JSON, remplacement miroir dans `static/sounds/halo_infinite/`).

### 5.2 Coût estimé

**Plus petit que le lot armes déjà livré.** Le lot armes a traité 55 packs, résolu 37 chaînes de tir, avec plusieurs cycles de correction de format (conteneurs `Switch`, médias embarqués, perspectives 1P/3P — tout ce travail dur est fait et RÉUTILISABLE tel quel). Le lot véhicules porte sur **environ 11-12 couples véhicule/tourelle** (15 packs `_veh_` moins les variantes boss, 8 packs `_tur_`) — un périmètre notablement plus restreint, sur un format déjà maîtrisé.

Travail réellement neuf :
- ~40 lignes Go (dump AKPK) ou réutilisation du script PowerShell de preuve (§6), au choix du pilote.
- Identification de 2 noms de pack ambigus par chaîne de tags (méthode déjà écrite, pas à réinventer).
- Écoute et désignation d'environ 12 boucles moteur + tirs de tourelle/arme intégrée.

### 5.3 Risques

1. **Identité de pack incertaine** (`rockethog`, `wargoose`) — risque **faible**, mitigé par la chaîne de tags déjà validée sur le cas Gungoose.
2. **Boucle moteur pilotée par RTPC de vitesse** (crossfade multi-couches probable, par analogie avec le translocateur) — risque **moyen** pour le réalisme sonore (un seul clip fixe ne rendra qu'un régime), mais **acceptable** pour un rejeu 2D qui ne simule déjà aucun RTPC pour les armes (doctrine existante, pas une dérogation nouvelle).
3. **Razorback / Mongoose / Gungoose sans pack `_veh_` nommé propre** — nécessite de vérifier un partage de châssis (Razorback ≈ Warthog) par la chaîne de tags plutôt que par supposition.
4. **Tourelles de carte statiques** (chaingun ramassable à la main, hors véhicule) : `sb_010_tur_un_machinegun.pck` en est vraisemblablement la source aussi (même modèle d'arme, relation many-to-one déjà établie pour les armes de joueur — `ETAT_DE_L_ART_KILLWEAPON.md` §2.2 « Un tag = une arme ? Non ») — à confirmer, pas à supposer, lors du lot V3.
5. **CGO/Oodle** pour la décompression des tags `sbnk` — prérequis déjà résolu pour tout le chantier son, aucun risque nouveau.

### 5.4 Livrables proposés

Catalogue par véhicule : un fichier boucle moteur + un ou plusieurs fichiers de tir, plus un manifeste JSON au même format que les livraisons précédentes (`static/weapons-assets/.../index.json` ou équivalent véhicule). Le branchement dans le moteur sonore du rejeu (`replaySound.ts` et les fichiers `*Sound.ts` déjà en place) est une suite naturelle, mais dépend de la position/état des véhicules dans le document de rejeu — donc du lot V1/V2, pas de ce scout.

---

## 6. Preuve minimale réalisée

### 6.1 Méthode

Aucune compilation Go, aucun script Python exécuté. Deux extractions, chacune en trois étapes :

1. Lecture du conteneur `AKPK` du `.pck` cible via un script PowerShell (`lire_akpk.ps1`, scratchpad de session — transcription directe du format déjà documenté dans `apps/go-api/cmd/weapon-sounds/pck.go`, aucune recherche nouvelle).
2. Découpe des octets bruts d'UNE entrée (`Offset`, `Taille` lus dans la table) vers un fichier `.wem`.
3. Décodage par `vgmstream-cli.exe` (outillage existant, non modifié).

### 6.2 Tir de tourelle — `sb_010_tur_un_machinegun.pck` (mitrailleuse/chaingun UNSC)

- Pack source : `C:\Program Files (x86)\Steam\steamapps\common\Halo Infinite\Sound\win\SFX\sb_010_tur_un_machinegun.pck` (2 287 900 octets, 178 entrées).
- Entrée extraite : ID `506708444`, offset `1 077 248`, taille `4 416` octets (taille proche de la médiane du pack — candidat « un coup isolé », pas un fragment atypique).
- Métadonnées vgmstream : `48000 Hz, stereo, Custom Vorbis, Audiokinetic Wwise RIFF header, 171 kbps, 0,200 s`.
- Fichiers écrits : `scout_sons_preuve/tur_un_machinegun_506708444.wem` (source brute) et `.wav` (38 444 octets, écoutable).

### 6.3 Candidat boucle moteur — `sb_010_veh_cv_ghost.pck` (Ghost)

- Pack source : même dossier, `sb_010_veh_cv_ghost.pck` (1 448 965 octets, 248 entrées, tailles de 1,4 à 16 Ko).
- Entrée extraite : ID `146409670`, offset `172 032`, taille `16 013` octets (la plus grande du pack — candidat le plus proche d'un cycle de boucle, PAS confirmé bouclant : le drapeau `sLoopCount` n'a pas été lu, cf. §3.2).
- Métadonnées vgmstream : `48000 Hz, stereo, Custom Vorbis, Audiokinetic Wwise RIFF header, 144 kbps, 0,879 s`.
- Fichiers écrits : `scout_sons_preuve/veh_cv_ghost_146409670.wem` et `.wav` (168 844 octets, écoutable).

### 6.4 Ce que la preuve établit, et ce qu'elle n'établit pas

Établit : le format `.pck`/AKPK, le décodeur Vorbis Wwise et l'ensemble de la chaîne fonctionnent pour les véhicules et les tourelles **sans aucune différence** par rapport aux armes de joueur déjà traitées. Aucun obstacle de format nouveau.

N'établit PAS : l'identité exacte des deux clips au sens produit (« c'est LE bon tir », « c'est LA bonne boucle ») — cela demande l'écoute et le graphe de tags complets, hors périmètre d'une preuve minimale et explicitement laissé à la voie de désignation habituelle (§5.1 étape 5).
