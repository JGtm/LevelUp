void FUN_1407cbc24(longlong param_1,undefined8 param_2,void *param_3,int param_4)
{
  int iVar1;
  ulonglong uVar2;
  ulonglong *puVar3;
  int iVar4;
  size_t _Size;
  byte bVar5;
  ulonglong uVar6;
  longlong lVar7;
  int iVar8;
  uint uVar9;
  _Size = (size_t)param_4;
  memset(param_3,0,_Size);
  iVar8 = 0;
  lVar7 = 0;
  if (0 < param_4) {
    do {
      iVar1 = *(int *)(param_1 + 0x38);
      bVar5 = (byte)((ulonglong)*(longlong *)(param_1 + 0x30) >> 0x38);
      if (0x40 - iVar1 < 8) {
        puVar3 = *(ulonglong **)(param_1 + 0x40);
        uVar6 = 0;
        iVar4 = 0;
        if (*(ulonglong **)(param_1 + 0x10) < puVar3 + 1) {
          if (puVar3 < *(ulonglong **)(param_1 + 0x10)) {
            do {
              uVar2 = *puVar3;
              iVar4 = iVar4 + 8;
              puVar3 = (ulonglong *)((longlong)puVar3 + 1);
              uVar6 = uVar6 << 8 | (ulonglong)(byte)uVar2;
              *(ulonglong **)(param_1 + 0x40) = puVar3;
            } while (puVar3 < *(ulonglong **)(param_1 + 0x10));
            uVar6 = uVar6 << (0x40U - (char)iVar4 & 0x3f);
          }
        }
        else {
          uVar6 = *puVar3;
          iVar4 = 0x40;
          uVar6 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 |
                  (uVar6 & 0xff0000000000) >> 0x18 | (uVar6 & 0xff00000000) >> 8 |
                  (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 | (uVar6 & 0xff00) << 0x28
                  | uVar6 << 0x38;
          *(ulonglong **)(param_1 + 0x40) = puVar3 + 1;
        }
        *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + iVar4;
        uVar9 = iVar1 - 0x38;
        *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 8;
        uVar2 = -(ulonglong)(uVar9 < 0x40) & uVar6 << ((byte)uVar9 & 0x3f);
        bVar5 = (byte)(uVar6 >> (0x40 - (byte)uVar9 & 0x3f)) | bVar5;
      }
      else {
        *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 8;
        uVar2 = *(longlong *)(param_1 + 0x30) << 8;
        uVar9 = iVar1 + 8;
      }
      *(ulonglong *)(param_1 + 0x30) = uVar2;
      *(uint *)(param_1 + 0x38) = uVar9;
      *(byte *)(lVar7 + (longlong)param_3) = bVar5;
      if (bVar5 == 0) break;
      iVar8 = iVar8 + 1;
      lVar7 = lVar7 + 1;
    } while (lVar7 < (longlong)_Size);
  }
  if (param_4 <= iVar8) {
    *(undefined1 *)((_Size - 1) + (longlong)param_3) = 0;
    *(undefined1 *)(param_1 + 0x24) = 1;
  }
  return;
}
