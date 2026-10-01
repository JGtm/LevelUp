
void FUN_140518b0c(void)

{
  int *piVar1;
  longlong lVar2;
  int iVar3;
  bool bVar4;
  longlong lVar5;
  char cVar6;
  longlong lVar7;
  longlong lVar8;
  longlong *plVar9;
  int iVar10;
  int iVar11;
  longlong local_128;
  undefined1 local_120 [248];
  
  lVar5 = DAT_144e61a00;
  if (((DAT_144de4c10 != '\0') && (DAT_144eadd40 != '\0')) &&
     (cVar6 = FUN_1406cb0a4(), cVar6 == '\0')) {
    lVar8 = 0;
    while (lVar8 = FUN_140518c20(lVar8), lVar8 != 0) {
      if (3 < *(int *)(lVar8 + 0x3044)) {
        FUN_142fd0ca0();
      }
    }
    FUN_142fd145c();
    FUN_142f27fdc();
  }
  memset(local_120,0,0xf8);
  iVar10 = 0;
  lVar7 = FUN_140518c20(0);
  lVar8 = 0;
  if (lVar7 != 0) {
    plVar9 = &local_128;
    do {
      *plVar9 = lVar7;
      plVar9 = plVar9 + 1;
      iVar10 = iVar10 + 1;
      lVar7 = FUN_140518c20(lVar7);
    } while (lVar7 != 0);
    lVar8 = 0;
    if (iVar10 != 0) {
      lVar7 = (longlong)*(int *)(lVar5 + 0x908) % (longlong)iVar10;
      bVar4 = false;
      lVar2 = lVar7;
      do {
        iVar3 = (int)lVar2;
        cVar6 = FUN_140516fa0(*(undefined8 *)(local_120 + (longlong)iVar3 * 8 + -8));
        iVar11 = (int)lVar7;
        if ((cVar6 != '\0') && (iVar3 == iVar11)) {
          bVar4 = true;
        }
        lVar2 = (longlong)(iVar3 + 1) % (longlong)iVar10;
      } while ((int)lVar2 != iVar11);
      if (bVar4) {
        piVar1 = (int *)(lVar5 + 0x908);
        *piVar1 = *piVar1 + 1;
      }
    }
  }
  while (lVar8 = FUN_140518c20(lVar8), lVar8 != 0) {
    (**(code **)(**(longlong **)(lVar8 + 0x3928) + 0x28))
              (*(longlong **)(lVar8 + 0x3928),DAT_144714890);
  }
  DAT_144714890 = DAT_144714890 + 1;
  return;
}

