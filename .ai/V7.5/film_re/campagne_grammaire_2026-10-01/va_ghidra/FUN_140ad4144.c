
char FUN_140ad4144(uint *param_1)

{
  ulonglong uVar1;
  char cVar2;
  char cVar3;
  int iVar4;
  longlong lVar5;
  char *pcVar6;
  longlong lVar7;
  ulonglong uVar8;
  int iVar9;
  ulonglong uVar10;
  byte bVar11;
  ulonglong uVar12;
  ulonglong uVar13;
  uint uVar14;
  uint uVar15;
  ulonglong uVar16;
  undefined8 uVar17;
  longlong lVar18;
  ulonglong uVar19;
  
  uVar14 = *param_1;
  uVar13 = (ulonglong)uVar14;
  if (4 < uVar14) {
    return '\0';
  }
  if (3 < param_1[1]) {
    return '\0';
  }
  if (4 < (byte)param_1[8]) {
    return '\0';
  }
  uVar8 = 0;
  lVar18 = -1;
  do {
    lVar5 = lVar18 + 1;
    lVar7 = lVar18 + 0xea565;
    lVar18 = lVar5;
  } while (*(char *)((longlong)param_1 + lVar7) != '\0');
  if (lVar5 == 0) {
    return '\0';
  }
  if (2 < param_1[0x3a9c7]) {
    return '\0';
  }
  iVar4 = 0;
  if (((char)param_1[0x3a9c8] != '\0') && (param_1[0x3a9c7] != 0)) {
    return '\0';
  }
  uVar1 = *(ulonglong *)(param_1 + 0x3aa86);
  uVar10 = uVar8;
  if ((uVar1 & 0xfffffffe00000000) == 0) {
    while (uVar14 = (uint)uVar10, (int)uVar14 < 0x21) {
      uVar15 = uVar14;
      if ((uVar1 >> (uVar10 & 0x3f) & 1) != 0) {
        while( true ) {
          uVar15 = uVar15 + 1;
          uVar14 = (uint)uVar10;
          if (0x20 < (int)uVar15) break;
          if (((uVar1 >> ((ulonglong)uVar15 & 0x3f) & 1) != 0) &&
             (cVar3 = FUN_140496898((longlong)param_1 + ((longlong)(int)uVar14 + 0x271b0) * 6,
                                    (longlong)param_1 + ((longlong)(int)uVar15 + 0x271b0) * 6),
             cVar3 != '\0')) {
            cVar3 = FUN_142393588();
            return cVar3;
          }
        }
      }
      uVar10 = (ulonglong)(uVar14 + 1);
    }
    lVar18 = 1;
    uVar12 = 1;
    uVar10 = uVar8;
    uVar16 = uVar8;
    uVar19 = uVar8;
    do {
      cVar3 = (char)uVar12;
      iVar4 = (int)uVar19;
      uVar14 = (uint)uVar13;
      iVar9 = (int)uVar10;
      if (0x1f < iVar9) break;
      if (*(char *)(uVar16 + 0xeaaf0 + (longlong)param_1) != '\0') {
        uVar19 = (ulonglong)(iVar4 + 1);
        if ((cVar3 == '\0') || (*(longlong *)(uVar16 + 0xeab00 + (longlong)param_1) != 0)) {
          if (cVar3 != '\0') {
            pcVar6 = (char *)((longlong)param_1 + uVar16 + 0xebf40);
            lVar7 = lVar18;
            do {
              if (0x1f < lVar7) break;
              if ((*pcVar6 != '\0') &&
                 (*(longlong *)(uVar16 + 0xeab00 + (longlong)param_1) ==
                  *(longlong *)(pcVar6 + 0x10))) {
                uVar12 = uVar8;
              }
              lVar7 = lVar7 + 1;
              pcVar6 = pcVar6 + 0x1450;
            } while ((char)uVar12 != '\0');
          }
        }
        else {
          uVar12 = 0;
        }
        bVar11 = (byte)uVar12;
        if (*(char *)(uVar16 + 0xeaaf1 + (longlong)param_1) == '\0') {
          if (bVar11 != 0) {
            uVar12 = (ulonglong)(bVar11 & -(*(byte *)(uVar16 + 0xeaaf8 + (longlong)param_1) < 4));
          }
          if (param_1[0x3a9c7] == 0) {
            uVar17 = 0xffffffff;
            uVar10 = uVar8;
            do {
              if (((uVar1 >> (uVar10 & 0x3f) & 1) != 0) &&
                 (cVar3 = FUN_140496898((longlong)iVar9 * 0x1450 + 0xeaaf9 + (longlong)param_1,
                                        (longlong)param_1 + (uVar10 + 0x271b0) * 6), cVar3 != '\0'))
              {
                iVar4 = (int)uVar10;
                break;
              }
              iVar4 = (int)uVar17;
              uVar14 = (int)uVar10 + 1;
              uVar10 = (ulonglong)uVar14;
            } while ((int)uVar14 < 0x21);
            if (((char)uVar12 != '\0') && (iVar4 == -1)) {
              uVar12 = uVar8;
            }
          }
        }
        else if ((bVar11 != 0) &&
                (cVar3 = FUN_140496898((longlong)iVar9 * 0x1450 + 0xeaaf9 + (longlong)param_1,
                                       &DAT_143b8c570), cVar3 == '\0')) {
          uVar12 = 0;
        }
      }
      iVar4 = (int)uVar19;
      uVar14 = (uint)uVar13;
      uVar10 = (ulonglong)(iVar9 + 1);
      lVar18 = lVar18 + 1;
      uVar16 = uVar16 + 0x1450;
      cVar3 = '\0';
    } while ((char)uVar12 != '\0');
  }
  else {
    cVar3 = '\0';
  }
  if (uVar14 == 1) {
    lVar18 = FUN_1406aed80(param_1 + 10);
    if (cVar3 == '\0') {
      return '\0';
    }
    if (3 < *(uint *)(lVar18 + 0xe21a0)) {
      return '\0';
    }
    if (3 < iVar4 - 1U) {
      return '\0';
    }
  }
  else if (uVar14 == 2) {
    if (cVar3 == '\0') {
      return '\0';
    }
    if (0x1f < iVar4 - 1U) {
      return '\0';
    }
    lVar18 = FUN_1406aed80(param_1 + 10);
    iVar4 = FUN_14051a4b8(*(undefined4 *)(lVar18 + 4));
    if (iVar4 == 1) {
      cVar3 = FUN_1423935e3();
      return cVar3;
    }
  }
  else {
    if (uVar14 != 3) {
      cVar2 = cVar3;
      if (uVar14 == 4) {
        return cVar3;
      }
      goto LAB_140ad43a1;
    }
    if (DAT_144eadd48 == '\0') {
      if (cVar3 == '\0') {
        return '\0';
      }
    }
    else {
      if (cVar3 == '\0') {
        return '\0';
      }
      if (iVar4 == 0) {
        return '\0';
      }
    }
  }
  cVar2 = *(char *)((longlong)param_1 + 0xeaae6);
LAB_140ad43a1:
  if (cVar2 == '\0') {
    return '\0';
  }
  return cVar3;
}

