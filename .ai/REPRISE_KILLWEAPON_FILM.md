# Reprise — Kill-weapon dans le film (handoff, état 2026-06-05 soir)

> Doc de reprise pour demain. Résume où on a atterri sur "l'arme-de-kill est-elle dans le film ?",
> la clarification modules vs décodeur FRAME, la décision en attente, et le script de l'étape 0 prêt à lancer.
> Contexte complet : `RE_EXE_GHIDRA_FINDINGS.md` §4 (RAFFINEMENT) + §7 ; `thought_log.md` [2026-06-05].

## TL;DR

- **Question** : l'arme-de-kill exacte est-elle récupérable du film Theater ?
- **Réponse atteinte** : OUI en principe. L'arme est un **u32 asset-id dans un composant ECS `'obje'`** (group tag
  "object" de Halo), **pas** un champ du type-3 highlight event. Même substrat (et même mur) que le score perdant +
  les positions per-joueur déjà étudiés.
- **L'utilisateur avait raison sur le fond** ("toutes les données sont dans le film") ; le "négatif" antérieur ne
  valait que pour le **type-3 event** (octet libre), pas pour la réplication ECS.
- **Reste** : (a) **CONFIRMER** que `'obje'` (ou sa source) est bien **répliqué** (≠ composant client-local) ;
  (b) construire le **décodeur FRAME** (le seul vrai blocage, multi-sessions).
- **DÉCISION EN ATTENTE** : **parquer** (reco) vs **pousser** le décodeur FRAME.

## La chaîne tracée (preuve que l'arme = composant ECS, pas une valeur UI)

```
FUN_1420ca9a0   (peuple le kill-feed : itère une liste d'entrées [id,key,?])
   └─ FUN_1407e941c(VM_killfeed, handle, key)
        └─ FUN_1407e8afc(handle, idx, 0)                 // récupère l'item-arme
             └─ FUN_1405839d0(entity+0x2c, 0x6f626a65)   // 0x6f626a65 = 'obje' (group tag object)
   build : FUN_1407e9724(weaponDisplay, srcItem)
        → *(u32*)(srcItem + 0x18) = asset-id de l'arme   (+ srcItem+0x1c, +0x20)
```
- Composant `'obje'` : tableau de records **0x344 o**, item-arme = sous-bloc **0x24 o**, arme = **u32 asset-id** à +0x18.
- Accesseurs du champ `KillerWeapon` (`FUN_14347db40`/`FUN_14347db88`) = **génériques type-4 "ItemDisplayData"**,
  partagés par store/flag/emblem/career → l'arme est rangée comme tout item à icône (utile app : weapon icon =
  item-display asset).

## Modules vs décodeur FRAME (clarification — ne plus confondre)

- **"Décoder les modules"** (Reclaimer / AusarDocs / InfiniteModuleEditor…) = décoder les fichiers `.module`
  **statiques** → **dictionnaire** : définitions de tags/composants + catalogue d'armes (asset-id → nom/icône).
  Plein de projets GitHub le font. **On a déjà l'essentiel** (notre enum `analysis.WeaponIDToName` + `weapon_labels`).
- Le **film** n'est PAS un module : c'est un **flux de réplication enregistré** (valeurs de composants bit-packées,
  deltas, par frame). **Aucun projet public ne le décode** (frontière dend/acurtis).
- Pour sortir l'arme du film il faut **les DEUX** : le dictionnaire (modules, ~acquis) **+** le **décodeur FRAME**
  (lire la valeur du composant `'obje'` dans CE match). Les modules seuls ne donnent pas les valeurs par-match ; le
  décodeur FRAME sans schéma donne des bits ininterprétables. ⇒ le décodeur FRAME est **la moitié manquante**.

## Roadmap si on POUSSE (push path)

