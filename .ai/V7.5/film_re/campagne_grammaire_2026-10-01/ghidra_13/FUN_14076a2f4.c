
void FUN_14076a2f4(uint *param_1,longlong param_2)

{
  undefined4 *puVar1;
  uint uVar2;
  char cVar3;
  longlong lVar4;
  longlong lVar5;
  ulonglong *puVar6;
  undefined8 *puVar7;
  longlong lVar8;
  longlong lVar9;
  undefined8 local_148;
  undefined8 uStack_140;
  undefined8 local_138;
  undefined8 uStack_130;
  undefined8 local_128;
  undefined8 uStack_120;
  undefined8 local_118;
  undefined8 uStack_110;
  undefined4 local_108;
  undefined4 uStack_104;
  undefined4 uStack_100;
  undefined4 uStack_fc;
  undefined4 local_f8;
  undefined4 uStack_f4;
  undefined4 uStack_f0;
  undefined4 uStack_ec;
  undefined4 local_e8;
  undefined4 uStack_e4;
  ulonglong local_d8;
  ulonglong uStack_d0;
  ulonglong local_c8;
  ulonglong uStack_c0;
  ulonglong local_b8;
  ulonglong uStack_b0;
  ulonglong local_a8;
  ulonglong uStack_a0;
  undefined4 local_98;
  undefined4 uStack_94;
  undefined4 uStack_90;
  undefined4 uStack_8c;
  ulonglong local_88;
  undefined1 local_6c;
  undefined1 local_1f;
  
  cVar3 = FUN_1405f1e0c();
  if (((cVar3 == '\0') && (DAT_1451d2628 == -1)) && (param_1[2] - 1 < 2)) {
    lVar4 = FUN_1405d3b40(*(undefined8 *)(param_1 + 8));
    if (lVar4 != 0) {
      lVar4 = *(longlong *)(lVar4 + 0x10);
      lVar8 = lVar4 + 0x1b908;
      if (lVar8 != 0) {
        uVar2 = *param_1;
        lVar9 = param_2 + 0x10 + (longlong)(int)uVar2 * 0xc0;
        cVar3 = FUN_14048ee34();
        if (cVar3 == '\0') {
          FUN_14076a280(&local_148);
          lVar5 = FUN_140497308(param_1[1]);
          FUN_1406d175c(lVar9,*(undefined4 *)(lVar5 + 0x2e0),&local_148);
          lVar5 = (longlong)(int)uVar2 * 0x68;
          local_d8 = local_d8 & 0xffffffffffffff00;
          puVar7 = (undefined8 *)(lVar5 + 0x25f0 + lVar8);
          *puVar7 = local_148;
          puVar7[1] = uStack_140;
          puVar7 = (undefined8 *)(lVar5 + 0x2600 + lVar8);
          *puVar7 = local_138;
          puVar7[1] = uStack_130;
          puVar7 = (undefined8 *)(lVar5 + 0x2610 + lVar8);
          *puVar7 = local_128;
          puVar7[1] = uStack_120;
          puVar7 = (undefined8 *)(lVar5 + 0x2620 + lVar8);
          *puVar7 = local_118;
          puVar7[1] = uStack_110;
          puVar7 = (undefined8 *)(lVar5 + 0x2630 + lVar8);
          *puVar7 = CONCAT44(uStack_104,local_108);
          puVar7[1] = CONCAT44(uStack_fc,uStack_100);
          puVar1 = (undefined4 *)(lVar5 + 0x2640 + lVar8);
          *puVar1 = local_f8;
          puVar1[1] = uStack_f4;
          puVar1[2] = uStack_f0;
          puVar1[3] = uStack_ec;
          *(ulonglong *)(lVar5 + 0x2650 + lVar8) = CONCAT44(uStack_e4,local_e8);
          *(uint *)(lVar4 + 0x1de58) = *(uint *)(lVar4 + 0x1de58) | 1 << (uVar2 & 0x1f);
          FUN_14047bdd0(&local_a8,0xc,5,&LAB_14047bf30);
          uStack_b0 = 0;
          local_6c = 0;
          local_1f = 0;
          cVar3 = FUN_14076a484(param_1[1],lVar9,&local_d8);
          if ((cVar3 != '\0') && (param_1[2] == 1)) {
            FUN_14076d11c(lVar8,uVar2,&local_d8);
          }
        }
        else {
          puVar6 = (ulonglong *)FUN_142bca92c(&local_148,lVar9);
          local_d8 = *puVar6;
          uStack_d0 = puVar6[1];
          local_c8 = puVar6[2];
          uStack_c0 = puVar6[3];
          local_b8 = puVar6[4];
          uStack_b0 = puVar6[5];
          local_a8 = puVar6[6];
          uStack_a0 = puVar6[7];
          local_98 = (undefined4)puVar6[8];
          uStack_94 = *(undefined4 *)((longlong)puVar6 + 0x44);
          uStack_90 = (undefined4)puVar6[9];
          uStack_8c = *(undefined4 *)((longlong)puVar6 + 0x4c);
          local_88 = puVar6[10];
          FUN_142f2aa58(lVar8,uVar2,&local_d8);
          puVar7 = (undefined8 *)FUN_142bcaa50(&local_d8,param_1[1]);
          local_148 = *puVar7;
          uStack_140 = puVar7[1];
          local_138 = puVar7[2];
          uStack_130 = puVar7[3];
          local_128 = puVar7[4];
          uStack_120 = puVar7[5];
          local_118 = puVar7[6];
          uStack_110 = puVar7[7];
          local_108 = *(undefined4 *)(puVar7 + 8);
          uStack_104 = *(undefined4 *)((longlong)puVar7 + 0x44);
          uStack_100 = *(undefined4 *)(puVar7 + 9);
          uStack_fc = *(undefined4 *)((longlong)puVar7 + 0x4c);
          local_f8 = *(undefined4 *)(puVar7 + 10);
          uStack_f4 = *(undefined4 *)((longlong)puVar7 + 0x54);
          uStack_f0 = *(undefined4 *)(puVar7 + 0xb);
          uStack_ec = *(undefined4 *)((longlong)puVar7 + 0x5c);
          local_e8 = *(undefined4 *)(puVar7 + 0xc);
          FUN_142f2aee0(lVar8,uVar2,&local_148);
        }
      }
    }
  }
  return;
}

