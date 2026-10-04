
bool FUN_142f16ad8(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4,
                  undefined1 param_5)

{
  undefined8 uVar1;
  char cVar2;
  byte bVar3;
  undefined1 uVar4;
  undefined4 uVar5;
  uint uVar6;
  undefined4 *puVar7;
  undefined1 local_res18 [16];
  undefined8 local_48;
  undefined4 local_40;
  undefined8 local_38;
  undefined4 uStack_30;
  uint uStack_2c;
  undefined4 local_28;
  undefined4 uStack_24;
  undefined4 uStack_20;
  uint uStack_1c;
  
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  FUN_14080dec4(param_4,"variant-name",param_3 + 4);
  cVar2 = FUN_1405838f0(param_3);
  if (cVar2 == '\0') {
    uVar5 = 0xffffffff;
  }
  else {
    puVar7 = (undefined4 *)FUN_14080d61c(local_res18,param_3,*(undefined4 *)(param_3 + 4));
    uVar5 = *puVar7;
  }
  *(undefined4 *)(param_3 + 8) = uVar5;
  FUN_142af27f8(param_4);
  bVar3 = FUN_142ed7f24(param_4);
  *(float *)(param_3 + 0x10) = (float)bVar3;
  cVar2 = FUN_1406cf008(param_4);
  *(char *)(param_3 + 0x16) = cVar2;
  if (cVar2 == '\0') {
    uVar4 = FUN_140c1e31c(param_4);
  }
  else {
    uVar4 = 0xff;
  }
  *(undefined1 *)(param_3 + 0x17) = uVar4;
  if (0x40 - *(int *)(param_4 + 0x38) < 4) {
    uVar6 = FUN_1406d6c7c(param_4,4);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 4;
    uVar6 = (uint)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x3c);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 4;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 4;
  }
  *(uint *)(param_3 + 0xc) = uVar6;
  uVar4 = FUN_14101d200(param_4);
  *(undefined1 *)(param_3 + 0x15) = uVar4;
  if (*(char *)(param_3 + 0x14) == '\x03') {
    *(undefined4 *)(param_3 + 0x24) = 0xffffffff;
    *(undefined4 *)(param_3 + 0x28) = 0xffffffff;
    *(undefined4 *)(param_3 + 0x1c) = 0xffffffff;
    *(undefined1 *)(param_3 + 0x18) = 0;
    *(undefined1 *)(param_3 + 0x20) = 0;
    FUN_1408eff64((undefined1 *)(param_3 + 0x18),param_4,param_5);
    if (*(char *)(param_3 + 0x20) != '\0') {
      FUN_140c9e738(param_3 + 0x2c,param_4,1);
    }
  }
  cVar2 = FUN_1406cf008(param_4);
  if (cVar2 == '\0') {
    uStack_1c = uStack_1c & 0xffffff00;
    local_40 = uStack_20;
    uVar6 = uStack_1c;
  }
  else {
    FUN_14076e494(param_4,&local_48,0x10,0,param_5,0);
    local_38 = local_48;
    uVar1 = local_38;
    uStack_30 = local_40;
    uStack_2c = CONCAT31(uStack_2c._1_3_,1);
    local_38._0_4_ = (undefined4)local_48;
    local_38._4_4_ = (undefined4)((ulonglong)local_48 >> 0x20);
    local_28 = (undefined4)local_38;
    uStack_24 = local_38._4_4_;
    uVar6 = uStack_2c;
    local_38 = uVar1;
  }
  *(undefined4 *)(param_3 + 0x38) = local_28;
  *(undefined4 *)(param_3 + 0x3c) = uStack_24;
  *(undefined4 *)(param_3 + 0x40) = local_40;
  *(uint *)(param_3 + 0x44) = uVar6;
  cVar2 = FUN_1404785a0(param_3 + 8);
  return cVar2 != '\0';
}

