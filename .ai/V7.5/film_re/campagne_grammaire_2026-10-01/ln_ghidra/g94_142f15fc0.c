
undefined8 FUN_142f15fc0(ulonglong param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  uint uVar3;
  ulonglong uVar4;
  byte bVar5;
  ulonglong *puVar6;
  uint uVar7;
  ulonglong uVar8;
  undefined4 local_res18 [4];
  
  iVar1 = *(int *)(param_4 + 0x38);
  uVar3 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 6) {
    puVar6 = *(ulonglong **)(param_4 + 0x40);
    uVar8 = 0;
    uVar7 = 0;
    if (*(ulonglong **)(param_4 + 0x10) < puVar6 + 1) {
      uVar4 = uVar8;
      if (puVar6 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar6;
          uVar7 = (int)uVar8 + 8;
          uVar8 = (ulonglong)uVar7;
          puVar6 = (ulonglong *)((longlong)puVar6 + 1);
          uVar4 = uVar4 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar6;
        } while (puVar6 < *(ulonglong **)(param_4 + 0x10));
        uVar8 = uVar4 << (0x40U - (char)uVar7 & 0x3f);
      }
    }
    else {
      uVar8 = *puVar6;
      uVar7 = 0x40;
      uVar8 = uVar8 >> 0x38 | (uVar8 & 0xff000000000000) >> 0x28 | (uVar8 & 0xff0000000000) >> 0x18
              | (uVar8 & 0xff00000000) >> 8 | (uVar8 & 0xff000000) << 8 | (uVar8 & 0xff0000) << 0x18
              | (uVar8 & 0xff00) << 0x28 | uVar8 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar6 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar7;
    uVar7 = iVar1 - 0x3a;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    bVar5 = 0x40 - (byte)uVar7;
    param_1 = (ulonglong)bVar5;
    uVar4 = -(ulonglong)(uVar7 < 0x40) & uVar8 << ((byte)uVar7 & 0x3f);
    uVar3 = (uint)(uVar8 >> (bVar5 & 0x3f)) | uVar3 >> 0x1a;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 6;
    uVar4 = *(longlong *)(param_4 + 0x30) << 6;
    uVar7 = iVar1 + 6;
    uVar3 = uVar3 >> 0x1a;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar4;
  *(uint *)(param_4 + 0x38) = uVar7;
  local_res18[0] = 0xffffffff;
  *param_3 = uVar3;
  FUN_1406d3140(param_1,param_4,7,local_res18);
  uVar3 = FUN_140809d20(local_res18[0],0);
  param_3[1] = uVar3;
  return 1;
}

