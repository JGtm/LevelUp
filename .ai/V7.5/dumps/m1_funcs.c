// ============================================================================
// M1 — Fonctions de fondation du décodeur ECS du film Halo (décompilation Ghidra
// statique de HaloInfinite.exe, image_base 0x140000000). Pour reconstruction de
// la spec bit-exacte du BitReader + grammaire record statborg.
//
// BitReader = la struct passée en param (souvent param_2). Offsets observés :
//   +0x10 = pointeur fin de buffer (end)
//   +0x28 = compteur de bits refillés (bookkeeping)
//   +0x2c = compteur de bits consommés (bookkeeping)
//   +0x30 = accumulateur 64-bit ; bits consommés depuis le HAUT (MSB-first) par shift gauche
//   +0x38 = nb de bits déjà consommés de l'accumulateur (0..0x40) ; refill quand il manque des bits
//   +0x40 = curseur octet dans le buffer
// Refill : lit 8 octets au curseur en u64 PUIS byte-swap big-endian (1er octet -> MSB).
//          Si <8 octets restants : lecture octet-par-octet en big-endian puis shift.
// ============================================================================


//////// 0x1406d6c7c FUN_1406d6c7c — read-N-bits MSB-first (primitif générique) ////////
// Appel : FUN_1406d6c7c(BitReader* br, int n) -> ulonglong (les n bits, alignés en bas)

ulonglong FUN_1406d6c7c(longlong param_1,int param_2)
{
  ulonglong uVar1;
  ulonglong uVar2;
  uint uVar3;
  ulonglong *puVar4;
  ulonglong uVar5;
  uint uVar6;
  ulonglong uVar7;

  puVar4 = *(ulonglong **)(param_1 + 0x40);
  uVar5 = 0;
  uVar1 = *(ulonglong *)(param_1 + 0x30);
  uVar6 = 0;
  if (*(ulonglong **)(param_1 + 0x10) < puVar4 + 1) {
    uVar7 = uVar5;
    if (puVar4 < *(ulonglong **)(param_1 + 0x10)) {
      do {
        uVar2 = *puVar4;
        uVar6 = (int)uVar7 + 8;
        uVar7 = (ulonglong)uVar6;
        puVar4 = (ulonglong *)((longlong)puVar4 + 1);
        uVar5 = uVar5 << 8 | (ulonglong)(byte)uVar2;
        *(ulonglong **)(param_1 + 0x40) = puVar4;
      } while (puVar4 < *(ulonglong **)(param_1 + 0x10));
      uVar5 = uVar5 << (0x40U - (char)uVar6 & 0x3f);
    }
  }
  else {
    uVar5 = *puVar4;
    uVar6 = 0x40;
    uVar5 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 | (uVar5 & 0xff0000000000) >> 0x18 |
            (uVar5 & 0xff00000000) >> 8 | (uVar5 & 0xff000000) << 8 | (uVar5 & 0xff0000) << 0x18 |
            (uVar5 & 0xff00) << 0x28 | uVar5 << 0x38;
    *(ulonglong **)(param_1 + 0x40) = puVar4 + 1;
  }
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + param_2;
  uVar3 = param_2 + -0x40 + *(int *)(param_1 + 0x38);
  *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + uVar6;
  *(uint *)(param_1 + 0x38) = uVar3;
  *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar3 < 0x40) & uVar5 << ((byte)uVar3 & 0x3f);
  return uVar1 >> (0x40U - (char)param_2 & 0x3f) | uVar5 >> (0x40 - (byte)uVar3 & 0x3f);
}


//////// 0x1406cf008 FUN_1406cf008 — read 1 bit MSB-first ////////
// Appel : FUN_1406cf008(BitReader* br) -> bool (1 bit)

bool FUN_1406cf008(longlong param_1)
{
  longlong lVar1;
  bool bVar2;

  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    bVar2 = SUB81((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x3f,0);  // top bit
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;  // shift left 1
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
  }
  else {
    lVar1 = FUN_1406d6c7c(param_1,1);   // refill path -> read 1 bit via generic
    bVar2 = lVar1 != 0;
  }
  return bVar2;
}


