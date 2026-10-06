longlong FUN_140ad2d3c(longlong param_1,undefined8 param_2,undefined8 param_3,undefined1 param_4)
{
  undefined8 uVar1;
  FUN_1404ed848();
  FUN_1404ed848(param_1 + 0x20,&DAT_14367fb48);
  *(undefined1 *)(param_1 + 0x40) = param_4;
  *(undefined8 *)(param_1 + 0x48) = &PTR_LAB_14369ee90;
  *(undefined8 **)(param_1 + 0x80) = (undefined8 *)(param_1 + 0x48);
  *(undefined1 *)(param_1 + 0x88) = param_4;
  *(undefined1 *)(param_1 + 0x89) = 0;
  uVar1 = FUN_140ad32c4();
  FUN_140ad2dbc(uVar1,param_2,param_1);
  return param_1;
}
