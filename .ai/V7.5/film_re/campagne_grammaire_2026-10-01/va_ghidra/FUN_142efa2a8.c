void FUN_142efa2a8(undefined8 param_1,undefined8 param_2,byte *param_3,undefined8 param_4)
{
  char cVar1;
  FUN_1406d49c4(param_4,param_2,
                CONCAT71((int7)((ulonglong)param_3 >> 8),*param_3) & 0xffffffffffffff01);
  if ((*param_3 & 1) != 0) {
    FUN_141f86118(param_4,param_3 + 4,param_3 + 0x10,0);
  }
  FUN_1406d49c4(param_4);
  cVar1 = FUN_1407edb6c();
  if (cVar1 != '\0') {
    FUN_141f860b0(param_4,param_3 + 0x20);
    FUN_141f860b0(param_4,param_3 + 0x2c);
  }
  return;
}
