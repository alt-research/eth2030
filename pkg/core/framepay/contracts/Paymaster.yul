// SPDX-License-Identifier: CC0-1.0
// ETH2030 experimental paymaster v1, derived from EIP-8141 CanonicalPaymaster.
// Single secp256k1 owner; signature[1] must authorize the canonical tx hash.
// Storage: owner=0, pendingWithdrawalTo=1, amount=2, readyAt=3.
object "ETH2030PaymasterV1" {
 code {
  codecopy(0, sub(codesize(), 32), 32)
  let owner := mload(0)
  if or(iszero(owner), shr(160, owner)) { revert(0, 0) }
  sstore(0, owner)
  datacopy(0, dataoffset("Runtime"), datasize("Runtime"))
  return(0, datasize("Runtime"))
 }
 object "Runtime" {
  code {
   function requireOwner() {
    if iszero(eq(caller(), sload(0))) { revert(0, 0) }
   }
   function txparam(param) -> value {
    value := verbatim_1i_1o(hex"b0", param)
   }
   function sigparam(index, param) -> value {
    value := verbatim_2i_1o(hex"b4", index, param)
   }
   function frameparam(index, param) -> value {
    value := verbatim_2i_1o(hex"b3", index, param)
   }
   function returnWord(value) { mstore(0, value) return(0, 32) }
   // Empty calldata is the normal payable receive entrypoint.
   if iszero(calldatasize()) { stop() }
   if callvalue() { revert(0, 0) }
   if lt(calldatasize(), 4) { revert(0, 0) }
   switch shr(224, calldataload(0))
   case 0x634b01e8 {
    if iszero(eq(calldatasize(), 4)) { revert(0, 0) }
    if iszero(eq(txparam(0), 6)) { revert(0, 0) }
    let index := txparam(10)
    if iszero(eq(frameparam(index, 2), 1)) { revert(0, 0) }
    if iszero(eq(frameparam(index, 3), 1)) { revert(0, 0) }
    if iszero(eq(sigparam(1, 1), 1)) { revert(0, 0) }
    if sigparam(1, 2) { revert(0, 0) }
    if iszero(eq(sigparam(1, 0), sload(0))) { revert(0, 0) }
    // Return-like APPROVE(payment) with empty return data.
    verbatim_0i_0o(hex"600160006000aa")
    stop()
   }
   case 0x8da5cb5b { returnWord(sload(0)) }
   case 0x0ebb172a { returnWord(43200) }
   case 0xb21ab01f { returnWord(sload(1)) }
   case 0x2c5b1066 { returnWord(sload(2)) }
   case 0x9d060f01 { returnWord(sload(3)) }
   case 0xda95ebf7 {
    if iszero(eq(calldatasize(), 68)) { revert(0, 0) }
    requireOwner()
    let to := calldataload(4)
    if or(iszero(to), shr(160, to)) { revert(0, 0) }
    let amount := calldataload(36)
    let readyAt := add(timestamp(), 43200)
    sstore(1, to)
    sstore(2, amount)
    sstore(3, readyAt)
    mstore(0, amount)
    mstore(32, readyAt)
    log2(0, 64, 0x24b91f4f47caf44230a57777a9be744924e82bf666f2d5702faf97df35e60f9f, to)
    stop()
   }
   case 0xfa45aa00 {
    if iszero(eq(calldatasize(), 4)) { revert(0, 0) }
    requireOwner()
    let to := sload(1)
    let amount := sload(2)
    let readyAt := sload(3)
    if or(iszero(readyAt), lt(timestamp(), readyAt)) { revert(0, 0) }
    sstore(1, 0)
    sstore(2, 0)
    sstore(3, 0)
    if iszero(call(gas(), to, amount, 0, 0, 0, 0)) { revert(0, 0) }
    mstore(0, amount)
    log2(0, 32, 0x1131c2afe9ce72c59360d085f89bfdd088fcf4a3c93eec6cc5ff42f5d6b0801f, to)
    stop()
   }
   default { revert(0, 0) }
  }
 }
}
