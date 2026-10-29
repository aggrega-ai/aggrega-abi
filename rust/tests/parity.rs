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

/// golden_json reads a JSON golden VERBATIM. Its committed bytes are the wire
/// bytes, carrying no terminating newline, so the byte-compare against the encoder
/// output is itself the strictest possible check on the file's shape and a
/// separate assertion would only restate it. Reading it verbatim is what makes
/// that compare mean anything: a reader that trimmed surrounding whitespace could
/// not distinguish the committed file from one whose bytes had drifted, and a
/// stray newline would be absorbed here while a stricter harness rejected it —
/// leaving the same file meaning two different things in two languages, which is
/// exactly what a shared corpus cannot afford.
fn golden_json(name: &str) -> String {
    let p = testdata().join(format!("{name}.golden.json"));
    fs::read_to_string(&p).unwrap_or_else(|e| panic!("read golden {}: {}", p.display(), e))
}

/// golden_hex reads a canonical-CBOR golden and returns its hex body, holding the
/// file to an exact on-disk shape: the lowercase hex digits, then exactly one
/// terminating newline, and nothing else anywhere. Admitting exactly one form is
/// what keeps the corpus single-valued across the three harnesses — a whitespace
/// variant one of them silently accepted would be a fixture the languages read
/// differently.
fn golden_hex(name: &str) -> String {
    let p = testdata().join(format!("{name}.cbor.hex"));
    let raw = fs::read_to_string(&p)
        .unwrap_or_else(|e| panic!("read cbor golden {}: {}", p.display(), e));
    let body = raw.strip_suffix('\n').unwrap_or_else(|| {
        panic!("golden {name}.cbor.hex does not end in a newline, so its committed shape is not the canonical one: {raw:?}")
    });
    if let Some(i) = body.bytes().position(|c| !c.is_ascii_digit() && !(b'a'..=b'f').contains(&c)) {
        panic!(
            "golden {}.cbor.hex carries {:?} at offset {}, so it is not lowercase hex followed by one newline: {:?}",
            name,
            &body[i..i + 1],
            i,
            raw
        );
    }
    body.to_string()
}

fn golden_cbor(name: &str) -> Vec<u8> {
    hex_decode(&golden_hex(name))
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

/// cross_parity proves the COMMITTED CBOR bytes decode into a value whose JSON
/// serialization is the committed JSON golden.
///
/// This is the only assertion that feeds the CBOR decoder the committed canonical
/// bytes. Every other CBOR leg decodes this crate's own fresh output, so an
/// encoder and decoder sharing a compensating defect would agree with each other
/// and stay green.
///
/// Its power over map-key ordering in THIS crate depends on the container type.
/// A plain map field is a `BTreeMap` (see `LinkEdge::target_annotations`), which
/// serde walks in key order, so there is no sort call in the JSON path that could
/// regress. A map the schema marks order-preserving is the crate's `OrderedMap`
/// (see `AnnotationTarget::annotations` and `IRNode::labels`), which serde walks
/// in the order it holds, so a value decoded from the committed CBOR bytes carries
/// the binary wire's order into the JSON encoder — and this is the assertion that
/// proves the order survives the binary form.
fn cross_parity<T: Serialize>(name: &str, v: &T) {
    let want = golden_json(name);
    let got = serde_json::to_string(v).expect("serialize from committed CBOR");
    assert_eq!(
        want, got,
        "{name} JSON wire encoded from the committed CBOR bytes is not the JSON golden"
    );
}

/// A generated type carrying the CBOR codec. Implemented per vector type via the
/// cbor_case macro so the harness can assert encode==golden and a byte-stable
/// round-trip generically.
trait CborCase: Sized {
    fn enc_cbor(&self) -> Vec<u8>;
    fn dec_cbor(data: &[u8]) -> Self;
}

/// impl_cbor_case binds a type to the harness. It is separate from cbor_vector
/// because a type can carry several vectors while the trait may be implemented for
/// it only once.
macro_rules! impl_cbor_case {
    ($ty:ty, $name:literal) => {
        impl CborCase for $ty {
            fn enc_cbor(&self) -> Vec<u8> {
                self.encode_cbor().expect(concat!($name, " encode_cbor"))
            }
            fn dec_cbor(data: &[u8]) -> Self {
                <$ty>::decode_cbor(data).expect(concat!($name, " decode_cbor"))
            }
        }
    };
}

macro_rules! cbor_vector {
    ($fn:ident, $ty:ty, $name:literal) => {
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
            cross_parity($name, &<$ty>::dec_cbor(&want));
        }
    };
}

/// cbor_case is the common case: one type, one vector, so the trait binding and the
/// test are declared together.
macro_rules! cbor_case {
    ($fn:ident, $ty:ty, $name:literal) => {
        impl_cbor_case!($ty, $name);
        cbor_vector!($fn, $ty, $name);
    };
}

