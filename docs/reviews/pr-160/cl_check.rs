use std::{env, fs};
use types::{BeaconState, Epoch, EthSpec, ForkName, MainnetEthSpec, Slot};

fn main() {
    let args: Vec<_> = env::args().collect();
    let oracle: serde_json::Value = serde_json::from_slice(&fs::read(&args[1]).unwrap()).unwrap();
    let bytes = fs::read(&args[2]).unwrap();
    let spec = ForkName::Gloas.make_genesis_spec(MainnetEthSpec::default_spec());
    let original = BeaconState::<MainnetEthSpec>::from_ssz_bytes(&bytes, &spec).unwrap();
    assert_eq!(original.validators().len(), 384);
    let lookahead: Vec<usize> = oracle["lookahead"].as_array().unwrap().iter().map(|x| x.as_u64().unwrap() as usize).collect();
    assert!(original.get_beacon_proposer_indices(Epoch::new(original.current_epoch().as_u64() + 2), &spec).is_err());
    for case in oracle["proposer_cases"].as_array().unwrap() {
        let mut state = original.clone();
        *state.slot_mut() = Slot::new(oracle["decision_slot"].as_u64().unwrap());
        let stride = case["slash_stride"].as_u64().unwrap() as usize;
        if stride > 0 {
            for index in (0..state.validators().len()).step_by(stride) {
                state.get_validator_mut(index).unwrap().slashed = true;
            }
        }
        state.build_slashings_cache().unwrap();
        let future = Epoch::new(case["epoch"].as_u64().unwrap());
        let got = state.get_beacon_proposer_indices(future, &spec).unwrap();
        let want: Vec<_> = case["expected"].as_array().unwrap().iter().map(|x| x.as_u64().unwrap() as usize).collect();
        assert_eq!(got, want, "full independently selected proposer vector, stride {stride}");
        assert!(got.iter().all(|&index| !state.get_validator(index).unwrap().slashed));
        let current = state.current_epoch();
        assert_eq!(state.get_beacon_proposer_indices(current, &spec).unwrap(), lookahead[..32]);
        assert_eq!(state.get_beacon_proposer_indices(Epoch::new(current.as_u64() + 1), &spec).unwrap(), lookahead[32..]);
    }
    for case in oracle["churn_cases"].as_array().unwrap() {
        let mut state = original.clone();
        state.set_total_active_balance(state.current_epoch(), case["total"].as_u64().unwrap(), &spec);
        assert_eq!(state.get_activation_exit_churn_limit(&spec).unwrap(), case["activation"].as_u64().unwrap());
        assert_eq!(state.get_exit_churn_limit(&spec).unwrap(), case["exit"].as_u64().unwrap());
        assert_eq!(state.get_consolidation_churn_limit(&spec).unwrap(), case["consolidation"].as_u64().unwrap());
        assert_eq!(state.compute_weak_subjectivity_period(&spec).unwrap().as_u64(), case["weak_subjectivity"].as_u64().unwrap());
    }
    println!("{}", serde_json::json!({"passed": true, "proposer_cases": 3, "complete_proposer_indices": 96,
        "cached_lookahead_preserved_cases": 3, "churn_and_weak_subjectivity_cases": 10,
        "churn_scope": "synthetic aggregate cache inputs to unchanged actual Lighthouse helpers"}));
}
