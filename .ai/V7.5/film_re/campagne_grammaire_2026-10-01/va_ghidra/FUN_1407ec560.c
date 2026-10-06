
void FUN_1407ec560(longlong param_1,uint *param_2)

{
  byte bVar1;
  ulonglong *puVar2;
  uint uVar3;
  longlong lVar4;
  int iVar5;
  ulonglong uVar6;
  ulonglong uVar7;
  uint *puVar8;
  int iVar9;
  int iVar10;
  uint uVar11;
  
  uVar11 = *param_2;
  iVar10 = 0x40 - *(int *)(param_1 + 0x38);
  uVar7 = *(ulonglong *)(param_1 + 0x30);
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
  if (iVar10 < 3) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 3 - iVar10;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar7 << ((byte)iVar10 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    uVar7 = uVar7 * 8 | (ulonglong)uVar11;
    iVar10 = *(int *)(param_1 + 0x38) + 3;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
    *(int *)(param_1 + 0x38) = iVar10;
  }
  uVar11 = param_2[1];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 3) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 3 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 3;
    uVar7 = uVar7 << 3 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = param_2[2];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 2;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 2) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 2 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 2;
    uVar7 = uVar7 << 2 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = (uint)(short)param_2[3];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 7;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 7) {
    uVar3 = 7 - iVar5;
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      uVar6 = uVar7;
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          uVar7 = uVar6 << 8;
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar6 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar6 = uVar7;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      uVar7 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 | (uVar7 & 0xff0000000000) >> 0x18
              | (uVar7 & 0xff00000000) >> 8 | (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18
              | (uVar7 & 0xff00) << 0x28 | uVar7 << 0x38;
      *puVar2 = uVar7;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
  }
  else {
    uVar7 = uVar7 << 7 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10 + 7;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  FUN_1406d60f4(param_1,uVar7,param_2 + 4,0x40);
  uVar11 = param_2[6];
  uVar7 = *(ulonglong *)(param_1 + 0x30);
  iVar10 = 0x40 - *(int *)(param_1 + 0x38);
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  if (iVar10 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar10;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar10 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = *(int *)(param_1 + 0x38) + 0x20;
    uVar7 = uVar7 << 0x20 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  bVar1 = (byte)param_2[8];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 3;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 3) {
    uVar11 = 3 - iVar5;
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)bVar1;
    *(uint *)(param_1 + 0x38) = uVar11;
    if (uVar11 < 0x40) {
      uVar7 = (ulonglong)(bVar1 >> ((byte)uVar11 & 0x3f)) | uVar7 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 3;
    uVar7 = uVar7 << 3 | (ulonglong)bVar1;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = param_2[0x3a544];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar7 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 0x20;
    uVar7 = uVar7 << 0x20 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = param_2[0x3a545];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar7 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 0x20;
    uVar7 = uVar7 << 0x20 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = param_2[0x3a546];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar5 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
  }
  else {
    *(int *)(param_1 + 0x38) = iVar10 + 0x20;
    *(ulonglong *)(param_1 + 0x30) = uVar7 << 0x20 | (ulonglong)uVar11;
  }
  FUN_140b857d8(param_1,param_2 + 0x3a547);
  FUN_1407ebe7c(param_1);
  uVar11 = param_2[0x3a9c5];
  iVar10 = 0x40 - *(int *)(param_1 + 0x38);
  uVar7 = *(ulonglong *)(param_1 + 0x30);
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  if (iVar10 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar10;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = uVar7 << ((byte)iVar10 & 0x3f) | (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f));
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = *(int *)(param_1 + 0x38) + 0x20;
    uVar7 = uVar7 << 0x20 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  bVar1 = (byte)param_2[0x3a9c6];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)bVar1;
    uVar11 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar11;
    if (uVar11 < 0x40) {
      uVar7 = (ulonglong)(bVar1 >> ((byte)uVar11 & 0x3f)) | uVar7 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
  }
  else {
    *(int *)(param_1 + 0x38) = iVar10 + 0x20;
    *(ulonglong *)(param_1 + 0x30) = uVar7 << 0x20 | (ulonglong)bVar1;
  }
  FUN_1410bc140(param_1,param_2 + 0x3a9cb);
  uVar7 = (ulonglong)*(byte *)((longlong)param_2 + 0xea719) & 1;
  if (*(uint *)(param_1 + 0x38) < 0x40) {
    uVar11 = *(uint *)(param_1 + 0x38) + 1;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(uint *)(param_1 + 0x38) = uVar11;
    *(ulonglong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2 | uVar7;
  }
  else {
    FUN_1406d6e28(param_1,uVar7,1);
    uVar11 = *(uint *)(param_1 + 0x38);
  }
  uVar7 = (ulonglong)*(byte *)((longlong)param_2 + 0xea71a) & 1;
  if (uVar11 < 0x40) {
    iVar10 = uVar11 + 1;
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2 | uVar7;
  }
  else {
    FUN_1406d6e28(param_1,uVar7,1);
    iVar10 = *(int *)(param_1 + 0x38);
  }
  uVar11 = param_2[0x3a9c7];
  iVar5 = *(int *)(param_1 + 0x2c) + 2;
  uVar7 = *(ulonglong *)(param_1 + 0x30);
  iVar9 = 0x40 - iVar10;
  *(int *)(param_1 + 0x2c) = iVar5;
  if (iVar9 < 2) {
    uVar3 = 2 - iVar9;
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar7 << ((byte)iVar9 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    uVar3 = *(uint *)(param_1 + 0x38);
    iVar5 = *(int *)(param_1 + 0x2c);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    uVar3 = iVar10 + 2;
    uVar7 = uVar7 << 2 | (ulonglong)uVar11;
    *(uint *)(param_1 + 0x38) = uVar3;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar6 = (ulonglong)(byte)param_2[0x3a9c8] & 1;
  if (uVar3 < 0x40) {
    iVar10 = uVar3 + 1;
    *(int *)(param_1 + 0x38) = iVar10;
    uVar6 = uVar7 * 2 | uVar6;
    *(int *)(param_1 + 0x2c) = iVar5 + 1;
    *(ulonglong *)(param_1 + 0x30) = uVar6;
  }
  else {
    FUN_1406d6e28(param_1,uVar6,1);
    iVar10 = *(int *)(param_1 + 0x38);
    uVar6 = *(ulonglong *)(param_1 + 0x30);
  }
  uVar11 = param_2[0x3a9c9];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar6 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar6 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar6 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar6 = uVar6 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar6 >> 0x38 | (uVar6 & 0xff000000000000) >> 0x28 |
                (uVar6 & 0xff0000000000) >> 0x18 | (uVar6 & 0xff00000000) >> 8 |
                (uVar6 & 0xff000000) << 8 | (uVar6 & 0xff0000) << 0x18 | (uVar6 & 0xff00) << 0x28 |
                uVar6 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    iVar10 = *(int *)(param_1 + 0x38);
    uVar7 = *(ulonglong *)(param_1 + 0x30);
  }
  else {
    iVar10 = iVar10 + 0x20;
    uVar7 = uVar6 << 0x20 | (ulonglong)uVar11;
    *(int *)(param_1 + 0x38) = iVar10;
    *(ulonglong *)(param_1 + 0x30) = uVar7;
  }
  uVar11 = param_2[0x3a9ca];
  *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 0x20;
  iVar5 = 0x40 - iVar10;
  if (iVar5 < 0x20) {
    *(ulonglong *)(param_1 + 0x30) = (ulonglong)uVar11;
    uVar3 = 0x20 - iVar5;
    *(uint *)(param_1 + 0x38) = uVar3;
    if (uVar3 < 0x40) {
      uVar7 = (ulonglong)(uVar11 >> ((byte)uVar3 & 0x3f)) | uVar7 << ((byte)iVar5 & 0x3f);
    }
    puVar2 = *(ulonglong **)(param_1 + 0x40);
    if (*(ulonglong **)(param_1 + 0x10) < puVar2 + 1) {
      if (puVar2 < *(ulonglong **)(param_1 + 0x10)) {
        do {
          **(undefined1 **)(param_1 + 0x40) = (char)(uVar7 >> 0x38);
          *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 1;
          uVar7 = uVar7 << 8;
        } while (*(ulonglong *)(param_1 + 0x40) < *(ulonglong *)(param_1 + 0x10));
      }
    }
    else {
      *puVar2 = uVar7 >> 0x38 | (uVar7 & 0xff000000000000) >> 0x28 |
                (uVar7 & 0xff0000000000) >> 0x18 | (uVar7 & 0xff00000000) >> 8 |
                (uVar7 & 0xff000000) << 8 | (uVar7 & 0xff0000) << 0x18 | (uVar7 & 0xff00) << 0x28 |
                uVar7 << 0x38;
      *(longlong *)(param_1 + 0x40) = *(longlong *)(param_1 + 0x40) + 8;
    }
    *(int *)(param_1 + 0x28) = *(int *)(param_1 + 0x28) + 0x40;
    uVar3 = *(uint *)(param_1 + 0x38);
  }
  else {
    uVar3 = iVar10 + 0x20;
    *(uint *)(param_1 + 0x38) = uVar3;
    *(ulonglong *)(param_1 + 0x30) = uVar7 << 0x20 | (ulonglong)uVar11;
  }
  uVar7 = (ulonglong)(byte)param_2[0x3aa03] & 1;
  if (uVar3 < 0x40) {
    *(int *)(param_1 + 0x2c) = *(int *)(param_1 + 0x2c) + 1;
    *(uint *)(param_1 + 0x38) = uVar3 + 1;
    *(ulonglong *)(param_1 + 0x30) = *(longlong *)(param_1 + 0x30) * 2 | uVar7;
  }
  else {
    FUN_1406d6e28(param_1,uVar7,1);
  }
  puVar8 = param_2 + 10;
  lVar4 = FUN_1406aed80(puVar8);
  FUN_14051a4b8(*(undefined4 *)(lVar4 + 4));
  FUN_1406d49c4(param_1);
  lVar4 = FUN_1406aed80(puVar8);
  iVar10 = FUN_14051a4b8(*(undefined4 *)(lVar4 + 4));
  if (iVar10 != 0) {
    FUN_140b85504(puVar8,param_1);
  }
  FUN_1407ebe7c(param_1);
  FUN_1407ebe7c(param_1);
  FUN_1406d60f4(param_1);
  FUN_1406d60f4(param_1);
  for (puVar8 = param_2 + 0x3aabc; puVar8 != param_2 + 0x44d3c; puVar8 = puVar8 + 0x514) {
    FUN_1407ecb08(puVar8,param_1);
  }
  return;
}

