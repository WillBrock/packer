package packer

import (
	"reflect"
	"testing"
)

func testBuild() *coreBuild {
	return &coreBuild{
		name:          "test",
		builder:       &MockBuilder{ArtifactId: "b"},
		builderConfig: 42,
		builderType:   "foo",
		hooks: map[string][]Hook{
			"foo": []Hook{&MockHook{}},
		},
		provisioners: []coreBuildProvisioner{
			coreBuildProvisioner{&MockProvisioner{}, []interface{}{42}},
		},
		postProcessors: [][]coreBuildPostProcessor{
			[]coreBuildPostProcessor{
				coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp"}, "testPP", make(map[string]interface{}), true},
			},
		},
		variables: make(map[string]coreBuildVariable),
	}
}

func testDefaultPackerConfig() map[string]interface{} {
	return map[string]interface{}{
		BuildNameConfigKey:     "test",
		BuilderTypeConfigKey:   "foo",
		DebugConfigKey:         false,
		ForceConfigKey:         false,
		UserVariablesConfigKey: make(map[string]string),
	}
}
func TestBuild_Name(t *testing.T) {
	build := testBuild()
	if build.Name() != "test" {
		t.Fatalf("bad: %s", build.Name())
	}
}

func TestBuild_Prepare(t *testing.T) {
	packerConfig := testDefaultPackerConfig()

	build := testBuild()
	builder := build.builder.(*MockBuilder)

	build.Prepare(nil)
	if !builder.PrepareCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(builder.PrepareConfig, []interface{}{42, packerConfig}) {
		t.Fatalf("bad: %#v", builder.PrepareConfig)
	}

	coreProv := build.provisioners[0]
	prov := coreProv.provisioner.(*MockProvisioner)
	if !prov.PrepCalled {
		t.Fatal("prep should be called")
	}
	if !reflect.DeepEqual(prov.PrepConfigs, []interface{}{42, packerConfig}) {
		t.Fatalf("bad: %#v", prov.PrepConfigs)
	}

	corePP := build.postProcessors[0][0]
	pp := corePP.processor.(*TestPostProcessor)
	if !pp.configCalled {
		t.Fatal("should be called")
	}
	if !reflect.DeepEqual(pp.configVal, []interface{}{make(map[string]interface{}), packerConfig}) {
		t.Fatalf("bad: %#v", pp.configVal)
	}
}

func TestBuild_Prepare_Twice(t *testing.T) {
	build := testBuild()
	warn, err := build.Prepare(nil)
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err != nil {
		t.Fatalf("bad error: %s", err)
	}

	defer func() {
		p := recover()
		if p == nil {
			t.Fatalf("should've paniced")
		}

		if p.(string) != "prepare already called" {
			t.Fatalf("Invalid panic: %s", p)
		}
	}()

	build.Prepare(nil)
}

func TestBuildPrepare_BuilderWarniings(t *testing.T) {
	expected := []string{"foo"}

	build := testBuild()
	builder := build.builder.(*MockBuilder)
	builder.PrepareWarnings = expected

	warn, err := build.Prepare(nil)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if !reflect.DeepEqual(warn, expected) {
		t.Fatalf("bad: %#v", warn)
	}
}

func TestBuild_Prepare_Debug(t *testing.T) {
	packerConfig := testDefaultPackerConfig()
	packerConfig[DebugConfigKey] = true

	build := testBuild()
	builder := build.builder.(*MockBuilder)

	build.SetDebug(true)
	build.Prepare(nil)
	if !builder.PrepareCalled {
		t.Fatalf("should be called")
	}
	if !reflect.DeepEqual(builder.PrepareConfig, []interface{}{42, packerConfig}) {
		t.Fatalf("bad: %#v", builder.PrepareConfig)
	}

	coreProv := build.provisioners[0]
	prov := coreProv.provisioner.(*MockProvisioner)
	if !prov.PrepCalled {
		t.Fatal("prepare should be called")
	}
	if !reflect.DeepEqual(prov.PrepConfigs, []interface{}{42, packerConfig}) {
		t.Fatalf("bad: %#v", prov.PrepConfigs)
	}
}