- **Étape 0 — VÉRIF (1 script, prêt ci-dessous)** : confirmer que `'obje'` est **répliqué**. Chercher l'usage du
  FourCC `0x6f626a65` + comprendre le getter `FUN_1405839d0` → trouver la **registration** du composant et son
  **type-id** dans la table de dispatch réplication `0x145435cd8` (+ vtable/désérialiseur, cf. `0x143c96b38` modèle
  statborg). Si `'obje'` (ou son parent) apparaît dans le dispatch de réplication → **confirmé filmé** + on tient la
  **clé de routage**. Sinon (client-local) → l'arme n'est PAS filmée et le négatif redevient ferme.
- **Étape 1** : BitReader **bit-exact** en Go — port de `FUN_140c18a1c` (int signé `[2-bit largeur]`) + `FUN_1406cf008`
  (1 bit MSB-first) + machine refill 8o/byte-swap big-endian (cf. §2 RE_EXE_GHIDRA_FINDINGS).
- **Étape 2** : framing FRAME — router les records par type-id (dispatch `0x145435cd8` → vtable → slot désérialiseur),
  appliquer au composant `'obje'`.
- **Étape 3** : extraire le sous-champ arme (offset dans le record 0x344 / item 0x24, asset-id u32) + mapper via notre
  catalogue (`WeaponIDToName` / `weapon_labels`).
- **Étape 4** : valider sur un match connu (`7344d24f`, `000d5950`) + comparer per-kill à `weapon_kills_v3`.

## Caveats à garder en tête

- **Pertinence réseau** : même piège que le score perdant — un kill hors-écran d'un joueur non-répliqué chez
  l'enregistreur pourrait ne pas avoir son `'obje'` dans le flux. À quantifier (Étape 4).
- **Kill-bound** : l'arme du feed est figée à l'instant du kill (argument sniper de l'utilisateur, validé) → la source
  `'obje'` doit être un objet d'event/crédit figé, pas l'inventaire courant du tueur. À confirmer Étape 0/2.
- **Gel maintenu** : aucun backfill, aucun commit sans autorisation explicite.

## Note utilisateur (2026-06-05 soir) — réalisme stockage + piste "table de référence par terrain"

**Correction de modèle (l'utilisateur a raison)** : le film ne stocke PAS les objets gras (insensé pour des millions
de matchs). Le record **0x344 o** vu dans `FUN_1407e8afc` = **RAM runtime** (composant ECS reconstruit côté client),
**pas** les octets du film. Le film ne porte que **deltas + références compactes** filtrés par pertinence. Donc
"l'arme dans le film" = une **petite référence/id**, possiblement un **handle d'objet per-match**, pas forcément le
tag-id global.

**Piste alternative (plus légère que le décodeur FRAME) — known-plaintext / table de référence** :
- Sur un match **récent à kills connus** (vérité terrain : observation replay / partie jouée), localiser l'id qui
  apparaît près de chaque kill (ancré sur les type-3 kill events : on a `time_ms` + killer/victim), corréler, et
  **construire la table id→arme**. Court-circuite potentiellement le décodeur FRAME complet.
- **Test de viabilité décisif** : l'id du film est-il **stable (global)** [table réutilisable] ou un **handle
  per-match** [résolution par match requise] ? Premier test terrain = même arme → même id sur 2 matchs.
- **Déjà acquis** : table stable pour le tag-id des **fire events** (suffixe `42c9679f` → `analysis.WeaponIDToName`).
  La valeur ajoutée du chemin `'obje'`/kill-feed = couvrir les kills de **TOUS les joueurs** (pas juste
  l'enregistreur), **si** son id est stable.
- **Pré-requis** : (1) vérité terrain per-kill (observation) ; (2) **octets d'events film** (on n'a que le header en
  cache → nécessite un re-fetch = arbitrage du gel) ; (3) ancrage temporel kill→région film.
- **Condition (utilisateur)** : à n'explorer que **si la piste est envisageable ET qu'on n'a pas d'autre option** —
  i.e. si le décodeur FRAME complet est jugé trop lourd ET que v3 fire-events ne couvre pas assez les kills des
  autres joueurs. Sinon, parquer.

## Décision recommandée

**Parquer.** `weapon_kills_v3` (inférence fire events type-2) couvre le besoin produit ; le décodeur FRAME est lourd
(multi-sessions) et était gelé pending retour RE externe (dend/acurtis). Mais le dossier "arme dans le film" est
désormais **étayé** (pas juste une intuition) → argument fort pour s'y mettre quand les RE externes répondent ou quand
on décide d'investir le chantier ECS générique (qui débloque **arme + score perdant + positions** d'un coup).

