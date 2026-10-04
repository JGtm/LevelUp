
undefined8 FUN_141fd8740(undefined8 param_1,undefined8 param_2,undefined4 *param_3,longlong param_4)

{
  longlong lVar1;
  undefined1 uVar2;
  undefined4 uVar3;
  undefined4 local_res18 [4];
  
  uVar3 = FUN_1407f2058(param_4);
  FUN_140495860(local_res18,uVar3);
  *param_3 = local_res18[0];
  if (0 < 0x40 - *(int *)(param_4 + 0x38)) {
    lVar1 = *(longlong *)(param_4 + 0x30);
    *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 1;
    *(longlong *)(param_4 + 0x30) = lVar1 * 2;
    *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 1;
    *(byte *)(param_3 + 1) = (byte)((ulonglong)lVar1 >> 0x3f);
    return 1;
  }
  uVar2 = FUN_1406d6c7c(param_4,1);
  *(undefined1 *)(param_3 + 1) = uVar2;
  return 1;
}

