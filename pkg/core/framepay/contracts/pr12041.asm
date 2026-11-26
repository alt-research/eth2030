;; CC0-1.0; ethereum/EIPs PR #12041 at 4a98fcefa5c6f02a25289bf55b9d92902d02b4c6.
;; Source: assets/eip-8141/canonical-paymaster.md (annotated assembly).
;; ---- dispatch ---------------------------------------------------
    callvalue                       ; value carried?
    push2 has_value
    jumpi                           ;   -> deposit / reject
    calldatasize                    ; any calldata?
    push2 admin
    jumpi                           ;   -> admin op dispatch
                                    ; else empty calldata + zero value = pay frame

;; ---- payment-validation gate (pay frame) ------------------------
    push1 0x01                      ; param = scheme
    push1 0x01                      ; signature index 1
    sigparam                        ; scheme(1)
    iszero                          ; scheme == ARBITRARY (0x0)?
    push2 fail
    jumpi                           ;   ARBITRARY -> revert
    push0                           ; param = resolved_signer
    push1 0x01
    sigparam                        ; resolved_signer(1)
    push0
    sload                           ; slot0 = signer
    eq
    iszero                          ; resolved_signer != signer?
    push2 fail
    jumpi                           ;   -> revert
    push1 0x02                      ; param = msg
    push1 0x01
    sigparam                        ; msg(1)
    push2 fail
    jumpi                           ;   non-empty msg -> revert
    push1 0x01                      ; scope = APPROVE_PAYMENT (0x1)
    push0                           ; length = 0
    push0                           ; offset = 0
    approve                         ; APPROVE(PAYMENT); exits frame successfully

;; ---- has_value: deposit iff no calldata, else reject ------------
has_value:
    jumpdest
    calldatasize
    push2 fail
    jumpi                           ;   value + calldata -> revert (admin non-payable)
    stop                            ;   plain deposit accepted

;; ---- admin: dispatch on first calldata byte ---------------------
admin:
    jumpdest
    push0
    calldataload
    push1 0xf8
    shr                             ; op = calldata[0]
    dup1
    push1 0x01
    eq
    push2 w_init
    jumpi                           ; op == 1 -> initiate withdrawal
    dup1
    push1 0x02
    eq
    push2 r_init
    jumpi                           ; op == 2 -> initiate rotation
    dup1
    push1 0x03
    eq
    push2 cancel
    jumpi                           ; op == 3 -> cancel
    push1 0x04
    eq
    push2 finalize
    jumpi                           ; op == 4 -> finalize
                                    ; fall through to fail on any other op

;; ---- fail: revert -----------------------------------------------
fail:
    jumpdest
    push0
    push0
    revert

;; ---- w_init: initiate withdrawal(amount = calldata[1:33]) -------
w_init:
    jumpdest
    pop                             ; drop dispatch op byte
    ;; authorized(): caller == signer, OR a signer-signed entry@1
    caller
    push0
    sload                           ; slot0
    eq
    push2 auth_ok_0
    jumpi                           ;   caller == signer -> authorized
    push1 0x01                      ; else signature route: scheme(1) != ARBITRARY
    push1 0x01
    sigparam
    iszero
    push2 fail
    jumpi
    push0                           ; resolved_signer(1) == signer
    push1 0x01
    sigparam
    push0
    sload
    eq
    iszero
    push2 fail
    jumpi
    push1 0x02                      ; msg(1) == 0
    push1 0x01
    sigparam
    push2 fail
    jumpi
auth_ok_0:
    jumpdest
    push1 0x02
    sload                           ; slot2
    push2 fail
    jumpi                           ;   action already pending -> revert
    push1 0x01
    calldataload                    ; amount = calldata[1:33]
    dup1
    iszero
    push2 fail
    jumpi                           ;   amount == 0 -> revert
    push1 0x01
    sstore                          ; slot1 = amount
    timestamp
    push3 0x015180                  ; DELAY = 86400
    add
    push1 0x02
    sstore                          ; slot2 = now + DELAY
    stop

;; ---- r_init: initiate rotation(new_signer = calldata[1:33]) -----
r_init:
    jumpdest
    pop
    caller                          ; authorized() (identical gate)
    push0
    sload
    eq
    push2 auth_ok_1
    jumpi
    push1 0x01
    push1 0x01
    sigparam
    iszero
    push2 fail
    jumpi
    push0
    push1 0x01
    sigparam
    push0
    sload
    eq
    iszero
    push2 fail
    jumpi
    push1 0x02
    push1 0x01
    sigparam
    push2 fail
    jumpi
auth_ok_1:
    jumpdest
    push1 0x02
    sload
    push2 fail
    jumpi                           ;   action already pending -> revert
    push1 0x01
    calldataload                    ; new_signer = calldata[1:33]
    dup1
    iszero
    push2 fail
    jumpi                           ;   new_signer == 0 -> revert
    push1 0x03
    sstore                          ; slot3 = new_signer
    timestamp
    push3 0x015180                  ; DELAY = 86400
    add
    push1 0x02
    sstore                          ; slot2 = now + DELAY
    stop

;; ---- cancel: clear pending state --------------------------------
cancel:
    jumpdest
    pop
    caller                          ; authorized() (identical gate)
    push0
    sload
    eq
    push2 auth_ok_2
    jumpi
    push1 0x01
    push1 0x01
    sigparam
    iszero
    push2 fail
    jumpi
    push0
    push1 0x01
    sigparam
    push0
    sload
    eq
    iszero
    push2 fail
    jumpi
    push1 0x02
    push1 0x01
    sigparam
    push2 fail
    jumpi
auth_ok_2:
    jumpdest
    push0
    push1 0x01
    sstore                          ; slot1 = 0
    push0
    push1 0x02
    sstore                          ; slot2 = 0
    push0
    push1 0x03
    sstore                          ; slot3 = 0
    stop

;; ---- finalize: anyone, once matured -----------------------------
finalize:
    jumpdest
    push1 0x02
    sload                           ; slot2 = unlock time
    dup1
    iszero
    push2 fail
    jumpi                           ;   no pending action -> revert
    timestamp
    lt                              ; now < unlock?
    push2 fail
    jumpi                           ;   not matured -> revert
    push1 0x01
    sload                           ; slot1 = amount
    dup1
    iszero
    push2 f_rot
    jumpi                           ;   amount == 0 -> rotation branch
    ;; withdrawal: clear slots 1 and 2 before the value-bearing call
    push0
    push1 0x01
    sstore                          ; slot1 = 0
    push0
    push1 0x02
    sstore                          ; slot2 = 0
    push0                           ; retLength
    push0                           ; retOffset
    push0                           ; argsLength
    push0                           ; argsOffset
    dup5                            ; value = amount
    push0
    sload                           ; to = signer (slot0)
    gas                             ; gas
    call                            ; CALL(gas, signer, amount, 0, 0, 0, 0)
    iszero
    push2 fail
    jumpi                           ;   failed send -> revert
    stop

;; ---- f_rot: finalize rotation -----------------------------------
f_rot:
    jumpdest
    pop                             ; drop amount (== 0)
    push1 0x03
    sload                           ; slot3 = new signer
    push0
    sstore                          ; slot0 = new signer
    push0
    push1 0x03
    sstore                          ; slot3 = 0
    push0
    push1 0x02
    sstore                          ; slot2 = 0
    stop
