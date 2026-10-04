
undefined4 FUN_142ef94f4(undefined8 param_1,undefined8 param_2,longlong param_3,undefined8 param_4)

{
  undefined4 uVar1;
  
  FUN_14080d69c(param_1,param_4,param_3,0xffffffff);
  FUN_14080dec4(param_4,"variant-name",param_3 + 4);
  uVar1 = FUN_1406d00ec(param_4);
  *(undefined4 *)(param_3 + 8) = uVar1;
  return 1;
}

