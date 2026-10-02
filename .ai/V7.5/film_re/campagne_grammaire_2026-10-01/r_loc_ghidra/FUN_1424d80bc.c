
void FUN_1424d80bc(longlong param_1,int param_2,undefined4 param_3,undefined8 param_4,
                  longlong param_5)

{
  longlong *plVar1;
  ulonglong *puVar2;
  char cVar3;
  uint uVar4;
  int iVar5;
  ulonglong uVar6;
  
  plVar1 = *(longlong **)
            (*(longlong *)(*(longlong *)(param_1 + 8) + 0x18) + 0x210 + (longlong)param_2 * 8);
  (**(code **)(*plVar1 + 0x60))(plVar1,param_3,param_4,param_5,1);
  cVar3 = FUN_14076cea8();
  if (cVar3 != '\0') {
    if (DAT_1450e2520 == '\0') {
      FUN_1406d49c4(param_5);
      uVar6 = *(ulonglong *)(param_5 + 0x30);
      *(int *)(param_5 + 0x2c) = *(int *)(param_5 + 0x2c) + 0x20;
      iVar5 = 0x40 - *(int *)(param_5 + 0x38);
      if (iVar5 < 0x20) {
        uVar4 = 0x20 - iVar5;
        *(uint *)(param_5 + 0x38) = uVar4;
        *(undefined8 *)(param_5 + 0x30) = 0xbcddcba;
        if (uVar4 < 0x40) {
          uVar6 = uVar6 << ((byte)iVar5 & 0x3f) | 0xbcddcbaUL >> ((byte)uVar4 & 0x3f);
        }
        puVar2 = *(ulonglong **)(param_5 + 0x40);
        if (*(ulonglong **)(param_5 + 0x10) < puVar2 + 1) {
          if (puVar2 < *(ulonglong **)(param_5 + 0x10)) {
            do {
              **(undefined1 **)(param_5 + 0x40) = (char)(uVar6 >> 0x38);
              *(longlong *)(param_5 + 0x40) = *(longlong *)(param_5 + 0x40) + 1;
              uVar6 = uVar6 << 8;
            } while (*(ulonglong *)(param_5 + 0x40) < *(ulonglong *)(param_5 + 0x10));
          }
        }
        else {
          *puVar2 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 |
                    (uVar6 & 0xff0000000000) >> 0x18 | (uVar6 & 0xff00000000) >> 8 |
                    (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 |
                    (uVar6 & 0xff00) << 0x28 | uVar6 << 0x38;
          *(longlong *)(param_5 + 0x40) = *(longlong *)(param_5 + 0x40) + 8;
        }
        *(int *)(param_5 + 0x28) = *(int *)(param_5 + 0x28) + 0x40;
      }
      else {
        *(int *)(param_5 + 0x38) = *(int *)(param_5 + 0x38) + 0x20;
        *(ulonglong *)(param_5 + 0x30) = uVar6 << 0x20 | 0xbcddcba;
      }
    }
    else {
      FUN_1406d49c4(param_5);
    }
  }
  return;
}