cbor_case!(cli_run_input, aggrega_abi::CLIRunInput, "CLIRunInput");
cbor_case!(frp_step_input, aggrega_abi::FRPStepInput, "FRPStepInput");
cbor_case!(frp_step_output, aggrega_abi::FRPStepOutput, "FRPStepOutput");
cbor_case!(mutator_input, aggrega_abi::MutatorInput, "MutatorInput");
cbor_case!(mutator_output, aggrega_abi::MutatorOutput, "MutatorOutput");
cbor_case!(linter_input, aggrega_abi::LinterInput, "LinterInput");
cbor_case!(linter_output, aggrega_abi::LinterOutput, "LinterOutput");
// A second vector for LinterInput, so cbor_vector rather than cbor_case: the trait
// binding for this type is declared by the vector above. This one carries the
// input of a rule that reads a whole Release, with the Release and the Component
// map POPULATED — both are elided when empty, so the vector above carries the same
// bytes whether the codec handles them or drops them. The Component map is
// order-preserving and holds "agent" before "db", which is not the length-first
// order a sorted binary map would take, and each opaque document keeps its keys out
// of key order.
cbor_vector!(linter_input_release, aggrega_abi::LinterInput, "LinterInputRelease");
cbor_case!(ir_graph, aggrega_abi::IRGraph, "IRGraph");
cbor_case!(policy_input, aggrega_abi::PolicyInput, "PolicyInput");
cbor_case!(policy_output, aggrega_abi::PolicyOutput, "PolicyOutput");
cbor_case!(render_node_input, aggrega_abi::RenderNodeInput, "RenderNodeInput");
cbor_case!(render_node_output, aggrega_abi::RenderNodeOutput, "RenderNodeOutput");
// A second vector for RenderNodeOutput, so cbor_vector rather than cbor_case: the
// trait binding for this type is declared by the vector above. This one carries a
// POPULATED fact list, which no other vector on this socket does — the field is
// elided when empty, so every other vector's bytes are the same whether the codec
// handles it or drops it. Its facts are listed out of name order and its object
// value's keys out of key order, so a codec that sorted either would not match.
cbor_vector!(render_node_output_render_facts, aggrega_abi::RenderNodeOutput, "RenderNodeOutputRenderFacts");
// Second vectors for RenderNodeInput and RenderNodeOutput, so cbor_vector for the
// reason given above. The input carries POPULATED omit entries — one with a
// namespace and one without, listed out of kind and name order — and the output
// carries one count per entry, a non-zero count before a zero. Both lists are
// elided when empty, so every other vector on this socket has the same bytes
// whether the codec handles them or drops them.
cbor_vector!(render_node_input_omits, aggrega_abi::RenderNodeInput, "RenderNodeInputOmits");
cbor_vector!(render_node_output_omitted, aggrega_abi::RenderNodeOutput, "RenderNodeOutputOmitted");
// A third vector for RenderNodeInput: its first omit entry carries the sensitivity
// mark and its second does not. The mark is elided when false, so every other
// vector carrying omit entries has the same bytes whether the codec handles it or
// drops it.
cbor_vector!(render_node_input_omits_sensitive, aggrega_abi::RenderNodeInput, "RenderNodeInputOmitsSensitive");
// A fourth vector for RenderNodeInput: its first omit entry names the subchart its
// unit came from and its second does not. The subchart is elided when empty, so
// every other vector carrying omit entries has the same bytes whether the codec
// handles it or drops it.
cbor_vector!(render_node_input_omits_subchart, aggrega_abi::RenderNodeInput, "RenderNodeInputOmitsSubchart");
// A second vector for IRNodeProjection, so cbor_vector rather than cbor_case: the
// trait binding for this type is declared once with the map-order probe below.
// This vector carries a POPULATED per-key sensitivity marking, which no other
// vector on this socket does — the field is elided when empty, so every other
// vector's bytes are the same whether the codec handles it or drops it.
cbor_vector!(ir_node_projection_sensitive_rendered_keys, aggrega_abi::IRNodeProjection, "IRNodeProjectionSensitiveRenderedKeys");
// A third vector for IRNodeProjection, cbor_vector for the same reason: its values
// are a values mapping's output and it says so. The mark is elided when false, so
// every other vector on this socket has the same bytes whether the codec handles it
// or drops it, and this is the vector where dropping a true mark moves the bytes.
cbor_vector!(ir_node_projection_values_mapped, aggrega_abi::IRNodeProjection, "IRNodeProjectionValuesMapped");
// A second vector for RenderedNode, cbor_vector because the trait binding for this
// type is declared once with the map-order probe below: a rendered node on the
// assembly input whose values are a values mapping's output and say so. The mark is
// elided when false, so this is the vector where dropping a true mark moves the bytes.
cbor_vector!(rendered_node_values_mapped, aggrega_abi::RenderedNode, "RenderedNodeValuesMapped");
// A second vector for IRFeatureEntry, so cbor_vector rather than cbor_case for the
// same reason: the trait binding for this type is declared once with the map-order
// probe below. This vector carries a POPULATED per-leaf sensitivity marking,
// which no other vector on this socket does — the field is elided when empty, so
// every other vector's bytes are the same whether the codec handles the field or
// drops it, and a regression there would move nothing and fail nowhere.
cbor_vector!(ir_feature_entry_sensitive_payload_leaves, aggrega_abi::IRFeatureEntry, "IRFeatureEntrySensitivePayloadLeaves");
// An attachment entry carrying the connection it was written on, beside one of the
// node's own carrying none — the field is elided when absent, so every other vector
// with an attachment entry has the same bytes whether the codec handles it or drops
// it. The two share one node's bucket, so the vector is the graph's spec, together
// with the edge the connection names. IRSpec is cbor_vector rather than cbor_case
// because its trait binding is declared by the map-order vector further down.
cbor_vector!(ir_spec_connection_attachment, aggrega_abi::IRSpec, "IRSpecConnectionAttachment");
// The realization an attachment carries, on a node's entries and on a Release
// boundary's — the field is elided when absent, so this is the vector where dropping
// it moves the bytes, and the one entry carrying none is where emitting it
// unconditionally would. IRSpec is cbor_vector for the reason given above.
cbor_vector!(ir_spec_attachment_realization, aggrega_abi::IRSpec, "IRSpecAttachmentRealization");
// The same marking at its other two units. A node's own values and infra are
// governed by the node's Component, and a boundary attachment's infra by the
// Feature it names, so neither is recoverable from the entry vector above.
// IRNode is cbor_vector rather than cbor_case because its trait binding is
// already declared by the map-order vector further down; IRBoundaryFeature is
// cbor_case because no other vector carries that shape on its own.
cbor_vector!(ir_node_sensitive_payload_leaves, aggrega_abi::IRNode, "IRNodeSensitivePayloadLeaves");
cbor_case!(ir_boundary_feature_sensitive_payload_leaves, aggrega_abi::IRBoundaryFeature, "IRBoundaryFeatureSensitivePayloadLeaves");
cbor_case!(assemble_boundary_input, aggrega_abi::AssembleBoundaryInput, "AssembleBoundaryInput");
cbor_case!(assemble_boundary_output, aggrega_abi::AssembleBoundaryOutput, "AssembleBoundaryOutput");
cbor_case!(link_publish_input, aggrega_abi::LinkPublishInput, "LinkPublishInput");
cbor_case!(link_publish_output, aggrega_abi::LinkPublishOutput, "LinkPublishOutput");
cbor_case!(link_resolve_input, aggrega_abi::LinkResolveInput, "LinkResolveInput");
cbor_case!(link_resolve_output, aggrega_abi::LinkResolveOutput, "LinkResolveOutput");
// The populated payloads of the render family's remaining exports. A type whose
// trait binding another vector already declares — a failure envelope or a map-order
// probe below — is cbor_vector; the rest are cbor_case. Each carries its members
// populated: validate_node_input's diagnostics, carve's and reassemble's file bytes
// and units, name_bound's proposal and verdicts (one carrying a bound, one not),
// and rendered_name's identities and answer.
cbor_vector!(validate_node_output, aggrega_abi::ValidateNodeOutput, "ValidateNodeOutput");
cbor_case!(carve_input, aggrega_abi::CarveInput, "CarveInput");
cbor_case!(carve_output, aggrega_abi::CarveOutput, "CarveOutput");
cbor_case!(reassemble_input, aggrega_abi::ReassembleInput, "ReassembleInput");
cbor_case!(reassemble_output, aggrega_abi::ReassembleOutput, "ReassembleOutput");
cbor_case!(name_bound_input, aggrega_abi::NameBoundInput, "NameBoundInput");
cbor_vector!(name_bound_output, aggrega_abi::NameBoundOutput, "NameBoundOutput");
cbor_case!(rendered_name_input, aggrega_abi::RenderedNameInput, "RenderedNameInput");
cbor_vector!(rendered_name_output, aggrega_abi::RenderedNameOutput, "RenderedNameOutput");
cbor_case!(parse_input, aggrega_abi::ParseInput, "ParseInput");
// A second vector for ParseInput, so cbor_vector rather than cbor_case: the trait
// binding for this type is declared by the vector above. Its source bytes are not
// valid UTF-8 — a bare 0xff 0xfe pair, a NUL, and a lead byte followed by a
// non-continuation byte — so a codec that carried them as text, or as base64 text,
// rather than as a raw byte string cannot reproduce the canonical bytes. It omits
// the options document the vector above carries, so it also pins that omission.
cbor_vector!(parse_input_non_utf8_source, aggrega_abi::ParseInput, "ParseInputNonUTF8Source");
cbor_case!(parse_output, aggrega_abi::ParseOutput, "ParseOutput");
cbor_case!(import_input, aggrega_abi::ImportInput, "ImportInput");
// A second vector for ImportInput, so cbor_vector rather than cbor_case: the trait
// binding for this type is declared by the vector above. This one carries a
// POPULATED catalog view, which the vector above does not — the field is elided
// when absent, so every other ImportInput vector has the same bytes whether the
// codec handles it or drops it. One Component declares sockets and one declares
// none, and the sockets are listed out of name order, so a codec that sorted a
// list would not match.
cbor_vector!(import_input_catalog, aggrega_abi::ImportInput, "ImportInputCatalog");
// A third: a file imported on its own, carrying the primary bytes and the name of
// the file they were read from, with no named source. The name is elided when
// empty, so this is the vector where dropping it moves the bytes, and its
// four-byte key sorts ahead of every other key of the shape.
cbor_vector!(import_input_named, aggrega_abi::ImportInput, "ImportInputNamed");
cbor_case!(import_output, aggrega_abi::ImportOutput, "ImportOutput");
cbor_case!(parse_source_response, aggrega_abi::ParseSourceResponse, "ParseSourceResponse");
// A second vector for ParseSourceResponse, so cbor_vector rather than cbor_case:
// the trait binding for this type is declared once, on the failure vector above.
// This one is the SUCCESS shape, the half that failure vector cannot stand in for
// — the two are the two directions of one omission property, and an encoder
// emitting an error key unconditionally passes any test built only from success
// fixtures while one dropping it unconditionally passes any test built only from
// failure fixtures. It also carries the opaque tree, which crosses as a byte
// string of the JSON document's verbatim bytes; the failure envelope reaches no
// opaque field at all, so the tree member's bytes are pinned here. The document
// is long enough that its byte-string length needs a two-byte head, so the length
// rides in the head rather than in the initial byte.
cbor_vector!(parse_source_response_tree, aggrega_abi::ParseSourceResponse, "ParseSourceResponseTree");
cbor_case!(fetch_external_input_response, aggrega_abi::FetchExternalInputResponse, "FetchExternalInputResponse");
// A zero-length fetched payload, so cbor_vector rather than cbor_case: the trait
// binding for this type is declared by the failure vector above. The content field
// is omitted when empty, so the reply carries no field at all and stays
// distinguishable from a failure only because a failure always carries its error
// fields. This crate holds the content as a base64 `String`, so the vector also
// pins that an empty one is omitted rather than written as an empty byte string.
cbor_vector!(fetch_external_input_response_empty_content, aggrega_abi::FetchExternalInputResponse, "FetchExternalInputResponseEmptyContent");

