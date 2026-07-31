# HANDOFF — le POC du rejeu 2D dans l'app

> Écrit le 2026-07-29, branche `feat/filmdec-continuation`.
> Documents qui font foi : `V7.5/replay2d/PLAN_POC_DANS_L_APP.md` (le plan et ses mesures),
> `SUIVI_REPLAY_2D.md` (l'avancement global), `thought_log.md` (les décisions datées).
> Ce fichier ne répète aucun des trois : il dit **où reprendre, et ce qui bloque**.

---

## OÙ EN EST LE REJEU

Le POC était un poste de travail à quatre colonnes ; la production n'en avait qu'une. Elle en a
maintenant trois — la carte, le fil des joueurs, les fiches d'équipe — et il en manque une, le
fil des éliminations.

| ce que le POC montre | dans l'app ? | où |
|---|---|---|
| sol de carte reconstruit | **oui** | `mapFloor.ts` + `drawFloorLayer` |
| cône de visée, bouclier, étages, apparition, mort | **oui** | `replayMarkers.ts` |
| trajectoires de projectile | **oui** | `drawProjectilesLayer` |
| identité des joueurs (nom, équipe) | **oui** | `rosterLogic.ts` + roster de l'artefact |
| fiches d'équipe : K/D/A, bouclier, armes, réapparition | **oui** | `ReplayTeams.tsx` |
| inventaire : grenades, capacité, munitions | **oui** | `inventory_decode.go` → `ReplayTeams` |
| effets de tir par famille d'arme | **oui** (formes, pas teintes) | `shotEffects.ts` |
| **fil des éliminations** | **non** | bloqué, voir plus bas |
| **médailles en images** | **non** | dépend du fil |
| zones nommées, objectifs, dispositifs de carte | non | jamais commencé |

---

## CE QUI BLOQUE, ET LA DÉCISION QUI ATTEND

**Le fil des éliminations n'est pas décodable ici sans une décision de branche.**

Il l'est chez le voisin : `feat/filmdec-killweapon`, paquet `killsource`, qui rend par mort la
victime, le tueur crédité, l'assistant, les deux parts de dégâts, la source du dégât et le kill
par véhicule. C'est **sa source de vérité** (arbitré le 2026-07-29) et il y évolue.

Trois faits mesurés qui commandent la suite :

1. `feat/filmdec-killweapon` **n'a aucun ancêtre commun avec `main`**.
   `feat/filmdec-continuation` en a un (`811be64ec`). **Une seule des deux peut être livrée.**
2. `killsource` ne compile pas contre notre `filmdec` : six symboles manquent
   (`KillSourceHealth`, `SetStrictGeneration`, `SetMobilityActionBodyPorted`,
   `World.GenerationMatches`, `World.BindWildcard`).
3. Les deux branches ont fait diverger `filmdec` : **36 fichiers, +1 472 / −2 972**, avec des
   suppressions **des deux côtés**. Les sept fichiers dont le rejeu dépend (`fire_events`,
   `grenade_events`, `keyframe_loadout`, `map_bounds`, `vitality`, `i0_layout`, `capture`)
   **n'existent pas** chez le voisin.

Le rapprochement est donc un chantier de décodeur, **symétrique quel que soit son sens**. Seule
la direction « faire venir `killsource` vers la branche connectée à `main` » débouche sur une
livraison. L'utilisateur avait proposé l'inverse ; la mesure ci-dessus est la raison de ne pas
le suivre, et la décision lui appartient.

**Quand elle sera prise**, le branchement est prévu : le fil entre par une **entrée de données**
(`Options.Kills`), comme `Deaths` ou `Loadouts`, et un adaptateur écrit à part connaît les deux
formes. Aucun type `Kill` n'est déclaré tant qu'aucun producteur ne l'alimente.

---

## CE QUI N'A PAS ÉTÉ VÉRIFIÉ, ET QUI DOIT L'ÊTRE

**Aucun gate visuel n'est déclaré franchi.** La revue écran est restée à la charge de
l'utilisateur (le pilotage navigateur n'était pas exploitable depuis cette session). Tout ce qui
est décrit ici est vérifié par tests et par mesure sur l'artefact, **jamais par l'œil**.

Deux points pratiques pour cette revue :
- les serveurs qui tournaient sur `:8000` et `:5173` venaient du **dépôt principal**, pas de ce
  worktree ;
