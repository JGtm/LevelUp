void FUN_140c5f938(undefined8 param_1,undefined8 param_2,undefined8 param_3,int param_4)
{
  undefined1 local_18 [16];
  if (((DAT_145121140 != '\x01') || (0 < param_4)) && (param_4 != 1)) {
    if (param_4 != 2) {
      FUN_140c5fa84(local_18,param_1);
      FUN_140c5f9c8(local_18,param_2,param_3);
      return;
    }
    FUN_1406d676c(param_1,param_2,param_2,0x60);
    FUN_1406d676c(param_1);
    return;
  }
  FUN_142e29bac(param_1,param_2,param_2,param_3);
  return;
}
