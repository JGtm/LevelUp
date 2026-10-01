
void FUN_1423b456a(void)

{
  uint *puVar1;
  longlong lVar2;
  undefined8 *puVar3;
  longlong unaff_RBX;
  longlong unaff_RBP;
  
  FUN_141f85f0c(*(longlong *)(unaff_RBX + 8) + 0x42d0);
  FUN_141f85a84();
  DAT_144e61e98 = &stack0x00000030;
  lVar2 = *(longlong *)(*(longlong *)ThreadLocalStoragePointer + 0x4c8);
  puVar1 = (uint *)(lVar2 + 0x180e0);
  *puVar1 = *puVar1 | 1;
  *(uint *)(lVar2 + 0x1040) = *(uint *)(lVar2 + 0x1040) | 1;
  FUN_1405f3b08(&stack0x00000030);
  if (0 < *(int *)(unaff_RBP + 0x3d24)) {
    for (puVar3 = (undefined8 *)FUN_141f86518(unaff_RBP + 0x3d20); puVar3 != (undefined8 *)0x0;
        puVar3 = (undefined8 *)FUN_141f865a4(unaff_RBP + 0x3d20,puVar3)) {
      FUN_141f86238(*puVar3);
    }
  }
  FUN_14059dc7c(puVar1);
  *puVar1 = *puVar1 & 0xfffffffe;
  FUN_140671ee4(lVar2);
  FUN_140a17be4(lVar2);
  FUN_140b87b84(&stack0x00000030);
  FUN_140bea558(unaff_RBP + 0x3cf8);
  FUN_140bea51c(unaff_RBP + 0x3d20);
  return;
}