//////// 0x140c18a1c FUN_140c18a1c — read signed var-width int (prefixe 2-bit) ////////
// Appel : FUN_140c18a1c(_, BitReader* br) -> int32 (sign-extended)
// Logique : lit 2 bits (selecteur sel) ; w = 8 << sel (8/16/32/64) ; lit w bits MSB-first ;
//           sign-extend si w < 32. (Refill bswap64 inline, branche <8 octets gérée.)

ulonglong FUN_140c18a1c(undefined8 param_1,longlong param_2)
{
  ulonglong uVar1;
  ulonglong *puVar2;
  uint uVar3;
  int iVar4;
  uint uVar5;
  int iVar6;
  byte bVar7;
  ulonglong uVar8;
  ulonglong uVar9;
  ulonglong uVar10;

  iVar6 = 0x40 - *(int *)(param_2 + 0x38);
  uVar9 = 0;
  uVar3 = 0;
  bVar7 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (iVar6 < 2) {
    puVar2 = *(ulonglong **)(param_2 + 0x40);
    if (*(ulonglong **)(param_2 + 0x10) < puVar2 + 1) {
      uVar8 = uVar9;
      uVar10 = uVar9;
      uVar5 = uVar3;
      if (puVar2 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar5 = (int)uVar8 + 8;
          uVar8 = (ulonglong)uVar5;
          uVar1 = *puVar2;
          puVar2 = (ulonglong *)((longlong)puVar2 + 1);
          uVar10 = (ulonglong)(byte)uVar1 | uVar10 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar2;
        } while (puVar2 < *(ulonglong **)(param_2 + 0x10));
        uVar10 = uVar10 << (-(char)uVar5 & 0x3fU);
      }
    }
    else {
      uVar8 = *puVar2;
      *(ulonglong **)(param_2 + 0x40) = puVar2 + 1;
      uVar10 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
               | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 |
               (uVar8 & 0xff0000) << 0x18 | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      uVar5 = 0x40;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + uVar5;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    uVar5 = 2 - iVar6;
    iVar6 = *(int *)(param_2 + 0x2c);
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar5 < 0x40) & uVar10 << ((byte)uVar5 & 0x3f);
    bVar7 = (byte)(uVar10 >> (-(byte)uVar5 & 0x3f)) | bVar7 >> 6;
    *(uint *)(param_2 + 0x38) = uVar5;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    iVar6 = *(int *)(param_2 + 0x2c);
    uVar5 = *(int *)(param_2 + 0x38) + 2;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) * 4;  // consume 2 bits
    *(uint *)(param_2 + 0x38) = uVar5;
    bVar7 = bVar7 >> 6;   // sel = top 2 bits
  }
  uVar8 = *(ulonglong *)(param_2 + 0x30);
  iVar4 = 8 << (bVar7 & 0x1f);   // w = 8 << sel
  bVar7 = (byte)iVar4;
  if ((int)(0x40 - uVar5) < iVar4) {
    puVar2 = *(ulonglong **)(param_2 + 0x40);
    if (*(ulonglong **)(param_2 + 0x10) < puVar2 + 1) {
      uVar10 = uVar9;
      if (puVar2 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar1 = *puVar2;
          uVar3 = (int)uVar9 + 8;
          uVar9 = (ulonglong)uVar3;
          puVar2 = (ulonglong *)((longlong)puVar2 + 1);
          uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar1;
          *(ulonglong **)(param_2 + 0x40) = puVar2;
        } while (puVar2 < *(ulonglong **)(param_2 + 0x10));
        uVar9 = uVar10 << (-(char)uVar3 & 0x3fU);
      }
    }
    else {
      uVar9 = *puVar2;
      uVar3 = 0x40;
      uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 | (uVar9 & 0xff0000000000) >> 0x18
              | (uVar9 & 0xff00000000) >> 8 | (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18
              | (uVar9 & 0xff00) << 0x28 | uVar9 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar2 + 1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + uVar3;
    *(int *)(param_2 + 0x2c) = iVar6 + iVar4;
    uVar5 = iVar4 + -0x40 + uVar5;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar5 < 0x40) & uVar9 << ((byte)uVar5 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar5;
    uVar8 = uVar8 >> (-bVar7 & 0x3f) | uVar9 >> (-(byte)uVar5 & 0x3f);
  }
  else {
    *(int *)(param_2 + 0x2c) = iVar6 + iVar4;
    *(ulonglong *)(param_2 + 0x30) = uVar8 << (bVar7 & 0x3f);
    *(uint *)(param_2 + 0x38) = iVar4 + uVar5;
    uVar8 = uVar8 >> (-bVar7 & 0x3f);
  }
  if ((iVar4 < 0x20) && (((uint)uVar8 >> (iVar4 - 1U & 0x1f) & 1) != 0)) {
    uVar8 = (ulonglong)((uint)uVar8 | -(1 << (bVar7 & 0x3f)));   // sign-extend
  }
  return uVar8 & 0xffffffff;
}


