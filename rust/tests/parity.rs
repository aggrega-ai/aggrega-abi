// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

//! Cross-language wire-parity tests for the generated socket codecs.
//!
//! The goldens under the shared `../testdata` tree are the canonical wire bytes
//! — the same tree the Go and TypeScript harnesses validate against, so a match
//! here is a cross-language parity assertion: the three codecs are generated from
//! one schema and produce one wire. Each vector carries a JSON document
//! (`<Name>.golden.json`) and the canonical CBOR bytes (`<Name>.cbor.hex`).
//!
//! Two contracts are proven per vector: the JSON wire round-trips byte-identical
//! through serde, and the generated CBOR encoder reproduces the committed
//! canonical bytes and decodes back to a byte-stable value.

use aggrega_abi::CandidateSignals;
use serde::{de::DeserializeOwned, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

fn testdata() -> PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR")).join("..").join("testdata")
}

fn golden_json(name: &str) -> String {
    let p = testdata().join(format!("{name}.golden.json"));
    fs::read_to_string(&p)
        .unwrap_or_else(|e| panic!("read golden {}: {}", p.display(), e))
        .trim()
        .to_string()
}

fn golden_cbor(name: &str) -> Vec<u8> {
    let p = testdata().join(format!("{name}.cbor.hex"));
    let h = fs::read_to_string(&p)
        .unwrap_or_else(|e| panic!("read cbor golden {}: {}", p.display(), e));
    hex_decode(h.trim())
}

fn hex_decode(s: &str) -> Vec<u8> {
    let b = s.as_bytes();
    assert!(b.len() % 2 == 0, "odd hex length");
    let nib = |c: u8| -> u8 {
        match c {
            b'0'..=b'9' => c - b'0',
            b'a'..=b'f' => c - b'a' + 10,
            b'A'..=b'F' => c - b'A' + 10,
            _ => panic!("bad hex byte {c}"),
        }
    };
    (0..b.len() / 2).map(|i| (nib(b[2 * i]) << 4) | nib(b[2 * i + 1])).collect()
}

fn hex_encode(b: &[u8]) -> String {
    let mut s = String::with_capacity(b.len() * 2);
    for x in b {
        s.push_str(&format!("{x:02x}"));
    }
    s
}

/// json_parity proves the JSON wire round-trips byte-identical through serde.
fn json_parity<T: Serialize + DeserializeOwned>(name: &str) -> T {
    let want = golden_json(name);
    let v: T = serde_json::from_str(&want).expect("deserialize golden");
    let got = serde_json::to_string(&v).expect("serialize");
    assert_eq!(want, got, "{name} JSON wire not byte-identical to golden");
    v
}

/// A generated type carrying the CBOR codec. Implemented per vector type via the
/// cbor_case macro so the harness can assert encode==golden and a byte-stable
/// round-trip generically.
trait CborCase: Sized {
    fn enc_cbor(&self) -> Vec<u8>;
    fn dec_cbor(data: &[u8]) -> Self;
}

macro_rules! cbor_case {
    ($fn:ident, $ty:ty, $name:literal) => {
        impl CborCase for $ty {
            fn enc_cbor(&self) -> Vec<u8> {
                self.encode_cbor().expect(concat!($name, " encode_cbor"))
            }
            fn dec_cbor(data: &[u8]) -> Self {
                <$ty>::decode_cbor(data).expect(concat!($name, " decode_cbor"))
            }
        }
        #[test]
        fn $fn() {
            let v: $ty = json_parity::<$ty>($name);
            let want = golden_cbor($name);
            let enc = v.enc_cbor();
            assert_eq!(
                hex_encode(&enc),
                hex_encode(&want),
                "{} CBOR not byte-identical to golden",
                $name
            );
            let rt = <$ty>::dec_cbor(&enc).enc_cbor();
            assert_eq!(hex_encode(&rt), hex_encode(&enc), "{} CBOR round-trip not stable", $name);
        }
    };
}

