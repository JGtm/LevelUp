
undefined8 FUN_142eec01c(undefined8 param_1,undefined8 param_2,undefined4 *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  undefined4 *puVar3;
  ulonglong uVar4;
  ulonglong *puVar5;
  byte bVar6;
  ulonglong uVar7;
  uint uVar8;
  undefined1 local_res18 [16];
  
  puVar3 = (undefined4 *)FUN_1407f2034(local_res18,param_4);
  *param_3 = *puVar3;
  iVar1 = *(int *)(param_4 + 0x38);
  bVar6 = (byte)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x38);
  if (0x40 - iVar1 < 8) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    uVar8 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar4 = uVar7;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar5;
          uVar8 = (int)uVar7 + 8;
          uVar7 = (ulonglong)uVar8;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar4 << (0x40U - (char)uVar8 & 0x3f);
      }
    }
    else {
      uVar7 = *puVar5;
      uVar8 = 0x40;
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar5 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar8;
    uVar8 = iVar1 - 0x38;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    uVar4 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
    bVar6 = (byte)(uVar7 >> (0x40 - (byte)uVar8 & 0x3f)) | bVar6;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 8;
    uVar4 = *(longlong *)(param_4 + 0x30) << 8;
    uVar8 = iVar1 + 8;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar4;
  *(uint *)(param_4 + 0x38) = uVar8;
  *(byte *)(param_3 + 1) = bVar6;
  return CONCAT71((int7)(uVar4 >> 8),1);
}

