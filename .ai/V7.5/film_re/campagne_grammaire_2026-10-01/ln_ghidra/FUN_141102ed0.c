
ulonglong FUN_141102ed0(int param_1)

{
  char cVar1;
  ulonglong uVar2;
  
  cVar1 = FUN_1404f2b4c();
  if (cVar1 == '\0') {
    uVar2 = (ulonglong)*(uint *)(&DAT_14474cd90 + (longlong)param_1 * 8);
  }
  else {
    uVar2 = FUN_1428e1c64(&DAT_144c23178,param_1);
  }
  return uVar2;
}

