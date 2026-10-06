undefined1 FUN_142ef93e0(undefined8 param_1,undefined8 param_2,byte *param_3,undefined8 param_4)
{
  byte bVar1;
  char cVar2;
  bVar1 = FUN_1406cf008(param_4);
  *param_3 = *param_3 & 0xfe | bVar1;
  if ((bVar1 & 1) != 0) {
    FUN_140c5f938(param_4,param_3 + 4,param_3 + 0x10,0);
  }
  cVar2 = FUN_1406cf008(param_4);
  *param_3 = *param_3 & 0xfd;
  *param_3 = *param_3 | cVar2 * '\x02';
  cVar2 = FUN_14080d69c();
  if (cVar2 != '\0') {
    FUN_1424e0e38(param_4,param_3 + 0x20,0x10);
    FUN_1424e0e38(param_4,param_3 + 0x2c,0x10);
  }
  return 1;
}