// The pack-source and read-source-file request and SUCCESS reply vectors. The
// success reply is the half a failure envelope cannot stand in for: the two are the
// two directions of one omission property, and an encoder emitting an error key
// unconditionally passes any test built only from success fixtures while one
// dropping it unconditionally passes any test built only from failure fixtures.
// Each reply type's failure envelope is registered further down; the pair is the
// assertion.
//
// The request vectors declare no omittable member, so their map head is a constant
// and there is no omission direction to prove. They pin head and key order on the
// payload the guest builds and the host decodes.
//
// The packed reply carries three files in the host's byte-order walk — Chart.yaml
// before charts/ before templates/, byte order rather than case-insensitive order —
// and one file's content is deliberately not valid UTF-8, proving the file bytes
// cross as an opaque byte string rather than as text. This crate declares those
// bytes as a base64 `String`, so the vector also pins that the CBOR half decodes to
// the same byte string the other two languages emit; its base64 is the only place
// in the corpus exercising both non-alphanumeric alphabet characters, which this
// crate's hand-written base64 codec reaches on no other vector.
//
// SourcePackResponse and SourceFileResponse already carry a harness binding from
// their failure-envelope vectors below, so they take cbor_vector alone — a second
// impl_cbor_case for the same type would not compile.
cbor_case!(source_pack_request, aggrega_abi::SourcePackRequest, "SourcePackRequest");
cbor_vector!(source_pack_response, aggrega_abi::SourcePackResponse, "SourcePackResponse");
cbor_case!(source_file_request, aggrega_abi::SourceFileRequest, "SourceFileRequest");
cbor_vector!(source_file_response, aggrega_abi::SourceFileResponse, "SourceFileResponse");

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
// The annotation map is order-preserving, so neither wire form sorts it. Its
// field is the crate's `OrderedMap`, whose serde form and whose array form are
// both written in the order it holds; the map-order probe vector below, whose
// order is neither sorted order, is what fails if either half sorted it. The Go
// and TypeScript harnesses each pin the same order a third way, from a
// constructed value.
cbor_case!(render_policy_input, aggrega_abi::RenderPolicyInput, "RenderPolicyInput");
cbor_case!(annotation_transform_input, aggrega_abi::AnnotationTransformInput, "AnnotationTransformInput");
cbor_case!(annotation_transform_output, aggrega_abi::AnnotationTransformOutput, "AnnotationTransformOutput");

