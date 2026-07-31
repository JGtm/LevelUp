# HANDOFF — reprise du chantier killsource

> Ecrit le 2026-07-31 en fin de session. **Point d entree unique** : lis ce fichier en premier, il
> renvoie vers les trois autres.

## LES TROIS BRANCHES, ET LAQUELLE EST VIVANTE

| Branche | Role | On y travaille ? |
|---|---|---|
| **`feat/killsource-prod`** | **LA BRANCHE VIVANTE.** Partie d un `main` a jour (`6c2b00402`, 2026-07-31). Porte le decodeur, `filmdec`, `damagetag`, le CLI et la documentation. | **OUI, tout le travail neuf** |
| `feat/filmdec-killweapon` | ARCHIVE de recherche. 321 commits, les 67 outils de mesure que le journal cite comme commandes de reproduction, et le code d app qu on a decide de REECRIRE plutot que de porter. | non — lecture seule |
| `feat/filmdec-continuation` | L autre chantier (rejeu 2D, positions, loadout). Poussee, intacte. | non — chantier voisin |

### LE PIEGE A CONNAITRE : LE JOURNAL EST DESORMAIS DUPLIQUE

`.ai/RE_LOG_KILLWEAPON.md` existe **sur les deux branches**. Il est APPEND-ONLY, donc deux
ajouts paralleles ne se signalent pas : ils divergent en silence.

**REGLE : a partir du 2026-07-31, l exemplaire de `feat/killsource-prod` fait foi.** Toute section
neuve y va. L exemplaire de `filmdec-killweapon` est GELE — on le lit, on n y ecrit plus.
Derniere section commune aux deux : **`7ter.104`**. La prochaine libre est `7ter.105`.

## CE QUI EST FAIT

Le decodeur rend, pour chaque mort : victime, tueur credite, **source du degat fatal**, categorie,
**assistant**, et **les deux parts de degats**. 380 morts sur 380 de l API sur les quatre films de
reference. **34 confrontations Theater sans une seule etiquette publiee fausse**, plus, ce jour :
7/7 sur les kills au vehicule, 5/5 en BTB, l ecrasement confirme par le binaire, et les parts de
degats validees par le deroule d un combat (1/61 — l assistant fait tout le travail, le credite
acheve au sniper).

Le portage sur `main` s est fait **sans une ligne adaptee** : `filmdec` n existe pas sur `main`, la
derive redoutee d `internal/analysis` n a pas touche les 4 symboles utilises. Suite complete verte
en 508 s, 30 ancres et goldens intacts.

## CE QUI RESTE, DANS L ORDRE

### 1. Verifier la CI du scan de secrets — A FAIRE EN PREMIER

Quatre exceptions ont ete posees sur `filmdec-killweapon` (`.gitleaks.toml`), **CIBLEES PAR
EMPREINTE** et non par chemin, pour des faux positifs de documentation (des gamertags lus comme des
cles). **Je n ai pas pu verifier qu elles passent au vert.** Si le format d empreinte ne convient
pas a gitleaks 8.30.1, c est a reprendre — et a reporter sur `killsource-prod`, qui n a PAS ces
exceptions.

### 2. REECRIRE le branchement contre le `main` du jour — ne PAS le porter

Decision de l utilisateur, et les chiffres la soutiennent : `internal/migration` a recu 117 commits
et `internal/sync` 172 depuis la base commune. Ce qui est a reecrire :

- le pont de telechargement (`killsource_bridge.go`)
- les deux migrations (`match_kill_events`, `match_weapon_shots`)
- les deux persisters (INSERT-only, ADR 0019/0030)
- **le correctif d indice du tueur** — il repare un bug VIVANT sur `main` : le pipeline d armes y
  fait **moins bien qu un tirage au sort** (22,3 % contre 22,7 % pour une permutation aleatoire,
  76,4 % apres correction, McNemar z = 65,88, 116 films sur 116)

**Leur conception est dans `PLAN_BRANCHEMENT_KILLSOURCE.md`, leurs mesures dans le journal.** Le
diff d origine est sur la branche d archive si besoin de le relire, mais il ne se cherry-picke pas.

### 3. Puis le collecteur, la bascule des 8 lecteurs, le backfill

Tout est detaille dans `PLAN_BRANCHEMENT_KILLSOURCE.md`, avec ses gates par phase. Le gain final :
`killer_victim_pairs` porte **46,8 % de doublons** et gonfle les agregats carriere d un facteur
**1,879** — sur un duo reel, l ecran affiche **101 frags la ou il y en a 29**.

### 4. Parkes, sans dependance

- `HANDOFF_PRECISION_PROJECTILES.md` — rendre la precision universelle. Lot CIBLE : deux verrous
  mesures, **une seule piste nommee** (que porte l enregistrement qui CREE le slot du code 7).
- `PLAN_RECHERCHE_ASSETS_ICONES.md` — les icones d armes pour l interface. Aucune dependance, peut
  echouer sans dette.

## CE QUI ATTEND L UTILISATEUR

- Tester le decodeur sur d autres matchs (`go run ./cmd/killsource kills <film> -cache <...>/data/cache`
  — **le drapeau attend la racine du cache, il ajoute `film_chunks` lui-meme**). Signaler : une arme
  NOMMEE qui serait fausse, une divergence `<>` fausse, ou une mort manquante.
- Facultatif : confirmer le kill au pistolet plasma de `fccc61cd` a 08:55, sur la voie faible.

## LES DOCUMENTS, TOUS A LA RACINE DE `.ai/`

| Fichier | Ce qu il porte |
|---|---|
| `HANDOFF_KILLSOURCE_REPRISE.md` | ce fichier — point d entree |
| `PLAN_BRANCHEMENT_KILLSOURCE.md` | les 4 phases du branchement, la strategie de merge chiffree |
| `HANDOFF_PRECISION_PROJECTILES.md` | le lot cible sur la precision des armes a projectile |
| `PLAN_RECHERCHE_ASSETS_ICONES.md` | la recherche d icones pour l UI |
| `GUIDE_KILLSOURCE.md` | comment UTILISER le decodeur |
| `GUIDE_WEAPON_SHOTS.md` | comment utiliser les tirs et la precision — **porte le piege de l inversion MA40/Sidekick** |
| `ETAT_DE_L_ART_KILLWEAPON.md` | **l index interrogeable — a greper AVANT d ouvrir une piste** |
| `RE_LOG_KILLWEAPON.md` | le journal, APPEND-ONLY, **a ne jamais lire par le haut** |