//////// 0x140c18794 FUN_140c18794 — STATBORG deser (record 2-equipes) ////////
// Appel : FUN_140c18794(binding* param_1, BitReader* param_2, target* param_3)
//   param_1[8] = index statline A ; param_1[0xc] = index statline B
//   param_3+0x10 = base du tableau de stats (stat i : valeur u32 a base+i*8, flag a base+i*8+4)
//   base+0x1c0 = dirty bits ; base+0x1c8+i*4 = valeur conditionnelle
// Grammaire : [5b headerA][5b headerB][valA][valB][1b flagA][1b flagB][valA2 si flagA][valB2 si flagB]
//   (les 2 premiers reads de 5 bits sont inline ; valA/valB/valA2/valB2 = FUN_140c18a1c ; flags = FUN_1406cf008)

undefined8 FUN_140c18794(ulonglong param_1,longlong param_2,longlong param_3)
{
  longlong lVar1;
  char cVar2;
  char cVar3;
  undefined4 uVar4;
  ulonglong uVar5;
  byte bVar6;
  ulonglong uVar7;
  ulonglong uVar8;
  ulonglong *puVar9;
  int iVar10;
  int iVar11;
  byte bVar12;
  byte bVar13;
  uint uVar14;

  lVar1 = *(longlong *)(param_3 + 0x10);
  iVar11 = 0x40 - *(int *)(param_2 + 0x38);
  bVar13 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (iVar11 < 5) {
    puVar9 = *(ulonglong **)(param_2 + 0x40);
    uVar7 = 0;
    iVar10 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar9 + 1) {
      if (puVar9 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          iVar10 = iVar10 + 8;
          uVar8 = *puVar9;
          puVar9 = (ulonglong *)((longlong)puVar9 + 1);
          uVar7 = (ulonglong)(byte)uVar8 | uVar7 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar9;
        } while (puVar9 < *(ulonglong **)(param_2 + 0x10));
        uVar7 = uVar7 << (-(char)iVar10 & 0x3fU);
      }
    }
    else {
      uVar7 = *puVar9;
      iVar10 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar9 + 1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar10;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 5;
    uVar14 = 5 - iVar11;
    iVar11 = *(int *)(param_2 + 0x2c);
    bVar12 = 0x40 - (byte)uVar14;
    uVar8 = (ulonglong)bVar12;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar14 < 0x40) & uVar7 << ((byte)uVar14 & 0x3f);
    bVar13 = (byte)(uVar7 >> (bVar12 & 0x3f)) | bVar13 >> 3;   // headerA = top 5 bits
    *(uint *)(param_2 + 0x38) = uVar14;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 5;
    iVar11 = *(int *)(param_2 + 0x2c);
    uVar14 = *(int *)(param_2 + 0x38) + 5;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 5;
    bVar13 = bVar13 >> 3;
    *(uint *)(param_2 + 0x38) = uVar14;
    uVar8 = param_1;
  }
  bVar12 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if ((int)(0x40 - uVar14) < 5) {
    puVar9 = *(ulonglong **)(param_2 + 0x40);
    uVar7 = 0;
    iVar10 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar9 + 1) {
      if (puVar9 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar8 = *puVar9;
          iVar10 = iVar10 + 8;
          puVar9 = (ulonglong *)((longlong)puVar9 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar8;
          *(ulonglong **)(param_2 + 0x40) = puVar9;
        } while (puVar9 < *(ulonglong **)(param_2 + 0x10));
        uVar7 = uVar7 << (-(char)iVar10 & 0x3fU);
      }
    }
    else {
      uVar7 = *puVar9;
      iVar10 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar9 + 1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar10;
    uVar14 = uVar14 - 0x3b;
    *(int *)(param_2 + 0x2c) = iVar11 + 5;
    bVar6 = 0x40 - (byte)uVar14;
    uVar8 = (ulonglong)bVar6;
    uVar5 = -(ulonglong)(uVar14 < 0x40) & uVar7 << ((byte)uVar14 & 0x3f);
    bVar12 = (byte)(uVar7 >> (bVar6 & 0x3f)) | bVar12 >> 3;   // headerB = top 5 bits
  }
  else {
    uVar14 = uVar14 + 5;
    *(int *)(param_2 + 0x2c) = iVar11 + 5;
    uVar5 = *(longlong *)(param_2 + 0x30) << 5;
    bVar12 = bVar12 >> 3;
  }
  *(ulonglong *)(param_2 + 0x30) = uVar5;
  *(uint *)(param_2 + 0x38) = uVar14;
  *(byte *)(lVar1 + 4 + (longlong)*(int *)(param_1 + 8) * 8) = bVar13;   // headerA -> stat A flag
  *(byte *)(lVar1 + 4 + (longlong)*(int *)(param_1 + 0xc) * 8) = bVar12; // headerB -> stat B flag
  uVar4 = FUN_140c18a1c(uVar8,param_2);                                  // valA
  *(undefined4 *)(lVar1 + (longlong)*(int *)(param_1 + 8) * 8) = uVar4;
  uVar4 = FUN_140c18a1c(uVar4,param_2);                                  // valB
  *(undefined4 *)(lVar1 + (longlong)*(int *)(param_1 + 0xc) * 8) = uVar4;
  cVar2 = FUN_1406cf008(param_2);                                        // flagA (1 bit)
  cVar3 = FUN_1406cf008(param_2);                                        // flagB (1 bit)
  if (cVar2 == '\0') {
    uVar7 = *(ulonglong *)(lVar1 + 0x1c0) & ~(1L << ((longlong)*(int *)(param_1 + 8) & 0x3fU));
  }
  else {
    uVar7 = *(ulonglong *)(lVar1 + 0x1c0) | 1L << ((longlong)*(int *)(param_1 + 8) & 0x3fU);
  }
  *(ulonglong *)(lVar1 + 0x1c0) = uVar7;
  if (cVar3 == '\0') {
    uVar7 = uVar7 & ~(1L << ((longlong)*(int *)(param_1 + 0xc) & 0x3fU));
  }
  else {
    uVar7 = uVar7 | 1L << ((longlong)*(int *)(param_1 + 0xc) & 0x3fU);
  }
  *(ulonglong *)(lVar1 + 0x1c0) = uVar7;
  if (cVar2 != '\0') {                                                   // valA2 si flagA
    uVar4 = FUN_140c18a1c(uVar7,param_2);
    uVar7 = (ulonglong)*(int *)(param_1 + 8);
    *(undefined4 *)(lVar1 + 0x1c8 + uVar7 * 4) = uVar4;
  }
  if (cVar3 != '\0') {                                                   // valB2 si flagB
    uVar4 = FUN_140c18a1c(uVar7,param_2);
    *(undefined4 *)(lVar1 + 0x1c8 + (longlong)*(int *)(param_1 + 0xc) * 4) = uVar4;
  }
  return 1;
}


