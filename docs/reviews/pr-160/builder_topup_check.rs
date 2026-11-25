use std::{env, fs};
use types::{BeaconState, BuilderDepositRequest, EthSpec, ForkName, Hash256, MainnetEthSpec};
use state_processing::per_block_processing::process_operations::process_builder_deposit_requests;

fn main() {
    let args: Vec<_> = env::args().collect();
    let bytes = fs::read(&args[1]).unwrap();
    let spec = ForkName::Gloas.make_genesis_spec(MainnetEthSpec::default_spec());
    let mut cases = vec![];
    for prefix in [0xb0u8, 0xffu8] {
        let mut state = BeaconState::<MainnetEthSpec>::from_ssz_bytes(&bytes, &spec).unwrap();
        let builder = state.get_builder(0).unwrap();
        let before = builder.balance;
        let recipient = builder.execution_address;
        let mut credentials = [0u8; 32];
        credentials[0] = prefix;
        let request = BuilderDepositRequest {
            pubkey: builder.pubkey,
            withdrawal_credentials: Hash256::from(credentials),
            amount: 1_000_000_000,
            signature: bls::SignatureBytes::empty(),
        };
        process_builder_deposit_requests(&mut state, &[request], &spec).unwrap();
        let after = state.get_builder(0).unwrap();
        let credited = after.balance - before;
        assert_eq!(credited, if prefix == 0xb0 { 1_000_000_000 } else { 0 });
        assert_eq!(after.execution_address, recipient);
        cases.push(serde_json::json!({"prefix":prefix,"before":before,"after":after.balance,
            "credited":credited,"execution_address_unchanged":true}));
    }
    println!("{}", serde_json::json!({"client_matches_detailed_consensus_reference":true,
        "topup_prose_and_prefix_precondition_need_clarification":true,"cases":cases}));
}
