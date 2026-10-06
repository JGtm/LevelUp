byte FUN_1407ee138(longlong param_1,uint *param_2,int param_3)
{
  uint *puVar1;
  char cVar2;
  ulonglong uVar3;
  longlong lVar4;
  ulonglong *puVar5;
  int iVar6;
  int iVar7;
  uint *puVar8;
  int iVar9;
  byte bVar10;
  ushort uVar11;
  uint uVar12;
  ulonglong uVar13;
  int iVar14;
  uint uVar15;
  bool bVar16;
  bool bVar17;
  undefined1 local_218 [256];
  undefined1 local_118 [256];
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 3) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    uVar15 = 3 - iVar14;
    puVar8 = (uint *)(uVar13 << ((byte)uVar15 & 0x3f));
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & (ulonglong)puVar8;
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12 >> 0x1d;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 8;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 3;
    uVar12 = uVar12 >> 0x1d;
    puVar8 = param_2;
  }
  *param_2 = uVar12;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 3) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    uVar15 = 3 - iVar14;
    puVar8 = (uint *)(uVar13 << ((byte)uVar15 & 0x3f));
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & (ulonglong)puVar8;
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12 >> 0x1d;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 8;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 3;
    uVar12 = uVar12 >> 0x1d;
  }
  param_2[1] = uVar12;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 2) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;
    uVar15 = 2 - iVar14;
    puVar8 = (uint *)(uVar13 << ((byte)uVar15 & 0x3f));
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & (ulonglong)puVar8;
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12 >> 0x1e;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 4;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 2;
    uVar12 = uVar12 >> 0x1e;
  }
  param_2[2] = uVar12;
  iVar14 = 0x40;
  iVar9 = 0x40 - *(int *)(param_1 + 0x38);
  uVar11 = (ushort)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x30);
  if (iVar9 < 7) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar6 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar7 = iVar7 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar7 & 0x3fU);
        iVar6 = iVar7;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar6 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar6;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 7;
    uVar12 = 7 - iVar9;
    puVar8 = (uint *)(uVar13 << ((byte)uVar12 & 0x3f));
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar12 < 0x40) & (ulonglong)puVar8;
    *(uint *)(param_1 + 0x38) = uVar12;
    uVar11 = (ushort)(uVar13 >> (-(byte)uVar12 & 0x3f)) | uVar11 >> 9;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 7;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 7;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 7;
    uVar11 = uVar11 >> 9;
  }
  *(ushort *)(param_2 + 3) = uVar11;
  FUN_1406d676c(param_1,puVar8,param_2 + 4,0x40);
  iVar9 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar9 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar6 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar7 = iVar6 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
          iVar6 = iVar7;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar7 & 0x3fU);
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = iVar14;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar9;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[6] = uVar12;
  iVar9 = 0x40 - *(int *)(param_1 + 0x38);
  bVar10 = (byte)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x38);
  if (iVar9 < 3) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar6 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar7 = iVar6 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
          iVar6 = iVar7;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar7 & 0x3fU);
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = iVar14;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    uVar12 = 3 - iVar9;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar12 < 0x40) & uVar13 << ((byte)uVar12 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar12;
    bVar10 = (byte)(uVar13 >> (-(byte)uVar12 & 0x3f)) | bVar10 >> 5;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 8;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 3;
    bVar10 = bVar10 >> 5;
  }
  *(byte *)(param_2 + 8) = bVar10;
  iVar9 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar9 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar7 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar14 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar7 = iVar14 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
          iVar14 = iVar7;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar7 & 0x3fU);
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = iVar14;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar9;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[0x3a544] = uVar12;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[0x3a545] = uVar12;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[0x3a546] = uVar12;
  FUN_140ee5f40(param_1);
  FUN_1407cbc24(param_1);
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[0x3a9c5] = uVar12;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  bVar10 = (byte)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar12 = 0x20 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar12 < 0x40) & uVar13 << ((byte)uVar12 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar12;
    bVar10 = (byte)(uVar13 >> (-(byte)uVar12 & 0x3f)) | bVar10;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  *(byte *)(param_2 + 0x3a9c6) = bVar10;
  cVar2 = FUN_140ee5ef8(param_1,param_2 + 0x3a9cb);
  bVar16 = cVar2 != '\0';
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    bVar17 = SUB81((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x3f,0);
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1,1);
    bVar17 = lVar4 != 0;
  }
  *(bool *)((longlong)param_2 + 0xea719) = bVar17;
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    bVar17 = SUB81((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x3f,0);
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1);
    bVar17 = lVar4 != 0;
  }
  *(bool *)((longlong)param_2 + 0xea71a) = bVar17;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 2) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;
    uVar15 = 2 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12 >> 0x1e;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 4;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 2;
    uVar12 = uVar12 >> 0x1e;
  }
  param_2[0x3a9c7] = uVar12;
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    bVar17 = SUB81((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x3f,0);
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1);
    bVar17 = lVar4 != 0;
  }
  *(bool *)(param_2 + 0x3a9c8) = bVar17;
  iVar14 = 0x40 - *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          iVar9 = iVar9 + 8;
          uVar3 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = (ulonglong)(byte)uVar3 | uVar13 << 8;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar15 = 0x20 - iVar14;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    *(uint *)(param_1 + 0x38) = uVar15;
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) << 0x20;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 0x20;
  }
  param_2[0x3a9c9] = uVar12;
  iVar14 = *(int *)(param_1 + 0x38);
  uVar12 = (uint)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x20);
  if (0x40 - iVar14 < 0x20) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    uVar13 = 0;
    iVar9 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      iVar7 = 0;
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          uVar3 = *puVar5;
          iVar9 = iVar9 + 8;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar13 = uVar13 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_1 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_1 + 0x10));
        uVar13 = uVar13 << (-(char)iVar9 & 0x3fU);
        iVar7 = iVar9;
      }
    }
    else {
      uVar13 = *puVar5;
      uVar13 = uVar13 >> 0x38 | (uVar13 & 0xff000000000000) >> 0x28 |
               (uVar13 & 0xff0000000000) >> 0x18 | (uVar13 & 0xff00000000) >> 8 |
               (uVar13 & 0xff000000) << 8 | (uVar13 & 0xff0000) << 0x18 | (uVar13 & 0xff00) << 0x28
               | uVar13 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar5 + 1;
      iVar7 = 0x40;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar7;
    uVar15 = iVar14 - 0x20;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar3 = -(ulonglong)(uVar15 < 0x40) & uVar13 << ((byte)uVar15 & 0x3f);
    uVar12 = (uint)(uVar13 >> (-(byte)uVar15 & 0x3f)) | uVar12;
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
    uVar3 = *(longlong *)(param_1 + 0x30) << 0x20;
    uVar15 = iVar14 + 0x20;
  }
  *(ulonglong *)(param_1 + 0x30) = uVar3;
  *(uint *)(param_1 + 0x38) = uVar15;
  param_2[0x3a9ca] = uVar12;
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    bVar17 = SUB81((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x3f,0);
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1,1);
    bVar17 = lVar4 != 0;
  }
  *(bool *)(param_2 + 0x3aa03) = bVar17;
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    lVar4 = *(longlong *)(param_1 + 0x30);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(longlong *)(param_1 + 0x30) = lVar4 * 2;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
    if (-1 < lVar4) {
LAB_1407ee6ae:
      FUN_140959bbc(param_2 + 10,0);
      goto LAB_1407ee5bc;
    }
  }
  else {
    lVar4 = FUN_1406d6c7c(param_1,1);
    if (lVar4 == 0) goto LAB_1407ee6ae;
  }
  if ((bVar16) && (cVar2 = FUN_140b3a118(param_2 + 10,param_1), cVar2 != '\0')) {
    lVar4 = FUN_1406aed80(param_2 + 10);
    iVar14 = FUN_14051a4b8(*(undefined4 *)(lVar4 + 4));
    if (iVar14 != 0) {
      bVar16 = true;
      goto LAB_1407ee5bc;
    }
  }
  bVar16 = false;
LAB_1407ee5bc:
  FUN_1407cbc24(param_1);
  FUN_14051c0f4(param_2 + 0x3aa04,local_218,0x100);
  FUN_1407cbc24(param_1);
  FUN_14051c0f4(param_2 + 0x3aa44,local_118,0x100);
  FUN_1406d676c(param_1);
  FUN_1406d676c(param_1);
  puVar8 = param_2 + 0x3aabc;
  puVar1 = param_2 + 0x44d3c;
  if (puVar8 != puVar1) {
    if (param_3 < 0xf) {
      do {
        FUN_1406d676c(param_1);
        FUN_142be052c(puVar8 + 6,param_1);
        FUN_1406d676c(param_1);
        puVar8 = puVar8 + 0x514;
      } while (puVar8 != puVar1);
    }
    else {
      do {
        FUN_1407ee98c(puVar8,param_1);
        puVar8 = puVar8 + 0x514;
      } while (puVar8 != puVar1);
    }
  }
  cVar2 = FUN_140ad4144(param_2);
  return -(cVar2 != '\0') & bVar16;
}
