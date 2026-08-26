# Plan — revue carte par carte des fonds de rejeu 2D

> Ecrit le 2026-08-26, sur le go utilisateur. Branche `wt/cartes-revue-par-carte` (depuis
> `feat/v75`), **worktree PRINCIPAL obligatoire** : la cuisson exige `data/` et l'installation
> du jeu. Aucun autre agent dans ce worktree pendant le chantier.
> Contrat `plan-execution`. Registre : `REGISTRE_CARTES.md`.

## La decision qui fonde ce plan

**On renonce a une formule qui vaudrait pour toutes les cartes.** Les regles universelles
restent la BASE ; le reglage final est PAR CARTE. Pour que « par carte » ne devienne pas une
foret de branches dans le code, le reglage vit **en DONNEE** — une entree par carte, avec sa
raison ecrite et la date de son gate — jamais un `if` sur un nom de carte. Le precedent est
`CarteForge.FondFige` (Vagabond, Corpo), sanctionne au registre le 2026-08-13.

## Etat de l'art verifie sur pieces (2026-08-26)

1. **Trois cuissons de production, la derniere date du 13/08.** 10/08 20:34 (21 fonds, lot A) ;
   13/08 01:00-01:32 (natifs, lot toits — 9 changes, catalyst/ridgeline/behemoth identiques au
   bit) ; 13/08 12:16-13:46 (37 Forge sous cle map_id). Rien depuis.
2. **La cle des fonds Forge a change le 13/08** : de la cle lisible du canevas
   (`fo08_wetland.png`) vers le **map_id** (`105f5d84-....png`). Les natifs, eux, ont GARDE
   leur cle lisible (`catalyst.png`). C'est bien le nom intelligible qui a ete lache au profit
   de l'identifiant, et pour les Forge seulement.
3. **Le cadre n'est pas l'aire de jeu.** `CadreSurAncres` (`internal/himap/cuisson.go` L167) le
   pose a la boite des ancres d'objectifs elargie d'une CONSTANTE, `MargeCadre = 2 x
   PorteeAncre = 50 m` (L47). La coquille de mort (`RestreintALaCoquille`, active par defaut)
   efface ensuite tout ce qui tombe hors de la frontiere — **sans que le cadre soit
   recalcule**. D'ou des images largement transparentes.
4. **La mesure le confirme et separe les deux familles** (`cmd/mapfond-cadrage`, part de la
   largeur du cadre occupee par la matiere) : mediane **natifs 53,5 %**, **Forge 88,3 %**.
   Les pires natifs : sgh_blueprint 28,8 %, ctf_aquarius 33,7 %, forest 35,8 %, ctf_bazaar
   39,3 %, sgh_streets 40,6 %, **catalyst 50,0 %** (41,4 % en hauteur, 12,0 % en aire) —
   le match `530820e5` qui a declenche la remarque utilisateur est sur Catalyst.
   **Les deux familles ont le defaut INVERSE** : les natifs cadrent trop large sur du vide,
   les Forge remplissent le cadre de matiere non jouee (« bouillie », refus du 13/08).
5. **Le temoin dont l'utilisateur se souvient n'a jamais ete en production.**
   `Desktop/COULEUR_jeu_catalyst_COQUILLE.png` est date du **10/08 12:24**, soit AVANT
   `f78f2ebfa` (12:41, la coquille entre en production) et AVANT `7652fff83` (14:46, la
   frontiere devient un maillage teste par parite de rayon). Le `catalyst.png` publie
   (cuisson 10/08 20:34, inchange depuis) porte pourtant le meme `style: "jeu"`, le meme
   `boundaryApplied: true`, le meme `playLevelZ` et le meme `instancesDrawn` — **et les deux
   images ne coincident pas a l'oeil** : le temoin est nettement plus clair et son arene se
   lit mieux. L'ecart n'est pas explique ; il est mesure a la phase 2.
6. **Le correctif Forge existe, chiffre, et n'a jamais ete applique ni gate** : ecretage
   « arene par la reference » (`INVESTIGATION_BOUILLIE_FORGE_2026-08-13.md` §4-§6, diff en
   annexe A, non committe ; non-regression au bit prouvee sur les 21 fonds valides).

## Methode de suivi

- **Le registre** (`REGISTRE_CARTES.md`) est la source de verite : une ligne par fond, une
  ligne par carte sans fond, un statut par ligne, un journal des verdicts avec verbatim.
- **La planche** : un artefact web par lot d'environ 10 cartes, **avant / apres cote a cote**,
  avec sous chaque paire le nom, les matchs, la famille, l'occupation du cadre et les
  parametres appliques. L'utilisateur repond en UN message. Les cartes refusees ne rebouclent
  pas immediatement : elles entrent dans la file de reglage et reviennent dans la planche
  suivante.
- **Regle anti-derive** : toute cuisson qui modifie un PNG repasse sa ligne en `ATTENTE`. Un
  `VALIDEE` ne survit pas a une image changee.

## Phases

### Phase 0 — instruments et registre — FAITE (2026-08-26)

