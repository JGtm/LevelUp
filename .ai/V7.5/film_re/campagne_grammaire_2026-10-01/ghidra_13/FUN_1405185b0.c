
char FUN_1405185b0(undefined8 param_1,int param_2,char param_3,char param_4,undefined1 param_5,
                  undefined1 *param_6,undefined1 *param_7,int *param_8)

{
  ulonglong uVar1;
  undefined1 auVar2 [16];
  char cVar3;
  char cVar4;
  bool bVar5;
  undefined4 uVar6;
  longlong lVar7;
  int iVar8;
  int iVar9;
  int iVar10;
  longlong lVar11;
  uint uVar12;
  longlong lVar13;
  char cVar14;
  bool bVar15;
  bool bVar16;
  float fVar17;
  float extraout_XMM0_Da;
  undefined8 uVar18;
  float fVar19;
  uint local_res10 [2];
  undefined1 local_78 [8];
  undefined8 local_70;
  
  cVar14 = '\0';
  bVar16 = false;
  lVar13 = (longlong)param_2;
  iVar9 = -1;
  FUN_1404e59a0();
  local_res10[0] = local_res10[0] & 0xffffff00;
  cVar3 = FUN_140518934(param_2);
  bVar15 = cVar3 != '\0';
  if (bVar15) {
    param_5 = 1;
  }
  uVar12 = 1 << ((byte)param_2 & 0x1f);
  if ((uVar12 & DAT_144de7be0) != 0) {
    param_5 = 1;
    bVar16 = (&DAT_144de55e5)[lVar13 * 0x130] != '\0';
  }
  fVar19 = DAT_143cd84ec;
  if (!bVar15) {
    if ((uVar12 & DAT_144de7be0) == 0) {
      if ((param_3 == '\0') && (param_4 == '\0')) {
LAB_14051892b:
        fVar19 = 0.0;
      }
    }
    else {
      lVar7 = lVar13 * 0x130;
      if ((param_3 == '\0') && (param_4 == '\0')) {
        if ((&DAT_144de55e5)[lVar7] == '\0') goto LAB_14051892b;
      }
      else if (((&DAT_144de55e5)[lVar7] == '\0') && ((&DAT_144de55e2)[lVar7] == '\0')) {
        if (param_3 == '\0') {
          fVar19 = DAT_143cd893c;
        }
        goto LAB_140518689;
      }
      fVar19 = *(float *)(&DAT_144de55ec + lVar7);
    }
  }
LAB_140518689:
  cVar3 = FUN_1404e5b9c();
  cVar4 = FUN_140517a78(param_2);
  if ((cVar4 != '\0') && (iVar8 = 0, cVar3 != '\0' || !bVar15)) {
    lVar7 = FUN_140518cb8();
    lVar11 = lVar13 * 0x3930;
    iVar10 = 0x100;
    if (*(int *)(lVar7 + 0x307c + lVar11) == 5) {
      iVar8 = (*(int *)(lVar7 + 0x2360 + lVar11) - *(int *)(lVar7 + 0xb44 + lVar11)) + 0x100;
    }
    else {
      iVar10 = 0;
    }
    if ((float)iVar8 / ((float)iVar10 * *(float *)(DAT_144de7d40 + 0xa4)) < DAT_143cd8374) {
      cVar3 = FUN_1404e5b9c();
      fVar17 = extraout_XMM0_Da;
      if (cVar3 != '\0') {
        fVar17 = DAT_143cd8648;
      }
      if ((fVar19 <= DAT_143cd8370) || (fVar17 <= fVar19)) {
        local_res10[0] = CONCAT31(local_res10[0]._1_3_,1);
        fVar19 = fVar17;
      }
    }
  }
  uVar1 = *(ulonglong *)(*(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x2a0) + 0x18);
  auVar2._8_8_ = 0;
  auVar2._0_8_ = uVar1;
  lVar7 = SUB168(ZEXT816(0x624dd2f1a9fbe77) * auVar2,8);
  iVar8 = FUN_140517a54(param_2);
  iVar10 = (int)((uVar1 - lVar7 >> 1) + lVar7 >> 9);
  iVar8 = iVar10 - iVar8;
  if (iVar8 < 0) {
    lVar7 = FUN_140518cb8();
    *(int *)(lVar7 + 0x70 + lVar13 * 0x3930) = iVar10;
    iVar8 = 0;
  }
  if (fVar19 <= DAT_143cd8370) {
    bVar5 = false;
    if (fVar19 != DAT_143cd8370) goto LAB_140518782;
  }
  else if (fVar19 <= DAT_143cd8394) {
    fVar17 = (float)roundf(DAT_143cd8348 / fVar19);
    bVar5 = (int)fVar17 + -1 <= iVar8;
  }
  else {
LAB_140518782:
    bVar5 = true;
  }
  if (((bVar15) && ((char)local_res10[0] == '\0')) || (bVar5)) {
    cVar14 = '\x01';
    iVar9 = -1;
    FUN_1404e59a0();
    iVar8 = -1;
    if ((DAT_144de7be0 & uVar12) != 0) {
      iVar10 = *(int *)(&DAT_144de55e8 + lVar13 * 0x130);
      iVar8 = -1;
      if ((-1 < iVar10) && ((&DAT_144de55e0)[lVar13 * 0x130] != '\0')) {
        local_res10[0] = 0x2400;
        FUN_140be219c(param_2,local_78);
        uVar18 = FUN_1405f4d74(local_70);
        iVar10 = iVar10 + (int)(float)uVar18 * -8;
        if (0 < iVar10) {
          local_res10[0] = iVar10 + 0x2400;
        }
        FUN_141ff6cc8(uVar18,param_2,local_res10);
        iVar8 = (int)local_res10[0] / 8;
        if (0x480 < iVar8) {
          iVar8 = 0x480;
        }
        iVar9 = iVar8;
        if (iVar8 < DAT_14498bf0c) {
          iVar9 = 0;
          cVar14 = '\0';
        }
      }
    }
    if (fVar19 == DAT_143cd84ec) {
      cVar14 = '\x01';
    }
    else if (cVar14 == '\0') goto LAB_140518857;
    uVar6 = FUN_1405f50b8();
    lVar7 = FUN_140518cb8();
    *(undefined4 *)(lVar7 + 0x70 + lVar13 * 0x3930) = uVar6;
    if (iVar9 < 0) {
      iVar9 = 0x480;
    }
    else if (iVar9 < DAT_14498bf0c) {
      iVar9 = DAT_14498bf0c;
    }
    if (((DAT_144de7be0 & uVar12) != 0) && (iVar8 < iVar9)) {
      *(int *)(&DAT_144de5684 + lVar13 * 0x130) =
           *(int *)(&DAT_144de5684 + lVar13 * 0x130) + (iVar9 - iVar8) * 8;
    }
    FUN_140518cb8();
    FUN_1404e59a0();
    if (8 < iVar9) {
      iVar9 = iVar9 - iVar9 % 8;
    }
    if (0x480 < iVar9) {
      iVar9 = 0x480;
    }
  }
LAB_140518857:
  *param_8 = iVar9;
  *param_6 = param_5;
  *param_7 = bVar16;
  return cVar14;
}

