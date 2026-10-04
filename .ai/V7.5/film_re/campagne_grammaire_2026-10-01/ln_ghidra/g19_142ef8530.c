
undefined8 FUN_142ef8530(undefined8 param_1,undefined8 param_2,byte *param_3,longlong param_4)

{
  ulonglong uVar1;
  ulonglong uVar2;
  ulonglong *puVar3;
  int iVar4;
  byte bVar5;
  uint uVar6;
  ulonglong uVar7;
  ulonglong uVar8;
  uint uVar9;
  
  iVar4 = 0x40 - *(int *)(param_4 + 0x38);
  uVar8 = 0;
  uVar9 = 0;
  bVar5 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (iVar4 < 2) {
    puVar3 = *(ulonglong **)(param_4 + 0x40);
    uVar6 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar3 + 1) {
      uVar2 = uVar8;
      uVar7 = uVar8;
      if (puVar3 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar6 = (int)uVar2 + 8;
          uVar2 = (ulonglong)uVar6;
          uVar1 = *puVar3;
          puVar3 = (ulonglong *)((longlong)puVar3 + 1);
          uVar7 = (ulonglong)(byte)uVar1 | uVar7 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar3;
        } while (puVar3 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (-(char)uVar6 & 0x3fU);
      }
    }
    else {
      uVar2 = *puVar3;
      uVar6 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar3 + 1;
      uVar7 = uVar2 >> 0x38 | (uVar2 & 0xff000000000000) >> 0x28 | (uVar2 & 0xff0000000000) >> 0x18
              | (uVar2 & 0xff00000000) >> 8 | (uVar2 & 0xff000000) << 8 | (uVar2 & 0xff0000) << 0x18
              | (uVar2 & 0xff00) << 0x28 | uVar2 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar6;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar6 = 2 - iVar4;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar6 < 0x40) & uVar7 << ((byte)uVar6 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar6;
    bVar5 = (byte)(uVar7 >> (-(byte)uVar6 & 0x3f)) | bVar5 >> 6;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) * 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 2;
    bVar5 = bVar5 >> 6;
  }
  *param_3 = bVar5;
  iVar4 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar4 < 7) {
    puVar3 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar3 + 1) {
      uVar2 = uVar8;
      if (puVar3 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar7 = *puVar3;
          uVar9 = (int)uVar2 + 8;
          uVar2 = (ulonglong)uVar9;
          puVar3 = (ulonglong *)((longlong)puVar3 + 1);
          uVar8 = uVar8 << 8 | (ulonglong)(byte)uVar7;
          *(ulonglong **)(param_4 + 0x40) = puVar3;
        } while (puVar3 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (-(char)uVar9 & 0x3fU);
      }
    }
    else {
      uVar8 = *puVar3;
      uVar9 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar3 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar9;
    uVar9 = iVar4 - 0x39;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 7;
    uVar2 = -(ulonglong)(uVar9 < 0x40) & uVar8 << ((byte)uVar9 & 0x3f);
    uVar6 = (uint)(uVar8 >> (-(byte)uVar9 & 0x3f)) | uVar6 >> 0x19;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 7;
    uVar9 = iVar4 + 7;
    uVar2 = *(longlong *)(param_4 + 0x30) << 7;
    uVar6 = uVar6 >> 0x19;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar2;
  *(uint *)(param_4 + 0x38) = uVar9;
  *(uint *)(param_3 + 4) = uVar6;
  return CONCAT71((int7)(uVar2 >> 8),1);
}

