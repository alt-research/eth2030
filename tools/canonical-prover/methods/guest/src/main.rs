use eth2030_canonical_protocol::{execute, ExecutionRequest};
use risc0_zkvm::guest::env;

fn main() {
    let request: ExecutionRequest = env::read();
    let statement = execute(&request).expect("canonical guest execution failed");
    env::commit(&statement);
}
