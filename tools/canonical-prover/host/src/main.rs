use base64::{engine::general_purpose::STANDARD, Engine};
use bincode::Options;
use eth2030_canonical_methods::{CANONICAL_GUEST_ELF, CANONICAL_GUEST_ID};
use eth2030_canonical_protocol::{execute, ExecutionRequest, MachineConfig, Statement};
use risc0_zkvm::{ExecutorEnv, ExternalProver, InnerReceipt, Prover, Receipt};
use serde::{Deserialize, Serialize};
use std::io::{self, Read};

const MAX_COMMAND: u64 = 32 << 20;
const MAX_RECEIPT: u64 = 16 << 20;

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
struct Envelope {
    version: u32,
    image_id: String,
    receipt: String,
    statement: Statement,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
struct Command {
    action: String,
    program: Option<String>,
    input: Option<String>,
    config: Option<MachineConfig>,
    proof: Option<Envelope>,
    expected: Option<Statement>,
}

fn image_id() -> String {
    hex::encode(
        CANONICAL_GUEST_ID
            .iter()
            .flat_map(|v| v.to_le_bytes())
            .collect::<Vec<_>>(),
    )
}

fn verify(envelope: &Envelope) -> Result<(), Box<dyn std::error::Error>> {
    if envelope.version != 1 || envelope.image_id != image_id() {
        return Err("unsupported proof image/version".into());
    }
    let bytes = STANDARD.decode(&envelope.receipt)?;
    if bytes.len() as u64 > MAX_RECEIPT {
        return Err("receipt too large".into());
    }
    let receipt: Receipt = bincode::DefaultOptions::new()
        .with_fixint_encoding()
        .with_limit(MAX_RECEIPT)
        .reject_trailing_bytes()
        .deserialize(&bytes)?;
    // Fake receipts are never admissible, even if a developer environment has
    // RISC0_DEV_MODE set. The expected image is compiled into this verifier.
    if matches!(receipt.inner, InnerReceipt::Fake(_)) {
        return Err("fake receipt rejected".into());
    }
    receipt.verify(CANONICAL_GUEST_ID)?;
    let statement: Statement = receipt.journal.decode()?;
    if statement != envelope.statement {
        return Err("journal/statement mismatch".into());
    }
    Ok(())
}

fn run(command: Command) -> Result<serde_json::Value, Box<dyn std::error::Error>> {
    match command.action.as_str() {
        "metadata" => Ok(
            serde_json::json!({"system":"risc0", "sdk":"3.1.0", "version":1, "image_id":image_id()}),
        ),
        "verify" => {
            let proof = command.proof.ok_or("missing proof")?;
            verify(&proof)?;
            if command.expected.as_ref() != Some(&proof.statement) {
                return Err("unexpected execution statement".into());
            }
            Ok(serde_json::json!({"valid":true,"statement":proof.statement}))
        }
        "execute" | "prove" => {
            let request = ExecutionRequest {
                program: hex::decode(command.program.ok_or("missing program")?)?,
                input: hex::decode(command.input.ok_or("missing input")?)?,
                config: command.config.ok_or("missing config")?,
            };
            let expected = execute(&request)?;
            if command.action == "execute" {
                return Ok(serde_json::to_value(expected)?);
            }
            // Force real proving. A locally set dev-mode variable must never
            // manufacture an apparently successful production proof.
            std::env::set_var("RISC0_DEV_MODE", "0");
            let sibling = std::env::current_exe()?.with_file_name("r0vm");
            let server = if sibling.is_file() {
                sibling
            } else {
                "r0vm".into()
            };
            let env = ExecutorEnv::builder()
                .write(&request)?
                .session_limit(Some(1 << 30))
                .build()?;
            // Explicit local IPC prover: ambient cloud credentials or SDK prover
            // selectors cannot redirect a node request to a paid remote service.
            let receipt = ExternalProver::new("eth2030-local", server)
                .prove(env, CANONICAL_GUEST_ELF)?
                .receipt;
            let proof = Envelope {
                version: 1,
                image_id: image_id(),
                receipt: STANDARD.encode(bincode::serialize(&receipt)?),
                statement: expected,
            };
            verify(&proof)?;
            Ok(serde_json::to_value(proof)?)
        }
        _ => Err("unknown command".into()),
    }
}

fn main() {
    let result = (|| {
        let mut input = Vec::new();
        io::stdin().take(MAX_COMMAND + 1).read_to_end(&mut input)?;
        if input.len() as u64 > MAX_COMMAND {
            return Err("command too large".into());
        }
        run(serde_json::from_slice(&input)?)
    })();
    match result {
        Ok(value) => println!("{}", value),
        Err(error) => {
            eprintln!("canonical prover: {error}");
            std::process::exit(1);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use risc0_zkvm::{FakeReceipt, ReceiptClaim};

    fn fixture() -> Envelope {
        let bytes =
            include_bytes!("../../../../pkg/zkvm/canonicalproof/testdata/risc0-echo-v1.json.gz");
        let decoder = flate2::read::GzDecoder::new(bytes.as_slice());
        serde_json::from_reader(decoder).unwrap()
    }

    #[test]
    fn independently_verify_real_receipt_and_reject_modified_seal() {
        let mut proof = fixture();
        verify(&proof).unwrap();
        let mut receipt: Receipt =
            bincode::deserialize(&STANDARD.decode(&proof.receipt).unwrap()).unwrap();
        match &mut receipt.inner {
            InnerReceipt::Composite(composite) => composite.segments[0].seal[0] ^= 1,
            _ => panic!("fixture must retain its actual composite STARK receipt"),
        }
        proof.receipt = STANDARD.encode(bincode::serialize(&receipt).unwrap());
        assert!(verify(&proof).is_err());
    }

    #[test]
    fn reject_sdk_fake_receipt_even_when_developer_mode_is_enabled() {
        let mut proof = fixture();
        let real: Receipt =
            bincode::deserialize(&STANDARD.decode(&proof.receipt).unwrap()).unwrap();
        let fake = Receipt::new(
            InnerReceipt::Fake(FakeReceipt::new(ReceiptClaim::ok(
                CANONICAL_GUEST_ID,
                real.journal.bytes.clone(),
            ))),
            real.journal.bytes,
        );
        std::env::set_var("RISC0_DEV_MODE", "1");
        proof.receipt = STANDARD.encode(bincode::serialize(&fake).unwrap());
        assert!(verify(&proof).is_err());
        std::env::remove_var("RISC0_DEV_MODE");
    }
}
