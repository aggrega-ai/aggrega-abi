// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

//! Cross-language wire-parity tests for the generated socket codecs.
//!
//! The goldens under `testdata/` are the canonical wire bytes — the same
//! goldens the Go and TypeScript codecs validate against, so a byte-match here
//! is a cross-language parity assertion: the three codecs are generated from one
//! schema and produce one wire. Each golden carries the determinism-sensitive
//! cases (sorted map keys, verbatim opaque carry, omitted empty fields), so a
//! codec regression breaks the byte-compare loudly.
//!
//! The contract proven per payload: deserialize the golden through the generated
//! type, re-serialize, and assert byte-identical output.

use aggrega_abi::{
    AssembleBoundaryInput, AssembleBoundaryOutput, CLIRunInput, FRPStepInput, FRPStepOutput,
    IRGraph, LinkPublishInput, LinkPublishOutput, LinkResolveInput, LinkResolveOutput, LinterInput,
    LinterOutput, MutatorInput, MutatorOutput, PolicyValidatorInput, PolicyValidatorOutput,
    RenderNodeInput, RenderNodeOutput, SearchInput, SearchOutput,
};
use serde::{de::DeserializeOwned, Serialize};
use std::fs;
use std::path::Path;

fn golden(name: &str) -> String {
    let p = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("testdata")
        .join(name);
    fs::read_to_string(&p)
        .unwrap_or_else(|e| panic!("read golden {}: {}", p.display(), e))
        .trim()
        .to_string()
}

fn round_trip<T: Serialize + DeserializeOwned>(name: &str) {
    let want = golden(name);
    let v: T = serde_json::from_str(&want).expect("deserialize golden");
    let got = serde_json::to_string(&v).expect("serialize");
    assert_eq!(want, got, "{name} wire not byte-identical to golden");
}

#[test]
fn cli_run_input_parity() {
    round_trip::<CLIRunInput>("CLIRunInput.golden.json");
}

#[test]
fn frp_step_input_parity() {
    round_trip::<FRPStepInput>("FRPStepInput.golden.json");
}

#[test]
fn frp_step_output_parity() {
    round_trip::<FRPStepOutput>("FRPStepOutput.golden.json");
}

#[test]
fn search_input_parity() {
    round_trip::<SearchInput>("SearchInput.golden.json");
}

#[test]
fn search_output_parity() {
    round_trip::<SearchOutput>("SearchOutput.golden.json");
}

#[test]
fn search_output_int_score_parity() {
    round_trip::<SearchOutput>("SearchOutputIntScore.golden.json");
}

#[test]
fn mutator_input_parity() {
    round_trip::<MutatorInput>("MutatorInput.golden.json");
}

#[test]
fn mutator_output_parity() {
    round_trip::<MutatorOutput>("MutatorOutput.golden.json");
}

#[test]
fn linter_input_parity() {
    round_trip::<LinterInput>("LinterInput.golden.json");
}

#[test]
fn linter_output_parity() {
    round_trip::<LinterOutput>("LinterOutput.golden.json");
}

#[test]
fn ir_graph_parity() {
    round_trip::<IRGraph>("IRGraph.golden.json");
}

#[test]
fn policy_validator_input_parity() {
    round_trip::<PolicyValidatorInput>("PolicyValidatorInput.golden.json");
}

#[test]
fn policy_validator_output_parity() {
    round_trip::<PolicyValidatorOutput>("PolicyValidatorOutput.golden.json");
}

#[test]
fn render_node_input_parity() {
    round_trip::<RenderNodeInput>("RenderNodeInput.golden.json");
}

#[test]
fn render_node_output_parity() {
    round_trip::<RenderNodeOutput>("RenderNodeOutput.golden.json");
}

#[test]
fn assemble_boundary_input_parity() {
    round_trip::<AssembleBoundaryInput>("AssembleBoundaryInput.golden.json");
}

#[test]
fn assemble_boundary_output_parity() {
    round_trip::<AssembleBoundaryOutput>("AssembleBoundaryOutput.golden.json");
}

#[test]
fn link_publish_input_parity() {
    round_trip::<LinkPublishInput>("LinkPublishInput.golden.json");
}

#[test]
fn link_publish_output_parity() {
    round_trip::<LinkPublishOutput>("LinkPublishOutput.golden.json");
}

#[test]
fn link_resolve_input_parity() {
    round_trip::<LinkResolveInput>("LinkResolveInput.golden.json");
}

#[test]
fn link_resolve_output_parity() {
    round_trip::<LinkResolveOutput>("LinkResolveOutput.golden.json");
}
