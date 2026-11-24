//! The versioned RV32IM execution statement proved by the fixed zkVM image.
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::collections::BTreeMap;

pub const MAX_PROGRAM: usize = 1 << 20;
pub const MAX_INPUT: usize = 1 << 20;
pub const MAX_OUTPUT: usize = 1 << 20;
pub const MAX_CYCLES: u64 = 1 << 24;
const PAGE_SIZE: usize = 4096;

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct MachineConfig {
    pub max_cycles: u64,
    pub memory_pages: u32,
    pub program_base: u32,
    pub entry_pc: u32,
    pub stack_base: u32,
    pub input_base: u32,
    pub load_input: bool,
}

impl MachineConfig {
    pub fn validate(&self) -> Result<(), String> {
        if self.max_cycles == 0 || self.max_cycles > MAX_CYCLES {
            return Err("invalid cycle bound".into());
        }
        if self.memory_pages == 0 || self.memory_pages > 16384 {
            return Err("invalid page bound".into());
        }
        if self.program_base >= 0xf0000000 || self.entry_pc & 3 != 0 {
            return Err("invalid program address".into());
        }
        Ok(())
    }
}

#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(deny_unknown_fields)]
pub struct ExecutionRequest {
    pub program: Vec<u8>,
    pub input: Vec<u8>,
    pub config: MachineConfig,
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, Eq)]
#[serde(deny_unknown_fields)]
pub struct Statement {
    pub version: u32,
    pub program_hash: String,
    pub input_hash: String,
    pub output: String,
    pub output_hash: String,
    pub cycles: u64,
    pub exit_code: u32,
    pub final_pc: u32,
    pub input_consumed: u64,
    pub state_hash: String,
    pub config: MachineConfig,
}

pub fn digest(bytes: &[u8]) -> String {
    hex::encode(Sha256::digest(bytes))
}

struct Memory {
    pages: BTreeMap<u32, Box<[u8; PAGE_SIZE]>>,
    limit: u32,
}

impl Memory {
    fn page(&mut self, address: u32) -> Result<&mut [u8; PAGE_SIZE], String> {
        let index = address >> 12;
        if !self.pages.contains_key(&index) {
            if self.pages.len() >= self.limit as usize {
                return Err("page limit exceeded".into());
            }
            self.pages.insert(index, Box::new([0; PAGE_SIZE]));
        }
        Ok(self.pages.get_mut(&index).unwrap())
    }

    fn read(&mut self, address: u32, size: u32) -> Result<u32, String> {
        let mut value = 0;
        for i in 0..size {
            let at = address.wrapping_add(i);
            value |= (self.page(at)?[(at & 4095) as usize] as u32) << (i * 8);
        }
        Ok(value)
    }

    fn write(&mut self, address: u32, value: u32, size: u32) -> Result<(), String> {
        // The adopted guest ABI has no external MMIO callbacks. Its high-address
        // writes are no-ops, matching the standalone Go interpreter.
        if address >= 0xf0000000 {
            return Ok(());
        }
        for i in 0..size {
            let at = address.wrapping_add(i);
            if at < 0xf0000000 {
                self.page(at)?[(at & 4095) as usize] = (value >> (i * 8)) as u8;
            }
        }
        Ok(())
    }

    fn load(&mut self, base: u32, bytes: &[u8]) -> Result<(), String> {
        if base as u64 + bytes.len() as u64 > 1u64 << 32 {
            return Err("segment address overflow".into());
        }
        for (i, value) in bytes.iter().enumerate() {
            self.write(base.wrapping_add(i as u32), *value as u32, 1)?;
        }
        Ok(())
    }
}

fn sext(value: u32, bits: u32) -> u32 {
    ((value << (32 - bits)) as i32 >> (32 - bits)) as u32
}

