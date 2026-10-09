
int FUN_1406d310c(uint param_1)

{
  int iVar1;
  int iVar2;
  
  iVar1 = 0x1f;
  if (param_1 != 0) {
    for (; param_1 >> iVar1 == 0; iVar1 = iVar1 + -1) {
    }
  }
  if (param_1 == 0) {
    iVar1 = -1;
  }
  iVar2 = 0;
  if (iVar1 != -1) {
    iVar2 = (uint)((param_1 & (1 << ((byte)iVar1 & 0x3f)) - 1U) != 0) + iVar1;
  }
  return iVar2;
}