// The map-order probe vectors carry one golden pair per map-owning payload, with
// every map that payload declares populated by a probe key set. Map-key ordering
// is a determinism contract — identical input must produce byte-identical output —
// and it reads differently for the two kinds of map. A plain map carries no order
// of its own, so it is populated by keys whose lexicographic order is the exact
// reverse of their canonical CBOR order, and every encoder sorts it. A map the
// schema marks order-preserving carries its order as content, so it is populated
// in an order that is neither sorted order and holds a key that is an array index,
// and every encoder writes it in that order.
//
// In this crate the JSON half of a plain map's ordering is structural rather than
// encoder logic, because a plain map field is declared `BTreeMap`, so
// serialization walks it in key order; its CBOR ordering, whose length-led rule is
// NOT the BTreeMap order, is produced by a real sort call these vectors pin. An
// order-preserving field is declared `OrderedMap` and both its halves are written
// in the order it holds, which these vectors pin on both wires.
//
// FRPStepInput, FRPStepOutput, LinkPublishOutput, LinkResolveInput, LinterInput,
// PolicyInput and RenderNodeOutput already carry a harness binding from their socket
// vectors above, so they take cbor_vector alone — a second impl_cbor_case for the
// same type would not compile. PolicyInput declares no map of its own; its vector
// here holds the graph's order-preserving nodes and labels inside a populated policy
// payload, as the ir-verdict role hands it to a guest.
cbor_case!(map_order_annotation_target, aggrega_abi::AnnotationTarget, "MapOrderAnnotationTarget");
cbor_case!(map_order_artifact_projection_entry, aggrega_abi::ArtifactProjectionEntry, "MapOrderArtifactProjectionEntry");
cbor_case!(map_order_document_set, aggrega_abi::DocumentSet, "MapOrderDocumentSet");
cbor_case!(map_order_emitted_entity, aggrega_abi::EmittedEntity, "MapOrderEmittedEntity");
cbor_vector!(map_order_frp_step_input, aggrega_abi::FRPStepInput, "MapOrderFRPStepInput");
cbor_vector!(map_order_frp_step_output, aggrega_abi::FRPStepOutput, "MapOrderFRPStepOutput");
cbor_case!(map_order_ir_feature_entry, aggrega_abi::IRFeatureEntry, "MapOrderIRFeatureEntry");
cbor_case!(map_order_ir_node, aggrega_abi::IRNode, "MapOrderIRNode");
cbor_case!(map_order_ir_node_projection, aggrega_abi::IRNodeProjection, "MapOrderIRNodeProjection");
cbor_case!(map_order_ir_spec, aggrega_abi::IRSpec, "MapOrderIRSpec");
cbor_case!(map_order_link_edge, aggrega_abi::LinkEdge, "MapOrderLinkEdge");
// A second LinkEdge vector, so cbor_vector rather than cbor_case: the binding for
// this type is declared by the vector above. This one populates the endpoints, the
// socket and the provides values beside the probe maps, and carries
// read_provides_keys, which the vector above leaves absent — so that vector holds
// the same bytes whether the codec handles the list or drops it. The list holds
// the provides keys in the order the provides map holds them, which is neither
// sorted order, so a codec that sorted it fails the byte-compare.
cbor_vector!(map_order_link_edge_read_provides_keys, aggrega_abi::LinkEdge, "MapOrderLinkEdgeReadProvidesKeys");
cbor_vector!(map_order_link_publish_output, aggrega_abi::LinkPublishOutput, "MapOrderLinkPublishOutput");
cbor_vector!(map_order_link_resolve_input, aggrega_abi::LinkResolveInput, "MapOrderLinkResolveInput");
cbor_vector!(map_order_linter_input, aggrega_abi::LinterInput, "MapOrderLinterInput");
cbor_case!(map_order_mutator_graph, aggrega_abi::MutatorGraph, "MapOrderMutatorGraph");
cbor_case!(map_order_mutator_node, aggrega_abi::MutatorNode, "MapOrderMutatorNode");
cbor_vector!(map_order_policy_input, aggrega_abi::PolicyInput, "MapOrderPolicyInput");
cbor_vector!(map_order_render_node_output, aggrega_abi::RenderNodeOutput, "MapOrderRenderNodeOutput");
cbor_case!(map_order_rendered_node, aggrega_abi::RenderedNode, "MapOrderRenderedNode");
cbor_case!(map_order_sub_chart, aggrega_abi::SubChart, "MapOrderSubChart");
cbor_case!(map_order_workload_projection, aggrega_abi::WorkloadProjection, "MapOrderWorkloadProjection");

