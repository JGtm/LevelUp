
ulonglong FUN_1405f50b8(void)

{
  undefined1 auVar1 [16];
  ulonglong uVar2;
  longlong lVar3;
  LARGE_INTEGER local_res8;
  LARGE_INTEGER local_res10 [3];
  
  if (DAT_144989d58 == '\0') {
    QueryPerformanceCounter(local_res10);
    QueryPerformanceFrequency(&local_res8);
    uVar2 = ((ulonglong)local_res10[0].QuadPart >> 0x20) * 1000000;
    uVar2 = (uVar2 / (ulonglong)local_res8.QuadPart << 0x20) +
            ((local_res10[0].QuadPart & 0xffffffffU) * 1000000 +
            (uVar2 % (ulonglong)local_res8.QuadPart << 0x20)) / (ulonglong)local_res8.QuadPart;
    auVar1._8_8_ = 0;
    auVar1._0_8_ = uVar2;
    lVar3 = SUB168(ZEXT816(0x624dd2f1a9fbe77) * auVar1,8);
    uVar2 = (uVar2 - lVar3 >> 1) + lVar3 >> 9;
  }
  else {
    uVar2 = (ulonglong)DAT_144989d5c;
  }
  return uVar2 & 0xffffffff;
}

