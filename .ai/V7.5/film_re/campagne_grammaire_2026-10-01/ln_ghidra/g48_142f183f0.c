
bool FUN_142f183f0(undefined8 param_1,undefined8 param_2,uint *param_3,undefined8 param_4)

{
  undefined1 uVar1;
  uint uVar2;
  uint uVar3;
  uint *puVar4;
  undefined1 local_res18 [16];
  
  uVar2 = FUN_1406d00ec(param_4);
  *param_3 = uVar2;
  FUN_14080d69c();
  FUN_14080dec4(param_4,"variant_name",param_3 + 2);
  puVar4 = (uint *)FUN_14080d61c(local_res18,param_3 + 1,param_3[2]);
  param_3[3] = *puVar4;
  uVar3 = FUN_141102ed0(0x30);
  if (1 < uVar3) {
    uVar1 = FUN_1406cf008(param_4);
    *(undefined1 *)(param_3 + 4) = uVar1;
  }
  return uVar2 == 0xffffffff || uVar2 < 4;
}

