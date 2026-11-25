# Status

Implemented and locally validated. The eth/72 mask now has the fixed CellMask type, encoding 16 zero bytes for ordinary transactions and rejecting every wrong-width RLP field during decode. Non-blob mask bits remain ignored; eth/71 retains its original three-field schema. Independent Geth SDK decoding and real-handler negatives failed on original b4b88ed0 and now pass.

Fresh checks: complete eth suite; complete eth race suite; real native-node TCP full-blob promotion, Engine cells 0/127 KZG verification and receiver restart/reacquisition; native and Geth-wrapper builds; diff check. EIP-8070 source 81a90efc, merged PR 12696 and its approved review are the authority. No EVM/activation change or new duration qualification is claimed. Master-target PR follows.
