
undefined8 FUN_142ed4aec(undefined8 param_1,longlong param_2,longlong param_3)

{
  int iVar1;
  uint *puVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  int iVar5;
  int iVar6;
  uint *puVar7;
  int iVar8;
  byte bVar9;
  ushort uVar10;
  uint uVar11;
  ulonglong uVar12;
  uint uVar13;
  
  puVar2 = *(uint **)(param_3 + 0x10);
  FUN_1424e0e38(param_2,puVar2 + 0x142,0x10);
  FUN_140c5f938(param_2,puVar2 + 0x145,puVar2 + 0x148,0);
  iVar1 = 0x40;
  iVar8 = 0x40 - *(int *)(param_2 + 0x38);
  uVar10 = (ushort)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x30);
  if (iVar8 < 0x10) {
    puVar4 = *(ulonglong **)(param_2 + 0x40);
    uVar12 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar4 + 1) {
      iVar5 = 0;
      if (puVar4 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          iVar6 = iVar5 + 8;
          uVar3 = *puVar4;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar12 = (ulonglong)(byte)uVar3 | uVar12 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar4;
          iVar5 = iVar6;
        } while (puVar4 < *(ulonglong **)(param_2 + 0x10));
        uVar12 = uVar12 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar12 = *puVar4;
      uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
               (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
               (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 | (uVar12 & 0xff00) << 0x28
               | uVar12 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar4 + 1;
      iVar6 = iVar1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar6;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
    uVar11 = 0x10 - iVar8;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar12 << ((byte)uVar11 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar11;
    uVar10 = (ushort)(uVar12 >> (0x40 - (byte)uVar11 & 0x3f)) | uVar10;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 0x10;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 0x10;
  }
  *(ushort *)(puVar2 + 0x14b) = uVar10;
  iVar8 = 0x40 - *(int *)(param_2 + 0x38);
  bVar9 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (iVar8 < 8) {
    puVar4 = *(ulonglong **)(param_2 + 0x40);
    uVar12 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar4 + 1) {
      iVar5 = 0;
      if (puVar4 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          iVar6 = iVar5 + 8;
          uVar3 = *puVar4;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar12 = (ulonglong)(byte)uVar3 | uVar12 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar4;
          iVar5 = iVar6;
        } while (puVar4 < *(ulonglong **)(param_2 + 0x10));
        uVar12 = uVar12 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar12 = *puVar4;
      uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
               (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
               (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 | (uVar12 & 0xff00) << 0x28
               | uVar12 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar4 + 1;
      iVar6 = iVar1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar6;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 8;
    uVar11 = 8 - iVar8;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar12 << ((byte)uVar11 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar11;
    bVar9 = (byte)(uVar12 >> (0x40 - (byte)uVar11 & 0x3f)) | bVar9;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 8;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 8;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 8;
  }
  *(byte *)((longlong)puVar2 + 0x52e) = bVar9;
  iVar8 = 0x40 - *(int *)(param_2 + 0x38);
  bVar9 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (iVar8 < 2) {
    puVar4 = *(ulonglong **)(param_2 + 0x40);
    uVar12 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar4 + 1) {
      iVar5 = 0;
      if (puVar4 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          iVar6 = iVar5 + 8;
          uVar3 = *puVar4;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar12 = (ulonglong)(byte)uVar3 | uVar12 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar4;
          iVar5 = iVar6;
        } while (puVar4 < *(ulonglong **)(param_2 + 0x10));
        uVar12 = uVar12 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar12 = *puVar4;
      uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
               (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
               (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 | (uVar12 & 0xff00) << 0x28
               | uVar12 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar4 + 1;
      iVar6 = iVar1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar6;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    uVar11 = 2 - iVar8;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar11 < 0x40) & uVar12 << ((byte)uVar11 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar11;
    bVar9 = (byte)(uVar12 >> (0x40 - (byte)uVar11 & 0x3f)) | bVar9 >> 6;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) * 4;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 2;
    bVar9 = bVar9 >> 6;
  }
  *(byte *)((longlong)puVar2 + 0x52f) = bVar9;
  iVar8 = 0x40 - *(int *)(param_2 + 0x38);
  uVar11 = (uint)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x20);
  if (iVar8 < 6) {
    puVar4 = *(ulonglong **)(param_2 + 0x40);
    uVar12 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_2 + 0x10) < puVar4 + 1) {
      iVar5 = 0;
      if (puVar4 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          iVar6 = iVar5 + 8;
          uVar3 = *puVar4;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar12 = (ulonglong)(byte)uVar3 | uVar12 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar4;
          iVar5 = iVar6;
        } while (puVar4 < *(ulonglong **)(param_2 + 0x10));
        uVar12 = uVar12 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar12 = *puVar4;
      uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
               (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
               (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 | (uVar12 & 0xff00) << 0x28
               | uVar12 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar4 + 1;
      iVar6 = iVar1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar6;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 6;
    uVar13 = 6 - iVar8;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar13 < 0x40) & uVar12 << ((byte)uVar13 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar13;
    uVar11 = (uint)(uVar12 >> (0x40 - (byte)uVar13 & 0x3f)) | uVar11 >> 0x1a;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 6;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 6;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 6;
    uVar11 = uVar11 >> 0x1a;
  }
  *puVar2 = uVar11;
  for (puVar7 = puVar2 + 2; puVar7 != puVar2 + ((longlong)(int)uVar11 * 5 + 1) * 2;
      puVar7 = puVar7 + 10) {
    FUN_1424d9a30(param_2);
    bVar9 = *(byte *)((longlong)puVar7 + 0x27);
    if ((bVar9 & 1) != 0) {
      FUN_1424e0e38(param_2,puVar7,0x10,0);
      bVar9 = *(byte *)((longlong)puVar7 + 0x27);
    }
    if ((bVar9 & 2) != 0) {
      FUN_140c5f938(param_2,puVar7 + 3,puVar7 + 6,0);
    }
    iVar8 = *(int *)(param_2 + 0x38);
    uVar10 = (ushort)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x30);
    if (0x40 - iVar8 < 0x10) {
      puVar4 = *(ulonglong **)(param_2 + 0x40);
      uVar12 = 0;
      iVar6 = 0;
      if (*(ulonglong **)(param_2 + 0x10) < puVar4 + 1) {
        iVar5 = 0;
        if (puVar4 < *(ulonglong **)(param_2 + 0x10)) {
          do {
            uVar3 = *puVar4;
            iVar6 = iVar5 + 8;
            puVar4 = (ulonglong *)((longlong)puVar4 + 1);
            uVar12 = uVar12 << 8 | (ulonglong)(byte)uVar3;
            *(ulonglong **)(param_2 + 0x40) = puVar4;
            iVar5 = iVar6;
          } while (puVar4 < *(ulonglong **)(param_2 + 0x10));
          uVar12 = uVar12 << (0x40U - (char)iVar6 & 0x3f);
        }
      }
      else {
        uVar12 = *puVar4;
        uVar12 = uVar12 >> 0x38 | (uVar12 & 0xff000000000000) >> 0x28 |
                 (uVar12 & 0xff0000000000) >> 0x18 | (uVar12 & 0xff00000000) >> 8 |
                 (uVar12 & 0xff000000) << 8 | (uVar12 & 0xff0000) << 0x18 |
                 (uVar12 & 0xff00) << 0x28 | uVar12 << 0x38;
        *(ulonglong **)(param_2 + 0x40) = puVar4 + 1;
        iVar6 = iVar1;
      }
      *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + iVar6;
      uVar13 = iVar8 - 0x30;
      *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
      uVar3 = -(ulonglong)(uVar13 < 0x40) & uVar12 << ((byte)uVar13 & 0x3f);
      uVar10 = (ushort)(uVar12 >> (0x40 - (byte)uVar13 & 0x3f)) | uVar10;
    }
    else {
      *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
      uVar3 = *(longlong *)(param_2 + 0x30) << 0x10;
      uVar13 = iVar8 + 0x10;
    }
    *(ulonglong *)(param_2 + 0x30) = uVar3;
    *(uint *)(param_2 + 0x38) = uVar13;
    *(ushort *)(puVar7 + 9) = uVar10;
    FUN_1424ccc74(param_2);
  }
  return 1;
}

