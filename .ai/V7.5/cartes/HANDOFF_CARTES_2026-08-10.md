# Handoff — fonds de carte Halo (2026-08-10, fin de contexte)

> Worktree `LevelUp-wt-replay2d`, branche `feat/v75`, HEAD `7652fff83`. Aucun merge, `main`
> intact. Historique complet et chiffre : `HANDOFF_PORT_TRIANGLES_2026-08-08.md` §10 a §14.6.
> Reports et portes fermees : `.ai/V7.5/REGISTRE_REPORTS.md`.

## 0. LE POINT LE PLUS IMPORTANT — AUCUN ASSET N'EST PRODUIT

La chaine de rendu est validee sur 25 cartes natives et une carte Forge. **Rien n'est livre.**

- Les PNG existants sont des ARTEFACTS DE REVUE ecrits par des TESTS, via des variables
  d'environnement (`RENDU_PNG_CARTE`, `RENDU_PNG_FORGE`...), a des chemins arbitraires du
  Bureau. Aucun n'est versionne, aucun ne passe par `PathResolver`.
- `cmd/mapstruct-build` produit toujours l'ANCIEN format : du JSON de surfaces AABB. Il n'a
  jamais ete branche sur la chaine des triangles.
- Les etapes C1/C2/C3 du `PLAN_PORT_TRIANGLES_GO.md` §3 (« cuire les cartes ») sont TOUTES
  encore ouvertes.

**C'est le premier lot a traiter.** Tout le reste est de l'amelioration ; ca, c'est le livrable.

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

- `TestBancCliffhanger` — ASSERTE accord 66,7 % et positions jouees 93,82 % contre l'oracle des
  29 221 positions et la carte validee. C'est le gate de non-regression : toute modification qui
  le fait bouger est un echec.
- `TestBalayageCoquille` — 25 cartes : **0 ancre perdue, 0 coquille refusee**, decor retire de
  0 a 87,8 %.
- `TestRenduCarte` (RENDU_CARTE=<module>) et `TestRenduForgeVagabond` — oracle des ancres.

**Trois cartes validees a l'oeil par l'utilisateur** : Cliffhanger, Catalyst, Vagabond (Forge).

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
| **produire les assets** (§0) | jamais commence |
| 1 139 objets Forge sans modele `rtgo` (24 %) | saut `bloc`/`scen`/`mach` -> `hlmt` |
| toile `fo08_wetland` non rendue sous les objets | registre |
| `MeshResourcePackingPolicy` @186 du `rtgo` non porte | registre |
| seuil de grain 0,005 calibre sur UNE carte | registre |
| 9 modules absents de l'installation locale | non mesurables ici |
| `live_fire` : aucun tag sbsp | exception instruite §1 ter |

---

## 5. PROMPT POUR LA SUITE

```
LOT CARTES — produire les ASSETS, puis finir Forge

## Ou travailler

Worktree `C:\Users\Guillaume\Projects\LevelUp-wt-replay2d`, branche `feat/v75`, HEAD `7652fff83`.
JAMAIS sur `main`. Ne pas merger.

A lire avant d'agir, dans cet ordre : `CLAUDE.md` (Go uniquement, JAMAIS de Python, pas
d'emojis dans les fichiers versionnes), le skill `plan-execution` (deux lots = plan
multi-etapes), puis `.ai/V7.5/cartes/HANDOFF_CARTES_2026-08-10.md` — son §0, son §2 et son §3
en entier. Les sept criteres refutes du §3 ne se rejouent pas.

**PIEGE D'ENTREE** : copier `data/cache/replays/halo_infinite/000d5950.json` depuis
`C:\Users\Guillaume\Projects\LevelUp\data\...` — sans lui les sondes d'oracle se declarent
`SKIP` EN SILENCE.

## LOT A — LES ASSETS (c'est le livrable, tout le reste attend)

La chaine de rendu est validee mais **aucun fichier n'est produit** : les PNG existants sont
des artefacts de revue ecrits par des tests a des chemins arbitraires.

1. **Sortir la chaine des tests.** Le rendu d'une carte doit etre une fonction de PRODUCTION
   prenant (module, ancres) et rendant l'image — pas un `TestRenduCarte`. Les tests l'appellent
   ensuite, ils ne la portent plus.
2. **Brancher la sortie** sur `PathResolver` (`MapStructurePath` ou un chemin voisin a decider et
   a JUSTIFIER). Jamais de `filepath.Join(..., "data", ...)` a la main — regle projet.
3. **Decider le format et le PUBLIER dans le fichier** : PNG a quelle resolution, quel calage
   monde->pixel, fond transparent ou non. Le calage DOIT etre lisible par le consommateur, sinon
   l'image est inexploitable — c'est la lecon de `carte_validee_v1.png`, dont le calage
   (0,0920 m/px, X0 -43,5, Y1 61,0) a du etre retrouve a la main.
4. **Cuire les 25 cartes natives + Vagabond**, avec leur compte d'ancres, et un rapport.
5. **`internal/ooz` est en GPLv3** : rien de cette chaine ne doit etre linke par le serveur.
   L'app LIT l'asset fige, elle ne le fabrique pas. Verifier que la separation tient.

**Gate du lot A** : `TestBancCliffhanger` toujours vert (accord 66,7 %, positions 93,82 %) ·
`TestBalayageCoquille` 0 ancre perdue · `go build`, `go vet ./...`, `make go-api-lint` 0 issue ·
les assets produits, listes avec leur taille · gate visuel utilisateur sur 3 cartes au choix.

## LOT B — LES 1 139 OBJETS FORGE

Sur Vagabond, 3 558 des 4 697 objets placés se rendent (75,7 %) ; **1 139 n'ont pas de modele
`rtgo` direct** et passent par `bloc`/`scen`/`mach` -> `hlmt`. Methode qui a ferme F1 : le scan
des octets du tag contre l'index complet de l'installation (cf. handoff du 10/08 §13).

**Gate du lot B** : oracle des ancres sur Vagabond au moins aussi bon qu'aujourd'hui (4/4), part
d'objets rendus publiee avant/apres, gate visuel utilisateur.

## Discipline

Ordre strict : lot A CLOS avant d'ouvrir le lot B. A chaque cloture : entree
`.ai/thought_log.md`, mise a jour du handoff, tout report au `REGISTRE_REPORTS.md` avec sa
condition de reprise. Commits sur `feat/v75`, un par lot minimum, pousses. CI de branche verte
AU NIVEAU JOB avant de declarer un lot clos.

Ne pas lancer deux commandes Go concurrentes (cache corrompu), et ne pas travailler a deux
sessions dans ce worktree.

Reponds en francais.
```
