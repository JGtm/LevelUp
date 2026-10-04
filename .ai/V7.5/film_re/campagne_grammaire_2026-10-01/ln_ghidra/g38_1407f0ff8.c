
undefined8
FUN_1407f0ff8(undefined8 param_1,undefined8 param_2,undefined1 *param_3,undefined8 param_4)

{
  undefined1 uVar1;
  undefined4 uVar2;
  undefined4 local_res18 [4];
  
  uVar1 = FUN_1406cf008(param_4);
  *param_3 = uVar1;
  uVar1 = FUN_1406cf008(param_4);
  param_3[1] = uVar1;
  uVar1 = FUN_1406cf008(param_4);
  param_3[2] = uVar1;
  uVar1 = FUN_1406cf008(param_4);
  param_3[3] = uVar1;
  uVar2 = FUN_1407f2058(param_4);
  FUN_140495860(local_res18,uVar2);
  *(undefined4 *)(param_3 + 4) = local_res18[0];
  return CONCAT71((uint7)(uint3)((uint)local_res18[0] >> 8),1);
}

