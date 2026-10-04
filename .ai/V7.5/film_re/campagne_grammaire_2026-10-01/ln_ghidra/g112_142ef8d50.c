
undefined1 FUN_142ef8d50(undefined8 param_1,undefined8 param_2,longlong param_3,longlong param_4)

{
  undefined1 uVar1;
  
  FUN_142ef2660(param_4);
  FUN_142ef2744(param_4,param_2,param_3 + 4);
  uVar1 = 0;
  if ((*(char *)(param_4 + 0x24) == '\0') &&
     (*(int *)(param_4 + 0x2c) <= *(int *)(param_4 + 0x18) * 8)) {
    uVar1 = 1;
  }
  return uVar1;
}

