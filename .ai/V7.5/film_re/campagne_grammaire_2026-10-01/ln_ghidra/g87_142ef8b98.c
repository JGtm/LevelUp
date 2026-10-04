
undefined4 FUN_142ef8b98(undefined8 param_1,undefined8 param_2,byte *param_3,longlong param_4)

{
  ulonglong uVar1;
  ulonglong *puVar2;
  int iVar3;
  int iVar4;
  int iVar5;
  uint uVar6;
  byte bVar7;
  ulonglong uVar8;
  undefined1 uVar9;
  
  iVar5 = 0x40 - *(int *)(param_4 + 0x38);
  uVar9 = 1;
  bVar7 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (iVar5 < 2) {
    puVar2 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    iVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar2 + 1) {
      iVar3 = 0;
      if (puVar2 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          iVar4 = iVar4 + 8;
          uVar1 = *puVar2;
          puVar2 = (ulonglong *)((longlong)puVar2 + 1);
          uVar8 = (ulonglong)(byte)uVar1 | uVar8 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar2;
        } while (puVar2 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (-(char)iVar4 & 0x3fU);
        iVar3 = iVar4;
      }
    }
    else {
      uVar8 = *puVar2;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar2 + 1;
      iVar3 = 0x40;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar3;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar6 = 2 - iVar5;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar6 < 0x40) & uVar8 << ((byte)uVar6 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar6;
    bVar7 = (byte)(uVar8 >> (-(byte)uVar6 & 0x3f)) | bVar7 >> 6;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 2;
    bVar7 = bVar7 >> 6;
  }
  *param_3 = bVar7;
  iVar5 = *(int *)(param_4 + 0x38);
  bVar7 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (0x40 - iVar5 < 2) {
    puVar2 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    iVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar2 + 1) {
      iVar3 = 0;
      if (puVar2 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar1 = *puVar2;
          iVar4 = iVar4 + 8;
          puVar2 = (ulonglong *)((longlong)puVar2 + 1);
          uVar8 = uVar8 << 8 | (ulonglong)(byte)uVar1;
          *(ulonglong **)(param_4 + 0x40) = puVar2;
        } while (puVar2 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (-(char)iVar4 & 0x3fU);
        iVar3 = iVar4;
      }
    }
    else {
      uVar8 = *puVar2;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar2 + 1;
      iVar3 = 0x40;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar3;
    uVar6 = iVar5 - 0x3e;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar1 = -(ulonglong)(uVar6 < 0x40) & uVar8 << ((byte)uVar6 & 0x3f);
    bVar7 = (byte)(uVar8 >> (-(byte)uVar6 & 0x3f)) | bVar7 >> 6;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar1 = *(longlong *)(param_4 + 0x30) * 4;
    uVar6 = iVar5 + 2;
    bVar7 = bVar7 >> 6;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar1;
  *(uint *)(param_4 + 0x38) = uVar6;
  param_3[1] = bVar7;
  iVar5 = *(int *)(param_4 + 0x18) * 8;
  if ((*(char *)(param_4 + 0x24) != '\0') || (iVar5 < *(int *)(param_4 + 0x2c))) {
    uVar9 = 0;
  }
  return CONCAT31((int3)((uint)iVar5 >> 8),uVar9);
}

