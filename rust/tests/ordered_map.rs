// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio s.r.l.

//! The order-keeping container: insertion order kept through a JSON decode and
//! encode, a key inserted again keeping its position, removal closing the gap,
//! and a repeated key refused with the key named.
//!
//! The probe order `d, 8080, bbbb, aaaaaaaa, cc` is neither lexicographic, nor
//! length-first, nor the order a JavaScript object gives an array-index key, so a
//! container that ordered its keys by any of those rules fails here on the first
//! key rather than by coincidence.

use aggrega_abi::OrderedMap;
use serde::{Deserialize, Serialize};

const PROBE_KEYS: [&str; 5] = ["d", "8080", "bbbb", "aaaaaaaa", "cc"];
const PROBE_OBJECT: &str = r#"{"d":"1","8080":"2","bbbb":"3","aaaaaaaa":"4","cc":"5"}"#;

/// A struct shaped like a generated payload type, deriving its serde codec, so
/// the container is exercised the way a payload field carries it.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
struct Holder {
    name: String,
    labels: OrderedMap<String>,
}

fn probe_map() -> OrderedMap<String> {
    serde_json::from_str(PROBE_OBJECT).expect("decode the probe object")
}

fn keys_of<V>(m: &OrderedMap<V>) -> Vec<&str> {
    m.keys().map(String::as_str).collect()
}

#[test]
fn decode_then_encode_keeps_the_probe_order_byte_for_byte() {
    let m = probe_map();
    assert_eq!(keys_of(&m), PROBE_KEYS);
    assert_eq!(m.len(), PROBE_KEYS.len());
    assert!(!m.is_empty());
    for (i, key) in PROBE_KEYS.iter().enumerate() {
        assert_eq!(m.get(key), Some(&(i + 1).to_string()), "value under {key}");
    }
    assert_eq!(serde_json::to_string(&m).expect("encode"), PROBE_OBJECT);
}

