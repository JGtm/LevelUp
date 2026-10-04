
undefined8 FUN_142f168c0(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  uint uVar5;
  ulonglong uVar6;
  uint uVar7;
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 6) {
    puVar4 = *(ulonglong **)(param_4 + 0x40);
    uVar6 = 0;
    uVar7 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar4 + 1) {
      uVar3 = uVar6;
      if (puVar4 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar4;
          uVar7 = (int)uVar6 + 8;
          uVar6 = (ulonglong)uVar7;
          puVar4 = (ulonglong *)((longlong)puVar4 + 1);
          uVar3 = uVar3 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar4;
        } while (puVar4 < *(ulonglong **)(param_4 + 0x10));
        uVar6 = uVar3 << (0x40U - (char)uVar7 & 0x3f);
      }
    }
    else {
      uVar6 = *puVar4;
      uVar7 = 0x40;
      uVar6 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 | (uVar6 & 0xff0000000000) >> 0x18
              | (uVar6 & 0xff00000000) >> 8 | (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18
              | (uVar6 & 0xff00) << 0x28 | uVar6 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar4 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar7;
    uVar7 = iVar1 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = -(ulonglong)(uVar7 < 0x40) & uVar6 << ((byte)uVar7 & 0x3f);
    uVar5 = (uint)(uVar6 >> (0x40 - (byte)uVar7 & 0x3f)) | uVar5 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar3 = *(longlong *)(param_4 + 0x30) << 6;
    uVar7 = iVar1 + 6;
    uVar5 = uVar5 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar7;
  *param_3 = uVar5;
  return CONCAT71((int7)(uVar3 >> 8),1);
}