/// Execute the complete integer RV32IM guest ABI. No host-provided execution
/// result or trace is accepted: the fixed zkVM image executes every instruction.
pub fn execute(request: &ExecutionRequest) -> Result<Statement, String> {
    request.config.validate()?;
    let cfg = &request.config;
    if request.program.is_empty()
        || request.program.len() > MAX_PROGRAM
        || request.input.len() > MAX_INPUT
    {
        return Err("invalid program/input length".into());
    }
    let mut memory = Memory {
        pages: BTreeMap::new(),
        limit: cfg.memory_pages,
    };
    memory.load(cfg.program_base, &request.program)?;
    let mut registers = [0u32; 32];
    registers[2] = cfg.stack_base;
    if cfg.load_input {
        if !request.input.is_empty() {
            memory.load(cfg.input_base, &request.input)?;
        }
        registers[10] = cfg.input_base;
        registers[11] = request.input.len() as u32;
    }
    let mut pc = cfg.entry_pc;
    let mut output = Vec::new();
    let mut input_pos = 0usize;
    let mut cycles = 0u64;
    let mut halted = false;
    let mut exit_code = 0;
    while !halted {
        if cycles >= cfg.max_cycles {
            return Err("cycle limit exceeded".into());
        }
        if pc & 3 != 0 {
            return Err("unaligned instruction address".into());
        }
        let instruction = memory.read(pc, 4)?;
        let opcode = instruction & 127;
        let rd = ((instruction >> 7) & 31) as usize;
        let f3 = (instruction >> 12) & 7;
        let rs1 = ((instruction >> 15) & 31) as usize;
        let rs2 = ((instruction >> 20) & 31) as usize;
        let f7 = instruction >> 25;
        let a = registers[rs1];
        let b = registers[rs2];
        let imm = sext(instruction >> 20, 12);
        let mut next_pc = pc.wrapping_add(4);
        match opcode {
            0x37 => registers[rd] = instruction & 0xfffff000,
            0x17 => registers[rd] = pc.wrapping_add(instruction & 0xfffff000),
            0x6f => {
                let offset = sext(
                    ((instruction >> 31) << 20)
                        | (((instruction >> 12) & 255) << 12)
                        | (((instruction >> 20) & 1) << 11)
                        | (((instruction >> 21) & 1023) << 1),
                    21,
                );
                registers[rd] = next_pc;
                next_pc = pc.wrapping_add(offset);
            }
            0x67 if f3 == 0 => {
                registers[rd] = next_pc;
                next_pc = a.wrapping_add(imm) & !1;
            }
            0x63 => {
                let offset = sext(
                    ((instruction >> 31) << 12)
                        | (((instruction >> 7) & 1) << 11)
                        | (((instruction >> 25) & 63) << 5)
                        | (((instruction >> 8) & 15) << 1),
                    13,
                );
                let take = match f3 {
                    0 => a == b,
                    1 => a != b,
                    4 => (a as i32) < (b as i32),
                    5 => (a as i32) >= (b as i32),
                    6 => a < b,
                    7 => a >= b,
                    _ => return Err("invalid branch".into()),
                };
                if take {
                    next_pc = pc.wrapping_add(offset);
                }
            }
            0x03 => {
                let at = a.wrapping_add(imm);
                registers[rd] = match f3 {
                    0 => sext(memory.read(at, 1)?, 8),
                    1 => sext(memory.read(at, 2)?, 16),
                    2 => memory.read(at, 4)?,
                    4 => memory.read(at, 1)?,
                    5 => memory.read(at, 2)?,
                    _ => return Err("invalid load".into()),
                };
            }
            0x23 => {
                let offset = sext(((instruction >> 25) << 5) | ((instruction >> 7) & 31), 12);
                let size = match f3 {
                    0 => 1,
                    1 => 2,
                    2 => 4,
                    _ => return Err("invalid store".into()),
                };
                memory.write(a.wrapping_add(offset), b, size)?;
            }
            0x13 => {
                registers[rd] = match f3 {
                    0 => a.wrapping_add(imm),
                    2 => ((a as i32) < (imm as i32)) as u32,
                    3 => (a < imm) as u32,
                    4 => a ^ imm,
                    6 => a | imm,
                    7 => a & imm,
                    1 if f7 == 0 => a << (rs2 as u32),
                    5 if f7 == 0 => a >> (rs2 as u32),
                    5 if f7 == 0x20 => ((a as i32) >> (rs2 as u32)) as u32,
                    _ => return Err("invalid immediate instruction".into()),
                };
            }
            0x33 if f7 == 1 => {
                registers[rd] = match f3 {
                    0 => a.wrapping_mul(b),
                    1 => (((a as i32 as i64) * (b as i32 as i64)) >> 32) as u32,
                    2 => (((a as i32 as i64) * (b as i64)) >> 32) as u32,
                    3 => (((a as u64) * (b as u64)) >> 32) as u32,
                    4 => {
                        if b == 0 {
                            u32::MAX
                        } else {
                            (a as i32).wrapping_div(b as i32) as u32
                        }
                    }
                    5 => {
                        if b == 0 {
                            u32::MAX
                        } else {
                            a / b
                        }
                    }
                    6 => {
                        if b == 0 {
                            a
                        } else {
                            (a as i32).wrapping_rem(b as i32) as u32
                        }
                    }
                    7 => {
                        if b == 0 {
                            a
                        } else {
                            a % b
                        }
                    }
                    _ => unreachable!(),
                };
            }
            0x33 => {
                registers[rd] = match (f7, f3) {
                    (0, 0) => a.wrapping_add(b),
                    (0x20, 0) => a.wrapping_sub(b),
                    (0, 1) => a << (b & 31),
                    (0, 2) => ((a as i32) < (b as i32)) as u32,
                    (0, 3) => (a < b) as u32,
                    (0, 4) => a ^ b,
                    (0, 5) => a >> (b & 31),
                    (0x20, 5) => ((a as i32) >> (b & 31)) as u32,
                    (0, 6) => a | b,
                    (0, 7) => a & b,
                    _ => return Err("invalid register instruction".into()),
                };
            }
            0x0f if f3 == 0 => {}
            0x73 if instruction == 0x73 => match registers[17] {
                0 => {
                    exit_code = registers[10];
                    halted = true;
                }
                1 => {
                    if output.len() >= MAX_OUTPUT {
                        return Err("output limit exceeded".into());
                    }
                    output.push(registers[10] as u8);
                }
                2 => {
                    registers[10] = if input_pos < request.input.len() {
                        let value = request.input[input_pos] as u32;
                        input_pos += 1;
                        value
                    } else {
                        u32::MAX
                    };
                }
                _ => return Err("unsupported syscall".into()),
            },
            0x73 if instruction == 0x00100073 => halted = true,
            _ => return Err("invalid RV32IM instruction".into()),
        }
        registers[0] = 0;
        pc = next_pc;
        cycles += 1;
    }
    if exit_code != 0 {
        return Err("nonzero guest exit code".into());
    }
    let mut state = Sha256::new();
    state.update(b"ETH2030_CANONICAL_RV32_STATE_V1");
    state.update(pc.to_le_bytes());
    state.update(cycles.to_le_bytes());
    state.update((input_pos as u64).to_le_bytes());
    for value in registers {
        state.update(value.to_le_bytes());
    }
    state.update((memory.pages.len() as u32).to_le_bytes());
    for (index, page) in memory.pages {
        state.update(index.to_le_bytes());
        state.update(page.as_slice());
    }
    Ok(Statement {
        version: 1,
        program_hash: digest(&request.program),
        input_hash: digest(&request.input),
        output: hex::encode(&output),
        output_hash: digest(&output),
        cycles,
        exit_code,
        final_pc: pc,
        input_consumed: input_pos as u64,
        state_hash: hex::encode(state.finalize()),
        config: cfg.clone(),
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    fn cfg() -> MachineConfig {
        MachineConfig {
            max_cycles: 100,
            memory_pages: 4,
            program_base: 0,
            entry_pc: 0,
            stack_base: 0x80000000,
            input_base: 0,
            load_input: false,
        }
    }
    #[test]
    fn echo_and_bind_statement() {
        let words = [
            0x00200893u32,
            0x00000073,
            0x00100893,
            0x00000073,
            0x00000513,
            0x00000893,
            0x00000073,
        ];
        let program = words.iter().flat_map(|x| x.to_le_bytes()).collect();
        let request = ExecutionRequest {
            program,
            input: vec![42],
            config: cfg(),
        };
        let statement = execute(&request).unwrap();
        assert_eq!(statement.output, "2a");
        assert_eq!(statement.cycles, 7);
        assert_eq!(statement.input_consumed, 1);
        let mut changed = request.clone();
        changed.input[0] = 43;
        assert_ne!(execute(&changed).unwrap(), statement);
        changed.config.max_cycles = 1;
        assert!(execute(&changed).is_err());
    }
}
