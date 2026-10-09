void FUN_141f86118(undefined8 param_1,undefined8 param_2,undefined8 param_3,int param_4)
{
  undefined8 uVar1;
  undefined1 local_18 [16];
  if (((DAT_145121140 != '\x01') || (0 < param_4)) && (param_4 != 1)) {
    if (param_4 != 2) {
      uVar1 = FUN_140cfff40(local_18);
      FUN_142e2d8cc(param_1,uVar1);
      return;
    }
    FUN_1406d60f4(param_1,param_2,param_2,0x60);
    FUN_1406d60f4(param_1);
    return;
  }
  FUN_142e29e70(param_1,param_2,param_2,param_3);
  return;
}
