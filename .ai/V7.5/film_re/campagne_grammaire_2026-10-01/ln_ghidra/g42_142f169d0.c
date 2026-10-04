
undefined8 FUN_142f169d0(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  int iVar1;
  byte bVar2;
  ulonglong uVar3;
  ulonglong uVar4;
  undefined4 *puVar5;
  ulonglong *puVar6;
  ushort uVar7;
  ulonglong uVar8;
  uint uVar9;
  undefined1 local_res18 [16];
  
  FUN_141015740(param_4);
  FUN_14076dc04(param_4);
  iVar1 = *(int *)(param_4 + 0x38);
  bVar2 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (0x40 - iVar1 < 8) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    uVar9 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar4 = uVar8;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar3 = *puVar6;
          uVar9 = (int)uVar8 + 8;
          uVar8 = (ulonglong)uVar9;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar3;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar4 << (0x40U - (char)uVar9 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar6;
      uVar9 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar9;
    uVar9 = iVar1 - 0x38;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    uVar4 = -(ulonglong)(uVar9 < 0x40) & uVar8 << ((byte)uVar9 & 0x3f);
    uVar7 = (ushort)(uVar8 >> (0x40 - (byte)uVar9 & 0x3f)) | (ushort)bVar2;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    uVar4 = *(longlong *)(param_4 + 0x30) << 8;
    uVar9 = iVar1 + 8;
    uVar7 = (ushort)bVar2;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar4;
  *(uint *)(param_4 + 0x38) = uVar9;
  *(ushort *)(param_3 + 0x10) = uVar7;
  puVar5 = (undefined4 *)FUN_1407f2034(local_res18,param_4);
  *(undefined4 *)(param_3 + 0x14) = *puVar5;
  return 1;
}

