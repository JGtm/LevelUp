
void FUN_1406d60f4(longlong param_1,undefined8 param_2,ulonglong *param_3,uint param_4)

{
  int iVar1;
  ulonglong uVar2;
  int iVar3;
  ulonglong uVar4;
  ulonglong *puVar5;
  ulonglong uVar6;
  uint uVar7;
  
  for (; 0x3f < param_4; param_4 = param_4 - 0x40) {
    uVar6 = *param_3;
    param_3 = param_3 + 1;
    uVar7 = *(uint *)(param_1 + 0x38);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x40;
    uVar4 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 | (uVar6 & 0xff0000000000) >> 0x18 |
            (uVar6 & 0xff00000000) >> 8 | (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 |
            (uVar6 & 0xff00) << 0x28 | uVar6 << 0x38;
    if (uVar7 == 0) {
      puVar5 = *(ulonglong **)(param_1 + 0x40);
      if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
        FUN_142266102();
        return;
      }
LAB_1406d6280:
      *puVar5 = uVar6;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
LAB_1406d6288:
      *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    }
    else {
      if ((int)(0x40 - uVar7) < 0x40) {
        uVar6 = *(ulonglong *)(param_1 + 0x30);
        *(ulonglong *)(param_1 + 0x30) = uVar4;
        if (uVar7 < 0x40) {
          uVar6 = uVar6 << ((byte)(0x40 - uVar7) & 0x3f) | uVar4 >> ((byte)uVar7 & 0x3f);
        }
        puVar5 = *(ulonglong **)(param_1 + 0x40);
        if (puVar5 + 1 <= *(ulonglong **)(param_1 + 0x10)) {
          uVar6 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 |
                  (uVar6 & 0xff0000000000) >> 0x18 | (uVar6 & 0xff00000000) >> 8 |
                  (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 | (uVar6 & 0xff00) << 0x28
                  | uVar6 << 0x38;
          goto LAB_1406d6280;
        }
        if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
          do {
            **(undefined1 **)(param_1 + 0x40) = (char)(uVar6 >> 0x38);
            *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
            uVar6 = uVar6 << 8;
          } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
        }
        goto LAB_1406d6288;
      }
      *(ulonglong *)(param_1 + 0x30) = uVar4;
      *(uint *)(param_1 + 0x38) = uVar7 + 0x40;
    }
  }
  if (param_4 == 0) {
    return;
  }
  uVar6 = 0;
  uVar7 = param_4;
  if ((int)param_4 < 8) {
LAB_1406d62a4:
    uVar6 = (ulonglong)(byte)((byte)*param_3 >> (8 - (byte)uVar7 & 0x3f)) |
            uVar6 << ((byte)uVar7 & 0x3f);
  }
  else {
    uVar4 = (ulonglong)(param_4 >> 3);
    uVar7 = param_4 + (param_4 >> 3) * -8;
    do {
      uVar2 = *param_3;
      param_3 = (ulonglong *)((longlong)param_3 + 1);
      uVar6 = uVar6 << 8 | (ulonglong)(byte)uVar2;
      uVar4 = uVar4 - 1;
    } while (uVar4 != 0);
    if (uVar7 != 0) goto LAB_1406d62a4;
  }
  iVar1 = *(int *)(param_1 + 0x38);
  if ((iVar1 == 0) && (param_4 == 0x40)) {
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x40;
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar6 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar6 = uVar6 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
      goto LAB_1406d6222;
    }
    *puVar5 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 | (uVar6 & 0xff0000000000) >> 0x18
              | (uVar6 & 0xff00000000) >> 8 | (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18
              | (uVar6 & 0xff00) << 0x28 | uVar6 << 0x38;
  }
  else {
    uVar4 = *(ulonglong *)(param_1 + 0x30);
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + param_4;
    iVar3 = 0x40 - iVar1;
    if ((int)param_4 <= iVar3) {
      *(ulonglong *)(param_1 + 0x30) =
           -(ulonglong)(param_4 < 0x40) & uVar4 << ((byte)param_4 & 0x3f) | uVar6;
      *(uint *)(param_1 + 0x38) = iVar1 + param_4;
      return;
    }
    param_4 = param_4 - iVar3;
    *(ulonglong *)(param_1 + 0x30) = uVar6;
    *(uint *)(param_1 + 0x38) = param_4;
    if (param_4 < 0x40) {
      uVar4 = uVar6 >> ((byte)param_4 & 0x3f) | uVar4 << ((byte)iVar3 & 0x3f);
    }
    puVar5 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar5 + 1) {
      if (puVar5 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar4 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar4 = uVar4 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
      goto LAB_1406d6222;
    }
    *puVar5 = uVar4 >> 0x38 | (uVar4 & 0xff000000000000) >> 0x28 | (uVar4 & 0xff0000000000) >> 0x18
              | (uVar4 & 0xff00000000) >> 8 | (uVar4 & 0xff000000) << 8 | (uVar4 & 0xff0000) << 0x18
              | (uVar4 & 0xff00) << 0x28 | uVar4 << 0x38;
  }
  *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
LAB_1406d6222:
  *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
  return;
}

