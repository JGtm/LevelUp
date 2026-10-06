undefined1 FUN_14076d018(void)
{
  longlong lVar1;
  undefined1 uVar2;
  char cVar3;
  uVar2 = 0;
  if (DAT_1451789b8 == '\0') {
    uVar2 = 0;
  }
  else {
    cVar3 = FUN_1406aed00();
    if ((cVar3 != '\0') && (DAT_145121140 != '\x01')) {
      lVar1 = FUN_1404f1614();
      lVar1 = FUN_1406aed80(lVar1 + 0x28);
      uVar2 = *(undefined1 *)(lVar1 + 0x238);
    }
  }
  return uVar2;
}
