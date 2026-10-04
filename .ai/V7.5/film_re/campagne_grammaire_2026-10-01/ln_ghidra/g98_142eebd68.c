
undefined8 FUN_142eebd68(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  ulonglong uVar2;
  uint uVar3;
  ulonglong *puVar4;
  int iVar5;
  byte bVar6;
  ulonglong uVar7;
  ulonglong uVar8;
  uint uVar9;
  
  iVar5 = 0x40 - *(int *)(param_4 + 0x38);
  uVar8 = 0;
  uVar9 = 0;
  bVar6 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (iVar5 < 8) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    uVar3 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      uVar2 = uVar8;
      uVar7 = uVar8;
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = (int)uVar2 + 8;
          uVar2 = (ulonglong)uVar3;
          uVar1 = *puVar4;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar7 = (ulonglong)(byte)uVar1 | uVar7 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (-(char)uVar3 & 0x3fU);
      }
    }
    else {
      uVar2 = *puVar4;
      uVar3 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
      uVar7 = uVar2 >> 0x38 | (uVar2 & 0xff000000000000) >> 0x28 | (uVar2 & 0xff0000000000) >> 0x18
              | (uVar2 & 0xff00000000) >> 8 | (uVar2 & 0xff000000) << 8 | (uVar2 & 0xff0000) << 0x18
              | (uVar2 & 0xff00) << 0x28 | uVar2 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar3;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    uVar3 = 8 - iVar5;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar3 < 0x40) & uVar7 << ((byte)uVar3 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar3;
    uVar3 = (uint)(uVar7 >> (-(byte)uVar3 & 0x3f)) | (uint)bVar6;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 8;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 8;
    uVar3 = (uint)bVar6;
  }
  *param_3 = uVar3;
  iVar5 = *(int *)(param_4 + 0x38);
  bVar6 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (0x40 - iVar5 < 1) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      uVar2 = uVar8;
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar7 = *puVar4;
          uVar9 = (int)uVar2 + 8;
          uVar2 = (ulonglong)uVar9;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar8 = uVar8 << 8 | (ulonglong)(byte)uVar7;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (-(char)uVar9 & 0x3fU);
      }
    }
    else {
      uVar8 = *puVar4;
      uVar9 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar9;
    uVar9 = iVar5 - 0x3f;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar2 = -(ulonglong)(uVar9 < 0x40) & uVar8 << ((byte)uVar9 & 0x3f);
    bVar6 = (byte)(uVar8 >> (-(byte)uVar9 & 0x3f)) | bVar6 >> 7;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar2 = *(longlong *)(param_4 + 0x30) * 2;
    bVar6 = bVar6 >> 7;
    uVar9 = iVar5 + 1;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar2;
  *(uint *)(param_4 + 0x38) = uVar9;
  *(byte *)(param_3 + 1) = bVar6;
  return CONCAT71((int7)(uVar2 >> 8),1);
}

