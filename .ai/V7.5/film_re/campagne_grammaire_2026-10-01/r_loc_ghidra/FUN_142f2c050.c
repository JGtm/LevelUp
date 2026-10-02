
void FUN_142f2c050(undefined8 param_1)

{
  longlong *plVar1;
  longlong lVar2;
  ulonglong uVar3;
  longlong lVar4;
  longlong lVar5;
  ulonglong uVar6;
  longlong *plVar7;
  longlong local_38;
  
  lVar4 = 0;
  if (DAT_144e61d78 != 0) {
    lVar4 = *(longlong *)(DAT_144e61d78 + 8) + 0x180;
  }
  plVar7 = (longlong *)(lVar4 + 0x130);
  uVar3 = 0;
  lVar5 = DAT_144de4b90;
  do {
    if (0 < *(int *)((longlong)&DAT_14521d928 + uVar3)) {
      if ((lVar4 != 0) && (*plVar7 != 0)) {
        uVar6 = *(ulonglong *)(*plVar7 + 0x50);
        FUN_142f20160();
        if ((*(char *)(local_38 + 0x19) != '\0') ||
           (lVar2 = local_38, uVar6 < *(ulonglong *)(local_38 + 0x20))) {
          lVar2 = lVar5;
        }
        if (lVar2 != lVar5) goto LAB_142f2c133;
        plVar1 = (longlong *)FUN_142f20ff4();
        *(undefined1 *)(*plVar1 + 0x28) = 1;
      }
      FUN_14080ada0(*(undefined4 *)((longlong)&DAT_14521d920 + uVar3));
      FUN_1406d60f4(param_1);
      lVar5 = DAT_144de4b90;
    }
LAB_142f2c133:
    uVar3 = uVar3 + 0x10;
    plVar7 = plVar7 + 1;
    if (0x7fef < uVar3) {
      return;
    }
  } while( true );
}

