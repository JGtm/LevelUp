# Audit du contenu tiers suivi dans le dépôt — 2026-10-16

Contexte : passage MIT vers AGPL-3.0 (code) et CC BY 4.0 (docs). La licence ne couvre que ce
que l'auteur a écrit ; ce document recense ce qui, dans l'arbre suivi, appartient à Microsoft ou
à des joueurs. `NOTICE` §4 le déclare hors licence. Rien n'a été supprimé : retirer du contenu
d'un dépôt public est une décision de l'utilisateur, et seule une purge de l'historique
retire ce qui a déjà été publié.

## Inventaire (arbre au 2026-10-16)

| Catégorie | Emplacement | Volume | Nature |
|---|---|---|---|
| Sortie de décompilation du jeu (RETIRÉE le 2026-10-16) | copie hors dépôt : `Downloads/LevelUp_decompilation_Ghidra`, motif ignoré par `.gitignore` | 273 fichiers `.c` | Dérivé direct du binaire Microsoft : retiré de l'arbre courant, reste dans l'historique |
| Captures mémoire du jeu | `.ai/V7.5/dumps/` (`ce_run2_cliffhanger.bin.gz` 8 Mo, `ce_capture_delta.csv` 19 Mo, trajectoires, rosetta), `tools/ce/*.bin` | ~60 Mo | Contenu mémoire du jeu en cours d'exécution |
| Audio du jeu | `static/sounds/halo_infinite` | 182 wav, 83 Mo | Servi par l'application ; GARDÉ (décision du 2026-10-16 : le rejeu en a besoin) |
| Images du jeu | `static/{medals,maps,ranks,commendations,weapons-assets,vehicles-assets,abilities-assets,prestige-assets}`, `apps/web/public/titles/halo_5/spartan/{emblems,nameplates}*`, `data/titles/*/reference/map_backgrounds`, `data/cache/career-rank-image` | plus de 2 000 png/webp | Servies par l'application ; certaines viennent de l'API de contenu officielle |
| Référentiel prébuilt | `data/titles/halo_infinite/warehouse/metadata-prebuilt.zip` | 5 Mo | Données de l'API |
| Films de matchs réels (fixtures) | `**/testdata/**` (film, replay, killsource, objectifs), `apps/web/src/features/match-replay/test/fixtures/go`, `.ai/V7.5/replay2d/registre_film/lotBP` | ~100 fichiers `.bin(.gz)` | Contenu Microsoft + gamertags de joueurs réels |

## Risques, du plus au moins exposé

1. **Décompilation et captures mémoire** : aucune tolérance de type « contenu de fan » ne les
   couvre ; une demande de retrait par Microsoft est plausible. Les faits qu'elles ont permis de
   découvrir sont libres et déjà écrits dans les docs de recherche ; les fichiers eux-mêmes ne
   sont pas nécessaires à la compilation ni à l'exécution.
2. **Fixtures de films** : nécessaires aux tests du décodeur ; contiennent des gamertags de
   joueurs tiers dans un dépôt public (angle RGPD, en plus du droit d'auteur).
3. **Assets servis** (images, sons) : l'application en a besoin ; c'est l'usage « fan content »
   des règles de Microsoft pour le contenu de jeu (non commercial, mention de non-affiliation,
   déjà au pied de page). Le volume audio (83 Mo) est le plus lourd à justifier.

## Options (aucune appliquée)

- **A. Minimale** : garder tel quel, `NOTICE` §4 déclare la situation et offre le retrait sur
  demande. Ne règle ni le risque 1 ni le risque 2.
- **B. Recommandée** : retirer de l'arbre courant les `.c` Ghidra et les captures mémoire
  (`git rm`, aucune conséquence sur le build), pseudonymiser ou anonymiser les gamertags des
  fixtures quand c'est possible. Les docs de recherche en prose restent.
- **C. Complète** : B + purge de l'historique (`git filter-repo`), ce qui réécrit tous les
  hashs publiés, casse les branches et worktrees ouverts, et demande un push forcé. À ne faire
  qu'à froid, après sauvegarde, et jamais sans l'accord explicite de l'utilisateur.

## Autres constats

- Auteurs : le dépôt contient aussi 47 commits d'un bot Copilot et 17 de dependabot (code généré
  ou trivial). Des commits d'une adresse non personnelle ont été repérés ; leur nettoyage
  (réécriture d'historique) est une décision distincte, hors de ce lot.
- Dépendances : 161 modules Go et 121 paquets npm de production examinés (licences de leurs
  fichiers). Aucun GPL/AGPL/SSPL. Un seul copyleft faible : `hashicorp/golang-lru/v2` et un
  paquet npm en MPL-2.0, compatibles AGPL-3.0.
- Tag de licence choisi : `AGPL-3.0-only` (pas `-or-later`) pour garder la main sur les versions
  futures de la licence.
