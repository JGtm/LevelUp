
void FUN_14076c008(longlong param_1,longlong param_2)

{
  bool bVar1;
  char cVar2;
  int iVar3;
  uint uVar4;
  int iVar5;
  undefined8 uVar6;
  longlong *plVar7;
  longlong lVar8;
  char cVar9;
  longlong *plVar10;
  
  if (param_2 == 0) {
    param_2 = *(longlong *)(param_1 + 0x188);
  }
  iVar3 = FUN_14076c748(*(undefined8 *)(param_1 + 0x178));
  lVar8 = (longlong)iVar3;
  *(undefined4 *)(&DAT_14498cd40 + lVar8 * 4) = 0x7f7fffff;
  *(undefined4 *)(&DAT_14498cdc4 + lVar8 * 4) = 0xff7fffff;
  *(int *)(&DAT_14498cf50 + lVar8 * 4) = *(int *)(&DAT_14498cf50 + lVar8 * 4) + 1;
  uVar4 = FUN_140514010(*(undefined4 *)(param_1 + 0xc));
  if (uVar4 < 0x21) {
    cVar9 = (&DAT_144de4348)[(int)uVar4];
    if (cVar9 != '\0') {
      cVar2 = FUN_142f30380(param_1);
      if (cVar2 != '\0') {
        bVar1 = true;
        goto LAB_14076c095;
      }
    }
  }
  else {
    cVar9 = '\0';
  }
  bVar1 = false;
LAB_14076c095:
  *(undefined4 *)(param_1 + 400) = 0;
  *(undefined4 *)(param_1 + 0x194) = 0;
  if (cVar9 == '\0') {
    uVar6 = *(undefined8 *)(param_1 + 0x178);
  }
  else {
    uVar6 = *(undefined8 *)(param_1 + 0x180);
  }
  uVar6 = FUN_14076bf98(uVar6);
  if (bVar1) {
    lVar8 = *(longlong *)(DAT_144e61d78 + 0xd8 + (longlong)*(int *)(param_1 + 0xc) * 8);
    cVar2 = FUN_142f23290(&DAT_144de3ea0,uVar4,lVar8);
    if (cVar2 == '\0') {
      FUN_141fda824(&DAT_144de3ea0,lVar8);
    }
    else {
      FUN_142f2ad88(*(undefined8 *)(*(longlong *)(*(longlong *)(lVar8 + 0x18) + 8) + 0x42c8),
                    *(undefined4 *)(param_1 + 0xc));
    }
  }
  iVar3 = DAT_1447061ac;
  plVar10 = (longlong *)(param_1 + 0x128);
  lVar8 = 3;
  do {
    plVar7 = plVar10 + 3;
    if (cVar9 == '\0') {
      plVar7 = plVar10;
    }
    plVar7 = (longlong *)*plVar7;
    if (plVar7 != (longlong *)0x0) {
      cVar2 = (**(code **)(*plVar7 + 8))(plVar7);
      if (cVar2 != '\0') {
        iVar5 = (**(code **)(*plVar7 + 0x10))
                          (plVar7,uVar6,iVar3 - *(int *)(param_1 + 400),
                           param_2 + (longlong)*(int *)(param_1 + 400) * 4);
        *(int *)(param_1 + 400) = *(int *)(param_1 + 400) + iVar5;
      }
    }
    plVar10 = plVar10 + 1;
    lVar8 = lVar8 + -1;
  } while (lVar8 != 0);
  FUN_14076c304(param_2,*(undefined4 *)(param_1 + 400));
  iVar3 = *(int *)(param_1 + 400);
  if (0x100 < iVar3) {
    iVar3 = 0x100;
  }
  *(int *)(param_1 + 400) = iVar3;
  iVar5 = DAT_14498bdd8;
  if (DAT_14498bdd8 != 0) {
    lVar8 = FUN_140514044();
    if ((lVar8 != 0) && (0 < *(int *)(lVar8 + 0xd0))) {
      iVar5 = iVar5 / *(int *)(lVar8 + 0xd0);
      if (iVar3 <= iVar5) {
        iVar5 = iVar3;
      }
      *(int *)(param_1 + 400) = iVar5;
    }
  }
  FUN_14076c2c0(*(undefined8 *)(param_1 + 0x178));
  return;
}

