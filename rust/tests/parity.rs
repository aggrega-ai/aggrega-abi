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
    ImportInput, ImportOutput, IRGraph, LinkPublishInput, LinkPublishOutput, LinkResolveInput,
    LinkResolveOutput, LinterInput, LinterOutput, MutatorInput, MutatorOutput, ParseInput,
    ParseOutput, PluginDescriptor, PolicyInput, PolicyOutput, RenderNodeInput, RenderNodeOutput,
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
fn policy_input_parity() {
    round_trip::<PolicyInput>("PolicyInput.golden.json");
}

#[test]
fn policy_output_parity() {
    round_trip::<PolicyOutput>("PolicyOutput.golden.json");
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

#[test]
fn plugin_descriptor_cli_parity() {
    round_trip::<PluginDescriptor>("PluginDescriptorCLI.golden.json");
}

#[test]
fn plugin_descriptor_provider_parity() {
    round_trip::<PluginDescriptor>("PluginDescriptorProvider.golden.json");
}

// A descriptor declaring none of the optional nested objects: the codec must omit
// every absent optional rather than emit an empty placeholder.
#[test]
fn plugin_descriptor_minimal_parity() {
    round_trip::<PluginDescriptor>("PluginDescriptorMinimal.golden.json");
}

// The wire a real CLI plugin (moo) produces: no contract, metadata present.
#[test]
fn plugin_descriptor_moo_parity() {
    round_trip::<PluginDescriptor>("PluginDescriptorMoo.golden.json");
}

#[test]
fn parse_input_parity() {
    round_trip::<ParseInput>("ParseInput.golden.json");
}

#[test]
fn parse_output_parity() {
    round_trip::<ParseOutput>("ParseOutput.golden.json");
}

// A parser output that omits every optional (no diagnostics, no spans) and
// carries a top-level-array tree: the codec must omit absent optionals and
// carry a non-object tree verbatim.
#[test]
fn parse_output_bare_parity() {
    round_trip::<ParseOutput>("ParseOutputBare.golden.json");
}

#[test]
fn import_input_parity() {
    round_trip::<ImportInput>("ImportInput.golden.json");
}

#[test]
fn import_output_parity() {
    round_trip::<ImportOutput>("ImportOutput.golden.json");
}
