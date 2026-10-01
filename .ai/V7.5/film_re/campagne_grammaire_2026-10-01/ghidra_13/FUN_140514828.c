
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_140514828(void)

{
  char cVar1;
  int iVar2;
  LARGE_INTEGER local_res8;
  LARGE_INTEGER local_res10 [3];
  
  QueryPerformanceCounter(&local_res8);
  _DAT_144db4350 = _DAT_144db4350 + 1;
  if ((DAT_144db4318 != '\0') && (DAT_144db4328 != '\0')) {
    DAT_144db431a = 1;
    DAT_144989d58 = 1;
    DAT_144989d5c = FUN_1405a70d4();
    FUN_1404e5f44();
    cVar1 = FUN_1406cb0a4();
    if (cVar1 == '\0') {
      FUN_14076bb10();
    }
    FUN_140518b0c();
    FUN_140514114(DAT_144e61a10);
    iVar2 = FUN_1405f6254();
    if (iVar2 == 4) {
      cVar1 = FUN_1406cb0a4();
      if (cVar1 == '\0') {
        FUN_142f2c3b0();
      }
    }
    DAT_144989d58 = 0;
    DAT_144989d5c = FUN_1405a70d4();
    DAT_144db431a = 0;
  }
  QueryPerformanceCounter(local_res10);
  return;
}

