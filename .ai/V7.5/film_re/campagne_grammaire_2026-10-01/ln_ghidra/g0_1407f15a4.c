
undefined8 FUN_1407f15a4(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  ulonglong uVar1;
  char cVar2;
  undefined2 uVar3;
  int iVar4;
  undefined4 uVar5;
  undefined4 *puVar6;
  undefined8 uVar7;
  longlong lVar8;
  uint uVar9;
  uint uVar10;
  ulonglong *puVar11;
  int iVar12;
  int *piVar13;
  int iVar14;
  byte bVar15;
  ulonglong uVar16;
  ulonglong uVar17;
  ushort uVar18;
  undefined4 uVar19;
  undefined4 extraout_XMM0_Da;
  undefined4 extraout_XMM0_Da_00;
  undefined1 local_res18 [16];
  
  uVar5 = 0xffffffff;
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  puVar6 = (undefined4 *)FUN_1407f21b4(local_res18);
  *(undefined4 *)(param_3 + 4) = *puVar6;
  uVar3 = FUN_1407f2058(param_4);
  *(undefined4 *)(param_3 + 8) = 0;
  *(undefined2 *)(param_3 + 0x10) = uVar3;
  FUN_14076dc04(param_4);
  cVar2 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 0x20) = cVar2;
  if (cVar2 != '\0') {
    FUN_14076dc04(param_4);
    uVar19 = FUN_1406d84b4(param_4);
    *(undefined4 *)(param_3 + 0x30) = uVar19;
  }
  uVar19 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x34) = uVar19;
  uVar19 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x38) = uVar19;
  uVar19 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x3c) = uVar19;
  cVar2 = FUN_1406cf008(param_4);
  uVar19 = DAT_143cd8374;
  if (cVar2 != '\0') {
    uVar19 = FUN_1406d84b4(param_4);
  }
  *(undefined4 *)(param_3 + 0x40) = uVar19;
  *(undefined4 *)(param_3 + 0x44) = 0;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffffffd;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 2;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffffffb;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 4;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffffff7;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 8;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffffffef;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x10;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffffffdf;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x20;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffffffbf;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x40;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffffdff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x200;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffffff7f;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x80;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffffeff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x100;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xfffff7ff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x800;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffefffff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x100000;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xffdfffff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x200000;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xff7fffff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x800000;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xbfffffff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x40000000;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uVar9 = *(uint *)(param_3 + 0x44) & 0xefffffff;
  }
  else {
    uVar9 = *(uint *)(param_3 + 0x44) | 0x10000000;
  }
  *(uint *)(param_3 + 0x44) = uVar9;
  if (cVar2 != '\0') {
    iVar14 = 0x40 - *(int *)(param_4 + 0x38);
    uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
    if (iVar14 < 0x20) {
      puVar11 = *(ulonglong **)(param_4 + 0x40);
      uVar17 = 0;
      iVar4 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar11 + 1) {
        if (puVar11 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            iVar4 = iVar4 + 8;
            uVar16 = *puVar11;
            puVar11 = (ulonglong *)((longlong)puVar11 + 1);
            uVar17 = (ulonglong)(byte)uVar16 | uVar17 << 8;
            *(ulonglong **)(param_4 + 0x40) = puVar11;
          } while (puVar11 < *(ulonglong **)(param_4 + 0x10));
          uVar17 = uVar17 << (-(char)iVar4 & 0x3fU);
        }
      }
      else {
        uVar17 = *puVar11;
        iVar4 = 0x40;
        uVar17 = uVar17 >> 0x38 | (uVar17 & 0xff000000000000) >> 0x28 |
                 (uVar17 & 0xff0000000000) >> 0x18 | (uVar17 & 0xff00000000) >> 8 |
                 (uVar17 & 0xff000000) << 8 | (uVar17 & 0xff0000) << 0x18 |
                 (uVar17 & 0xff00) << 0x28 | uVar17 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar11 + 1;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar4;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar10 = 0x20 - iVar14;
      *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar10 < 0x40) & uVar17 << ((byte)uVar10 & 0x3f)
      ;
      *(uint *)(param_4 + 0x38) = uVar10;
      uVar9 = (uint)(uVar17 >> (-(byte)uVar10 & 0x3f)) | uVar9;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
    }
    *(uint *)(param_3 + 0xc) = uVar9;
  }
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    lVar8 = *(longlong *)(param_4 + 0x30);
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    *(longlong *)(param_4 + 0x30) = lVar8 * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
    if (lVar8 < 0) goto LAB_1407f1893;
