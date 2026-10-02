
undefined8 * FUN_141137dec(void)

{
  void *_Dst;
  
  _Dst = (void *)FUN_140689b88(0x9900,0x65,
                               "D:\\Enlistment\\shared\\engine\\includes\\i343\\STLAllocator.h",
                               0x108);
  if (_Dst == (void *)0x0) {
    _Dst = (void *)0x0;
  }
  else {
    memset(_Dst,0,0x9900);
    FUN_14047bdd0(_Dst,0x4c8,0x20,FUN_141045e6c);
  }
  DAT_145178b58 = _Dst;
  return &DAT_145178b58;
}

