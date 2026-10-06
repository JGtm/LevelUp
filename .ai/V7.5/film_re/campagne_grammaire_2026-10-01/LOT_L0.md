# Lot L0 — la fermeture d'un paquet suit les règles de l'écrivain (2026-10-02)

> Lot L0 du plan `.ai/V7.5/PLAN_CAMPAGNE_GRAMMAIRE_2026-10-01.md` (§6.2 L0), sous le contrat
> `plan-execution`, dans le cadre de la décision du 2026-10-02 au soir : **corrections d'abord,
> uniquement générales, lues dans le jeu** (aucun réglage par film, par carte ni par version).
> Décision ferme D2 : un paquet est **fermé** si son reste est nul (`vueCFermee`) **et** si aucune
> règle de l'écrivain du jeu n'est contredite. Chaque règle retenue cite la fonction de l'écrivain
> qui la fonde ; une règle sans écrivain n'entre pas.
>
> Worktree `LevelUp-wt-campagne-grammaire`, branche `feat/campagne-grammaire`, tête `ff42fcf40`,
> **rien de commité**. Films en lecture seule (20 films = 19 témoins de `config/replay_corpus.toml`
> + `1c4c63c2`), un à la fois, plafond 4 Gio. Aucune base ouverte, aucune cuisson en lot.
> Convention : **mesuré** = compté par l'outil sur les films ; **établi** = déduit de l'écrivain
> lu (Ghidra) ; **supposé** = hypothèse écrite.

## 0. Statut des sous-items

| Item | Statut | En une ligne |
|---|---|---|
| L0.1 | fait | Règles de l'écrivain et sortie de vue B dans la carte ; `bloquantDuPaquet` : sortie par rejet avant les causes de vue C ; `source.Bits.Deborde()` |
| L0.2 | fait | Naissance de génération 0 (D-43), NEW lu désynchronisé (D-44), bloc 0xbc requalifié en désalignement (D-53) |
| L0.3 | fait | Garde-fou de la recopie : `ks_000d5950` 446 listes localisées / 27 non localisées, `ks_e5adf7b2` 52 / 2 ; aucune bobine ni paquet synthétique à ajouter |
| L0.4 | fait | Records utiles lus, dénominateurs fixe (recalculé : maximum du fixe consolidé et des lus) et variable, par film et par build |
| L0.5 | fait | D-44 : 23 338 des 213 040 paquets « naissance non lue » de la référence (11,0 %) ont un NEW lu désynchronisé ; 3 082 records utiles en jeu sur 2 742 723 (0,11 %) |
| L0.6 | fait, adopté | Sous-groupes mesurés (§1.1) ; règle « sortie de vue B par rejet ⇒ non fermé » dans la définition |
| L0.7 | fait, adopté | Bit de masque au-delà du dernier composant classé violé à la lecture (`EntityTrace.MasqueNonEcrit`) |
| L0.8 | non retenu | La mesure ne le soutient pas (§1.3) : la règle se juge contre l'archétype du datum, que le paquet ne porte pas |
| L0.9 | fait | Témoins décalés de −8 à +8 bits, sous la même définition (règles de la vue C comprises) |
| Définition | faite | `LectureVueC.Fermee` ; `debutParFermeture` et le tir continu la lisent (§3) |

## 1. Mesures préalables

Sonde `campagne_l0_research_test.go` (`TestCampagneL0Mesures`), jouée **définition neutralisée**
(verdict « fermé » = fermé au bit près, la marche de la référence) : sorties dans `l0_tsv/`
(`l0_mesures.tsv`, `l0_rejets_sans_debut_anterieur.tsv`). Contrôles : 284 704 paquets fermés au bit
près (= la carte v2 de la phase 1) ; 4 598 fermés après rejet dont 1 008 sains au juge, 196 sans
début antérieur dont 32 sains (= R-L1 (b), D-113). **Le juge de la sonde** (`cmContredit`, trois
invariants relus) **et les règles de production s'accordent sur les 284 704 paquets** (ordre 1 072
/ 283 632, masque 8 150 / 276 554, vue C 279 / 284 425 : aucun désaccord).

### 1.1 L0.6 — sortie de vue B par rejet (D-113)

Écrivain (établi, T3 §1.4, vérifié adverse) : `FUN_142f2e174` n'écrit un DELTA que pour une entité à
l'état 3, que seuls l'écriture et l'acquittement d'un NEW posent (`FUN_142f2cee0`, `FUN_142f2f8f0`) ;
la vue du film s'acquitte elle-même (`FUN_142f2cc78`). Un en-tête DELTA rejeté n'est donc jamais un
record lu à sa place.

| Fermés au bit après rejet | Début antérieur qui ferme | Aucun début antérieur | dont reste < 9 bits | reste 9-64 | reste > 64 |
|---|---|---|---|---|---|
| sains au juge (1 008) | 976 | **32** | 16 | 2 | 14 |
| contredits (3 590) | 3 426 | **164** | 20 | 11 | 133 |

- **Les 648 paquets à reste ≤ 8 bits** : 648 / 648 ont moins de 9 bits derrière l'en-tête rejeté.
  Un DELTA réel laisse derrière son en-tête au moins 9 bits : corps (`R(1)` de base de
  `FUN_140769e08`, masque épars vide de 4 bits de `FUN_142e2da44`), terminateur de vue B de 3 bits
  (`FUN_14076c75c`), terminateur de vue C de 1 bit (`FUN_142f2c3b0`). **Établi** : l'en-tête rejeté
  n'est pas un record de l'écrivain.
