
void FUN_1424cbe18(void)

{
  undefined8 *puVar1;
  undefined8 *puVar2;
  void *_Dst;
  size_t _Size;
  
  _Dst = (void *)CONCAT71(DAT_1451738a0._1_7_,(byte)DAT_1451738a0);
  if ((DAT_1451738b8 != '\0') && ((DAT_1451738a8 - (longlong)_Dst & 0xfffffffffffffff0U) != 0x1fff0)
     ) {
    FUN_1411b3d34(&DAT_1451738a0);
    _Dst = (void *)CONCAT71(DAT_1451738a0._1_7_,(byte)DAT_1451738a0);
  }
  _Size = DAT_1451738a8 - (longlong)_Dst & 0xfffffffffffffff0;
  if ((_Size < 0x21) && (((byte)DAT_1451738a0 & 3) == 0)) {
    if (_Size >> 2 == 0) goto LAB_1424cbe8e;
    _Size = (_Size >> 2) << 2;
  }
  memset(_Dst,0,_Size);
LAB_1424cbe8e:
  puVar1 = DAT_144de4b68;
  DAT_1451738b8 = DAT_145121140 == '\x01';
  DAT_144dbfc94 = 0xffffffff;
  for (puVar2 = DAT_144de4b60; puVar2 != puVar1; puVar2 = puVar2 + 2) {
    *puVar2 = 0;
    *(undefined2 *)(puVar2 + 1) = 0;
  }
  DAT_144dbfc98 = 0xffffffff;
  return;
}

