
undefined8
FUN_142f15ec0(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4,
             undefined1 param_5)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong *puVar3;
  int iVar4;
  uint uVar5;
  uint uVar6;
  ulonglong uVar7;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 6) {
    puVar3 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    iVar4 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar3 + 1) {
      if (puVar3 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar3;
          iVar4 = iVar4 + 8;
          puVar3 = (ulonglong *)((longlong)puVar3 + 1);
          uVar7 = uVar7 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar3;
        } while (puVar3 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar7 << (0x40U - (char)iVar4 & 0x3f);
      }
    }
    else {
      uVar7 = *puVar3;
      iVar4 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar3 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar4;
    uVar5 = iVar1 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar2 = -(ulonglong)(uVar5 < 0x40) & uVar7 << ((byte)uVar5 & 0x3f);
    uVar6 = (uint)(uVar7 >> (0x40 - (byte)uVar5 & 0x3f)) | uVar6 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar2 = *(longlong *)(param_4 + 0x30) << 6;
    uVar5 = iVar1 + 6;
    uVar6 = uVar6 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar2;
  *(uint *)(param_4 + 0x38) = uVar5;
  *param_3 = uVar6;
  FUN_14076e494(param_4,param_3 + 1,0x10,0,param_5,0);
  return 1;
}