// The failure-envelope vectors pin the error member every socket output type
// carries — the path that reports "I ran but could not complete", which the host
// halts the build on. It is the path that carries bad news, so a codec defect
// there surfaces only once something has already gone wrong and what gets
// corrupted is the diagnostic itself.
//
// Each sets the error message with its success members left at their zero value,
// because the host abandons the payload the moment it reads a non-empty error: an
// error beside substantively populated success content is a state nothing ever acts
// on. Where the schema makes a success member required it stays present but empty,
// which is what a failing guest must emit to satisfy the contract. ParseSourceResponse
// and FetchExternalInputResponse already carried a failure golden and are not
// duplicated.
//
// Each type carries exactly one harness binding: a type bound by a vector
// elsewhere takes cbor_vector alone, because a second impl_cbor_case for the same
// type would not compile, and a type whose only binding is here takes cbor_case,
// which declares the binding and the test together.
cbor_vector!(error_envelope_annotation_transform_output, aggrega_abi::AnnotationTransformOutput, "ErrorEnvelopeAnnotationTransformOutput");
cbor_vector!(error_envelope_assemble_boundary_output, aggrega_abi::AssembleBoundaryOutput, "ErrorEnvelopeAssembleBoundaryOutput");
cbor_vector!(error_envelope_frp_step_output, aggrega_abi::FRPStepOutput, "ErrorEnvelopeFRPStepOutput");
cbor_vector!(error_envelope_import_output, aggrega_abi::ImportOutput, "ErrorEnvelopeImportOutput");
cbor_vector!(error_envelope_link_publish_output, aggrega_abi::LinkPublishOutput, "ErrorEnvelopeLinkPublishOutput");
cbor_vector!(error_envelope_link_resolve_output, aggrega_abi::LinkResolveOutput, "ErrorEnvelopeLinkResolveOutput");
cbor_vector!(error_envelope_linter_output, aggrega_abi::LinterOutput, "ErrorEnvelopeLinterOutput");
cbor_vector!(error_envelope_mutator_output, aggrega_abi::MutatorOutput, "ErrorEnvelopeMutatorOutput");
cbor_case!(error_envelope_name_bound_output, aggrega_abi::NameBoundOutput, "ErrorEnvelopeNameBoundOutput");
cbor_vector!(error_envelope_parse_output, aggrega_abi::ParseOutput, "ErrorEnvelopeParseOutput");
cbor_vector!(error_envelope_policy_output, aggrega_abi::PolicyOutput, "ErrorEnvelopePolicyOutput");
cbor_vector!(error_envelope_render_node_output, aggrega_abi::RenderNodeOutput, "ErrorEnvelopeRenderNodeOutput");
cbor_case!(error_envelope_rendered_name_output, aggrega_abi::RenderedNameOutput, "ErrorEnvelopeRenderedNameOutput");
cbor_case!(error_envelope_source_file_response, aggrega_abi::SourceFileResponse, "ErrorEnvelopeSourceFileResponse");
cbor_case!(error_envelope_source_pack_response, aggrega_abi::SourcePackResponse, "ErrorEnvelopeSourcePackResponse");
cbor_case!(error_envelope_validate_node_output, aggrega_abi::ValidateNodeOutput, "ErrorEnvelopeValidateNodeOutput");

