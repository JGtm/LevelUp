
void FUN_1407eb6a8(longlong param_1,undefined8 param_2,uint param_3)

{
  ulonglong *puVar1;
  int iVar2;
  int iVar3;
  uint uVar4;
  ulonglong uVar5;
  
  FUN_1406d49c4(param_1,param_2,param_3 == 0xffffffff);
  iVar2 = DAT_144632be0;
  if (param_3 != 0xffffffff) {
    uVar5 = *(ulonglong *)(param_1 + 0x30);
    iVar3 = 0x40 - *(int *)(param_1 + 0x38);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + DAT_144632be0;
    if (iVar3 < iVar2) {
      uVar4 = iVar2 - iVar3;
      *(ulonglong *)(param_1 + 0x30) = (ulonglong)param_3;
      *(uint *)(param_1 + 0x38) = uVar4;
      if (uVar4 < 0x40) {
        uVar5 = (ulonglong)(param_3 >> ((byte)uVar4 & 0x3f)) | uVar5 << ((byte)iVar3 & 0x3f);
      }
      puVar1 = *(ulonglong **)(param_1 + 0x40);
      if (*(ulonglong **)(param_1 + 0x10) < puVar1 + 1) {
        if (puVar1 < *(ulonglong **)(param_1 + 0x10)) {
          do {
            **(undefined1 **)(param_1 + 0x40) = (char)(uVar5 >> 0x38);
            *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
            uVar5 = uVar5 << 8;
          } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
        }
      }
      else {
        *puVar1 = uVar5 >> 0x38 | (uVar5 & 0xff000000000000) >> 0x28 |
                  (uVar5 & 0xff0000000000) >> 0x18 | (uVar5 & 0xff00000000) >> 8 |
                  (uVar5 & 0xff000000) << 8 | (uVar5 & 0xff0000) << 0x18 | (uVar5 & 0xff00) << 0x28
                  | uVar5 << 0x38;
        *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
      }
      *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    }
    else {
      *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + iVar2;
      *(ulonglong *)(param_1 + 0x30) = uVar5 << ((byte)iVar2 & 0x3f) | (ulonglong)param_3;
    }
  }
  FUN_140ffc6b0(param_1);
  return;
}

