# Patch de la copie de libfilmshell (`filmshell-patch/`)

Copie de `C:\Users\Guillaume\Downloads\Scripts\libfilmshell` (dossiers `src/`, `Cargo.toml`,
`Cargo.lock`, `LICENSE`). Le dossier d'origine n'est pas touche. Le patch n'expose que des
positions de bit : aucune decision de lecture ne change (la marche `Walker` ne relit jamais
`Fields.blocks` ni `ControlMessage.at`).

Reproduire la liste : `diff -u <libfilmshell>/src/<f> filmshell-patch/src/<f>`.

## src/payload.rs (fonction `Walker::data`, branche des objets, ~l.465)

Avant :

    if t == MANAGED_PARTICIPANT || t == DEVICE || t == CRATE {
        fields.blocks = parsed.starts;
    }

Apres :

    // ORACLE PATCH: keep every object type's block starts, not only 47/43/38.
    let _ = (MANAGED_PARTICIPANT, DEVICE, CRATE);
    fields.blocks = parsed.starts;

Effet : `Fields.blocks` (liste `(composant, bit absolu du debut du bloc)`) est garde pour TOUS les
types d'objet (35 a 43 et au-dela), plus seulement pour 47, 43 et 38. Les types non objet (< 35)
le gardaient deja. Les blocs gardes sont ceux lus jusqu'a l'echec eventuel (un composant illisible
est le dernier de la liste). `let _ = (...)` n'existe que pour ne pas laisser trois constantes
inutilisees.

## src/components.rs

1. Structure `ControlMessage` : ajout d'un champ en tete, `pub at: i64` (+ commentaire doc). Le
   `Default` derive le met a 0.
2. `control_view`, boucle `while r.flag() {` : ajout de la ligne
   `let msg_at = r.at - 1;` juste apres (le drapeau `[more:1]` vient d'etre lu).
3. `control_view`, construction du message : `ControlMessage { player, ..Default::default() }`
   devient `ControlMessage { at: msg_at, player, ..Default::default() }`.

Effet : chaque message de la vue de controle porte le bit absolu de son drapeau `[more:1]`
d'ouverture.

## Controle "sans patch"

`unpatched/Cargo.toml` recompile le meme `src/main.rs` (sans la feature `patched`) contre la
libfilmshell d'origine ; les lignes `P` et `A` sortent octet pour octet identiques, les lignes `B`
identiques hors colonnes `blocs` et `composants` d'un create (vides sans patch pour les types
d'objet), les lignes `C` n'ont plus de bit.
