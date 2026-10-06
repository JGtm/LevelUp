
void FUN_140be946c(longlong param_1)

{
  int iVar1;
  int iVar2;
  char cVar3;
  uint uVar4;
  longlong lVar5;
  longlong lVar6;
  ulonglong uVar8;
  char *pcVar9;
  char *pcVar10;
  uint uVar11;
  ulonglong uVar12;
  ulonglong uVar7;
  
  lVar5 = FUN_140be96d8();
  cVar3 = FUN_140be96b8();
  uVar8 = 0;
  if (cVar3 != '\0') {
    FUN_14095944c(lVar5,param_1);
    *(undefined1 *)(lVar5 + 0xea719) = 0;
    FUN_140be9708();
  }
  FUN_1404f1614();
  FUN_1406f6df4();
  pcVar9 = (char *)(param_1 + 0xeaaf0);
  pcVar10 = pcVar9;
  uVar7 = uVar8;
  do {
    FUN_1424d8ab0(uVar7,-(ulonglong)(*pcVar10 != '\0') & (ulonglong)(pcVar10 + 0xcd8));
    uVar11 = (int)uVar7 + 1;
    uVar7 = (ulonglong)uVar11;
    pcVar10 = pcVar10 + 0x1450;
  } while (uVar11 < 0x20);
  FUN_140be977c();
  lVar6 = FUN_1404f1614();
  lVar5 = *(longlong *)ThreadLocalStoragePointer;
  **(undefined4 **)(lVar5 + 0x5a8) = *(undefined4 *)(lVar6 + 0x18);
  lVar6 = *(longlong *)(lVar5 + 0x238);
  *(undefined1 *)(lVar6 + 3) = 0;
  *(undefined2 *)(lVar6 + 0x92) = 0;
  *(undefined4 *)(lVar6 + 0x94) = 0;
  *(undefined1 *)(lVar6 + 0x98) = 0;
  *(undefined2 *)(lVar6 + 0x9a) = 0;
  *(undefined4 *)(lVar6 + 0x9c) = 0;
  *(undefined1 *)(lVar6 + 0xa2) = 0;
  FUN_14051c1ec("game_instance","%016I64X",*(undefined8 *)(param_1 + 0x10));
  FUN_14051c1ec("game_simulation",&DAT_143686030,(&PTR_DAT_143cef4b0)[*(int *)(param_1 + 4)]);
  FUN_14051c1ec("game_playback",&DAT_143686030,(&PTR_DAT_143ce70c0)[*(int *)(param_1 + 0xea71c)]);
  *(undefined4 *)(*(longlong *)(lVar5 + 0x238) + 0xa4) = 0xffffffff;
  cVar3 = FUN_1404779b4(DAT_144976b60 + 0x774);
  if (cVar3 != '\0') {
    lVar6 = FUN_140583a94(DAT_144976b60 + 0x78c,0x67707464);
    uVar7 = uVar8;
    uVar12 = uVar8;
    do {
      cVar3 = *pcVar9;
      pcVar9 = pcVar9 + 0x1450;
      uVar11 = (uint)uVar12 + 1;
      if (cVar3 == '\0') {
        uVar11 = (uint)uVar12;
      }
      uVar4 = (int)uVar7 + 1;
      uVar7 = (ulonglong)uVar4;
      uVar12 = (ulonglong)uVar11;
    } while (uVar4 < 0x20);
    uVar7 = uVar8;
    if (0 < *(int *)(lVar6 + 0x20)) {
      do {
        iVar1 = *(int *)(uVar8 + *(longlong *)(lVar6 + 0x10));
        if (iVar1 <= (int)uVar11) {
          iVar2 = *(int *)(*(longlong *)(lVar5 + 0x238) + 0xa4);
          if ((iVar2 == -1) ||
             (*(int *)((longlong)iVar2 * 0x1c + *(longlong *)(lVar6 + 0x10)) < iVar1)) {
            *(int *)(*(longlong *)(lVar5 + 0x238) + 0xa4) = (int)uVar7;
          }
        }
        uVar4 = (int)uVar7 + 1;
        uVar8 = uVar8 + 0x1c;
        uVar7 = (ulonglong)uVar4;
      } while ((int)uVar4 < *(int *)(lVar6 + 0x20));
    }
  }
  return;
}

