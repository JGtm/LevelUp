
undefined1 FUN_142ef8b5c(undefined8 param_1,undefined8 param_2,undefined1 *param_3,longlong param_4)

{
  undefined1 uVar1;
  
  uVar1 = FUN_1406cf008(param_4);
  *param_3 = uVar1;
  uVar1 = 0;
  if ((*(char *)(param_4 + 0x24) == '\0') &&
     (*(int *)(param_4 + 0x2c) <= *(int *)(param_4 + 0x18) * 8)) {
    uVar1 = 1;
  }
  return uVar1;
}

