
undefined8 FUN_142f16fd4(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  ulonglong uVar2;
  uint uVar3;
  ulonglong *puVar4;
  int iVar5;
  uint uVar6;
  ulonglong uVar7;
  ulonglong uVar8;
  uint uVar9;
  
  iVar5 = 0x40 - *(int *)(param_4 + 0x38);
  uVar8 = 0;
  uVar9 = 0;
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar5 < 6) {
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
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = 6 - iVar5;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar3 < 0x40) & uVar7 << ((byte)uVar3 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar3;
    uVar6 = (uint)(uVar7 >> (-(byte)uVar3 & 0x3f)) | uVar6 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    uVar6 = uVar6 >> 0x1a;
  }
  *param_3 = uVar6;
  iVar5 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar5 < 3) {
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
    uVar9 = iVar5 - 0x3d;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar2 = -(ulonglong)(uVar9 < 0x40) & uVar8 << ((byte)uVar9 & 0x3f);
    uVar6 = (uint)(uVar8 >> (-(byte)uVar9 & 0x3f)) | uVar6 >> 0x1d;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar2 = *(longlong *)(param_4 + 0x30) * 8;
    uVar6 = uVar6 >> 0x1d;
    uVar9 = iVar5 + 3;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar2;
  *(uint *)(param_4 + 0x38) = uVar9;
  param_3[1] = uVar6;
  return CONCAT71((int7)(uVar2 >> 8),1);
}

