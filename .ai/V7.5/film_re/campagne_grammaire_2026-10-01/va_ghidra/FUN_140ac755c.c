void FUN_140ac755c(undefined8 *param_1,undefined8 param_2,char param_3)
{
  undefined4 *puVar1;
  if ((param_3 == '\0') && (*(short *)(param_1 + 2) == 2)) {
    puVar1 = (undefined4 *)param_1[1];
    param_1[1] = puVar1 + 1;
    FUN_140ac7668(*param_1,*puVar1);
  }
  return;
}
