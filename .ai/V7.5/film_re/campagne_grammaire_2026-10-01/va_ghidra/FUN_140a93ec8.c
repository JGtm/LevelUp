int FUN_140a93ec8(int param_1)
{
  int iVar1;
  if (param_1 == 0) {
    if (DAT_145121140 != 0) {
      if (DAT_145121140 == 1) {
        FUN_142b6b170(&DAT_145120f00,0);
      }
      else {
        (*(code *)*DAT_145120f00)(&DAT_145120f00,0);
      }
    }
    DAT_145121140 = 0;
  }
  else if (param_1 == 1) {
    FUN_142b5c658();
  }
  else if (param_1 == 2) {
    FUN_1424d29d8(&DAT_145120f00);
    FUN_1409d326c(&DAT_145120f00);
    DAT_145121080 = 0;
    DAT_145120f00 = &PTR_FUN_143dedf48;
    DAT_145121140 = 2;
  }
  else {
    if (DAT_145121140 != 0) {
      if (DAT_145121140 == 1) {
        FUN_142b6b170(&DAT_145120f00,0);
      }
      else {
        (*(code *)*DAT_145120f00)(&DAT_145120f00,0);
      }
    }
    FUN_1409d30d0(&DAT_145120f00);
    DAT_145121140 = 3;
  }
  if (DAT_145121140 == 0) {
    DAT_1452f2f08 = (undefined8 *)0x0;
    iVar1 = 0;
  }
  else {
    iVar1 = DAT_145121140 - 1;
    DAT_1452f2f08 = &DAT_145120f00;
  }
  return iVar1;
}
