# oracle-filmshell : temoin de lecture de libfilmshell

Outil jetable, hors de tout depot. Pour chaque film, une ligne TSV par element lu par la marche
`filmshell::research::payload` (`Walker` / `for_each_payload_in`).

    oracle <racine_des_films> <dossier_de_sortie> <id8> [<id8> ...]
    # racine : contient <id8>\chunk_NN.bin (zlib ou clair). Le dernier chunk (temps forts) est ecarte.

Sources : `src\main.rs`, `Cargo.toml`, copie patchee `filmshell-patch\` (voir `PATCH.md`).
Construction : `set CARGO_TARGET_DIR=C:\t\oracle-filmshell\target` puis `cargo build --release`.

## Colonnes

`film chunk paquet octet_debut octet_fin tick vue bit type index generation type_entite composants blocs detail`

- `chunk` : rang du chunk dans le film apres tri par numero de fichier ; 0 = en-tete (jamais marche),
  le chunk des temps forts n'est pas charge.
- `paquet` : rang du paquet dans le chunk, TOUS types confondus, depuis 0. Le rang parmi les seuls
  paquets de type 0 est `rang0=` dans le `detail` de la ligne `P`.
- `octet_debut` / `octet_fin` : `[debut, fin[` de la charge utile dans le chunk decompresse
  (octets). C'est `Packet.start` / `Packet.end` (apres l'en-tete de 16 octets du paquet).
- `tick` : `Film::frame_ticks()[i]`, tick de jeu a 60 Hz deroule depuis les compteurs de tick.
- Un paquet dont la charge utile fait moins de 2 octets n'est pas marche par la bibliotheque : pas de
  ligne du tout.

## Definition de `bit` (toutes les vues)

`bit` = position RELATIVE a la charge utile du paquet : `bit = bit_absolu_dans_le_chunk - 8 * octet_debut`.
Bit 0 = bit de poids fort du premier octet de la charge utile ; numerotation MSB d'abord (bit 0 est le
bit 7 de l'octet `octet_debut`, bit 8 le bit 7 de l'octet suivant, etc.).

Structure d'une charge utile : `[drapeau:1] vue_evenements vue_entites vue_controle`. Le bit 0 est le
drapeau initial ; la vue des evenements commence au bit 1.

| vue / type | ce que vaut `bit` |
|---|---|
| `P / paquet` | 0 (le paquet entier). Les positions utiles sont dans `detail`, relatives aussi. |
| `A / <kind>` | `Event.at` : bit du champ de type de 7 bits de l'evenement, APRES son drapeau de continuation `[1]` (ce drapeau est au bit `bit - 1`). Les poignees (3 x `[present:1]` + poignee) suivent le type, puis le corps. |
| `B / update` | `EntityOp.at` : bit du champ d'INDEX (13 bits), APRES les bits d'operation. L'operation d'une mise a jour = 1 bit (`1`) au bit `bit - 1` (`op_bit` du detail). Ensuite `generation` (2 bits) a `bit + 13`, puis l'en-tete de liste a `bit + 15` : `[baseline:1]` (+7 bits si 1), `[forme:1]`, forme 0 `[n:3]` puis n indices de 6 bits, forme 1 masque de 64 bits. Les donnees des composants suivent. |
| `B / create`, `B / delete` | idem : bit du champ d'index. L'operation = 3 bits (`0` + 2 bits) a `bit - 3` (`op_bit`). `generation` a `bit + 13`. Create : `type` (6 bits) a `bit + 15`, donnees de creation a `bit + 21`. Delete : 32 bits suivent la generation. |
| `C / message` | (patch) bit du drapeau `[more:1]` qui ouvre le message ; le genre (2 bits) est a `bit + 1`, puis `[baseline:1]` (+7) puis `joueur` (5 bits). |

Colonne `blocs` (B) : liste `composant:bit`, separes par des virgules, dans l'ordre de la liste. `bit`
= bit (relatif) du PREMIER bit des donnees du bloc de ce composant (ce que rend `parsed.starts`, la
valeur de `r.at` juste avant la lecture du bloc). Le bloc du composant k s'etend jusqu'au debut du
bloc suivant ; le dernier jusqu'a la fin de la mise a jour (non rendue par la bibliotheque : c'est
le `op_bit` de l'entree suivante quand la lecture est continue). Pour un create, `blocs` et
`composants` viennent de `EntityOp.init` (composants initiaux), vides si le create n'en liste pas ou si
leur lecture a echoue (`init=absent` dans `detail`).

## Lignes

### `P / paquet` (une par paquet marche)

`detail`, champs `cle=valeur;` :

- `rang0` : rang parmi les paquets de type 0 du chunk ; `taille` : octets de la charge utile ; `ts_us`.
- `event_stop`, `stop` : `aucun` ou la variante de `Stop` (ex. `EventBody(105)`, `CreateTail(7)`).
  `stop` = arret definitif de la marche des entites.
- `reprises` : nombre d'entrees de `skipped`, puis `repriseN=<cause>@<depart>><reprise>` : `depart`
  = bit ou la lecture a echoue (debut de l'entree, ou bit 1 si c'est la vue des evenements) ;
  `reprise` = bit du message de tick sur lequel la marche reprend. Tous deux relatifs.
- `entity_start` : premier bit de la vue des entites (juste apres le `[0]` qui termine la vue des
  evenements), ou `-` si la vue des evenements ne s'est pas terminee.
- `end` : `Payload.end`. Si `stop=aucun` : bit APRES l'operation 0 qui termine la vue des entites
  (= debut de l'entree terminale + 3 ; l'index et la generation de cette entree ne sont pas consommes).
  Si `stop` est pose : bit de DEBUT (avant les bits d'operation) de l'entree qui a echoue ; si la vue des
  evenements echoue sans message de tick retrouvable : 1.
- `ctrl_valide` : `oui` si `stop=aucun` (le `end` est alors un vrai debut de vue de controle), sinon
  `non(stop)` : `control_view` est lancee quand meme a `end` mais ses chiffres n'ont pas de sens.
- `ctrl_msgs` : nombre de messages que `components::control_view(c, end, params)` rend ; `ctrl_fin` : bit
  apres le dernier `[more:0]` (relatif), `-` si la vue s'arrete sur un genre non lu ; `ctrl_reste_bits` =
  `8 * octet_fin - fin_absolue` ; `ctrl_fin_dernier_octet` = `oui` si ce reste est dans `0..8`, c'est-a-dire
  si la fin tombe dans le dernier octet du paquet (critere de la sonde d'origine ; `na` si pas de fin).

### `A / <kind>` (une par evenement de la vue des evenements)

`type` = kind numerique (0 a 127). `detail` : `h0=..;h1=..;h2=..;body=...` ; poignee =
`<entite>:<generation>` avec suffixe `u` si c'est une poignee d'unite, `-` si absente
(`Handle::entity()` : l'index est deja ramene a l'espace des entites) ; `body` = `Debug` de
`events::Body`. Les evenements lus avant un `event_stop` sont presents.

### `B / create|update|delete` (une par `EntityOp`)

`index`, `generation`, `type_entite` (type du registre, vide si inconnu ou efface par la resolution
d'ambiguite), `composants` (update : `list` ; create : composants initiaux), `blocs` (voir plus haut).
`detail` : `op_bit=<bit>` toujours ; create : `tag=0x........` (tag de l'objet, `-` pour un non-objet) et
`init=absent` le cas echeant ; `tireur=<joueur>,<poignee>` pour un projectile. Une entree n'est rendue
que si la bibliotheque l'a poussee dans `ops` : une mise a jour dont le type est inconnu
(`UnknownEntity`) n'apparait pas (elle est dans les reprises/arret de la ligne `P`).

### `C / message` (une par `ControlMessage` rendu, seulement si `stop=aucun`)

`detail` : `joueur`, `genre` (1 si `command_at` est pose, sinon 0), `debit` (deux valeurs de 6 bits),
`etat` (premier champ du bloc d'unite), `cmd_bit` (`command_at`, relatif), `tete` (4 champs
optionnels du bloc de controle), `plan` (plan de tir, 6 valeurs).

## Limites

- Les messages de controle de genre 2 sont lus et ignores par la bibliotheque (`continue` sans
  `push`) : pas de ligne `C`. Genre 3 : arret de la lecture (`ctrl_fin=-`).
- Pas de bit de FIN par entree ni par bloc : la bibliotheque ne le garde pas. Fin d'une mise a jour
  = `op_bit` de l'entree suivante quand rien n'a ete repris entre les deux ; sinon inconnue. Idem
  pour la fin d'un create et pour le bit de la liste initiale d'un create.
- Une mise a jour dont la lecture echoue en cours de liste rend les blocs jusqu'au composant fautif
  (inclus) ; l'echec se lit dans la ligne `P` (reprise ou arret), non dans la ligne `B`.
- `blocs`/`composants` d'un create absents quand `init` n'a pas ete rempli (lecture echouee, ou liste
  vide).
