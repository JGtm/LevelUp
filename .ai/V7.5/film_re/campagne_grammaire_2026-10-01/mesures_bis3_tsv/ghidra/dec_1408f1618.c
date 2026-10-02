
/* WARNING: Globals starting with '_' overlap smaller symbols at the same address */

void FUN_1408f1618(longlong param_1,uint param_2)

{
  longlong *plVar1;
  longlong lVar2;
  uint uVar3;
  ulonglong uVar4;
  ulonglong uVar5;
  uint uVar6;
  uint uVar7;
  ulonglong uVar8;
  
  plVar1 = (longlong *)(param_1 + 0x120);
  uVar7 = param_2 & 0x3fffffff;
  uVar8 = (ulonglong)uVar7;
  uVar5 = (*(longlong *)(param_1 + 0x128) - *plVar1) / 0x18;
  if (((uVar5 <= uVar8) && (*(char *)(param_1 + 0x138) != '\0')) &&
     (*(char *)(param_1 + 0x158) != '\0')) {
    uVar6 = uVar7 + 1;
    uVar4 = (ulonglong)uVar6;
    if ((*(char *)(param_1 + 0x138) != '\0') && (uVar4 != uVar5)) {
      uVar3 = 0x1fff;
      if (0x1fff < uVar4) {
        uVar3 = uVar6;
      }
      FUN_140c4f510(plVar1,uVar3);
    }
    if ((*(char *)(param_1 + 0x158) != '\0') &&
       (uVar4 != *(longlong *)(param_1 + 0x148) - *(longlong *)(param_1 + 0x140) >> 5)) {
      uVar5 = 0x1fff;
      if (0x1fff < uVar4) {
        uVar5 = uVar4;
      }
      FUN_1411b239c((longlong *)(param_1 + 0x140),uVar5);
    }
    DAT_1451f98d4 = uVar7 - 0x1ff;
    DAT_144706100 = uVar6;
    DAT_1451f98dc = DAT_1451f98d4;
    DAT_1451f990c = uVar6;
    _DAT_1451f9914 = uVar6;
  }
  lVar2 = *plVar1;
  *(undefined8 *)(lVar2 + 8 + uVar8 * 0x18) = 0;
  *(undefined8 *)(lVar2 + 0x10 + uVar8 * 0x18) = 0;
  *(undefined1 *)(lVar2 + uVar8 * 0x18) = 1;
  *(undefined4 *)(lVar2 + 4 + uVar8 * 0x18) = 1;
  *(byte *)(lVar2 + 1 + uVar8 * 0x18) = (byte)(param_2 >> 0x1e);
  return;
}

