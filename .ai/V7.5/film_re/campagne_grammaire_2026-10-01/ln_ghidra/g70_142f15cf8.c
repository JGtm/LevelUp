
undefined8
FUN_142f15cf8(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4,
             undefined1 param_5)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong *puVar3;
  int iVar4;
  int iVar5;
  uint uVar6;
  ulonglong uVar7;
  uint uVar8;
  undefined4 uVar9;
  
  FUN_14076e494(param_4,param_3,0x10,0,param_5,0);
  FUN_14076e494(param_4,param_3 + 0x18,0x10,0,param_5,0);
  FUN_14076dc04(param_4);
  uVar9 = FUN_1406d84b4(param_4);
  *(undefined4 *)(param_3 + 0x24) = uVar9;
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 2) {
    puVar3 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    iVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar3 + 1) {
      iVar4 = 0;
      if (puVar3 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar3;
          iVar5 = iVar4 + 8;
          puVar3 = (ulonglong *)((longlong)puVar3 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar3;
          iVar4 = iVar5;
        } while (puVar3 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (-(char)iVar5 & 0x3fU);
      }
    }
    else {
      uVar7 = *puVar3;
      iVar5 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar3 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar5;
    uVar8 = iVar1 - 0x3e;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar2 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
    uVar6 = (uint)(uVar7 >> (-(byte)uVar8 & 0x3f)) | uVar6 >> 0x1e;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar2 = *(longlong *)(param_4 + 0x30) * 4;
    uVar8 = iVar1 + 2;
    uVar6 = uVar6 >> 0x1e;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar2;
  *(uint *)(param_4 + 0x38) = uVar8;
  *(uint *)(param_3 + 0x28) = uVar6;
  FUN_141015740(param_4);
  *(undefined4 *)(param_3 + 0x2c) = 0;
  FUN_14080bd28(param_3 + 0x30,param_4);
  return 1;
}