- **Les 32 sains sans début antérieur** (détail par paquet dans `l0_rejets_sans_debut_anterieur.tsv`) :
  16 à reste < 9 bits (établi, ci-dessus) ; 2 dont l'eid est « libéré avant le chunk » (un DELTA
  d'une entité morte : l'écrivain écrit un DEL, `FUN_142f2e174`) ; 2 de génération 2 ou 3 sans
  aucune allocation aux deux blocs ; 12 de génération 0 sans allocation à reste ≥ 9 bits. Pour ces
  14 derniers, `FUN_142f2e598` (`gen = (gen + 1) & 3` à chaque allocation) exige quatre allocations
  du même slot entre deux images-clés pour revenir à la génération lue : établi, à cette réserve près.
- **Verdict** : les 4 598 fermetures après rejet sont factices (976 + 3 426 par le début antérieur,
  648 par la place, les 32 sains un par un). L0.6 est **adopté**.

### 1.2 L0.7 — bit de masque au-delà de l'archétype (D-85)

Écrivain (établi, T2 §2-§3) : `FUN_142e2da44` ne pose un bit que pour `i < *(desc+0x4320)` ; le
registre du film est la copie verbatim du descripteur (`FUN_14299b674`, `FUN_142998c7c`).

| État du paquet | Records à masque lu | Au-delà de l'archétype | Dense ≤ 7 | Épars non croissant |
|---|---|---|---|---|
| fermé au bit, sain au juge | 3 372 973 | 0 | 0 | 0 |
| fermé au bit, contredit | 27 887 | 8 995 | 0 | 225 |
| non fermé au bit | 4 160 968 | 8 619 | 3 | 251 |

La règle est désormais posée **à la lecture** : `traverseComponentLoopFrom` marque la trace
(`EntityTrace.MasqueNonEcrit`) au lieu d'ignorer le bit en silence ; `lireMasque` juge la forme
(dense ≤ 7, épars non croissant). **Adopté.**

### 1.3 L0.8 — mot de 32 bits d'un DEL (D-83) : non retenu

Écrivain (lu ce jour, Ghidra HTTP, lecture seule) : `FUN_142f304a8` écrit `W(32)` = l'appel
`vtable+0xd8` quand l'archétype du datum (`*(table + slot*200 + 4)`, indice de `DAT_144e61d88`)
vaut `0x10`, zéro sinon.

Mesure (relecture du mot, `l0MotsDeDel`) :

| État du paquet | DEL lus | Mot non nul |
|---|---|---|
| fermé au bit, sain au juge | 15 962 | 106 |
| fermé au bit, contredit | 1 454 | 1 017 |
| non fermé au bit | 3 715 | 2 682 |

Ventilation par archétype du slot, **selon le monde hors ligne** (instrumentation temporaire, une
passe, décrite dans `l0_tsv/l0_8_instrumentation_temporaire.txt`) : dans les paquets sains, 26 DEL à
mot non nul sur un slot lié à un archétype autre que 0x10 (ti=5 neuf fois sur les slots 52-55, ti=8
trois fois sur le slot 246, ti=18, ti=25, ti=35…), et 80 sur un slot non lié. Les mots sont
STRUCTURÉS (`0x8021203b`, `0x9821203b`, `0x8121205b`, `0x7e21209b`…), répétés d'un film à l'autre
sur les mêmes slots : ce n'est pas l'allure d'une lecture désalignée. Lecture (supposée) : le monde
hors ligne lie ces slots à un autre archétype que le datum du jeu. La règle n'est donc jugeable que
contre l'archétype du DATUM, que le paquet ne porte pas : l'adopter ferait dépendre la fermeture
d'une liaison du monde, pas d'une lecture. **Non retenu** ; la règle a été retirée du code
(`ecrivain_invariants.go` dit pourquoi en une ligne). Découverte D-L0-1 (§9).

**Pourquoi L0.7 est retenu et L0.8 non, alors que les deux dépendent de l'archétype que le monde
attribue au slot (estimé, non vérifié paquet par paquet).** Un archétype faux change le nombre de bits
qu'un DELTA consomme (la boucle de composants suit le descripteur) : quand L0.7 voit un bit de masque
au-delà de l'archétype, la lecture du paquet est déjà fausse, et la juger non fermée ne retire rien de
juste. Le mot de 32 bits d'un DEL, lui, est lu quel que soit l'archétype : une liaison fausse ne change
pas la consommation, la lecture peut être juste, et L0.8 retirerait des paquets bien lus.

## 2. Les règles retenues, et leur écrivain