## Script prêt — Étape 0 : `FindObjeComponent.java`

Enregistrer dans `C:\Users\Guillaume\ghidra_scripts\FindObjeComponent.java`, lancer → `ghidra_obje_component.c`.

```java
import ghidra.app.script.GhidraScript;
import ghidra.app.decompiler.DecompInterface;
import ghidra.app.decompiler.DecompileResults;
import ghidra.program.model.address.Address;
import ghidra.program.model.listing.Function;
import ghidra.program.model.listing.FunctionManager;
import ghidra.util.task.ConsoleTaskMonitor;
import java.io.FileWriter;
import java.io.PrintWriter;
import java.util.HashSet;
import java.util.Set;

public class FindObjeComponent extends GhidraScript {
    DecompInterface di; ConsoleTaskMonitor mon; FunctionManager fm; PrintWriter out;
    Set<Long> done = new HashSet<Long>();

    void dumpAt(long va, String tag) throws Exception {
        Function f = fm.getFunctionContaining(toAddr(va));
        if (f == null) { out.println("\n// NO FUNC @ 0x" + Long.toHexString(va) + " [" + tag + "]"); return; }
        long ea = f.getEntryPoint().getOffset();
        if (done.contains(ea)) return; done.add(ea);
        DecompileResults res = di.decompileFunction(f, 120, mon);
        out.println("\n//////// 0x" + Long.toHexString(ea) + " " + f.getName() + "  [" + tag + "] ////////");
        if (res != null && res.decompileCompleted()) out.print(res.getDecompiledFunction().getC());
        else out.println("// decompile failed");
    }

    @Override
    public void run() throws Exception {
        di = new DecompInterface(); di.openProgram(currentProgram);
        mon = new ConsoleTaskMonitor(); fm = currentProgram.getFunctionManager();
        out = new PrintWriter(new FileWriter("C:\\Users\\Guillaume\\Downloads\\ghidra_obje_component.c"));

        // 1. la plomberie ECS que le kill-feed a utilisée
        dumpAt(0x1405839d0L, "component-getByFourCC");
        dumpAt(0x14049a384L, "handle-to-object");

        // 2. tous les sites référençant le FourCC 'obje' (0x6f626a65 = octets 65 6a 62 6f)
        out.println("\n===== sites FourCC 'obje' (65 6a 62 6f) =====");
        Address a = currentProgram.getMinAddress();
        int n = 0;
        while (a != null && n < 40) {
            Address hit = findBytes(a, "\\x65\\x6a\\x62\\x6f");
            if (hit == null) break;
            Function f = fm.getFunctionContaining(hit);
            out.println("// 'obje' @0x" + hit + (f != null ? ("  in " + f.getName()) : "  (data/no func)"));
            if (f != null) dumpAt(f.getEntryPoint().getOffset(), "obje-site");
            a = hit.add(1);
            n++;
        }
        out.close();
        println("DONE -> ghidra_obje_component.c  (" + n + " hits FourCC)");
    }
}
```

Ce qu'on cherchera dedans : la **registration** du composant `'obje'` (l'équivalent des registreurs ViewModel qu'on a
vus) et surtout s'il est branché sur le **dispatch de réplication** `0x145435cd8` → preuve "filmé" + clé de routage
pour le décodeur FRAME.
