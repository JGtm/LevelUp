
undefined1
FUN_142f16cac(undefined8 param_1,undefined8 param_2,ushort *param_3,undefined8 param_4,
             undefined1 param_5)

{
  undefined1 uVar1;
  
  uVar1 = 0;
  if (*(int *)(DAT_144c1cfa8 + 4) == 2) {
    uVar1 = 1;
    FUN_142ef46e8(param_3 + 1,param_4);
    FUN_1424d9b10(param_4);
    if (*param_3 < 2) {
      if (*param_3 == 0) {
        FUN_1424d9a30(param_4);
      }
      else if (*param_3 == 1) {
        FUN_14080d69c(1,param_4,param_3 + 2,0xffffffff);
        FUN_14080dec4(param_4,"variant-name",param_3 + 4);
      }
      FUN_14076e494(param_4,param_3 + 6,0x10,0,param_5,0);
      FUN_14076dc04(param_4);
      param_3[0x1a] = 0xffff;
      param_3[0x1b] = 0xffff;
      *(undefined1 *)(param_3 + 0x12) = 0;
      *(undefined1 *)(param_3 + 0x16) = 0;
      param_3[0x18] = 0xffff;
      param_3[0x19] = 0xffff;
      param_3[0x14] = 0xffff;
      param_3[0x15] = 0xffff;
      FUN_1408eff64(param_3 + 0x12,param_4,param_5);
      if ((char)param_3[0x16] != '\0') {
        FUN_140c9e738(param_3 + 0x1c,param_4,1);
      }
      if (*param_3 == 0) {
        uVar1 = FUN_1408dcb2c(CONCAT71((int7)((ulonglong)(param_3 + 2) >> 8),(char)param_3[2]));
      }
      else {
        uVar1 = FUN_1405838f0();
      }
    }
  }
  return uVar1;
}