//////// 0x142ed9298 FUN_142ed9298 — deser simple (flag + valeur optionnelle) ////////
// 2eme appelant de FUN_140c18a1c. Lit 1 bit ; si 0, lit une valeur 2-bit-selecteur.

undefined8 FUN_142ed9298(undefined8 param_1,undefined8 param_2)
{
  char cVar1;
  undefined8 uVar2;

  cVar1 = FUN_1406cf008(param_2);
  if (cVar1 != '\0') {
    return 0;
  }
  uVar2 = FUN_140c18a1c(/*_,*/ param_2);
  return uVar2;
}


//////// 0x140807ebc FUN_140807ebc — APPLY (objet deserialise -> statline monde) ////////
// Recopie l'objet source (param_5) vers world+team*0x1DF0+0x38, boucle 56 stats (stride 0x88 = 0x22 dwords),
// gatee par dirty-bits (param_3). + finalisation 2-equipes (round values) en fin de boucle.

undefined8
FUN_140807ebc(longlong param_1,uint param_2,longlong param_3,int param_4,undefined4 *param_5)
{
  longlong lVar1;
  byte bVar2;
  undefined4 uVar3; undefined4 uVar4; int iVar5; int iVar6; undefined4 uVar7;
  longlong *plVar8; undefined8 uVar9; byte bVar10; char cVar11;
  undefined4 *puVar12; undefined4 *puVar13; longlong lVar14; longlong lVar15;
  ulonglong uVar16; undefined4 *puVar17; uint uVar18; longlong lVar19; ulonglong uVar20; uint uVar21;
  undefined4 extraout_XMM0_Da; undefined4 extraout_XMM0_Da_00;
  longlong local_res8; uint local_res10; longlong local_res18; int local_res20 [2];
  longlong local_88; longlong local_78 [2]; longlong local_68;
  undefined *local_60; undefined4 *local_58; undefined *local_50;

  local_68 = (longlong)(int)param_2;
  lVar1 = local_68 * 0x1df0 + param_1;
  puVar12 = (undefined4 *)(lVar1 + 0x38);
  uVar21 = 0; uVar20 = 0; local_res8 = 0; local_78[0] = 0; local_88 = 0;
  lVar19 = local_68 * 0x1df0 - (longlong)puVar12;
  puVar13 = param_5; local_res10 = param_2; local_res18 = param_3; local_res20[0] = param_4;
  local_58 = puVar12;
  do {
    bVar2 = (byte)(uVar21 >> 1);
    uVar18 = 1 << (bVar2 + 0x1c & 0x1f) & *(uint *)(param_3 + ((ulonglong)(uVar21 >> 1) + 0x1c >> 5) * 4);
    if (uVar18 != 0) {
      bVar10 = 0; puVar17 = puVar12;
      do {
        if ((param_5[0xaa] & 1 << (bVar10 & 0x1f)) != 0) {
          *puVar17 = *(undefined4 *)((longlong)param_5 + ((local_78[0] + 0x2ac) - (longlong)puVar12) + (longlong)puVar17);
        }
        bVar10 = bVar10 + 1; puVar17 = puVar17 + 1;
      } while (bVar10 < 0x20);
      puVar12[0x20] = param_5[0xaa];
    }
    uVar16 = 1L << ((byte)uVar20 & 0x3f) & *(ulonglong *)(param_5 + 0x70);
    if ((1 << (bVar2 & 0x1f) & *(uint *)(param_3 + (ulonglong)(uVar21 >> 6) * 4)) == 0) {
      if ((uVar16 == 0) && (uVar18 != 0)) {
LAB_140808026:
        uVar9 = FUN_1404f2650(); lVar15 = FUN_1406aed80(uVar9);
        if ((uVar20 < (ulonglong)(longlong)*(int *)(lVar15 + 0xdf77c)) &&
           (*(int *)(local_88 + 0xdf78c + lVar15) != 4)) {
          uVar7 = FUN_14080817c(puVar12);
          *(undefined4 *)((longlong)puVar12 + lVar19 + param_1 + 0xbc) = uVar7;
        }
      }
    }
    else {
      if ((puVar12[0x20] & 1 << (*(byte *)(puVar13 + 1) & 0x1f)) == 0) {
        puVar12[*(byte *)(puVar13 + 1)] = *puVar13;
      }
      if (uVar16 == 0) goto LAB_140808026;
      puVar12[0x21] = *(undefined4 *)(local_res8 + 0x1c8 + (longlong)param_5);
    }
    lVar15 = local_68;
    uVar21 = uVar21 + 1; uVar20 = uVar20 + 1;
    local_88 = local_88 + 0xc0; local_78[0] = local_78[0] + 0x80;
    puVar13 = puVar13 + 2; local_res8 = local_res8 + 4; puVar12 = puVar12 + 0x22;
    if (0x37 < (int)uVar21) {
      // ... finalisation round-values 2-equipes (branches param_3+4 & 0x1000000 / & 0x2000000) ...
      return 1;
    }
  } while( true );
}
