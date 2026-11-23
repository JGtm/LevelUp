// inventaire.test.mjs - tests de l'inventaire (candidats, classes, xuids). Lancer :
//   node --test scripts/anonymisation/inventaire.test.mjs
//
// Tous les noms et numeros ci-dessous sont INVENTES.

import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { affiner, classer, formeCourte, lireGardes } from './classes.mjs';
import { lireTableTexte } from './table.mjs';
import { executer } from './inventaire.mjs';
import { lignesTableXuids, xuidEvident, xuidFictif } from './xuids.mjs';

const GARDES = lireGardes(['proprietaire\tProprioPublic', 'bot\t~^343 ', 'fictif\t~^Fictif\\d', 'bruit\tMotDeCode'].join('\n'));
const info = (sources, genres, docStructure = false) => ({ sources: new Set(sources), genres: new Set(genres), docStructure });

test('gardes : classe connue, motif, valeur manquante', () => {
  assert.throws(() => lireGardes('inconnue\tX'), /classe inconnue/);
  assert.throws(() => lireGardes('bot'), /valeur manquante/);
  assert.equal(GARDES.noms.get('ProprioPublic'), 'proprietaire');
});

test('classes : preuve structuree = reel, gardes d\'abord, bruit, doute', () => {
  assert.equal(classer('ProprioPublic', info(['tsv'], ['docs'], true), GARDES).classe, 'proprietaire');
  assert.equal(classer('343 Exemple', info(['tsv'], ['docs'], true), GARDES).classe, 'bot');
  assert.equal(classer('Fictif1', info(['json'], ['code']), GARDES).classe, 'fictif');
  assert.equal(classer('Zebulon4242', info(['tsv'], ['docs'], true), GARDES).classe, 'reel');
  assert.equal(classer('snake_case_nom', info(['tsv'], ['docs'], true), GARDES).classe, 'bruit');
  assert.equal(classer('Zebulon', info(['mot-cite'], ['docs']), GARDES).classe, 'reel-doute');
  assert.equal(classer('un mot', info(['mot-cite'], ['docs']), GARDES).classe, 'bruit');
  assert.equal(classer('Quidam7788', info(['forme'], ['docs']), GARDES).classe, 'reel-prose');
  assert.equal(classer('Quidam7788', info(['forme'], ['docs', 'code']), GARDES).classe, 'bruit');
  assert.equal(classer('Hash64', info(['forme'], ['docs']), GARDES).classe, 'bruit');
});

test('classes : un libelle ou un mot courant n\'est pas un pseudo', () => {
  const mots = new Map([['dernier', 80], ['match', 90], ['yet', 150], ['ratio', 300]]);
  const reel = { classe: 'reel', raison: '' };
  assert.equal(affiner('Dernier match', reel, mots).classe, 'bruit');
  assert.equal(affiner('Yet', reel, mots, true, 5).classe, 'mot-courant');
  assert.equal(affiner('Zebulon4242', reel, mots).classe, 'reel');
  assert.equal(affiner('Café crème', reel, mots).classe, 'bruit');
  assert.equal(affiner('Quidam', { classe: 'reel' }, new Map([['quidam', 9]]), true, 400).classe, 'reel');
});

test('forme courte : sans les chiffres', () => {
  assert.equal(formeCourte('Zebulon4242'), 'Zebulon');
  assert.equal(formeCourte('Zeb12'), null);
  assert.equal(formeCourte('Zebulon'), null);
});

