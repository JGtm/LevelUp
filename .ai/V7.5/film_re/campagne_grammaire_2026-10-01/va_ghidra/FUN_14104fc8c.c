
void FUN_14104fc8c(undefined8 param_1,undefined8 param_2,short *param_3,longlong param_4)

{
  ulonglong *puVar1;
  int iVar2;
  uint uVar3;
  ulonglong uVar4;
  uint uVar5;
  
  if (*(int *)(DAT_144c1cfa8 + 4) == 2) {
    FUN_1424e43f4(param_4,param_2,*param_3);
    if (*param_3 == 0) {
      FUN_1424cab44(param_4);
    }
    else if (*param_3 == 1) {
      FUN_1407edb6c();
      FUN_1406d270c(param_4);
    }
    if (*(uint *)(param_3 + 8) == 0xffffffff) {
      uVar5 = 0xffffffff;
    }
    else {
      uVar5 = *(uint *)(param_3 + 8) >> 1 & 0x7fff;
    }
    FUN_1406d49c4(param_4);
    if (uVar5 != 0xffffffff) {
      uVar4 = *(ulonglong *)(param_4 + 0x30);
      *(int *)(param_4 + 0x2c) = *(int *)(param_4 + 0x2c) + 5;
      iVar2 = 0x40 - *(int *)(param_4 + 0x38);
      if (iVar2 < 5) {
        uVar3 = 5 - iVar2;
        *(ulonglong *)(param_4 + 0x30) = (ulonglong)uVar5;
        *(uint *)(param_4 + 0x38) = uVar3;
        if (uVar3 < 0x40) {
          uVar4 = (ulonglong)(uVar5 >> ((byte)uVar3 & 0x3f)) | uVar4 << ((byte)iVar2 & 0x3f);
        }
        puVar1 = *(ulonglong **)(param_4 + 0x40);
        if (*(ulonglong **)(param_4 + 0x10) < puVar1 + 1) {
          if (puVar1 < *(ulonglong **)(param_4 + 0x10)) {
            do {
              **(undefined1 **)(param_4 + 0x40) = (char)(uVar4 >> 0x38);
              *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 1;
              uVar4 = uVar4 << 8;
            } while (*(ulonglong *)(param_4 + 0x40) < *(ulonglong *)(param_4 + 0x10));
          }
        }
        else {
          *puVar1 = uVar4 >> 0x38 | (uVar4 & 0xff000000000000) >> 0x28 |
                    (uVar4 & 0xff0000000000) >> 0x18 | (uVar4 & 0xff00000000) >> 8 |
                    (uVar4 & 0xff000000) << 8 | (uVar4 & 0xff0000) << 0x18 |
                    (uVar4 & 0xff00) << 0x28 | uVar4 << 0x38;
          *(longlong *)(param_4 + 0x40) = *(longlong *)(param_4 + 0x40) + 8;
        }
        *(int *)(param_4 + 0x28) = *(int *)(param_4 + 0x28) + 0x40;
      }
      else {
        *(int *)(param_4 + 0x38) = *(int *)(param_4 + 0x38) + 5;
        *(ulonglong *)(param_4 + 0x30) = uVar4 << 5 | (ulonglong)uVar5;
      }
    }
  }
  return;
}

