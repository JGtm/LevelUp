# Handoff — fonds de carte Halo (2026-08-10, fin de contexte)

> Worktree `LevelUp-wt-replay2d`, branche `feat/v75`, HEAD `5b5b0d48a`. Aucun merge, `main`
> intact. Historique complet et chiffre : `HANDOFF_PORT_TRIANGLES_2026-08-08.md` §10 a §14.6.
> Reports et portes fermees : `.ai/V7.5/REGISTRE_REPORTS.md`.

## 0. LOT A FAIT — LES ASSETS SONT PRODUITS (mise a jour du 2026-08-10, seconde session)

> Ce paragraphe disait « AUCUN ASSET N'EST PRODUIT ». Ce n'est plus vrai, et c'etait le
> livrable. Ce qui suit le remplace.

**21 assets produits** — 19 cartes natives cuites dans un module + Vagabond et Corpo (Forge) :
`data/titles/halo_infinite/reference/map_backgrounds/{cle}.png` et son sidecar `{cle}.json`.
**7,70 Mio au total, 413/448 ancres avec sol, style `jeu`, 0,0920 m/px, fond transparent.**
Rapport chiffre carte par carte : `RAPPORT_CUISSON_FONDS_2026-08-10.md`. Compte rendu pour
l'orchestrateur : `COMPTE_RENDU_LOT_A_2026-08-10.md`.

Ou vit quoi :

| brique | ou |
|---|---|
| chaine native (cadre, tranche, grain, frontiere, eau, oracle) | `internal/himap/cuisson.go` |
| chaine Forge (objets `.mvar` -> `food` -> `rtgo`) | `internal/himap/cuisson_forge.go` |
| format publie, styles, calage monde->pixel | `internal/himap/fond_png.go` |
| sidecar de calage, cote CONSOMMATEUR (GPL-propre) | `internal/analysis/replay/map_background.go` |
| chemins | `PathResolver.MapBackgroundDir/Path/MetaPath` |
| orchestration + rapport | `cmd/mapfond-build` |

**Les « cartes » du balayage sont des ENTREES DE CATALOGUE, pas des cartes.** `aquarius_map` et
`aquarius_-_ranked_map` designent le meme dossier d'installation. La cle d'un asset est le
DOSSIER INSTALLE, et les entrees qui s'y rattachent versent l'UNION dedupliquee de leurs ancres.