#[test]
fn a_derived_struct_carries_the_order_through_its_field() {
    let doc = format!(r#"{{"name":"n","labels":{PROBE_OBJECT}}}"#);
    let h: Holder = serde_json::from_str(&doc).expect("decode the holder");
    assert_eq!(keys_of(&h.labels), PROBE_KEYS);
    assert_eq!(serde_json::to_string(&h).expect("encode the holder"), doc);
}

#[test]
fn a_nested_map_keeps_the_order_at_both_levels() {
    let doc = format!(r#"{{"zz":{PROBE_OBJECT},"8080":{{}},"a":{{"y":"1","x":"2"}}}}"#);
    let m: OrderedMap<OrderedMap<String>> = serde_json::from_str(&doc).expect("decode");
    assert_eq!(keys_of(&m), ["zz", "8080", "a"]);
    assert_eq!(keys_of(m.get("zz").expect("zz")), PROBE_KEYS);
    assert!(m.get("8080").expect("8080").is_empty());
    assert_eq!(keys_of(m.get("a").expect("a")), ["y", "x"]);
    assert_eq!(serde_json::to_string(&m).expect("encode"), doc);
}

#[test]
fn an_empty_map_reads_and_writes_as_an_empty_object() {
    let m: OrderedMap<String> = serde_json::from_str("{}").expect("decode");
    assert!(m.is_empty());
    assert_eq!(m.len(), 0);
    assert_eq!(m.keys().count(), 0);
    assert_eq!(m.get("d"), None);
    assert_eq!(serde_json::to_string(&m).expect("encode"), "{}");

    assert!(OrderedMap::<String>::new().is_empty());
    assert!(OrderedMap::<String>::default().is_empty());

    let h = Holder::default();
    assert_eq!(
        serde_json::to_string(&h).expect("encode"),
        r#"{"name":"","labels":{}}"#
    );
}

#[test]
fn a_repeated_key_is_refused_naming_the_key() {
    let err = serde_json::from_str::<OrderedMap<String>>(r#"{"a":"1","b":"2","a":"3"}"#)
        .expect_err("a repeated key must be refused");
    assert!(
        err.to_string().contains(r#"duplicate key "a""#),
        "the refusal names the key: {err}"
    );

    let doc = r#"{"name":"n","labels":{"8080":"1","8080":"2"}}"#;
    let err = serde_json::from_str::<Holder>(doc)
        .expect_err("a repeated key inside a field must be refused");
    assert!(
        err.to_string().contains(r#"duplicate key "8080""#),
        "the refusal names the key: {err}"
    );
}

#[test]
fn a_non_object_is_refused() {
    for doc in ["null", "[]", r#""d""#, "1"] {
        assert!(
            serde_json::from_str::<OrderedMap<String>>(doc).is_err(),
            "{doc} is not an object and must be refused"
        );
    }
}

#[test]
fn inserting_a_present_key_replaces_the_value_and_keeps_its_position() {
    let mut m = probe_map();
    assert_eq!(
        m.insert("8080".to_string(), "x".to_string()),
        Some("2".to_string())
    );
    assert_eq!(keys_of(&m), PROBE_KEYS);
    assert_eq!(m.get("8080"), Some(&"x".to_string()));
    assert_eq!(m.len(), PROBE_KEYS.len());

    assert_eq!(m.insert("0".to_string(), "6".to_string()), None);
    assert_eq!(keys_of(&m), ["d", "8080", "bbbb", "aaaaaaaa", "cc", "0"]);
    assert_eq!(
        serde_json::to_string(&m).expect("encode"),
        r#"{"d":"1","8080":"x","bbbb":"3","aaaaaaaa":"4","cc":"5","0":"6"}"#
    );
}

#[test]
fn removing_a_key_closes_the_gap_and_keeps_every_lookup() {
    let mut m = probe_map();
    assert_eq!(m.remove("8080"), Some("2".to_string()));
    assert_eq!(m.remove("8080"), None);
    assert!(!m.contains_key("8080"));
    assert_eq!(keys_of(&m), ["d", "bbbb", "aaaaaaaa", "cc"]);
    // Every entry after the removed one moved up a position; each must still be
    // found under its own key.
    for (key, want) in [("d", "1"), ("bbbb", "3"), ("aaaaaaaa", "4"), ("cc", "5")] {
        assert_eq!(m.get(key), Some(&want.to_string()), "value under {key}");
    }
    assert_eq!(m.remove("missing"), None);

    assert_eq!(m.insert("8080".to_string(), "2".to_string()), None);
    assert_eq!(keys_of(&m), ["d", "bbbb", "aaaaaaaa", "cc", "8080"]);
}

#[test]
fn every_iteration_runs_in_the_map_order() {
    let m = probe_map();
    let values: Vec<&str> = m.values().map(String::as_str).collect();
    assert_eq!(values, ["1", "2", "3", "4", "5"]);
    let pairs: Vec<(&str, &str)> = m.iter().map(|(k, v)| (k.as_str(), v.as_str())).collect();
    assert_eq!(
        pairs,
        [
            ("d", "1"),
            ("8080", "2"),
            ("bbbb", "3"),
            ("aaaaaaaa", "4"),
            ("cc", "5")
        ]
    );
    let mut borrowed = Vec::new();
    for (k, _) in &m {
        borrowed.push(k.as_str());
    }
    assert_eq!(borrowed, PROBE_KEYS);
    assert_eq!(m.keys().next_back().map(String::as_str), Some("cc"));
    assert_eq!(m.iter().len(), PROBE_KEYS.len());
    let owned: Vec<(String, String)> = m.clone().into_iter().collect();
    assert_eq!(owned[0], ("d".to_string(), "1".to_string()));
    assert_eq!(owned.len(), PROBE_KEYS.len());
}

#[test]
fn collecting_keeps_the_first_position_of_a_key_and_its_last_value() {
    let m: OrderedMap<i64> = [("b", 1), ("a", 2), ("b", 3)]
        .into_iter()
        .map(|(k, v)| (k.to_string(), v))
        .collect();
    assert_eq!(keys_of(&m), ["b", "a"]);
    assert_eq!(m.get("b"), Some(&3));
}

#[test]
fn equality_and_debug_follow_the_map_order() {
    let a = probe_map();
    let b = probe_map();
    assert_eq!(a, b);
    let reordered: OrderedMap<String> =
        serde_json::from_str(r#"{"8080":"2","d":"1","bbbb":"3","aaaaaaaa":"4","cc":"5"}"#)
            .expect("decode");
    assert_ne!(
        a, reordered,
        "the same entries in another order are a different map"
    );
    assert_eq!(
        format!("{a:?}"),
        r#"{"d": "1", "8080": "2", "bbbb": "3", "aaaaaaaa": "4", "cc": "5"}"#
    );
}
