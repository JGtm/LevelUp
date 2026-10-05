
void FUN_141fd07f8(ushort *param_1,longlong param_2)

{
  ushort uVar1;
  ulonglong *puVar2;
  int iVar3;
  uint uVar4;
  ulonglong uVar5;
  
  uVar1 = *param_1;
  uVar5 = *(ulonglong *)(param_2 + 0x30);
  iVar3 = 0x40 - *(int *)(param_2 + 0x38);
  *(int *)(param_2 + 0x2c) = *(int *)(param_2 + 0x2c) + 0xf;
  if (0xe < iVar3) {
    *(int *)(param_2 + 0x38) = *(int *)(param_2 + 0x38) + 0xf;
    *(ulonglong *)(param_2 + 0x30) = uVar5 << 0xf | (ulonglong)uVar1;
    return;
  }
  uVar4 = 0xf - iVar3;
  *(ulonglong *)(param_2 + 0x30) = (ulonglong)uVar1;
  *(uint *)(param_2 + 0x38) = uVar4;
  if (uVar4 < 0x40) {
    uVar5 = (ulonglong)(uVar1 >> ((byte)uVar4 & 0x3f)) | uVar5 << ((byte)iVar3 & 0x3f);
  }
  puVar2 = *(ulonglong **)(param_2 + 0x40);
  if (*(ulonglong **)(param_2 + 0x10) < puVar2 + 1) {
    if (puVar2 < *(ulonglong **)(param_2 + 0x10)) {
      do {
        **(undefined1 **)(param_2 + 0x40) = (char)(uVar5 >> 0x38);
        *(longlong *)(param_2 + 0x40) = *(longlong *)(param_2 + 0x40) + 1;
        uVar5 = uVar5 << 8;
      } while (*(ulonglong *)(param_2 + 0x40) < *(ulonglong *)(param_2 + 0x10));
    }
  }
  else {
    *puVar2 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 | (uVar5 & 0xff0000000000) >> 0x18
              | (uVar5 & 0xff00000000) >> 8 | (uVar5 & 0xff000000) << 8 | (uVar5 & 0xff0000) << 0x18
              | (uVar5 & 0xff00) << 0x28 | uVar5 << 0x38;
    *(longlong *)(param_2 + 0x40) = *(longlong *)(param_2 + 0x40) + 8;
  }
  *(int *)(param_2 + 0x28) = *(int *)(param_2 + 0x28) + 0x40;
  return;
}

