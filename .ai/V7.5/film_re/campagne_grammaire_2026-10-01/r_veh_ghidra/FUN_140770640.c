
void FUN_140770640(undefined8 *param_1,uint param_2,int param_3,float *param_4,uint *param_5,
                  int *param_6,longlong param_7)

{
  int iVar1;
  float fVar2;
  undefined8 uVar3;
  uint uVar4;
  float *pfVar5;
  longlong lVar6;
  float *pfVar7;
  int iVar8;
  longlong lVar9;
  float fVar10;
  undefined8 local_38;
  float local_30;
  
  local_38 = *param_1;
  local_30 = *(float *)(param_1 + 1);
  lVar9 = (longlong)param_3;
  *param_6 = param_3;
  param_6[1] = param_3;
  param_6[2] = param_3;
  *param_5 = param_2;
  uVar3 = local_38;
  if (param_4 == (float *)0x0) {
    if (((((param_2 == 0xffffffff) ||
          (lVar6 = (longlong)(int)param_2,
          ((uint)(&DAT_1445ccb60)[param_2 >> 5] >> (param_2 & 0x1f) & 1) == 0)) ||
         ((float)local_38 < *(float *)(&DAT_14462cbe0 + lVar6 * 3))) ||
        ((*(float *)((longlong)&DAT_14462cbe0 + lVar6 * 0x18 + 4) < (float)local_38 ||
         (local_38._4_4_ = (float)((ulonglong)local_38 >> 0x20),
         local_38._4_4_ < *(float *)(&DAT_14462cbe8 + lVar6 * 3))))) ||
       ((*(float *)((longlong)&DAT_14462cbe8 + lVar6 * 0x18 + 4) < local_38._4_4_ ||
        ((local_30 < *(float *)(&DAT_14462cbf0 + lVar6 * 3) ||
         (local_38 = uVar3, *(float *)((longlong)&DAT_14462cbf0 + lVar6 * 0x18 + 4) < local_30))))))
    {
      local_38 = uVar3;
      uVar4 = FUN_14077084c(&local_38);
      *param_5 = uVar4;
    }
    if (*param_5 == 0xffffffff) {
      param_4 = (float *)&DAT_1445cc9c8;
      *(undefined8 *)param_6 = *(undefined8 *)(&DAT_1445cc9e0 + lVar9 * 0xc);
      param_6[2] = *(int *)(&DAT_1445cc9e8 + lVar9 * 0xc);
      FUN_141c105bc(&DAT_1445cc9c8,param_1,&local_38);
    }
    else {
      param_4 = (float *)(&DAT_14462cbe0 + (longlong)(int)*param_5 * 3);
      lVar9 = (longlong)(int)*param_5 * 0x20 + lVar9;
      *(undefined8 *)param_6 = *(undefined8 *)(&DAT_1445ccbe0 + lVar9 * 0xc);
      param_6[2] = *(int *)(&DAT_1445ccbe8 + lVar9 * 0xc);
    }
  }
  else {
    *param_5 = 0xffffffff;
    FUN_141c105bc(param_4,param_1,&local_38);
    FUN_140be9b88(param_3,param_4);
  }
  pfVar7 = (float *)&local_38;
  lVar9 = 3;
  do {
    fVar2 = param_4[1];
    iVar8 = 1 << ((byte)*(undefined4 *)
                         (((longlong)param_6 - (longlong)&local_38) + (longlong)pfVar7) & 0x1f);
    pfVar5 = param_4;
    if (0.0 <= *pfVar7 - *param_4) {
      pfVar5 = pfVar7;
    }
    fVar10 = *pfVar5;
    if (0.0 <= *pfVar5 - fVar2) {
      fVar10 = fVar2;
    }
    iVar1 = iVar8 + -1;
    iVar8 = (int)((fVar10 - *param_4) / ((fVar2 - *param_4) / (float)iVar8));
    if (iVar8 < 1) {
      iVar8 = 0;
    }
    if (iVar1 < iVar8) {
      iVar8 = iVar1;
    }
    param_4 = param_4 + 2;
    *(int *)((param_7 - (longlong)&local_38) + (longlong)pfVar7) = iVar8;
    pfVar7 = pfVar7 + 1;
    lVar9 = lVar9 + -1;
  } while (lVar9 != 0);
  return;
}

