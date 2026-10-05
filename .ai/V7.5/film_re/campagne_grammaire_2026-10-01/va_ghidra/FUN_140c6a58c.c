
undefined1 FUN_140c6a58c(undefined8 param_1,undefined8 param_2,ushort *param_3,undefined8 param_4)

{
  undefined1 uVar1;
  undefined4 uVar2;
  uint local_18 [4];
  
  uVar1 = 0;
  if (*(int *)(DAT_144c1cfa8 + 4) == 2) {
    FUN_1424d9b10(param_4);
    if (*param_3 < 2) {
      if (*param_3 == 0) {
        FUN_1424d9a30(param_4);
      }
      else if (*param_3 == 1) {
        FUN_14080d69c(1,param_4,param_3 + 2,0xffffffff);
        uVar2 = FUN_142ed0674(param_4);
        *(undefined4 *)(param_3 + 6) = uVar2;
      }
      if (*param_3 == 0) {
        uVar1 = FUN_1408dcb2c(CONCAT71((int7)((ulonglong)(param_3 + 2) >> 8),(char)param_3[2]));
      }
      else {
        uVar1 = FUN_1405838f0();
      }
      uVar2 = FUN_1407f2058(param_4);
      FUN_140495860(local_18,uVar2);
      *(uint *)(param_3 + 8) = local_18[0];
    }
  }
  return uVar1;
}

