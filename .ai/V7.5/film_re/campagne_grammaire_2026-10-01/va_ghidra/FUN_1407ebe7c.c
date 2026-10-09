void FUN_1407ebe7c(longlong param_1,undefined8 param_2,char *param_3,int param_4)
{
  byte bVar1;
  ulonglong *puVar2;
  uint uVar3;
  ulonglong uVar4;
  size_t _MaxCount;
  int iVar5;
  longlong lVar6;
  _MaxCount = (size_t)param_4;
  strnlen(param_3,_MaxCount);
  if (0 < (longlong)_MaxCount) {
    lVar6 = 0;
    do {
      bVar1 = param_3[lVar6];
      uVar4 = *(ulonglong *)(param_1 + 0x30);
      iVar5 = 0x40 - *(int *)(param_1 + 0x38);
      *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 8;
      if (iVar5 < 8) {
        *(ulonglong *)(param_1 + 0x30) = (ulonglong)bVar1;
        uVar3 = 8 - iVar5;
        *(uint *)(param_1 + 0x38) = uVar3;
        if (uVar3 < 0x40) {
          uVar4 = (ulonglong)(bVar1 >> ((byte)uVar3 & 0x3f)) | uVar4 << ((byte)iVar5 & 0x3f);
        }
        puVar2 = *(ulonglong **)(param_1 + 0x40);
        if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
          if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
            do {
              **(undefined1 **)(param_1 + 0x40) = (char)(uVar4 >> 0x38);
              *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
              uVar4 = uVar4 << 8;
            } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
          }
        }
        else {
          *puVar2 = uVar4 >> 0x38 | (uVar4 & 0xff000000000000) >> 0x28 |
                    (uVar4 & 0xff0000000000) >> 0x18 | (uVar4 & 0xff00000000) >> 8 |
                    (uVar4 & 0xff000000) << 8 | (uVar4 & 0xff0000) << 0x18 |
                    (uVar4 & 0xff00) << 0x28 | uVar4 << 0x38;
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
        }
        *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
      }
      else {
        *(int *)(param_1 + 0x38) = *(int *)(param_1 + 0x38) + 8;
        *(ulonglong *)(param_1 + 0x30) = uVar4 << 8 | (ulonglong)bVar1;
      }
    } while ((bVar1 != 0) && (lVar6 = lVar6 + 1, lVar6 < (longlong)_MaxCount));
  }
  return;
}