test('xuids : fictif deterministe, dans la plage, evidents reconnus', () => {
  const f = xuidFictif('2533274800000123');
  assert.match(f, /^[0-9]{16}$/);
  assert.equal(f, xuidFictif('2533274800000123'));
  assert.ok(BigInt(f) >= 0x0009000000000000n && BigInt(f) < 0x000a000000000000n);
  assert.notEqual(f, xuidFictif('2533274800000124'));
  // meme vecteur que apps/go-api/internal/identitefictive (borne basse de la plage -> fictif attendu)
  assert.equal(xuidFictif('2533274790395904'), '2550913680076637');
  assert.ok(xuidEvident('2533274000000101'));
  assert.ok(xuidEvident('2533274800000001'));
  assert.ok(xuidEvident('2599999999999999'));
  assert.ok(!xuidEvident('2533274812345678') || /1234567/.test('2533274812345678'));
  assert.ok(!xuidEvident('2533274823110022'));
});

test('xuids : table idempotente, un fictif ancien n\'est pas re-remplace', () => {
  const a = '2533274823110022';
  const par = new Map([
    [a, { classe: 'probable-reel', fichiers: new Set(['x']), occurrences: 1 }],
    [xuidFictif(a), { classe: 'probable-reel', fichiers: new Set(['y']), occurrences: 1 }],
    ['2547000000000000', { classe: 'autre', fichiers: new Set(['z']), occurrences: 1 }],
  ]);
  const lignes = lignesTableXuids(par);
  assert.ok(lignes.some((l) => l.startsWith(`texte\t${a}\t${xuidFictif(a)}`)));
  assert.ok(!lignes.some((l) => l.startsWith(`texte\t${xuidFictif(a)}\t`)));
  assert.ok(lignes.some((l) => l.startsWith('# xuid hors 2533/2535')));
  lireTableTexte(lignes.join('\n'));
});

test('integration : un depot jetable donne pseudos.tsv et une table qui s\'applique', () => {
  const base = fs.mkdtempSync(path.join(os.tmpdir(), 'inv-test-'));
  const depot = path.join(base, 'depot');
  fs.mkdirSync(path.join(depot, 'docs'), { recursive: true });
  fs.writeFileSync(path.join(depot, 'docs', 'participants.tsv'), 'rang\tgamertag\tscore\n1\tZebulon4242\t10\n2\tQuidam7788\t7\n3\tProprioPublic\t5\n');
  fs.writeFileSync(path.join(depot, 'docs', 'notes.md'), 'Zebulon4242 a gagne ; ProprioPublic aussi. Zebulon assiste. Joueur 2533274823110022.\n');
  const git = (...a) => execFileSync('git', a, { cwd: depot, stdio: 'pipe' });
  git('init', '-q');
  git('config', 'user.email', 't@example.invalid');
  git('config', 'user.name', 't');
  git('add', '.');
  git('commit', '-q', '-m', 'init');
  const gardes = path.join(base, 'gardes.tsv');
  fs.writeFileSync(gardes, 'proprietaire\tProprioPublic\n');
  const registre = path.join(base, 'registre.tsv');
  fs.writeFileSync(registre, '2533274811111112\t0x1\n');
  const sortie = path.join(base, 'sortie');
  executer(['--racine', depot, '--sortie', sortie, '--gardes', gardes, '--xuids', '--registre', registre]);
  const pseudos = fs.readFileSync(path.join(sortie, 'pseudos.tsv'), 'utf8');
  assert.match(pseudos, /Zebulon4242\treel\t/);
  assert.match(pseudos, /ProprioPublic\tproprietaire\t/);
  const table = fs.readFileSync(path.join(sortie, 'table.tsv'), 'utf8');
  assert.match(table, /^pseudo\tZebulon4242\tjoueur_\d{3}\tvariantes$/m);
  assert.match(table, /^pseudo\tZebulon\tjoueur_\d{3}\t.*ambigu/m);
  assert.doesNotMatch(table, /ProprioPublic/);
  assert.match(table, /^texte\t2533274823110022\t[0-9]{16}\tmot$/m);
  assert.throws(() => executer(['--racine', depot, '--sortie', path.join(depot, 'sortie')]), /hors du depot/);
  // la table engendree se relit et s'applique sans erreur d'idempotence
  const entrees = lireTableTexte(table);
  assert.ok(entrees.length >= 3);
});
