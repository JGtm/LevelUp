
undefined8 FUN_142ed4880(undefined8 param_1,longlong param_2,longlong param_3)

{
  longlong lVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  uint uVar4;
  ulonglong *puVar5;
  int iVar6;
  byte bVar7;
  ushort uVar8;
  ulonglong uVar9;
  ulonglong uVar10;
  uint uVar11;
  
  lVar1 = *(longlong *)(param_3 + 0x10);
  iVar6 = 0x40 - *(int *)(param_2 + 0x38);
  uVar10 = 0;
  uVar11 = 0;
  uVar8 = (ushort)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x30);
  if (iVar6 < 0x10) {
    puVar5 = *(ulonglong **)(param_2 + 0x40);
    if (*(ulonglong **)(param_2 + 0x10) < puVar5 + 1) {
      uVar3 = uVar10;
      uVar9 = uVar10;
      uVar4 = uVar11;
      if (puVar5 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar4 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar4;
          uVar2 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar9 = (ulonglong)(byte)uVar2 | uVar9 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_2 + 0x10));
        uVar9 = uVar9 << (-(char)uVar4 & 0x3fU);
      }
    }
    else {
      uVar3 = *puVar5;
      *(ulonglong **)(param_2 + 0x40) = puVar5 + 1;
      uVar9 = uVar3 >> 0x38 | (uVar3 & 0xff000000000000) >> 0x28 | (uVar3 & 0xff0000000000) >> 0x18
              | (uVar3 & 0xff00000000) >> 8 | (uVar3 & 0xff000000) << 8 | (uVar3 & 0xff0000) << 0x18
              | (uVar3 & 0xff00) << 0x28 | uVar3 << 0x38;
      uVar4 = 0x40;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + uVar4;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
    uVar4 = 0x10 - iVar6;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar4 < 0x40) & uVar9 << ((byte)uVar4 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar4;
    uVar8 = (ushort)(uVar9 >> (-(byte)uVar4 & 0x3f)) | uVar8;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0x10;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 0x10;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 0x10;
  }
  *(ushort *)(lVar1 + 0x52c) = uVar8;
  iVar6 = 0x40 - *(int *)(param_2 + 0x38);
  bVar7 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (iVar6 < 8) {
    puVar5 = *(ulonglong **)(param_2 + 0x40);
    if (*(ulonglong **)(param_2 + 0x10) < puVar5 + 1) {
      uVar3 = uVar10;
      uVar9 = uVar10;
      uVar4 = uVar11;
      if (puVar5 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar4 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar4;
          uVar2 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar9 = (ulonglong)(byte)uVar2 | uVar9 << 8;
          *(ulonglong **)(param_2 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_2 + 0x10));
        uVar9 = uVar9 << (-(char)uVar4 & 0x3fU);
      }
    }
    else {
      uVar3 = *puVar5;
      *(ulonglong **)(param_2 + 0x40) = puVar5 + 1;
      uVar9 = uVar3 >> 0x38 | (uVar3 & 0xff000000000000) >> 0x28 | (uVar3 & 0xff0000000000) >> 0x18
              | (uVar3 & 0xff00000000) >> 8 | (uVar3 & 0xff000000) << 8 | (uVar3 & 0xff0000) << 0x18
              | (uVar3 & 0xff00) << 0x28 | uVar3 << 0x38;
      uVar4 = 0x40;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + uVar4;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 8;
    uVar4 = 8 - iVar6;
    *(ulonglong *)(param_2 + 0x30) = -(ulonglong)(uVar4 < 0x40) & uVar9 << ((byte)uVar4 & 0x3f);
    *(uint *)(param_2 + 0x38) = uVar4;
    bVar7 = (byte)(uVar9 >> (-(byte)uVar4 & 0x3f)) | bVar7;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 8;
    *(longlong *)(param_2 + 0x30) = *(longlong *)(param_2 + 0x30) << 8;
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 8;
  }
  *(byte *)(lVar1 + 0x52e) = bVar7;
  iVar6 = *(int *)(param_2 + 0x38);
  bVar7 = (byte)((ulonglong)*(longlong *)(param_2 + 0x30) >> 0x38);
  if (0x40 - iVar6 < 2) {
    puVar5 = *(ulonglong **)(param_2 + 0x40);
    if (*(ulonglong **)(param_2 + 0x10) < puVar5 + 1) {
      uVar3 = uVar10;
      if (puVar5 < *(ulonglong **)(param_2 + 0x10)) {
        do {
          uVar9 = *puVar5;
          uVar11 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar11;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar10 = uVar10 << 8 | (ulonglong)(byte)uVar9;
          *(ulonglong **)(param_2 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_2 + 0x10));
        uVar10 = uVar10 << (-(char)uVar11 & 0x3fU);
      }
    }
    else {
      uVar10 = *puVar5;
      uVar11 = 0x40;
      uVar10 = uVar10 >> 0x38 | (uVar10 & 0xff000000000000) >> 0x28 |
               (uVar10 & 0xff0000000000) >> 0x18 | (uVar10 & 0xff00000000) >> 8 |
               (uVar10 & 0xff000000) << 8 | (uVar10 & 0xff0000) << 0x18 | (uVar10 & 0xff00) << 0x28
               | uVar10 << 0x38;
      *(ulonglong **)(param_2 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + uVar11;
    uVar11 = iVar6 - 0x3e;
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    uVar3 = -(ulonglong)(uVar11 < 0x40) & uVar10 << ((byte)uVar11 & 0x3f);
    bVar7 = (byte)(uVar10 >> (-(byte)uVar11 & 0x3f)) | bVar7 >> 6;
  }
  else {
    *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 2;
    uVar3 = *(longlong *)(param_2 + 0x30) * 4;
    bVar7 = bVar7 >> 6;
    uVar11 = iVar6 + 2;
  }
  *(ulonglong *)(param_2 + 0x30) = uVar3;
  *(uint *)(param_2 + 0x38) = uVar11;
  *(byte *)(lVar1 + 0x52f) = bVar7;
  return CONCAT71((int7)(uVar3 >> 8),1);
}