- [x] 0.1 Branche `wt/cartes-revue-par-carte` depuis `feat/v75`.
- [x] 0.2 `cmd/mapfond-cadrage` : mesure de l'occupation du cadre, fond par fond, hors ligne.
- [x] 0.3 `REGISTRE_CARTES.md` : 56 fonds + 44 cartes sans fond, statuts d'entree etablis
      sur pieces (14 valides sur 56).

### Phase 1 — le temoin Catalyst : d'ou vient l'ecart

- [ ] 1.1 Re-cuire Catalyst avec la chaine d'aujourd'hui vers un `--out-dir` scratch, et
      comparer au PNG publie : identique au bit ou non. Tranche la question « la chaine
      a-t-elle bouge depuis le 10/08 sans que le fond soit re-cuit ».
- [ ] 1.2 Reproduire le temoin du 10/08 12:24 (coquille d'alors + couleur d'alors) et
      diffuser les trois images cote a cote. Objectif : nommer l'ecart, pas le supposer.
- **Gate 1** : l'ecart est explique par une cause ecrite, ou la piste est declaree refutee.

### Phase 1 bis — le cadrage COTE REJEU — FAITE (2026-08-26)

Signale par l'utilisateur sur capture : la carte s'affiche minuscule et decentree dans
le cadre du rejeu. Cause etablie sur pieces : `sceneBounds` (`replayLogic.ts`) gonflait le
cadre avec `geometryBounds` — l'etendue des props Forge, qui **debordent de la zone
parcourue** (godoc de `ReplayDocument.GeometryBounds`) — MEME quand un fond de carte est
pose. Or `ReplayCanvas` ne dessine PAS les props dans ce cas : ils sont le `else if` du
fond. Le cadre etait donc dimensionne sur de la matiere INVISIBLE.

- [x] 1b.1 La regle deja ecrite pour `structure` (« avec un sol reconstruit, le cadre est la
      zone jouee ») est etendue au fond de carte : `sceneBounds(doc, hasMapImage)`.
- [x] 1b.2 Trois temoins dans `replayLogic.test.ts`, mordant prouve par mutation (retrait
      de `hasMapImage` -> 2 temoins rouges).
- [x] 1b.3 Gates : 74 fichiers / 1 127 tests vitest verts, `tsc -b` sur cache purge, ESLint 0,
      garde-rail de taille `ReplayCanvas.tsx` respecte SANS etre relache (741 <= 742).

Effet de bord utile : le cadre devenant la zone jouee, les marges vides des fonds natifs
(phase 2) sont rognees a l'affichage — `coversPlayedArea` garantit que l'image couvre la
zone. La phase 2 reste due : elle corrige l'ASSET, pas seulement sa vue.

### Phase 2 — le cadrage, correctif universel

- [ ] 2.1 Re-cadrer APRES la coquille : le cadre publie devient la boite de la matiere
      retenue, plus une marge. La marge est un chiffre unique, justifie, pas un reglage par
      carte.
- [ ] 2.2 Second instrument : occupation mesuree contre les POSITIONS JOUEES (oracle des
      29 221 positions), pas seulement contre la matiere dessinee — une carte dont le decor
      remplit le cadre passe le premier test et rate le second.
- [ ] 2.3 Re-cuisson des 56 fonds, diff explique cle par cle, calage sidecar re-publie.
- **Gate 2** : occupation mediane des natifs > 80 % en largeur, aucune ancre perdue,
  planche de 7 temoins soumise.

### Phase 3 — la base Forge

- [ ] 3.1 Appliquer l'ecretage « arene par la reference » (annexe A du 13/08), re-verifier la
      non-regression au bit sur les fonds valides, promouvoir ou supprimer la sonde brouillon.
- [ ] 3.2 Re-cuire les 35 Forge non geles, cadrage de la phase 2 inclus.
- **Gate 3** : planche des 7 temoins du 13/08 (domicile, goliath, dynasty, smallhalla,
  the_pit, starboard, dredge), verdict utilisateur.

### Phase 4 — le gate carte par carte

- [ ] 4.1 Paquets d'environ 10 cartes, **ordre des matchs decroissants** : les 19 natifs
      d'abord (1 176 matchs, 61 % du corpus), puis les 37 Forge.
- [ ] 4.2 Chaque refus ouvre une entree de reglage EN DONNEE (parametres + raison + date),
      jamais une branche de code. Garde-rail : aucune entree sans raison ni date.
- [ ] 4.3 Registre tenu a jour a chaque verdict, verbatim inclus.

### Phase 5 — les 44 cartes sans fond

- [ ] 5.1 Reliquat Forge (39 map_id) : la chaine s'applique telle quelle.
- [ ] 5.2 Les trois blocages nommes, instruits separement : Live Fire (71 matchs, natif sans
      tag sbsp), Detachment (25), Argyle (22).
- [ ] 5.3 Statuer `HORS PERIMETRE` avec raison ecrite ce qui ne sera pas cuit.

## Discipline

Entree `.ai/thought_log.md` a chaque cloture de phase ; tout report au `REGISTRE_REPORTS.md`
avec sa condition de reprise ; jamais deux commandes Go concurrentes (cache corrompu) ; les
cuissons durent 25 a 30 min, lancees en fond, rien d'autre en Go pendant.