- le rejeu n'est servi **qu'aux appels venus de la machine locale** (`replay_local_gate.go`), et
  ce garde porte sa date de bascule, sa cible de retrait et son critère mesurable.

---

## RÉSERVES OUVERTES — à ne pas perdre

### La cellule de munitions *k* n'est pas l'arme *k*

Mesuré sur 300 appariements : la correspondance est parfaite pour 15 armes sur 22, et **échoue**
sur le Gravity Hammer (13 chargeur / 17 jauge / 8 aucune), l'Energy Sword (3/5/3), le Stalker
Rifle et le Ravager. Le marteau et l'épée sont précisément les deux armes que le décodeur
documente comme n'émettant *ni chargeur ni jauge*.

L'écran affiche donc le **numéro d'emplacement du record** et porte la réserve en infobulle.
**Non tranché** : est-ce l'ORDRE des emplacements qui diffère de celui du loadout, ou le parse
qui dérive quand une arme n'émet rien ? Les deux expliqueraient la mesure. C'est une question de
décodage, pas d'affichage.

### `ownersFromLives` tranche, alors que ce chantier a supprimé les arbitrages

Sur collision de slot, la première identité gagne et la contradictoire est écartée. Un test
l'exige explicitement, donc c'est délibéré — mais c'est bien un choix, dans un paquet qui a par
ailleurs supprimé le vote. 0 collision sur le film de référence. À reposer sur le prochain film.

### Les effets de tir portent la forme, pas la teinte

Le POC distinguait sept familles par sept teintes `hsla` en dur. Ici la couleur appartient au
**tireur** (c'est elle qui permet de le suivre des yeux) et les couleurs sémantiques passent par
des tokens. La famille est donc portée par la **forme** de l'effet. Si un jour on veut la teinte
en plus, il faudra sept tokens, pas sept littéraux.

### Le fond de carte n'a pas d'emprises orientées

0 `poly` sur 10 223 pour `ridgeline` : les fichiers de structure figés sont antérieurs au champ.
Les produire exige les fichiers du jeu installé (`cmd/mapstruct-build`). Sans elles, la boîte
alignée d'une pièce tournée déborde de la pièce — le POC donnait déjà ce mode comme une option,
pas comme le défaut.

---

## PIÈGES RENCONTRÉS, POUR NE PAS LES REFAIRE

- **`git stash` est interdit par le dépôt** (commit WIP à la place). Utilisé une fois par
  erreur pour comparer un état antérieur ; rien n'a été perdu, mais la pile de stash de
  l'utilisateur contenait déjà trois entrées et le risque était réel.
- **La syntaxe here-string PowerShell (`@'…'@`) ne marche pas dans l'outil Bash** : elle laisse
  un `@` parasite en tête et en pied du message de commit. Utiliser un heredoc, ou `-F fichier`.
- **`/tmp` n'existe pas** dans cet environnement : écrire dans le répertoire de scratch.
- **Le pré-push a deux gates faciles à oublier** : le ratchet de code mort (`knip`, un export
  non consommé suffit à bloquer) et `govulncheck`, qui **charge tout le module** — donc un
  seul `cmd/tmp_*` qui ne compile plus bloque le push de tout le monde.

---

## COMMANDES UTILES

```bash
# Reconstruire l'artefact (depuis apps/go-api ; le film vit dans le depot PRINCIPAL)
CGO_ENABLED=0 LEVELUP_REPO_ROOT=<ce-worktree> \
  go run ./cmd/replay-build --map Cliffhanger 000d5950 <depot-principal>/data/cache/film_chunks/000d5950

# Verifications
cd apps/go-api && CGO_ENABLED=0 go test ./internal/analysis/replay/...
make check-types && cd apps/web && npx vitest run src/features/match-replay/
node tools/knip-ratchet.mjs                      # ratchet de code mort (bloque le push)
bash scripts/git-hooks/lefthook/govulncheck.sh   # charge TOUT le module Go
```

---

## PROTOCOLE DE REPRISE

1. Lire `V7.5/replay2d/PLAN_POC_DANS_L_APP.md` — les étapes closes y sont statuées, les mesures y sont.
2. Trancher la question de branche ci-dessus. **Rien du fil des éliminations ne peut avancer
   avant.**
3. Sinon, la suite non bloquée est : zones nommées et objectifs de carte (rang 4 du
   `SUIVI_REPLAY_2D.md`, qui exige une règle valable pour les 30 cartes, pas seulement
   Cliffhanger).
