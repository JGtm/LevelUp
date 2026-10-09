
void FUN_141f85a84(longlong param_1,uint *param_2,longlong param_3)

{
  int iVar1;
  uint uVar2;
  bool bVar3;
  char cVar4;
  longlong lVar5;
  longlong lVar6;
  undefined8 *puVar7;
  int *piVar8;
  int *piVar9;
  char local_res20;
  int local_244;
  int local_240;
  undefined2 local_23c;
  longlong local_238;
  undefined4 local_230;
  undefined4 local_22c;
  undefined8 local_228;
  undefined8 uStack_220;
  undefined8 local_218;
  undefined8 uStack_210;
  ulonglong local_208;
  undefined8 uStack_200;
  undefined8 local_1f8;
  undefined8 uStack_1f0;
  undefined4 local_1e8;
  undefined4 uStack_1e4;
  undefined4 uStack_1e0;
  undefined4 uStack_1dc;
  ulonglong local_1d8;
  undefined1 local_1c8 [112];
  undefined1 local_158 [96];
  undefined1 local_f8 [192];
  
  piVar8 = (int *)(param_1 + 0x1d8);
  local_244 = 0;
  piVar9 = piVar8;
  do {
    if ((*piVar8 != -1) && ((1 << ((byte)local_244 & 0x1f) & *param_2) != 0)) {
      local_240 = piVar8[6];
      local_23c = (undefined2)piVar8[7];
      lVar5 = FUN_141f864ac(param_1,&local_240);
      if ((lVar5 == 0) ||
         (cVar4 = FUN_1405f0adc(&DAT_144de3ea0,*(undefined4 *)(lVar5 + 0x2c)), cVar4 == '\0')) {
        iVar1 = *piVar8;
        FUN_14076a280();
        local_res20 = '\0';
        local_208 = local_208 & 0xffffffff00000000;
        uVar2 = (uint)local_1f8;
        local_1f8 = CONCAT44(0xffffffff,uVar2 & 0xffffff00);
        uStack_1f0 = CONCAT44(uStack_1f0._4_4_,0xffffffff);
        local_1d8 = (local_1d8 >> 0x10 & 0xffff) << 0x10;
        bVar3 = false;
        cVar4 = FUN_14048ee34();
        if (cVar4 == '\0') {
          local_res20 = FUN_1406d16b8(piVar9,param_3,0,local_1c8);
        }
        else {
          lVar6 = FUN_140497308(piVar8[1]);
          if (*(int *)(lVar6 + 0x2e0) != -1) {
            puVar7 = (undefined8 *)FUN_142bca92c(local_158,param_3);
            bVar3 = true;
            local_228 = *puVar7;
            uStack_220 = puVar7[1];
            local_218 = puVar7[2];
            uStack_210 = puVar7[3];
            local_208 = puVar7[4];
            uStack_200 = puVar7[5];
            local_1f8 = puVar7[6];
            uStack_1f0 = puVar7[7];
            local_1e8 = *(undefined4 *)(puVar7 + 8);
            uStack_1e4 = *(undefined4 *)((longlong)puVar7 + 0x44);
            uStack_1e0 = *(undefined4 *)(puVar7 + 9);
            uStack_1dc = *(undefined4 *)((longlong)puVar7 + 0x4c);
            local_1d8 = puVar7[10];
          }
        }
        lVar6 = 0;
        if (lVar5 != 0) {
          lVar6 = *(longlong *)(lVar5 + 0x10) + 0x1b908;
        }
        FUN_14076abd0(local_f8);
        if (lVar6 != 0) {
          FUN_1405d3b10(lVar6,iVar1,local_f8);
        }
        local_238 = 0;
        local_22c = 0;
        local_230 = 4;
        cVar4 = FUN_1405d3b78(param_1,&local_230);
        lVar5 = local_238;
        while (local_238 = lVar5, cVar4 != '\0') {
          cVar4 = FUN_1405f0adc(&DAT_144de3ea0,*(undefined4 *)(lVar5 + 0x2c));
          if ((cVar4 == '\0') && (lVar5 = *(longlong *)(lVar5 + 0x10) + 0x1b908, lVar5 != 0)) {
            if (local_res20 != '\0') {
              FUN_14076ac14(lVar5,iVar1,local_1c8);
            }
            if (bVar3) {
              FUN_142f2aa58(lVar5,iVar1,&local_228);
            }
          }
          cVar4 = FUN_1405d3b78(param_1,&local_230);
          lVar5 = local_238;
        }
      }
    }
    local_244 = local_244 + 1;
    piVar9 = piVar9 + 0x3e;
    piVar8 = piVar8 + 0x3e;
    param_3 = param_3 + 0xc0;
  } while (local_244 < 0x20);
  return;
}

