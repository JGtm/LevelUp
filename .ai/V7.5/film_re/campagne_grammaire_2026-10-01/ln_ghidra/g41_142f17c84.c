
undefined8 FUN_142f17c84(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  ulonglong uVar3;
  uint *puVar4;
  ulonglong *puVar5;
  uint uVar6;
  ulonglong uVar7;
  uint uVar8;
  undefined1 local_res18 [16];
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 3) {
    puVar5 = *(ulonglong **)(param_4 + 0x40);
    uVar7 = 0;
    uVar8 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar5 + 1) {
      uVar3 = uVar7;
      if (puVar5 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar5;
          uVar8 = (int)uVar7 + 8;
          uVar7 = (ulonglong)uVar8;
          puVar5 = (ulonglong *)((longlong)puVar5 + 1);
          uVar3 = uVar3 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar5;
        } while (puVar5 < *(ulonglong **)(param_4 + 0x10));
        uVar7 = uVar3 << (0x40U - (char)uVar8 & 0x3f);
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
    uVar8 = iVar1 - 0x3d;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar3 = -(ulonglong)(uVar8 < 0x40) & uVar7 << ((byte)uVar8 & 0x3f);
    uVar6 = (uint)(uVar7 >> (0x40 - (byte)uVar8 & 0x3f)) | uVar6 >> 0x1d;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 3;
    uVar3 = *(longlong *)(param_4 + 0x30) * 8;
    uVar8 = iVar1 + 3;
    uVar6 = uVar6 >> 0x1d;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar3;
  *(uint *)(param_4 + 0x38) = uVar8;
  *param_3 = uVar6;
  puVar4 = (uint *)FUN_1407f2034(local_res18,param_4);
  param_3[1] = *puVar4;
  return 1;
}

