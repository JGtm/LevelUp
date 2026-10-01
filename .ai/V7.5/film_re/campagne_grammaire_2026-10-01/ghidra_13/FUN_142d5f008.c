
undefined1 FUN_142d5f008(void)

{
  undefined1 uVar1;
  int iVar2;
  LARGE_INTEGER local_res10;
  LARGE_INTEGER local_res18 [2];
  
  FUN_1424c9188();
  QueryPerformanceCounter(&local_res10);
  if ((DAT_144db4318 != '\0') && (DAT_144db4328 != '\0')) {
    FUN_14076bb10();
    FUN_140518b0c();
    FUN_140514114(DAT_144e61a10);
    iVar2 = FUN_1405f6254();
    if (iVar2 == 4) {
      FUN_142f2c3b0();
    }
    FUN_14051aecc();
    DAT_144989d58 = 0;
    DAT_144989d5c = FUN_1405a70d4();
  }
  QueryPerformanceCounter(local_res18);
  uVar1 = DAT_144e61a30;
  LOCK();
  DAT_144e61a30 = 1;
  UNLOCK();
  return uVar1;
}