func TestBuildPrepare_variables_default(t *testing.T) {
	packerConfig := testDefaultPackerConfig()
	packerConfig[UserVariablesConfigKey] = map[string]string{
		"foo": "bar",
	}

	build := testBuild()
	build.variables["foo"] = coreBuildVariable{Default: "bar"}
	builder := build.builder.(*MockBuilder)

	warn, err := build.Prepare(nil)
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !builder.PrepareCalled {
		t.Fatal("prepare should be called")
	}

	if !reflect.DeepEqual(builder.PrepareConfig[1], packerConfig) {
		t.Fatalf("prepare bad: %#v", builder.PrepareConfig[1])
	}
}

func TestBuildPrepare_variables_nonexist(t *testing.T) {
	build := testBuild()
	build.variables["foo"] = coreBuildVariable{Default: "bar"}

	warn, err := build.Prepare(map[string]string{"bar": "baz"})
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err == nil {
		t.Fatal("should have had error")
	}
}

func TestBuildPrepare_variables_override(t *testing.T) {
	packerConfig := testDefaultPackerConfig()
	packerConfig[UserVariablesConfigKey] = map[string]string{
		"foo": "baz",
	}

	build := testBuild()
	build.variables["foo"] = coreBuildVariable{Default: "bar"}
	builder := build.builder.(*MockBuilder)

	warn, err := build.Prepare(map[string]string{"foo": "baz"})
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !builder.PrepareCalled {
		t.Fatal("prepare should be called")
	}

	if !reflect.DeepEqual(builder.PrepareConfig[1], packerConfig) {
		t.Fatalf("prepare bad: %#v", builder.PrepareConfig[1])
	}
}

func TestBuildPrepare_variablesRequired(t *testing.T) {
	build := testBuild()
	build.variables["foo"] = coreBuildVariable{Required: true}

	warn, err := build.Prepare(map[string]string{})
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err == nil {
		t.Fatal("should have had error")
	}

	// Test with setting the value
	build = testBuild()
	build.variables["foo"] = coreBuildVariable{Required: true}
	warn, err = build.Prepare(map[string]string{"foo": ""})
	if len(warn) > 0 {
		t.Fatalf("bad: %#v", warn)
	}
	if err != nil {
		t.Fatalf("should not have error: %s", err)
	}
}

func TestBuild_Run(t *testing.T) {
	cache := &TestCache{}
	ui := testUi()

	build := testBuild()
	build.Prepare(nil)
	artifacts, err := build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("bad: %#v", artifacts)
	}

	// Verify builder was run
	builder := build.builder.(*MockBuilder)
	if !builder.RunCalled {
		t.Fatal("should be called")
	}

	// Verify hooks are disapatchable
	dispatchHook := builder.RunHook
	dispatchHook.Run("foo", nil, nil, 42)

	hook := build.hooks["foo"][0].(*MockHook)
	if !hook.RunCalled {
		t.Fatal("should be called")
	}
	if hook.RunData != 42 {
		t.Fatalf("bad: %#v", hook.RunData)
	}

	// Verify provisioners run
	dispatchHook.Run(HookProvision, nil, nil, 42)
	prov := build.provisioners[0].provisioner.(*MockProvisioner)
	if !prov.ProvCalled {
		t.Fatal("should be called")
	}

	// Verify post-processor was run
	pp := build.postProcessors[0][0].processor.(*TestPostProcessor)
	if !pp.ppCalled {
		t.Fatal("should be called")
	}
}

