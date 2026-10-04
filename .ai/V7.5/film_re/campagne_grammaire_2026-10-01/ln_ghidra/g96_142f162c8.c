
undefined8 FUN_142f162c8(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  uint uVar5;
  uint uVar6;
  ulonglong uVar7;
  
  FUN_142af27f8(param_4,param_2,param_3 + 1);
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 6) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    uVar5 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      uVar3 = uVar7;
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar4;
          uVar5 = (int)uVar7 + 8;
          uVar7 = (ulonglong)uVar5;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar3 = uVar3 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar3 << (0x40U - (char)uVar5 & 0x3f);
      }
    }
    else {
      uVar7 = *puVar4;
      uVar5 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar5;
    uVar5 = iVar1 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = -(ulonglong)(uVar5 < 0x40) & uVar7 << ((byte)uVar5 & 0x3f);
    uVar6 = (uint)(uVar7 >> (0x40 - (byte)uVar5 & 0x3f)) | uVar6 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = *(longlong *)(param_4 + 0x30) << 6;
    uVar5 = iVar1 + 6;
    uVar6 = uVar6 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar5;
  *param_3 = uVar6;
  FUN_142b1cf3c(param_4);
  FUN_142efd1ac(param_4);
  return 1;
}

