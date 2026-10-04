
undefined8 FUN_142ef9284(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  ulonglong *puVar4;
  ushort uVar5;
  ulonglong uVar6;
  uint uVar7;
  undefined8 uVar8;
  
  FUN_1424e0e38(param_4,param_3,0x10);
  iVar1 = *(int *)(param_4 + 0x38);
  uVar5 = (ushort)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x30);
  if (0x40 - iVar1 < 2) {
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
    uVar7 = iVar1 - 0x3e;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar3 = -(ulonglong)(uVar7 < 0x40) & uVar6 << ((byte)uVar7 & 0x3f);
    uVar5 = (ushort)(uVar6 >> (0x40 - (byte)uVar7 & 0x3f)) | uVar5 >> 0xe;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 2;
    uVar3 = *(longlong *)(param_4 + 0x30) * 4;
    uVar7 = iVar1 + 2;
    uVar5 = uVar5 >> 0xe;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar7;
  *(ushort *)(param_3 + 0x24) = uVar5;
  if (((byte)uVar5 & DAT_14473faa8 & 3) == 0) {
    *(undefined8 *)(param_3 + 0xc) = DAT_143d81f98;
    *(undefined4 *)(param_3 + 0x14) = DAT_143d81fa0;
    *(undefined8 *)(param_3 + 0x18) = DAT_143d81f98;
    *(undefined4 *)(param_3 + 0x20) = DAT_143d81fa0;
  }
  else {
    FUN_140c5f938(param_4,param_3 + 0xc,param_3 + 0x18,0);
  }
  uVar8 = FUN_1407f08bc(param_4,param_3 + 0x26);
  FUN_14080d69c(uVar8,param_4,param_3 + 0x28,0xffffffff);
  return 1;
}

