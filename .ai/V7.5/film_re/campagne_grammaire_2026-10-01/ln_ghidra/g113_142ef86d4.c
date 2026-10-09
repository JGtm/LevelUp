
undefined4 FUN_142ef86d4(undefined8 param_1,undefined8 param_2,byte *param_3,longlong param_4)

{
  uint uVar1;
  ulonglong uVar2;
  int iVar3;
  undefined4 *puVar4;
  ulonglong *puVar5;
  int iVar6;
  byte bVar7;
  ulonglong uVar8;
  undefined1 uVar9;
  undefined1 local_res18 [16];
  
  puVar4 = (undefined4 *)FUN_1407f1ff4(local_res18,param_4,"player");
  uVar9 = 1;
  *(undefined4 *)(param_3 + 4) = *puVar4;
  iVar3 = *(int *)(param_4 + 0x38);
  bVar7 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (0x40 - iVar3 < 8) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    iVar6 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar5;
          iVar6 = iVar6 + 8;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar8 = uVar8 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar8 << (0x40U - (char)iVar6 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar5;
      iVar6 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + iVar6;
    uVar1 = iVar3 - 0x38;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    *(ulonglong *)(param_4 + 0x30) = -(ulonglong)(uVar1 < 0x40) & uVar8 << ((byte)uVar1 & 0x3f);
    *(uint *)(param_4 + 0x38) = uVar1;
    bVar7 = (byte)(uVar8 >> (0x40 - (byte)uVar1 & 0x3f)) | bVar7;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 8;
    *(int *)(param_4 + 0x38) = iVar3 + 8;
  }
  *param_3 = bVar7;
  iVar3 = *(int *)(param_4 + 0x18) * 8;
  if ((*(char *)(param_4 + 0x24) != '\0') || (iVar3 < *(int *)(param_4 + 0x2c))) {
    uVar9 = 0;
  }
  return CONCAT31((int3)((uint)iVar3 >> 8),uVar9);
}

