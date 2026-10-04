
undefined8 FUN_1404f293c(void)

{
  char *pcVar1;
  ulonglong uVar2;
  
  uVar2 = 0;
  pcVar1 = *(char **)(*(longlong *)ThreadLocalStoragePointer + 0x238);
  if ((pcVar1 != (char *)0x0) && ((*pcVar1 != '\0' || (pcVar1[1] != '\0')))) {
    if (DAT_144e518fc == '\0') {
      FUN_141168be4();
    }
    uVar2 = (ulonglong)*(uint *)((ulonglong)(DAT_1445c5838 & 0xffff) * 0x1134f0 + 4 + DAT_145121d28)
    ;
  }
  return CONCAT71((int7)(uVar2 >> 8),(int)uVar2 != 2);
}

