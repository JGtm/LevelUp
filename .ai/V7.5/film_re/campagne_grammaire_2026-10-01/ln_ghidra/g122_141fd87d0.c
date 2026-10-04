
undefined8 FUN_141fd87d0(undefined8 param_1,undefined8 param_2,undefined4 *param_3,longlong param_4)

{
  longlong lVar1;
  undefined4 uVar2;
  undefined4 local_res18 [4];
  
  uVar2 = FUN_1407f2058(param_4);
  FUN_140495860(local_res18,uVar2);
  *param_3 = local_res18[0];
  if (0x40 - *(int *)(param_4 + 0x38) < 0x20) {
    uVar2 = FUN_1406d6c7c(param_4,0x20);
  }
  else {
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    uVar2 = (undefined4)((ulonglong)*(longlong *)(param_4 + 0x30) >> 0x20);
    *(longlong *)(param_4 + 0x30) = *(longlong *)(param_4 + 0x30) << 0x20;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
  }
  param_3[1] = uVar2;
  if (0x1f < 0x40 - *(int *)(param_4 + 0x38)) {
    lVar1 = *(longlong *)(param_4 + 0x30);
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 0x20;
    *(longlong *)(param_4 + 0x30) = lVar1 << 0x20;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 0x20;
    param_3[2] = (int)((ulonglong)lVar1 >> 0x20);
    return 1;
  }
  uVar2 = FUN_1406d6c7c(param_4,0x20);
  param_3[2] = uVar2;
  return 1;
}

