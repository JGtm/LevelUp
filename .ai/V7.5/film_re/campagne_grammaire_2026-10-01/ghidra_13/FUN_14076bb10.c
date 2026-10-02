
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_14076bb10(void)

{
  undefined8 uVar1;
  undefined4 uVar2;
  char cVar3;
  double *pdVar4;
  longlong lVar5;
  int *piVar6;
  longlong lVar7;
  ulonglong uVar8;
  int iVar9;
  ulonglong uVar10;
  undefined1 local_res10 [8];
  undefined8 local_res18;
  
  lVar5 = DAT_144e61d78;
  if (((DAT_144e61d63 != '\0') || (*(int *)(DAT_144e61d78 + 0x10) == 0)) ||
     (cVar3 = FUN_1404f1b74(), cVar3 == '\0')) goto LAB_14076bc09;
  lVar7 = *(longlong *)ThreadLocalStoragePointer;
  uVar2 = *(undefined4 *)(lVar7 + 0x710);
  *(undefined4 *)(lVar7 + 0x710) = 1;
  pdVar4 = (double *)FUN_1404efbd4(local_res10);
  uVar1 = *(undefined8 *)(lVar7 + 0x7f8);
  *(double *)(lVar7 + 0x7f8) =
       (double)(float)(*(uint *)((ulonglong)*(byte *)(lVar7 + 0x810) * 0x20 + 0x18 +
                                *(longlong *)(lVar7 + 0x6c0)) ^ DAT_143cd84d0) + *pdVar4;
  local_res18 = uVar1;
  FUN_14076bf64(lVar5);
  if ((*(int *)(DAT_144e61d78 + 0x10) - 3U < 3) && (*(int *)(DAT_144e61d78 + 0x20) == 4)) {
    if ((DAT_1451789b8 == '\0') ||
       ((cVar3 = FUN_1406aed00(), cVar3 == '\0' || (DAT_145121140 == '\x01')))) {
LAB_14076bbf2:
      if ((DAT_145178a48 != '\0') && (cVar3 = FUN_1406aed00(), cVar3 != '\0')) {
        lVar5 = FUN_1404f1614();
        lVar5 = FUN_1406aed80(lVar5 + 0x28);
        if (*(char *)(lVar5 + 0x240) != '\0') goto LAB_14230c6a7;
      }
    }
    else {
      lVar5 = FUN_1404f1614();
      lVar5 = FUN_1406aed80(lVar5 + 0x28);
      if (*(char *)(lVar5 + 0x238) == '\0') goto LAB_14076bbf2;
LAB_14230c6a7:
      cVar3 = FUN_1404f293c();
      if (cVar3 == '\0') {
        if ((((DAT_144de3ea0 != '\0') && (lVar5 = FUN_14049739c(DAT_144de3ea4), lVar5 != 0)) &&
            (iVar9 = *(int *)(lVar5 + 0x2e0), iVar9 != -1)) &&
           (piVar6 = (int *)FUN_14059c630(local_res10,0), *piVar6 == -1)) {
          FUN_140ad3344(DAT_144de3ea4,iVar9);
        }
      }
      else {
        if (DAT_144de3fe5 == '\0') {
          cVar3 = FUN_1406aa4a4();
          if (cVar3 == '\0') {
            FUN_141fda950(&DAT_144de3ea0);
          }
        }
        else {
          FUN_141fdaa54(&DAT_144de3ea0);
        }
        if ((DAT_144de4000 != '\0') && (lVar5 = _Xtime_get_ticks(), DAT_144de3ff8 < lVar5)) {
          DAT_144de3ff8 = 0;
          DAT_144de4000 = '\0';
          FUN_141fda514(&DAT_144de3ea0);
        }
      }
    }
  }
  *(undefined8 *)(lVar7 + 0x7f8) = uVar1;
  *(undefined4 *)(lVar7 + 0x710) = uVar2;
LAB_14076bc09:
  FUN_1424cbe18();
  FUN_14076be40();
  uVar8 = 0;
  uVar10 = uVar8;
  do {
    iVar9 = (int)uVar10;
    if (*(longlong *)(uVar8 + 0x480 + DAT_145178b58) == 0) {
      *(longlong *)(uVar8 + 0x480 + DAT_145178b58) = (longlong)iVar9 * 0xd8 + _DAT_145178b60;
    }
    lVar7 = DAT_145178b58;
    lVar5 = *(longlong *)(uVar8 + 0x480 + DAT_145178b58);
    uVar10 = (ulonglong)(iVar9 + 1U);
    uVar8 = uVar8 + 0x4c8;
    *(undefined4 *)(lVar5 + 0x18) = 0x480;
    lVar7 = (longlong)iVar9 * 0x4c8 + lVar7;
    *(undefined4 *)(lVar5 + 0x20) = 0;
    *(longlong *)(lVar5 + 8) = lVar7;
    *(undefined8 *)(lVar5 + 0x28) = 0;
    *(undefined4 *)(lVar5 + 0x48) = 0;
    *(undefined1 *)(lVar5 + 0x24) = 0;
    *(longlong *)(lVar5 + 0x10) = lVar7 + 0x480;
    *(longlong *)(lVar5 + 0x40) = lVar7;
    *(undefined8 *)(lVar5 + 0x30) = 0;
    *(undefined4 *)(lVar5 + 0x38) = 0;
  } while (iVar9 + 1U < 0x20);
  FUN_14076bea8(&DAT_144de3ea0);
  return;
}

