
void FUN_141fd72c0(longlong param_1,undefined8 param_2,ushort *param_3)

{
  longlong lVar1;
  ushort uVar2;
  
  if (8 < 0x40 - *(int *)(param_1 + 0x38)) {
    lVar1 = *(longlong *)(param_1 + 0x30);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 9;
    *(longlong *)(param_1 + 0x30) = lVar1 << 9;
    *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 9;
    *param_3 = (ushort)((ulonglong)lVar1 >> 0x37);
    return;
  }
  uVar2 = FUN_1406d6c7c(param_1,9);
  *param_3 = uVar2;
  return;
}

