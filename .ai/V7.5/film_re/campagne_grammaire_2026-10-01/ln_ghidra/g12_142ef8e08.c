
undefined4
FUN_142ef8e08(undefined8 param_1,undefined8 param_2,undefined4 *param_3,undefined8 param_4,
             undefined1 param_5)

{
  char cVar1;
  undefined4 uVar2;
  undefined4 uVar3;
  
  uVar2 = FUN_1407f2058(param_4);
  *param_3 = uVar2;
  uVar2 = FUN_1407f2058(param_4);
  uVar3 = 0xffffffff;
  param_3[1] = uVar2;
  FUN_14080d69c();
  cVar1 = FUN_1405838f0(param_3 + 2);
  if (cVar1 != '\0') {
    uVar3 = FUN_142e2de9c(param_3 + 2,DAT_144b404f0);
  }
  param_3[3] = uVar3;
  FUN_14076e494(param_4,param_3 + 4,0xc,0,param_5,0);
  FUN_14076dc04(param_4);
  return 1;
}

