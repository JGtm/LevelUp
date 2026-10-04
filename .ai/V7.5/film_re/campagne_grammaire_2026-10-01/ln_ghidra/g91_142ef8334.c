
undefined8
FUN_142ef8334(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4,char param_5)

{
  ulonglong uVar1;
  uint uVar2;
  int iVar3;
  uint *puVar4;
  ulonglong uVar5;
  uint uVar6;
  ulonglong *puVar7;
  ulonglong uVar8;
  ulonglong uVar9;
  uint uVar10;
  
  uVar9 = 0;
  uVar10 = 0;
  if (param_5 == '\0') {
    iVar3 = 0x40 - *(int *)(param_4 + 0x38);
    uVar2 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
    if (iVar3 < 0x20) {
      puVar7 = *(ulonglong **)(param_4 + 0x40);
      uVar6 = 0;
      if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
        uVar8 = uVar9;
        uVar5 = uVar9;
        if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar6 = (int)uVar8 + 8;
            uVar8 = (ulonglong)uVar6;
            uVar1 = *puVar7;
            puVar7 = (ulonglong *)((longlong)puVar7 + 1);
            uVar5 = (ulonglong)(byte)uVar1 | uVar5 << 8;
            *(ulonglong **)(param_4 + 0x40) = puVar7;
          } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
          uVar5 = uVar5 << (-(char)uVar6 & 0x3fU);
        }
      }
      else {
        uVar8 = *puVar7;
        uVar6 = 0x40;
        *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
        uVar5 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 |
                (uVar8 & 0xff0000000000) >> 0x18 | (uVar8 & 0xff00000000) >> 8 |
                (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18 | (uVar8 & 0xff00) << 0x28 |
                uVar8 << 0x38;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar6;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar6 = 0x20 - iVar3;
      *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar6 < 0x40) & uVar5 << ((byte)uVar6 & 0x3f);
      *(uint *)(param_4 + 0x38) = uVar6;
      uVar2 = (uint)(uVar5 >> (-(byte)uVar6 & 0x3f)) | uVar2;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
      *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
    }
    *param_3 = uVar2;
  }
  else {
    FUN_1406d3140(param_1,param_4,6,param_3);
  }
  uVar2 = FUN_141102ed0(0x5b);
  if (uVar2 < 3) {
    iVar3 = FUN_141102ed0(0x5b);
    if (iVar3 != 2) {
      return 1;
    }
    iVar3 = *(int *)(param_4 + 0x38);
    uVar8 = *(ulonglong *)(param_4 + 0x30);
    if (0x40 - iVar3 < 0x20) {
      puVar7 = *(ulonglong **)(param_4 + 0x40);
      if (*(ulonglong **)(param_4 + 0x10) < puVar7 + 1) {
        uVar5 = uVar9;
        if (puVar7 < *(ulonglong **)(param_4 + 0x10)) {
          do {
            uVar1 = *puVar7;
            uVar10 = (int)uVar9 + 8;
            uVar9 = (ulonglong)uVar10;
            puVar7 = (ulonglong *)((longlong)puVar7 + 1);
            uVar5 = uVar5 << 8 | (ulonglong)(byte)uVar1;
            *(ulonglong **)(param_4 + 0x40) = puVar7;
          } while (puVar7 < *(ulonglong **)(param_4 + 0x10));
          uVar9 = uVar5 << (-(char)uVar10 & 0x3fU);
        }
      }
      else {
        uVar9 = *puVar7;
        uVar10 = 0x40;
        uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 |
                (uVar9 & 0xff0000000000) >> 0x18 | (uVar9 & 0xff00000000) >> 8 |
                (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18 | (uVar9 & 0xff00) << 0x28 |
                uVar9 << 0x38;
        *(ulonglong **)(param_4 + 0x40) = puVar7 + 1;
      }
      *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar10;
      uVar10 = iVar3 - 0x20;
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar5 = -(ulonglong)(uVar10 < 0x40) & uVar9 << ((byte)uVar10 & 0x3f);
      uVar8 = uVar9 >> (-(byte)uVar10 & 0x3f) | uVar8 >> 0x20;
    }
    else {
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
      uVar5 = uVar8 << 0x20;
      uVar10 = iVar3 + 0x20;
      uVar8 = uVar8 >> 0x20;
    }
    *(ulonglong *)(param_4 + 0x30) = uVar5;
    *(uint *)(param_4 + 0x38) = uVar10;
    puVar4 = (uint *)FUN_140495860(&param_5,uVar8 & 0xffffffff);
  }
  else {
    puVar4 = (uint *)FUN_1407f2034(&param_5,param_4);
  }
  param_3[1] = *puVar4;
  return 1;
}

