ulonglong FUN_1410301c4(undefined8 param_1,longlong *param_2)
{
  int iVar1;
  int *piVar2;
  longlong lVar3;
  undefined8 *puVar4;
  ulonglong uVar5;
  undefined8 *puVar6;
  undefined8 local_res8;
  puVar6 = (undefined8 *)*param_2;
  piVar2 = *(int **)puVar6[2];
  if ((DAT_1449f7970 != 0) || (local_res8 = param_1, *piVar2 != DAT_1449f7980)) {
    puVar6 = *(undefined8 **)*puVar6;
    local_res8._1_7_ = (undefined7)((ulonglong)param_1 >> 8);
    local_res8 = CONCAT71(local_res8._1_7_,0x10);
    FUN_140ac78d0(*puVar6,&local_res8,1);
    iVar1 = *piVar2;
    FUN_140ac7668(*puVar6,iVar1 >> 0x1f ^ iVar1 * 2);
    puVar6 = (undefined8 *)*param_2;
  }
  lVar3 = *(longlong *)puVar6[2];
  if ((DAT_1449f7a30 != 0) || (*(int *)(lVar3 + 4) != DAT_1449f7a40)) {
    puVar6 = *(undefined8 **)*puVar6;
    local_res8 = CONCAT71(local_res8._1_7_,0x30);
    FUN_140ac78d0(*puVar6,&local_res8,1);
    iVar1 = *(int *)(lVar3 + 4);
    FUN_140ac7668(*puVar6,iVar1 >> 0x1f ^ iVar1 * 2);
    puVar6 = (undefined8 *)*param_2;
  }
  FUN_14100a474(*puVar6,2);
  FUN_140ed5ce8(*(undefined8 *)*param_2,3,&DAT_1449f7b60,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x18);
  FUN_140ed5ce8(*(undefined8 *)*param_2,4,&DAT_1449f7c20,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x98);
  puVar6 = (undefined8 *)*param_2;
  if ((DAT_1449f7d30 != 0) || (*(char *)(*(longlong *)puVar6[2] + 0x118) != '\0')) {
    FUN_1424e4dd8(*puVar6);
    puVar6 = (undefined8 *)*param_2;
  }
  FUN_140d1a268(*puVar6,6,&DAT_1449f7da0,*(longlong *)puVar6[2] + 0x148);
  FUN_140d1a268(*(undefined8 *)*param_2,7,&DAT_1449f7e60,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x14c);
  FUN_140d1a268(*(undefined8 *)*param_2,8,&DAT_1449f7f20,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x150);
  FUN_140d1a268(*(undefined8 *)*param_2,9,&DAT_1449f7fe0,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x154);
  FUN_140db745c(*(undefined8 *)*param_2,10,&DAT_1449f80a0,
                *(longlong *)((undefined8 *)*param_2)[2] + 0x158);
  puVar6 = (undefined8 *)*param_2;
  puVar4 = (undefined8 *)*puVar6;
  lVar3 = *(longlong *)puVar6[2];
  if ((DAT_1449f81b0 != 0) || (*(char *)(lVar3 + 0x159) != '\0')) {
    FUN_140ac75e8(*puVar4,0xc,0xb);
    FUN_140ff9830(puVar4,lVar3 + 0x159);
    puVar6 = (undefined8 *)*param_2;
  }
  FUN_14100a474(*puVar6,0xd);
  uVar5 = FUN_140ed5ce8(*(undefined8 *)*param_2,0xe,&DAT_1449f82e0,
                        *(longlong *)((undefined8 *)*param_2)[2] + 0x16c);
  return uVar5 & 0xffffffffffffff00;
}
