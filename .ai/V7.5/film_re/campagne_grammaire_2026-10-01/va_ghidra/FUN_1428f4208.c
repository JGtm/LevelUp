
/* WARNING: Removing unreachable block (ram,0x0001428f43e7) */
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_1428f4208(void)

{
  undefined8 uVar1;
  char cVar2;
  undefined4 uVar3;
  undefined4 *puVar4;
  undefined8 uVar5;
  undefined8 *puVar6;
  longlong lVar7;
  undefined8 local_res10;
  longlong local_res18;
  longlong local_res20;
  undefined8 local_58;
  undefined8 uStack_50;
  undefined4 local_48;
  undefined4 uStack_44;
  undefined4 uStack_40;
  undefined4 uStack_3c;
  undefined1 local_38 [24];
  
  FUN_1428f385c(&local_res20);
  if ((local_res20 != 0) && (cVar2 = FUN_1428f339c(local_res20), cVar2 != '\0')) {
    FUN_140a97f74(&local_res18,local_res20);
    if (local_res18 == 0) {
      FUN_140a97d20(&local_res18);
      FUN_1424c98e0(&local_res20);
      return;
    }
    cVar2 = FUN_1428f40e8(local_res20);
    if ((cVar2 != '\0') && (cVar2 = FUN_1428f3ff8(DAT_144c25358,local_res20), cVar2 != '\0')) {
      FUN_1429aa630(4);
      FUN_1428f3b9c(&local_res10);
      lVar7 = (longlong)(DAT_14470619c + 0x10000 + DAT_144e51208 + DAT_144e51210 + DAT_1447061a0);
      uVar3 = FUN_1428f3c10(local_res20);
      local_58 = DAT_144c25340;
      uStack_50 = DAT_144c25340 + lVar7;
      puVar4 = (undefined4 *)FUN_1429aa858(local_res10,local_38);
      local_48 = *puVar4;
      uStack_44 = puVar4[1];
      uStack_40 = puVar4[2];
      uStack_3c = puVar4[3];
      cVar2 = FUN_1428f4534(local_48,&local_58,&local_48,uVar3);
      uVar5 = FUN_140be96d8();
      FUN_142bad944(uVar5,0);
      FUN_142545678();
      FUN_1429aa56c(4);
      puVar6 = (undefined8 *)FUN_1429aa858(local_res10,local_38);
      uVar5 = *puVar6;
      uVar1 = puVar6[1];
      local_58._0_4_ = (undefined4)uVar5;
      local_58._4_4_ = (undefined4)((ulonglong)uVar5 >> 0x20);
      uStack_50._0_4_ = (undefined4)uVar1;
      uStack_50._4_4_ = (undefined4)((ulonglong)uVar1 >> 0x20);
      local_48 = (undefined4)local_58;
      uStack_44 = local_58._4_4_;
      uStack_40 = (undefined4)uStack_50;
      uStack_3c = uStack_50._4_4_;
      uVar3 = (undefined4)local_58;
      local_58 = uVar5;
      uStack_50 = uVar1;
      FUN_1429aaa0c(uVar3,&local_48);
      FUN_1429aa410(4);
      FUN_142bbb274(1);
      if (cVar2 != '\0') {
        lVar7 = *(longlong *)ThreadLocalStoragePointer;
        if (*(char *)(lVar7 + 200) == '\0') {
          __dyn_tls_on_demand_init();
        }
        puVar4 = *(undefined4 **)(lVar7 + 0x488);
        _DAT_144b440a0 = *puVar4;
        _DAT_144b440a4 = puVar4[1];
        _DAT_144b440a8 = puVar4[2];
        _DAT_144b440ac = puVar4[3];
        DAT_144b440b0 = puVar4[4];
      }
      FUN_1411c60c4(&local_res10);
    }
    FUN_140a97d20(&local_res18);
  }
  FUN_1424c98e0(&local_res20);
  return;
}