// A refused vector is canonical CBOR every decoder of its shape must REFUSE: a
// document no encoder of the shape can write, malformed in exactly one way. A
// vector pair holds the decoder to what it must read; a refused vector holds it to
// what it must not, and the two cannot stand in for each other — a decoder that
// kept one of two repeated keys would read every well-formed vector correctly.
//
// These pin the order-preserving map's array form: a key the array repeats, named
// in the refusal; an entry holding three items; and an entry whose key is not a
// text string. The same vectors bind the Go and TypeScript decoders, and the
// compiler's independent byte-parity oracle confirms each is well-formed CBOR whose
// only fault is the one its name states.

/// REFUSED_SUFFIX is the file a refused vector is. It does not end in the vector
/// pair's CBOR suffix, so the pair census never counts it as half of one.
const REFUSED_SUFFIX: &str = ".refused-cbor.hex";

/// refused_cbor reads a refused vector, holding it to the hex golden shape.
fn refused_cbor(name: &str) -> Vec<u8> {
    let p = testdata().join(format!("{name}{REFUSED_SUFFIX}"));
    let raw = fs::read_to_string(&p).unwrap_or_else(|e| panic!("read refused vector {}: {}", p.display(), e));
    let body = raw.strip_suffix('\n').unwrap_or_else(|| {
        panic!("refused vector {name} does not end in a newline, so its committed shape is not the canonical one: {raw:?}")
    });
    assert!(
        body.bytes().all(|c| c.is_ascii_digit() || (b'a'..=b'f').contains(&c)),
        "refused vector {name} is not lowercase hex followed by one newline: {raw:?}"
    );
    hex_decode(body)
}

macro_rules! refused_vector {
    ($fn:ident, $ty:ty, $name:literal, $want:literal) => {
        #[test]
        fn $fn() {
            match <$ty>::decode_cbor(&refused_cbor($name)) {
                Ok(v) => panic!("{} decoded a vector every decoder must refuse, to {:?}", $name, v),
                Err(e) => assert!(
                    e.0.contains($want),
                    "{} was refused, but the error does not name the fault: got {:?}, want it to contain {:?}",
                    $name,
                    e.0,
                    $want
                ),
            }
        }
    };
}

refused_vector!(refused_repeated_key, aggrega_abi::MutatorNode, "MapOrderMutatorNodeRepeatedKey", "duplicate key \"a\"");
refused_vector!(refused_entry_arity, aggrega_abi::MutatorNode, "MapOrderMutatorNodeEntryArity", "holds 3 items, want 2");
refused_vector!(refused_entry_key_not_text, aggrega_abi::MutatorNode, "MapOrderMutatorNodeEntryKeyNotText", "expected text string");

/// contains_bytes reports whether needle occurs in haystack.
fn contains_bytes(haystack: &[u8], needle: &[u8]) -> bool {
    haystack.windows(needle.len()).any(|w| w == needle)
}