**Portee** : 31 dossiers installes, dont 2 tutoriels, 1 PvE et 8 canevas Forge — soit **20
cartes multijoueur natives**, dont 19 cuites (`sgh_interlock` n'a aucun tag sbsp). Le gisement
suivant est ailleurs : **199 `.mvar` au dump, deux branches** (Vagabond, Corpo).

**GATE VISUEL TENU** (verdict complet au §3 du plan). Le style `jeu` est valide de fait. La
TRANCHE a ete basculee en relative sur decision utilisateur — Chasm et Highpower etaient
inexploitables ; le banc est RE-BASE par ecrit a accord 64,7 % / positions 93,95 %.

**CE QUE LE GATE A OUVERT, et qu'aucun chiffre ne disait** :

1. **Les TOITS.** Sur Illusion, Prism et Aquarius, « la forme globale est correcte mais on ne
   voit que les toits ou plafonds ». Le z-buffer garde la surface la plus HAUTE ; sur une arene
   couverte c'est le plafond. Forbidden en souffre partiellement — legitime, elle a une partie
   a ciel ouvert. **L'ecart median aux ancres NE PREDIT PAS ce defaut** : verifie et ecarte
   (Behemoth -12,73 m est nickel, Forbidden -0,35 m est touche).
2. **Streets et Bazaar « un peu rudimentaires »**, cause non identifiee.

## 1. Ce qui EST acquis, avec ses chiffres

La chaine complete, **sans aucun reglage par carte** :

| etape | regle | d'ou elle vient |
|---|---|---|
| cadre | voisinage des ancres d'objectifs (`margeCadre`) | `map_objectives.json` |
| tranche | `[TrancheDeJeuMin ; TrancheDeJeuMax]` = [-12 ; +28], translatee au sol des ancres | prototype `s31_raster.py` |
| decor | grain du maillage, `AireMaxTriangleJouable` = 0,005 m2 | mesure, valide utilisateur |
| frontiere | maillage `sddt` par PARITE DE RAYON (`Sddt.ContientFrontiere`) | tag de la carte |
| eau | volumes `sddt` (`Rendu.PoseEau`) | tag de la carte |
| couleur | ecart au NIVEAU JOUE (`TeinteNiveauDeJeu`, style `jeu`) | ancres |

**Gates en place, tous verts** :

- `TestBancCliffhanger` — ASSERTE accord **64,7 %** et positions jouees **93,95 %** contre
  l'oracle des 29 221 positions et la carte validee. Seuils RE-BASES le 2026-08-10 avec la
  bascule de la tranche (cf. §0) ; le banc applique desormais `TrancheDeJeu`, la MEME fonction
  que la production.
- `TestBalayageCoquille` — **0 ancre perdue** ; l'assertion a ete AJOUTEE le 2026-08-10, ce test
  ne faisait que journaliser et rendait `ok` quoi qu'il arrive.
- `TestRenduCarte` (RENDU_CARTE=<module>) et `TestRenduForgeVagabond` — oracle des ancres.
- **Determinisme** : deux cuissons independantes rendent des PNG identiques au bit.

**ATTENTION — L'ORACLE DES ANCRES NE JUGE PAS L'ASPECT.** Il dit « y a-t-il du sol sous les
points d'objectif », rien de plus. La bascule de la tranche a change l'image de Catalyst
(+11,5 % de matiere) en laissant ses ancres a 24/24 : l'oracle a dit « inchange » et
l'utilisateur a vu une regression. Toute modification de la regle de rendu passe par un artefact
VISUEL sur au moins Cliffhanger ET Catalyst.

## 2. Les pieges qui ont coute le plus, a ne pas rejouer

1. **Un temoin qui ne departage pas ne teste rien.** Cinq occurrences dans ce chantier. Toute
   mutation annoncee dans un commentaire doit etre JOUEE : casser, voir rouge, revert, voir vert.
2. **Un oracle absent passe au vert.** `data/cache/replays/halo_infinite/000d5950.json` n'est
   PAS dans le worktree (`data/` ignore par git) : les sondes s'y declarent SKIP en silence. Le
   copier depuis `C:\Users\Guillaume\Projects\LevelUp\data\...` avant toute mesure d'oracle.
3. **Un oracle faible ne voit pas ce qu'un oracle fort detecte.** La coquille semblait gratuite
   sur Catalyst (19 ancres) et coutait 10 points de positions sur Cliffhanger (29 221 positions).
4. **Un NO-GO se mesure la ou l'hypothese a une chance.** L'hypothese « environnement ferme » a
   ete refutee sur Cliffhanger — une arene A CIEL OUVERT, le pire cas pour elle.
5. **« Cause presumee » ne dispense pas de mesurer.** La cause du bug de la coquille etait
   annoncee « plusieurs volumes a unir » : faux, une sonde de trois minutes l'a defaite.
6. **Deux sessions dans un meme worktree se marchent dessus.** Meme paquet Go = meme espace de
   noms. Incident survenu, a ne pas reproduire.
7. **Un `t.Fatal` dans une boucle de balayage tue tout le balayage.** Un sous-test `t.Run` par
   carte.

## 3. Sept criteres de zone jouable REFUTES — chiffres au registre, ne pas les rejouer

module d'origine · emprise bornee aux ancres · tranche d'altitude seule · grain seul (vaut sur
la roche, muet sur le construit) · collision (`instanced physics instances` : le decor porte
aussi la collision) · accessibilite pietonne (perd des positions jouees) · cloture par
inondation exterieure (Cliffhanger n'est pas close).

**Ce qui marche** : la frontiere declaree par la carte (`sddt`), testee A L'ALTITUDE DE JEU.

## 4. Ce qui reste ouvert

| item | ou |
|---|---|
| ~~produire les assets~~ | **FAIT le 2026-08-10, cf. §0** |
| ~~GATE VISUEL utilisateur~~ | **TENU** — verdict complet au §3 du plan |
| ~~TRANCHE ABSOLUE~~ | **TRANCHEE** : relative au sol joue, banc re-base |
| ~~3 cartes « non installees » a tort~~ | **CORRIGE** : Deadlock, Oasis, Scarr + Corpo (Forge) |
| ~~LES TOITS : Illusion, Prism, Aquarius rendent le plafond~~ | **CORRIGE le 2026-08-13** (lot toits) : voie de reference branchee sur le rendu (`rendu_reference.go`) — une carte dont >1/3 de la matiere cache un sol praticable est COUVERTE et montre, dans la portee des ancres, la surface la plus proche du sol de reference. 9 cartes couvertes re-cuites (illusion, crystalcaves, aquarius, forbidden, chasm + btb_engine, va_launchsite, fo08_wetland, fo11_blank) ; ridgeline, catalyst, behemoth IDENTIQUES AU BIT ; banc inchange (64,7 % / 93,95 %). Mesures et pistes refutees : `INVESTIGATION_TOITS_2026-08-13.md`. GATE VISUEL utilisateur PENDANT (Bureau, gate_cartes_v75/toits/) |
| ~~Chasm et Forbidden toujours juges non satisfaisants~~ | couvertes par le lot toits (37,2 % et 35,1 %) — a re-juger au meme gate |
| **Regression visuelle sur Catalyst** (+11,5 % de matiere) | non detectee : l'oracle des ancres ne juge pas l'aspect |
| **Regles generales contre regles par carte** | question utilisateur ouverte, cf. compte rendu |
| **T5 : appariement sommets/indices jamais MESURE** | invariant applique, contraste jamais joue |
| 1 113 objets Forge sans modele `rtgo` (23,6 %) | saut `bloc`/`scen`/`mach` -> `hlmt` — LOT B |
| 199 `.mvar` au dump, DEUX branches | le gisement suivant |
| toile `fo08_wetland` non rendue sous les objets | registre |
| aucun consommateur ne lit encore `map_backgrounds/` | chantier d'affichage |
| `MeshResourcePackingPolicy` @186 du `rtgo` non porte | registre |
| seuil de grain 0,005 calibre sur UNE carte | registre |
| `live_fire` (`sgh_interlock`) : aucun tag sbsp | exception instruite §1 ter, desormais SENTINELLE `himap.ErrAucunTagSbsp` — signalee « non cuisinable », elle ne fait plus echouer la cuisson |

---

## 5. PROMPT POUR LA SUITE

```
LOT B — les 1 113 objets Forge sans modele, puis la toile du canevas

## Ou travailler

Worktree `C:\Users\Guillaume\Projects\LevelUp-wt-replay2d`, branche `feat/v75`. JAMAIS sur
`main`. Ne pas merger.

A lire avant d'agir, dans cet ordre : `CLAUDE.md` (Go uniquement, JAMAIS de Python, pas
d'emojis dans les fichiers versionnes), le skill `plan-execution`, puis
`.ai/V7.5/cartes/HANDOFF_CARTES_2026-08-10.md` — son §0 (ce qui EXISTE maintenant), son §2 et
son §3 en entier. Les sept criteres refutes du §3 ne se rejouent pas.

**PIEGE D'ENTREE** : verifier que `data/cache/replays/halo_infinite/000d5950.json` est present
dans le worktree — sans lui les sondes d'oracle se declarent `SKIP` EN SILENCE.

## D'ABORD : LES TOITS — le defaut de rendu qui reste

Verdict utilisateur au gate visuel du 2026-08-10 : sur Illusion, Prism et Aquarius, « la forme
globale est correcte mais on ne voit que les toits ou plafonds ». Chasm et Forbidden restent
juges non satisfaisants apres la bascule de la tranche. Cause : le z-buffer garde la surface la
plus HAUTE, et sur une arene couverte c'est le plafond.

PISTE : `SurfaceReference` / `CarteParReference` (`reference.go`) retiennent deja « la bande
praticable la plus PROCHE du sol de reference » au lieu de « la plus haute ». Elles sont ecrites
pour le `Volume` et n'ont jamais ete branchees sur le `Rendu`.

DEUX GARDE-FOUS AVANT DE TOUCHER A CA :

1. **Mesurer sur l'oracle FORT d'abord.** Sur Cliffhanger, les rochers hauts font partie de
   l'identite de la carte et l'utilisateur les valide ; les remplacer par ce qui est dessous
   changerait une carte jugee nickel.
2. **L'oracle des ancres NE JUGE PAS L'ASPECT.** Il a dit « Catalyst inchange, 24/24 » pendant
   que l'image gagnait 11,5 % de matiere, et l'utilisateur a vu la regression. Produire un
   artefact VISUEL sur Cliffhanger ET Catalyst avant/apres, systematiquement.

PISTE REFUTEE, ne pas rejouer : l'ecart median aux ancres ne predit PAS ce defaut (Behemoth
-12,73 m est nickel, Forbidden -0,35 m est touche).

QUESTION UTILISATEUR OUVERTE : faut-il des regles GENERALES et des regles PAR CARTE ? Position
argumentee au compte rendu — une seule carte possede un oracle fort, donc un reglage par carte
ne se valide qu'a l'oeil, avec ~100 cartes Forge derriere. Ce qui reste defendable est une
exception declaree en DONNEE (table + raison ecrite + oracle), jamais une branche dans le code.

## LOT B — LES 1 113 OBJETS FORGE

Sur Vagabond, 3 558 des 4 709 objets se rendent (75,6 %) ; **1 113 n'ont pas de modele `rtgo`
direct** et passent par `bloc`/`scen`/`mach` -> `hlmt`. Methode qui a ferme F1 : le scan des
octets du tag contre l'index complet de l'installation (§13 du handoff du 08/08).

La chaine est en PRODUCTION (`internal/himap/cuisson_forge.go`) : le saut supplementaire se
branche dans `rtgoParType`, et le gain se mesure en re-cuisant `--maps fo08_wetland`.

**Gate du lot B** : oracle des ancres sur Vagabond au moins aussi bon qu'aujourd'hui (4/4), part
d'objets rendus publiee AVANT/APRES (75,6 % est le point de depart), gate visuel utilisateur.

## Si le lot B se ferme : la toile du canevas

`fo08_wetland` porte 13 830 instances de bsp qui ne sont PAS dessinees — la carte Forge est
faite des seuls objets du `.mvar`. Rendre le bsp du canevas SOUS les objets est la chaine
native, deja en production : `peupleDepuisModule` puis `poseObjetsForge` sur le meme rendu.
Decision au gate visuel : si le sol nature manque a l'oeil, le poser.

## Discipline

A chaque cloture : entree `.ai/thought_log.md`, mise a jour du handoff, tout report au
`REGISTRE_REPORTS.md` avec sa condition de reprise. Commits sur `feat/v75`, pousses. CI de
branche verte AU NIVEAU JOB avant de declarer un lot clos.

Ne pas lancer deux commandes Go concurrentes (cache corrompu), et ne pas travailler a deux
sessions dans ce worktree. Les cuissons et balayages durent 25 a 30 min : les lancer en fond et
ne PAS lancer autre chose en Go pendant.

Reponds en francais.
```
