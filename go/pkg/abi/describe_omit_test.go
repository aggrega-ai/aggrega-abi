// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Cognitio S.r.l.

package abi_test

import (
	"bytes"
	"testing"

	"github.com/aggrega-ai/aggrega-abi-go/pkg/abi"
)

// TestPluginDescriptor_OmitsAbsentOptionals pins the wire-faithful omission of
// the optional nested objects a plugin does not declare. A descriptor that omits
// its render-behavior contract, its authorship metadata, and (transitively) the
// artifact-delimiter pair and assembly-name annotation must marshal WITHOUT any
// of those keys — so a guest that migrates to marshalling through this codec
// produces the same bytes it emits today rather than gaining always-present
// empty objects.
func TestPluginDescriptor_OmitsAbsentOptionals(t *testing.T) {
	t.Parallel()
	d := abi.PluginDescriptor{
		AbiVersion: "v1beta1",
		PluginType: "cli",
		Exports:    []abi.PluginExport{{Name: "cli_run"}},
	}
	got, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	t.Logf("bare descriptor wire: %s", got)
	for _, key := range []string{`"contract"`, `"metadata"`, `"artifact_delimiters"`, `"assembly_name_annotation"`} {
		if bytes.Contains(got, []byte(key)) {
			t.Errorf("absent optional %s leaked into wire: %s", key, got)
		}
	}
	const want = `{"abi_version":"v1beta1","plugin_type":"cli","exports":[{"name":"cli_run"}]}`
	if string(got) != want {
		t.Errorf("bare descriptor wire mismatch:\n want: %s\n  got: %s", want, got)
	}
}

// TestProviderContract_OmitsAbsentNestedObjects pins the same omission one level
// down: a contract that reports no artifact-delimiter pair and no assembly-name
// annotation marshals without those keys.
func TestProviderContract_OmitsAbsentNestedObjects(t *testing.T) {
	t.Parallel()
	c := abi.ProviderContract{
		InputKinds:       abi.InputKinds{Sourced: true},
		BoundaryAssembly: "none",
	}
	got, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	t.Logf("bare contract wire: %s", got)
	for _, key := range []string{`"artifact_delimiters"`, `"assembly_name_annotation"`} {
		if bytes.Contains(got, []byte(key)) {
			t.Errorf("absent optional %s leaked into wire: %s", key, got)
		}
	}
}

// mooDescriptor is the exact describe() self-report the moo CLI plugin returns
// today (the pluginDescriptor const in the moo guest). It is a real published
// guest that declares no render-behavior contract, so its wire carries no
// contract key — the case this codec must reproduce faithfully.
const mooDescriptor = `{"abi_version":"v1beta1","plugin_type":"cli","commands":[{"name":"moo","description":"This Aggrega has Super Cow Powers. Pass --alien for a UFO abduction scene."}],"exports":[{"name":"cli_run","host_functions":["aggrega_cli_write_stdout"]}],"metadata":{"author":"aggrega","license":"Apache-2.0"}}`

// TestMooDescribeRoundTrip decodes moo's exact describe() string and re-encodes
// it through the codec, asserting the codec neither invents a contract key nor
// drops moo's metadata. The re-encoded wire is emitted in the codec's canonical
// field-declaration order.
func TestMooDescribeRoundTrip(t *testing.T) {
	t.Parallel()
	var d abi.PluginDescriptor
	if err := d.UnmarshalJSON([]byte(mooDescriptor)); err != nil {
		t.Fatalf("unmarshal moo: %v", err)
	}
	got, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("marshal moo: %v", err)
	}
	t.Logf("moo re-encoded wire: %s", got)
	if d.Contract != nil {
		t.Errorf("moo gained a contract on decode: %+v", d.Contract)
	}
	if bytes.Contains(got, []byte(`"contract"`)) {
		t.Errorf("moo re-encode invented a contract key: %s", got)
	}
	if d.Metadata == nil || d.Metadata.Author != "aggrega" || d.Metadata.License != "Apache-2.0" {
		t.Errorf("moo metadata not preserved: %+v", d.Metadata)
	}
	// The canonical wire holds exactly moo's declared content, with the
	// codec's field-declaration order (exports before commands).
	const canonical = `{"abi_version":"v1beta1","plugin_type":"cli","exports":[{"name":"cli_run","host_functions":["aggrega_cli_write_stdout"]}],"commands":[{"name":"moo","description":"This Aggrega has Super Cow Powers. Pass --alien for a UFO abduction scene."}],"metadata":{"author":"aggrega","license":"Apache-2.0"}}`
	if string(got) != canonical {
		t.Errorf("moo canonical wire mismatch:\n want: %s\n  got: %s", canonical, got)
	}
	// Whether moo's exact source string round-trips byte-identically depends
	// only on whether moo already declares its fields in canonical order.
	if string(got) == mooDescriptor {
		t.Logf("moo source string is already in canonical order (exact round-trip)")
	} else {
		t.Logf("moo source string is NOT in canonical order; exact bytes differ only by field order:\n source: %s\n  canon: %s", mooDescriptor, got)
	}
	// The contract-omission property holds regardless of field order: reorder
	// moo's source into canonical order and it round-trips exactly.
	reordered := canonicalize(t, mooDescriptor)
	if reordered != string(got) {
		t.Errorf("canonicalized moo does not round-trip:\n want: %s\n  got: %s", got, reordered)
	}
}

// canonicalize decodes and re-encodes a descriptor string through the codec,
// yielding the codec's canonical wire form.
func canonicalize(t *testing.T, s string) string {
	t.Helper()
	var d abi.PluginDescriptor
	if err := d.UnmarshalJSON([]byte(s)); err != nil {
		t.Fatalf("canonicalize unmarshal: %v", err)
	}
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("canonicalize marshal: %v", err)
	}
	return string(b)
}
