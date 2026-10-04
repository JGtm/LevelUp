
bool FUN_142f17ffc(undefined8 param_1,undefined8 param_2,uint *param_3,longlong param_4)

{
  int iVar1;
  ulonglong uVar2;
  char cVar3;
  uint uVar4;
  uint uVar5;
  ulonglong uVar6;
  uint *puVar7;
  ulonglong *puVar8;
  ulonglong uVar9;
  uint uVar10;
  undefined1 local_res18 [16];
  
  uVar4 = FUN_1406d00ec(param_4);
  uVar5 = 0xffffffff;
  *param_3 = uVar4;
  uVar9 = 0;
  uVar10 = 0;
  puVar7 = param_3 + 1;
  FUN_14080d69c(uVar4,param_4,puVar7,0xffffffff);
  FUN_14080dec4(param_4,"variant_name",param_3 + 2);
  cVar3 = FUN_1405838f0(puVar7);
  if (cVar3 != '\0') {
    uVar5 = FUN_142e2de9c(puVar7,param_3[2]);
  }
  param_3[3] = uVar5;
  iVar1 = *(int *)(param_4 + 0x38);
  uVar5 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
  if (0x40 - iVar1 < 1) {
    puVar8 = *(ulonglong **)(param_4 + 0x40);
    if (*(ulonglong **)(param_4 + 0x10) < puVar8 + 1) {
      uVar6 = uVar9;
      if (puVar8 < *(ulonglong **)(param_4 + 0x10)) {
        do {
          uVar2 = *puVar8;
          uVar10 = (int)uVar9 + 8;
          uVar9 = (ulonglong)uVar10;
          puVar8 = (ulonglong *)((longlong)puVar8 + 1);
          uVar6 = uVar6 << 8 | (ulonglong)(byte)uVar2;
          *(ulonglong **)(param_4 + 0x40) = puVar8;
        } while (puVar8 < *(ulonglong **)(param_4 + 0x10));
        uVar9 = uVar6 << (-(char)uVar10 & 0x3fU);
      }
    }
    else {
      uVar9 = *puVar8;
      uVar10 = 0x40;
      uVar9 = uVar9 >> 0x38 | (uVar9 & 0xff000000000000) >> 0x28 | (uVar9 & 0xff0000000000) >> 0x18
              | (uVar9 & 0xff00000000) >> 8 | (uVar9 & 0xff000000) << 8 | (uVar9 & 0xff0000) << 0x18
              | (uVar9 & 0xff00) << 0x28 | uVar9 << 0x38;
      *(ulonglong **)(param_4 + 0x40) = puVar8 + 1;
    }
    *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + uVar10;
    uVar10 = iVar1 - 0x3f;
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar6 = -(ulonglong)(uVar10 < 0x40) & uVar9 << ((byte)uVar10 & 0x3f);
    uVar5 = (uint)(uVar9 >> (-(byte)uVar10 & 0x3f)) | uVar5 >> 0x1f;
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    uVar6 = *(longlong *)(param_4 + 0x30) * 2;
    uVar10 = iVar1 + 1;
    uVar5 = uVar5 >> 0x1f;
  }
  *(ulonglong *)(param_4 + 0x30) = uVar6;
  *(uint *)(param_4 + 0x38) = uVar10;
  param_3[4] = uVar5;
  puVar7 = (uint *)FUN_1407f2034(local_res18,param_4);
  param_3[5] = *puVar7;
  return uVar4 == 0xffffffff || uVar4 < 4;
}