func TestBuild_Run_Artifacts(t *testing.T) {
	cache := &TestCache{}
	ui := testUi()

	// Test case: Test that with no post-processors, we only get the
	// main build.
	build := testBuild()
	build.postProcessors = [][]coreBuildPostProcessor{}

	build.Prepare(nil)
	artifacts, err := build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedIds := []string{"b"}
	artifactIds := make([]string, len(artifacts))
	for i, artifact := range artifacts {
		artifactIds[i] = artifact.Id()
	}

	if !reflect.DeepEqual(artifactIds, expectedIds) {
		t.Fatalf("unexpected ids: %#v", artifactIds)
	}

	// Test case: Test that with a single post-processor that doesn't keep
	// inputs, only that post-processors results are returned.
	build = testBuild()
	build.postProcessors = [][]coreBuildPostProcessor{
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp"}, "pp", make(map[string]interface{}), false},
		},
	}

	build.Prepare(nil)
	artifacts, err = build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedIds = []string{"pp"}
	artifactIds = make([]string, len(artifacts))
	for i, artifact := range artifacts {
		artifactIds[i] = artifact.Id()
	}

	if !reflect.DeepEqual(artifactIds, expectedIds) {
		t.Fatalf("unexpected ids: %#v", artifactIds)
	}

	// Test case: Test that with multiple post-processors, as long as one
	// keeps the original, the original is kept.
	build = testBuild()
	build.postProcessors = [][]coreBuildPostProcessor{
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp1"}, "pp", make(map[string]interface{}), false},
		},
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp2"}, "pp", make(map[string]interface{}), true},
		},
	}

	build.Prepare(nil)
	artifacts, err = build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedIds = []string{"b", "pp1", "pp2"}
	artifactIds = make([]string, len(artifacts))
	for i, artifact := range artifacts {
		artifactIds[i] = artifact.Id()
	}

	if !reflect.DeepEqual(artifactIds, expectedIds) {
		t.Fatalf("unexpected ids: %#v", artifactIds)
	}

	// Test case: Test that with sequences, intermediaries are kept if they
	// want to be.
	build = testBuild()
	build.postProcessors = [][]coreBuildPostProcessor{
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp1a"}, "pp", make(map[string]interface{}), false},
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp1b"}, "pp", make(map[string]interface{}), true},
		},
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp2a"}, "pp", make(map[string]interface{}), false},
			coreBuildPostProcessor{&TestPostProcessor{artifactId: "pp2b"}, "pp", make(map[string]interface{}), false},
		},
	}

	build.Prepare(nil)
	artifacts, err = build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedIds = []string{"pp1a", "pp1b", "pp2b"}
	artifactIds = make([]string, len(artifacts))
	for i, artifact := range artifacts {
		artifactIds[i] = artifact.Id()
	}

	if !reflect.DeepEqual(artifactIds, expectedIds) {
		t.Fatalf("unexpected ids: %#v", artifactIds)
	}

	// Test case: Test that with a single post-processor that forcibly
	// keeps inputs, that the artifacts are kept.
	build = testBuild()
	build.postProcessors = [][]coreBuildPostProcessor{
		[]coreBuildPostProcessor{
			coreBuildPostProcessor{
				&TestPostProcessor{artifactId: "pp", keep: true}, "pp", make(map[string]interface{}), false,
			},
		},
	}

	build.Prepare(nil)
	artifacts, err = build.Run(ui, cache)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expectedIds = []string{"b", "pp"}
	artifactIds = make([]string, len(artifacts))
	for i, artifact := range artifacts {
		artifactIds[i] = artifact.Id()
	}

	if !reflect.DeepEqual(artifactIds, expectedIds) {
		t.Fatalf("unexpected ids: %#v", artifactIds)
	}
}

func TestBuild_RunBeforePrepare(t *testing.T) {
	defer func() {
		p := recover()
		if p == nil {
			t.Fatal("should panic")
		}

		if p.(string) != "Prepare must be called first" {
			t.Fatalf("bad: %s", p.(string))
		}
	}()

	testBuild().Run(testUi(), &TestCache{})
}

func TestBuild_Cancel(t *testing.T) {
	build := testBuild()
	build.Cancel()

	builder := build.builder.(*MockBuilder)
	if !builder.CancelCalled {
		t.Fatal("cancel should be called")
	}
}
