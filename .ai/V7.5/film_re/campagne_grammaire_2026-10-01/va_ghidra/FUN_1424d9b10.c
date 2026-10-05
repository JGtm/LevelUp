
void FUN_1424d9b10(longlong param_1,undefined8 param_2,ushort *param_3)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong *puVar3;
  ushort uVar4;
  ulonglong uVar5;
  ulonglong uVar6;
  uint uVar7;
  
  iVar1 = *(int *)(param_1 + 0x38);
  uVar4 = (ushort)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x30);
  if (0x40 - iVar1 < 1) {
    puVar3 = *(ulonglong **)(param_1 + 0x40);
    uVar5 = 0;
    uVar7 = 0;
    if (*(ulonglong **)(param_1 + 0x10) < puVar3 + 1) {
      uVar6 = uVar5;
      if (puVar3 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          uVar2 = *puVar3;
          uVar7 = (int)uVar5 + 8;
          uVar5 = (ulonglong)uVar7;
          puVar3 = (ulonglong *)((longlong)puVar3 + 1);
          uVar6 = uVar6 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_1 + 0x40) = puVar3;
        } while (puVar3 < *(ulonglong **)(param_1 + 0x10));
        uVar5 = uVar6 << (0x40U - (char)uVar7 & 0x3f);
      }
    }
    else {
      uVar5 = *puVar3;
      uVar7 = 0x40;
      uVar5 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 | (uVar5 & 0xff0000000000) >> 0x18
              | (uVar5 & 0xff00000000) >> 8 | (uVar5 & 0xff000000) << 8 | (uVar5 & 0xff0000) << 0x18
              | (uVar5 & 0xff00) << 0x28 | uVar5 << 0x38;
      *(ulonglong **)(param_1 + 0x40) = puVar3 + 1;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + uVar7;
    uVar7 = iVar1 - 0x3f;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    uVar4 = (ushort)(uVar5 >> (0x40 - (byte)uVar7 & 0x3f)) | uVar4 >> 0xf;
    *(ulonglong *)(param_1 + 0x30) = -(ulonglong)(uVar7 < 0x40) & uVar5 << ((byte)uVar7 & 0x3f);
  }
  else {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    uVar7 = iVar1 + 1;
    *(longlong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2;
    uVar4 = uVar4 >> 0xf;
  }
  *(uint *)(param_1 + 0x38) = uVar7;
  *param_3 = uVar4;
  return;
}