cbor_case!(cli_run_input, aggrega_abi::CLIRunInput, "CLIRunInput");
cbor_case!(frp_step_input, aggrega_abi::FRPStepInput, "FRPStepInput");
cbor_case!(frp_step_output, aggrega_abi::FRPStepOutput, "FRPStepOutput");
cbor_case!(mutator_input, aggrega_abi::MutatorInput, "MutatorInput");
cbor_case!(mutator_output, aggrega_abi::MutatorOutput, "MutatorOutput");
cbor_case!(linter_input, aggrega_abi::LinterInput, "LinterInput");
cbor_case!(linter_output, aggrega_abi::LinterOutput, "LinterOutput");
cbor_case!(ir_graph, aggrega_abi::IRGraph, "IRGraph");
cbor_case!(policy_input, aggrega_abi::PolicyInput, "PolicyInput");
cbor_case!(policy_output, aggrega_abi::PolicyOutput, "PolicyOutput");
cbor_case!(render_node_input, aggrega_abi::RenderNodeInput, "RenderNodeInput");
cbor_case!(render_node_output, aggrega_abi::RenderNodeOutput, "RenderNodeOutput");
cbor_case!(assemble_boundary_input, aggrega_abi::AssembleBoundaryInput, "AssembleBoundaryInput");
cbor_case!(assemble_boundary_output, aggrega_abi::AssembleBoundaryOutput, "AssembleBoundaryOutput");
cbor_case!(link_publish_input, aggrega_abi::LinkPublishInput, "LinkPublishInput");
cbor_case!(link_publish_output, aggrega_abi::LinkPublishOutput, "LinkPublishOutput");
cbor_case!(link_resolve_input, aggrega_abi::LinkResolveInput, "LinkResolveInput");
cbor_case!(link_resolve_output, aggrega_abi::LinkResolveOutput, "LinkResolveOutput");
cbor_case!(parse_input, aggrega_abi::ParseInput, "ParseInput");
cbor_case!(parse_output, aggrega_abi::ParseOutput, "ParseOutput");
cbor_case!(import_input, aggrega_abi::ImportInput, "ImportInput");
cbor_case!(import_output, aggrega_abi::ImportOutput, "ImportOutput");
cbor_case!(parse_source_response, aggrega_abi::ParseSourceResponse, "ParseSourceResponse");
cbor_case!(fetch_external_input_response, aggrega_abi::FetchExternalInputResponse, "FetchExternalInputResponse");

// The governance vectors cover the payload shapes reached only by a governance
// attachment. RenderPolicyInput is the payload a policy attachment receives at
// the render stage — rendered artifacts instead of the IR graph — carrying one
// artifact with both optional members (the parsed tree and the declared format)
// and one that omits both. The two annotation-transform vectors are the wire an
// annotation-transform attachment receives and returns: the input carries an
// empty annotation map (pinning that an empty object is emitted rather than
// dropped), a binding-key list whose entries are NOT in sorted order (pinning
// that a list is carried in author order rather than sorted), and an opaque
// settings object with out-of-order keys (pinning the verbatim carry); the
// output omits the failure member.
//
// The annotation map's key sort needs no separate assertion in this crate. Its
// CBOR half is already pinned by the vector: CBOR orders map keys by their
// encoded bytes, whose length head leads, so a short key precedes a long one —
// an order that differs from the lexicographic order the golden's JSON carries,
// so the encoder has to re-sort actively and dropping that sort changes the
// bytes. Its JSON half is structural rather than encoder logic, because the
// generated map field is a BTreeMap and serialization walks it in key order;
// there is no sort call in the JSON path that could regress. The Go and
// TypeScript harnesses, whose map types carry neither guarantee, each pin the
// ordering with a dedicated constructed-value test.
cbor_case!(render_policy_input, aggrega_abi::RenderPolicyInput, "RenderPolicyInput");
cbor_case!(annotation_transform_input, aggrega_abi::AnnotationTransformInput, "AnnotationTransformInput");
cbor_case!(annotation_transform_output, aggrega_abi::AnnotationTransformOutput, "AnnotationTransformOutput");