// A non-required order-preserving field is omitted when it is empty and emitted
// when it is populated, on both wires. The two fixtures are asserted together
// because each hides the other's failure: an encoder emitting the key
// unconditionally passes the populated one, and one dropping it unconditionally
// passes the empty one. The binary form's map head declares its pair count, so the
// empty form's bytes are also read back, which a head counting the omitted key
// would make fail.
#[test]
fn an_empty_order_preserving_field_is_omitted_and_a_populated_one_is_emitted() {
    let empty = aggrega_abi::RenderNodeOutput::default();
    let json = serde_json::to_string(&empty).expect("serialize the empty form");
    assert!(!json.contains("derived_keys"), "an empty order-preserving field is on the JSON wire: {json}");
    let cbor = empty.encode_cbor().expect("encode the empty form");
    assert!(
        !contains_bytes(&cbor, b"derived_keys"),
        "an empty order-preserving field is on the binary wire: {}",
        hex_encode(&cbor)
    );
    aggrega_abi::RenderNodeOutput::decode_cbor(&cbor).expect("the empty form's bytes read back");

    let mut populated = aggrega_abi::RenderNodeOutput::default();
    populated.derived_keys.insert("resource-names".to_string(), vec!["web".to_string()]);
    let json = serde_json::to_string(&populated).expect("serialize the populated form");
    assert!(
        json.contains("\"derived_keys\":{\"resource-names\":[\"web\"]}"),
        "a populated order-preserving field is missing from the JSON wire: {json}"
    );
    let cbor = populated.encode_cbor().expect("encode the populated form");
    assert!(
        contains_bytes(&cbor, b"derived_keys"),
        "a populated order-preserving field is missing from the binary wire: {}",
        hex_encode(&cbor)
    );
    let back = aggrega_abi::RenderNodeOutput::decode_cbor(&cbor).expect("the populated form's bytes read back");
    assert_eq!(back.derived_keys, populated.derived_keys, "the populated field did not survive the binary form");
}

