
undefined8 FUN_142eebb3c(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  ulonglong uVar1;
  undefined1 uVar2;
  ulonglong uVar3;
  uint uVar4;
  ulonglong *puVar5;
  int iVar6;
  uint uVar7;
  ulonglong uVar8;
  ulonglong uVar9;
  uint uVar10;
  
  uVar9 = 0;
  uVar10 = 0;
  iVar6 = 0x40 - *(int *)(param_4 + 0x38);
  uVar7 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (iVar6 < 6) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar3 = uVar9;
      uVar8 = uVar9;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar4 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar4;
          uVar1 = *puVar5;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar8 = (ulonglong)(byte)uVar1 | uVar8 << 8;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (-(char)uVar4 & 0x3fU);
      }
    }
    else {
      uVar3 = *puVar5;
      uVar4 = 0x40;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
      uVar8 = uVar3 >> 0x38 | (uVar3 & 0xff000000000000) >> 0x28 | (uVar3 & 0xff0000000000) >> 0x18
              | (uVar3 & 0xff00000000) >> 8 | (uVar3 & 0xff000000) << 8 | (uVar3 & 0xff0000) << 0x18
              | (uVar3 & 0xff00) << 0x28 | uVar3 << 0x38;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar4;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar4 = 6 - iVar6;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar4 < 0x40) & uVar8 << ((byte)uVar4 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar4;
    uVar7 = (uint)(uVar8 >> (-(byte)uVar4 & 0x3f)) | uVar7 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 6;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 6;
    uVar7 = uVar7 >> 0x1a;
  }
  *param_3 = uVar7;
  iVar6 = *(int *)(param_4 + 0x38);
  uVar7 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar6 < 6) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar3 = uVar9;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar8 = *puVar5;
          uVar10 = (int)uVar3 + 8;
          uVar3 = (ulonglong)uVar10;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar9 = uVar9 << 8 | (ulonglong)(byte)uVar8;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar9 << (-(char)uVar10 & 0x3fU);
      }
    }
    else {
      uVar9 = *puVar5;
      uVar10 = 0x40;
      uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 | (uVar9 & 0xff0000000000) >> 0x18
              | (uVar9 & 0xff00000000) >> 8 | (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18
              | (uVar9 & 0xff00) << 0x28 | uVar9 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar10;
    uVar10 = iVar6 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = -(ulonglong)(uVar10 < 0x40) & uVar9 << ((byte)uVar10 & 0x3f);
    uVar7 = (uint)(uVar9 >> (-(byte)uVar10 & 0x3f)) | uVar7 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar10 = iVar6 + 6;
    uVar3 = *(longlong *)(param_4 + 0x30) << 6;
    uVar7 = uVar7 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar10;
  param_3[1] = uVar7;
  uVar2 = FUN_1406cf008(param_4);
  *(undefined1 *)(param_3 + 2) = uVar2;
  uVar2 = FUN_1406cf008(param_4);
  *(undefined1 *)((longlong)param_3 + 9) = uVar2;
  return 1;
}

