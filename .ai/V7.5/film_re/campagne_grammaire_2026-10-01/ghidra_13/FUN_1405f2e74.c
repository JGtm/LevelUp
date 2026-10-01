
void FUN_1405f2e74(longlong param_1,undefined8 param_2)

{
  longlong lVar1;
  char cVar2;
  int iVar3;
  longlong lVar4;
  uint uVar5;
  uint uVar6;
  longlong lVar7;
  int *piVar8;
  IMAGE_DOS_HEADER *pIVar9;
  undefined2 local_res8 [4];
  undefined8 local_res10;
  
  if (*(int *)(param_1 + 0x10) != 2) {
    uVar5 = 0;
    local_res10 = param_2;
    while (pIVar9 = &switchD_14156874c::caseD_4, uVar5 != 0xffffffff) {
      iVar3 = FUN_14049753c(param_1 + 0x15);
      if (iVar3 != -1) {
        for (piVar8 = (int *)(param_1 + 0x1d8); piVar8 != (int *)(param_1 + 0x20d8);
            piVar8 = piVar8 + 0x3e) {
          if (*piVar8 != -1) {
            uVar6 = piVar8[1];
            lVar4 = (ulonglong)(uVar6 & 1) * 8;
            if (*(longlong *)(pIVar9[0x9d8a7].e_magic + lVar4) == 0) {
              lVar4 = *(longlong *)(pIVar9[0x9d5d8].e_program + lVar4 + 0x10);
            }
            else {
              lVar7 = *(longlong *)ThreadLocalStoragePointer;
              if (*(char *)(lVar7 + 200) == '\0') {
                __dyn_tls_on_demand_init();
                pIVar9 = &switchD_14156874c::caseD_4;
              }
              lVar4 = *(longlong *)(lVar4 + 0x4aab0 + lVar7);
            }
            if (uVar6 == 0xffffffff) {
              uVar6 = 0xffffffff;
            }
            else {
              uVar6 = uVar6 >> 1 & 0x7fff;
            }
            lVar7 = (ulonglong)(uVar6 & 0xffff) * 0x358 + *(longlong *)(lVar4 + 0x78);
            uVar6 = *(uint *)(lVar7 + 0x2e4);
            lVar4 = (ulonglong)(uVar6 & 1) * 8;
            if (*(longlong *)(pIVar9[0x9d866].e_program + lVar4 + 0x28) == 0) {
              lVar4 = *(longlong *)(pIVar9[0x9d5fa].e_magic + lVar4);
            }
            else {
              lVar1 = *(longlong *)ThreadLocalStoragePointer;
              if (*(char *)(lVar1 + 200) == '\0') {
                __dyn_tls_on_demand_init();
                uVar6 = *(uint *)(lVar7 + 0x2e4);
                pIVar9 = &switchD_14156874c::caseD_4;
              }
              lVar4 = *(longlong *)(lVar4 + 0x4ab20 + lVar1);
            }
            if (uVar6 == 0xffffffff) {
              uVar6 = 0xffffffff;
            }
            else {
              uVar6 = uVar6 >> 1 & 0x7fff;
            }
            if ((*(short *)((ulonglong)(uVar6 & 0xffff) * 8000 + 0x28c + *(longlong *)(lVar4 + 0x78)
                           ) == iVar3) && (*(uint *)(lVar7 + 0x18) == uVar5)) {
              if (piVar8 != (int *)0x0) {
                FUN_14076a2f4(piVar8,local_res10);
              }
              break;
            }
          }
        }
      }
      uVar6 = uVar5 + 1;
      uVar5 = 0xffffffff;
      if (uVar6 < 4) {
        uVar5 = uVar6;
      }
    }
    cVar2 = FUN_14048ee34();
    if (((cVar2 != '\0') && (cVar2 = FUN_1404f25f4(), cVar2 != '\0')) &&
       (lVar4 = FUN_1405d3b40(param_1), lVar4 != 0)) {
      local_res8[0] = *(undefined2 *)(DAT_1451f8890 + 0xc);
      FUN_142f2aba0(*(longlong *)(lVar4 + 0x10) + 0x1b908,local_res8);
    }
  }
  return;
}

