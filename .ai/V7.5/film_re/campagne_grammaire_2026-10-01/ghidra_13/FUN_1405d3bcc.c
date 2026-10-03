
void FUN_1405d3bcc(longlong param_1,char param_2,undefined4 *param_3,ushort *param_4)

{
  undefined2 uVar1;
  undefined4 uVar2;
  void *pvVar3;
  char cVar4;
  int iVar5;
  longlong lVar6;
  ushort uVar7;
  undefined8 local_res8;
  longlong local_38 [2];
  
  if (*(int *)(param_1 + 0x10) == 2) {
    FUN_142e358e4(param_1,param_3,param_4);
  }
  else {
    if ((*(int *)(param_1 + 0x10) == 5) && (*(int *)(param_1 + 0x20) == 4)) {
      local_38[0] = 0;
      local_res8 = 4;
      cVar4 = FUN_1405d3b78(param_1,&local_res8,local_38);
      if ((cVar4 != '\0') &&
         ((*(int *)(local_38[0] + 0x6c) == 0 && (*(int *)(local_38[0] + 0x70) == 5)))) {
        FUN_1428e27c0(&DAT_144c23178);
      }
    }
    pvVar3 = ThreadLocalStoragePointer;
    *param_4 = *param_4 | 1;
    uVar2 = *(undefined4 *)(param_1 + 0x3c);
    lVar6 = *(longlong *)pvVar3;
    *param_3 = uVar2;
    param_3[0xf5f] = *(undefined4 *)(*(longlong *)(lVar6 + 0x6c0) + 0x40);
    param_3[0xf60] = **(undefined4 **)(lVar6 + 0x5a8);
    *(undefined4 *)(param_4 + 6) = uVar2;
    FUN_140cb0348(DAT_144e61d80,param_3 + 0x604,param_3 + 0x606);
    if (*(char *)(param_3 + 0x604) != '\0') {
      FUN_140a1e948(param_3 + 0x606,param_3 + 0x608);
    }
    FUN_1405d2c04();
    if ((*(int *)(param_1 + 0x10) == 4) && (cVar4 = FUN_1406cb0cc(), cVar4 != '\0')) {
      FUN_1406cb670(*(longlong *)(param_1 + 8) + 0x42d0);
    }
    iVar5 = *(int *)(param_1 + 0x10);
    if (iVar5 - 3U < 3) {
      if (*(int *)(DAT_144e61d78 + 0x10) - 3U < 3) {
        FUN_1405d57e0(&DAT_145173738);
        iVar5 = *(int *)(param_1 + 0x10);
      }
      if ((((iVar5 == 4) && (*(char *)(param_1 + 0x6d) != '\0')) &&
          (*(char *)(*(longlong *)(param_1 + 8) + 0x42d3) != '\0')) &&
         (cVar4 = FUN_1406cb0cc(), cVar4 != '\0')) {
        FUN_142e2a9f4(*(longlong *)(param_1 + 8) + 0x42d0);
        *(undefined1 *)(param_1 + 0x6d) = 0;
      }
    }
    uVar7 = *(ushort *)(param_3 + 1) & 0xffef | 1;
    if (param_2 == '\0') {
      uVar7 = *(ushort *)(param_3 + 1) & 0xffee;
    }
    *(ushort *)(param_3 + 1) = uVar7;
    if ((uVar7 & 1) != 0) {
      FUN_1404975a0(param_1);
      cVar4 = FUN_14048ee34();
      if (((cVar4 != '\0') && (cVar4 = FUN_1404f25f4(), cVar4 != '\0')) &&
         ((lVar6 = FUN_1405d3b40(param_1), lVar6 != 0 &&
          (lVar6 = *(longlong *)(lVar6 + 0x10), *(char *)(lVar6 + 0x1de6d) != '\0')))) {
        uVar1 = *(undefined2 *)(lVar6 + 0x1de6d);
        *(undefined1 *)(lVar6 + 0x1de6d) = 0;
        cVar4 = (char)((ushort)uVar1 >> 8);
        if ((*(char *)(DAT_1451f8890 + 0xd) != cVar4) && (*(char *)(DAT_1451f8890 + 0xf) != cVar4))
        {
          *(undefined2 *)(DAT_1451f8890 + 0xe) = uVar1;
        }
      }
      iVar5 = *(int *)(param_1 + 0x10);
      if (iVar5 != 4) {
        if (*(char *)(param_1 + 0x45) != '\0') {
          *(ushort *)(param_3 + 1) = *(ushort *)(param_3 + 1) | 2;
          iVar5 = *(int *)(param_1 + 0x10);
          *(undefined1 *)(param_1 + 0x45) = 0;
        }
        if (iVar5 == 3 || iVar5 == 4) {
          FUN_142e2ff5c(param_1);
        }
      }
    }
    if (*(char *)(param_1 + 0x46) != '\0') {
      *(ushort *)(param_3 + 1) = *(ushort *)(param_3 + 1) | 4;
      *(undefined1 *)(param_1 + 0x46) = 0;
    }
    *(undefined8 *)(param_3 + 0xf73) = 0;
    *(undefined8 *)(param_3 + 0xf75) = 0;
    *(undefined8 *)(param_3 + 0xf78) = 0;
    *(undefined8 *)(param_3 + 0xf7a) = 0;
    param_3[0xf77] = 64000;
    *(undefined1 *)(param_3 + 0xf72) = 1;
    memset((void *)((longlong)param_3 + 0x3df1),0,0x9e0f);
    *(undefined1 *)(param_3 + 0xf7c) = 1;
    if ((*(char *)(param_1 + 0x2100) != '\0') && (0 < *(int *)(param_1 + 0x210c))) {
      FUN_141f86fd0(param_3 + 0xf72);
    }
    if ((*(char *)(param_1 + 0x2128) != '\0') && (0 < *(int *)(param_1 + 0x2134))) {
      *(ushort *)(param_3 + 1) = *(ushort *)(param_3 + 1) | 8;
    }
    if (((((*(ushort *)(param_3 + 1) & 1) != 0) || ((*(ushort *)(param_3 + 1) & 8) != 0)) &&
        (*(char *)(param_1 + 0x2128) != '\0')) && (0 < *(int *)(param_1 + 0x2134))) {
      FUN_141f8706c(param_3 + 0xf7c);
    }
  }
  return;
}