LAB_1407f1d3c:
    *(uint *)(param_3 + 0x44) = *(uint *)(param_3 + 0x44) & 0xfff7ffff;
  }
  else {
    lVar8 = FUN_1406d6c7c(param_4);
    if (lVar8 == 0) goto LAB_1407f1d3c;
LAB_1407f1893:
    *(uint *)(param_3 + 0x44) = *(uint *)(param_3 + 0x44) | 0x80000;
  }
  piVar13 = (int *)(param_4 + 0x2c);
  iVar14 = 0x40 - *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar14 < 3) {
    puVar11 = *(ulonglong **)(param_4 + 0x40);
    uVar17 = 0;
    iVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar11 + 1) {
      if (puVar11 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar4 = iVar4 + 8;
          uVar16 = *puVar11;
          puVar11 = (ulonglong *)((longlong)puVar11 + 1);
          uVar17 = (ulonglong)(byte)uVar16 | uVar17 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar11;
        } while (puVar11 < *(ulonglong **)(param_4 + 0x10));
        uVar17 = uVar17 << (-(char)iVar4 & 0x3fU);
      }
    }
    else {
      uVar17 = *puVar11;
      iVar4 = 0x40;
      uVar17 = uVar17 >> 0x38 | (uVar17 & 0xff000000000000) >> 0x28 |
               (uVar17 & 0xff0000000000) >> 0x18 | (uVar17 & 0xff00000000) >> 8 |
               (uVar17 & 0xff000000) << 8 | (uVar17 & 0xff0000) << 0x18 | (uVar17 & 0xff00) << 0x28
               | uVar17 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar11 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar4;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar10 = 3 - iVar14;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar10 < 0x40) & uVar17 << ((byte)uVar10 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar10;
    uVar9 = (uint)(uVar17 >> (-(byte)uVar10 & 0x3f)) | uVar9 >> 0x1d;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 8;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 3;
    uVar9 = uVar9 >> 0x1d;
  }
  uVar10 = 4;
  if (uVar9 < 4) {
    uVar10 = uVar9;
  }
  *(short *)(param_3 + 0x5c) = (short)uVar10;
  uVar19 = FUN_1424cd17c(param_4);
  *(undefined4 *)(param_3 + 0x4c) = uVar19;
  uVar19 = FUN_1424cd150(param_4);
  *(undefined4 *)(param_3 + 0x48) = uVar19;
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 != '\0') {
    *(float *)(param_3 + 0x4c) = *(float *)(param_3 + 0x4c) * DAT_143cd84ec;
    *(float *)(param_3 + 0x48) = *(float *)(param_3 + 0x48) * DAT_143cd84ec;
  }
  uVar3 = FUN_1407f1f24(param_4);
  *(undefined2 *)(param_3 + 0x52) = uVar3;
  uVar3 = FUN_1407f1e4c(param_4);
  *(undefined2 *)(param_3 + 0x54) = uVar3;
  iVar14 = 0x40 - *(int *)(param_4 + 0x38);
  uVar9 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar14 < 4) {
    puVar11 = *(ulonglong **)(param_4 + 0x40);
    uVar17 = 0;
    iVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar11 + 1) {
      iVar12 = 0;
      if (puVar11 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar4 = iVar12 + 8;
          uVar16 = *puVar11;
          puVar11 = (ulonglong *)((longlong)puVar11 + 1);
          uVar17 = (ulonglong)(byte)uVar16 | uVar17 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar11;
          iVar12 = iVar4;
        } while (puVar11 < *(ulonglong **)(param_4 + 0x10));
        uVar17 = uVar17 << (-(char)iVar4 & 0x3fU);
      }
    }
    else {
      uVar17 = *puVar11;
      iVar4 = 0x40;
      uVar17 = uVar17 >> 0x38 | (uVar17 & 0xff000000000000) >> 0x28 |
               (uVar17 & 0xff0000000000) >> 0x18 | (uVar17 & 0xff00000000) >> 8 |
               (uVar17 & 0xff000000) << 8 | (uVar17 & 0xff0000) << 0x18 | (uVar17 & 0xff00) << 0x28
               | uVar17 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar11 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar4;
    *piVar13 = *piVar13 + 4;
    uVar10 = 4 - iVar14;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar10 < 0x40) & uVar17 << ((byte)uVar10 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar10;
    uVar9 = (uint)(uVar17 >> (-(byte)uVar10 & 0x3f)) | uVar9 >> 0x1c;
  }
  else {
    *piVar13 = *piVar13 + 4;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
    uVar9 = uVar9 >> 0x1c;
  }
  *(uint *)(param_3 + 0x58) = uVar9;
  uVar9 = *(uint *)(param_4 + 0x38);
  if (uVar9 < 0x40) {
    lVar8 = *(longlong *)(param_4 + 0x30);
    *piVar13 = *piVar13 + 1;
    *(longlong *)(param_4 + 0x30) = lVar8 * 2;
    *(uint *)(param_4 + 0x38) = uVar9 + 1;
    if (-1 < lVar8) goto LAB_1407f1984;
    FUN_141015740(param_4,uVar9,param_3 + 0x60);
  }
  else {
    lVar8 = FUN_1406d6c7c(param_4);
    if (lVar8 != 0) {
      uVar7 = FUN_14232613c();
      return uVar7;
    }
LAB_1407f1984:
    *(undefined4 *)(param_3 + 0x60) = 0xffffffff;
  }
  if (*(int *)(param_3 + 0x58) == 1) {
    iVar14 = 0x40 - *(int *)(param_4 + 0x38);
    bVar15 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
    if (iVar14 < 8) {
      puVar11 = *(ulonglong **)(param_4 + 0x40);
      uVar17 = 0;
      iVar4 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar11 + 1) {
        iVar12 = 0;
        if (puVar11 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            iVar4 = iVar12 + 8;
            uVar16 = *puVar11;
            puVar11 = (ulonglong *)((longlong)puVar11 + 1);
            uVar17 = (ulonglong)(byte)uVar16 | uVar17 << 8;
            *(ulonglong **)(param_4 + 0x40) = puVar11;
            iVar12 = iVar4;
          } while (puVar11 < *(ulonglong **)(param_4 + 0x10));
          uVar17 = uVar17 << (-(char)iVar4 & 0x3fU);
        }
      }
      else {
        uVar17 = *puVar11;
        iVar4 = 0x40;
        uVar17 = uVar17 >> 0x38 | (uVar17 & 0xff000000000000) >> 0x28 |
                 (uVar17 & 0xff0000000000) >> 0x18 | (uVar17 & 0xff00000000) >> 8 |
                 (uVar17 & 0xff000000) << 8 | (uVar17 & 0xff0000) << 0x18 |
                 (uVar17 & 0xff00) << 0x28 | uVar17 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar11 + 1;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar4;
      *piVar13 = *piVar13 + 8;
      uVar9 = 8 - iVar14;
      *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar9 < 0x40) & uVar17 << ((byte)uVar9 & 0x3f);
      *(uint *)(param_4 + 0x38) = uVar9;
      bVar15 = (byte)(uVar17 >> (-(byte)uVar9 & 0x3f)) | bVar15;
    }
    else {
      *piVar13 = *piVar13 + 8;
      *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 8;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 8;
    }
  }
  else {
    bVar15 = 0;
  }
  *(byte *)(param_3 + 100) = bVar15;
  iVar4 = FUN_1406d310c(10);
  iVar14 = *(int *)(param_4 + 0x38);
  uVar17 = *(ulonglong *)(param_4 + 0x30);
  bVar15 = (byte)iVar4;
  if (0x40 - iVar14 < iVar4) {
    puVar11 = *(ulonglong **)(param_4 + 0x40);
    uVar16 = 0;
    iVar12 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar11 + 1) {
      if (puVar11 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar1 = *puVar11;
          iVar12 = iVar12 + 8;
          puVar11 = (ulonglong *)((longlong)puVar11 + 1);
          uVar16 = uVar16 << 8 | (ulonglong)(byte)uVar1;
          *(ulonglong **)(param_4 + 0x40) = puVar11;
        } while (puVar11 < *(ulonglong **)(param_4 + 0x10));
        uVar16 = uVar16 << (-(char)iVar12 & 0x3fU);
      }
    }
    else {
      uVar16 = *puVar11;
      iVar12 = 0x40;
      uVar16 = uVar16 >> 0x38 | (uVar16 & 0xff000000000000) >> 0x28 |
               (uVar16 & 0xff0000000000) >> 0x18 | (uVar16 & 0xff00000000) >> 8 |
               (uVar16 & 0xff000000) << 8 | (uVar16 & 0xff0000) << 0x18 | (uVar16 & 0xff00) << 0x28
               | uVar16 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar11 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar12;
    *piVar13 = *piVar13 + iVar4;
    uVar9 = iVar14 + -0x40 + iVar4;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar9 < 0x40) & uVar16 << ((byte)uVar9 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar9;
    uVar18 = (ushort)(uVar17 >> (-bVar15 & 0x3f)) | (ushort)(uVar16 >> (-(byte)uVar9 & 0x3f));
  }
  else {
    *piVar13 = *piVar13 + iVar4;
    *(int *)(param_4 + 0x38) = iVar14 + iVar4;
    *(ulonglong *)(param_4 + 0x30) = uVar17 << (bVar15 & 0x3f);
    uVar18 = (ushort)(uVar17 >> (-bVar15 & 0x3f));
  }
  *(ushort *)(param_3 + 0x70) = uVar18;
  if (*(uint *)(param_4 + 0x38) < 0x40) {
    lVar8 = *(longlong *)(param_4 + 0x30);
    *piVar13 = *piVar13 + 1;
    *(longlong *)(param_4 + 0x30) = lVar8 * 2;
    *(uint *)(param_4 + 0x38) = *(uint *)(param_4 + 0x38) + 1;
    uVar19 = extraout_XMM0_Da;
    if (lVar8 < 0) {
LAB_1407f1aa9:
      FUN_1406d3140(uVar19,param_4,0,param_3 + 0x68);
      uVar5 = FUN_140809d20(*(undefined4 *)(param_3 + 0x68),0);
      goto LAB_1407f1a39;
    }
  }
  else {
    lVar8 = FUN_1406d6c7c(param_4,1);
    uVar19 = extraout_XMM0_Da_00;
    if (lVar8 != 0) goto LAB_1407f1aa9;
  }
  *(undefined4 *)(param_3 + 0x68) = 0xffffffff;
LAB_1407f1a39:
  *(undefined4 *)(param_3 + 0x6c) = uVar5;
  return 1;
}