// PluginDescriptor and a few types recur across several vectors; each vector is
// its own JSON+CBOR assertion, so they are spelled out rather than macro-bound to
// one name.
#[test]
fn plugin_descriptor_variants() {
    for name in [
        "PluginDescriptorCLI",
        "PluginDescriptorProvider",
        "PluginDescriptorMinimal",
        "PluginDescriptorMoo",
    ] {
        let v: aggrega_abi::PluginDescriptor = json_parity(name);
        let want = golden_cbor(name);
        let enc = v.encode_cbor().expect("encode_cbor");
        assert_eq!(hex_encode(&enc), hex_encode(&want), "{name} CBOR mismatch");
        let rt = aggrega_abi::PluginDescriptor::decode_cbor(&enc).expect("decode").encode_cbor().expect("re-encode");
        assert_eq!(hex_encode(&rt), hex_encode(&enc), "{name} CBOR round-trip");
    }
}

// MutatorInputSources is the label-transform payload WITH its pinned sources and
// settings populated. The mutator vector above leaves both absent, so the file
// list a governance attachment hands its guest crossed the wire with no parity
// coverage. It reuses the MutatorInput type, so it is spelled out rather than
// macro-bound to a second name.
#[test]
fn mutator_input_sources() {
    let v: aggrega_abi::MutatorInput = json_parity("MutatorInputSources");
    let want = golden_cbor("MutatorInputSources");
    let enc = v.encode_cbor().expect("encode_cbor");
    assert_eq!(hex_encode(&enc), hex_encode(&want), "MutatorInputSources CBOR mismatch");
    let rt = aggrega_abi::MutatorInput::decode_cbor(&enc)
        .expect("decode")
        .encode_cbor()
        .expect("re-encode");
    assert_eq!(hex_encode(&rt), hex_encode(&enc), "MutatorInputSources CBOR round-trip");
}

#[test]
fn parse_output_bare_and_content() {
    for (name, is_parse) in [("ParseOutputBare", true), ("FetchExternalInputResponseContent", false)] {
        let want = golden_cbor(name);
        let enc = if is_parse {
            let v: aggrega_abi::ParseOutput = json_parity(name);
            v.encode_cbor().expect("encode")
        } else {
            let v: aggrega_abi::FetchExternalInputResponse = json_parity(name);
            v.encode_cbor().expect("encode")
        };
        assert_eq!(hex_encode(&enc), hex_encode(&want), "{name} CBOR mismatch");
    }
}

// The float-boundary suite carries values a JSON number cannot express (NaN,
// ±Inf, −0.0) as the raw f64 bits, so the CBOR CDE §4.2 float rule is proven
// cross-language on exactly those cases. Each entry's five-field CandidateSignals
// must encode to the committed canonical bytes.
#[derive(serde::Deserialize)]
struct FloatVector {
    label: String,
    bits: String,
    cbor: String,
}

#[test]
fn float_boundary_vectors() {
    let p = testdata().join("float_vectors.json");
    let raw = fs::read_to_string(&p).unwrap_or_else(|e| panic!("read {}: {}", p.display(), e));
    let vecs: Vec<FloatVector> = serde_json::from_str(&raw).expect("decode float_vectors.json");
    for fv in vecs {
        let bits = hex_decode(&fv.bits);
        assert_eq!(bits.len(), 8, "{}: bad bits", fv.label);
        let mut u: u64 = 0;
        for b in &bits {
            u = (u << 8) | *b as u64;
        }
        let f = f64::from_bits(u);
        let v = CandidateSignals {
            table: f,
            category: f,
            domain_labels: f,
            shape: f,
            name: f,
        };
        let enc = v.encode_cbor().expect("encode_cbor");
        assert_eq!(hex_encode(&enc), fv.cbor, "{}: CBOR float mismatch", fv.label);
        // Round-trip: NaN != NaN by value, but the canonical bytes are stable.
        let rt = CandidateSignals::decode_cbor(&enc).expect("decode").encode_cbor().expect("re-encode");
        assert_eq!(hex_encode(&rt), fv.cbor, "{}: float round-trip not stable", fv.label);
    }
}
