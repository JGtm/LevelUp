ulonglong FUN_140d1a268(undefined8 *param_1,undefined2 param_2,longlong param_3,uint *param_4)
{
  ulonglong uVar1;
  if ((*(int *)(param_3 + 0x50) != 0) ||
     (uVar1 = (ulonglong)*(uint *)(param_3 + 0x60), *param_4 != *(uint *)(param_3 + 0x60))) {
    param_1 = (undefined8 *)*param_1;
    FUN_140ac75e8(param_1,0x10,param_2);
    uVar1 = FUN_140ac7668(*param_1,(int)*param_4 >> 0x1f ^ *param_4 * 2);
  }
  return uVar1 & 0xffffffffffffff00;
}