// PluginDescriptor and a few types recur across several vectors; each vector is
// its own JSON+CBOR assertion, so they are spelled out rather than macro-bound to
// one name.
#[test]
fn plugin_descriptor_variants() {
    for name in [
        "PluginDescriptorCLI",
        "PluginDescriptorProvider",
        "PluginDescriptorProviderOmits",
        // A provider contract claiming to accept the files a Recipe contributes to
        // its boundary's shared output: omitted when false, so this is the
        // descriptor vector where dropping the claim moves the bytes.
        "PluginDescriptorProviderBoundaryArtifacts",
        // An importer's recognition contract populated in every field; omitted
        // when absent, so this is the descriptor vector where dropping it, or
        // reordering a list it carries, moves the bytes.
        "PluginDescriptorImporter",
        "PluginDescriptorMinimal",
        "PluginDescriptorMoo",
    ] {
        let v: aggrega_abi::PluginDescriptor = json_parity(name);
        let want = golden_cbor(name);
        let enc = v.encode_cbor().expect("encode_cbor");
        assert_eq!(hex_encode(&enc), hex_encode(&want), "{name} CBOR mismatch");
        let rt = aggrega_abi::PluginDescriptor::decode_cbor(&enc).expect("decode").encode_cbor().expect("re-encode");
        assert_eq!(hex_encode(&rt), hex_encode(&enc), "{name} CBOR round-trip");
        cross_parity(
            name,
            &aggrega_abi::PluginDescriptor::decode_cbor(&want).expect("decode committed CBOR"),
        );
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
    cross_parity(
        "MutatorInputSources",
        &aggrega_abi::MutatorInput::decode_cbor(&want).expect("decode committed CBOR"),
    );
}

#[test]
fn parse_output_bare_and_content() {
    for (name, is_parse) in [("ParseOutputBare", true), ("FetchExternalInputResponseContent", false)] {
        let want = golden_cbor(name);
        let enc = if is_parse {
            let v: aggrega_abi::ParseOutput = json_parity(name);
            let enc = v.encode_cbor().expect("encode");
            cross_parity(
                name,
                &aggrega_abi::ParseOutput::decode_cbor(&want).expect("decode committed CBOR"),
            );
            enc
        } else {
            let v: aggrega_abi::FetchExternalInputResponse = json_parity(name);
            let enc = v.encode_cbor().expect("encode");
            cross_parity(
                name,
                &aggrega_abi::FetchExternalInputResponse::decode_cbor(&want)
                    .expect("decode committed CBOR"),
            );
            enc
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

// The vector corpus is ONE golden read by every harness that enumerates it, and
// the value of that arrangement is entirely in the word "one". Codecs generated
// from a single schema are only proven to produce a single wire if all of them are
// held to the same bytes; a vector exercised in one language and not another
// proves that language's codec self-consistent and says nothing about parity.
//
// Nothing structural keeps the harnesses aligned. Each enumerates its vectors in
// its own idiom — the macro invocations above, call sites in Go, an array in
// TypeScript — so agreement across them is a hand-maintained property, and a
// vector pair committed to the shared tree is wired by whoever remembers to wire
// it everywhere. The failure mode is silent in the worst way: the new vector
// passes wherever it was wired, every existing vector keeps passing, and the
// harness that never received it reports nothing at all. Coverage does not go
// red when it is missing, because missing coverage has no assertion to fail.
//
// The census below closes that on this side by deriving coverage FROM THE CORPUS
// rather than from a list: the directory is the authority, so a pair is covered
// the moment it lands or this test names it. Other harnesses carry the same guard
// against the same directory, and NOT ALL OF THEM LIVE IN THIS REPOSITORY — the
// compiler's independent byte-parity oracle carries one too. So a guard added or
// removed elsewhere is invisible from here, which is why this says what the
// property is rather than how many harnesses hold it.
//
// Only the corpus-to-harness direction needs asserting. The reverse — a harness
// asserting a vector the corpus does not carry — cannot pass quietly anywhere,
// because the golden reader panics on the missing file.

/// SOURCE is this harness's own text, embedded at compile time so the census reads
/// the real vector declarations. A hand-written list of names checked against the
/// directory would only move the drift one step, leaving the list and the macro
/// invocations free to disagree.
const SOURCE: &str = include_str!("parity.rs");

/// STANDALONE_FIXTURES are the corpus files that are deliberately NOT vector
/// pairs, each consumed by its own dedicated test in every harness that reads this
/// tree. They are named so that a fixture arriving in the corpus and wired nowhere
/// cannot pass as one of these.
const STANDALONE_FIXTURES: &[&str] = &["float_vectors.json"];

/// corpus_vectors returns the vector names committed to the shared tree, having
/// first asserted that every name carries BOTH halves of its pair and that every
/// file present is either half of a pair or a named standalone fixture. An
/// unrecognized file fails rather than being skipped, because a fixture nothing
/// consumes is the same coverage hole this census exists to close.
fn corpus_vectors() -> std::collections::BTreeSet<String> {
    let dir = testdata();
    let mut json = std::collections::BTreeSet::new();
    let mut cbor = std::collections::BTreeSet::new();
    let entries = fs::read_dir(&dir)
        .unwrap_or_else(|e| panic!("read the shared vector corpus {}: {}", dir.display(), e));
    for entry in entries {
        let entry = entry.unwrap_or_else(|e| panic!("read a corpus entry: {e}"));
        let name = entry.file_name().to_string_lossy().into_owned();
        if STANDALONE_FIXTURES.contains(&name.as_str()) {
            continue;
        }
        // A refused vector is a single file rather than a pair, held to its own
        // census below.
        if name.ends_with(REFUSED_SUFFIX) {
            continue;
        }
        if let Some(base) = name.strip_suffix(".golden.json") {
            json.insert(base.to_string());
        } else if let Some(base) = name.strip_suffix(".cbor.hex") {
            cbor.insert(base.to_string());
        } else {
            panic!(
                "the shared vector corpus carries {name:?}, which is neither half of a vector pair \
                 nor a fixture any harness is known to consume, so nothing asserts anything about it"
            );
        }
    }
    let half_only: Vec<&String> = json.symmetric_difference(&cbor).collect();
    assert!(
        half_only.is_empty(),
        "a vector is the JSON and CBOR pair, and a name carrying one half leaves the other codec \
         unpinned: {half_only:?}"
    );
    assert!(
        !json.is_empty(),
        "found no vector pairs in the shared corpus, so this census certifies nothing"
    );
    json
}

/// declarations returns this harness's text with line comments removed, so a
/// vector name that appears only in prose cannot be mistaken for one that is
/// wired.
fn declarations() -> String {
    SOURCE
        .lines()
        .map(|l| match l.find("//") {
            Some(i) => &l[..i],
            None => l,
        })
        .collect::<Vec<_>>()
        .join("\n")
}

#[test]
fn every_golden_vector_is_exercised() {
    let corpus = corpus_vectors();
    let declared = declarations();

    // Calibration: the census is worthless if the instrument reads nothing, so a
    // pattern that has stopped matching the code fails here rather than reporting
    // a clean sweep over an empty search.
    assert!(
        declared.contains("cbor_case!"),
        "found no vector declarations in this harness's own text, so the coverage census certifies nothing"
    );

    let missing: Vec<&String> = corpus
        .iter()
        .filter(|name| !declared.contains(&format!("\"{name}\"")))
        .collect();
    assert!(
        missing.is_empty(),
        "these vectors are committed to the shared corpus but no assertion in this harness reads them, \
         so this language's codec is unpinned on a wire the others prove: {missing:?}"
    );
}

/// refused_corpus returns the refused-vector names committed to the shared tree.
fn refused_corpus() -> std::collections::BTreeSet<String> {
    let dir = testdata();
    let mut refused = std::collections::BTreeSet::new();
    let entries = fs::read_dir(&dir)
        .unwrap_or_else(|e| panic!("read the shared vector corpus {}: {}", dir.display(), e));
    for entry in entries {
        let entry = entry.unwrap_or_else(|e| panic!("read a corpus entry: {e}"));
        let name = entry.file_name().to_string_lossy().into_owned();
        if let Some(base) = name.strip_suffix(REFUSED_SUFFIX) {
            refused.insert(base.to_string());
        }
    }
    assert!(
        !refused.is_empty(),
        "found no {REFUSED_SUFFIX} vectors in the shared corpus, so this census certifies nothing"
    );
    refused
}

#[test]
fn every_refused_vector_is_asserted() {
    let refused = refused_corpus();
    let declared = declarations();

    // Calibration, as for the pair census above.
    assert!(
        declared.contains("refused_vector!("),
        "found no refused-vector declarations in this harness's own text, so the census certifies nothing"
    );

    let missing: Vec<&String> = refused
        .iter()
        .filter(|name| !declared.contains(&format!("\"{name}\"")))
        .collect();
    assert!(
        missing.is_empty(),
        "these refused vectors are committed to the shared corpus but no assertion in this harness reads \
         them, so this language's decoder is not held to refusing them: {missing:?}"
    );
}