| Règle (`InvariantEcrivain`) | Écrivain | Où elle se juge |
|---|---|---|
| sortie de vue B par rejet | `FUN_142f2e174`, `FUN_142f2cee0`, `FUN_142f2f8f0`, `FUN_142f2cc78` | `decodeInferLoop` pose `Lecteur.rejetVueB` |
| ordre de la vue B (NEW*, DELTA*, DEL*, slots strictement croissants par groupe) | `FUN_14076b9c8`, `FUN_142f2e174`, `FUN_142f24a78` | records de la vue B |
| masque au-delà de l'archétype | `FUN_142e2da44` (`i < *(desc+0x4320)`), `FUN_14064dd28`, `FUN_142998c7c` | traversée |
| masque dense de 7 composants au plus | `FUN_142e2da44` (`|S| > 7` ⇒ dense) | `lireMasque` |
| masque épars non croissant | `FUN_142e2da44` (`i croissant dans S`) | `lireMasque` |
| vue C : kind non nul | `FUN_142f2c3b0` (tampons par joueur), `FUN_14076b0e8` (kind 0 seul) | flux de la vue C |
| vue C : plus de 32 entrées | `FUN_142f2c3b0` (k = 0..31) | flux de la vue C |
| vue C : index non strictement croissants | `FUN_142f2c3b0` (tampon choisi par l'index) | flux de la vue C |
| vue C : bit d'en-tête `FUN_1406cdc04` posé | `FUN_14076b0e8` (`W(1)=0`) ; la variante froide `14230bf83` écrit dans la vue réseau (`vue+0x6088`, vérification adverse) | flux de la vue C |
| vue C : code analogique 63 | `FUN_1406d5bf4` (borné à `[0, 0x3e]`), `FUN_1406d33cc` | flux de la vue C |

Non retenus : le mot du DEL (§1.3) ; « +0x14 présent et nul » de T5 §6 (`FUN_1406d143c`) — la
largeur `W(5 | 7)` peut tronquer une valeur non nulle, non vérifié : hors de ce lot.

**Réserve (estimé).** Les règles sont lues dans l'exécutable installé et s'appliquent sans
distinction aux builds dont aucun exécutable n'est disponible : sur `a349fea8` (`version-33`), elles
retirent 43 des 463 paquets fermés au bit près (§8). Qu'elles valent pour l'écrivain de ces builds
est supposé, non vérifié dans le jeu.

## 3. La définition, et ce qu'elle change dans la marche

`ecrivain_invariants.go` : `verdictDeVueC` rend `LectureVueC{FermeeAuBit, Invariant, Fermee}` ;
`Fermee = FermeeAuBit && Invariant == aucun`. Les deux marches (`decodeFrameParRangs`, production ; 
`marcheDetaillee.marcherParRangs`, carte) l'appellent : une seule définition.

Lecteurs de `Fermee` en production : `debutParFermeture` (`debut_de_liste.go`) et le collecteur du
tir continu (`tir_continu.go`, inchangé : un paquet non fermé est un trou).

**`debutParFermeture` : deux rangs (décision prise dans le lot, mesurée).** La forme « premier
candidat d'où le paquet ferme, sinon liste non localisée » a été mesurée d'abord
(`l0_tsv/` : « pur ») : 29 paquets sains de la référence perdus — `e5adf7b2` −20 sains nets
(−692 records utiles sains), en cascade : le paquet 10:20, fermé de façon factice depuis un NEW
`0x434` `ti=41` dont le corps est mal lu (masque `0xc9b2…` au-delà de l'archétype) mais dont
l'en-tête est juste (l'entité est relue proprement par 23 paquets suivants), n'est plus localisé ;
la liaison disparaît et les 23 paquets sortent par rejet. La forme retenue garde la tête FERMÉE AU
BIT quand aucun candidat ne ferme : le paquet reste non fermé (son verdict contredit l'écrivain, le
tir continu y voit un trou), ses records sont lus et ses NEW liés, comme ceux de tout paquet lu et
non fermé. Résultat : **0 paquet sain perdu sur les 20 films, sous le juge de L0** (voir §8 pour
l'ancien juge). C'est la seule forme qui tienne le gate 2 ; elle n'ajoute aucune lecture ni aucun
réglage par film.

**Le second rang est un repli nommé et compté (décision du pilote D-A, 2026-10-02, corrections du
contrôle).** Ce rang ne prouve rien : il garde une tête dont le paquet ne ferme pas. Il entre donc au
registre des replis (ADR 0034 D-10) sous le nom `repli_debut_de_liste_ferme_au_bit`
(`registre_filmdec_marche.go`, condition « lecture non portée » : le corps du record de tête est mal
lu, D-L0-2 ; ordre « après lecture » : le premier rang est essayé sur tous les candidats avant).
Compte = listes dont le début est pris à ce rang : `Observation.DebutsDeListeParRepliFermeAuBit`,
rendu par la marche des trames (`MarcheDesTrames.DebutsDeListeParRepliFermeAuBit`) et versé au
compteur de la cuisson par `replay/film_scan_mouvement.go` (`coverage.fallbacks`), le même chemin
que `repli_liaison_par_anticipation` ; aucun champ persisté neuf dans les faits. Cible de retrait : la
lecture du corps de ces records NEW (D-L0-2). Le commentaire de `debutParFermeture` ne porte plus que
le contrat ; l'histoire (`e5adf7b2` 10:20, NEW `0x434` `ti=41`, 23 paquets) est celle de ce §3.
Déclenchements par film : §7, gate 6.

## 4. L'instrument (L0.1 à L0.5, L0.9)

- Carte (`frame_closure_detail.go`) : `PaquetDeCarte` porte `FermeeAuBit`, `Invariant`, `Invariants`
  (toutes les règles contredites, jugées sur tout paquet dont la vue B est atteinte),
  `ListeLocalisee`, `NeufsDesynchronises`, `UtilesLus`, `Deborde`, `TemoinsDecales`.
  `FrameClosureReport` porte `PaquetsFermesAuBit` et `Utiles.RecordsFermesAuBit`.
- Classement (`frame_closure_classement.go`) : la sortie par rejet passe avant les causes de la vue C
  (la vue B n'est plus « terminée » sur un rejet) ; la règle contredite d'un paquet fermé au bit est
  sa cause ; « vue C : bloc 0xbc (désalignement) ».
- `cmd_fermeture -mode v2` : `-denominateur-fixe <tsv>` (colonnes `film`, `fixe` ;
  `r_comb2_denominateurs.tsv`), `-paquets` (une ligne par paquet) ; TSV neufs
  `fermeture_ecrivain.tsv`, `fermeture_denominateurs.tsv`, `fermeture_temoins_decales.tsv`,
  `fermeture_paquets.tsv` ; section « Lot L0 » du résumé. Classes de naissance corrigées
  (`v2_datums.go`) et, à l'identique, `cmBlocs.naissance` des sondes.
- L0.5 (D-44, référence) : « naissance non lue » 213 040 → 189 702 + **23 338** « NEW lu
  désynchronisé · naissance non lue » ; « réalloué sous une autre génération » 13 035 → 180, dont
  12 855 deviennent « naissance non lue, génération 0 » (8 770 + 4 085 désynchronisés), D-43.
- L0.4, corpus après L0 : utiles fermés 2 585 919 sur 5 963 499 lus (**43,4 %**, variable) et sur
  7 758 290 (**33,3 %**, fixe consolidé, inchangé : aucune marche ne lit plus que lui).
- L0.9, corpus après L0 (paquets fermés relus depuis un départ de vue C décalé de k bits, même
  définition) : k = −1 à −8 : 3,8 / 3,8 / 3,7 / 5,1 / 1,7 / 1,2 / 0,6 / 0,4 % ; k = +1 à +8 : 8,6 /
  8,3 / 5,7 / 5,3 / 5,1 / 5,0 / 4,9 / 0,0 %. C'est le hasard de l'oracle de cadrage : un gain de
  fermeture qui ne le dépasse pas n'est pas une preuve (D-80, D-99).

## 5. Vecteurs et mutations

Vecteurs construits d'après l'écrivain, un qui respecte et un qui viole chaque règle :
`ecrivain_invariants_test.go` (`TestOrdreDeLaVueB`, `TestMasqueEcrit`,
`TestMasqueHorsArchetypeALaTraversee`, `TestVueCEcrite`, `TestPremiereRegleEtEnsemble`,
`TestVerdictDeVueC`, `TestPaquetFermeAuBitQuiContreditLOrdre`,
`TestPaquetFermeAuBitQuiContreditLeMasque`), `debut_par_fermeture_test.go`,
`frame_closure_detail_test.go` (sortie par rejet), `frame_closure_temoins_test.go`,
`source/bits_deborde_test.go`, `cmd_fermeture/v2_research_test.go` (classes de naissance).

Mutations jouées (`l0_tsv/mutations.sh`, résultat `l0_tsv/mutations.txt`) : chaque règle retirée
l'une après l'autre (M1 à M10), la définition retirée (M11), l'ancienne règle de
`debutParFermeture` (M12) : **12 / 12 ROUGES**.

Corrections du contrôle (2026-10-02) : le contrôleur indépendant a rejoué M1 à M12 et ajouté trois
mutations de contrôle ; M15 (témoins décalés jugés sans les règles de la vue C,
`return true` dans `vueCFermeDepuis`) restait **VERTE** : aucun test ne la voyait. Ajouts :
`TestTemoinDecaleJugeParLesReglesDeLaVueC` (`frame_closure_temoins_test.go` : une vue C d'une entrée
d'index `0b11100`, relue depuis +4, lit `kind` 3 puis son terminateur et ferme au bit près ; le témoin
doit rester à 0) et `TestLeRepliDuDebutFermeAuBitEstCompte` (`debut_par_fermeture_test.go`). Rejeu par
overlay (`go test -overlay`, script du contrôleur étendu) : **16 / 16 ROUGES** — M1 à M12, M13 (second
rang retiré), M14 (`rejetVueB` jamais posé), M15 (désormais rouge par le test neuf) et M16 (appel du
compteur du repli retiré) ; la base est verte.

## 6. Révisions

- `grammar.Rev` : `grammar-2026-09-27.3` → **`grammar-2026-10-02`**, entrée dans `rev_chronique.go`,
  empreinte régénérée (`3ddf9961…`), puis à révision constante aux corrections du contrôle (`f5ec3c5f…`).
- `source.Rev` inchangée (`source-2026-09-16.2`) : un accesseur neuf, aucune valeur lue changée ;
  golden à révision constante (empreinte `08e339b2…`), historique écrit.
- `killsource.Rev` inchangée (`killsource-2026-09-27`) : son empreinte bouge par la valeur de
  `grammar.Rev`, mais sa sortie est identique à l'octet (§7) ; golden à révision constante
  (`3d8c497b…`), historique écrit. `objectives.Rev` : non concernée (sa fermeture ne rencontre que
  `source`, dont la valeur ne bouge pas).
- `replay.SchemaVersion` inchangé (76) : le document ne change que par les révisions de calque
  (`grammar.Rev`) et les compteurs de couverture du tir continu, que la révision de grammaire des
  calques signale déjà (verdict « à recuire : redécoder ») ; aucun champ neuf.
- Régénérés : `frame_closure.golden` (historique dans `frame_closure_ratchet_test.go`),
  `types/testdata/shapes.golden` (ligne des révisions), fixtures de contrat
  `replay_schema_76_*.json.gz` + `manifest.json` (8 films : identiques hors chaîne
  `grammar-2026-…`, vérifié par `jq -S` et substitution).

## 7. Gates

**Bilan corrigé (contrôle du 2026-10-02) : les gates ne sont PAS tous verts.** (1) Le banc de vérité
sort en MANQUE sur P-1 (ci-dessous) : P-1 compte désormais les paquets fermés selon la nouvelle
définition, la baisse est la requalification D2 elle-même ; il n'a été joué que sur 2 films.
(2) `make replay-corpus-gate` n'avait pas été joué par le lot ; il l'est aux corrections (§7.1) et
sort FAUX sur les 19 témoins, pour P-1 et pour le repli neuf seulement.
(3) Le compteur `repli_liaison_par_anticipation` bouge sur `1c4c63c2` (1 440 → 1 414) : ce n'est
pas une étape de `replay-equiv` à part, il vit dans l'étape `artifact` (`coverage.fallbacks`) ; la
marche de ce film change de tête sur des listes (`movementStates` diverge), donc lit d'autres
records et pose d'autres liaisons par anticipation (cause estimée, non instruite paquet par paquet).
Les « 56 autres étapes identiques » restent exactes ; « aucun changement non expliqué » ne l'était
pas sans cette ligne.

Joués sur l'arbre du lot (depuis `apps/go-api`, `GOCACHE` dédié, une commande `go` à la fois) :

| Gate | Sortie |
|---|---|
| `gofmt -l ./internal/games/halo_infinite/film/ ./cmd/` | vide |
| `go vet ./internal/games/halo_infinite/film/...` | rc 0, aucune sortie |
| `go vet -tags=research ./...` | rc 0, aucune sortie |
| `go test ./internal/archlint/` | `ok` (ratchets de taille, de tri, du tag `research`, chemins `.ai` cités) |
| `go test ./internal/games/halo_infinite/film/...` | 17 paquets `ok` (dont `grammar` 27,6 s, `replay` 25,0 s, `revision`, `types`, `source`, `facts/killsource`) |
| `go test -tags=research ./internal/games/halo_infinite/film/research/cmd_fermeture/` | `ok` |
| Mutations (§5) | 12 / 12 rouges |

**Gate 2 — carte v2, 20 films** : §8. Aucune baisse saine, ni en paquets ni en records utiles, sur
aucun film ; les baisses en fermés sont les fermetures factices retirées (exception D2) : 8 743
retirées directement par une règle, 5 de plus (`1c4c63c2`, toutes contredites dans la référence)
devenues listes non localisées.

**Gate 3 — killsource** : `cmd/killsource json <film> -carte <carte> -cache … -catalogue …` sur les
19 témoins de `config/replay_corpus.toml`, binaire de la tête contre binaire du lot : **19 / 19
identiques à l'octet** (`1c4c63c2` sans carte lisible, non joué, comme R-COMB-2).

**Gate 6 — `replay-equiv`** (références `film/replay/testdata/equivalence`, 20 films, racine de
dépôt factice dans le scratchpad : copie des 20 films, de `data/titles/halo_infinite/reference` et
de `config/` ; verrou et journaux hors des données du dépôt) :
- tête `ff42fcf40` : **20 / 20 identiques** ;
- lot L0 : **20 / 20 différents, sur 5 étapes seulement sur 61** (`l0_tsv/replay_equiv_etapes_divergentes.tsv`) :
  - `continuousFire.stats` (20 films) : les compteurs du tir continu — les paquets fermés de façon
    factice deviennent des trous ;
  - `continuousFire` (9 films) : les rafales lues sur ces paquets disparaissent : `1c4c63c2`
    399 → 196, `084a804d` 158 → 125, `111fa685` 46 → 32, `e5adf7b2` 62 → 48, `11de8353` 41 → 37,
    `d9781168` 6 → 1, `9f57c612` 6 → 2, `60ae07c4` 3 → 0, `696a9d7c` 16 → 15 ;
  - `movementStates` (3 films) et `movementStates.stats` (4 films) : `debutParFermeture` choisit
    une autre tête (`084a804d` 12 556 → 12 552 états, `1c4c63c2` 10 384 → 10 381, `e5adf7b2`
    5 932 → 5 931 ; `7344d24f` : compteurs seuls) ;
  - `artifact` (20 films) : les révisions de calque (`grammar-2026-10-02`) et les quatre étapes
    ci-dessus. Les 56 autres étapes (positions, morts, killsource, véhicules, objectifs…) sont
    identiques à l'octet sur les 20 films. Aucun changement non expliqué.
- Durées et pics (avant → après) : semblables (± 5 %), sauf le pic de `1c4c63c2` 1,79 → 2,11 Gio
  (+18 %) — `debutParFermeture` essaie plus de candidats quand le premier ferme de façon factice ;
  le plafond du gate 4 ne s'applique pas à L0 (plan §6.0), mais l'écart est publié. Les durées
  d'après ont été prises machine chargée (tests de mutation en parallèle).

**Banc de vérité** (`cmd/replay-verite`, invocation de `docs/COMMANDS.md`) sur deux témoins, un
artefact cuit par film et par révision, un processus à la fois (`cmd/replay-build --facts`, faits du
dossier d'équivalence ; registre des replis de la tête) :
- `e5adf7b2` : **MANQUE**, une seule ligne : `P-1 paquets fermes au bit pres : 4285 -> 4123` ;
- `1c4c63c2` : **MANQUE**, une seule ligne : `P-1 … : 19354 -> 13333` ; information : repli
  `repli_liaison_par_anticipation` 1 440 → 1 414.
- Lecture : P-1 est `coverage.continuousFire.closed`, qui compte désormais les paquets FERMÉS (règles
  tenues) et non plus fermés au bit près : la baisse EST la requalification D2, et elle égale la
  carte (4 285 → 4 123 ; 19 354 → 13 333). Aucun oracle (kills, morts, assistances, équipes, vies) ni
  aucune classe de violation ne bouge. Le libellé de P-1 est désormais inexact (D-L0-4).
- Non joué sur les 18 autres témoins : une cuisson par film et par révision serait une cuisson en
  lot (interdit du lot) ; `replay-corpus-gate` créerait un worktree de base (interdit du lot).

### 7.1 Gates rejoués aux corrections du contrôle (2026-10-02)

Arbre corrigé (D-A à D-E), depuis `apps/go-api`, une commande `go` à la fois :
`gofmt -l` (fichiers touchés) vide ; `go vet` film et `go vet -tags=research` film rc 0 ;
`go test ./internal/archlint/` ok ; G-film (`film/...`, `replaybuild/...`, `sync/killcollector/...`)
19 paquets ok ; empreinte `grammar_rev` régénérée À RÉVISION CONSTANTE `grammar-2026-10-02`
(`3ddf9961…` → `f5ec3c5f…` : le code de la couche change, sa sortie hors `coverage.fallbacks` non) ;
mutations 16 / 16 rouges (§5).

**`replay-equiv`** (20 films, racine factice du lot `scratchpad/L0/repo` : le worktree ne porte
aucun film ; références d'équivalence et `config/` identiques à l'octet à celles du worktree) :
20 / 20 différents de la référence, sur EXACTEMENT les mêmes étapes que le lot (`continuousFire.stats`
20, `continuousFire` 9, `movementStates` 3, `movementStates.stats` 4, `artifact` 20). Contre les
digests du lot, la seule étape qui change est `artifact`, sur les 20 films, de 54 à 57 unités : 53 +
le nombre de chiffres du compte, la taille d'une entrée `coverage.fallbacks` de plus (déduit par la
taille, estimé : le contenu octet par octet n'est pas conservé par l'outil). Déclenchements de
`repli_debut_de_liste_ferme_au_bit` (journal `rejeu : repli declenche`) :

| Film | Listes au second rang | Film | Listes au second rang |
|---|---|---|---|
| 000d5950 | 2 | 60ae07c4 | 103 |
| 01e1f945 | 9 | 64e8adfa | 59 |
| 084a804d | 457 | 696a9d7c | 17 |
| 111fa685 | 138 | 7344d24f | 25 |
| 11de8353 | 125 | 9f57c612 | 112 |
| 1c4c63c2 | 6 039 | a349fea8 | 42 |
| 50247b26 | 14 | a521164d | 4 |
| 51101d1d | 1 | bcb6d393 | 4 |
| 53ce4390 | 18 | d9781168 | 254 |
| e5adf7b2 | 158 | fb1a1a72 | 39 |

Le repli se déclenche sur les 20 films (7 620 listes en tout) : il n'est pas candidat au retrait de
la règle 4 de D-10. `repli_liaison_par_anticipation` sur `1c4c63c2` : 1 414 (identique au lot).

**killsource** (`cmd/killsource json`, binaire de la tête `ff42fcf40` par overlay de tous les `.go`
modifiés ou neufs du lot, contre le binaire corrigé) : `e5adf7b2` (Fragmentation, 314 183 octets) et
`084a804d` (Fortitude Heavies, 554 265 octets) **identiques à l'octet**, et identiques aux sorties de
tête du contrôleur.

**`replay-corpus-gate`** (`--reference=base`, 19 témoins de `config/replay_corpus.toml`) : rc 1, les
19 témoins au statut **FAUX** (rapport `scratchpad/corr/l0_corpus_gate.json`). Deux écarts à la
commande demandée, nécessaires : (a) `--base=ff42fcf40` explicite — sans lui la base est
`origin/feat/v75`, qui a avancé (`756de5b71`, 82 fichiers de production d'écart avec la tête du lot)
et mêlerait d'autres changements à ceux du lot ; (b) `--parc-root` sur une copie du parc dans le
scratchpad (base partagée, `metadata.duckdb`, films et manifestes des 19 témoins, copiés en lecture)
— la gate écrit son cache d'artefacts de base et son verrou sous `data/cache` du parc, c'est-à-dire
dans le checkout principal, qui devait rester en lecture seule. Le worktree de base (sous la racine de
travail du scratchpad) a été créé puis retiré par l'outil (`git worktree list` propre). Aucune cuisson
du parc.

Ce qui fait FAUX, sur les 19 témoins et rien d'autre :
- banc de vérité, `P-1 paquets fermes au bit pres` en **MANQUE** sur 19 / 19 (la requalification D2,
  D-L0-4) ;
- banc de vérité, `R-1 repli repli_debut_de_liste_ferme_au_bit` **FAUX** (« repli nouveau ») sur 19 / 19 :
  le banc classe tout repli NOMMÉ pour la première fois en défaut, c'est la conséquence directe de D-A
  (D-L0-5) ;
- information seulement : `repli_liaison_par_anticipation` 586 → 587 (`111fa685`), 499 → 501
  (`11de8353`), 299 → 297 (`4f77afc1`) ; `repli_physique_de_type_de_vehicule_supposee`
  14 116 → 14 117 (`4f77afc1`).

Métriques en perte : le tir continu sur les 19 témoins (`coverage.continuousFire.closed`, `holes`,
`entries`, `withAction`… ; rafales `bursts.*` sur 5 témoins) — la requalification D2 ;
`coverage.fallbacks/n` +1 sur 19 (le nom neuf) ; les états de mouvement sur 3 témoins (`084a804d`
15 métriques, `4f77afc1` 10, `e5adf7b2` 2 : `stances`, `jumpDerived`, `sprint`), par la tête de
liste que choisit `debutParFermeture`, comme au `replay-equiv`. Aucune perte hors de ces trois
familles : ni positions, ni morts, ni killsource, ni identités, ni objectifs, ni véhicules. Les
changements (8 témoins) sont du même ordre (`continuousFire.otherInput`, `onFoot`, `onVehicle`,
`notContinuous` ; `stances.byKind` sur `084a804d` et `4f77afc1`).


## 8. Carte v2, avant / après, par film

« Avant » = la marche de référence (tête `ff42fcf40`, carte identique à la phase 1 sur les 10 TSV),
lue par l'instrument L0 définition neutralisée ; « après » = la définition L0. Comparaison paquet par
paquet (`l0_tsv/comparer.awk`, `net_sains.awk`). « Sains » = fermés sans règle contredite.

| Film | Build | Fermés avant | Fermés après | Retirés par une règle (D2) | Autres baisses (factices, via la marche) | Gains | Sains avant | Sains après | Sains perdus | Utiles fermés avant | Utiles fermés après | Utiles retirés (D2) | Utiles sains avant | Utiles sains après | Utiles lus avant | Utiles lus après |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 0797ce72 | HI_1_13_0 | 19205 | 19152 | 53 | 0 | 0 | 19152 | 19152 | 0 | 160230 | 159920 | 310 | 159920 | 159920 | 205685 | 205685 |
| 084a804d | HI_1_10_0 | 5235 | 4773 | 462 | 0 | 0 | 4761 | 4773 | 0 | 89145 | 88036 | 1032 | 88028 | 88036 | 535881 | 535804 |
| 111fa685 | HI_1_10_0 | 4152 | 4012 | 140 | 0 | 0 | 4010 | 4012 | 0 | 42832 | 42601 | 233 | 42599 | 42601 | 245554 | 245556 |
| 11de8353 | HI_1_9_0 | 5739 | 5612 | 127 | 0 | 0 | 5611 | 5612 | 0 | 65452 | 65230 | 224 | 65228 | 65230 | 248285 | 248287 |
| 1c4c63c2 | HI_1_10_0 | 19354 | 13333 | 6033 | 5 | 17 | 12770 | 13333 | 0 | 170685 | 165404 | 5066 | 164868 | 165404 | 777616 | 777075 |
| 396cfc92 | HI_1_13_0 | 22828 | 22819 | 9 | 0 | 0 | 22819 | 22819 | 0 | 166999 | 166984 | 15 | 166984 | 166984 | 225495 | 225495 |
| 4f77afc1 | HI_1_13_0 | 22910 | 22260 | 1004 | 0 | 354 | 21823 | 22260 | 0 | 543450 | 550289 | 4804 | 537798 | 550289 | 735384 | 738490 |
| 50247b26 | version-31 | 153 | 139 | 14 | 0 | 0 | 139 | 139 | 0 | 277 | 274 | 3 | 274 | 274 | 319053 | 319053 |
| 51ebbc0f | HI_1_13_0 | 9847 | 9759 | 88 | 0 | 0 | 9759 | 9759 | 0 | 58750 | 58585 | 165 | 58585 | 58585 | 88493 | 88493 |
| 60ae07c4 | HI_1_8_0 | 13969 | 13802 | 167 | 0 | 0 | 13801 | 13802 | 0 | 83225 | 82730 | 495 | 82728 | 82730 | 268352 | 268352 |
| a349fea8 | version-33 | 463 | 420 | 43 | 0 | 0 | 420 | 420 | 0 | 3602 | 3595 | 7 | 3595 | 3595 | 469451 | 469451 |
| a521164d | HI_1_4_1 | 701 | 692 | 9 | 0 | 0 | 692 | 692 | 0 | 80 | 78 | 2 | 78 | 78 | 181607 | 181607 |
| bcb6d393 | HI_1_12_0 | 5835 | 5830 | 5 | 0 | 0 | 5830 | 5830 | 0 | 35158 | 35143 | 15 | 35143 | 35143 | 121628 | 121628 |
| bf15f7ab | HI_1_13_0 | 28476 | 28465 | 11 | 0 | 0 | 28465 | 28465 | 0 | 214998 | 214974 | 24 | 214974 | 214974 | 227637 | 227637 |
| bfecd02b | HI_1_13_0 | 26403 | 26380 | 23 | 0 | 0 | 26380 | 26380 | 0 | 226580 | 226529 | 51 | 226529 | 226529 | 254963 | 254963 |
| c75f33b8 | HI_1_13_0 | 22944 | 22854 | 90 | 0 | 0 | 22854 | 22854 | 0 | 144577 | 144430 | 147 | 144430 | 144430 | 155136 | 155136 |
| d9781168 | HI_1_13_0 | 26464 | 26210 | 254 | 0 | 0 | 26210 | 26210 | 0 | 180366 | 180052 | 314 | 180052 | 180052 | 244872 | 244872 |
| e5adf7b2 | HI_1_11_0 | 4285 | 4123 | 162 | 0 | 0 | 4120 | 4123 | 0 | 80032 | 79457 | 554 | 79402 | 79457 | 289378 | 289357 |
| f75e7053 | HI_1_13_0 | 23423 | 23416 | 7 | 0 | 0 | 23416 | 23416 | 0 | 160051 | 160042 | 9 | 160042 | 160042 | 187304 | 187304 |
| fb1a1a72 | HI_1_13_0 | 22318 | 22276 | 42 | 0 | 0 | 22276 | 22276 | 0 | 161678 | 161566 | 112 | 161566 | 161566 | 179254 | 179254 |
| **corpus** | | 284704 | 276327 | 8743 | 5 | 371 | 275308 | 276327 | 0 | 2588167 | 2585919 | 13582 | 2572823 | 2585919 | 5961028 | 5963499 |


Lecture : « retirés par une règle » = fermés au bit dont la lecture contredit l'écrivain (exception
D2) ; « autres baisses » = 5 paquets de `1c4c63c2`, tous contredits dans la référence (factices),
devenus « liste non localisée » ; **aucun paquet sain perdu, aucun film en baisse en paquets sains
ni en records utiles sains, sous le juge de L0** — sous l'ancien juge à trois invariants (ordre,
masque, vue C, `cmContredit`), la règle L0.6 requalifie environ 1 008 paquets sains en non fermés (924
sur `e5adf7b2`, `4f77afc1` et `1c4c63c2`, mesure du contrôleur), et c'est l'objet même de D2 / L0.6
(§1.1) (gain net : +1 019 paquets sains, +13 096 utiles sains ; `4f77afc1`
+437 / +12 491, `1c4c63c2` +563 / +536). Effet sur la marche (N5) : records utiles lus 5 961 028 →
5 963 499 (+2 471 : `4f77afc1` +3 106, `1c4c63c2` −541, `084a804d` −77, `e5adf7b2` −21) ; listes
non localisées 48 720 → 48 688.

## 9. Écarts et découvertes

- Écart : `debutParFermeture` à deux rangs (§3), décision prise dans le lot pour tenir le gate 2 ;
  la forme pure perd 29 paquets sains. Tranché par le pilote (D-A, corrections du contrôle) : gardé,
  comme repli nommé et compté (`repli_debut_de_liste_ferme_au_bit`, §3).
- Écart : L0.8 non retenu (§1.3).
- D-L0-1 : mots de DEL non nuls et structurés sur des slots que le monde lie à `ti=5`, `ti=8`,
  `ti=18`, `ti=25` (slots 52-55, 117, 119, 246) dans des paquets sains : liaison d'archétype du monde
  probablement fausse pour ces slots (`FUN_142f304a8` n'écrit un mot que pour l'archétype 0x10). Non
  instruit.
- D-L0-2 : `e5adf7b2` 10:20 (et ses semblables) : un NEW `ti=41` dont l'en-tête est juste et le corps
  mal lu (masque au-delà de l'archétype), lu comme tête de liste. Piste : l'état par défaut de
  `ti=41` dans un record NEW. Non instruit.
- D-L0-3 : le commentaire de `frame_chain_infer.go` (« un masque trop large est inoffensif, pas un
  indice de mauvaise lecture ») contredit `FUN_142e2da44` ; chemin d'inférence hors production, non
  touché.
- Écart visible au rejeu : le calque des rafales du tir continu perd les rafales lues sur des
  paquets fermés de façon factice (9 films sur 20, `1c4c63c2` 399 → 196) ; c'est l'effet attendu de
  D2 (le tir continu ne lit que des paquets fermés), à signaler avant la recuisson de la vague.
- Écart : pic mémoire de cuisson de `1c4c63c2` +18 % (§7).
- D-L0-4 : le banc de vérité nomme P-1 « paquets fermés au bit près » et lit
  `coverage.continuousFire.closed`, qui compte depuis L0 les paquets fermés (règles tenues). Tout lot
  qui change la définition fait sortir le banc en MANQUE par construction. Non traité (hors lot).
- Écart aux interdits : un fichier VIDE écrit par erreur dans `%TEMP%\x` (hors scratchpad), effacé
  dans la session ; une écriture tentée à la racine (`/tmp_unused`) refusée par le système, rien
  créé. Sans effet sur les livrables.
- D-L0-5 (corrections du contrôle) : le banc de vérité (`R-1`) classe FAUX tout repli nommé pour la
  première fois (« repli nouveau »). Nommer et compter un repli existant, ce que D-10 exige, fait
  donc sortir le banc en FAUX par construction, comme D-L0-4 pour P-1. Non traité.
- D-L0-6 (corrections du contrôle) : `replay-equiv` et `replay-corpus-gate` ne tournent pas sur un
  worktree dédié sans données (aucun film sous `data/cache`), et `replay-corpus-gate` écrit son cache
  de base et son verrou sous `data/cache` du parc, sans option pour le cache hors du parc. Joués ici
  sur des racines factices du scratchpad. Non traité.
- Les témoins décalés à +8 rendent 0 sur le corpus : attendu pour une vue C vide (un départ au-delà
  du dernier bit déborde), non vérifié pour les vues C non vides.
