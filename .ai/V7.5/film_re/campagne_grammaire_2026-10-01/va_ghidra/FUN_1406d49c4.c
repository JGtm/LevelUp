
void FUN_1406d49c4(longlong param_1,undefined8 param_2,byte param_3)

{
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(uint *)(param_1 + 0x38) = *(uint *)(param_1 + 0x38) + 1;
    *(ulonglong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2 | (ulonglong)param_3;
    return;
  }
  FUN_1406d6e28(param_1,(ulonglong)param_3,1);
  return;
}

