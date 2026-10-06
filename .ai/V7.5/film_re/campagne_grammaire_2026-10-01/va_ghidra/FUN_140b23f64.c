ulonglong FUN_140b23f64(undefined8 param_1,undefined8 param_2,longlong param_3,char *param_4)
{
  ulonglong uVar1;
  if ((*(int *)(param_3 + 0x50) != 0) ||
     (uVar1 = 0, (bool)*param_4 != (*(longlong *)(param_3 + 0x58) != 0))) {
    uVar1 = FUN_1424d6668();
  }
  return uVar1 & 0xffffffffffffff00;
}
