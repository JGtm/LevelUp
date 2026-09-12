# `ecs_table.tsv` — la grammaire ECS du film, une ligne par (archetype, composant)

CE QUE C'EST : la table de reference, versionnee et lisible par machine, des composants que le
registre du film (chunk_00) declare — 1 067 lignes reelles + 14 alias d'ecriture. Chaque ligne
porte l'index d'archetype `ti`, l'index de composant `i`, le nom, le niveau de precision, le
STATUT de portage du decodeur, l'adresse du deserialiseur, la source `fichier:ligne`, le champ
du `ReplayDocument` alimente, et ce qu'on sait du sens.

CE QUE CE N'EST PAS : la grammaire bit-exacte. Celle-ci vit dans le code et nulle part ailleurs
(`traverse.go` `consumeByName`, `components_*.go`, `keyframe_*.go`, `default_state*.go`). La
table dit QUI EXISTE, QUEL EST SON STATUT et OU LE LIRE — jamais combien de bits lire.

STATUTS : `porte` (tous les retours du cas rendent `true`) · `partiel` (au moins un retour
data-dependant ou garde par un drapeau : desync propre possible) · `non_porte` (aucun `case` ;
la traversee s'arrete, 0 bit consomme) · `deser_non_cable` (grammaire ecrite, aucun appelant) ·
`alias` (autre orthographe acceptee par le dispatch, `ti = -1`).

NIVEAU : colonne `level` = `Flags[k]`, ce que `registry.go` sert et ce que le traverseur passe
aux desers. Le JEU lit `Flags[k+1]` (decalage mesure au lot R7-e, 178 lignes concernees) :
quand les deux different, `notes` porte `niveau_jeu=N`. L'ecart est consigne, non traite.

MISE A JOUR : la table se corrige a la main, ligne par ligne. Porter un composant = editer sa
ligne (`status`, `deser_addr`, `code_source`) dans le meme commit que le code — les
garde-rails de `ecs_table_guard_test.go` echouent sinon.

GARDE-RAILS : **G1** code <-> table (sans film, toujours joue) · **G2** film <-> table (garde
`ECS_TABLE_FILM`, SKIP propre sans film) · **G3** table <-> `replay/document.go`.
Le lecteur du TSV vit dans le fichier de test : la table ne sert qu'aux garde-rails, elle n'a
aucun lecteur de production, et un lecteur hors test serait du code mort.
