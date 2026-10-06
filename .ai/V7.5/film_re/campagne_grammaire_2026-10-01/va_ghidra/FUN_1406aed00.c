undefined1 FUN_1406aed00(void)
{
  char *pcVar1;
  int iVar2;
  undefined1 uVar3;
  pcVar1 = *(char **)(*(longlong *)ThreadLocalStoragePointer + 0x238);
  uVar3 = 0;
  if ((pcVar1 != (char *)0x0) && ((*pcVar1 != '\0' || (pcVar1[1] != '\0')))) {
    uVar3 = 0;
    if (DAT_144e518fc == '\0') {
      uVar3 = 0;
      FUN_141168be4();
    }
    iVar2 = FUN_1406aed60((ulonglong)(DAT_1445c5838 & 0xffff) * 0x1134f0 + DAT_145121d28);
    if (iVar2 == 2) {
      uVar3 = 1;
    }
  }
  return uVar3;
}
