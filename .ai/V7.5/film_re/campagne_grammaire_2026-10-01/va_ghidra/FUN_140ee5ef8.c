undefined4 FUN_140ee5ef8(longlong param_1)
{
  char cVar1;
  int iVar2;
  undefined1 uVar3;
  cVar1 = FUN_1406cf008();
  uVar3 = 0;
  if (cVar1 != '\0') {
    FUN_1406d676c(param_1);
  }
  iVar2 = *(int *)(param_1 + 0x18) * 8;
  if ((*(char *)(param_1 + 0x24) == '\0') && (*(int *)(param_1 + 0x2c) <= iVar2)) {
    uVar3 = 1;
  }
  return CONCAT31((int3)((uint)iVar2 >> 8),uVar3);
}
