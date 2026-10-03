
/* WARNING: Function: _alloca_probe replaced with injection: alloca_probe */

void FUN_142987460(longlong param_1,undefined8 param_2)

{
  int iVar1;
  longlong *plVar2;
  int *piVar3;
  undefined1 *puVar4;
  int iVar5;
  uint uVar6;
  uint uVar7;
  ulonglong uVar8;
  undefined8 *puVar9;
  undefined8 *puVar10;
  int local_res18 [2];
  int local_res20 [2];
  int aiStack_78060 [4];
  undefined8 uStack_78050;
  undefined8 uStack_78048;
  undefined1 auStack_78038 [491520];
  
  uVar6 = 0;
  uStack_78050 = 0;
  uStack_78048 = 0;
  iVar5 = 0;
  aiStack_78060[0] = 0;
  aiStack_78060[1] = 0;
  aiStack_78060[2] = 0;
  aiStack_78060[3] = 0;
  DAT_144706104 = FUN_1406cf008(param_2);
  puVar9 = (undefined8 *)(param_1 + 0x228);
  piVar3 = aiStack_78060;
  uVar7 = 0;
  puVar10 = puVar9;
  do {
    plVar2 = (longlong *)*puVar10;
    local_res18[0] = 0;
    *piVar3 = iVar5;
    (**(code **)(*plVar2 + 0x60))
              (plVar2,0xa00 - iVar5,auStack_78038 + (longlong)iVar5 * 0xc0,local_res18);
    iVar5 = iVar5 + local_res18[0];
    local_res20[0] = 0;
    (**(code **)(*plVar2 + 0x40))
              (plVar2,&uStack_78050,param_2,0xa00 - iVar5,auStack_78038 + (longlong)iVar5 * 0xc0,
               local_res20);
    iVar5 = iVar5 + local_res20[0];
    puVar10 = puVar10 + 1;
    uVar7 = uVar7 + 1;
    piVar3 = piVar3 + 1;
  } while (uVar7 < 3);
  piVar3 = aiStack_78060;
  aiStack_78060[3] = iVar5;
  do {
    iVar1 = *piVar3;
    piVar3 = piVar3 + 1;
    plVar2 = (longlong *)*puVar9;
    if (iVar1 < *piVar3) {
      puVar4 = auStack_78038 + (longlong)iVar1 * 0xc0;
      uVar8 = (ulonglong)(uint)(*piVar3 - iVar1);
      do {
        (**(code **)(*plVar2 + 0x48))(plVar2,puVar4);
        puVar4 = puVar4 + 0xc0;
        uVar8 = uVar8 - 1;
      } while (uVar8 != 0);
    }
    uVar6 = uVar6 + 1;
    puVar9 = puVar9 + 1;
  } while (uVar6 < 3);
  FUN_1406d07b0(auStack_78038,iVar5,0);
  return;
}

